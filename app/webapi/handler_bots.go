package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage"
)

// listBotsHandler handles GET /api/v2/bots
func (s *Server) listBotsHandler(w http.ResponseWriter, r *http.Request) {
	bots, err := s.BotsStore.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] failed to list bots: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list bots")
		return
	}

	// mask tokens in the response for security
	type botResponse struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Token     string    `json:"token"`
		Username  string    `json:"username"`
		Active    bool      `json:"active"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	result := make([]botResponse, 0, len(bots))
	for _, b := range bots {
		result = append(result, botResponse{
			ID:        b.ID,
			Name:      b.Name,
			Token:     maskToken(b.Token),
			Username:  b.Username,
			Active:    b.Active,
			CreatedAt: b.CreatedAt,
			UpdatedAt: b.UpdatedAt,
		})
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// createBotHandler handles POST /api/v2/bots
func (s *Server) createBotHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Token == "" {
		writeJSONError(w, http.StatusBadRequest, "name and token are required")
		return
	}

	// validate token format (should be like "123456:ABC...")
	if !strings.Contains(req.Token, ":") {
		writeJSONError(w, http.StatusBadRequest, "invalid token format, expected 'id:secret'")
		return
	}

	// validate token via Telegram API to get username
	username, validateErr := validateBotToken(r.Context(), req.Token)
	if validateErr != nil {
		log.Printf("[WARN] bot token validation failed: %v", validateErr)
	}

	id, err := s.BotsStore.Create(r.Context(), storage.BotInfo{
		Name:     req.Name,
		Token:    req.Token,
		Username: username,
		Active:   true,
	})
	if err != nil {
		log.Printf("[ERROR] failed to create bot: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create bot")
		return
	}

	writeJSONResponse(w, http.StatusCreated, map[string]any{"id": id, "username": username})
}

// updateBotHandler handles PUT /api/v2/bots/{id}
func (s *Server) updateBotHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid bot id")
		return
	}

	var req struct {
		Name   string `json:"name"`
		Token  string `json:"token"`
		Active bool   `json:"active"`
	}
	if errDec := json.NewDecoder(r.Body).Decode(&req); errDec != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := s.BotsStore.FindByID(r.Context(), id)
	if err != nil {
		log.Printf("[ERROR] failed to find bot id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to find bot")
		return
	}
	if existing == nil {
		writeJSONError(w, http.StatusNotFound, "bot not found")
		return
	}

	// if token looks masked, keep the existing one
	token := req.Token
	if token == "" || strings.Contains(token, "***") {
		token = existing.Token
	}

	if err := s.BotsStore.Update(r.Context(), storage.BotInfo{
		ID:       id,
		Name:     req.Name,
		Token:    token,
		Username: existing.Username,
		Active:   req.Active,
	}); err != nil {
		log.Printf("[ERROR] failed to update bot id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update bot")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteBotHandler handles DELETE /api/v2/bots/{id}
func (s *Server) deleteBotHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid bot id")
		return
	}

	if err := s.BotsStore.Delete(r.Context(), id); err != nil {
		log.Printf("[ERROR] failed to delete bot id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete bot")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// validateBotHandler handles POST /api/v2/bots/{id}/validate
func (s *Server) validateBotHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid bot id")
		return
	}

	bot, err := s.BotsStore.FindByID(r.Context(), id)
	if err != nil {
		log.Printf("[ERROR] failed to find bot id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to find bot")
		return
	}
	if bot == nil {
		writeJSONError(w, http.StatusNotFound, "bot not found")
		return
	}

	// validate by calling Telegram getMe API
	username, validateErr := validateBotToken(r.Context(), bot.Token)
	if validateErr != nil {
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"valid": false, "error": validateErr.Error(),
		})
		return
	}

	// update username if it changed
	if username != bot.Username {
		bot.Username = username
		if updErr := s.BotsStore.Update(r.Context(), *bot); updErr != nil {
			log.Printf("[WARN] failed to update bot username id=%d: %v", id, updErr)
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"valid": true, "username": username,
	})
}

// validateBotToken calls Telegram getMe API to validate a bot token
func validateBotToken(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("can't create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't reach Telegram API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("can't decode response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result.Username, nil
}

// maskToken masks a bot token for display, showing only the bot ID part
func maskToken(token string) string {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "***"
	}
	return parts[0] + ":***"
}
