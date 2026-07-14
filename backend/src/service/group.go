package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
)

const defaultCurrency = "THB"

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
	Update(ctx context.Context, groupId int, group *models.Group) error
}

type groupService struct {
	db       *sql.DB
	repo     repository.GroupRepository
	billRepo repository.BillRepository
}

func NewGroupService(db *sql.DB, repo repository.GroupRepository, billRepo repository.BillRepository) GroupService {
	return &groupService{db: db, repo: repo, billRepo: billRepo}
}

// recalcAmountPerMember recomputes the per-member share as the total amount
// split across active members (the members who are actually billed).
func recalcAmountPerMember(g *models.Group) {
	active := 0
	for _, m := range g.Members {
		if m.Status == models.MemberStatusActive {
			active++
		}
	}
	if active > 0 {
		g.AmountPerMember = g.Amount / active
	} else {
		g.AmountPerMember = 0
	}
}

func (s *groupService) CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	if group.Name == "" {
		return nil, exception.ErrEmptyName
	}
	if group.Amount < 0 {
		return nil, exception.ErrNegetiveAmount
	}
	if group.DueDay > 31 || group.DueDay < 1 {
		return nil, exception.ErrInvalidDueDay
	}
	if group.DiscordGuildID == "" || group.OwnerDiscordID == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	group.Members = []models.GroupMember{
		{
			MemberID: group.OwnerDiscordID,
			Dept:     0,
			Status:   models.MemberStatusActive,
			Payment:  models.PaymentStatusPaid,
		},
	}
	recalcAmountPerMember(group)
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
	for _, g := range groups {
		for _, member := range g.Members {
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
	for _, g := range groups {
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

	for _, m := range g.Members {
		for _, newMemId := range memberIds {
			if m.MemberID == newMemId && m.Status == models.MemberStatusActive {
				return nil, exception.ErrAlreadyMember
			}
		}
	}

	for _, memberId := range memberIds {
		if memberId == "" {
			return nil, exception.ErrInvalidDiscordId
		}

		g.Members = append(g.Members, models.GroupMember{
			MemberID: memberId,
			Dept:     0,
			Status:   models.MemberStatusInvited,
			Payment:  models.PaymentStatusPaid,
		})
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
	for _, g := range groups {
		for _, m := range g.Members {
			if m.MemberID == userId && m.Status == models.MemberStatusInvited {
				res = append(res, g)
				break
			}
		}
	}

	return res, nil
}

// findInvitedMember locates a member who can respond to an invite (must not be
// already active). Returns the index or an error.
func findInvitedMember(g *models.Group, userId string) (int, error) {
	for i, m := range g.Members {
		if m.MemberID == userId {
			if m.Status == models.MemberStatusActive {
				return -1, exception.ErrAlreadyMember
			}
			if m.Status == models.MemberStatusInvited {
				return i, nil
			}
		}
	}
	return -1, exception.ErrNotInvited
}

func (s *groupService) AcceptInvite(ctx context.Context, groupId int, userId string) (*models.Group, error) {
	if userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	g, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return nil, err
	}

	index, err := findInvitedMember(g, userId)
	if err != nil {
		return nil, err
	}

	g.Members[index].Status = models.MemberStatusActive
	recalcAmountPerMember(g)

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

	index, err := findInvitedMember(g, userId)
	if err != nil {
		return nil, err
	}

	g.Members[index].Status = models.MemberStatusLeft

	if err := s.repo.Update(ctx, groupId, g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *groupService) CreateBillCycle(ctx context.Context, g *models.Group) error {
	// Use the same (local) clock the cron uses to select due-day, so the
	// recorded year/month can't disagree with the selection near midnight.
	now := time.Now()
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
			AmountDue:   g.AmountPerMember,
			Currency:    defaultCurrency,
			Status:      models.BillStatusPending,
			Description: "Monthly Cycle",
			ProofJSON:   "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := s.billRepo.Create(ctx, &b); err != nil {
			return err
		}
	}

	return s.repo.Update(ctx, g.ID, g)
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	billTx := s.billRepo.WithTx(tx)
	groupTx := s.repo.WithTx(tx)

	bills, err := billTx.GetByGroupID(ctx, groupId)
	if err != nil {
		return err
	}
	for _, b := range bills {
		if err := billTx.DeleteById(ctx, b.ID); err != nil {
			return err
		}
	}

	if err := groupTx.DeleteById(ctx, groupId); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *groupService) Update(ctx context.Context, groupId int, groupUpdate *models.Group) error {
	existing, err := s.repo.GetById(ctx, groupId)
	if err != nil {
		return err
	}

	if groupUpdate.Name != "" {
		existing.Name = groupUpdate.Name
	}

	if groupUpdate.Amount > 0 {
		existing.Amount = groupUpdate.Amount
		recalcAmountPerMember(existing)
	}

	if groupUpdate.DueDay >= 1 && groupUpdate.DueDay <= 31 {
		existing.DueDay = groupUpdate.DueDay
	}

	if groupUpdate.Payment.Account != "" {
		existing.Payment.Account = groupUpdate.Payment.Account
	}

	if groupUpdate.Payment.Method != "" {
		existing.Payment.Method = groupUpdate.Payment.Method
	}

	return s.repo.Update(ctx, groupId, existing)
}
