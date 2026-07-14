package schemas

import (
	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type BillCreateRequest struct {
	GroupID     int               `json:"group_id"`
	GuildID     string            `json:"guild_id"`
	MemberID    string            `json:"member_id"`
	Year        int               `json:"year"`
	Month       int               `json:"month"`
	AmountDue   int               `json:"amount_due"`
	Currency    string            `json:"currency"`
	Description string            `json:"description,omitempty"`
	Status      models.BillStatus `json:"status,omitempty"`
}

type BillPayRequest struct {
	UserID   string `json:"user_id"`
	GuildID  string `json:"guild_id"`
	BillID   int    `json:"bill_id"`
	ProofURL string `json:"proof_url"`
}

type BillPayMultipleRequest struct {
	UserID   string `json:"user_id"`
	GuildID  string `json:"guild_id"`
	BillIDs  []int  `json:"bill_ids"`
	ProofURL string `json:"proof_url"`
}
