package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"

	"github.com/umputun/tg-spam/app/storage/engine"
)

// ChannelSettings is a storage for per-channel spam detection settings
type ChannelSettings struct {
	*engine.SQL
	engine.RWLocker
}

// ChannelSettingsInfo represents per-channel spam detection configuration
type ChannelSettingsInfo struct {
	ID                   int64   `db:"id" json:"id"`
	GID                  string  `db:"gid" json:"gid"`
	SimilarityThreshold  float64 `db:"similarity_threshold" json:"similarity_threshold"`
	MinMsgLen            int     `db:"min_msg_len" json:"min_msg_len"`
	MaxEmoji             int     `db:"max_emoji" json:"max_emoji"`
	MinSpamProbability   float64 `db:"min_spam_probability" json:"min_spam_probability"`
	FirstMessagesCount   int     `db:"first_messages_count" json:"first_messages_count"`
	ParanoidMode         bool    `db:"paranoid_mode" json:"paranoid_mode"`
	CASEnabled           bool    `db:"cas_enabled" json:"cas_enabled"`
	OpenAIEnabled        bool    `db:"openai_enabled" json:"openai_enabled"`
	OpenAIModel          string  `db:"openai_model" json:"openai_model"`
	OpenAIVeto           bool    `db:"openai_veto" json:"openai_veto"`
	MetaLinksLimit       int     `db:"meta_links_limit" json:"meta_links_limit"`
	MetaLinksOnly        bool    `db:"meta_links_only" json:"meta_links_only"`
	MetaImageOnly        bool    `db:"meta_image_only" json:"meta_image_only"`
	MetaVideoOnly        bool    `db:"meta_video_only" json:"meta_video_only"`
	MetaAudioOnly        bool    `db:"meta_audio_only" json:"meta_audio_only"`
	MetaContactOnly      bool    `db:"meta_contact_only" json:"meta_contact_only"`
	MetaForward          bool    `db:"meta_forward" json:"meta_forward"`
	MetaKeyboard         bool    `db:"meta_keyboard" json:"meta_keyboard"`
	MetaUsernameSymbols  bool    `db:"meta_username_symbols" json:"meta_username_symbols"`
	MetaGiveaway         bool    `db:"meta_giveaway" json:"meta_giveaway"`
	DuplicatesThreshold  int     `db:"duplicates_threshold" json:"duplicates_threshold"`
	DuplicatesWindow     string  `db:"duplicates_window" json:"duplicates_window"`
	TrainingMode         bool    `db:"training_mode" json:"training_mode"`
	DryMode              bool    `db:"dry_mode" json:"dry_mode"`
	SoftBan              bool    `db:"soft_ban" json:"soft_ban"`
	NoSpamReply          bool    `db:"no_spam_reply" json:"no_spam_reply"`
	AggressiveCleanup    bool    `db:"aggressive_cleanup" json:"aggressive_cleanup"`
	AggressiveCleanupLim int     `db:"aggressive_cleanup_limit" json:"aggressive_cleanup_limit"`
	SuppressJoinMessage  bool    `db:"suppress_join_message" json:"suppress_join_message"`
	DeleteJoinMessages   bool    `db:"delete_join_messages" json:"delete_join_messages"`
	DeleteLeaveMessages  bool    `db:"delete_leave_messages" json:"delete_leave_messages"`
	AdminGroup           string  `db:"admin_group" json:"admin_group"`
	OpenAIToken          string  `db:"openai_token" json:"openai_token"`
	OpenAIAPIBase        string  `db:"openai_api_base" json:"openai_api_base"`
}

// channel settings command constants
const (
	CmdCreateChannelSettingsTable engine.DBCmd = iota + 800
	CmdCreateChannelSettingsIndexes
)

var channelSettingsQueries = engine.NewQueryMap().
	Add(CmdCreateChannelSettingsTable, engine.Query{
		Sqlite: `CREATE TABLE IF NOT EXISTS channel_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			gid TEXT NOT NULL UNIQUE,
			similarity_threshold REAL NOT NULL DEFAULT 0.5,
			min_msg_len INTEGER NOT NULL DEFAULT 50,
			max_emoji INTEGER NOT NULL DEFAULT 2,
			min_spam_probability REAL NOT NULL DEFAULT 50,
			first_messages_count INTEGER NOT NULL DEFAULT 1,
			paranoid_mode INTEGER NOT NULL DEFAULT 0,
			cas_enabled INTEGER NOT NULL DEFAULT 1,
			openai_enabled INTEGER NOT NULL DEFAULT 0,
			openai_model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
			openai_veto INTEGER NOT NULL DEFAULT 0,
			meta_links_limit INTEGER NOT NULL DEFAULT -1,
			meta_links_only INTEGER NOT NULL DEFAULT 0,
			meta_image_only INTEGER NOT NULL DEFAULT 0,
			meta_video_only INTEGER NOT NULL DEFAULT 0,
			meta_audio_only INTEGER NOT NULL DEFAULT 0,
			meta_contact_only INTEGER NOT NULL DEFAULT 0,
			meta_forward INTEGER NOT NULL DEFAULT 0,
			meta_keyboard INTEGER NOT NULL DEFAULT 0,
			meta_username_symbols INTEGER NOT NULL DEFAULT 0,
			meta_giveaway INTEGER NOT NULL DEFAULT 0,
			duplicates_threshold INTEGER NOT NULL DEFAULT 0,
			duplicates_window TEXT NOT NULL DEFAULT '1h',
			training_mode INTEGER NOT NULL DEFAULT 0,
			dry_mode INTEGER NOT NULL DEFAULT 0,
			soft_ban INTEGER NOT NULL DEFAULT 0,
			no_spam_reply INTEGER NOT NULL DEFAULT 0,
			aggressive_cleanup INTEGER NOT NULL DEFAULT 0,
			aggressive_cleanup_limit INTEGER NOT NULL DEFAULT 100,
			suppress_join_message INTEGER NOT NULL DEFAULT 0,
			delete_join_messages INTEGER NOT NULL DEFAULT 0,
			delete_leave_messages INTEGER NOT NULL DEFAULT 0,
			admin_group TEXT NOT NULL DEFAULT '',
			openai_token TEXT NOT NULL DEFAULT '',
			openai_api_base TEXT NOT NULL DEFAULT ''
		)`,
		Postgres: `CREATE TABLE IF NOT EXISTS channel_settings (
			id SERIAL PRIMARY KEY,
			gid TEXT NOT NULL UNIQUE,
			similarity_threshold REAL NOT NULL DEFAULT 0.5,
			min_msg_len INTEGER NOT NULL DEFAULT 50,
			max_emoji INTEGER NOT NULL DEFAULT 2,
			min_spam_probability REAL NOT NULL DEFAULT 50,
			first_messages_count INTEGER NOT NULL DEFAULT 1,
			paranoid_mode BOOLEAN NOT NULL DEFAULT false,
			cas_enabled BOOLEAN NOT NULL DEFAULT true,
			openai_enabled BOOLEAN NOT NULL DEFAULT false,
			openai_model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
			openai_veto BOOLEAN NOT NULL DEFAULT false,
			meta_links_limit INTEGER NOT NULL DEFAULT -1,
			meta_links_only BOOLEAN NOT NULL DEFAULT false,
			meta_image_only BOOLEAN NOT NULL DEFAULT false,
			meta_video_only BOOLEAN NOT NULL DEFAULT false,
			meta_audio_only BOOLEAN NOT NULL DEFAULT false,
			meta_contact_only BOOLEAN NOT NULL DEFAULT false,
			meta_forward BOOLEAN NOT NULL DEFAULT false,
			meta_keyboard BOOLEAN NOT NULL DEFAULT false,
			meta_username_symbols BOOLEAN NOT NULL DEFAULT false,
			meta_giveaway BOOLEAN NOT NULL DEFAULT false,
			duplicates_threshold INTEGER NOT NULL DEFAULT 0,
			duplicates_window TEXT NOT NULL DEFAULT '1h',
			training_mode BOOLEAN NOT NULL DEFAULT false,
			dry_mode BOOLEAN NOT NULL DEFAULT false,
			soft_ban BOOLEAN NOT NULL DEFAULT false,
			no_spam_reply BOOLEAN NOT NULL DEFAULT false,
			aggressive_cleanup BOOLEAN NOT NULL DEFAULT false,
			aggressive_cleanup_limit INTEGER NOT NULL DEFAULT 100,
			suppress_join_message BOOLEAN NOT NULL DEFAULT false,
			delete_join_messages BOOLEAN NOT NULL DEFAULT false,
			delete_leave_messages BOOLEAN NOT NULL DEFAULT false,
			admin_group TEXT NOT NULL DEFAULT '',
			openai_token TEXT NOT NULL DEFAULT '',
			openai_api_base TEXT NOT NULL DEFAULT ''
		)`,
	}).
	AddSame(CmdCreateChannelSettingsIndexes,
		`CREATE INDEX IF NOT EXISTS idx_channel_settings_gid ON channel_settings(gid)`,
	)

// NewChannelSettings creates a new ChannelSettings storage
func NewChannelSettings(ctx context.Context, db *engine.SQL) (*ChannelSettings, error) {
	if db == nil {
		return nil, fmt.Errorf("db connection is nil")
	}
	res := &ChannelSettings{SQL: db, RWLocker: db.MakeLock()}
	cfg := engine.TableConfig{
		Name:          "channel_settings",
		CreateTable:   CmdCreateChannelSettingsTable,
		CreateIndexes: CmdCreateChannelSettingsIndexes,
		MigrateFunc:   migrateChannelSettings,
		QueriesMap:    channelSettingsQueries,
	}
	if err := engine.InitTable(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("failed to init channel settings storage: %w", err)
	}
	return res, nil
}

// Create adds settings for a new channel with defaults
func (cs *ChannelSettings) Create(ctx context.Context, gid string) (int64, error) {
	cs.Lock()
	defer cs.Unlock()

	if cs.Type() == engine.Postgres {
		var id int64
		err := cs.QueryRowContext(ctx, "INSERT INTO channel_settings (gid) VALUES ($1) RETURNING id", gid).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to create channel settings for gid=%s: %w", gid, err)
		}
		log.Printf("[INFO] channel settings created for gid=%s", gid)
		return id, nil
	}

	query := cs.Adopt("INSERT INTO channel_settings (gid) VALUES (?)")
	result, err := cs.ExecContext(ctx, query, gid)
	if err != nil {
		return 0, fmt.Errorf("failed to create channel settings for gid=%s: %w", gid, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	log.Printf("[INFO] channel settings created for gid=%s", gid)
	return id, nil
}

// Get returns channel settings by gid
func (cs *ChannelSettings) Get(ctx context.Context, gid string) (*ChannelSettingsInfo, error) {
	cs.RLock()
	defer cs.RUnlock()

	query := cs.Adopt("SELECT * FROM channel_settings WHERE gid = ?")
	var settings ChannelSettingsInfo
	err := cs.GetContext(ctx, &settings, query, gid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel settings for gid=%s: %w", gid, err)
	}
	return &settings, nil
}

// Update saves channel settings
func (cs *ChannelSettings) Update(ctx context.Context, s ChannelSettingsInfo) error {
	cs.Lock()
	defer cs.Unlock()

	query := cs.Adopt(`UPDATE channel_settings SET
		similarity_threshold = ?, min_msg_len = ?, max_emoji = ?, min_spam_probability = ?,
		first_messages_count = ?, paranoid_mode = ?, cas_enabled = ?, openai_enabled = ?,
		openai_model = ?, openai_veto = ?, meta_links_limit = ?, meta_links_only = ?,
		meta_image_only = ?, meta_video_only = ?, meta_audio_only = ?, meta_contact_only = ?,
		meta_forward = ?, meta_keyboard = ?, meta_username_symbols = ?, meta_giveaway = ?,
		duplicates_threshold = ?, duplicates_window = ?, training_mode = ?, dry_mode = ?,
		soft_ban = ?, no_spam_reply = ?, aggressive_cleanup = ?, aggressive_cleanup_limit = ?,
		suppress_join_message = ?, delete_join_messages = ?, delete_leave_messages = ?,
		admin_group = ?, openai_token = ?, openai_api_base = ?
		WHERE gid = ?`)

	_, err := cs.ExecContext(ctx, query,
		s.SimilarityThreshold, s.MinMsgLen, s.MaxEmoji, s.MinSpamProbability,
		s.FirstMessagesCount, s.ParanoidMode, s.CASEnabled, s.OpenAIEnabled,
		s.OpenAIModel, s.OpenAIVeto, s.MetaLinksLimit, s.MetaLinksOnly,
		s.MetaImageOnly, s.MetaVideoOnly, s.MetaAudioOnly, s.MetaContactOnly,
		s.MetaForward, s.MetaKeyboard, s.MetaUsernameSymbols, s.MetaGiveaway,
		s.DuplicatesThreshold, s.DuplicatesWindow, s.TrainingMode, s.DryMode,
		s.SoftBan, s.NoSpamReply, s.AggressiveCleanup, s.AggressiveCleanupLim,
		s.SuppressJoinMessage, s.DeleteJoinMessages, s.DeleteLeaveMessages,
		s.AdminGroup, s.OpenAIToken, s.OpenAIAPIBase,
		s.GID)
	if err != nil {
		return fmt.Errorf("failed to update channel settings for gid=%s: %w", s.GID, err)
	}
	log.Printf("[INFO] channel settings updated for gid=%s", s.GID)
	return nil
}

// Delete removes channel settings by gid
func (cs *ChannelSettings) Delete(ctx context.Context, gid string) error {
	cs.Lock()
	defer cs.Unlock()

	query := cs.Adopt("DELETE FROM channel_settings WHERE gid = ?")
	_, err := cs.ExecContext(ctx, query, gid)
	if err != nil {
		return fmt.Errorf("failed to delete channel settings for gid=%s: %w", gid, err)
	}
	return nil
}

// migrateChannelSettings adds admin_group, openai_token, openai_api_base columns if they don't exist
func migrateChannelSettings(_ context.Context, tx *sqlx.Tx, _ string) error {
	columns := []struct {
		name string
		def  string
	}{
		{"admin_group", "TEXT NOT NULL DEFAULT ''"},
		{"openai_token", "TEXT NOT NULL DEFAULT ''"},
		{"openai_api_base", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, col := range columns {
		var count int
		checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM channel_settings WHERE %s = '' OR %s IS NOT NULL", col.name, col.name)
		if err := tx.Get(&count, checkQuery); err == nil {
			continue // column already exists
		}
		alterQuery := fmt.Sprintf("ALTER TABLE channel_settings ADD COLUMN %s %s", col.name, col.def)
		if _, err := tx.Exec(alterQuery); err != nil {
			return fmt.Errorf("failed to add %s column to channel_settings: %w", col.name, err)
		}
		log.Printf("[INFO] channel_settings table migrated: added %s column", col.name)
	}
	return nil
}
