package handlers

import (
	"net/http"

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
	bills, err := h.service.GetByGuildId(r.Context(), r.PathValue("guildId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) GetByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	bills, err := h.service.GetByGuildIdAndUserId(r.Context(), r.PathValue("guildId"), r.PathValue("userId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) Pay(w http.ResponseWriter, r *http.Request) {
	var req schemas.BillPayRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	b, err := h.service.Pay(r.Context(), req.UserID, req.GuildID, req.BillID, req.ProofURL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, b)
}

func (h *BillHandler) GetUnpaidByGuildId(w http.ResponseWriter, r *http.Request) {
	bills, err := h.service.GetUnpaidByGuildId(r.Context(), r.PathValue("guildId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) GetUnpaidByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	bills, err := h.service.GetUnpaidByGuildIdAndUserId(r.Context(), r.PathValue("guildId"), r.PathValue("userId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.BillCreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	b := models.Bill{
		GroupID:     req.GroupID,
		GuildID:     req.GuildID,
		MemberID:    req.MemberID,
		Year:        req.Year,
		Month:       req.Month,
		AmountDue:   req.AmountDue,
		Currency:    req.Currency,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := h.service.Create(r.Context(), &b); err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusCreated, b)
}
