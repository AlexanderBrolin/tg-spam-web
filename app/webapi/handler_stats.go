package webapi

import (
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/tg-spam/app/storage/engine"
)

// getStatsHandler handles GET /api/v2/stats
func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := s.DetectedSpam.Read(r.Context())
	if err != nil {
		log.Printf("[ERROR] failed to read detected spam for stats: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	// apply gid filter
	gid := r.URL.Query().Get("gid")
	if gid != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.GID == gid {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
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

	approvedUsers := s.Detector.ApprovedUsers()

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
		"approved_users":   len(approvedUsers),
		"by_detector":      byDetector,
		"by_day":           byDay,
		"uptime_seconds":   int64(uptime.Seconds()),
		"version":          s.Version,
		"database_type":    dbType,
	})
}
