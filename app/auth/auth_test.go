package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mock user store
type mockUserStore struct {
	users map[string]*UserInfo
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*UserInfo)}
}

func (m *mockUserStore) FindByUsername(_ context.Context, username string) (*UserInfo, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserStore) FindByID(_ context.Context, id int64) (*UserInfo, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) UpdateLastLogin(_ context.Context, _ int64) error { return nil }

func (m *mockUserStore) UpdatePassword(_ context.Context, id int64, hash string) error {
	for _, u := range m.users {
		if u.ID == id {
			u.PasswordHash = hash
			return nil
		}
	}
	return nil
}

// mock token store
type mockTokenStore struct {
	tokens map[string]*TokenInfo
	nextID int64
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*TokenInfo), nextID: 1}
}

func (m *mockTokenStore) Create(_ context.Context, token TokenInfo) (int64, error) {
	id := m.nextID
	m.nextID++
	token.ID = id
	m.tokens[token.TokenHash] = &token
	return id, nil
}

func (m *mockTokenStore) FindByHash(_ context.Context, hash string) (*TokenInfo, error) {
	t, ok := m.tokens[hash]
	if !ok {
		return nil, nil
	}
	return t, nil
}

func (m *mockTokenStore) DeleteByHash(_ context.Context, hash string) error {
	delete(m.tokens, hash)
	return nil
}

func (m *mockTokenStore) DeleteByUserID(_ context.Context, userID int64) error {
	for k, t := range m.tokens {
		if t.UserID == userID {
			delete(m.tokens, k)
		}
	}
	return nil
}

func (m *mockTokenStore) CleanExpired(_ context.Context) (int64, error) {
	var count int64
	for k, t := range m.tokens {
		if time.Now().After(t.ExpiresAt) {
			delete(m.tokens, k)
			count++
		}
	}
	return count, nil
}

func setupTestService(t *testing.T) (*Service, *mockUserStore, *mockTokenStore) {
	t.Helper()
	userStore := newMockUserStore()
	tokenStore := newMockTokenStore()

	hash, err := HashPassword("password123")
	require.NoError(t, err)

	userStore.users["admin"] = &UserInfo{
		ID:           1,
		Username:     "admin",
		PasswordHash: hash,
		Role:         "superadmin",
		DisplayName:  "Admin",
		Active:       true,
	}

	svc := NewService(userStore, tokenStore, "test-secret")
	return svc, userStore, tokenStore
}

func TestServiceLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("successful login", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		pair, err := svc.Login(ctx, "admin", "password123")
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEmpty(t, pair.RefreshToken)
		assert.Positive(t, pair.ExpiresIn)

		// verify access token is valid
		claims, err := svc.ValidateAccessToken(pair.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, int64(1), claims.UserID)
		assert.Equal(t, "admin", claims.Username)
		assert.Equal(t, "superadmin", claims.Role)
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		_, err := svc.Login(ctx, "admin", "wrong-password")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("non-existent user", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		_, err := svc.Login(ctx, "nonexistent", "password123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("disabled user", func(t *testing.T) {
		svc, userStore, _ := setupTestService(t)
		userStore.users["admin"].Active = false
		_, err := svc.Login(ctx, "admin", "password123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

func TestServiceRefresh(t *testing.T) {
	ctx := context.Background()

	t.Run("successful refresh", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		pair, err := svc.Login(ctx, "admin", "password123")
		require.NoError(t, err)

		newPair, err := svc.Refresh(ctx, pair.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newPair.AccessToken)
		assert.NotEmpty(t, newPair.RefreshToken)
		assert.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		_, err := svc.Refresh(ctx, "invalid-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})

	t.Run("reuse revoked token", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		pair, err := svc.Login(ctx, "admin", "password123")
		require.NoError(t, err)

		// first refresh works
		_, err = svc.Refresh(ctx, pair.RefreshToken)
		require.NoError(t, err)

		// second use of same token fails (already revoked)
		_, err = svc.Refresh(ctx, pair.RefreshToken)
		require.Error(t, err)
	})
}

func TestServiceChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("successful change", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		err := svc.ChangePassword(ctx, 1, "password123", "newpassword456")
		require.NoError(t, err)

		// old password should not work
		_, err = svc.Login(ctx, "admin", "password123")
		require.Error(t, err)

		// new password should work
		pair, err := svc.Login(ctx, "admin", "newpassword456")
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
	})

	t.Run("wrong old password", func(t *testing.T) {
		svc, _, _ := setupTestService(t)
		err := svc.ChangePassword(ctx, 1, "wrong-old", "newpassword456")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect")
	})
}

func TestServiceLogout(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := setupTestService(t)

	pair, err := svc.Login(ctx, "admin", "password123")
	require.NoError(t, err)

	err = svc.Logout(ctx, pair.RefreshToken)
	require.NoError(t, err)

	// refresh should fail after logout
	_, err = svc.Refresh(ctx, pair.RefreshToken)
	require.Error(t, err)
}
