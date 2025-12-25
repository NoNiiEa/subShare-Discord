package exception

import (
	"errors"
)

var (
	ErrInvalidID       = errors.New("invalid group ID.")
	ErrInvalidYear     = errors.New("invalid year.")
	ErrInvalidMonth    = errors.New("invalid month.")
	ErrInvalidAmount   = errors.New("invalid amonth")
	ErrInvalidCurrency = errors.New("invalid currency")
)

var (
	ErrNoUrl           = errors.New("empty url.")
	ErrBillNotFound    = errors.New("bill not found.")
	ErrNotMatch        = errors.New("bill not match information that provide.")
	ErrWrongReciever   = errors.New("worng reciever.")
	ErrBillAlreadyPaid = errors.New("bill is already paid")
	ErrSlipExpired     = errors.New("payment slip is too old. Please upload the slip within 30 minutes of making the transfer.")
	ErrDupeSlip		   = errors.New("bill is dupe")
)
