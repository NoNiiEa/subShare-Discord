package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	easyslip "github.com/NoNiiEa/subShare-Discord/src/easySlip"
	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/helper"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/okslip"
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
	okSlipClient   okslip.OkSlipClient
}

func NewBillService(repo repository.BillRepository, groupRepo repository.GroupRepository, easySlipClient easyslip.EasySlipClient, okSlipClient okslip.OkSlipClient) BillService {
	return &billService{repo: repo, groupRepo: groupRepo, easySlipClient: easySlipClient, okSlipClient: okSlipClient}
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

    // 1. Get Bill and Group Data
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
		if errors.Is(err, exception.ErrNotFound) {
			return nil, exception.ErrGroupNotFound
		}
		return nil, err
	}

    // 2. Call the NEW okSlipClient
    billSlip, err := s.okSlipClient.CheckSlip(ctx, proofUrl)
    if err != nil {
        return nil, err
    }

    // 3. Validate Slip Freshness (using the ISO timestamp from your documentation)
    now := time.Now().UTC()
    // Using TransTimestamp from the OkSlip response we defined earlier
    timeDiff := now.Sub(billSlip.Data.TransTimestamp)
    if timeDiff > 30*time.Minute {
        return nil, exception.ErrSlipExpired
    }

    // 4. Extract Receiver Details for Verification
    // Based on documentation: receiver.account.type (BANKAC, TOKEN)
    acc := billSlip.Data.Receiver.Account
    proxy := billSlip.Data.Receiver.Proxy
    
    amountPaid := billSlip.Data.Amount // Data.Amount is float64 in our struct
    var method models.PaymentMethod
    var slipAccount string

    if acc.Type == "BANKAC" {
        method = models.BankAccount
        slipAccount = acc.Value
    } else if proxy.Type != "" { // If there's a proxy (MSISDN, NATID, etc)
        method = models.PromptPay
        slipAccount = proxy.Value
    } else {
        // Fallback or default
        method = models.BankAccount
        slipAccount = acc.Value
    }

    slipAccount = helper.ExtractNumericCharacters(slipAccount)

    // Verify if the receiver on the slip matches the Group's registered payment info
    // Note: okSlip returns masked accounts (xxx-x-x0209-x), so we compare Last4
    if g.Payment.Method != method || helper.Last4(g.Payment.Account) != helper.Last4(slipAccount) {
        return nil, exception.ErrWrongReciever
    }

    // 5. Update Bill Status
    b.UpdatedAt = now
    b.SubmittedAt = &now
    b.Status = models.BillStatusVerified

    billSlipJson, err := json.Marshal(billSlip)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize slip data: %w", err)
	}
    b.ProofJSON = string(billSlipJson)

    // 6. Handle Underpayment
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
            CreatedAt:   now,
            UpdatedAt:   now,
        }
        if err := s.repo.Create(ctx, &remainingBill); err != nil {
    		return nil, fmt.Errorf("failed to create underpayment bill: %w", err)
		}
    }

    // 7. Reduce Member Debt
    memberFound := false
    for i := range g.Members {
        m := &g.Members[i]
        if m.MemberID == userId {
            debtReduction := b.AmountDue
            if debtReduction > m.Dept {
                debtReduction = m.Dept
            }
            m.Dept -= debtReduction
            
            if m.Dept <= 0 {
                m.Dept = 0
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

    // 8. Atomic-like updates (ideally these should be in a transaction)
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