package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/melth-cmd/jellyfin-share-backend/internal/models"
)

func (db *DB) CreateShare(ctx context.Context, share *models.Share) error {
	query := `
		INSERT INTO shares (
			id, public_token, jellyfin_item_id, jellyfin_user_id, title, overview,
			runtime_seconds, poster_path, backdrop_path, item_type, max_total_plays,
			max_concurrent_viewers, expires_at, password_hash, created_at,
			max_video_height, max_video_bitrate, allow_download
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)`

	_, err := db.ExecContext(ctx, query,
		share.ID, share.PublicToken, share.JellyfinItemID, share.JellyfinUserID,
		share.Title, share.Overview, share.RuntimeSeconds, share.PosterPath,
		share.BackdropPath, share.ItemType, share.MaxTotalPlays, share.MaxConcurrentViewers,
		share.ExpiresAt, share.PasswordHash, share.CreatedAt,
		share.MaxVideoHeight, share.MaxVideoBitrate, share.AllowDownload,
	)
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}
	return nil
}

func (db *DB) GetShareByToken(ctx context.Context, token string) (*models.Share, error) {
	var share models.Share
	query := `SELECT * FROM shares WHERE public_token = $1`

	err := db.GetContext(ctx, &share, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get share: %w", err)
	}
	return &share, nil
}

func (db *DB) GetShareByID(ctx context.Context, id uuid.UUID) (*models.Share, error) {
	var share models.Share
	query := `SELECT * FROM shares WHERE id = $1`

	err := db.GetContext(ctx, &share, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get share: %w", err)
	}
	return &share, nil
}

func (db *DB) GetSharesByUser(ctx context.Context, jellyfinUserID string) ([]models.Share, error) {
	var shares []models.Share
	query := `SELECT * FROM shares WHERE jellyfin_user_id = $1 ORDER BY created_at DESC`

	err := db.SelectContext(ctx, &shares, query, jellyfinUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}
	return shares, nil
}

// activeShareSQL is the one definition of a link that still works: not
// revoked, not past its expiry, and not out of plays.
const activeShareSQL = `revoked_at IS NULL
	AND (expires_at IS NULL OR expires_at > NOW())
	AND (max_total_plays IS NULL OR total_plays < max_total_plays)`

func statusFilterSQL(status string) string {
	switch status {
	case "active":
		return activeShareSQL
	case "inactive":
		return "NOT (" + activeShareSQL + ")"
	default:
		return ""
	}
}

// GetSharesPage returns up to limit links, newest first, starting below the
// cursor - or from the top when there is none. id breaks ties between links
// created in the same instant, so no link falls between two pages. status is
// active, inactive, or anything else for every link.
func (db *DB) GetSharesPage(ctx context.Context, limit int, after *models.PageCursor, status string) ([]models.Share, error) {
	var shares []models.Share
	where := []string{}
	args := []interface{}{}

	if filter := statusFilterSQL(status); filter != "" {
		where = append(where, filter)
	}
	if after != nil {
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)+1, len(args)+2))
		args = append(args, after.CreatedAt, after.ID)
	}

	query := "SELECT * FROM shares"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	if err := db.SelectContext(ctx, &shares, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}
	return shares, nil
}

// CountShares counts every link and those that can still be watched. The index
// shows the counts however many rows it has loaded so far.
func (db *DB) CountShares(ctx context.Context) (total, active int, err error) {
	query := `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE ` + activeShareSQL + `)
		FROM shares`
	if err := db.QueryRowContext(ctx, query).Scan(&total, &active); err != nil {
		return 0, 0, fmt.Errorf("failed to count shares: %w", err)
	}
	return total, active, nil
}

// RevokeShares revokes every id that is not already revoked and returns the
// ids it changed. Ids that do not exist or were already revoked are simply not
// in the result.
func (db *DB) RevokeShares(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	query := `UPDATE shares SET revoked_at = NOW()
		WHERE id = ANY($1) AND revoked_at IS NULL
		RETURNING id`
	revoked := []uuid.UUID{}
	if err := db.SelectContext(ctx, &revoked, query, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("failed to revoke shares: %w", err)
	}
	return revoked, nil
}

// DeleteInactiveShares permanently removes the ids that can no longer be
// watched. Active ids are left alone even when the caller asks for them.
func (db *DB) DeleteInactiveShares(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	query := `DELETE FROM shares
		WHERE id = ANY($1) AND NOT (` + activeShareSQL + `)
		RETURNING id`
	deleted := []uuid.UUID{}
	if err := db.SelectContext(ctx, &deleted, query, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("failed to delete shares: %w", err)
	}
	return deleted, nil
}

func (db *DB) RevokeShare(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE shares SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke share: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("share not found or already revoked")
	}
	return nil
}

func (db *DB) UpdateShare(ctx context.Context, id uuid.UUID, maxTotalPlays, maxConcurrentViewers *int, extendMinutes *int, passwordHash *string, removePassword bool) error {
	// Build dynamic update
	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if maxTotalPlays != nil {
		updates = append(updates, fmt.Sprintf("max_total_plays = $%d", argIdx))
		args = append(args, *maxTotalPlays)
		argIdx++
	}

	if maxConcurrentViewers != nil {
		updates = append(updates, fmt.Sprintf("max_concurrent_viewers = $%d", argIdx))
		args = append(args, *maxConcurrentViewers)
		argIdx++
	}

	if extendMinutes != nil {
		// A NULL expiry stays NULL: extending a share that never expires is a no-op
		// rather than silently giving it a deadline.
		updates = append(updates, fmt.Sprintf("expires_at = expires_at + INTERVAL '%d minutes'", *extendMinutes))
	}

	if passwordHash != nil {
		updates = append(updates, fmt.Sprintf("password_hash = $%d", argIdx))
		args = append(args, *passwordHash)
		argIdx++
	}

	if removePassword {
		updates = append(updates, "password_hash = NULL")
	}

	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE shares SET "
	for i, u := range updates {
		if i > 0 {
			query += ", "
		}
		query += u
	}
	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update share: %w", err)
	}
	return nil
}

// IncrementTotalPlays charges a play without taking a concurrent-viewer slot,
// for a download: there is no session that would later give the slot back.
func (db *DB) IncrementTotalPlays(ctx context.Context, shareID uuid.UUID) error {
	query := `
		UPDATE shares
		SET total_plays = total_plays + 1,
		    last_activity_at = NOW()
		WHERE id = $1`

	if _, err := db.ExecContext(ctx, query, shareID); err != nil {
		return fmt.Errorf("failed to increment total plays: %w", err)
	}
	return nil
}

func (db *DB) IncrementPlayCount(ctx context.Context, shareID uuid.UUID) error {
	query := `
		UPDATE shares
		SET total_plays = total_plays + 1,
		    current_concurrent_viewers = current_concurrent_viewers + 1,
		    last_activity_at = NOW()
		WHERE id = $1`

	_, err := db.ExecContext(ctx, query, shareID)
	if err != nil {
		return fmt.Errorf("failed to increment play count: %w", err)
	}
	return nil
}

func (db *DB) DecrementConcurrentViewers(ctx context.Context, shareID uuid.UUID) error {
	query := `
		UPDATE shares
		SET current_concurrent_viewers = GREATEST(current_concurrent_viewers - 1, 0)
		WHERE id = $1`

	_, err := db.ExecContext(ctx, query, shareID)
	if err != nil {
		return fmt.Errorf("failed to decrement concurrent viewers: %w", err)
	}
	return nil
}

func (db *DB) UpdateLastActivity(ctx context.Context, shareID uuid.UUID) error {
	query := `UPDATE shares SET last_activity_at = NOW() WHERE id = $1`
	_, err := db.ExecContext(ctx, query, shareID)
	return err
}

func (db *DB) GetActiveSharesCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM shares WHERE ` + activeShareSQL
	err := db.GetContext(ctx, &count, query)
	return count, err
}

func (db *DB) CleanupExpiredShares(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM shares WHERE expires_at IS NOT NULL AND expires_at < $1 AND revoked_at IS NOT NULL`
	result, err := db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
