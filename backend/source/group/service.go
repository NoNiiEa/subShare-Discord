package group

import (
	"context"
	"time"

	"github.com/NoNiiEa/subShare-Discord/source/bill"
)

type Store interface {
	NextGroupID(ctx context.Context) (int64, error)
	SaveGroup(ctx context.Context, g Group) error
	GetGroup(ctx context.Context, id int64) (*Group, error)
	DeleteGroup(ctx context.Context, id int64) error
	UpdateGroup(ctx context.Context, id int64, g Group) error
	GetGroupByDueday(ctx context.Context, dueDay int) ([]Group, error)
	NextBillID(ctx context.Context) (int64, error)
	SaveBill(ctx context.Context, b bill.Bill) error
	GetBillByID(ctx context.Context, id int64) (*bill.Bill, error)
	GetBillsByGroupAndMember(ctx context.Context, groupID int64, memberID string) ([]bill.Bill, error)
	GetBillByGroupMemberCycle(ctx context.Context, groupID int64, memberID string, year, month int) (*bill.Bill, error)
	GetBillsByMemberID(ctx context.Context, memberID string) ([]bill.Bill, error)
	GetBillsByGroupID(ctx context.Context, groupID int64) ([]bill.Bill, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateGroup(ctx context.Context, req CreateGroupRequest) (*Group, error) {
	if req.Name == "" {
		return nil, ErrInvalidName
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.DueDay < 1 || req.DueDay > 31 {
		return nil, ErrInvalidDueDay
	}
	if req.DiscordGuildID == "" {
		return nil, ErrInvalidGuildID
	}
	if req.OwnerDiscordID == "" {
		return nil, ErrInvalidOwnerID
	}

	id, err := s.store.NextGroupID(ctx)
	if err != nil {
		return nil, err
	}

	var owner GroupMember = GroupMember{
		MemberID: req.OwnerDiscordID,
		Dept: 0,
		Status: MemberStatusActive,
		Payment: PaymentStatusNotPaid,
	}

	members := []GroupMember{owner}

	g := Group{
		ID:             id,
		Name:           req.Name,
		Amount:         req.Amount,
		AmountPerMember: int64(req.Amount),
		DueDay:         req.DueDay,
		Members:        members,
		DiscordGuildID: req.DiscordGuildID,
		OwnerDiscordID: req.OwnerDiscordID,
		Payment:        req.Payment,
		CreateAt:      time.Now().UTC(),
	}

	if err := s.store.SaveGroup(ctx, g); err != nil {
		return nil, err
	}

	return &g, nil
}

func (s *Service) GetGroup(ctx context.Context, id int64) (*Group, error) {
	return s.store.GetGroup(ctx, id)
}

func (s *Service) DeleteGroup(ctx context.Context, id int64) error {
	return s.store.DeleteGroup(ctx, id)
}

func (s *Service) UpdateGroup(ctx context.Context, req UpdateGroupRequest, id int64) (*Group, error) {
	if req.Name == "" {
		return nil, ErrInvalidName
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.DueDay < 1 || req.DueDay > 31 {
		return nil, ErrInvalidDueDay
	}
	if req.DiscordGuildID == "" {
		return nil, ErrInvalidGuildID
	}
	if req.OwnerDiscordID == "" {
		return nil, ErrInvalidOwnerID
	}
	if len(req.Members) == 0 {
		return nil, ErrNoMembersProvided
	}

	g, err := s.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}

	newGroup := Group{
		ID: g.ID,
		Name: req.Name,
		Amount: req.Amount,
		DueDay: req.DueDay,
		Members: req.Members,
		DiscordGuildID: req.DiscordGuildID,
		OwnerDiscordID: req.OwnerDiscordID,
		Payment: req.Payment,
		CreateAt: g.CreateAt,
	}

	err = s.store.UpdateGroup(ctx, g.ID, newGroup)
	if err != nil {
		return nil, err
	}

	g, err = s.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}


	return g, err
}

func (s *Service) InviteGroup(ctx context.Context, req InviteGroupRequest, id int64) (*Group, error) {
	g, err := s.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}

	if g.OwnerDiscordID != req.OwnerID {
		return nil, ErrInvitedPermission
	}

	if len(req.MemberIDs) == 0 {
		return nil, ErrNoMembersProvided
	}

	for _, newID := range req.MemberIDs {
		for _, m := range g.Members {
			if m.MemberID != newID {
				continue
			}

			switch m.Status {
			case MemberStatusActive:
				return nil, ErrAleadyMembered
			case MemberStatusInvited:
				return nil, ErrAleadyInvited
			case MemberStatusLeft:
				// allow re-invite -> do nothing here
			default:
				// unknown status -> treat as conflict or log
				return nil, ErrAleadyInvited
			}
		}
	}

	members := g.Members

	for _, newID := range req.MemberIDs {
		member := GroupMember{
			MemberID: newID,
			Dept: 0,
			Status:   MemberStatusInvited,
			Payment:  PaymentStatusNotPaid,
		}
		members = append(members, member)
	}

	g.Members = members
	if err := s.store.UpdateGroup(ctx, id, *g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *Service) AcceptInvite(ctx context.Context, req AcceptInviteRequest, id int64) (*Group, error) {
	g, err := s.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}

	var index = -1
	for i, member := range g.Members {
		if member.MemberID == req.UserID && member.Status == MemberStatusInvited {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, ErrNotInvited
	}

	g.Members[index].Status = MemberStatusActive
	g.AmountPerMember = int64(int(g.Amount) / len(g.Members))
	
	if err := s.store.UpdateGroup(ctx, id, *g); err != nil {
		return nil, err
	}

	return g, nil
}

func (s *Service) ResetPaymentForDueday(ctx context.Context, dueDay int) error {
	if dueDay < 1 || dueDay > 31 {
		return ErrInvalidDueDay
	}

	groups, err := s.store.GetGroupByDueday(ctx, dueDay)
	if err != nil {
		return err
	}

	for _, g := range groups {
		for i := range g.Members {
			if g.Members[i].Status != MemberStatusLeft {
				g.Members[i].Payment = PaymentStatusNotPaid
				g.Members[i].Dept += g.AmountPerMember
			}

			newID, err := s.store.NextBillID(ctx)
			if err != nil {
				return err
			}

			year := time.Now().Year()
			month := time.Now().Month()
			now := time.Now().UTC()

			b := bill.Bill{
				ID: newID,
				GroupID: g.ID,
				MemberID: g.Members[i].MemberID,
				Year: year,
				Month: int(month),
				AmountDue: float64(g.AmountPerMember),
				AmountPaid: 0,
				Currency: "THB",
				Status: bill.BillStatusSubmitted,
				Description: "",
				ProofJSON: "",
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := s.store.SaveBill(ctx, b); err != nil {
				return err
			}
		}

		if err := s.store.UpdateGroup(ctx, g.ID, g); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) MarkMemberPaid(ctx context.Context, req MarkAsPaidRequest, groupID int64, memberID string) (*GroupMember, error) {
	if groupID <= 0 {
		return nil, ErrInvalidGroupID
	}

	if memberID == "" {
		return nil, ErrNoUserID
	}

	g, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	index := -1
	for i := range g.Members {
		if g.Members[i].MemberID == memberID {
			index = i
		}
	}

	if index == -1 {
		return  nil, ErrMemberNotFound
	}

	if g.Members[index].Status != MemberStatusActive {
		return nil, ErrNotActiveMember
	}

	if g.Members[index].Payment == PaymentStatusPaid {
		return nil, ErrAlreadyPaid
	}

	g.Members[index].Dept -= req.Amount
	if g.Members[index].Dept < 0 {
		g.Members[index].Dept = 0
	}

	if g.Members[index].Dept == 0 {
		g.Members[index].Payment = PaymentStatusPaid
	}

	if err := s.store.UpdateGroup(ctx, groupID, *g); err != nil {
		return nil, err
	}

	return &g.Members[index], nil
}