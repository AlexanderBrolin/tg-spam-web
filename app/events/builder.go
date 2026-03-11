package events

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	tbapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/sashabaranov/go-openai"

	"github.com/umputun/tg-spam/app/bot"
	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/lib/approved"
	"github.com/umputun/tg-spam/lib/tgspam"
)

// ChannelBuilder constructs ChannelConfig from storage data and shared dependencies
type ChannelBuilder struct {
	SamplesStore       *storage.Samples
	DictStore          *storage.Dictionary
	DetectedSpamStore  *storage.DetectedSpam
	ApprovedUsersStore *storage.ApprovedUsers
	Locator            Locator
	SuperUsers         SuperUsers

	// defaults for fields not in channel settings
	SpamMsg    string
	SpamDryMsg string
}

// Build creates a ChannelConfig from storage data
func (b *ChannelBuilder) Build(
	ch storage.ChannelInfo,
	settings storage.ChannelSettingsInfo,
	botInfo storage.BotInfo,
) (ChannelConfig, error) {
	// create telegram bot API client from token
	api, err := tbapi.NewBotAPI(botInfo.Token)
	if err != nil {
		return ChannelConfig{}, fmt.Errorf("can't create bot API for %q: %w", botInfo.Name, err)
	}

	// build per-channel detector from settings
	detector := buildDetector(settings)

	// load approved users for this channel into the detector
	if b.ApprovedUsersStore != nil {
		userStore := &approvedUsersGIDAdapter{store: b.ApprovedUsersStore, gid: ch.GID}
		if count, err := detector.WithUserStorage(userStore); err != nil {
			log.Printf("[WARN] can't load approved users for gid=%s: %v", ch.GID, err)
		} else {
			log.Printf("[INFO] loaded %d approved users for gid=%s", count, ch.GID)
		}
	}

	// create per-channel SpamFilter wrapping the detector
	spamBot := bot.NewSpamFilter(detector, bot.SpamConfig{
		GroupID:      ch.GID,
		SamplesStore: b.SamplesStore,
		DictStore:    b.DictStore,
		SpamMsg:      b.SpamMsg,
		SpamDryMsg:   b.SpamDryMsg,
		Dry:          settings.DryMode,
	})

	if err := spamBot.ReloadSamples(); err != nil {
		return ChannelConfig{}, fmt.Errorf("can't reload samples for %q: %w", ch.GID, err)
	}

	// set detector samples updaters with shared stores
	detector.WithSpamUpdater(storage.NewSampleUpdater(b.SamplesStore, storage.SampleTypeSpam, 0))
	detector.WithHamUpdater(storage.NewSampleUpdater(b.SamplesStore, storage.SampleTypeHam, 0))

	// determine group identifier for telegram.
	// prefer TelegramID (explicit numeric ID of the group to monitor) over Username,
	// because Username resolves to the channel itself, not the linked discussion group.
	group := fmt.Sprintf("%d", ch.TelegramID)
	if ch.TelegramID == 0 && ch.Username != "" {
		group = ch.Username
	}

	// build spam logger that writes to detected_spam store
	spamLogger := b.makeSpamLogger(ch.GID)

	cfg := ChannelConfig{
		GID:                    ch.GID,
		Group:                  group,
		AdminGroup:             settings.AdminGroup,
		Bot:                    spamBot,
		SpamLogger:             spamLogger,
		Locator:                b.Locator,
		TbAPI:                  api,
		BotUsername:            botInfo.Username,
		SuperUsers:             b.SuperUsers,
		NoSpamReply:            settings.NoSpamReply,
		SuppressJoinMessage:    settings.SuppressJoinMessage,
		DeleteJoinMessages:     settings.DeleteJoinMessages,
		DeleteLeaveMessages:    settings.DeleteLeaveMessages,
		TrainingMode:           settings.TrainingMode,
		SoftBanMode:            settings.SoftBan,
		Dry:                    settings.DryMode,
		AggressiveCleanup:      settings.AggressiveCleanup,
		AggressiveCleanupLimit: settings.AggressiveCleanupLim,
	}

	log.Printf("[INFO] channel config built for gid=%s, group=%s, bot=%s", ch.GID, group, botInfo.Username)
	return cfg, nil
}

// buildDetector creates a per-channel tgspam.Detector from channel settings
func buildDetector(s storage.ChannelSettingsInfo) *tgspam.Detector {
	detectorConfig := tgspam.Config{
		MaxAllowedEmoji:     s.MaxEmoji,
		MinMsgLen:           s.MinMsgLen,
		SimilarityThreshold: s.SimilarityThreshold,
		MinSpamProbability:  s.MinSpamProbability,
		FirstMessageOnly:    !s.ParanoidMode,
		FirstMessagesCount:  s.FirstMessagesCount,
		OpenAIVeto:          s.OpenAIVeto,
	}

	if s.FirstMessagesCount > 0 {
		detectorConfig.FirstMessageOnly = true
	}
	if s.ParanoidMode {
		detectorConfig.FirstMessageOnly = false
		detectorConfig.FirstMessagesCount = 0
	}

	// duplicate detection
	if s.DuplicatesThreshold > 0 {
		detectorConfig.DuplicateDetection.Threshold = s.DuplicatesThreshold
		dur, err := time.ParseDuration(s.DuplicatesWindow)
		if err == nil {
			detectorConfig.DuplicateDetection.Window = dur
		}
	}

	if s.CASEnabled {
		detectorConfig.CasAPI = "https://api.cas.chat"
		detectorConfig.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	detector := tgspam.NewDetector(detectorConfig)

	// setup per-channel OpenAI if enabled and token is provided
	if s.OpenAIEnabled && s.OpenAIToken != "" {
		config := openai.DefaultConfig(s.OpenAIToken)
		if s.OpenAIAPIBase != "" {
			config.BaseURL = s.OpenAIAPIBase
		}
		openAIConfig := tgspam.OpenAIConfig{
			Model: s.OpenAIModel,
		}
		detector.WithOpenAIChecker(openai.NewClientWithConfig(config), openAIConfig)
	}

	// meta checks
	var metaChecks []tgspam.MetaCheck
	if s.MetaImageOnly {
		metaChecks = append(metaChecks, tgspam.ImagesCheck(s.MinMsgLen))
	}
	if s.MetaVideoOnly {
		metaChecks = append(metaChecks, tgspam.VideosCheck(s.MinMsgLen))
	}
	if s.MetaAudioOnly {
		metaChecks = append(metaChecks, tgspam.AudioCheck(s.MinMsgLen))
	}
	if s.MetaLinksLimit >= 0 {
		metaChecks = append(metaChecks, tgspam.LinksCheck(s.MetaLinksLimit))
	}
	if s.MetaLinksOnly {
		metaChecks = append(metaChecks, tgspam.LinkOnlyCheck())
	}
	if s.MetaForward {
		metaChecks = append(metaChecks, tgspam.ForwardedCheck())
	}
	if s.MetaKeyboard {
		metaChecks = append(metaChecks, tgspam.KeyboardCheck())
	}
	if s.MetaContactOnly {
		metaChecks = append(metaChecks, tgspam.ContactCheck())
	}
	if s.MetaGiveaway {
		metaChecks = append(metaChecks, tgspam.GiveawayCheck())
	}
	detector.WithMetaChecks(metaChecks...)

	return detector
}

// approvedUsersGIDAdapter adapts ApprovedUsers storage to tgspam.UserStorage for a specific GID.
// Write and Delete are no-ops because the detector auto-writes on every non-spam message,
// which would pollute the approved users list. DB writes/deletes are handled explicitly
// by the web admin handlers via ApprovedUsersStore + Bot.AddApprovedUser/RemoveApprovedUser.
type approvedUsersGIDAdapter struct {
	store *storage.ApprovedUsers
	gid   string
}

func (a *approvedUsersGIDAdapter) Read(ctx context.Context) ([]approved.UserInfo, error) {
	return a.store.ReadByGID(ctx, a.gid)
}

func (a *approvedUsersGIDAdapter) Write(_ context.Context, _ approved.UserInfo) error {
	return nil
}

func (a *approvedUsersGIDAdapter) Delete(_ context.Context, _ string) error {
	return nil
}

// makeSpamLogger creates a SpamLogger that writes detected spam to the DB store
func (b *ChannelBuilder) makeSpamLogger(gid string) SpamLogger {
	if b.DetectedSpamStore == nil {
		return SpamLoggerFunc(func(_ *bot.Message, _ *bot.Response) {})
	}
	return SpamLoggerFunc(func(msg *bot.Message, response *bot.Response) {
		// use channel identity when message is from a channel, so that spam records
		// are correctly associated with the actual channel, not the shared system user (e.g., 777000)
		userID := msg.From.ID
		userName := msg.From.Username
		if msg.SenderChat.ID != 0 {
			userID = msg.SenderChat.ID
			userName = msg.SenderChat.UserName
		}
		if userName == "" {
			userName = msg.From.DisplayName
		}
		text := strings.ReplaceAll(msg.Text, "\n", " ")
		text = strings.TrimSpace(text)
		log.Printf("[DEBUG] spam detected from %v in gid=%s, text: %s", msg.From, gid, text)

		rec := storage.DetectedSpamInfo{
			Text:      text,
			UserID:    userID,
			UserName:  userName,
			Timestamp: time.Now().In(time.Local),
			GID:       gid,
		}
		if err := b.DetectedSpamStore.Write(context.Background(), rec, response.CheckResults); err != nil {
			log.Printf("[WARN] can't write detected spam to db for gid=%s: %v", gid, err)
		}
	})
}
