package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/umputun/tg-spam/app/storage/engine"
)

// RefreshTokens is a storage for JWT refresh tokens
type RefreshTokens struct {
	*engine.SQL
	engine.RWLocker
}

// RefreshTokenInfo represents a stored refresh token
type RefreshTokenInfo struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// refresh tokens command constants
const (
	CmdCreateRefreshTokensTable engine.DBCmd = iota + 900
	CmdCreateRefreshTokensIndexes
)

var refreshTokensQueries = engine.NewQueryMap().
	Add(CmdCreateRefreshTokensTable, engine.Query{
		Sqlite: `CREATE TABLE IF NOT EXISTS refresh_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		Postgres: `CREATE TABLE IF NOT EXISTS refresh_tokens (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}).
	AddSame(CmdCreateRefreshTokensIndexes,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
		 CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
		 CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at)`,
	)

// NewRefreshTokens creates a new RefreshTokens storage
func NewRefreshTokens(ctx context.Context, db *engine.SQL) (*RefreshTokens, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	res := &RefreshTokens{SQL: db, RWLocker: db.MakeLock()}
	cfg := engine.TableConfig{
		Name:          "refresh_tokens",
		CreateTable:   CmdCreateRefreshTokensTable,
		CreateIndexes: CmdCreateRefreshTokensIndexes,
		MigrateFunc:   func(_ context.Context, _ *sqlx.Tx, _ string) error { return nil },
		QueriesMap:    refreshTokensQueries,
	}
	if err := engine.InitTable(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("failed to init refresh tokens storage: %w", err)
	}
	return res, nil
}

// Create stores a new refresh token
func (rt *RefreshTokens) Create(ctx context.Context, token RefreshTokenInfo) (int64, error) {
	rt.Lock()
	defer rt.Unlock()

	if rt.Type() == engine.Postgres {
		var id int64
		err := rt.QueryRowContext(ctx,
			"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id",
			token.UserID, token.TokenHash, token.ExpiresAt).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to create refresh token for user_id=%d: %w", token.UserID, err)
		}
		return id, nil
	}

	query := rt.Adopt("INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)")
	result, err := rt.ExecContext(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		return 0, fmt.Errorf("failed to create refresh token for user_id=%d: %w", token.UserID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

// FindByHash returns a refresh token by its hash
func (rt *RefreshTokens) FindByHash(ctx context.Context, tokenHash string) (*RefreshTokenInfo, error) {
	rt.RLock()
	defer rt.RUnlock()

	query := rt.Adopt("SELECT * FROM refresh_tokens WHERE token_hash = ?")
	var token RefreshTokenInfo
	err := rt.GetContext(ctx, &token, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	return &token, nil
}

// DeleteByHash removes a refresh token by its hash
func (rt *RefreshTokens) DeleteByHash(ctx context.Context, tokenHash string) error {
	rt.Lock()
	defer rt.Unlock()

	query := rt.Adopt("DELETE FROM refresh_tokens WHERE token_hash = ?")
	_, err := rt.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// DeleteByUserID removes all refresh tokens for a user
func (rt *RefreshTokens) DeleteByUserID(ctx context.Context, userID int64) error {
	rt.Lock()
	defer rt.Unlock()

	query := rt.Adopt("DELETE FROM refresh_tokens WHERE user_id = ?")
	_, err := rt.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh tokens for user_id=%d: %w", userID, err)
	}
	return nil
}

// CleanExpired removes all expired refresh tokens
func (rt *RefreshTokens) CleanExpired(ctx context.Context) (int64, error) {
	rt.Lock()
	defer rt.Unlock()

	query := rt.Adopt("DELETE FROM refresh_tokens WHERE expires_at < ?")
	result, err := rt.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to clean expired refresh tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	if count > 0 {
		log.Printf("[INFO] cleaned %d expired refresh tokens", count)
	}
	return count, nil
}
