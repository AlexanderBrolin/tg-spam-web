package webapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/app/storage/engine"
	"github.com/umputun/tg-spam/app/webapi/mocks"
	"github.com/umputun/tg-spam/lib/spamcheck"
)

func TestServer_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(Config{ListenAddr: ":9876", Version: "dev", Detector: &mocks.DetectorMock{},
		SpamFilter: &mocks.SpamFilterMock{}})
	done := make(chan struct{})
	go func() {
		err := srv.Run(ctx)
		assert.NoError(t, err)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:9876/ping")
	require.NoError(t, err)
	t.Log(resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	assert.Contains(t, resp.Header.Get("App-Name"), "tg-spam")
	assert.Contains(t, resp.Header.Get("App-Version"), "dev")

	cancel()
	<-done
}

func TestServer_GenerateRandomPassword(t *testing.T) {
	res1, err := GenerateRandomPassword(32)
	require.NoError(t, err)
	t.Log(res1)
	assert.Len(t, res1, 32)

	res2, err := GenerateRandomPassword(32)
	require.NoError(t, err)
	t.Log(res2)
	assert.Len(t, res2, 32)

	assert.NotEqual(t, res1, res2)
}

func TestServer_getSettingsHandler(t *testing.T) {
	t.Run("with lua plugins", func(t *testing.T) {
		detectorMock := &mocks.DetectorMock{
			GetLuaPluginNamesFunc: func() []string {
				return []string{"plugin1", "plugin2", "plugin3"}
			},
		}

		settings := Settings{
			InstanceID:        "test",
			LuaPluginsEnabled: true,
			LuaPluginsDir:     "/path/to/plugins",
			LuaEnabledPlugins: []string{"plugin1", "plugin2"},
		}

		server := NewServer(Config{Version: "1.0", Detector: detectorMock, Settings: settings})
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/settings", http.NoBody)
		require.NoError(t, err)

		handler := http.HandlerFunc(server.getSettingsHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))

		var respSettings Settings
		err = json.Unmarshal(rr.Body.Bytes(), &respSettings)
		require.NoError(t, err)
		assert.Equal(t, settings.InstanceID, respSettings.InstanceID)
		assert.Equal(t, settings.LuaPluginsEnabled, respSettings.LuaPluginsEnabled)
		assert.Equal(t, settings.LuaPluginsDir, respSettings.LuaPluginsDir)
		assert.Equal(t, settings.LuaEnabledPlugins, respSettings.LuaEnabledPlugins)
		assert.Equal(t, []string{"plugin1", "plugin2", "plugin3"}, respSettings.LuaAvailablePlugins)
		assert.Len(t, detectorMock.GetLuaPluginNamesCalls(), 1)
	})

	t.Run("with lua plugins disabled", func(t *testing.T) {
		detectorMock := &mocks.DetectorMock{
			GetLuaPluginNamesFunc: func() []string {
				return []string{}
			},
		}

		settings := Settings{
			InstanceID:        "test",
			LuaPluginsEnabled: false,
		}

		server := NewServer(Config{Version: "1.0", Detector: detectorMock, Settings: settings})
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/settings", http.NoBody)
		require.NoError(t, err)

		handler := http.HandlerFunc(server.getSettingsHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var respSettings Settings
		err = json.Unmarshal(rr.Body.Bytes(), &respSettings)
		require.NoError(t, err)
		assert.Equal(t, settings.InstanceID, respSettings.InstanceID)
		assert.Equal(t, settings.LuaPluginsEnabled, respSettings.LuaPluginsEnabled)
		assert.Empty(t, respSettings.LuaAvailablePlugins)
		assert.Len(t, detectorMock.GetLuaPluginNamesCalls(), 1)
	})
}

func TestServer_getDynamicSamplesHandler(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		mockSpamFilter := &mocks.SpamFilterMock{
			DynamicSamplesFunc: func() ([]string, []string, error) {
				return []string{"spam1", "spam2"}, []string{"ham1", "ham2"}, nil
			},
		}

		server := NewServer(Config{SpamFilter: mockSpamFilter})
		req, err := http.NewRequest("GET", "/samples", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.getDynamicSamplesHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))

		var response struct {
			Spam []string `json:"spam"`
			Ham  []string `json:"ham"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, []string{"spam1", "spam2"}, response.Spam)
		assert.Equal(t, []string{"ham1", "ham2"}, response.Ham)
	})

	t.Run("error response", func(t *testing.T) {
		mockSpamFilter := &mocks.SpamFilterMock{
			DynamicSamplesFunc: func() ([]string, []string, error) {
				return nil, nil, errors.New("test error")
			},
		}

		server := NewServer(Config{SpamFilter: mockSpamFilter})
		req, err := http.NewRequest("GET", "/samples", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.getDynamicSamplesHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))

		var response struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "can't get dynamic samples", response.Error)
		assert.Equal(t, "test error", response.Details)
	})
}

func Test_downloadSampleHandler(t *testing.T) {
	mockSpamFilter := &mocks.SpamFilterMock{
		DynamicSamplesFunc: func() ([]string, []string, error) {
			return []string{"spam1", "spam2"}, []string{"ham1", "ham2"}, nil
		},
	}

	server := NewServer(Config{
		SpamFilter: mockSpamFilter,
	})

	t.Run("successful spam response", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/download/spam", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := server.downloadSampleHandler(func(spam, ham []string) ([]string, string) {
			return spam, "spam.txt"
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment; filename=\"spam.txt\"")
	})

	t.Run("successful ham response", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/download/ham", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := server.downloadSampleHandler(func(spam, ham []string) ([]string, string) {
			return spam, "ham.txt"
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment; filename=\"ham.txt\"")
	})

	t.Run("error handling", func(t *testing.T) {
		mockSpamFilter.DynamicSamplesFunc = func() ([]string, []string, error) {
			return nil, nil, errors.New("test error")
		}

		req, err := http.NewRequest("GET", "/download/ham", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := server.downloadSampleHandler(func(spam, ham []string) ([]string, string) {
			return spam, "ham.txt"
		})

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var response struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "can't get dynamic samples", response.Error)
		assert.Equal(t, "test error", response.Details)
	})
}

func TestServer_reloadDynamicSamplesHandler(t *testing.T) {
	mockSpamFilter := &mocks.SpamFilterMock{
		ReloadSamplesFunc: func() error {
			return nil // simulate successful reload
		},
	}

	server := NewServer(Config{
		SpamFilter: mockSpamFilter,
	})

	t.Run("successful reload", func(t *testing.T) {
		req, err := http.NewRequest("PUT", "/samples", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.reloadDynamicSamplesHandler)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response struct {
			Reloaded bool `json:"reloaded"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.Reloaded)
	})

	t.Run("error during reload", func(t *testing.T) {
		mockSpamFilter.ReloadSamplesFunc = func() error {
			return errors.New("test error") // simulate error during reload
		}

		req, err := http.NewRequest("PUT", "/samples", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.reloadDynamicSamplesHandler)

		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var response struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "can't reload samples", response.Error)
		assert.Equal(t, "test error", response.Details)
	})
}

// TestServer_formatDuration tests the formatDuration function in webapi.go
func TestServer_formatDuration(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"Minutes only", 5 * time.Minute, "5m"},
		{"Hours and minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"Days, hours, minutes", 4*24*time.Hour + 2*time.Hour + 5*time.Minute, "4d 2h 5m"},
		{"Zero", 0, "0m"},
		{"Just seconds", 30 * time.Second, "0m"},
		{"Large duration", 100*24*time.Hour + 12*time.Hour + 45*time.Minute, "100d 12h 45m"},
		{"Exactly one day", 24 * time.Hour, "1d 0h 0m"},
		{"Exactly one hour", 1 * time.Hour, "1h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := formatDuration(tt.dur)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestServer_downloadDetectedSpamHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 25, 10, 0, 0, 0, time.UTC)

	t.Run("successful download", func(t *testing.T) {
		ds := &mocks.DetectedSpamMock{
			ReadFunc: func(ctx context.Context) ([]storage.DetectedSpamInfo, error) {
				return []storage.DetectedSpamInfo{
					{
						ID:        123,
						GID:       "gid123",
						Text:      "spam example",
						UserID:    123,
						UserName:  "user",
						Checks:    []spamcheck.Response{{Spam: true, Name: "test", Details: "details"}},
						Timestamp: testTime,
					},
				}, nil
			},
		}

		server := NewServer(Config{DetectedSpam: ds})
		req, err := http.NewRequest("GET", "/download/detected_spam", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.downloadDetectedSpamHandler)
		handler.ServeHTTP(rr, req)

		t.Run("verify headers", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "application/x-jsonlines", rr.Header().Get("Content-Type"))
			assert.Contains(t, rr.Header().Get("Content-Disposition"), "detected_spam.jsonl")
		})

		t.Run("verify content", func(t *testing.T) {
			var info struct {
				ID        int64                `json:"id"`
				GID       string               `json:"gid"`
				Text      string               `json:"text"`
				UserID    int64                `json:"user_id"`
				UserName  string               `json:"user_name"`
				Timestamp time.Time            `json:"timestamp"`
				Added     bool                 `json:"added"`
				Checks    []spamcheck.Response `json:"checks"`
			}
			err = json.Unmarshal([]byte(strings.TrimSpace(rr.Body.String())), &info)
			require.NoError(t, err)
			assert.Equal(t, int64(123), info.ID)
			assert.Equal(t, "gid123", info.GID)
			assert.Equal(t, "spam example", info.Text)
			assert.Equal(t, int64(123), info.UserID)
			assert.Equal(t, "user", info.UserName)
			assert.Equal(t, testTime, info.Timestamp)
			require.Len(t, info.Checks, 1)
			assert.Equal(t, "test", info.Checks[0].Name)
			assert.Equal(t, "details", info.Checks[0].Details)
			assert.True(t, info.Checks[0].Spam)
		})
	})

	t.Run("multiple entries", func(t *testing.T) {
		ds := &mocks.DetectedSpamMock{
			ReadFunc: func(ctx context.Context) ([]storage.DetectedSpamInfo, error) {
				return []storage.DetectedSpamInfo{
					{ID: 1, Text: "first"},
					{ID: 2, Text: "second"},
				}, nil
			},
		}

		server := NewServer(Config{DetectedSpam: ds})
		req, err := http.NewRequest("GET", "/download/detected_spam", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.downloadDetectedSpamHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		lines := strings.Split(strings.TrimSpace(rr.Body.String()), "\n")
		assert.Len(t, lines, 2)

		for i, line := range lines {
			var info struct {
				ID    int64  `json:"id"`
				Text  string `json:"text"`
				Added bool   `json:"added"`
			}
			err = json.Unmarshal([]byte(line), &info)
			require.NoError(t, err)
			assert.Equal(t, int64(i+1), info.ID)
			assert.Equal(t, []string{"first", "second"}[i], info.Text)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		ds := &mocks.DetectedSpamMock{
			ReadFunc: func(ctx context.Context) ([]storage.DetectedSpamInfo, error) {
				return nil, errors.New("test error")
			},
		}

		server := NewServer(Config{DetectedSpam: ds})
		req, err := http.NewRequest("GET", "/download/detected_spam", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.downloadDetectedSpamHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))

		var resp struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "can't get detected spam", resp.Error)
		assert.Equal(t, "test error", resp.Details)
	})
}

func TestServer_downloadBackupHandler(t *testing.T) {
	t.Run("successful backup with gzip", func(t *testing.T) {
		mockStorageEngine := &mocks.StorageEngineMock{
			BackupFunc: func(_ context.Context, w io.Writer) error {
				_, err := w.Write([]byte("-- SQL backup test content"))
				return err
			},
		}

		srv := NewServer(Config{
			StorageEngine: mockStorageEngine,
		})

		req := httptest.NewRequest("GET", "/download/backup", http.NoBody)
		w := httptest.NewRecorder()
		srv.downloadBackupHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		// check headers
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"), "content type should be binary")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment; filename=")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), ".sql.gz")

		// read the content
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// verify it's actually gzipped data by trying to decompress it
		gzipReader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err, "Content should be properly gzipped")
		defer gzipReader.Close()

		decompressedContent, err := io.ReadAll(gzipReader)
		require.NoError(t, err)

		assert.Contains(t, string(decompressedContent), "-- SQL backup test content")
	})

	t.Run("nil storage engine", func(t *testing.T) {
		srv := NewServer(Config{
			StorageEngine: nil,
		})

		req := httptest.NewRequest("GET", "/download/backup", http.NoBody)
		w := httptest.NewRecorder()
		srv.downloadBackupHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "storage engine not available")
	})
}

func TestServer_downloadExportToPostgresHandler(t *testing.T) {
	t.Run("successful export with sqlite engine", func(t *testing.T) {
		mockStorage := &mocks.StorageEngineMock{
			TypeFunc: func() engine.Type {
				return engine.Sqlite // return the string representation of Sqlite type
			},
			BackupSqliteAsPostgresFunc: func(_ context.Context, w io.Writer) error {
				_, err := w.Write([]byte("-- SQLite to PostgreSQL export test content"))
				return err
			},
		}

		srv := NewServer(Config{
			StorageEngine: mockStorage,
		})

		req := httptest.NewRequest("GET", "/download/export-to-postgres", http.NoBody)
		w := httptest.NewRecorder()

		srv.downloadExportToPostgresHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		// check headers
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"), "content type should be binary")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment; filename=")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "tg-spam-sqlite-to-postgres")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), ".sql.gz")

		// read the content
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// verify it's actually gzipped data by trying to decompress it
		gzipReader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err, "Content should be properly gzipped")
		defer gzipReader.Close()

		decompressedContent, err := io.ReadAll(gzipReader)
		require.NoError(t, err)

		assert.Contains(t, string(decompressedContent), "-- SQLite to PostgreSQL export test content")
	})

	t.Run("non-sqlite engine", func(t *testing.T) {
		mockStorage := &mocks.StorageEngineMock{
			TypeFunc: func() engine.Type {
				return engine.Postgres // return the string representation of Postgres type
			},
		}

		srv := NewServer(Config{
			StorageEngine: mockStorage,
		})

		req := httptest.NewRequest("GET", "/download/export-to-postgres", http.NoBody)
		w := httptest.NewRecorder()
		srv.downloadExportToPostgresHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, string(body), "source database must be SQLite")
	})

	t.Run("nil storage engine", func(t *testing.T) {
		srv := NewServer(Config{
			StorageEngine: nil,
		})

		req := httptest.NewRequest("GET", "/download/export-to-postgres", http.NoBody)
		w := httptest.NewRecorder()
		srv.downloadExportToPostgresHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "storage engine not available")
	})
}
