package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/lib/approved"
)

// listApprovedUsersHandler handles GET /api/v2/users/approved?gid=xxx
func (s *Server) listApprovedUsersHandler(w http.ResponseWriter, r *http.Request) {
	gid := r.URL.Query().Get("gid")
	if gid == "" {
		writeJSONError(w, http.StatusBadRequest, "gid parameter is required")
		return
	}

	users, err := s.ApprovedUsersStore.ReadByGID(r.Context(), gid)
	if err != nil {
		log.Printf("[ERROR] failed to read approved users for gid=%s: %v", gid, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to read approved users")
		return
	}
	if users == nil {
		users = []approved.UserInfo{}
	}
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"users": users,
		"total": len(users),
	})
}

// addApprovedUserHandler handles POST /api/v2/users/approved
func (s *Server) addApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GID      string `json:"gid"`
		UserID   string `json:"user_id"`
		UserName string `json:"user_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GID == "" {
		writeJSONError(w, http.StatusBadRequest, "gid is required")
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

	if err := s.ApprovedUsersStore.WriteByGID(r.Context(), req.GID, approved.UserInfo{
		UserID:   req.UserID,
		UserName: req.UserName,
	}); err != nil {
		log.Printf("[ERROR] failed to add approved user for gid=%s: %v", req.GID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add approved user")
		return
	}

	// sync with running channel's detector in-memory map
	if s.ChannelManager != nil {
		if chBot := s.ChannelManager.GetChannelBot(req.GID); chBot != nil {
			uid, _ := strconv.ParseInt(req.UserID, 10, 64)
			if err := chBot.AddApprovedUser(uid, req.UserName); err != nil {
				log.Printf("[WARN] failed to sync approved user to detector for gid=%s: %v", req.GID, err)
			}
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// removeApprovedUserHandler handles DELETE /api/v2/users/approved/{user_id}?gid=xxx
func (s *Server) removeApprovedUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	gid := r.URL.Query().Get("gid")
	if gid == "" {
		writeJSONError(w, http.StatusBadRequest, "gid parameter is required")
		return
	}

	if err := s.ApprovedUsersStore.DeleteByGID(r.Context(), gid, userID); err != nil {
		log.Printf("[ERROR] failed to remove approved user %s for gid=%s: %v", userID, gid, err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to remove approved user: %v", err))
		return
	}

	// sync with running channel's detector in-memory map
	if s.ChannelManager != nil {
		if chBot := s.ChannelManager.GetChannelBot(gid); chBot != nil {
			uid, _ := strconv.ParseInt(userID, 10, 64)
			if err := chBot.RemoveApprovedUser(uid); err != nil {
				log.Printf("[WARN] failed to sync approved user removal to detector for gid=%s: %v", gid, err)
			}
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
