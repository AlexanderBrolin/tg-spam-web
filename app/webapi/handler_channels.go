package webapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage"
)

// listChannelsHandler handles GET /api/v2/channels
func (s *Server) listChannelsHandler(w http.ResponseWriter, r *http.Request) {
	channels, err := s.ChannelsStore.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] failed to list channels: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}
	writeJSONResponse(w, http.StatusOK, channels)
}

// createChannelHandler handles POST /api/v2/channels
func (s *Server) createChannelHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GID        string `json:"gid"`
		TelegramID int64  `json:"telegram_id"`
		Name       string `json:"name"`
		Username   string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GID == "" || req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "gid and name are required")
		return
	}

	id, err := s.ChannelsStore.Create(r.Context(), storage.ChannelInfo{
		GID:        req.GID,
		TelegramID: req.TelegramID,
		Name:       req.Name,
		Username:   req.Username,
		Active:     true,
	})
	if err != nil {
		log.Printf("[ERROR] failed to create channel: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create channel")
		return
	}

	// create default settings for the channel
	if _, err := s.ChannelSettingsStore.Create(r.Context(), req.GID); err != nil {
		log.Printf("[WARN] failed to create default settings for channel gid=%s: %v", req.GID, err)
	}

	writeJSONResponse(w, http.StatusCreated, map[string]int64{"id": id})
}

// updateChannelHandler handles PUT /api/v2/channels/{id}
func (s *Server) updateChannelHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	var req struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Active   bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.ChannelsStore.Update(r.Context(), storage.ChannelInfo{
		ID:       id,
		Name:     req.Name,
		Username: req.Username,
		Active:   req.Active,
	}); err != nil {
		log.Printf("[ERROR] failed to update channel id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update channel")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteChannelHandler handles DELETE /api/v2/channels/{id}
func (s *Server) deleteChannelHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	if err := s.ChannelsStore.Delete(r.Context(), id); err != nil {
		log.Printf("[ERROR] failed to delete channel id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete channel")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// getChannelSettingsHandler handles GET /api/v2/channels/{gid}/settings
func (s *Server) getChannelSettingsHandler(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	settings, err := s.ChannelSettingsStore.Get(r.Context(), gid)
	if err != nil {
		log.Printf("[ERROR] failed to get channel settings gid=%s: %v", gid, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get channel settings")
		return
	}
	if settings == nil {
		writeJSONError(w, http.StatusNotFound, "channel settings not found")
		return
	}
	writeJSONResponse(w, http.StatusOK, settings)
}

// updateChannelSettingsHandler handles PUT /api/v2/channels/{gid}/settings
func (s *Server) updateChannelSettingsHandler(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")

	var settings storage.ChannelSettingsInfo
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	settings.GID = gid

	if err := s.ChannelSettingsStore.Update(r.Context(), settings); err != nil {
		log.Printf("[ERROR] failed to update channel settings gid=%s: %v", gid, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update channel settings")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
