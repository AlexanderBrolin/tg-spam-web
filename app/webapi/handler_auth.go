package webapi

import (
	"encoding/json"
	"net/http"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/auth"
)

// loginHandler handles POST /api/v2/auth/login
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"` //nolint:gosec // request field, not a hardcoded credential
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := s.AuthService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("[WARN] login failed for user %s: %v", req.Username, err)
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSONResponse(w, http.StatusOK, pair)
}

// refreshHandler handles POST /api/v2/auth/refresh
func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"` //nolint:gosec // request field, not a hardcoded credential
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pair, err := s.AuthService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	writeJSONResponse(w, http.StatusOK, pair)
}

// logoutHandler handles POST /api/v2/auth/logout
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"` //nolint:gosec // request field, not a hardcoded credential
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.AuthService.Logout(r.Context(), req.RefreshToken); err != nil {
		log.Printf("[WARN] logout error: %v", err)
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// meHandler handles GET /api/v2/auth/me
func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := s.AdminUsersStore.FindByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, user)
}

// changePasswordHandler handles PUT /api/v2/auth/password
func (s *Server) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.AuthService.ChangePassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
