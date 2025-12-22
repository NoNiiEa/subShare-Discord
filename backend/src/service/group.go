package service

import (
	"context"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
)

type GroupService interface {
	CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	GetById(ctx context.Context, groupId int) (*models.Group, error)
	GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Group, error)
	GetByGuildIdAndUserIdOwn(ctx context.Context, guildId string, userId string) ([]models.Group, error)
	InviteToGroup(ctx context.Context, groupId int, userId string, memberIds []string) (*models.Group, error)
	GetUserPendingInvite(ctx context.Context, userId string, guildId string) ([]models.Group, error)
	AcceptInvite(ctx context.Context, groupId int, userId string) (*models.Group, error)
	DeclineInvite(ctx context.Context, groupId int, userId string) (*models.Group, error)
	CreateBillCycle(ctx context.Context, g *models.Group) error
	GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error)
	DeleteById(ctx context.Context, groupId int, userId string) error
}

type groupService struct {
	repo repository.GroupRepository
	billRepo repository.BillRepository
}

func NewGroupService(repo repository.GroupRepository, billRepo repository.BillRepository) GroupService {
	return &groupService{repo: repo, billRepo: billRepo}
}

func (s *groupService) CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	if group.Name == "" {
		return nil, exception.ErrEmptyName
	}

	if group.Amount < 0 {
		return nil, exception.ErrNegetiveAmount
	}

	group.AmountPerMember = group.Amount

	if group.DueDay > 31 || group.DueDay < 1 {
		return nil, exception.ErrInvalidDueDay
	}

	var members []models.GroupMember
	member := models.GroupMember{
		MemberID: group.OwnerDiscordID,
		Dept: 0,
		Status: models.MemberStatusActive,
		Payment: models.PaymentStatusPaid,
	}

	members = append(members, member)

	group.Members = members

	if group.DiscordGuildID == "" || group.OwnerDiscordID == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	group.CreatedAt = time.Now()

	return s.repo.Create(ctx, group)
}

func (s *groupService) GetById(ctx context.Context, groupId int) (*models.Group, error) {
	return s.repo.GetById(ctx, groupId)
}

func (s *groupService) GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Group, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	groups, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Group

	for _, g := range(groups) {
		for _, member := range(g.Members) {
			if member.MemberID == userId {
				res = append(res, g)
				break
			}
		}
	}

	return res, nil
}

func (s *groupService) GetByGuildIdAndUserIdOwn(ctx context.Context, guildId string, userId string) ([]models.Group, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	groups, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Group

	for _, g := range(groups) {
		if g.OwnerDiscordID == userId {
			res = append(res, g)
		}
	}

	return res, nil
}

func (s *groupService) InviteToGroup(ctx context.Context, groupId int, userId string, memberIds []string) (*models.Group, error) {
	if userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	if len(memberIds) == 0 {
		return nil, exception.ErrNoMemberToinvite
	}

	g, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return nil, err
	}

	if g.OwnerDiscordID != userId {
		return nil, exception.ErrNoPermission
	}

	for _, m := range(g.Members) {
		for _, newMemId := range(memberIds) {
			if m.MemberID == newMemId && m.Status == models.MemberStatusActive{
				return nil, exception.ErrAlreadyMember
			}
		}
	}

	for _, m_id := range(memberIds) {
		if m_id == "" {
			return nil, exception.ErrInvalidDiscordId
		}

		var newMember = models.GroupMember{
			MemberID: m_id,
			Dept: 0,
			Status: models.MemberStatusInvited,
			Payment: models.PaymentStatusPaid,
		}

		g.Members = append(g.Members, newMember)
	}

	return g, s.repo.Update(ctx, groupId, g)
}

func (s *groupService) GetUserPendingInvite(ctx context.Context, userId string, guildId string) ([]models.Group, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	groups, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Group

	for _, g := range(groups) {
		for _, m := range(g.Members) {
			if m.MemberID == userId && m.Status == models.MemberStatusInvited {
				res = append(res, g)
				break
			}
		}
	}

	return res, nil
}

func (s *groupService) AcceptInvite(ctx context.Context, groupId int, userId string) (*models.Group, error) {
	if userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	g, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return nil, err
	}

	var index int = -1
	for i, m := range(g.Members) {
		if m.MemberID == userId {
			if m.Status == models.MemberStatusActive {
				return nil, exception.ErrAlreadyMember
			}

			index = i
			break
		}
	}

	if index == -1 {
		return nil, exception.ErrNotInvited
	}

	g.Members[index].Status = models.MemberStatusActive
	g.AmountPerMember = int(g.Amount / len(g.Members))

	if err := s.repo.Update(ctx, groupId, g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *groupService) DeclineInvite(ctx context.Context, groupId int, userId string) (*models.Group, error) {
	if userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	g, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return nil, err
	}

	var index int = -1
	for i, m := range(g.Members) {
		if m.MemberID == userId {
			if m.Status == models.MemberStatusActive {
				return nil, exception.ErrAlreadyMember
			}

			index = i
			break
		}
	}

	if index == -1 {
		return nil, exception.ErrNotInvited
	}

	g.Members[index].Status = models.MemberStatusLeft
	if err := s.repo.Update(ctx, groupId, g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *groupService) CreateBillCycle(ctx context.Context, g *models.Group) error {
    now := time.Now().UTC()
    year := now.Year()
    month := int(now.Month())

    for i := range g.Members {
        m := &g.Members[i] 

        if m.Status != models.MemberStatusActive || m.MemberID == g.OwnerDiscordID {
            continue
        }

        m.Payment = models.PaymentStatusNotPaid
        m.Dept += g.AmountPerMember

        b := models.Bill{
            GroupID:     g.ID,
            GuildID:     g.DiscordGuildID,
            MemberID:    m.MemberID,
            Year:        year,
            Month:       month,
            AmountDue:   g.AmountPerMember, // Changed to PerMember (Check if this is correct for you)
            Currency:    "THB",
            Status:      models.BillStatusPending,
            Description: "Monthly Cycle", // Optional: Add a default description
            ProofJSON:   "",
            CreatedAt:   now,
            UpdatedAt:   now,
        }

        if err := s.billRepo.Create(ctx, &b); err != nil {
            return err
        }
    }

    if err := s.repo.Update(ctx, g.ID, g); err != nil {
        return err
    }

    return nil
}

func (s *groupService) GetByDueDay(ctx context.Context, dueDay int) ([]models.Group, error) {
	return s.repo.GetByDueDay(ctx, dueDay)
}

func (s *groupService) DeleteById(ctx context.Context, groupId int, userId string) error {
	if groupId <= 0 {
		return exception.ErrInvalidID
	}

	g, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return err
	}

	if g.OwnerDiscordID != userId {
		return exception.ErrNoPermission
	}

	bills, err := s.billRepo.GetByGroupID(ctx, groupId)
	if err != nil {
		return err
	}

	for _, b := range(bills) {
		if err := s.billRepo.DeleteById(ctx, b.ID); err != nil {
			return err
		}
	}

	return s.repo.DeleteById(ctx, groupId)
}