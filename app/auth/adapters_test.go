package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/app/storage/engine"
)

func setupTestDB(t *testing.T) *engine.SQL {
	t.Helper()
	db, err := engine.New(context.Background(), ":memory:", "test")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestAdminUsersAdapter(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	store, err := storage.NewAdminUsers(ctx, db)
	require.NoError(t, err)

	adapter := NewAdminUsersAdapter(store)

	hash, err := HashPassword("password123")
	require.NoError(t, err)

	id, err := store.Create(ctx, storage.AdminUserInfo{
		Username:     "testuser",
		PasswordHash: hash,
		Role:         "admin",
		DisplayName:  "Test User",
		Active:       true,
	})
	require.NoError(t, err)
	require.Positive(t, id)

	t.Run("FindByUsername", func(t *testing.T) {
		user, err := adapter.FindByUsername(ctx, "testuser")
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "admin", user.Role)
		assert.Equal(t, "Test User", user.DisplayName)
		assert.True(t, user.Active)
	})

	t.Run("FindByUsername not found", func(t *testing.T) {
		user, err := adapter.FindByUsername(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("FindByID", func(t *testing.T) {
		user, err := adapter.FindByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, id, user.ID)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		err := adapter.UpdateLastLogin(ctx, id)
		require.NoError(t, err)
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		newHash, err := HashPassword("newpassword")
		require.NoError(t, err)
		err = adapter.UpdatePassword(ctx, id, newHash)
		require.NoError(t, err)

		user, err := adapter.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, newHash, user.PasswordHash)
	})
}

func TestRefreshTokensAdapter(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	store, err := storage.NewRefreshTokens(ctx, db)
	require.NoError(t, err)

	adapter := NewRefreshTokensAdapter(store)

	t.Run("Create and FindByHash", func(t *testing.T) {
		id, err := adapter.Create(ctx, TokenInfo{
			UserID:    1,
			TokenHash: "testhash123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)
		assert.Positive(t, id)

		token, err := adapter.FindByHash(ctx, "testhash123")
		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, int64(1), token.UserID)
		assert.Equal(t, "testhash123", token.TokenHash)
	})

	t.Run("FindByHash not found", func(t *testing.T) {
		token, err := adapter.FindByHash(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("DeleteByHash", func(t *testing.T) {
		_, err := adapter.Create(ctx, TokenInfo{
			UserID:    2,
			TokenHash: "deleteme",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		err = adapter.DeleteByHash(ctx, "deleteme")
		require.NoError(t, err)

		token, err := adapter.FindByHash(ctx, "deleteme")
		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("DeleteByUserID", func(t *testing.T) {
		_, err := adapter.Create(ctx, TokenInfo{
			UserID:    3,
			TokenHash: "user3token1",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)
		_, err = adapter.Create(ctx, TokenInfo{
			UserID:    3,
			TokenHash: "user3token2",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		err = adapter.DeleteByUserID(ctx, 3)
		require.NoError(t, err)

		token, err := adapter.FindByHash(ctx, "user3token1")
		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("CleanExpired", func(t *testing.T) {
		_, err := adapter.Create(ctx, TokenInfo{
			UserID:    4,
			TokenHash: "expired",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		})
		require.NoError(t, err)

		count, err := adapter.CleanExpired(ctx)
		require.NoError(t, err)
		assert.Positive(t, count)
	})
}
