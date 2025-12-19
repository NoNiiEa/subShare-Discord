package models

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
	ID int `json:"id"`

	GroupID int `json:"group_id"`
	GuildID string `json:"guild_id"`
	MemberID string `json:"member_id"`

	Year int `json:"year"`
	Month int `json:"month"`

	AmountDue int `json:"amount_due"`
	Currency string `json:"currency"`
	Status BillStatus `json:"status"`
	Description string `json:"description,omitempty"`

	ProofJSON   string `json:"proof_json"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
}