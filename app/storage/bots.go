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

// Bots is a storage for registered Telegram bot tokens
type Bots struct {
	*engine.SQL
	engine.RWLocker
}

// BotInfo represents a registered Telegram bot
type BotInfo struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Token     string    `db:"token" json:"token"`
	Username  string    `db:"username" json:"username"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// bots command constants
const (
	CmdCreateBotsTable engine.DBCmd = iota + 1000
	CmdCreateBotsIndexes
)

var botsQueries = engine.NewQueryMap().
	Add(CmdCreateBotsTable, engine.Query{
		Sqlite: `CREATE TABLE IF NOT EXISTS bots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		Postgres: `CREATE TABLE IF NOT EXISTS bots (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}).
	AddSame(CmdCreateBotsIndexes,
		`CREATE INDEX IF NOT EXISTS idx_bots_active ON bots(active);
		 CREATE INDEX IF NOT EXISTS idx_bots_token ON bots(token)`,
	)

// NewBots creates a new Bots storage
func NewBots(ctx context.Context, db *engine.SQL) (*Bots, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	res := &Bots{SQL: db, RWLocker: db.MakeLock()}
	cfg := engine.TableConfig{
		Name:          "bots",
		CreateTable:   CmdCreateBotsTable,
		CreateIndexes: CmdCreateBotsIndexes,
		MigrateFunc:   func(_ context.Context, _ *sqlx.Tx, _ string) error { return nil },
		QueriesMap:    botsQueries,
	}
	if err := engine.InitTable(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("failed to init bots storage: %w", err)
	}
	return res, nil
}

// Create adds a new bot
func (b *Bots) Create(ctx context.Context, bot BotInfo) (int64, error) {
	b.Lock()
	defer b.Unlock()

	if b.Type() == engine.Postgres {
		query := `INSERT INTO bots (name, token, username, active)
			VALUES ($1, $2, $3, $4) RETURNING id`
		var id int64
		err := b.QueryRowContext(ctx, query, bot.Name, bot.Token, bot.Username, bot.Active).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to create bot %q: %w", bot.Name, err)
		}
		log.Printf("[INFO] bot created: %s (username=%s)", bot.Name, bot.Username)
		return id, nil
	}

	query := b.Adopt(`INSERT INTO bots (name, token, username, active)
		VALUES (?, ?, ?, ?)`)
	result, err := b.ExecContext(ctx, query, bot.Name, bot.Token, bot.Username, bot.Active)
	if err != nil {
		return 0, fmt.Errorf("failed to create bot %q: %w", bot.Name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	log.Printf("[INFO] bot created: %s (username=%s)", bot.Name, bot.Username)
	return id, nil
}

// FindByID returns a bot by its id
func (b *Bots) FindByID(ctx context.Context, id int64) (*BotInfo, error) {
	b.RLock()
	defer b.RUnlock()

	query := b.Adopt("SELECT * FROM bots WHERE id = ?")
	var bot BotInfo
	err := b.GetContext(ctx, &bot, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find bot id=%d: %w", id, err)
	}
	return &bot, nil
}

// List returns all bots
func (b *Bots) List(ctx context.Context) ([]BotInfo, error) {
	b.RLock()
	defer b.RUnlock()

	var bots []BotInfo
	if err := b.SelectContext(ctx, &bots, "SELECT * FROM bots ORDER BY created_at"); err != nil {
		return nil, fmt.Errorf("failed to list bots: %w", err)
	}
	return bots, nil
}

// ListActive returns only active bots
func (b *Bots) ListActive(ctx context.Context) ([]BotInfo, error) {
	b.RLock()
	defer b.RUnlock()

	query := b.Adopt("SELECT * FROM bots WHERE active = ? ORDER BY created_at")
	var bots []BotInfo
	if err := b.SelectContext(ctx, &bots, query, true); err != nil {
		return nil, fmt.Errorf("failed to list active bots: %w", err)
	}
	return bots, nil
}

// Update modifies an existing bot
func (b *Bots) Update(ctx context.Context, bot BotInfo) error {
	b.Lock()
	defer b.Unlock()

	query := b.Adopt(`UPDATE bots SET name = ?, token = ?, username = ?, active = ?,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := b.ExecContext(ctx, query, bot.Name, bot.Token, bot.Username, bot.Active, bot.ID)
	if err != nil {
		return fmt.Errorf("failed to update bot id=%d: %w", bot.ID, err)
	}
	return nil
}

// Delete removes a bot by id
func (b *Bots) Delete(ctx context.Context, id int64) error {
	b.Lock()
	defer b.Unlock()

	query := b.Adopt("DELETE FROM bots WHERE id = ?")
	_, err := b.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bot id=%d: %w", id, err)
	}
	log.Printf("[INFO] bot id=%d deleted", id)
	return nil
}
