package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/NoNiiEa/subShare-Discord/source/bill"
	"github.com/NoNiiEa/subShare-Discord/source/billVer"
	"github.com/NoNiiEa/subShare-Discord/source/database"
	"github.com/NoNiiEa/subShare-Discord/source/group"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router   *chi.Mux
	groupSvc *group.Service
	billSvc  *bill.Service
	billVerSvc *billver.Service
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req group.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := s.groupSvc.CreateGroup(r.Context(), req)
	if err != nil {
		if errors.Is(err, group.ErrInvalidName) ||
			errors.Is(err, group.ErrInvalidAmount) ||
			errors.Is(err, group.ErrInvalidDueDay) ||
			errors.Is(err, group.ErrInvalidGuildID) ||
			errors.Is(err, group.ErrInvalidOwnerID) ||
			errors.Is(err, group.ErrNoMembersProvided) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	var req group.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	g, err := s.groupSvc.UpdateGroup(r.Context(), req, id)
	if err != nil {
		if errors.Is(err, group.ErrInvalidName) ||
			errors.Is(err, group.ErrInvalidAmount) ||
			errors.Is(err, group.ErrInvalidDueDay) ||
			errors.Is(err, group.ErrInvalidGuildID) ||
			errors.Is(err, group.ErrInvalidOwnerID) ||
			errors.Is(err, group.ErrNoMembersProvided) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	g, err := s.groupSvc.GetGroup(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}

		// Other errors → 500
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0{
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	err = s.groupSvc.DeleteGroup(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "group not found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleInviteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0{
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req group.InviteGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := s.groupSvc.InviteGroup(r.Context(), req, id)
	if err != nil {
		if errors.Is(err, group.ErrAleadyInvited) ||
		errors.Is(err, group.ErrAleadyMembered) {
			http.Error(w, "invalid members invite", http.StatusBadRequest)
			return
		}

		if errors.Is(err, group.ErrInvitedPermission) {
			http.Error(w, "user do not have permission to invite", http.StatusForbidden)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0{
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req group.AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	g, err := s.groupSvc.AcceptInvite(r.Context(), req, id)
	if err != nil {
		if errors.Is(err, group.ErrNotInvited) {
			http.Error(w, "user is not invited or is aleady a member", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleResetPayment(w http.ResponseWriter, r *http.Request) {
	DueDayStr := chi.URLParam(r, "DueDay")
	DueDay, err := strconv.Atoi(DueDayStr)
	if err != nil {
		http.Error(w, "invalid due day", http.StatusBadRequest)
		return
	}
	
	if err := s.groupSvc.ResetPaymentForDueday(r.Context(), DueDay); err != nil {
		if errors.Is(err, group.ErrInvalidDueDay) {
			http.Error(w, "invalid due day", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkAsPaid(w http.ResponseWriter, r *http.Request) {
	GroupIDStr := chi.URLParam(r, "GroupID")
	GroupID, err := strconv.ParseInt(GroupIDStr, 10, 64)
	if err != nil || GroupID <= 0{
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	MemberID := chi.URLParam(r, "MemberID")

	var req group.MarkAsPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	member, err := s.groupSvc.MarkMemberPaid(r.Context(), req, GroupID, MemberID)
	if err != nil {
		if errors.Is(err, group.ErrInvalidGroupID) || errors.Is(err, group.ErrNoUserID) {
			http.Error(w, "invalid group_id or user_id", http.StatusBadRequest)
			return
		}

		if errors.Is(err, group.ErrMemberNotFound) || errors.Is(err, group.ErrNotActiveMember) {
			http.Error(w, "member not found in group", http.StatusNotFound)
			return
		}

		if errors.Is(err, group.ErrAlreadyPaid) {
			http.Error(w, "member already paid", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleGetBillByGroupID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0{
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	bills, err := s.billSvc.GetBillsByGroup(r.Context(), id)
	if err != nil {
		if errors.Is(err, bill.ErrInvalidGroupID) {
			http.Error(w, "invalid groupID", http.StatusBadRequest)
			return
		}

		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "no bills found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bills)
}

func (s *Server) handleGetBillsByMemberID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	bills, err := s.billSvc.GetBillsByMember(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "no bills found", http.StatusNotFound)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bills)
}

func (s *Server) handleSubmitBill(w http.ResponseWriter, r *http.Request) {
	// 1) Parse bill ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// 2) Parse multipart form (for file upload)
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB
		http.Error(w, "invalid form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 3) member_id (required)
	memberID := r.FormValue("member_id")
	if memberID == "" {
		http.Error(w, "member_id is required", http.StatusBadRequest)
		return
	}

	// 4) amount_paid (optional)
	var amountPaid float64
	amountStr := r.FormValue("amount_paid")
	if amountStr != "" {
		amountPaid, err = strconv.ParseFloat(amountStr, 64)
		if err != nil {
			http.Error(w, "invalid amount_paid", http.StatusBadRequest)
			return
		}
	}

	// 5) Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "could not read file", http.StatusInternalServerError)
		return
	}

	// 6) Build request for service
	req := billver.SubmitBillProofRequest{
		BillID:     id,
		MemberID:   memberID,
		AmountPaid: amountPaid,
		ImageBytes: fileBytes,
		FileName:   header.Filename,
	}

	// 7) Call service
	b, _, err := s.billVerSvc.SubmitBillProof(r.Context(), req)
	if err != nil {
		// you can handle specific errors here (not invited, bill not found, verification failed, etc.)
		fmt.Println("SubmitBillProof error:", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	markReq := group.MarkAsPaidRequest{
		Amount: int64(b.AmountPaid),
	}
	_, err = s.groupSvc.MarkMemberPaid(r.Context(), markReq, b.GroupID, b.MemberID)
	if err != nil {
		if errors.Is(err, group.ErrInvalidGroupID) || errors.Is(err, group.ErrNoUserID) {
			http.Error(w, "invalid group_id or user_id", http.StatusBadRequest)
			return
		}

		if errors.Is(err, group.ErrMemberNotFound) || errors.Is(err, group.ErrNotActiveMember) {
			http.Error(w, "member not found in group", http.StatusNotFound)
			return
		}

		if errors.Is(err, group.ErrAlreadyPaid) {
			http.Error(w, "member already paid", http.StatusBadRequest)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 8) Return updated bill as JSON
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.router.Get("/health", s.handleHealth)

	s.router.Route("/groups", func(r chi.Router) {
		r.Post("/", s.handleCreateGroup)   // POST /groups
		r.Get("/{id}", s.handleGetGroup)   // GET /groups/1
		r.Delete("/{id}", s.handleDeleteGroup)
		r.Put("/{id}", s.handleUpdateGroup)
		r.Post("/{id}/invite", s.handleInviteGroup)
		r.Post("/{id}/accept-invite", s.handleAcceptInvite)
		r.Post("/{GroupID}/member/{MemberID}/pay", s.handleMarkAsPaid)
		r.Get("/{id}/bill", s.handleGetBillByGroupID)
	})

	s.router.Route("/member", func(r chi.Router) {
		r.Get("/{id}/bill", s.handleGetBillsByMemberID)
	})

	s.router.Route("/test", func(r chi.Router) {
		r.Post("/due-day/{DueDay}", s.handleResetPayment)
	})

	s.router.Route("/bill", func(r chi.Router) {
		r.Post("/{id}/pay", s.handleSubmitBill)
	})
}

func NewServer(groupSvc *group.Service, billSvc *bill.Service, billVerSvc *billver.Service) *Server {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	s := &Server{
		router:   r,
		groupSvc: groupSvc,
		billSvc: billSvc,
		billVerSvc: billVerSvc,
	}

	s.routes()
	return s
}


