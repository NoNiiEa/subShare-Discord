package exception

import (
	"errors"
)

var (
	ErrInvalidID       = errors.New("invalid group ID.")
	ErrInvalidYear     = errors.New("invalid year.")
	ErrInvalidMonth    = errors.New("invalid month.")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrInvalidCurrency = errors.New("invalid currency")
)

var (
	ErrNoUrl              = errors.New("empty url.")
	ErrBillNotFound       = errors.New("bill not found.")
	ErrNotMatch           = errors.New("bill does not match the information provided.")
	ErrWrongReciever      = errors.New("wrong receiver.")
	ErrBillAlreadyPaid    = errors.New("bill is already paid")
	ErrSlipExpired        = errors.New("payment slip is too old. Please upload the slip within 30 minutes of making the transfer.")
	ErrDupeSlip           = errors.New("duplicate payment slip")
	ErrSlipAmountMismatch = errors.New("slip amount does not match the bill")
)

var (
	ErrNoQRCode             = errors.New("No valid QR code could be found in the uploaded image")
	ErrSlipExpiredOrInvalid = errors.New("This QR code has expired or the transaction cannot be found.")
	ErrSlipImageInvalid     = errors.New("the uploaded file is not a valid slip image")
	ErrBankDataUnavailable  = errors.New("bank data temporarily unavailable")
	ErrSlipPendingBankDelay = errors.New("slip not yet confirmed by the bank")
	ErrTimeOut              = errors.New("the verification service is taking too long. Please try again later")
)
