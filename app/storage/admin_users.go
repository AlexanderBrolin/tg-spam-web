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

// AdminUsers is a storage for web admin panel users
type AdminUsers struct {
	*engine.SQL
	engine.RWLocker
}

// AdminUserInfo represents a web admin user
type AdminUserInfo struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	DisplayName  string    `db:"display_name" json:"display_name"`
	Active       bool      `db:"active" json:"active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	LastLogin    NullTime  `db:"last_login" json:"last_login"`
}

// NullTime is a nullable time wrapper for JSON serialization
type NullTime struct {
	sql.NullTime
}

// admin users command constants
const (
	CmdCreateAdminUsersTable engine.DBCmd = iota + 600
	CmdCreateAdminUsersIndexes
)

var adminUsersQueries = engine.NewQueryMap().
	Add(CmdCreateAdminUsersTable, engine.Query{
		Sqlite: `CREATE TABLE IF NOT EXISTS admin_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'moderator',
			display_name TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME
		)`,
		Postgres: `CREATE TABLE IF NOT EXISTS admin_users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'moderator',
			display_name TEXT NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP
		)`,
	}).
	AddSame(CmdCreateAdminUsersIndexes,
		`CREATE INDEX IF NOT EXISTS idx_admin_users_username ON admin_users(username);
		 CREATE INDEX IF NOT EXISTS idx_admin_users_role ON admin_users(role)`,
	)

// NewAdminUsers creates a new AdminUsers storage
func NewAdminUsers(ctx context.Context, db *engine.SQL) (*AdminUsers, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	res := &AdminUsers{SQL: db, RWLocker: db.MakeLock()}
	cfg := engine.TableConfig{
		Name:          "admin_users",
		CreateTable:   CmdCreateAdminUsersTable,
		CreateIndexes: CmdCreateAdminUsersIndexes,
		MigrateFunc:   func(_ context.Context, _ *sqlx.Tx, _ string) error { return nil },
		QueriesMap:    adminUsersQueries,
	}
	if err := engine.InitTable(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("failed to init admin users storage: %w", err)
	}
	return res, nil
}

// Create adds a new admin user
func (au *AdminUsers) Create(ctx context.Context, user AdminUserInfo) (int64, error) {
	au.Lock()
	defer au.Unlock()

	if au.Type() == engine.Postgres {
		query := `INSERT INTO admin_users (username, password_hash, role, display_name, active)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`
		var id int64
		err := au.QueryRowContext(ctx, query, user.Username, user.PasswordHash, user.Role, user.DisplayName, user.Active).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to create admin user %q: %w", user.Username, err)
		}
		log.Printf("[INFO] admin user created: %s, role: %s", user.Username, user.Role)
		return id, nil
	}

	query := au.Adopt(`INSERT INTO admin_users (username, password_hash, role, display_name, active)
		VALUES (?, ?, ?, ?, ?)`)
	result, err := au.ExecContext(ctx, query, user.Username, user.PasswordHash, user.Role, user.DisplayName, user.Active)
	if err != nil {
		return 0, fmt.Errorf("failed to create admin user %q: %w", user.Username, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	log.Printf("[INFO] admin user created: %s, role: %s", user.Username, user.Role)
	return id, nil
}

// FindByUsername returns an admin user by username
func (au *AdminUsers) FindByUsername(ctx context.Context, username string) (*AdminUserInfo, error) {
	au.RLock()
	defer au.RUnlock()

	query := au.Adopt("SELECT * FROM admin_users WHERE username = ?")
	var user AdminUserInfo
	err := au.GetContext(ctx, &user, query, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find admin user %q: %w", username, err)
	}
	return &user, nil
}

// FindByID returns an admin user by id
func (au *AdminUsers) FindByID(ctx context.Context, id int64) (*AdminUserInfo, error) {
	au.RLock()
	defer au.RUnlock()

	query := au.Adopt("SELECT * FROM admin_users WHERE id = ?")
	var user AdminUserInfo
	err := au.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find admin user id=%d: %w", id, err)
	}
	return &user, nil
}

// List returns all admin users
func (au *AdminUsers) List(ctx context.Context) ([]AdminUserInfo, error) {
	au.RLock()
	defer au.RUnlock()

	query := "SELECT * FROM admin_users ORDER BY created_at"
	var users []AdminUserInfo
	if err := au.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("failed to list admin users: %w", err)
	}
	return users, nil
}

// Update modifies an existing admin user
func (au *AdminUsers) Update(ctx context.Context, user AdminUserInfo) error {
	au.Lock()
	defer au.Unlock()

	query := au.Adopt(`UPDATE admin_users SET username = ?, role = ?, display_name = ?, active = ?,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := au.ExecContext(ctx, query, user.Username, user.Role, user.DisplayName, user.Active, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update admin user id=%d: %w", user.ID, err)
	}
	return nil
}

// UpdatePassword updates the password hash for an admin user
func (au *AdminUsers) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	au.Lock()
	defer au.Unlock()

	query := au.Adopt("UPDATE admin_users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")
	_, err := au.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("failed to update password for admin user id=%d: %w", id, err)
	}
	return nil
}

// UpdateLastLogin updates the last login time for an admin user
func (au *AdminUsers) UpdateLastLogin(ctx context.Context, id int64) error {
	au.Lock()
	defer au.Unlock()

	query := au.Adopt("UPDATE admin_users SET last_login = CURRENT_TIMESTAMP WHERE id = ?")
	_, err := au.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last login for admin user id=%d: %w", id, err)
	}
	return nil
}

// Delete removes an admin user by id
func (au *AdminUsers) Delete(ctx context.Context, id int64) error {
	au.Lock()
	defer au.Unlock()

	query := au.Adopt("DELETE FROM admin_users WHERE id = ?")
	_, err := au.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete admin user id=%d: %w", id, err)
	}
	log.Printf("[INFO] admin user id=%d deleted", id)
	return nil
}

// Count returns the total number of admin users
func (au *AdminUsers) Count(ctx context.Context) (int, error) {
	au.RLock()
	defer au.RUnlock()

	var count int
	if err := au.GetContext(ctx, &count, "SELECT COUNT(*) FROM admin_users"); err != nil {
		return 0, fmt.Errorf("failed to count admin users: %w", err)
	}
	return count, nil
}
