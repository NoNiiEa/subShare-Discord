package exception

import (
	"errors"
)

var (
	ErrNotFound = errors.New("group not found.")
)

var (
	ErrEmptyName = errors.New("group name must not be empty.")
	ErrNegetiveAmount = errors.New("amount must not be negative.")
	ErrInvalidDueDay = errors.New("invalid due day.")
	ErrInvalidDiscordId = errors.New("invalid discord id.")
	ErrNoMemberToinvite = errors.New("user don't provide member to invite")
	ErrAlreadyMember = errors.New("user is already a member.")
	ErrNoPermission = errors.New("user have no permission")
	ErrNotInvited = errors.New("user is not invite to this group.")
	ErrMemberNotFound = errors.New("member is not found")
)