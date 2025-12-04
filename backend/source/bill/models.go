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

	Year  int `json:"year"`  
	Month int `json:"month"`

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
}