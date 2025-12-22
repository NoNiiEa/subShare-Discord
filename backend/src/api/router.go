package api

import (
	"net/http"
	"github.com/NoNiiEa/subShare-Discord/src/api/handlers"
	"github.com/NoNiiEa/subShare-Discord/src/api/middleware"
)

// NewRouter registers all routes and returns the multiplexer
func NewRouter(groupHandler *handlers.GroupHandler, billHandler *handlers.BillHandler) http.Handler {
	mux := http.NewServeMux()

	// Register Group Routes
	// (Go 1.22+ supports "METHOD /path" syntax)
	mux.HandleFunc("POST /groups", groupHandler.Create)
	mux.HandleFunc("GET /groups/{groupId}", groupHandler.GetById)
mux.HandleFunc("PATCH /groups/{groupId}", groupHandler.Update)
	mux.HandleFunc("DELETE /groups/{groupId}/member/{userId}", groupHandler.DeleteById)
	mux.HandleFunc("POST /groups/{groupId}/invite", groupHandler.Invite)
	mux.HandleFunc("POST /groups/{groupId}/accept-invite", groupHandler.AcceptInvite)
	mux.HandleFunc("POST /groups/{groupId}/decline-invite", groupHandler.DeclineInvite)

	mux.HandleFunc("GET /member/{userId}/guild/{guildId}/groups", groupHandler.GetByGuildIdAndUserId)
	mux.HandleFunc("GET /member/{userId}/guild/{guildId}/own-groups", groupHandler.GetByGuildIdAndUserIdOwn)
	mux.HandleFunc("GET /member/{userId}/guild/{guildId}/pending-invite", groupHandler.GetPendingInvite)
	mux.HandleFunc("GET /member/{userId}/guild/{guildId}/bills", billHandler.GetByGuildIdAndUserId)
	mux.HandleFunc("GET /member/{userId}/guild/{guildId}/unpaid-bills", billHandler.GetUnpaidByGuildIdAndUserId)

	// You can add other handlers here later (e.g., UserHandler)
	mux.HandleFunc("GET /guild/{guildId}/bills", billHandler.GetByGuildId)
	mux.HandleFunc("GET /guild/{guildId}/unpaid-bills", billHandler.GetUnpaidByGuildId)

	mux.HandleFunc("POST /bill/pay", billHandler.Pay)
	mux.HandleFunc("POST /bill/create", billHandler.Create)
	
	return middleware.RequestLogger(
        middleware.AuthMiddleware(mux),
    )
}