package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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
	slipRepo	   repository.SlipRepository
	easySlipClient easyslip.EasySlipClient
	okSlipClient   okslip.OkSlipClient
}

func NewBillService(
	repo repository.BillRepository, 
	groupRepo repository.GroupRepository, 
	slipRepo repository.SlipRepository,
	easySlipClient easyslip.EasySlipClient, 
	okSlipClient okslip.OkSlipClient,
	) BillService {
	return &billService{repo: repo, groupRepo: groupRepo, easySlipClient: easySlipClient, okSlipClient: okSlipClient, slipRepo: slipRepo}
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

    billSlip, err := s.okSlipClient.CheckSlip(ctx, proofUrl)
    if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
        	return nil, exception.ErrTimeOut
    	}

        var apiErr *okslip.APIErrorResponse
        if errors.As(err, &apiErr) {
            switch apiErr.Code {
            case 1007: // No QR Code
                return nil, exception.ErrNoQRCode
            case 1011: // QR Code Expired or Invalid
                return nil, exception.ErrSlipExpiredOrInvalid
            case 1012: // Duplicate Slip detected by Provider
                return nil, exception.ErrDupeSlip
            default:
                // Log the unhandled code and return a generic payment error
                log.Printf("Unhandled OkSlip Code: %d - %s", apiErr.Code, apiErr.Message)
                return nil, fmt.Errorf("payment provider error: %s", apiErr.Message)
            }
        }
        return nil, err
    }

    now := time.Now().UTC()
    timeDiff := now.Sub(billSlip.Data.TransTimestamp)
    if timeDiff > 30*time.Minute {
        return nil, exception.ErrSlipExpired
    }

    acc := billSlip.Data.Receiver.Account
    proxy := billSlip.Data.Receiver.Proxy
    
    amountPaid := billSlip.Data.Amount 
    var method models.PaymentMethod
    var slipAccount string

    if acc.Type == "BANKAC" {
        method = models.BankAccount
        slipAccount = acc.Value
    } else if proxy.Type != "" { 
        method = models.PromptPay
        slipAccount = proxy.Value
    } else {
        method = models.BankAccount
        slipAccount = acc.Value
    }

    slipDigits := helper.ExtractNumericCharacters(slipAccount)
    
    dbDigits := helper.ExtractNumericCharacters(g.Payment.Account)

    if len(slipDigits) == 0 {
        return nil, errors.New("could not extract digits from slip")
    }

    match := strings.Contains(dbDigits, slipDigits)

    if g.Payment.Method != method || !match {
        return nil, exception.ErrWrongReciever
    }

	existingSlip, err := s.slipRepo.GetByTransRef(ctx, billSlip.Data.TransRef)
    if err != nil {
        return nil, err
    }
    if existingSlip != nil {
        return nil, exception.ErrDupeSlip
    }

    amountPaid = billSlip.Data.Amount 
    isUnderpaid := amountPaid < float64(b.AmountDue)

    b.UpdatedAt = now
    b.SubmittedAt = &now
    b.Status = models.BillStatusVerified
    
    billSlipJson, err := json.Marshal(billSlip)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize slip data: %w", err)
	}

    b.ProofJSON = string(billSlipJson)

    memberFound := false
    for i := range g.Members {
        m := &g.Members[i]
        if m.MemberID == userId {
            actualReduction := int(amountPaid)
            if actualReduction > m.Dept {
                actualReduction = m.Dept
            }
            m.Dept -= actualReduction
            
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

    if isUnderpaid {
        remainingAmount := b.AmountDue - int(amountPaid)
        remainingBill := models.Bill{
            GroupID:     g.ID,
            GuildID:     g.DiscordGuildID,
            MemberID:    userId,
            Year:        b.Year,
            Month:       b.Month,
            AmountDue:   remainingAmount,
            Currency:    "THB",
            Status:      models.BillStatusPending,
            Description: fmt.Sprintf("Balance remaining from Bill #%d", b.ID),
        }
        if err := s.repo.Create(ctx, &remainingBill); err != nil {
            return nil, err
        }
    }

    if err := s.repo.Update(ctx, billId, b); err != nil {
        return nil, err
    }
    if err := s.groupRepo.Update(ctx, g.ID, g); err != nil {
        return nil, err
    }
    
    slip := models.Slip{
        TransRef:    billSlip.Data.TransRef,
        SubmittedAt: &now,
    }
    if err := s.slipRepo.Create(ctx, &slip); err != nil {
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