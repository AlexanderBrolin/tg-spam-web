package webapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/lib/spamcheck"
)

// listDetectedSpamHandler handles GET /api/v2/spam/detected
func (s *Server) listDetectedSpamHandler(w http.ResponseWriter, r *http.Request) {
	gid := r.URL.Query().Get("gid")

	// parse date range, default to current calendar week (Monday to Sunday)
	now := time.Now()
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	mondayOffset := int(weekday) - int(time.Monday)
	defaultFrom := time.Date(now.Year(), now.Month(), now.Day()-mondayOffset, 0, 0, 0, 0, now.Location())
	defaultTo := now

	from := defaultFrom
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if parsed, pErr := time.ParseInLocation("2006-01-02", fromStr, now.Location()); pErr == nil {
			from = parsed
		}
	}
	to := defaultTo
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if parsed, pErr := time.ParseInLocation("2006-01-02", toStr, now.Location()); pErr == nil {
			to = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, now.Location())
		}
	}

	var entries []storage.DetectedSpamInfo
	var err error
	if gid != "" {
		entries, err = s.DetectedSpam.ReadByGIDAndDateRange(r.Context(), gid, from, to)
	} else {
		entries, err = s.DetectedSpam.Read(r.Context())
	}
	if err != nil {
		log.Printf("[ERROR] failed to read detected spam: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to read detected spam")
		return
	}

	// apply classifier filter
	filter := r.URL.Query().Get("filter")
	if filter == "non-classified" {
		filtered := entries[:0]
		for _, entry := range entries {
			for _, check := range entry.Checks {
				if check.Name == "classifier" && !check.Spam {
					filtered = append(filtered, entry)
					break
				}
			}
		}
		entries = filtered
	} else if filter == "openai" {
		filtered := entries[:0]
		for _, entry := range entries {
			for _, check := range entry.Checks {
				if check.Name == "openai" {
					filtered = append(filtered, entry)
					break
				}
			}
		}
		entries = filtered
	}

	// pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	if entries == nil {
		entries = entries[:0:0]
	}

	total := len(entries)
	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)

	pageEntries := entries[start:end]
	if pageEntries == nil {
		pageEntries = entries[:0:0]
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"entries":  pageEntries,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// addDetectedSpamToSamplesHandler handles POST /api/v2/spam/detected/{id}/add
func (s *Server) addDetectedSpamToSamplesHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid spam entry id")
		return
	}

	var req struct {
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.SpamFilter.UpdateSpam(req.Msg); err != nil {
		log.Printf("[ERROR] failed to add to spam samples: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add to spam samples")
		return
	}

	if err := s.DetectedSpam.SetAddedToSamplesFlag(r.Context(), id); err != nil {
		log.Printf("[ERROR] failed to set added flag: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update detected spam entry")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// unbanUserHandler handles POST /api/v2/spam/detected/{id}/unban
func (s *Server) unbanUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GID    string `json:"gid"`
		UserID int64  `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GID == "" || req.UserID == 0 {
		writeJSONError(w, http.StatusBadRequest, "gid and user_id are required")
		return
	}

	if s.ChannelManager == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "channel manager not available")
		return
	}

	if err := s.ChannelManager.UnbanUser(req.GID, req.UserID); err != nil {
		log.Printf("[ERROR] failed to unban user %d in channel %s: %v", req.UserID, req.GID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to unban user")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// spamCheckHandler handles POST /api/v2/spam/check
func (s *Server) spamCheckHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		spamcheck.Request
		GID string `json:"gid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.CheckOnly = true

	// use per-channel detector if gid is provided and channel is running
	if req.GID != "" && s.ChannelManager != nil {
		if chBot := s.ChannelManager.GetChannelBot(req.GID); chBot != nil {
			spam, checks := chBot.Check(req.Request)
			writeJSONResponse(w, http.StatusOK, map[string]any{
				"spam":   spam,
				"checks": checks,
			})
			return
		}
	}

	// fallback to global detector
	spam, checks := s.Detector.Check(req.Request)
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"spam":   spam,
		"checks": checks,
	})
}
