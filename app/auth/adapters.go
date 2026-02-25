package auth

import (
	"context"
	"fmt"

	"github.com/umputun/tg-spam/app/storage"
)

// AdminUsersAdapter adapts storage.AdminUsers to auth.UserStore interface
type AdminUsersAdapter struct {
	store *storage.AdminUsers
}

// NewAdminUsersAdapter creates a new AdminUsersAdapter
func NewAdminUsersAdapter(store *storage.AdminUsers) *AdminUsersAdapter {
	return &AdminUsersAdapter{store: store}
}

// FindByUsername returns user info by username
func (a *AdminUsersAdapter) FindByUsername(ctx context.Context, username string) (*UserInfo, error) {
	u, err := a.store.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	if u == nil {
		return nil, nil
	}
	return adminUserToAuthUser(u), nil
}

// FindByID returns user info by id
func (a *AdminUsersAdapter) FindByID(ctx context.Context, id int64) (*UserInfo, error) {
	u, err := a.store.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	if u == nil {
		return nil, nil
	}
	return adminUserToAuthUser(u), nil
}

// UpdateLastLogin updates the last login timestamp
func (a *AdminUsersAdapter) UpdateLastLogin(ctx context.Context, id int64) error {
	if err := a.store.UpdateLastLogin(ctx, id); err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// UpdatePassword updates the password hash
func (a *AdminUsersAdapter) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	if err := a.store.UpdatePassword(ctx, id, passwordHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func adminUserToAuthUser(u *storage.AdminUserInfo) *UserInfo {
	return &UserInfo{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		DisplayName:  u.DisplayName,
		Active:       u.Active,
	}
}

// RefreshTokensAdapter adapts storage.RefreshTokens to auth.TokenStore interface
type RefreshTokensAdapter struct {
	store *storage.RefreshTokens
}

// NewRefreshTokensAdapter creates a new RefreshTokensAdapter
func NewRefreshTokensAdapter(store *storage.RefreshTokens) *RefreshTokensAdapter {
	return &RefreshTokensAdapter{store: store}
}

// Create stores a new refresh token
func (a *RefreshTokensAdapter) Create(ctx context.Context, token TokenInfo) (int64, error) {
	id, err := a.store.Create(ctx, storage.RefreshTokenInfo{
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return 0, fmt.Errorf("create refresh token: %w", err)
	}
	return id, nil
}

// FindByHash returns a refresh token by its hash
func (a *RefreshTokensAdapter) FindByHash(ctx context.Context, tokenHash string) (*TokenInfo, error) {
	t, err := a.store.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("find token by hash: %w", err)
	}
	if t == nil {
		return nil, nil
	}
	return refreshTokenToAuthToken(t), nil
}

// DeleteByHash removes a refresh token by its hash
func (a *RefreshTokensAdapter) DeleteByHash(ctx context.Context, tokenHash string) error {
	if err := a.store.DeleteByHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("delete token by hash: %w", err)
	}
	return nil
}

// DeleteByUserID removes all refresh tokens for a user
func (a *RefreshTokensAdapter) DeleteByUserID(ctx context.Context, userID int64) error {
	if err := a.store.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("delete tokens by user id: %w", err)
	}
	return nil
}

// CleanExpired removes all expired refresh tokens
func (a *RefreshTokensAdapter) CleanExpired(ctx context.Context) (int64, error) {
	count, err := a.store.CleanExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("clean expired tokens: %w", err)
	}
	return count, nil
}

func refreshTokenToAuthToken(t *storage.RefreshTokenInfo) *TokenInfo {
	return &TokenInfo{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}

// compile-time interface checks
var (
	_ UserStore  = (*AdminUsersAdapter)(nil)
	_ TokenStore = (*RefreshTokensAdapter)(nil)
)
