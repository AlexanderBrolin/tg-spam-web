package webapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/lib/spamcheck"
)

// listDetectedSpamHandler handles GET /api/v2/spam/detected
func (s *Server) listDetectedSpamHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := s.DetectedSpam.Read(r.Context())
	if err != nil {
		log.Printf("[ERROR] failed to read detected spam: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to read detected spam")
		return
	}

	// apply gid filter if provided
	gid := r.URL.Query().Get("gid")
	if gid != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.GID == gid {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
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

// spamCheckHandler handles POST /api/v2/spam/check
func (s *Server) spamCheckHandler(w http.ResponseWriter, r *http.Request) {
	var req spamcheck.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.CheckOnly = true

	spam, checks := s.Detector.Check(req)
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"spam":   spam,
		"checks": checks,
	})
}
