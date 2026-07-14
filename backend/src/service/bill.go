package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/helper"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/okslip"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
)

const (
	slipFreshnessWindow = 30 * time.Minute
	bankAccountType     = "BANKAC"
)

type BillService interface {
	Create(ctx context.Context, b *models.Bill) error
	GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error)
	GetUnpaidByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetUnpaidByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error)
	Pay(ctx context.Context, userId string, guildId string, billId int, proofUrl string) (*models.Bill, error)
	DeleteById(ctx context.Context, billId int) error
}

type billService struct {
	db           *sql.DB
	repo         repository.BillRepository
	groupRepo    repository.GroupRepository
	slipRepo     repository.SlipRepository
	okSlipClient okslip.OkSlipClient
}

func NewBillService(
	db *sql.DB,
	repo repository.BillRepository,
	groupRepo repository.GroupRepository,
	slipRepo repository.SlipRepository,
	okSlipClient okslip.OkSlipClient,
) BillService {
	return &billService{
		db:           db,
		repo:         repo,
		groupRepo:    groupRepo,
		slipRepo:     slipRepo,
		okSlipClient: okSlipClient,
	}
}

func (s *billService) Create(ctx context.Context, b *models.Bill) error {
	if b.GroupID <= 0 {
		return exception.ErrInvalidID
	}
	if b.MemberID == "" || b.GuildID == "" {
		return exception.ErrInvalidDiscordId
	}
	if b.Year < 2000 || b.Year > 3000 {
		return exception.ErrInvalidYear
	}
	if b.Month < 1 || b.Month > 12 {
		return exception.ErrInvalidMonth
	}
	if b.AmountDue <= 0 {
		return exception.ErrInvalidAmount
	}
	if b.Currency == "" {
		return exception.ErrInvalidCurrency
	}

	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now

	return s.repo.Create(ctx, b)
}

func (s *billService) GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error) {
	bills, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}
	if bills == nil {
		bills = []models.Bill{}
	}
	return bills, nil
}

func (s *billService) GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}
	return s.filterGuildBills(ctx, guildId, func(b models.Bill) bool {
		return b.MemberID == userId
	})
}

func (s *billService) GetUnpaidByGuildId(ctx context.Context, guildId string) ([]models.Bill, error) {
	if guildId == "" {
		return nil, exception.ErrInvalidDiscordId
	}
	return s.filterGuildBills(ctx, guildId, func(b models.Bill) bool {
		return b.Status != models.BillStatusVerified
	})
}

func (s *billService) GetUnpaidByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}
	return s.filterGuildBills(ctx, guildId, func(b models.Bill) bool {
		return b.Status != models.BillStatusVerified && b.MemberID == userId
	})
}

// filterGuildBills loads a guild's bills and returns those matching keep.
func (s *billService) filterGuildBills(ctx context.Context, guildId string, keep func(models.Bill) bool) ([]models.Bill, error) {
	bills, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	res := make([]models.Bill, 0, len(bills))
	for _, b := range bills {
		if keep(b) {
			res = append(res, b)
		}
	}
	return res, nil
}

func (s *billService) Pay(ctx context.Context, userId string, guildId string, billId int, proofUrl string) (*models.Bill, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}
	if proofUrl == "" {
		return nil, exception.ErrNoUrl
	}

	b, err := s.repo.GetById(ctx, billId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, exception.ErrBillNotFound
		}
		return nil, err
	}
	if b.Status == models.BillStatusVerified {
		return nil, exception.ErrBillAlreadyPaid
	}
	if b.MemberID != userId || b.GuildID != guildId {
		return nil, exception.ErrNotMatch
	}

	g, err := s.groupRepo.GetById(ctx, b.GroupID)
	if err != nil {
		return nil, err
	}

	slip, err := s.verifySlip(ctx, proofUrl)
	if err != nil {
		return nil, err
	}
	data := slip.Data

	if time.Now().UTC().Sub(data.TransTimestamp) > slipFreshnessWindow {
		return nil, exception.ErrSlipExpired
	}

	if err := matchReceiver(g, data); err != nil {
		return nil, err
	}

	// Pre-check duplicate; the UNIQUE(transRef) constraint inside the tx is the
	// authoritative, race-safe guard.
	existingSlip, err := s.slipRepo.GetByTransRef(ctx, data.TransRef)
	if err != nil {
		return nil, err
	}
	if existingSlip != nil {
		return nil, exception.ErrDupeSlip
	}

	amountPaid := data.Amount
	now := time.Now().UTC()

	b.UpdatedAt = now
	b.SubmittedAt = &now
	b.VerifiedAt = &now
	b.Status = models.BillStatusVerified

	slipJSON, err := json.Marshal(slip)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize slip data: %w", err)
	}
	b.ProofJSON = string(slipJSON)

	if err := applyPaymentToMember(g, userId, amountPaid); err != nil {
		return nil, err
	}

	if err := s.persistPayment(ctx, billId, b, g, data.TransRef, amountPaid, now); err != nil {
		return nil, err
	}

	return b, nil
}

// verifySlip calls the provider and translates its errors into sentinels. The
// raw provider error/body is never returned to the caller.
func (s *billService) verifySlip(ctx context.Context, proofUrl string) (*okslip.Response, error) {
	slip, err := s.okSlipClient.CheckSlip(ctx, proofUrl)
	if err == nil {
		return slip, nil
	}

	if errors.Is(err, okslip.ErrTimeout) {
		return nil, exception.ErrTimeOut
	}

	var apiErr *okslip.APIErrorResponse
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 1005, 1006: // Invalid file / invalid image
			return nil, exception.ErrSlipImageInvalid
		case 1007, 1008: // No QR code / not a payment QR
			return nil, exception.ErrNoQRCode
		case 1009: // Bank data temporarily unavailable (retry later)
			return nil, exception.ErrBankDataUnavailable
		case 1010: // BBL/SCB bank delay — not verified yet, resubmit later
			return nil, exception.ErrSlipPendingBankDelay
		case 1011: // QR expired or invalid
			return nil, exception.ErrSlipExpiredOrInvalid
		case 1012: // Duplicate slip (provider-side)
			return nil, exception.ErrDupeSlip
		case 1013: // Amount mismatch
			return nil, exception.ErrSlipAmountMismatch
		default:
			// Config/quota codes (1001-1004/1015) are our-side problems.
			log.Printf("Unhandled OkSlip Code: %d - %s", apiErr.Code, apiErr.Message)
			return nil, err // maps to a generic 500 in the handler (no leak)
		}
	}

	return nil, err
}

// matchReceiver verifies the slip's receiver account/method matches the group's
// configured payment account.
func matchReceiver(g *models.Group, data okslip.Data) error {
	acc := data.Receiver.Account
	proxy := data.Receiver.Proxy

	var method models.PaymentMethod
	var slipAccount string
	switch {
	case acc.Type == bankAccountType:
		method = models.BankAccount
		slipAccount = acc.Value
	case proxy.Type != "":
		method = models.PromptPay
		slipAccount = proxy.Value
	default:
		method = models.BankAccount
		slipAccount = acc.Value
	}

	// The slip account/proxy value is masked (e.g. "086xxx7894"); compare it
	// against the group's stored account with mask-aware matching.
	if g.Payment.Method != method || !helper.MaskedDigitsMatch(slipAccount, g.Payment.Account) {
		return exception.ErrWrongReciever
	}
	return nil
}

// applyPaymentToMember reduces the paying member's debt and updates their
// status. Returns ErrMemberNotFound if the user is not in the group.
func applyPaymentToMember(g *models.Group, userId string, amountPaid float64) error {
	for i := range g.Members {
		m := &g.Members[i]
		if m.MemberID != userId {
			continue
		}

		reduction := int(amountPaid)
		if reduction > m.Dept {
			reduction = m.Dept
		}
		m.Dept -= reduction
		if m.Dept <= 0 {
			m.Dept = 0
			m.Payment = models.PaymentStatusPaid
		}
		if m.Status == models.MemberStatusInvited {
			m.Status = models.MemberStatusActive
		}
		return nil
	}
	return exception.ErrMemberNotFound
}

// persistPayment atomically records the slip, updates the bill and group, and
// creates a balance-remaining bill on underpayment. All-or-nothing.
func (s *billService) persistPayment(
	ctx context.Context,
	billId int,
	b *models.Bill,
	g *models.Group,
	transRef string,
	amountPaid float64,
	now time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	billTx := s.repo.WithTx(tx)
	groupTx := s.groupRepo.WithTx(tx)
	slipTx := s.slipRepo.WithTx(tx)

	// Record the slip first: a duplicate transRef aborts before any bill is
	// marked paid, so the same slip can never be replayed.
	if err := slipTx.Create(ctx, &models.Slip{TransRef: transRef, SubmittedAt: &now}); err != nil {
		return err
	}
	if err := billTx.Update(ctx, billId, b); err != nil {
		return err
	}
	if err := groupTx.Update(ctx, g.ID, g); err != nil {
		return err
	}

	if amountPaid < float64(b.AmountDue) {
		remaining := models.Bill{
			GroupID:     g.ID,
			GuildID:     g.DiscordGuildID,
			MemberID:    b.MemberID,
			Year:        b.Year,
			Month:       b.Month,
			AmountDue:   b.AmountDue - int(amountPaid),
			Currency:    defaultCurrency,
			Status:      models.BillStatusPending,
			Description: fmt.Sprintf("Balance remaining from Bill #%d", b.ID),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := billTx.Create(ctx, &remaining); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *billService) DeleteById(ctx context.Context, billId int) error {
	if billId <= 0 {
		return exception.ErrInvalidID
	}
	return s.repo.DeleteById(ctx, billId)
}
