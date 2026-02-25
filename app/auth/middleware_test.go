package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	svc := &Service{jwtSecret: secret}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := UserFromContext(r.Context())
		assert.True(t, ok) //nolint:testifylint // require not safe in http handler goroutine
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(claims.Username))
	})

	wrapped := svc.AuthMiddleware(handler)

	t.Run("valid token", func(t *testing.T) {
		token, err := GenerateToken(1, "admin", "superadmin", secret, 15*time.Minute)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "admin", rr.Body.String())
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("Authorization", "InvalidFormat")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := GenerateToken(1, "admin", "superadmin", secret, -1*time.Minute)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestRequireRole(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed role", func(t *testing.T) {
		middleware := RequireRole("superadmin", "admin")
		wrapped := middleware(handler)

		claims := &Claims{UserID: 1, Username: "admin", Role: "admin"}
		ctx := context.WithValue(context.Background(), userContextKey, claims)

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody).WithContext(ctx)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("forbidden role", func(t *testing.T) {
		middleware := RequireRole("superadmin")
		wrapped := middleware(handler)

		claims := &Claims{UserID: 2, Username: "mod", Role: "moderator"}
		ctx := context.WithValue(context.Background(), userContextKey, claims)

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody).WithContext(ctx)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("no auth context", func(t *testing.T) {
		middleware := RequireRole("superadmin")
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestUserFromContext(t *testing.T) {
	t.Run("with claims", func(t *testing.T) {
		claims := &Claims{UserID: 1, Username: "test"}
		ctx := context.WithValue(context.Background(), userContextKey, claims)
		got, ok := UserFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, int64(1), got.UserID)
	})

	t.Run("without claims", func(t *testing.T) {
		_, ok := UserFromContext(context.Background())
		assert.False(t, ok)
	})
}
