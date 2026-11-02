package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type AuditEventType string

const (
	AuditEventShareCreated    AuditEventType = "share_created"
	AuditEventShareRevoked    AuditEventType = "share_revoked"
	AuditEventShareUpdated    AuditEventType = "share_updated"
	AuditEventShareDeleted    AuditEventType = "share_deleted"
	AuditEventShareRestored   AuditEventType = "share_restored"
	AuditEventShareDuplicated AuditEventType = "share_duplicated"
	AuditEventShareAccessed   AuditEventType = "share_accessed"
	AuditEventPasswordAttempt AuditEventType = "password_attempt"
	AuditEventPlaybackStarted AuditEventType = "playback_started"
	AuditEventPlaybackEnded   AuditEventType = "playback_ended"
	AuditEventPlaybackDenied  AuditEventType = "playback_denied"
	AuditEventSessionTimeout  AuditEventType = "session_timeout"
	AuditEventDownloadStarted AuditEventType = "download_started"
)

type AuditLog struct {
	ID             uuid.UUID              `db:"id" json:"id"`
	EventType      string                 `db:"event_type" json:"eventType"`
	ShareID        *uuid.UUID             `db:"share_id" json:"shareId,omitempty"`
	SessionID      *uuid.UUID             `db:"session_id" json:"sessionId,omitempty"`
	JellyfinUserID *string                `db:"jellyfin_user_id" json:"jellyfinUserId,omitempty"`
	ClientIPHash   *string                `db:"client_ip_hash" json:"clientIpHash,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
	CreatedAt      string                 `db:"created_at" json:"createdAt"`
}

func (db *DB) LogAuditEvent(ctx context.Context, eventType AuditEventType, shareID, sessionID *uuid.UUID, jellyfinUserID, clientIPHash *string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (event_type, share_id, session_id, jellyfin_user_id, client_ip_hash, details)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = db.ExecContext(ctx, query, string(eventType), shareID, sessionID, jellyfinUserID, clientIPHash, detailsJSON)
	if err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}
	return nil
}

func (db *DB) GetAuditLogs(ctx context.Context, shareID *uuid.UUID, limit, offset int) ([]AuditLog, error) {
	var logs []AuditLog
	var query string
	var args []interface{}

	if shareID != nil {
		query = `SELECT id, event_type, share_id, session_id, jellyfin_user_id, client_ip_hash, created_at
		         FROM audit_logs WHERE share_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{shareID, limit, offset}
	} else {
		query = `SELECT id, event_type, share_id, session_id, jellyfin_user_id, client_ip_hash, created_at
		         FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	err := db.SelectContext(ctx, &logs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	return logs, nil
}

// ShareAnalytics holds analytics data for a share
type ShareAnalytics struct {
	TotalViews          int          `json:"totalViews"`
	UniqueViewers       int          `json:"uniqueViewers"`
	AvgWatchTimeSeconds int          `json:"avgWatchTimeSeconds"`
	ViewsByDay          []DailyViews `json:"viewsByDay"`
}

// DailyViews holds views per day
type DailyViews struct {
	Date  string `json:"date"`
	Views int    `json:"views"`
}

// ShareAnalyticsRow is one link's usage in the analytics overview.
type ShareAnalyticsRow struct {
	ID                  uuid.UUID `db:"id" json:"id"`
	Title               string    `db:"title" json:"title"`
	TotalPlays          int       `db:"total_plays" json:"totalPlays"`
	Views               int       `db:"views" json:"views"`
	UniqueViewers       int       `db:"unique_viewers" json:"uniqueViewers"`
	AvgWatchTimeSeconds int       `db:"avg_watch_time_seconds" json:"avgWatchTimeSeconds"`
}

// AnalyticsOverview is the aggregate shown on the admin Analytics tab.
type AnalyticsOverview struct {
	TotalShares         int                 `json:"totalShares"`
	ActiveShares        int                 `json:"activeShares"`
	TotalPlays          int                 `json:"totalPlays"`
	TotalViews          int                 `json:"totalViews"`
	UniqueViewers       int                 `json:"uniqueViewers"`
	AvgWatchTimeSeconds int                 `json:"avgWatchTimeSeconds"`
	ViewsByDay          []DailyViews        `json:"viewsByDay"`
	Shares              []ShareAnalyticsRow `json:"shares"`
}

// GetAnalyticsOverview aggregates usage across every share. It is one set of
// queries rather than one request per link, so the Analytics tab stays cheap.
func (db *DB) GetAnalyticsOverview(ctx context.Context) (*AnalyticsOverview, error) {
	overview := &AnalyticsOverview{
		ViewsByDay: []DailyViews{},
		Shares:     []ShareAnalyticsRow{},
	}

	total, active, err := db.CountShares(ctx)
	if err != nil {
		return nil, err
	}
	overview.TotalShares = total
	overview.ActiveShares = active

	if err := db.GetContext(ctx, &overview.TotalPlays, `SELECT COALESCE(SUM(total_plays), 0) FROM shares`); err != nil {
		return nil, fmt.Errorf("failed to sum plays: %w", err)
	}
	if err := db.GetContext(ctx, &overview.TotalViews, `
		SELECT COUNT(*) FROM audit_logs WHERE event_type = 'playback_started'
	`); err != nil {
		return nil, fmt.Errorf("failed to count views: %w", err)
	}
	if err := db.GetContext(ctx, &overview.UniqueViewers, `
		SELECT COUNT(DISTINCT client_ip_hash) FROM audit_logs
		WHERE event_type = 'playback_started' AND client_ip_hash IS NOT NULL
	`); err != nil {
		return nil, fmt.Errorf("failed to count unique viewers: %w", err)
	}
	if err := db.GetContext(ctx, &overview.AvgWatchTimeSeconds, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(finished_at, NOW()) - started_at))), 0)::int
		FROM share_sessions
	`); err != nil {
		return nil, fmt.Errorf("failed to average watch time: %w", err)
	}

	type dailyRow struct {
		Date  string `db:"date"`
		Views int    `db:"views"`
	}
	var dailyRows []dailyRow
	if err := db.SelectContext(ctx, &dailyRows, `
		SELECT DATE(created_at) AS date, COUNT(*) AS views
		FROM audit_logs
		WHERE event_type = 'playback_started'
			AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`); err != nil {
		return nil, fmt.Errorf("failed to get daily views: %w", err)
	}
	for _, row := range dailyRows {
		overview.ViewsByDay = append(overview.ViewsByDay, DailyViews{Date: row.Date, Views: row.Views})
	}

	if err := db.SelectContext(ctx, &overview.Shares, `
		SELECT s.id, s.title, s.total_plays,
		       COALESCE(v.views, 0) AS views,
		       COALESCE(v.unique_viewers, 0) AS unique_viewers,
		       COALESCE(w.avg_watch_time_seconds, 0) AS avg_watch_time_seconds
		FROM shares s
		LEFT JOIN (
			SELECT share_id, COUNT(*) AS views,
			       COUNT(DISTINCT client_ip_hash) AS unique_viewers
			FROM audit_logs
			WHERE event_type = 'playback_started'
			GROUP BY share_id
		) v ON v.share_id = s.id
		LEFT JOIN (
			SELECT share_id,
			       AVG(EXTRACT(EPOCH FROM (COALESCE(finished_at, NOW()) - started_at)))::int AS avg_watch_time_seconds
			FROM share_sessions
			GROUP BY share_id
		) w ON w.share_id = s.id
		ORDER BY s.created_at DESC
	`); err != nil {
		return nil, fmt.Errorf("failed to get per-share analytics: %w", err)
	}

	return overview, nil
}

// GetShareAnalytics retrieves analytics for a specific share
func (db *DB) GetShareAnalytics(ctx context.Context, shareID uuid.UUID) (*ShareAnalytics, error) {
	analytics := &ShareAnalytics{
		ViewsByDay: []DailyViews{},
	}

	// Get total views (playback_started events)
	var totalViews int
	err := db.GetContext(ctx, &totalViews, `
		SELECT COUNT(*) FROM audit_logs
		WHERE share_id = $1 AND event_type = 'playback_started'
	`, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total views: %w", err)
	}
	analytics.TotalViews = totalViews

	// Get unique viewers (distinct IP hashes)
	var uniqueViewers int
	err = db.GetContext(ctx, &uniqueViewers, `
		SELECT COUNT(DISTINCT client_ip_hash) FROM audit_logs
		WHERE share_id = $1 AND event_type = 'playback_started' AND client_ip_hash IS NOT NULL
	`, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique viewers: %w", err)
	}
	analytics.UniqueViewers = uniqueViewers

	// Get average watch time from sessions
	var avgWatchTime float64
	err = db.GetContext(ctx, &avgWatchTime, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(finished_at, NOW()) - started_at))), 0)
		FROM share_sessions WHERE share_id = $1
	`, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg watch time: %w", err)
	}
	analytics.AvgWatchTimeSeconds = int(avgWatchTime)

	// Get views by day (last 30 days)
	type dailyRow struct {
		Date  string `db:"date"`
		Views int    `db:"views"`
	}
	var dailyRows []dailyRow
	err = db.SelectContext(ctx, &dailyRows, `
		SELECT DATE(created_at) as date, COUNT(*) as views
		FROM audit_logs
		WHERE share_id = $1
			AND event_type = 'playback_started'
			AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, shareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily views: %w", err)
	}

	for _, row := range dailyRows {
		analytics.ViewsByDay = append(analytics.ViewsByDay, DailyViews{
			Date:  row.Date,
			Views: row.Views,
		})
	}

	return analytics, nil
}
