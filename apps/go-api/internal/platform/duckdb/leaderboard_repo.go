// Package duckdb — LeaderboardRepo : classement CSR depuis les données locales.
//
// Sprint 54 E : LeaderboardRepository.
package duckdb

import (
	"context"
	"fmt"
	"math"
	"time"

	"levelup/go-api/internal/domain"
)

// LeaderboardRepo implémente port.LeaderboardRepository.
type LeaderboardRepo struct {
	pdb *PlayerDB
}

// NewLeaderboardRepo crée un LeaderboardRepo.
func NewLeaderboardRepo(pdb *PlayerDB) *LeaderboardRepo {
	return &LeaderboardRepo{pdb: pdb}
}

// GetLocalLeaderboard retourne les joueurs locaux triés par CSR DESC.
// Utilise match_skill_rank (table v5.3) depuis la player DB du joueur courant.
// Dans l'architecture actuelle, ce chemin ne connaît pas les autres stats.duckdb locaux;
// il expose donc un classement dégradé mais fiable avec le CSR courant du joueur résolu.
// titleSlug, season et playlist sont des filtres optionnels (chaîne vide = tous).
func (r *LeaderboardRepo) GetLocalLeaderboard(ctx context.Context, titleSlug, season, playlist string) ([]domain.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	const q = `
		WITH ranked AS (
			SELECT
				msr.rating_value AS csr_value,
				COALESCE(NULLIF(msr.tier, ''), NULLIF(msr.tier_label, ''), '—') AS tier,
				COALESCE(msr.sub_tier, 0) AS sub_tier,
				COALESCE(mr.playlist_name, msr.playlist_group, '') AS playlist_name,
				COALESCE(
					mr.start_time_utc,
					mr.start_time AT TIME ZONE 'UTC',
					msr.start_time AT TIME ZONE 'UTC',
					msr.updated_at AT TIME ZONE 'UTC',
					msr.created_at AT TIME ZONE 'UTC'
				) AS sort_time,
				CASE
					WHEN mr.match_id IS NOT NULL AND (
						COALESCE(mr.is_ranked, FALSE)
						OR LOWER(COALESCE(mr.playlist_name, '')) LIKE '%ranked%'
						OR LOWER(COALESCE(mr.pair_name, '')) LIKE '%ranked%'
					) THEN 'CSR'
					WHEN mr.match_id IS NOT NULL THEN 'LUSR'
					ELSE UPPER(COALESCE(msr.rating_type, ''))
				END AS effective_type
			FROM match_skill_rank msr
			LEFT JOIN shared.match_registry mr ON mr.match_id = msr.match_id
		)
		SELECT csr_value, tier, sub_tier
		FROM ranked
		WHERE effective_type = 'CSR'
		  AND csr_value > 0
		  AND (? = '' OR playlist_name ILIKE ?)
		ORDER BY sort_time DESC NULLS LAST, csr_value DESC
		LIMIT 1`

	playlistFilter := "%" + playlist + "%"
	rows, err := r.pdb.ReadDB().Query(ctx, q, playlist, playlistFilter)
	if err != nil {
		return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard: %w", err)
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e domain.LeaderboardEntry
		var csrValue float64
		if err := rows.Scan(&csrValue, &e.Tier, &e.SubTier); err != nil {
			return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard scan: %w", err)
		}
		e.XUID = r.pdb.XUID
		e.Gamertag = r.pdb.Gamertag
		e.TitleSlug = titleSlug
		e.Season = season
		e.Playlist = playlist
		e.CSR = int(math.Round(csrValue))
		e.CSRValue = e.CSR
		e.IsLocal = true
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
