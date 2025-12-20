package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	easyslip "github.com/NoNiiEa/subShare-Discord/src/easySlip"
	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/helper"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/repository"
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
	repo           repository.BillRepository
	groupRepo      repository.GroupRepository
	easySlipClient easyslip.EasySlipClient
}

func NewBillService(repo repository.BillRepository, groupRepo repository.GroupRepository, easySlipClient easyslip.EasySlipClient) BillService {
	return &billService{repo: repo, groupRepo: groupRepo, easySlipClient: easySlipClient}
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

	if err := s.repo.Create(ctx, b); err != nil {
		return err
	}

	return nil
}

func (s *billService) GetByGuildId(ctx context.Context, guildId string) ([]models.Bill, error) {
	return s.repo.GetByGuildId(ctx, guildId)
}

func (s *billService) GetByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	bills, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Bill

	for _, b := range bills {
		if b.MemberID == userId {
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
		return nil, err
	}
	if b == nil {
		return nil, exception.ErrBillNotFound
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
	if g == nil {
		return nil, exception.ErrGroupNotFound
	}

	billSlip, err := s.easySlipClient.CheckSlip(ctx, proofUrl)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if billSlip.Data.Date != "" {
		slipDate, err := parseSlipDate(billSlip.Data.Date)
		if err == nil {
			timeDiff := now.Sub(slipDate)
			if timeDiff > 30*time.Minute {
				return nil, exception.ErrSlipExpired
			}
		}
	}

	acc := billSlip.Data.Receiver.Account
	amountPaid := billSlip.Data.Amount.Amount
	var method models.PaymentMethod
	var slipAccount string

	if acc.Bank != nil {
		method = models.BankAccount
		slipAccount = acc.Bank.Account
	} else if acc.Proxy != nil {
		method = models.PromptPay
		slipAccount = acc.Proxy.Account
	} else {
		method = models.BankAccount
		if acc.Bank != nil {
			slipAccount = acc.Bank.Account
		}
	}
	slipAccount = helper.ExtractNumericCharacters(slipAccount)

	if g.Payment.Method != method || helper.Last4(g.Payment.Account) != helper.Last4(slipAccount) {
		return nil, exception.ErrWrongReciever
	}

	b.UpdatedAt = now
	b.SubmittedAt = &now
	b.Status = models.BillStatusVerified

	billSlipJson, err := json.Marshal(billSlip)
	if err != nil {
		return nil, err
	}
	b.ProofJSON = string(billSlipJson)

	if amountPaid < float64(b.AmountDue) {
		remainingAmount := b.AmountDue - int(amountPaid)

		remainingBill := models.Bill{
			GroupID:     g.ID,
			GuildID:     g.DiscordGuildID,
			MemberID:    userId,
			Year:        now.Year(),
			Month:       int(now.Month()),
			AmountDue:   remainingAmount,
			Currency:    "THB",
			Status:      models.BillStatusPending,
			Description: "Remaining balance (Underpaid)",
			ProofJSON:   "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := s.repo.Create(ctx, &remainingBill); err != nil {
			return nil, err
		}
	}

	memberFound := false
	for i := range g.Members {
		m := &g.Members[i]

		if m.MemberID == userId {
			debtReduction := b.AmountDue
			if debtReduction > m.Dept {
				debtReduction = m.Dept
			}
			m.Dept -= debtReduction

			if m.Dept < 0 {
				m.Dept = 0
			}

			if m.Dept <= 0 {
				m.Payment = models.PaymentStatusPaid
			}

			if m.Status == models.MemberStatusInvited {
				m.Status = models.MemberStatusActive
			}

			memberFound = true
			break
		}
	}

	if !memberFound {
		return nil, exception.ErrMemberNotFound
	}

	if err := s.repo.Update(ctx, billId, b); err != nil {
		return nil, err
	}

	if err := s.groupRepo.Update(ctx, g.ID, g); err != nil {
		return nil, err
	}

	return b, nil
}

func (s *billService) GetUnpaidByGuildId(ctx context.Context, guildId string) ([]models.Bill, error) {
	if guildId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	bills, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Bill

	for _, b := range bills {
		if b.Status != models.BillStatusVerified {
			res = append(res, b)
		}
	}

	return res, nil
}

func (s *billService) GetUnpaidByGuildIdAndUserId(ctx context.Context, guildId string, userId string) ([]models.Bill, error) {
	if guildId == "" || userId == "" {
		return nil, exception.ErrInvalidDiscordId
	}

	bills, err := s.repo.GetByGuildId(ctx, guildId)
	if err != nil {
		return nil, err
	}

	var res []models.Bill

	for _, b := range bills {
		if b.Status != models.BillStatusVerified && b.MemberID == userId {
			res = append(res, b)
		}
	}

	return res, nil
}

func parseSlipDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func (s *billService) DeleteById(ctx context.Context, billId int) error {
	if billId <= 0 {
		return exception.ErrInvalidID
	}

	return s.repo.DeleteById(ctx, billId)
}