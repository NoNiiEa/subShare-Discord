package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/exception"
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
	
	// Decode JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	group := models.Group{
		Name: req.Name,
		Amount: req.Amount,
		DueDay: req.DueDay,
		DiscordGuildID: req.DiscordGuildID,
		OwnerDiscordID: req.OwnerDiscordID,
		Payment: req.Payment,
	}

	g, err := h.service.CreateGroup(r.Context(), &group); 
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) GetById(w http.ResponseWriter, r *http.Request) {
	groupIdStr := r.PathValue("groupId")

	groupId, err := strconv.Atoi(groupIdStr)
    if err != nil {
        http.Error(w, "Invalid ID format", http.StatusBadRequest)
        return
    }

	g, err := h.service.GetById(r.Context(), groupId)
	if err != nil {
		if errors.Is(err, exception.ErrNotFound) {
			http.Error(w, "group not found.", http.StatusNotFound)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) GetByGuildIdAndUserId(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	guildId := r.PathValue("guildId")

	groups, err := h.service.GetByGuildIdAndUserId(r.Context(), guildId, userId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid memberId or guildId.", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) GetByGuildIdAndUserIdOwn(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	guildId := r.PathValue("guildId")

	groups, err := h.service.GetByGuildIdAndUserId(r.Context(), guildId, userId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid memberId or guildId.", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) Invite(w http.ResponseWriter, r *http.Request) {
	groupIdStr := r.PathValue("groupId")

	groupId, err := strconv.Atoi(groupIdStr)
    if err != nil {
        http.Error(w, "Invalid ID format", http.StatusBadRequest)
        return
    }

	var req schemas.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := h.service.InviteToGroup(r.Context(), groupId, req.OwnerId, req.MemberIds)

	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrNoMemberToinvite) {
			http.Error(w, "empty list of user to invite", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrAlreadyMember) {
			http.Error(w, "user is already a member.", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrNoPermission) {
			http.Error(w, "user have no permission.", http.StatusForbidden)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, g)
}

func (h *GroupHandler) GetPendingInvite(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	guildId := r.PathValue("guildId")

	groups, err := h.service.GetUserPendingInvite(r.Context(), userId, guildId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid memberId or guildId.", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	groupIdStr := r.PathValue("groupId")

	groupId, err := strconv.Atoi(groupIdStr)
    if err != nil {
        http.Error(w, "Invalid ID format", http.StatusBadRequest)
        return
    }

	var req schemas.AorDInviteRequset
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := h.service.AcceptInvite(r.Context(), groupId, req.UserId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrAlreadyMember) {
			http.Error(w, "user is already a member.", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrNotInvited) {
			http.Error(w, "user is not invite to this group", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, 200, g)
}

func (h *GroupHandler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	groupIdStr := r.PathValue("groupId")

	groupId, err := strconv.Atoi(groupIdStr)
    if err != nil {
        http.Error(w, "Invalid ID format", http.StatusBadRequest)
        return
    }

	var req schemas.AorDInviteRequset
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := h.service.DeclineInvite(r.Context(), groupId, req.UserId)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidDiscordId) {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrAlreadyMember) {
			http.Error(w, "user is already a member.", http.StatusBadRequest)
			return
		}

		if errors.Is(err, exception.ErrNotInvited) {
			http.Error(w, "user is not invite to this group", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, 200, g)
}

func (h *GroupHandler) DeleteById(w http.ResponseWriter, r *http.Request) {
	groupIdStr := r.PathValue("groupId")

	groupId, err := strconv.Atoi(groupIdStr)
	if err != nil {
		http.Error(w, "invalid ID format", http.StatusBadRequest)
		return
	}

	userId := r.PathValue("userId")

	err = h.service.DeleteById(r.Context(), groupId, userId)
	if err != nil {
		if errors.Is(err, exception.ErrNotFound) {
			http.Error(w, "group is not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	helper.WriteJSON(w, http.StatusOK, nil)
}

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
    groupIdStr := r.PathValue("groupId")

    groupId, err := strconv.Atoi(groupIdStr)
    if err != nil {
        helper.WriteError(w, http.StatusBadRequest, "invalid ID format")
        return
    }
    var req schemas.UpdateGroupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        helper.WriteError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }

    groupUpdate := &models.Group{
        Name:    req.Name,
        Amount:  req.Amount,
        DueDay:  req.DueDay,
        Payment: req.Payment,
    }

    if err := h.service.Update(r.Context(), groupId, groupUpdate); err != nil {
        switch {
        case errors.Is(err, exception.ErrGroupNotFound):
            helper.WriteError(w, http.StatusNotFound, err.Error())
        case errors.Is(err, exception.ErrEmptyName), errors.Is(err, exception.ErrInvalidDueDay):
            helper.WriteError(w, http.StatusBadRequest, err.Error())
        default:
            helper.WriteError(w, http.StatusInternalServerError, "internal server error")
        }
        return
    }

    helper.WriteJSON(w, http.StatusOK, map[string]string{
        "message": "group updated successfully",
    })
}

func (h *GroupHandler) DailyPaymentReset(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
    day := now.Day()

	log.Printf("Starting bill cycle for day %d", day)
	groups, err := h.service.GetByDueDay(r.Context(), day)
	if err != nil {
		log.Printf("Error fetching groups for day %d: %v", day, err)
		helper.WriteError(w, 500, "There is error reset payment")
		return
	}

	for _, g := range(groups) {
		if err := h.service.CreateBillCycle(r.Context(), &g); err != nil {
			log.Printf("Failed to create bill cycle for Group ID %d: %v", g.ID, err)
			helper.WriteError(w, 500, "Failed to create bill cycle for Group")
			return
		}
	}

	log.Printf("Completed bill cycle for day %d", day)
	helper.WriteJSON(w, http.StatusOK, map[string]string{
        "message": "Completed bill cycle",
    })
}