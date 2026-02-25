package webapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/auth"
	"github.com/umputun/tg-spam/app/storage"
)

// listAdminUsersHandler handles GET /api/v2/admin/users
func (s *Server) listAdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := s.AdminUsersStore.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] failed to list admin users: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list admin users")
		return
	}
	writeJSONResponse(w, http.StatusOK, users)
}

// createAdminUserHandler handles POST /api/v2/admin/users
func (s *Server) createAdminUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"` //nolint:gosec // request field, not a hardcoded credential
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	if req.Role == "" {
		req.Role = "moderator"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("[ERROR] failed to hash password: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	id, err := s.AdminUsersStore.Create(r.Context(), storage.AdminUserInfo{
		Username:     req.Username,
		PasswordHash: hash,
		Role:         req.Role,
		DisplayName:  req.DisplayName,
		Active:       true,
	})
	if err != nil {
		log.Printf("[ERROR] failed to create admin user: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSONResponse(w, http.StatusCreated, map[string]int64{"id": id})
}

// updateAdminUserHandler handles PUT /api/v2/admin/users/{id}
func (s *Server) updateAdminUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Username    string `json:"username"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
		Active      bool   `json:"active"`
	}
	if errDec := json.NewDecoder(r.Body).Decode(&req); errDec != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if errUpd := s.AdminUsersStore.Update(r.Context(), storage.AdminUserInfo{
		ID:          id,
		Username:    req.Username,
		Role:        req.Role,
		DisplayName: req.DisplayName,
		Active:      req.Active,
	}); errUpd != nil {
		log.Printf("[ERROR] failed to update admin user id=%d: %v", id, errUpd)
		writeJSONError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteAdminUserHandler handles DELETE /api/v2/admin/users/{id}
func (s *Server) deleteAdminUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// prevent deleting yourself
	claims, ok := auth.UserFromContext(r.Context())
	if ok && claims.UserID == id {
		writeJSONError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	if err := s.AdminUsersStore.Delete(r.Context(), id); err != nil {
		log.Printf("[ERROR] failed to delete admin user id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// resetAdminUserPasswordHandler handles PUT /api/v2/admin/users/{id}/password
func (s *Server) resetAdminUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Password string `json:"password"` //nolint:gosec // request field, not a hardcoded credential
	}
	if errDec := json.NewDecoder(r.Body).Decode(&req); errDec != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("[ERROR] failed to hash password: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	if err := s.AdminUsersStore.UpdatePassword(r.Context(), id, hash); err != nil {
		log.Printf("[ERROR] failed to reset password for user id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
