package webapi

import (
	"encoding/json"
	"net/http"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage"
)

// getDictionaryHandler handles GET /api/v2/dictionary
func (s *Server) getDictionaryHandler(w http.ResponseWriter, r *http.Request) {
	stopPhrases, err := s.Dictionary.ReadWithIDs(r.Context(), storage.DictionaryTypeStopPhrase)
	if err != nil {
		log.Printf("[ERROR] failed to get stop phrases: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get stop phrases")
		return
	}

	ignoredWords, err := s.Dictionary.ReadWithIDs(r.Context(), storage.DictionaryTypeIgnoredWord)
	if err != nil {
		log.Printf("[ERROR] failed to get ignored words: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get ignored words")
		return
	}

	stats, err := s.Dictionary.Stats(r.Context())
	if err != nil {
		log.Printf("[WARN] failed to get dictionary stats: %v", err)
	}

	if stopPhrases == nil {
		stopPhrases = []storage.DictionaryEntry{}
	}
	if ignoredWords == nil {
		ignoredWords = []storage.DictionaryEntry{}
	}
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"stop_phrases":  stopPhrases,
		"ignored_words": ignoredWords,
		"stats":         stats,
	})
}

// addDictionaryHandler handles POST /api/v2/dictionary
func (s *Server) addDictionaryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Data == "" {
		writeJSONError(w, http.StatusBadRequest, "data is required")
		return
	}

	dictType := storage.DictionaryType(req.Type)
	if err := dictType.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid type: "+err.Error())
		return
	}

	if err := s.Dictionary.Add(r.Context(), dictType, req.Data); err != nil {
		log.Printf("[ERROR] failed to add dictionary entry: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add dictionary entry")
		return
	}

	// reload samples to apply dictionary changes
	if err := s.SpamFilter.ReloadSamples(); err != nil {
		log.Printf("[WARN] failed to reload samples after dictionary add: %v", err)
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteDictionaryHandler handles DELETE /api/v2/dictionary/{id}
func (s *Server) deleteDictionaryHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.Dictionary.Delete(r.Context(), req.ID); err != nil {
		log.Printf("[ERROR] failed to delete dictionary entry id=%d: %v", req.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete dictionary entry")
		return
	}

	// reload samples to apply dictionary changes
	if err := s.SpamFilter.ReloadSamples(); err != nil {
		log.Printf("[WARN] failed to reload samples after dictionary delete: %v", err)
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
