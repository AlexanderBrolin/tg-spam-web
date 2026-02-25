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

// Channels is a storage for registered Telegram channels/groups
type Channels struct {
	*engine.SQL
	engine.RWLocker
}

// ChannelInfo represents a registered Telegram channel or group
type ChannelInfo struct {
	ID         int64     `db:"id" json:"id"`
	GID        string    `db:"gid" json:"gid"`
	TelegramID int64     `db:"telegram_id" json:"telegram_id"`
	BotID      int64     `db:"bot_id" json:"bot_id"`
	Name       string    `db:"name" json:"name"`
	Username   string    `db:"username" json:"username"`
	Active     bool      `db:"active" json:"active"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// channels command constants
const (
	CmdCreateChannelsTable engine.DBCmd = iota + 700
	CmdCreateChannelsIndexes
)

var channelsQueries = engine.NewQueryMap().
	Add(CmdCreateChannelsTable, engine.Query{
		Sqlite: `CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gid TEXT NOT NULL UNIQUE,
			telegram_id INTEGER NOT NULL,
			bot_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		Postgres: `CREATE TABLE IF NOT EXISTS channels (
			id SERIAL PRIMARY KEY,
			gid TEXT NOT NULL UNIQUE,
			telegram_id BIGINT NOT NULL,
			bot_id BIGINT NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}).
	AddSame(CmdCreateChannelsIndexes,
		`CREATE INDEX IF NOT EXISTS idx_channels_gid ON channels(gid);
		 CREATE INDEX IF NOT EXISTS idx_channels_telegram_id ON channels(telegram_id);
		 CREATE INDEX IF NOT EXISTS idx_channels_active ON channels(active)`,
	)

// NewChannels creates a new Channels storage
func NewChannels(ctx context.Context, db *engine.SQL) (*Channels, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	res := &Channels{SQL: db, RWLocker: db.MakeLock()}
	cfg := engine.TableConfig{
		Name:          "channels",
		CreateTable:   CmdCreateChannelsTable,
		CreateIndexes: CmdCreateChannelsIndexes,
		MigrateFunc:   migrateChannels,
		QueriesMap:    channelsQueries,
	}
	if err := engine.InitTable(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("failed to init channels storage: %w", err)
	}
	return res, nil
}

// Create adds a new channel
func (ch *Channels) Create(ctx context.Context, channel ChannelInfo) (int64, error) {
	ch.Lock()
	defer ch.Unlock()

	if ch.Type() == engine.Postgres {
		query := `INSERT INTO channels (gid, telegram_id, bot_id, name, username, active)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
		var id int64
		err := ch.QueryRowContext(ctx, query, channel.GID, channel.TelegramID, channel.BotID,
			channel.Name, channel.Username, channel.Active).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to create channel %q: %w", channel.Name, err)
		}
		log.Printf("[INFO] channel created: %s (gid=%s, telegram_id=%d)", channel.Name, channel.GID, channel.TelegramID)
		return id, nil
	}

	query := ch.Adopt(`INSERT INTO channels (gid, telegram_id, bot_id, name, username, active)
		VALUES (?, ?, ?, ?, ?, ?)`)
	result, err := ch.ExecContext(ctx, query, channel.GID, channel.TelegramID, channel.BotID,
		channel.Name, channel.Username, channel.Active)
	if err != nil {
		return 0, fmt.Errorf("failed to create channel %q: %w", channel.Name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	log.Printf("[INFO] channel created: %s (gid=%s, telegram_id=%d)", channel.Name, channel.GID, channel.TelegramID)
	return id, nil
}

// FindByID returns a channel by its database ID
func (ch *Channels) FindByID(ctx context.Context, id int64) (*ChannelInfo, error) {
	ch.RLock()
	defer ch.RUnlock()

	query := ch.Adopt("SELECT * FROM channels WHERE id = ?")
	var channel ChannelInfo
	err := ch.GetContext(ctx, &channel, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find channel id=%d: %w", id, err)
	}
	return &channel, nil
}

// FindByGID returns a channel by its gid
func (ch *Channels) FindByGID(ctx context.Context, gid string) (*ChannelInfo, error) {
	ch.RLock()
	defer ch.RUnlock()

	query := ch.Adopt("SELECT * FROM channels WHERE gid = ?")
	var channel ChannelInfo
	err := ch.GetContext(ctx, &channel, query, gid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find channel gid=%s: %w", gid, err)
	}
	return &channel, nil
}

// FindByTelegramID returns a channel by its Telegram chat ID
func (ch *Channels) FindByTelegramID(ctx context.Context, telegramID int64) (*ChannelInfo, error) {
	ch.RLock()
	defer ch.RUnlock()

	query := ch.Adopt("SELECT * FROM channels WHERE telegram_id = ?")
	var channel ChannelInfo
	err := ch.GetContext(ctx, &channel, query, telegramID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find channel telegram_id=%d: %w", telegramID, err)
	}
	return &channel, nil
}

// List returns all channels
func (ch *Channels) List(ctx context.Context) ([]ChannelInfo, error) {
	ch.RLock()
	defer ch.RUnlock()

	var channels []ChannelInfo
	if err := ch.SelectContext(ctx, &channels, "SELECT * FROM channels ORDER BY created_at"); err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	return channels, nil
}

// ListActive returns only active channels
func (ch *Channels) ListActive(ctx context.Context) ([]ChannelInfo, error) {
	ch.RLock()
	defer ch.RUnlock()

	query := ch.Adopt("SELECT * FROM channels WHERE active = ? ORDER BY created_at")
	var channels []ChannelInfo
	if err := ch.SelectContext(ctx, &channels, query, true); err != nil {
		return nil, fmt.Errorf("failed to list active channels: %w", err)
	}
	return channels, nil
}

// Update modifies an existing channel
func (ch *Channels) Update(ctx context.Context, channel ChannelInfo) error {
	ch.Lock()
	defer ch.Unlock()

	query := ch.Adopt(`UPDATE channels SET name = ?, username = ?, bot_id = ?, active = ?,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := ch.ExecContext(ctx, query, channel.Name, channel.Username, channel.BotID, channel.Active, channel.ID)
	if err != nil {
		return fmt.Errorf("failed to update channel id=%d: %w", channel.ID, err)
	}
	return nil
}

// Delete removes a channel by id
func (ch *Channels) Delete(ctx context.Context, id int64) error {
	ch.Lock()
	defer ch.Unlock()

	query := ch.Adopt("DELETE FROM channels WHERE id = ?")
	_, err := ch.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel id=%d: %w", id, err)
	}
	log.Printf("[INFO] channel id=%d deleted", id)
	return nil
}

// migrateChannels adds bot_id column if it doesn't exist
func migrateChannels(_ context.Context, tx *sqlx.Tx, _ string) error {
	var count int
	if err := tx.Get(&count, "SELECT COUNT(*) FROM channels WHERE bot_id = 0 OR bot_id IS NOT NULL"); err == nil {
		return nil // bot_id column already exists
	}
	if _, err := tx.Exec("ALTER TABLE channels ADD COLUMN bot_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("failed to add bot_id column to channels: %w", err)
	}
	log.Printf("[INFO] channels table migrated: added bot_id column")
	return nil
}
