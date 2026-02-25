// Package webapi provides a web API spam detection service.
package webapi

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/didip/tollbooth/v8"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/tg-spam/app/auth"
	"github.com/umputun/tg-spam/app/events"
	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/app/storage/engine"
	"github.com/umputun/tg-spam/lib/approved"
	"github.com/umputun/tg-spam/lib/spamcheck"
)

//go:generate moq --out mocks/detector.go --pkg mocks --with-resets --skip-ensure . Detector
//go:generate moq --out mocks/spam_filter.go --pkg mocks --with-resets --skip-ensure . SpamFilter
//go:generate moq --out mocks/locator.go --pkg mocks --with-resets --skip-ensure . Locator
//go:generate moq --out mocks/detected_spam.go --pkg mocks --with-resets --skip-ensure . DetectedSpam
//go:generate moq --out mocks/storage_engine.go --pkg mocks --with-resets --skip-ensure . StorageEngine
//go:generate moq --out mocks/dictionary.go --pkg mocks --with-resets --skip-ensure . Dictionary

// startTime tracks when the server started
var startTime = time.Now()

// Server is a web API server.
type Server struct {
	Config
}

// Config defines  server parameters
type Config struct {
	Version       string        // version to show in /ping
	ListenAddr    string        // listen address
	Detector      Detector      // spam detector
	SpamFilter    SpamFilter    // spam filter (bot)
	DetectedSpam  DetectedSpam  // detected spam accessor
	Locator       Locator       // locator for user info
	Dictionary    Dictionary    // dictionary for stop phrases and ignored words
	StorageEngine StorageEngine // database engine access for backups
	Dbg           bool          // debug mode
	Settings      Settings      // application settings

	// v2 API fields
	AuthService          AuthServiceV2          // JWT auth service for v2 API
	AdminUsersStore      AdminUsersStore        // admin users storage for v2 API
	ChannelsStore        ChannelsStore          // channels storage for v2 API
	ChannelSettingsStore ChannelSettings        // channel settings storage for v2 API
	BotsStore            BotsStore              // bots storage for v2 API
	ChannelManager       ChannelManagerV2       // manages running channel listeners
	ChannelBuilder       *events.ChannelBuilder // builds ChannelConfig from storage data
}

// Settings contains all application settings
type Settings struct {
	InstanceID               string        `json:"instance_id"`
	PrimaryGroup             string        `json:"primary_group"`
	AdminGroup               string        `json:"admin_group"`
	DisableAdminSpamForward  bool          `json:"disable_admin_spam_forward"`
	LoggerEnabled            bool          `json:"logger_enabled"`
	SuperUsers               []string      `json:"super_users"`
	NoSpamReply              bool          `json:"no_spam_reply"`
	CasEnabled               bool          `json:"cas_enabled"`
	MetaEnabled              bool          `json:"meta_enabled"`
	MetaLinksLimit           int           `json:"meta_links_limit"`
	MetaMentionsLimit        int           `json:"meta_mentions_limit"`
	MetaLinksOnly            bool          `json:"meta_links_only"`
	MetaImageOnly            bool          `json:"meta_image_only"`
	MetaVideoOnly            bool          `json:"meta_video_only"`
	MetaAudioOnly            bool          `json:"meta_audio_only"`
	MetaForwarded            bool          `json:"meta_forwarded"`
	MetaKeyboard             bool          `json:"meta_keyboard"`
	MetaContactOnly          bool          `json:"meta_contact_only"`
	MetaUsernameSymbols      string        `json:"meta_username_symbols"`
	MetaGiveaway             bool          `json:"meta_giveaway"`
	MultiLangLimit           int           `json:"multi_lang_limit"`
	OpenAIEnabled            bool          `json:"openai_enabled"`
	OpenAIVeto               bool          `json:"openai_veto"`
	OpenAIHistorySize        int           `json:"openai_history_size"`
	OpenAIModel              string        `json:"openai_model"`
	OpenAICheckShortMessages bool          `json:"openai_check_short_messages"`
	OpenAICustomPrompts      []string      `json:"openai_custom_prompts"`
	LuaPluginsEnabled        bool          `json:"lua_plugins_enabled"`
	LuaPluginsDir            string        `json:"lua_plugins_dir"`
	LuaEnabledPlugins        []string      `json:"lua_enabled_plugins"`
	LuaDynamicReload         bool          `json:"lua_dynamic_reload"`
	LuaAvailablePlugins      []string      `json:"lua_available_plugins"` // the list of all available Lua plugins
	SamplesDataPath          string        `json:"samples_data_path"`
	DynamicDataPath          string        `json:"dynamic_data_path"`
	WatchIntervalSecs        int           `json:"watch_interval_secs"`
	SimilarityThreshold      float64       `json:"similarity_threshold"`
	MinMsgLen                int           `json:"min_msg_len"`
	MaxEmoji                 int           `json:"max_emoji"`
	MinSpamProbability       float64       `json:"min_spam_probability"`
	ParanoidMode             bool          `json:"paranoid_mode"`
	FirstMessagesCount       int           `json:"first_messages_count"`
	StartupMessageEnabled    bool          `json:"startup_message_enabled"`
	TrainingEnabled          bool          `json:"training_enabled"`
	StorageTimeout           time.Duration `json:"storage_timeout"`
	SoftBanEnabled           bool          `json:"soft_ban_enabled"`
	AbnormalSpacingEnabled   bool          `json:"abnormal_spacing_enabled"`
	HistorySize              int           `json:"history_size"`
	DebugModeEnabled         bool          `json:"debug_mode_enabled"`
	DryModeEnabled           bool          `json:"dry_mode_enabled"`
	TGDebugModeEnabled       bool          `json:"tg_debug_mode_enabled"`
}

// Detector is a spam detector interface.
type Detector interface {
	Check(req spamcheck.Request) (spam bool, cr []spamcheck.Response)
	ApprovedUsers() []approved.UserInfo
	AddApprovedUser(user approved.UserInfo) error
	RemoveApprovedUser(id string) error
	GetLuaPluginNames() []string // Returns the list of available Lua plugin names
}

// SpamFilter is a spam filter, bot interface.
type SpamFilter interface {
	UpdateSpam(msg string) error
	UpdateHam(msg string) error
	ReloadSamples() (err error)
	DynamicSamples() (spam, ham []string, err error)
	RemoveDynamicSpamSample(sample string) error
	RemoveDynamicHamSample(sample string) error
}

// Locator is a storage interface used to get user id by name and vice versa.
type Locator interface {
	UserIDByName(ctx context.Context, userName string) int64
	UserNameByID(ctx context.Context, userID int64) string
}

// DetectedSpam is a storage interface used to get detected spam messages and set added flag.
type DetectedSpam interface {
	Read(ctx context.Context) ([]storage.DetectedSpamInfo, error)
	SetAddedToSamplesFlag(ctx context.Context, id int64) error
	FindByUserID(ctx context.Context, userID int64) (*storage.DetectedSpamInfo, error)
}

// StorageEngine provides access to the database engine for operations like backup
type StorageEngine interface {
	Backup(ctx context.Context, w io.Writer) error
	Type() engine.Type
	BackupSqliteAsPostgres(ctx context.Context, w io.Writer) error
}

// Dictionary is a storage interface for managing stop phrases and ignored words
type Dictionary interface {
	Add(ctx context.Context, t storage.DictionaryType, data string) error
	Delete(ctx context.Context, id int64) error
	Read(ctx context.Context, t storage.DictionaryType) ([]string, error)
	ReadWithIDs(ctx context.Context, t storage.DictionaryType) ([]storage.DictionaryEntry, error)
	Stats(ctx context.Context) (*storage.DictionaryStats, error)
}

// AuthServiceV2 provides JWT authentication operations for v2 API
type AuthServiceV2 interface {
	Login(ctx context.Context, username, password string) (*auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error
	ValidateAccessToken(tokenString string) (*auth.Claims, error)
	AuthMiddleware(next http.Handler) http.Handler
}

// AdminUsersStore provides access to admin user data
type AdminUsersStore interface {
	Create(ctx context.Context, user storage.AdminUserInfo) (int64, error)
	FindByID(ctx context.Context, id int64) (*storage.AdminUserInfo, error)
	FindByUsername(ctx context.Context, username string) (*storage.AdminUserInfo, error)
	List(ctx context.Context) ([]storage.AdminUserInfo, error)
	Update(ctx context.Context, user storage.AdminUserInfo) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context) (int, error)
}

// ChannelsStore provides access to channel data
type ChannelsStore interface {
	Create(ctx context.Context, channel storage.ChannelInfo) (int64, error)
	FindByID(ctx context.Context, id int64) (*storage.ChannelInfo, error)
	FindByGID(ctx context.Context, gid string) (*storage.ChannelInfo, error)
	List(ctx context.Context) ([]storage.ChannelInfo, error)
	ListActive(ctx context.Context) ([]storage.ChannelInfo, error)
	Update(ctx context.Context, channel storage.ChannelInfo) error
	Delete(ctx context.Context, id int64) error
}

// ChannelSettings provides access to per-channel settings
type ChannelSettings interface {
	Create(ctx context.Context, gid string) (int64, error)
	Get(ctx context.Context, gid string) (*storage.ChannelSettingsInfo, error)
	Update(ctx context.Context, s storage.ChannelSettingsInfo) error
	Delete(ctx context.Context, gid string) error
}

// ChannelManagerV2 manages lifecycle of per-channel TelegramListener instances
type ChannelManagerV2 interface {
	AddChannel(cfg events.ChannelConfig) error
	RemoveChannel(gid string) error
	RestartChannel(gid string, cfg events.ChannelConfig) error
	StopAll()
	Running() []string
}

// BotsStore provides access to bot data
type BotsStore interface {
	Create(ctx context.Context, bot storage.BotInfo) (int64, error)
	FindByID(ctx context.Context, id int64) (*storage.BotInfo, error)
	List(ctx context.Context) ([]storage.BotInfo, error)
	ListActive(ctx context.Context) ([]storage.BotInfo, error)
	Update(ctx context.Context, bot storage.BotInfo) error
	Delete(ctx context.Context, id int64) error
}

// NewServer creates a new web API server.
func NewServer(config Config) *Server {
	return &Server{Config: config}
}

// Run starts server and accepts requests checking for spam messages.
func (s *Server) Run(ctx context.Context) error {
	router := routegroup.New(http.NewServeMux())
	router.Use(rest.Recoverer(log.Default()))
	router.Use(logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]")).Handler)
	router.Use(rest.Throttle(1000))
	router.Use(rest.AppInfo("tg-spam", "umputun", s.Version), rest.Ping)
	router.Use(tollbooth.HTTPMiddleware(tollbooth.NewLimiter(50, nil)))
	router.Use(rest.SizeLimit(1024 * 1024))

	if s.AuthService != nil {
		s.routesV2(router)
	}

	// redirect root to admin panel
	router.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/", http.StatusFound)
	})

	srv := &http.Server{Addr: s.ListenAddr, Handler: router, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[WARN] failed to shutdown webapi server: %v", err)
		} else {
			log.Printf("[INFO] webapi server stopped")
		}
	}()

	log.Printf("[INFO] start webapi server on %s", s.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to run server: %w", err)
	}
	return nil
}

func (s *Server) routesV2(router *routegroup.Bundle) {
	// public auth endpoints (no JWT required)
	router.Mount("/api/v2/auth").Route(func(r *routegroup.Bundle) {
		r.HandleFunc("POST /login", s.loginHandler)
		r.HandleFunc("POST /refresh", s.refreshHandler)
	})

	// protected v2 API endpoints (JWT required)
	router.Mount("/api/v2").Route(func(api *routegroup.Bundle) {
		api.Use(s.AuthService.AuthMiddleware)

		// auth (protected)
		api.HandleFunc("POST /auth/logout", s.logoutHandler)
		api.HandleFunc("GET /auth/me", s.meHandler)
		api.HandleFunc("PUT /auth/password", s.changePasswordHandler)

		// stats
		api.HandleFunc("GET /stats", s.getStatsHandler)

		// spam management
		api.HandleFunc("GET /spam/detected", s.listDetectedSpamHandler)
		api.HandleFunc("POST /spam/detected/{id}/add", s.addDetectedSpamToSamplesHandler)
		api.HandleFunc("POST /spam/check", s.spamCheckHandler)

		// approved users
		api.HandleFunc("GET /users/approved", s.listApprovedUsersHandler)
		api.HandleFunc("POST /users/approved", s.addApprovedUserHandler)
		api.HandleFunc("DELETE /users/approved/{user_id}", s.removeApprovedUserHandler)

		// samples management (admin and superadmin)
		api.Mount("/samples").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin", "admin"))
			r.HandleFunc("GET /", s.getSamplesHandler)
			r.HandleFunc("POST /spam", s.addSpamSampleHandler)
			r.HandleFunc("POST /ham", s.addHamSampleHandler)
			r.HandleFunc("DELETE /spam", s.deleteSpamSampleHandler)
			r.HandleFunc("DELETE /ham", s.deleteHamSampleHandler)
			r.HandleFunc("PUT /reload", s.reloadSamplesHandler)
		})

		// dictionary management (admin and superadmin)
		api.Mount("/dictionary").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin", "admin"))
			r.HandleFunc("GET /", s.getDictionaryHandler)
			r.HandleFunc("POST /", s.addDictionaryHandler)
			r.HandleFunc("DELETE /", s.deleteDictionaryHandler)
		})

		// channels management
		api.Mount("/channels").Route(func(r *routegroup.Bundle) {
			r.HandleFunc("GET /", s.listChannelsHandler)
			r.HandleFunc("GET /{gid}/settings", s.getChannelSettingsHandler)
		})

		// channels write operations (superadmin only)
		api.Mount("/channels").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin"))
			r.HandleFunc("POST /", s.createChannelHandler)
			r.HandleFunc("PUT /{id}", s.updateChannelHandler)
			r.HandleFunc("DELETE /{id}", s.deleteChannelHandler)
		})

		// channel settings write operations (superadmin and admin)
		api.Mount("/channels").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin", "admin"))
			r.HandleFunc("PUT /{gid}/settings", s.updateChannelSettingsHandler)
		})

		// bots management (superadmin only)
		api.Mount("/bots").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin"))
			r.HandleFunc("GET /", s.listBotsHandler)
			r.HandleFunc("POST /", s.createBotHandler)
			r.HandleFunc("PUT /{id}", s.updateBotHandler)
			r.HandleFunc("DELETE /{id}", s.deleteBotHandler)
			r.HandleFunc("POST /{id}/validate", s.validateBotHandler)
		})

		// admin user management (superadmin only)
		api.Mount("/admin/users").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin"))
			r.HandleFunc("GET /", s.listAdminUsersHandler)
			r.HandleFunc("POST /", s.createAdminUserHandler)
			r.HandleFunc("PUT /{id}", s.updateAdminUserHandler)
			r.HandleFunc("DELETE /{id}", s.deleteAdminUserHandler)
			r.HandleFunc("PUT /{id}/password", s.resetAdminUserPasswordHandler)
		})

		// settings (read-only)
		api.HandleFunc("GET /settings", s.getSettingsHandler)

		// download/backup (superadmin only)
		api.Mount("/download").Route(func(r *routegroup.Bundle) {
			r.Use(auth.RequireRole("superadmin"))
			r.HandleFunc("GET /spam", s.downloadSampleHandler(func(spam, _ []string) ([]string, string) {
				return spam, "spam.txt"
			}))
			r.HandleFunc("GET /ham", s.downloadSampleHandler(func(_, ham []string) ([]string, string) {
				return ham, "ham.txt"
			}))
			r.HandleFunc("GET /detected_spam", s.downloadDetectedSpamHandler)
			r.HandleFunc("GET /backup", s.downloadBackupHandler)
			r.HandleFunc("GET /export-to-postgres", s.downloadExportToPostgresHandler)
		})
	})

	// serve React SPA for the admin panel
	spaServe := spaHandler().ServeHTTP
	router.Mount("/app").Route(func(r *routegroup.Bundle) {
		r.HandleFunc("GET /", spaServe)
		r.HandleFunc("GET /{path...}", spaServe)
	})
}

// getDynamicSamplesHandler handles GET /samples request. It returns dynamic samples both for spam and ham.
func (s *Server) getDynamicSamplesHandler(w http.ResponseWriter, _ *http.Request) {
	spam, ham, err := s.SpamFilter.DynamicSamples()
	if err != nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "can't get dynamic samples", "details": err.Error()})
		return
	}
	rest.RenderJSON(w, rest.JSON{"spam": spam, "ham": ham})
}

// reloadDynamicSamplesHandler handles PUT /samples request. It reloads dynamic samples from db storage.
func (s *Server) reloadDynamicSamplesHandler(w http.ResponseWriter, _ *http.Request) {
	if err := s.SpamFilter.ReloadSamples(); err != nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "can't reload samples", "details": err.Error()})
		return
	}
	rest.RenderJSON(w, rest.JSON{"reloaded": true})
}

// downloadSampleHandler handles GET /download/spam|ham request.
// It returns dynamic samples both for spam and ham.
func (s *Server) downloadSampleHandler(pickFn func(spam, ham []string) ([]string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		spam, ham, err := s.SpamFilter.DynamicSamples()
		if err != nil {
			_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "can't get dynamic samples", "details": err.Error()})
			return
		}
		samples, name := pickFn(spam, ham)
		body := strings.Join(samples, "\n")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}

func (s *Server) downloadDetectedSpamHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spam, err := s.DetectedSpam.Read(ctx)
	if err != nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "can't get detected spam", "details": err.Error()})
		return
	}

	type jsonSpamInfo struct {
		ID        int64                `json:"id"`
		GID       string               `json:"gid"`
		Text      string               `json:"text"`
		UserID    int64                `json:"user_id"`
		UserName  string               `json:"user_name"`
		Timestamp time.Time            `json:"timestamp"`
		Added     bool                 `json:"added"`
		Checks    []spamcheck.Response `json:"checks"`
	}

	// convert entries to jsonl format with lowercase fields
	lines := make([]string, 0, len(spam))
	for _, entry := range spam {
		data, err := json.Marshal(jsonSpamInfo{
			ID:        entry.ID,
			GID:       entry.GID,
			Text:      entry.Text,
			UserID:    entry.UserID,
			UserName:  entry.UserName,
			Timestamp: entry.Timestamp,
			Added:     entry.Added,
			Checks:    entry.Checks,
		})
		if err != nil {
			_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "can't marshal entry", "details": err.Error()})
			return
		}
		lines = append(lines, string(data))
	}

	body := strings.Join(lines, "\n")
	w.Header().Set("Content-Type", "application/x-jsonlines")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "detected_spam.jsonl"))
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// downloadBackupHandler streams a database backup as an SQL file with gzip compression
// Files are always compressed and always have .gz extension to ensure consistency
func (s *Server) downloadBackupHandler(w http.ResponseWriter, r *http.Request) {
	if s.StorageEngine == nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "storage engine not available"})
		return
	}

	// set filename based on database type and timestamp
	dbType := "db"
	sqlEng, ok := s.StorageEngine.(*engine.SQL)
	if ok {
		dbType = string(sqlEng.Type())
	}
	timestamp := time.Now().Format("20060102-150405")

	// always use a .gz extension as the content is always compressed
	filename := fmt.Sprintf("tg-spam-backup-%s-%s.sql.gz", dbType, timestamp)

	// set headers for file download - note we're using application/octet-stream
	// instead of application/sql to prevent browsers from trying to interpret the file
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// create a gzip writer that streams to response
	gzipWriter := gzip.NewWriter(w)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			log.Printf("[ERROR] failed to close gzip writer: %v", err)
		}
	}()

	// stream backup directly to response through gzip
	if err := s.StorageEngine.Backup(r.Context(), gzipWriter); err != nil {
		log.Printf("[ERROR] failed to create backup: %v", err)
		// we've already started writing the response, so we can't send a proper error response
		return
	}

	// flush the gzip writer to ensure all data is written
	if err := gzipWriter.Flush(); err != nil {
		log.Printf("[ERROR] failed to flush gzip writer: %v", err)
	}
}

// downloadExportToPostgresHandler streams a PostgreSQL-compatible export from a SQLite database
func (s *Server) downloadExportToPostgresHandler(w http.ResponseWriter, r *http.Request) {
	if s.StorageEngine == nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": "storage engine not available"})
		return
	}

	// check if the database is SQLite
	if s.StorageEngine.Type() != engine.Sqlite {
		_ = rest.EncodeJSON(w, http.StatusBadRequest, rest.JSON{"error": "source database must be SQLite"})
		return
	}

	// set filename based on timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("tg-spam-sqlite-to-postgres-%s.sql.gz", timestamp)

	// set headers for file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// create a gzip writer that streams to response
	gzipWriter := gzip.NewWriter(w)
	defer func() {
		if err := gzipWriter.Close(); err != nil {
			log.Printf("[ERROR] failed to close gzip writer: %v", err)
		}
	}()

	// stream export directly to response through gzip
	if err := s.StorageEngine.BackupSqliteAsPostgres(r.Context(), gzipWriter); err != nil {
		log.Printf("[ERROR] failed to create export: %v", err)
		// we've already started writing the response, so we can't send a proper error response
		return
	}

	// flush the gzip writer to ensure all data is written
	if err := gzipWriter.Flush(); err != nil {
		log.Printf("[ERROR] failed to flush gzip writer: %v", err)
	}
}

// getSettingsHandler returns application settings, including the list of available Lua plugins
func (s *Server) getSettingsHandler(w http.ResponseWriter, _ *http.Request) {
	// get the list of available Lua plugins before returning settings
	s.Settings.LuaAvailablePlugins = s.Detector.GetLuaPluginNames()
	rest.RenderJSON(w, s.Settings)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

// GenerateRandomPassword generates a random password of a given length
func GenerateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	const charsetLen = int64(len(charset))

	result := make([]byte, length)
	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(charsetLen))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
