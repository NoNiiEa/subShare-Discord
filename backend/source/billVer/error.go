package billver

import "errors"

var (
	ErrConfigNotSet = errors.New("config not set")
	ErrSlipTooSmall = errors.New("Slip too small")
	ErrBillMemberMismatch = errors.New("bill and member mismatch")
	ErrBillAlreadyVerified = errors.New("bill is already verified")
	ErrVerificationFailed = errors.New("verification failed")
	ErrWrongReciever = errors.New("wrong reciever in slip")
)