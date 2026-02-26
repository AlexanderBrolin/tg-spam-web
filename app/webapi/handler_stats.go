package webapi

import (
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage"
	"github.com/umputun/tg-spam/app/storage/engine"
)

// getStatsHandler handles GET /api/v2/stats
func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	gid := r.URL.Query().Get("gid")

	var entries []storage.DetectedSpamInfo
	var err error
	if gid != "" {
		entries, err = s.DetectedSpam.ReadByGID(r.Context(), gid)
	} else {
		entries, err = s.DetectedSpam.Read(r.Context())
	}
	if err != nil {
		log.Printf("[ERROR] failed to read detected spam for stats: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := today.AddDate(0, 0, -7)

	var totalSpam, todaySpam, weekSpam int
	var addedToSamples int
	byDetector := map[string]int{}
	byDay := map[string]int{}

	for _, entry := range entries {
		totalSpam++
		if entry.Added {
			addedToSamples++
		}
		if entry.Timestamp.After(today) {
			todaySpam++
		}
		if entry.Timestamp.After(weekAgo) {
			weekSpam++
			day := entry.Timestamp.Format("2006-01-02")
			byDay[day]++
		}
		for _, check := range entry.Checks {
			if check.Spam {
				byDetector[check.Name]++
			}
		}
	}

	// get approved users count from per-channel store when gid is provided, otherwise from global detector
	var approvedUsersCount int
	if gid != "" && s.ApprovedUsersStore != nil {
		if users, auErr := s.ApprovedUsersStore.ReadByGID(r.Context(), gid); auErr == nil {
			approvedUsersCount = len(users)
		}
	} else {
		approvedUsersCount = len(s.Detector.ApprovedUsers())
	}

	// database info
	var dbType string
	if s.StorageEngine != nil {
		if sqlEng, ok := s.StorageEngine.(*engine.SQL); ok {
			dbType = string(sqlEng.Type())
		}
	}

	uptime := time.Since(startTime)

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"total_spam":       totalSpam,
		"today_spam":       todaySpam,
		"week_spam":        weekSpam,
		"added_to_samples": addedToSamples,
		"approved_users":   approvedUsersCount,
		"by_detector":      byDetector,
		"by_day":           byDay,
		"uptime_seconds":   int64(uptime.Seconds()),
		"version":          s.Version,
		"database_type":    dbType,
	})
}
