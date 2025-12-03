package group

import (
	"time"
)

type MemberStatus string
type PaymentStatus string
type PaymentMethod string

const (
	MemberStatusActive MemberStatus = "Active"
	MemberStatusInvited MemberStatus = "Invited"
	MemberStatusLeft MemberStatus = "Left"

	PaymentStatusNotPaid PaymentStatus = "Not_Paid"
	PaymentStatusPaid PaymentStatus = "Paid"

	BankAccount PaymentMethod = "BANKAC"
	PromptPay PaymentMethod = "MSISDN"
)

type GroupMember struct {
	MemberID string `json:"member_id"`
	Dept int64 `json:"dept"`
	Status MemberStatus `json:"status"`
	Payment PaymentStatus `json:"payment_status"`
}

type PaymentAccount struct {
	Method PaymentMethod `json:"method"`
	Account string `json:"account"`
}

type Group struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Amount float64 `json:"amount"`
	AmountPerMember int64 `json:"amount_per_person"`
	DueDay int `json:"due_day"`
	Members []GroupMember `json:"members"`
	DiscordGuildID string `json:"discord_guild_id"`
	OwnerDiscordID string `json:"owner_discord_id"`
	Payment PaymentAccount `json:"payment"`
	CreateAt time.Time `json:"create_at"`
}

type CreateGroupRequest struct {
	Name           string   `json:"name"`
	Amount         float64  `json:"amount"`
	DueDay         int      `json:"due_day"`
	DiscordGuildID string   `json:"discord_guild_id"`
	OwnerDiscordID string   `json:"owner_discord_id"`
	Payment        PaymentAccount `json:"payment"`
}

type UpdateGroupRequest struct {
	Name           string   `json:"name"`
	Amount         float64  `json:"amount"`
	DueDay         int      `json:"due_day"`
	Members        []GroupMember `json:"members"`
	DiscordGuildID string   `json:"discord_guild_id"`
	OwnerDiscordID string   `json:"owner_discord_id"`
	Payment        PaymentAccount `json:"payment"`
}

type InviteGroupRequest struct {
	OwnerID string `json:"owner_id"`
	MemberIDs []string `json:"member_ids"`
}

type AcceptInviteRequest struct {
	UserID string `json:"user_id"`
}

type MarkAsPaidRequest struct {
	Amount int64 `json:"amount"`
}