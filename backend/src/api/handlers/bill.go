package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/helper"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/schemas"
	"github.com/NoNiiEa/subShare-Discord/src/service"
)

type BillHandler struct {
	service service.BillService
}

func NewBillHandler(s service.BillService) *BillHandler {
	return &BillHandler{service: s}
}

func (h *BillHandler) GetByGuildId(w http.ResponseWriter, r *http.Request) {
	guildId := r.PathValue("guildId")

	bills, err := h.service.GetByGuildId(r.Context(), guildId)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) GetByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	guildId := r.PathValue("guildId")

	bills, err := h.service.GetByGuildIdAndUserId(r.Context(), guildId, userId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid userId or GuildId.", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", 500)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) Pay(w http.ResponseWriter, r *http.Request) {
	var req schemas.BillPayRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	b, err := h.service.Pay(r.Context(), req.UserID, req.GuildID, req.BillID, req.ProofURL)

	if err != nil {
		switch {
		case errors.Is(err, exception.ErrBillNotFound):
			helper.WriteError(w, http.StatusNotFound, "bill not found")
		case errors.Is(err, exception.ErrNotMatch):
			helper.WriteError(w, http.StatusForbidden, "bill does not belong to this user")
		case errors.Is(err, exception.ErrWrongReciever):
			helper.WriteError(w, http.StatusBadRequest, "payment receiver does not match group settings")
		case errors.Is(err, exception.ErrNoUrl):
			helper.WriteError(w, http.StatusBadRequest, "proof url is required")
		case errors.Is(err, exception.ErrMemberNotFound):
			helper.WriteError(w, http.StatusNotFound, "member not found in group")
		case errors.Is(err, exception.ErrBillAlreadyPaid):
			helper.WriteError(w, http.StatusBadRequest, "bill is already paid")
		case errors.Is(err, exception.ErrSlipExpired):
			helper.WriteError(w, http.StatusBadRequest, "payment slip is too old. Please upload the slip within 30 minutes of making the transfer.")
		case errors.Is(err, exception.ErrDupeSlip):
			helper.WriteError(w, http.StatusBadRequest, "This Slip is already use to other bill.")
		default:
			helper.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	helper.WriteJSON(w, http.StatusOK, b)
}

func (h *BillHandler) GetUnpaidByGuildId(w http.ResponseWriter, r *http.Request) {
	guildId := r.PathValue("guildId")

	bills, err := h.service.GetUnpaidByGuildId(r.Context(), guildId)
	if err != nil {
		http.Error(w, "internal error", 500)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) GetUnpaidByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	guildId := r.PathValue("guildId")

	bills, err := h.service.GetUnpaidByGuildIdAndUserId(r.Context(), guildId, userId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid userId or GuildId.", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", 500)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.BillCreateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var b = models.Bill{
		GroupID: req.GroupID,
		GuildID: req.GuildID,
		MemberID: req.MemberID,
		Year: req.Year,
		Month: req.Month,
		AmountDue: req.AmountDue,
		Currency: req.Currency,
		Description: req.Description,
		Status: req.Status,
	}

	err := h.service.Create(r.Context(), &b)
	if err != nil {
	switch {
	case errors.Is(err, exception.ErrInvalidID),
		errors.Is(err, exception.ErrInvalidDiscordId),
		errors.Is(err, exception.ErrInvalidYear),
		errors.Is(err, exception.ErrInvalidMonth),
		errors.Is(err, exception.ErrInvalidAmount),
		errors.Is(err, exception.ErrInvalidCurrency):
			helper.WriteError(w, http.StatusBadRequest, err.Error())
	default:
			helper.WriteError(w, http.StatusInternalServerError, "Failed to create bill")
	}
	return
}

	helper.WriteJSON(w, http.StatusOK, b)
}
