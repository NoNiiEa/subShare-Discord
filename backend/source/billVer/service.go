package billver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/NoNiiEa/subShare-Discord/source/group"
	"github.com/NoNiiEa/subShare-Discord/source/bill"
)

type Store interface {
	GetBillByID(ctx context.Context, id int64) (*bill.Bill, error)
	UpdateBill(ctx context.Context, b bill.Bill) (*bill.Bill, error)
	GetGroup(ctx context.Context, id int64) (*group.Group, error)
}

type Service struct {
	store Store
	httpClient *http.Client
	easySlipBaseURL string
	easySlipToken string
}

func NewService(store Store, httpClient *http.Client, easySlipBaseURL string, easySlipToken string) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Service{
		store: store,
		httpClient: httpClient,
		easySlipBaseURL: easySlipBaseURL,
		easySlipToken: easySlipToken,
	}
}

func (s *Service) callEasySlipVerify(ctx context.Context, imageByte []byte, filename string) (*SlipVerificationResult, error) {
	if s.easySlipBaseURL == "" || s.easySlipToken == "" {
		return nil, ErrConfigNotSet
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(imageByte); err != nil {
		return nil, err
	}

	_ = writer.WriteField("checkDuplicate", "false")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := s.easySlipBaseURL + "/verify"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.easySlipToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("easyslip error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var parsed easySlipResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}

	acc := parsed.Data.Receiver.Account
	var method group.PaymentMethod
	var account string
	switch {
	case acc.Bank != nil:
    // bank transfer
		method = group.BankAccount
		account = acc.Bank.Account
	case acc.Proxy != nil:
    // transferred via Proxy (PromptPay / phone / NATID)
		method = group.PromptPay
		account = acc.Proxy.Account
	default:
    // neither present (rare, possibly invalid slip)
		method = group.BankAccount
		account = acc.Bank.Account
	}

	// Build your internal verification result
	res := &SlipVerificationResult{
		IsValid:         parsed.Status == 200,
		MatchedAmount:   parsed.Data.Amount.Amount,
		Method: method,
		Account: account,
		RawResponse:     json.RawMessage(bodyBytes),
	}

	return res, nil

}

func (s *Service) SubmitBillProof(ctx context.Context, req SubmitBillProofRequest) (*bill.Bill, *SlipVerificationResult, error) {
	if req.BillID <= 0 {
		return nil, nil, bill.ErrInvalidBillID
	}

	if req.MemberID == "" {
		return nil, nil, group.ErrInvalidMemberID
	}

	if len(req.ImageBytes) == 0 {
		return nil, nil, ErrSlipTooSmall
	}

	b, err := s.store.GetBillByID(ctx, req.BillID)
	if err != nil {
		return nil, nil, err
	}

	if b.MemberID != req.MemberID {
		return nil, nil, ErrBillMemberMismatch
	}

	if b.Status == bill.BillStatusVerified || b.Status == bill.BillStatusCanceled {
		return nil, nil, ErrBillAlreadyVerified
	}

	verResult, err := s.callEasySlipVerify(ctx, req.ImageBytes, req.FileName)
	if err != nil {
		return nil, nil, err
	}

	amountFromSlip := verResult.MatchedAmount

	now := time.Now().UTC()
	b.AmountPaid = amountFromSlip
	b.UpdatedAt = now
	b.SubmittedAt = &now

	if len(verResult.RawResponse) > 0 {
		b.ProofJSON = string(verResult.RawResponse)
	}

	g, err := s.store.GetGroup(ctx, b.GroupID)
	if err != nil {
		return nil, nil, err
	}

	accStr := extractNumericCharacters(verResult.Account)

	if g.Payment.Method != verResult.Method || last4(g.Payment.Account) != last4(accStr) {
		return nil, nil, ErrWrongReciever
	}

	if verResult.IsValid && amountFromSlip+0.001 >= b.AmountDue {
		b.Status = bill.BillStatusVerified
		b.VerifiedAt = &now
	} else if !verResult.IsValid {
		b.Status = bill.BillStatusRejected
		b.RejectedAt = &now
	} else {
		// valid slip but underpaid
		b.Status = bill.BillStatusSubmitted
	}

	updated, err := s.store.UpdateBill(ctx, *b)
	if err != nil {
		return nil, nil, err
	}

	if !verResult.IsValid {
		return updated, verResult, ErrVerificationFailed
	}

	return updated, verResult, nil	
}

// func (s *Service) SubmitBillProof(ctx context.Context, req SubmitBillProofRequest) (*Bill, *SlipVerificationResult, error) {
// 	if req.BillID <= 0 {
// 		return nil, nil, errors.New("invalid bill_id")
// 	}
// 	if req.MemberID == "" {
// 		return nil, nil, ErrInvalidMemberID
// 	}
// 	if len(req.ImageBytes) == 0 {
// 		return nil, nil, ErrSlipTooSmall
// 	}

// 	// 1) Load bill
// 	b, err := s.store.GetBillByID(ctx, req.BillID)
// 	if err != nil {
// 		return nil, nil, ErrBillNotFound
// 	}
// 	if b.MemberID != req.MemberID {
// 		return nil, nil, ErrBillMemberMismatch
// 	}
// 	if b.Status == BillStatusVerified || b.Status == BillStatusRejected || b.Status == BillStatusCanceled {
// 		return nil, nil, ErrBillAlreadyVerified
// 	}

// 	verResult, err := s.callEasySlipVerify(ctx, req.ImageBytes, req.FileName)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	amountFromSlip := verResult.MatchedAmount

// 	now := time.Now().UTC()
// 	b.AmountPaid = amountFromSlip
// 	b.UpdatedAt = now
// 	b.SubmittedAt = &now

// 	if len(verResult.RawResponse) > 0 {
// 		b.ProofJSON = string(verResult.RawResponse)
// 	}

// 	if verResult.IsValid && amountFromSlip+0.001 >= b.AmountDue {
// 		b.Status = BillStatusVerified
// 		b.VerifiedAt = &now
// 	} else if !verResult.IsValid {
// 		b.Status = BillStatusRejected
// 		b.RejectedAt = &now
// 	} else {
// 		// valid slip but underpaid
// 		b.Status = BillStatusSubmitted
// 	}

// 	updated, err := s.store.UpdateBill(ctx, *b)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if !verResult.IsValid {
// 		return updated, verResult, ErrVerificationFailed
// 	}

// 	return updated, verResult, nil
// }

