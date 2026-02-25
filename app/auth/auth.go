// Package auth provides JWT-based authentication for the web admin panel.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserStore provides access to admin user data
type UserStore interface {
	FindByUsername(ctx context.Context, username string) (*UserInfo, error)
	FindByID(ctx context.Context, id int64) (*UserInfo, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
}

// TokenStore provides access to refresh token data
type TokenStore interface {
	Create(ctx context.Context, token TokenInfo) (int64, error)
	FindByHash(ctx context.Context, tokenHash string) (*TokenInfo, error)
	DeleteByHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID int64) error
	CleanExpired(ctx context.Context) (int64, error)
}

// UserInfo represents a user for authentication purposes
type UserInfo struct {
	ID           int64  `db:"id"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	Role         string `db:"role"`
	DisplayName  string `db:"display_name"`
	Active       bool   `db:"active"`
}

// TokenInfo represents a stored refresh token
type TokenInfo struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`  //nolint:gosec // not a credential, just a JSON field name
	RefreshToken string `json:"refresh_token"` //nolint:gosec // not a credential, just a JSON field name
	ExpiresIn    int64  `json:"expires_in"`
}

// Service provides authentication operations
type Service struct {
	userStore  UserStore
	tokenStore TokenStore
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService creates a new auth service
func NewService(userStore UserStore, tokenStore TokenStore, jwtSecret string) *Service {
	return &Service{
		userStore:  userStore,
		tokenStore: tokenStore,
		jwtSecret:  jwtSecret,
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

// Login authenticates user by username and password, returns token pair
func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.userStore.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if !user.Active {
		return nil, fmt.Errorf("account is disabled")
	}

	if errCmp := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); errCmp != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if err := s.userStore.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Printf("[WARN] failed to update last login for user %s: %v", username, err)
	}

	log.Printf("[INFO] user %s logged in", username)
	return pair, nil
}

// Refresh exchanges a valid refresh token for a new token pair
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	stored, err := s.tokenStore.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	if stored == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.tokenStore.DeleteByHash(ctx, tokenHash)
		return nil, fmt.Errorf("refresh token expired")
	}

	// revoke old token
	if errDel := s.tokenStore.DeleteByHash(ctx, tokenHash); errDel != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", errDel)
	}

	user, err := s.userStore.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil || !user.Active {
		return nil, fmt.Errorf("user not found or disabled")
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	return pair, nil
}

// ChangePassword updates the password for a user
func (s *Service) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	if errCmp := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); errCmp != nil {
		return fmt.Errorf("old password is incorrect")
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userStore.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// revoke all refresh tokens for this user to force re-login
	if err := s.tokenStore.DeleteByUserID(ctx, userID); err != nil {
		log.Printf("[WARN] failed to revoke tokens after password change for user_id=%d: %v", userID, err)
	}

	log.Printf("[INFO] password changed for user_id=%d", userID)
	return nil
}

// Logout revokes a specific refresh token
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	if err := s.tokenStore.DeleteByHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// ValidateAccessToken validates an access token and returns claims
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	return ValidateToken(tokenString, s.jwtSecret)
}

func (s *Service) generateTokenPair(ctx context.Context, user *UserInfo) (*TokenPair, error) {
	accessToken, err := GenerateToken(user.ID, user.Username, user.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := hashToken(refreshToken)
	_, err = s.tokenStore.Create(ctx, TokenInfo{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
