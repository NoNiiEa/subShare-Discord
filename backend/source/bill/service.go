package bill

import (
	"context"
	"time"
)


type Store interface {
	NextBillID(ctx context.Context) (int64, error)
	SaveBill(ctx context.Context, b Bill) error
	GetBillByID(ctx context.Context, id int64) (*Bill, error)
	GetBillsByGroupAndMember(ctx context.Context, groupID int64, memberID string) ([]Bill, error)
	GetBillByGroupMemberCycle(ctx context.Context, groupID int64, memberID string, year, month int) (*Bill, error)
	GetBillsByMemberID(ctx context.Context, memberID string) ([]Bill, error)
	GetBillsByGroupID(ctx context.Context, groupID int64) ([]Bill, error)
	UpdateBill(ctx context.Context, b Bill) (*Bill, error)
}

type Service struct {
	store      Store
}

func NewService(store Store) *Service {
	return &Service{
		store:          store,
	}
}

func (s *Service) CreateBill(ctx context.Context, req CreateBillRequest) (*Bill, error) {
	// Basic validation
	if req.GroupID <= 0 {
		return nil, ErrInvalidGroupID
	}
	if req.MemberID == "" {
		return nil, ErrInvalidMemberID
	}
	if req.Year < 2000 || req.Year > 3000 { // arbitrary sanity check
		return nil, ErrInvalidYear
	}
	if req.Month < 1 || req.Month > 12 {
		return nil, ErrInvalidMonth
	}
	if req.AmountDue <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Currency == "" {
		return nil, ErrInvalidCurrency
	}

	now := time.Now().UTC()

	// assign ID
	id, err := s.store.NextBillID(ctx)
	if err != nil {
		return nil, err
	}

	b := Bill{
		ID:          id,
		GroupID:     req.GroupID,
		MemberID:    req.MemberID,
		Year:        req.Year,
		Month:       req.Month,
		AmountDue:   req.AmountDue,
		AmountPaid:  0, // starts unpaid
		Currency:    req.Currency,
		Status:      BillStatusPending,
		Description: req.Description,

		CreatedAt: now,
		UpdatedAt: now,

		// Proof + SubmittedAt/VerifiedAt/RejectedAt = nil by default
	}

	if err := s.store.SaveBill(ctx, b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *Service) GetBillsByGroup(ctx context.Context, groupID int64) ([]Bill, error) {
	if groupID <= 0 {
		return nil, ErrInvalidGroupID
	}

	return s.store.GetBillsByGroupID(ctx, groupID)
}

func (s *Service) GetBillsByMember(ctx context.Context, memberID string) ([]Bill, error) {
	return s.store.GetBillsByMemberID(ctx, memberID)
}