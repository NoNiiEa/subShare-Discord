package schemas

import (
	"github.com/NoNiiEa/subShare-Discord/src/models"
)

type CreateGroupRequest struct {
	Name           string   `json:"name"`
	Amount         int  `json:"amount"`
	DueDay         int      `json:"due_day"`
	DiscordGuildID string   `json:"discord_guild_id"`
	OwnerDiscordID string   `json:"owner_discord_id"`
	Payment        models.PaymentAccount `json:"payment"`
}

type UpdateGroupRequest struct {
	Name           string   `json:"name"`
	Amount         *int  `json:"amount"`
	DueDay         *int      `json:"due_day"`
	Payment        models.PaymentAccount `json:"payment"`
}

type InviteRequest struct {
	OwnerId string `json:"owner_id"`
	MemberIds []string `json:"member_ids"`
}

type AorDInviteRequset struct {
	UserId string `json:"user_id"`
}