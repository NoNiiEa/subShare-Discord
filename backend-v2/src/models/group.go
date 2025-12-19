package models

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
	Dept int `json:"dept"`
	Status MemberStatus `json:"status"`
	Payment PaymentStatus `json:"payment_status"`
}

type PaymentAccount struct {
	Method PaymentMethod `json:"method"`
	Account string `json:"account"`
}

type Group struct {
	ID int `json:"id"`
	Name string `json:"name"`
	Amount int `json:"amount"`
	AmountPerMember int `json:"amount_per_person"`
	DueDay int `json:"due_day"`
	Members []GroupMember `json:"members"`
	DiscordGuildID string `json:"discord_guild_id"`
	OwnerDiscordID string `json:"owner_discord_id"`
	Payment PaymentAccount `json:"payment"`
	CreatedAt time.Time `json:"create_at"`
}