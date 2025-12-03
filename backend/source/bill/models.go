package bill

import (
	"time"

)

type BillStatus string

const (
	BillStatusPending   BillStatus = "pending"   // waiting for user to submit proof
	BillStatusSubmitted BillStatus = "submitted" // user submitted proof, waiting review
	BillStatusVerified  BillStatus = "verified"  // owner/admin accepted
	BillStatusRejected  BillStatus = "rejected"  // owner/admin rejected
	BillStatusCanceled  BillStatus = "canceled"  // group or user canceled it
)

type Bill struct {
	ID int64 `json:"id"`

	GroupID  int64  `json:"group_id"`  // links to Group.ID
	MemberID string `json:"member_id"` // Discord user ID of the member

	// Billing cycle (e.g. March 2026)
	Year  int `json:"year"`  // e.g. 2026
	Month int `json:"month"` // 1–12

	AmountDue   float64 `json:"amount_due"`   // how much this member should pay
	AmountPaid  float64 `json:"amount_paid"`  // how much they claimed to pay
	Currency    string  `json:"currency"`     // "THB", "USD", etc.
	Status      BillStatus `json:"status"`    // pending/submitted/verified/rejected/...
	Description string     `json:"description,omitempty"` // optional note like "Netflix March"

	// Proof & verification
	ProofJSON   string `json:"proof_json"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
}

type CreateBillRequest struct {
	GroupID    int64   `json:"group_id"`
	MemberID   string  `json:"member_id"`
	Year       int     `json:"year"`
	Month      int     `json:"month"`
	AmountDue  float64 `json:"amount_due"`
	Currency   string  `json:"currency"`
	Description string `json:"description,omitempty"`
	// you can add fields if you really want to allow setting AmountPaid at creation,
	// but usually it starts at 0 and becomes > 0 only when user pays.
}

// type easySlipResponse struct {
// 	Status int `json:"status"`
// 	Data   struct {
// 		Payload     string `json:"payload"`
// 		TransRef    string `json:"transRef"`
// 		Date        string `json:"date"`
// 		CountryCode string `json:"countryCode"`

// 		Amount struct {
// 			Amount float64 `json:"amount"`
// 			Local  struct {
// 				Amount   float64 `json:"amount"`
// 				Currency string  `json:"currency"`
// 			} `json:"local"`
// 		} `json:"amount"`

// 		Fee float64 `json:"fee"`

// 		Ref1 string `json:"ref1"`
// 		Ref2 string `json:"ref2"`
// 		Ref3 string `json:"ref3"`

// 		Sender struct {
// 			Bank struct {
// 				ID    string `json:"id"`
// 				Name  string `json:"name"`
// 				Short string `json:"short"`
// 			} `json:"bank"`
// 			Account struct {
// 				Name struct {
// 					Th string `json:"th"`
// 					En string `json:"en"`
// 				} `json:"name"`

// 				Bank *struct {
// 					Type    string `json:"type"`    // "BANKAC" | "TOKEN" | "DUMMY"
// 					Account string `json:"account"`
// 				} `json:"bank,omitempty"`

// 				Proxy *struct {
// 					Type    string `json:"type"`    // "NATID" | "MSISDN" | "EWALLETID" | "EMAIL" | "BILLERID"
// 					Account string `json:"account"`
// 				} `json:"proxy,omitempty"`
// 			} `json:"account"`
// 		} `json:"sender"`

// 		Receiver struct {
// 			Bank struct {
// 				ID    string `json:"id"`
// 				Name  string `json:"name"`
// 				Short string `json:"short"`
// 			} `json:"bank"`
// 			Account struct {
// 				Name struct {
// 					Th string `json:"th"`
// 					En string `json:"en"`
// 				} `json:"name"`

// 				Bank *struct {
// 					Type    string `json:"type"`    // "BANKAC" | "TOKEN" | "DUMMY"
// 					Account string `json:"account"`
// 				} `json:"bank,omitempty"`

// 				Proxy *struct {
// 					Type    string `json:"type"`    // "NATID" | "MSISDN" | "EWALLETID" | "EMAIL" | "BILLERID"
// 					Account string `json:"account"`
// 				} `json:"proxy,omitempty"`
// 			} `json:"account"`

// 			MerchantID string `json:"merchantId"`
// 		} `json:"receiver"`
// 	} `json:"data"`
// }

// type SlipVerificationResult struct {
// 	IsValid bool `json:"is_valid"`
// 	MatchedAmount float64 `json:"matched_amount"`
// 	Method PaymentMethod `json:"method"`
// 	Account string `json:"account"`
// 	RawResponse []byte `json:"raw_response"`
// }

// type SubmitBillProofRequest struct {
// 	BillID     int64   `json:"bill_id"`
// 	MemberID   string  `json:"member_id"`
// 	AmountPaid float64 `json:"amount_paid"` // user-claimed, optional
// 	ImageBytes []byte  `json:"-"`
// 	FileName   string  `json:"-"` // "slip.jpg"
// }