package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/helper"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/schemas"
	"github.com/NoNiiEa/subShare-Discord/src/service"
)

type GroupHandler struct {
	service service.GroupService
}

func NewGroupHandler(s service.GroupService) *GroupHandler {
	return &GroupHandler{service: s}
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateGroupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	group := models.Group{
		Name:           req.Name,
		Amount:         req.Amount,
		DueDay:         req.DueDay,
		DiscordGuildID: req.DiscordGuildID,
		OwnerDiscordID: req.OwnerDiscordID,
		Payment:        req.Payment,
	}

	g, err := h.service.CreateGroup(r.Context(), &group)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusCreated, g)
}

func (h *GroupHandler) GetById(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	g, err := h.service.GetById(r.Context(), groupId)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) GetByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.GetByGuildIdAndUserId(r.Context(), r.PathValue("guildId"), r.PathValue("userId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) GetByGuildIdAndUserIdOwn(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.GetByGuildIdAndUserIdOwn(r.Context(), r.PathValue("guildId"), r.PathValue("userId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) Invite(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	var req schemas.InviteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	g, err := h.service.InviteToGroup(r.Context(), groupId, req.OwnerId, req.MemberIds)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) GetPendingInvite(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.GetUserPendingInvite(r.Context(), r.PathValue("userId"), r.PathValue("guildId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	var req schemas.AorDInviteRequset
	if !decodeJSON(w, r, &req) {
		return
	}

	g, err := h.service.AcceptInvite(r.Context(), groupId, req.UserId)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	var req schemas.AorDInviteRequset
	if !decodeJSON(w, r, &req) {
		return
	}

	g, err := h.service.DeclineInvite(r.Context(), groupId, req.UserId)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) DeleteById(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	if err := h.service.DeleteById(r.Context(), groupId, r.PathValue("userId")); err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, nil)
}

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	groupId, ok := parseIDParam(w, r, "groupId")
	if !ok {
		return
	}

	var req schemas.UpdateGroupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	groupUpdate := &models.Group{
		Name:    req.Name,
		Amount:  req.Amount,
		DueDay:  req.DueDay,
		Payment: req.Payment,
	}

	if err := h.service.Update(r.Context(), groupId, groupUpdate); err != nil {
		writeServiceError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "group updated successfully",
	})
}

func (h *GroupHandler) DailyPaymentReset(w http.ResponseWriter, r *http.Request) {
	day := time.Now().Day()

	log.Printf("Starting bill cycle for day %d", day)
	groups, err := h.service.GetByDueDay(r.Context(), day)
	if err != nil {
		log.Printf("Error fetching groups for day %d: %v", day, err)
		helper.WriteError(w, http.StatusInternalServerError, "failed to run payment reset")
		return
	}

	failed := 0
	for _, g := range groups {
		if err := h.service.CreateBillCycle(r.Context(), &g); err != nil {
			failed++
			log.Printf("Failed to create bill cycle for Group ID %d: %v", g.ID, err)
		}
	}

	log.Printf("Completed bill cycle for day %d (%d groups, %d failed)", day, len(groups), failed)
	helper.WriteJSON(w, http.StatusOK, map[string]any{
		"message":   "Completed bill cycle",
		"processed": len(groups),
		"failed":    failed,
	})
}
