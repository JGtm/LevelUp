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

	"levelup/go-api/internal/ctxkeys"
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
		  AND snapshot_at > CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) - INTERVAL '%d' SECOND
		ORDER BY is_active DESC, snapshot_at DESC
		LIMIT 1`, secs)

	var trackPath string
	var rank, progress int
	var snapshotAt time.Time
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, query, r.pdb.XUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if isTableNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("home_repo: cache BP query: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(&trackPath, &rank, &progress, &snapshotAt); err != nil {
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
// Les champs title/description/image_url (présents sur les défis ACTIFS rendus) servent
// à reconstruire de vraies cartes hors-ligne quand le live est indisponible.
type challengeSnapshotRow struct {
	challengePath   string
	displayPath     sql.NullString // vrai chemin GameCMS (cadence) ; fallback sur challengePath
	status          string
	xpReward        int
	progressCurrent sql.NullInt64
	progressTarget  sql.NullInt64
	expiresAt       sql.NullTime
	snapshotAt      time.Time
	title           sql.NullString
	description     sql.NullString
	imageURL        sql.NullString
	locale          sql.NullString // langue des libellés résolus ; '' ou NULL = ligne legacy pré-migration
}

// LoadCachedChallenges retourne un résumé des snapshots récents depuis challenge_snapshots
// (la snapshot la plus récente par challenge_path dans la fenêtre ttl).
func (r *HomeRepo) LoadCachedChallenges(ctx context.Context, ttl time.Duration) (*domain.ChallengesResponse, bool, error) {
	snapshots, err := r.queryRecentChallengeSnapshots(ctx, ttl)
	if err != nil {
		return nil, false, err
	}
	if len(snapshots) == 0 {
		return nil, false, nil
	}
	resp := buildChallengesResponseFromSnapshots(snapshots)
	return resp, true, nil
}

// queryRecentChallengeSnapshots charge la dernière snapshot de chaque challenge_path
// dans la fenêtre TTL. Retourne nil sans erreur si la table est absente.
func (r *HomeRepo) queryRecentChallengeSnapshots(ctx context.Context, ttl time.Duration) ([]challengeSnapshotRow, error) {
	secs := int64(ttl.Seconds())
	// Locale de la requête (threadée par le middleware via ctxkeys). On ne sert que les
	// snapshots de CETTE langue, plus les lignes legacy (locale '' ou NULL, pré-migration
	// add_challenge_snapshots_locale) tolérées pour ne pas masquer un cache pré-existant.
	// PARTITION BY challenge_path : la ligne locale-spécifique (plus récente) l'emporte
	// naturellement sur une éventuelle legacy plus ancienne du même défi.
	locale := ctxkeys.Locale(ctx)
	query := fmt.Sprintf(`
		SELECT challenge_path, display_path, status, xp_reward, progress_current, progress_target,
		       expires_at, snapshot_at, title, description, image_url, locale
		FROM (
			SELECT challenge_path, display_path, status, xp_reward, progress_current, progress_target,
			       expires_at, snapshot_at, title, description, image_url, locale,
			       ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
			FROM challenge_snapshots
			WHERE xuid = ?
			  AND (locale = ? OR locale = '' OR locale IS NULL)
			  AND snapshot_at > CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) - INTERVAL '%d' SECOND
		) t
		WHERE rn = 1`, secs)

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, query, r.pdb.XUID, locale)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("home_repo: cache challenges query: %w", err)
	}
	defer rows.Close()

	var snapshots []challengeSnapshotRow
	for rows.Next() {
		var s challengeSnapshotRow
		if err := rows.Scan(&s.challengePath, &s.displayPath, &s.status, &s.xpReward, &s.progressCurrent,
			&s.progressTarget, &s.expiresAt, &s.snapshotAt, &s.title, &s.description, &s.imageURL, &s.locale); err != nil {
			return nil, fmt.Errorf("home_repo: cache challenges scan: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snapshots, nil
}

// buildChallengesResponseFromSnapshots agrège un slice de snapshots en ChallengesResponse.
// Reconstruit les Items (cartes) à partir des défis ACTIFS porteurs d'un titre rendu —
// ce qui rend le cache « renderable » et évite « Défis indisponibles » quand le live tombe.
func buildChallengesResponseFromSnapshots(snapshots []challengeSnapshotRow) *domain.ChallengesResponse {
	total := len(snapshots)
	completed := 0
	xpAvailable := 0
	var earliestExpiry *time.Time
	var latestSnapshot time.Time
	items := make([]domain.ChallengeItem, 0, len(snapshots))

	for _, s := range snapshots {
		if strings.EqualFold(s.status, "Completed") {
			completed++
		} else {
			xpAvailable += s.xpReward
			if item, ok := challengeItemFromSnapshot(s); ok {
				items = append(items, item)
			}
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
	if len(items) > 0 {
		resp.Items = items
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
	return resp
}

// challengeItemFromSnapshot reconstruit une carte de défi depuis une ligne de cache.
// Retourne ok=false si la ligne n'a pas de titre rendu (snapshot legacy d'avant la
// migration render-columns, ou état non encore re-snapshotté) → on l'omet plutôt que
// d'afficher une carte vide.
func challengeItemFromSnapshot(s challengeSnapshotRow) (domain.ChallengeItem, bool) {
	if !s.title.Valid || s.title.String == "" {
		return domain.ChallengeItem{}, false
	}
	// Le vrai chemin GameCMS (display_path) permet au front de dériver la cadence
	// daily/weekly ; fallback sur la clé synthétique si absent (snapshot legacy).
	challengePath := s.challengePath
	if s.displayPath.Valid && s.displayPath.String != "" {
		challengePath = s.displayPath.String
	}
	item := domain.ChallengeItem{
		ChallengePath: challengePath,
		Title:         s.title.String,
	}
	if s.description.Valid && s.description.String != "" {
		d := s.description.String
		item.Description = &d
	}
	if s.imageURL.Valid && s.imageURL.String != "" {
		u := s.imageURL.String
		item.ImageURL = &u
	}
	if s.xpReward > 0 {
		xp := s.xpReward
		item.XPReward = &xp
	}
	if s.progressCurrent.Valid {
		c := int(s.progressCurrent.Int64)
		item.ProgressCurrent = &c
	}
	if s.progressTarget.Valid {
		t := int(s.progressTarget.Int64)
		item.ProgressTarget = &t
	}
	if item.ProgressCurrent != nil && item.ProgressTarget != nil && *item.ProgressTarget > 0 {
		pct := float64(*item.ProgressCurrent) / float64(*item.ProgressTarget) * 100.0
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		item.ProgressPercent = &pct
	}
	return item, true
}
