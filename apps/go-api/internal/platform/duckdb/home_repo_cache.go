// Package duckdb — home_repo_cache.go : lecture des caches BattlePass et
// Challenges depuis les tables snapshots (BattlePassCacheRepository
// implementation).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// BattlePassCacheRepository implementation
// ---------------------------------------------------------------------------

// LoadCachedBattlePass retourne les données BP depuis battlepass_snapshots si une
// entrée récente du joueur existe dans la fenêtre ttl.
func (r *HomeRepo) LoadCachedBattlePass(ctx context.Context, ttl time.Duration) (*domain.BattlePassResponse, bool, error) {
	secs := int64(ttl.Seconds())
	query := fmt.Sprintf(`
		SELECT reward_track_path, current_rank, partial_progress, snapshot_at
		FROM battlepass_snapshots
		WHERE xuid = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		ORDER BY is_active DESC, snapshot_at DESC
		LIMIT 1`, secs)

	var trackPath string
	var rank, progress int
	var snapshotAt time.Time
	err := r.pdb.ReadDB().QueryRow(ctx, query, r.pdb.XUID).Scan(&trackPath, &rank, &progress, &snapshotAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache BP query: %w", err)
	}

	snapshotAtRFC := snapshotAt.UTC().Format(time.RFC3339)
	resp := &domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
		FromCache:   true,
		SnapshotAt:  &snapshotAtRFC,
	}
	return resp, true, nil
}

// challengeSnapshotRow est une ligne agrégée pour la reconstruction ChallengesResponse.
type challengeSnapshotRow struct {
	status     string
	xpReward   int
	expiresAt  sql.NullTime
	snapshotAt time.Time
}

// LoadCachedChallenges retourne un résumé des snapshots récents depuis challenge_snapshots
// (la snapshot la plus récente par challenge_path dans la fenêtre ttl).
func (r *HomeRepo) LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error) {
	secs := int64(ttl.Seconds())
	// Sélectionne la snapshot la plus récente par challenge_path dans la fenêtre TTL.
	query := fmt.Sprintf(`
		SELECT status, xp_reward, expires_at, snapshot_at
		FROM (
			SELECT status, xp_reward, expires_at, snapshot_at,
			       ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
			FROM challenge_snapshots
			WHERE xuid = ?
			  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		) t
		WHERE rn = 1`, secs)

	rows, err := r.pdb.ReadDB().Query(ctx, query, r.pdb.XUID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache challenges query: %w", err)
	}
	defer rows.Close()

	var snapshots []challengeSnapshotRow
	for rows.Next() {
		var s challengeSnapshotRow
		if err := rows.Scan(&s.status, &s.xpReward, &s.expiresAt, &s.snapshotAt); err != nil {
			return nil, false, fmt.Errorf("home_repo: cache challenges scan: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(snapshots) == 0 {
		return nil, false, nil
	}

	total := len(snapshots)
	completed := 0
	xpAvailable := 0
	var earliestExpiry *time.Time
	var latestSnapshot time.Time

	for _, s := range snapshots {
		if strings.EqualFold(s.status, "Completed") {
			completed++
		} else {
			xpAvailable += s.xpReward
		}
		if s.expiresAt.Valid {
			t := s.expiresAt.Time
			if earliestExpiry == nil || t.Before(*earliestExpiry) {
				earliestExpiry = &t
			}
		}
		if s.snapshotAt.After(latestSnapshot) {
			latestSnapshot = s.snapshotAt
		}
	}

	resp := &domain.ChallengesResponse{
		Available: true,
		Total:     &total,
		Completed: &completed,
		FromCache: true,
	}
	if xpAvailable > 0 {
		resp.XPAvailable = &xpAvailable
	}
	if earliestExpiry != nil {
		s := earliestExpiry.Format(time.RFC3339)
		resp.NextExpiry = &s
	}
	if !latestSnapshot.IsZero() {
		s := latestSnapshot.UTC().Format(time.RFC3339)
		resp.SnapshotAt = &s
	}
	return resp, true, nil
}
