package events

import (
	"fmt"
	"log"
	"time"

	tbapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/sashabaranov/go-openai"

	"github.com/umputun/tg-spam/app/bot"
	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/lib/tgspam"
)

// ChannelBuilder constructs ChannelConfig from storage data and shared dependencies
type ChannelBuilder struct {
	SamplesStore *storage.Samples
	DictStore    *storage.Dictionary
	Locator      Locator
	SuperUsers   SuperUsers

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

	// determine group identifier for telegram
	group := ch.Username
	if group == "" {
		group = fmt.Sprintf("%d", ch.TelegramID)
	}

	cfg := ChannelConfig{
		GID:                    ch.GID,
		Group:                  group,
		AdminGroup:             settings.AdminGroup,
		Bot:                    spamBot,
		SpamLogger:             SpamLoggerFunc(func(_ *bot.Message, _ *bot.Response) {}), // no-op logger by default
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
