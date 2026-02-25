package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/lib/approved"
)

// listApprovedUsersHandler handles GET /api/v2/users/approved
func (s *Server) listApprovedUsersHandler(w http.ResponseWriter, _ *http.Request) {
	users := s.Detector.ApprovedUsers()
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"users": users,
		"total": len(users),
	})
}

// addApprovedUserHandler handles POST /api/v2/users/approved
func (s *Server) addApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		UserName string `json:"user_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// resolve user ID from username if not provided
	if req.UserID == "" && req.UserName != "" {
		req.UserID = strconv.FormatInt(s.Locator.UserIDByName(r.Context(), req.UserName), 10)
	}

	if req.UserID == "" || req.UserID == "0" {
		writeJSONError(w, http.StatusBadRequest, "user_id or valid user_name is required")
		return
	}

	if err := s.Detector.AddApprovedUser(approved.UserInfo{
		UserID:   req.UserID,
		UserName: req.UserName,
	}); err != nil {
		log.Printf("[ERROR] failed to add approved user: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add approved user")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// removeApprovedUserHandler handles DELETE /api/v2/users/approved/{user_id}
func (s *Server) removeApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if err := s.Detector.RemoveApprovedUser(userID); err != nil {
		log.Printf("[ERROR] failed to remove approved user %s: %v", userID, err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to remove approved user: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
