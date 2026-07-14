package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/helper"
)

// parseIDParam reads an integer path parameter, writing a 400 and returning
// ok=false if it is missing or malformed.
func parseIDParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		helper.WriteError(w, http.StatusBadRequest, "invalid ID format")
		return 0, false
	}
	return id, true
}

// decodeJSON decodes the request body into dst, writing a 400 and returning
// false on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		helper.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

type errorMapping struct {
	sentinel error
	status   int
	message  string
}

// serviceErrorMap maps known sentinel errors to a safe HTTP status + message.
// Order matters only for readability — sentinels are distinct.
//
// NOTE: the /bill/pay messages are matched as substrings by the Discord bot
// (discord-bot/src/utils/errors.ts PAYMENT_ERROR_RULES). Do not change their
// wording without updating the bot.
var serviceErrorMap = []errorMapping{
	// Bill payment (bot-facing wording — keep stable)
	{exception.ErrBillNotFound, http.StatusNotFound, "bill not found"},
	{exception.ErrNotMatch, http.StatusForbidden, "bill does not belong to this user"},
	{exception.ErrWrongReciever, http.StatusBadRequest, "payment receiver does not match group settings"},
	{exception.ErrNoUrl, http.StatusBadRequest, "proof url is required"},
	{exception.ErrMemberNotFound, http.StatusNotFound, "member not found in group"},
	{exception.ErrBillAlreadyPaid, http.StatusBadRequest, "bill is already paid"},
	{exception.ErrSlipExpired, http.StatusBadRequest, "payment slip is too old. Please upload the slip within 30 minutes of making the transfer."},
	{exception.ErrNoQRCode, http.StatusBadRequest, "No valid QR code found in the image."},
	{exception.ErrSlipExpiredOrInvalid, http.StatusBadRequest, "The slip QR code has expired or is invalid."},
	{exception.ErrDupeSlip, http.StatusConflict, "This slip has already been used."},
	{exception.ErrSlipAmountMismatch, http.StatusBadRequest, "The amount in the slip does not match the record."},
	{exception.ErrSlipImageInvalid, http.StatusBadRequest, "Please upload a valid slip image (JPG/PNG)."},
	{exception.ErrBankDataUnavailable, http.StatusServiceUnavailable, "The bank's data is temporarily unavailable. Please try again in a few minutes."},
	{exception.ErrSlipPendingBankDelay, http.StatusServiceUnavailable, "This slip isn't confirmed by the bank yet. Please resubmit in a few minutes."},
	{exception.ErrTimeOut, http.StatusGatewayTimeout, "Verification timed out. Please try again."},

	// Batch bill payment (bot-facing wording — keep stable)
	{exception.ErrNoBillsSelected, http.StatusBadRequest, "no bills selected"},
	{exception.ErrTooManyBills, http.StatusBadRequest, "you can pay at most 25 bills at once"},
	{exception.ErrMixedPayee, http.StatusBadRequest, "selected bills must be paid to the same account"},
	{exception.ErrSlipInsufficient, http.StatusBadRequest, "slip amount is less than the total of the selected bills"},

	// Bill create validation
	{exception.ErrInvalidID, http.StatusBadRequest, "invalid id"},
	{exception.ErrInvalidYear, http.StatusBadRequest, "invalid year"},
	{exception.ErrInvalidMonth, http.StatusBadRequest, "invalid month"},
	{exception.ErrInvalidAmount, http.StatusBadRequest, "invalid amount"},
	{exception.ErrInvalidCurrency, http.StatusBadRequest, "invalid currency"},

	// Group
	{exception.ErrNotFound, http.StatusNotFound, "group not found"},
	{exception.ErrEmptyName, http.StatusBadRequest, "group name must not be empty"},
	{exception.ErrNegetiveAmount, http.StatusBadRequest, "amount must not be negative"},
	{exception.ErrInvalidDueDay, http.StatusBadRequest, "invalid due day"},
	{exception.ErrInvalidDiscordId, http.StatusBadRequest, "invalid discord id"},
	{exception.ErrNoMemberToinvite, http.StatusBadRequest, "no members provided to invite"},
	{exception.ErrAlreadyMember, http.StatusBadRequest, "user is already a member."},
	{exception.ErrNoPermission, http.StatusForbidden, "you do not have permission"},
	{exception.ErrNotInvited, http.StatusBadRequest, "user is not invited to this group"},
}

// writeServiceError maps a service error to a safe HTTP response. Unmapped
// errors are logged and returned as a generic 500 so internal/DB/provider
// details never reach the client.
func writeServiceError(w http.ResponseWriter, err error) {
	for _, m := range serviceErrorMap {
		if errors.Is(err, m.sentinel) {
			helper.WriteError(w, m.status, m.message)
			return
		}
	}
	log.Printf("[handler] unhandled error: %v", err)
	helper.WriteError(w, http.StatusInternalServerError, "internal server error")
}
