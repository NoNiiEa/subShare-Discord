package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
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

	// maxBatchBills matches Discord's cap on select-menu options, which is what
	// bounds a batch in practice.
	maxBatchBills = 25
	// amountEpsilon absorbs float noise when comparing a slip amount to an
	// integer total.
	amountEpsilon = 0.005
)

// PayMultipleResult is the outcome of settling several bills with one slip.
// SurplusCredited is how much of an overpayment was applied to the payer's
// remaining debt; any overpayment not credited is lost.
type PayMultipleResult struct {
	Bills           []models.Bill `json:"bills"`
	TotalDue        int           `json:"total_due"`
	AmountPaid      float64       `json:"amount_paid"`
	SurplusCredited float64       `json:"surplus_credited"`
}

type BillService interface {
	Create(ctx context.Context, b *models.Bill) error
	GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error)
	GetUnpaidByGuildId(ctx context.Context, guildId string) ([]models.Bill, error)
	GetUnpaidByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error)
	Pay(ctx context.Context, userId string, guildId string, billId int, proofUrl string) (*models.Bill, error)
	PayMultiple(ctx context.Context, userId string, guildId string, billIds []int, proofUrl string) (*PayMultipleResult, error)
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

	vs, err := s.prepareSlip(ctx, proofUrl, g.Payment)
	if err != nil {
		return nil, err
	}

	amountPaid := vs.Amount
	now := time.Now().UTC()

	b.UpdatedAt = now
	b.SubmittedAt = &now
	b.VerifiedAt = &now
	b.Status = models.BillStatusVerified
	b.ProofJSON = vs.ProofJSON

	if err := applyPaymentToMember(g, userId, amountPaid); err != nil {
		return nil, err
	}

	if err := s.persistPayment(ctx, billId, b, g, vs.TransRef, amountPaid, now); err != nil {
		return nil, err
	}

	return b, nil
}

// verifiedSlip is a slip that has passed provider verification, the freshness
// window, the receiver check and the duplicate pre-check.
type verifiedSlip struct {
	ProofJSON string
	TransRef  string
	Amount    float64
}

// prepareSlip verifies a slip against the payee it is supposed to have been
// sent to, and returns it ready to be persisted. Shared by Pay and PayMultiple.
func (s *billService) prepareSlip(ctx context.Context, proofUrl string, payee models.PaymentAccount) (*verifiedSlip, error) {
	slip, err := s.verifySlip(ctx, proofUrl)
	if err != nil {
		return nil, err
	}
	data := slip.Data

	if time.Now().UTC().Sub(data.TransTimestamp) > slipFreshnessWindow {
		return nil, exception.ErrSlipExpired
	}

	if err := matchReceiver(payee, data); err != nil {
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

	slipJSON, err := json.Marshal(slip)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize slip data: %w", err)
	}

	return &verifiedSlip{
		ProofJSON: string(slipJSON),
		TransRef:  data.TransRef,
		Amount:    data.Amount,
	}, nil
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

// matchReceiver verifies the slip's receiver account/method matches the
// configured payment account it was supposed to be sent to.
func matchReceiver(payee models.PaymentAccount, data okslip.Data) error {
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
	// against the stored account with mask-aware matching.
	if payee.Method != method || !helper.MaskedDigitsMatch(slipAccount, payee.Account) {
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

// PayMultiple settles several bills with a single slip. Every selected bill must
// belong to the caller and be payable to the same owner and the same account,
// and the slip must cover their combined total. It is all-or-nothing: any
// failure leaves every bill untouched.
//
// Unlike Pay, this does not create a "balance remaining" bill — an insufficient
// slip is rejected outright, since splitting a shortfall across several bills is
// ambiguous.
func (s *billService) PayMultiple(ctx context.Context, userId string, guildId string, billIds []int, proofUrl string) (*PayMultipleResult, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}
	if proofUrl == "" {
		return nil, exception.ErrNoUrl
	}

	ids, err := dedupeBillIds(billIds)
	if err != nil {
		return nil, err
	}

	bills, err := s.loadOwnedBills(ctx, ids, userId, guildId)
	if err != nil {
		return nil, err
	}

	// One *models.Group per group id: several bills may share a group, and they
	// must all reduce the debt on the same instance or the last write wins.
	groups := make(map[int]*models.Group, len(bills))
	for _, b := range bills {
		if _, ok := groups[b.GroupID]; ok {
			continue
		}
		g, err := s.groupRepo.GetById(ctx, b.GroupID)
		if err != nil {
			return nil, err
		}
		groups[b.GroupID] = g
	}

	payee := groups[bills[0].GroupID]
	if err := assertSamePayee(payee, groups); err != nil {
		return nil, err
	}

	total := 0
	for _, b := range bills {
		total += b.AmountDue
	}

	vs, err := s.prepareSlip(ctx, proofUrl, payee.Payment)
	if err != nil {
		return nil, err
	}

	if vs.Amount+amountEpsilon < float64(total) {
		return nil, exception.ErrSlipInsufficient
	}

	now := time.Now().UTC()
	for _, b := range bills {
		b.UpdatedAt = now
		b.SubmittedAt = &now
		b.VerifiedAt = &now
		b.Status = models.BillStatusVerified
		b.ProofJSON = vs.ProofJSON

		// Credit each bill's own amount to its own group. Crediting the whole
		// slip amount per bill would count the payment once per bill.
		if err := applyPaymentToMember(groups[b.GroupID], userId, float64(b.AmountDue)); err != nil {
			return nil, err
		}
	}

	// An overpayment is credited against the payer's remaining debt, as Pay does
	// — but only when the batch is confined to one group. Across several groups
	// there is no principled way to split it, so it is left uncredited.
	var surplusCredited float64
	if surplus := vs.Amount - float64(total); surplus > amountEpsilon && len(groups) == 1 {
		if err := applyPaymentToMember(groups[bills[0].GroupID], userId, surplus); err != nil {
			return nil, err
		}
		surplusCredited = surplus
	}

	if err := s.persistPaymentBatch(ctx, bills, groups, vs.TransRef, now); err != nil {
		return nil, err
	}

	paid := make([]models.Bill, 0, len(bills))
	for _, b := range bills {
		paid = append(paid, *b)
	}

	return &PayMultipleResult{
		Bills:           paid,
		TotalDue:        total,
		AmountPaid:      vs.Amount,
		SurplusCredited: surplusCredited,
	}, nil
}

// dedupeBillIds removes duplicates while preserving order, and enforces the
// batch bounds.
func dedupeBillIds(billIds []int) ([]int, error) {
	seen := make(map[int]bool, len(billIds))
	ids := make([]int, 0, len(billIds))

	for _, id := range billIds {
		if id <= 0 {
			return nil, exception.ErrInvalidID
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, exception.ErrNoBillsSelected
	}
	if len(ids) > maxBatchBills {
		return nil, exception.ErrTooManyBills
	}
	return ids, nil
}

// loadOwnedBills fetches every requested bill and asserts the caller may pay all
// of them. The returned bills follow the requested order.
func (s *billService) loadOwnedBills(ctx context.Context, ids []int, userId string, guildId string) ([]*models.Bill, error) {
	found, err := s.repo.GetByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	byId := make(map[int]*models.Bill, len(found))
	for i := range found {
		byId[found[i].ID] = &found[i]
	}

	bills := make([]*models.Bill, 0, len(ids))
	for _, id := range ids {
		b, ok := byId[id]
		if !ok {
			return nil, exception.ErrBillNotFound
		}
		if b.Status == models.BillStatusVerified {
			return nil, exception.ErrBillAlreadyPaid
		}
		if b.MemberID != userId || b.GuildID != guildId {
			return nil, exception.ErrNotMatch
		}
		// Summing amounts across currencies would be meaningless.
		if len(bills) > 0 && b.Currency != bills[0].Currency {
			return nil, exception.ErrInvalidCurrency
		}
		bills = append(bills, b)
	}

	return bills, nil
}

// assertSamePayee requires every group to be owned by the same person and to be
// paid to the same account, since one slip has exactly one receiver.
func assertSamePayee(ref *models.Group, groups map[int]*models.Group) error {
	refAccount := helper.ExtractNumericCharacters(ref.Payment.Account)

	for _, g := range groups {
		if g.OwnerDiscordID != ref.OwnerDiscordID ||
			g.Payment.Method != ref.Payment.Method ||
			helper.ExtractNumericCharacters(g.Payment.Account) != refAccount {
			return exception.ErrMixedPayee
		}
	}
	return nil
}

// persistPaymentBatch atomically records the slip and updates every bill and
// group. All-or-nothing.
func (s *billService) persistPaymentBatch(
	ctx context.Context,
	bills []*models.Bill,
	groups map[int]*models.Group,
	transRef string,
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

	for _, b := range bills {
		if err := billTx.Update(ctx, b.ID, b); err != nil {
			return err
		}
	}

	// Sorted so the write order is deterministic.
	groupIds := make([]int, 0, len(groups))
	for id := range groups {
		groupIds = append(groupIds, id)
	}
	sort.Ints(groupIds)

	for _, id := range groupIds {
		if err := groupTx.Update(ctx, id, groups[id]); err != nil {
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
