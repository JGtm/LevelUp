// Package duckdb — home_repo.go : accès DB pour la page d'accueil Mission Control.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// HomeRepo fournit les données de la page d'accueil depuis DuckDB.
type HomeRepo struct {
	pdb *PlayerDB
}

// NewHomeRepo crée un HomeRepo pour un joueur.
func NewHomeRepo(pdb *PlayerDB) *HomeRepo {
	return &HomeRepo{pdb: pdb}
}

// LoadHomeMatches charge les 200 derniers matchs du joueur (Q26).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]domain.HomeMatchRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q26HomeMatches, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMatchRow
	for rows.Next() {
		var row domain.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapName,
			&row.MapNameFR,
			&row.PairName,
			&row.PairNameFR,
			&row.PlaylistName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.TimePlayedSecs,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	var count int
	err := r.pdb.Player.QueryRow(ctx, Q26bCountPlayerMatches, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment (Q27).
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]domain.HomeSessionRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q27HomeSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeSessionRow
	for rows.Next() {
		var row domain.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.StartTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadRecentMedia charge les médias récents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister — dégradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// isTableNotFoundErr détecte les erreurs "table not found" DuckDB.
func isTableNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Table with name") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no such table")
}

// ---------------------------------------------------------------------------
// BattlePassCacheRepository implementation
// ---------------------------------------------------------------------------

// LoadCachedBattlePass retourne les données BP depuis battlepass_snapshots si une
// entrée récente du joueur existe dans la fenêtre ttl.
func (r *HomeRepo) LoadCachedBattlePass(ctx context.Context, ttl time.Duration) (*domain.BattlePassResponse, bool, error) {
	secs := int64(ttl.Seconds())
	query := fmt.Sprintf(`
		SELECT reward_track_path, current_rank, partial_progress
		FROM battlepass_snapshots
		WHERE xuid = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		ORDER BY is_active DESC, snapshot_at DESC
		LIMIT 1`, secs)

	var trackPath string
	var rank, progress int
	err := r.pdb.Player.QueryRow(ctx, query, r.pdb.XUID).Scan(&trackPath, &rank, &progress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache BP query: %w", err)
	}

	resp := &domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
		FromCache:   true,
	}
	return resp, true, nil
}

// challengeSnapshotRow est une ligne agrégée pour la reconstruction ChallengesResponse.
type challengeSnapshotRow struct {
	status    string
	xpReward  int
	expiresAt sql.NullTime
}

// LoadCachedChallenges retourne un résumé des snapshots récents depuis challenge_snapshots
// (la snapshot la plus récente par challenge_path dans la fenêtre ttl).
func (r *HomeRepo) LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error) {
	secs := int64(ttl.Seconds())
	// Sélectionne la snapshot la plus récente par challenge_path dans la fenêtre TTL.
	query := fmt.Sprintf(`
		SELECT status, xp_reward, expires_at
		FROM (
			SELECT status, xp_reward, expires_at,
			       ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
			FROM challenge_snapshots
			WHERE xuid = ?
			  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL '%d' SECOND
		) t
		WHERE rn = 1`, secs)

	rows, err := r.pdb.Player.Query(ctx, query, r.pdb.XUID)
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
		if err := rows.Scan(&s.status, &s.xpReward, &s.expiresAt); err != nil {
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
	return resp, true, nil
}
