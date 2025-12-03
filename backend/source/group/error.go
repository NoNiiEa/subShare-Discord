package group

import (
	"errors"
)

var (
	ErrInvalidName       = errors.New("group name is required")
	ErrInvalidAmount     = errors.New("amount must be > 0")
	ErrInvalidDueDay     = errors.New("due_day must be between 1 and 31")
	ErrInvalidGuildID    = errors.New("discord_guild_id is required")
	ErrInvalidOwnerID    = errors.New("owner_discord_id is required")
	ErrInvalidMemberID   = errors.New("invalid member id")
	ErrNoMembersProvided = errors.New("at least one member is required")
	ErrInvalidGroupID    = errors.New("invalid group_id")
	ErrMemberNotFound   = errors.New("member is not found in group")
	ErrNotActiveMember   = errors.New("member is not active in group")
	ErrAlreadyPaid       = errors.New("member is already paid")
)

var (
	ErrAleadyInvited = errors.New("User is aleady invited")
	ErrAleadyMembered = errors.New("User is aleady Member")
	ErrInvitedPermission = errors.New("User have no permission to invite")
)

var (
	ErrNotInvited = errors.New("User is not invited")
)

var ErrNoUserID = errors.New("userID is required")
var ErrNotValidSlip = errors.New("invalid slip")