package events

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	tbapi "github.com/OvyFlash/telegram-bot-api"
)

// ChannelConfig holds all parameters needed to start a listener for a channel
type ChannelConfig struct {
	GID                     string       // unique group identifier
	Group                   string       // telegram group name or ID
	AdminGroup              string       // admin group name or ID
	Bot                     Bot          // spam filter bot for this channel
	SpamLogger              SpamLogger   // spam logger for this channel
	Locator                 Locator      // message locator
	TbAPI                   TbAPI        // telegram API client
	BotUsername             string       // bot username
	SuperUsers              SuperUsers   // list of super users
	ReportConfig            ReportConfig // user report config
	NoSpamReply             bool
	SuppressJoinMessage     bool
	DeleteJoinMessages      bool
	DeleteLeaveMessages     bool
	TrainingMode            bool
	SoftBanMode             bool
	Dry                     bool
	DisableAdminSpamForward bool
	AggressiveCleanup       bool
	AggressiveCleanupLimit  int
	StartupMsg              string
	WarnMsg                 string
}

type runningChannel struct {
	config ChannelConfig
	cancel context.CancelFunc
	done   chan struct{}
}

// ChannelManager manages lifecycle of per-channel TelegramListener instances
type ChannelManager struct {
	channels map[string]*runningChannel
	mu       sync.Mutex
}

// NewChannelManager creates a new ChannelManager
func NewChannelManager() *ChannelManager {
	return &ChannelManager{
		channels: make(map[string]*runningChannel),
	}
}

// AddChannel starts a listener for a new channel
func (m *ChannelManager) AddChannel(cfg ChannelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.channels[cfg.GID]; exists {
		return fmt.Errorf("channel %s is already running", cfg.GID)
	}

	m.channels[cfg.GID] = m.startListener(cfg)
	log.Printf("[INFO] channel %s (%s) started", cfg.GID, cfg.Group)
	return nil
}

// RemoveChannel stops and removes a channel listener
func (m *ChannelManager) RemoveChannel(gid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rc, exists := m.channels[gid]
	if !exists {
		return fmt.Errorf("channel %s is not running", gid)
	}

	rc.cancel()
	<-rc.done
	delete(m.channels, gid)
	log.Printf("[INFO] channel %s stopped and removed", gid)
	return nil
}

// RestartChannel stops and restarts a channel with updated config
func (m *ChannelManager) RestartChannel(gid string, cfg ChannelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rc, exists := m.channels[gid]; exists {
		rc.cancel()
		<-rc.done
		delete(m.channels, gid)
	}

	m.channels[gid] = m.startListener(cfg)
	log.Printf("[INFO] channel %s (%s) restarted", cfg.GID, cfg.Group)
	return nil
}

// StopAll stops all running channel listeners
func (m *ChannelManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for gid, rc := range m.channels {
		rc.cancel()
		<-rc.done
		log.Printf("[INFO] channel %s stopped", gid)
	}

	m.channels = make(map[string]*runningChannel)
	log.Printf("[INFO] all channels stopped")
}

// GetChannelBot returns the Bot for a running channel, or nil if not found
func (m *ChannelManager) GetChannelBot(gid string) Bot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rc, exists := m.channels[gid]; exists {
		return rc.config.Bot
	}
	return nil
}

// UnbanUser unbans a user in the Telegram group for the specified channel
func (m *ChannelManager) UnbanUser(gid string, userID int64) error {
	m.mu.Lock()
	rc, exists := m.channels[gid]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("channel %s is not running", gid)
	}

	chatID, err := strconv.ParseInt(rc.config.Group, 10, 64)
	if err != nil {
		return fmt.Errorf("can't parse group chat ID %q for channel %s: %w", rc.config.Group, gid, err)
	}

	cfg := tbapi.UnbanChatMemberConfig{
		ChatMemberConfig: tbapi.ChatMemberConfig{
			UserID:     userID,
			ChatConfig: tbapi.ChatConfig{ChatID: chatID},
		},
		OnlyIfBanned: true,
	}
	if _, err := rc.config.TbAPI.Request(cfg); err != nil {
		return fmt.Errorf("telegram unban failed for user %d in channel %s: %w", userID, gid, err)
	}

	log.Printf("[INFO] user %d unbanned via web admin in channel %s", userID, gid)
	return nil
}

// Running returns list of running channel GIDs
func (m *ChannelManager) Running() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	gids := make([]string, 0, len(m.channels))
	for gid := range m.channels {
		gids = append(gids, gid)
	}
	return gids
}

func (m *ChannelManager) startListener(cfg ChannelConfig) *runningChannel {
	listener := &TelegramListener{
		TbAPI:                   cfg.TbAPI,
		BotUsername:             cfg.BotUsername,
		Group:                   cfg.Group,
		AdminGroup:              cfg.AdminGroup,
		Bot:                     cfg.Bot,
		SpamLogger:              cfg.SpamLogger,
		Locator:                 cfg.Locator,
		SuperUsers:              cfg.SuperUsers,
		ReportConfig:            cfg.ReportConfig,
		NoSpamReply:             cfg.NoSpamReply,
		SuppressJoinMessage:     cfg.SuppressJoinMessage,
		DeleteJoinMessages:      cfg.DeleteJoinMessages,
		DeleteLeaveMessages:     cfg.DeleteLeaveMessages,
		TrainingMode:            cfg.TrainingMode,
		SoftBanMode:             cfg.SoftBanMode,
		Dry:                     cfg.Dry,
		DisableAdminSpamForward: cfg.DisableAdminSpamForward,
		AggressiveCleanup:       cfg.AggressiveCleanup,
		AggressiveCleanupLimit:  cfg.AggressiveCleanupLimit,
		StartupMsg:              cfg.StartupMsg,
		WarnMsg:                 cfg.WarnMsg,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		if err := listener.Do(ctx); err != nil {
			log.Printf("[WARN] listener for channel %s (%s) exited: %v", cfg.GID, cfg.Group, err)
		}
	}()

	return &runningChannel{
		config: cfg,
		cancel: cancel,
		done:   done,
	}
}
