package bill

import "errors"

var (
	ErrInvalidGroupID  = errors.New("invalid group_id")
	ErrInvalidMemberID = errors.New("invalid member_id")
	ErrInvalidBillID = errors.New("invalid bill_id")
	ErrInvalidYear     = errors.New("invalid year")
	ErrInvalidMonth    = errors.New("invalid month")
	ErrInvalidAmount   = errors.New("amount_due must be > 0")
	ErrInvalidCurrency = errors.New("currency is required")
)

var (
	ErrSlipTooSmall = errors.New("slip too small")
	ErrBillNotFound = errors.New("bill not found")
	ErrBillMemberMismatch = errors.New("bill and member mismatch")
	ErrBillAlreadyVerified = errors.New("bill is already verified")
	ErrVerificationFailed = errors.New("slip is not valid")
)