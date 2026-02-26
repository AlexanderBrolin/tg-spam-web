package webapi

import (
	"encoding/json"
	"net/http"

	log "github.com/go-pkgz/lgr"
)

// getSamplesHandler handles GET /api/v2/samples
func (s *Server) getSamplesHandler(w http.ResponseWriter, _ *http.Request) {
	spam, ham, err := s.SpamFilter.AllSamples()
	if err != nil {
		log.Printf("[ERROR] failed to get dynamic samples: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get samples")
		return
	}
	if spam == nil {
		spam = []string{}
	}
	if ham == nil {
		ham = []string{}
	}
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"spam":       spam,
		"ham":        ham,
		"spam_count": len(spam),
		"ham_count":  len(ham),
	})
}

// addSpamSampleHandler handles POST /api/v2/samples/spam
func (s *Server) addSpamSampleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Msg == "" {
		writeJSONError(w, http.StatusBadRequest, "msg is required")
		return
	}

	if err := s.SpamFilter.UpdateSpam(req.Msg); err != nil {
		log.Printf("[ERROR] failed to add spam sample: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add spam sample")
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// addHamSampleHandler handles POST /api/v2/samples/ham
func (s *Server) addHamSampleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Msg == "" {
		writeJSONError(w, http.StatusBadRequest, "msg is required")
		return
	}

	if err := s.SpamFilter.UpdateHam(req.Msg); err != nil {
		log.Printf("[ERROR] failed to add ham sample: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add ham sample")
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteSpamSampleHandler handles DELETE /api/v2/samples/spam
func (s *Server) deleteSpamSampleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.SpamFilter.RemoveDynamicSpamSample(req.Msg); err != nil {
		log.Printf("[ERROR] failed to delete spam sample: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete spam sample")
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteHamSampleHandler handles DELETE /api/v2/samples/ham
func (s *Server) deleteHamSampleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.SpamFilter.RemoveDynamicHamSample(req.Msg); err != nil {
		log.Printf("[ERROR] failed to delete ham sample: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete ham sample")
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// reloadSamplesHandler handles PUT /api/v2/samples/reload
func (s *Server) reloadSamplesHandler(w http.ResponseWriter, _ *http.Request) {
	if err := s.SpamFilter.ReloadSamples(); err != nil {
		log.Printf("[ERROR] failed to reload samples: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reload samples")
		return
	}
	if s.ChannelManager != nil {
		if err := s.ChannelManager.ReloadAllSamples(); err != nil {
			log.Printf("[WARN] failed to reload channel samples: %v", err)
		}
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
