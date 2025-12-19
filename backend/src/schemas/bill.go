package schemas

type BillPayRequest struct {
	UserID string `json:"user_id"`
	GuildID string `json:"guild_id"`
	BillID int `json:"bill_id"`
	ProofURL string `json:"proof_url"`
}