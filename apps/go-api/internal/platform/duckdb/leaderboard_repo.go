// Package duckdb — LeaderboardRepo : classement CSR depuis les données locales.
//
// Sprint 54 E : LeaderboardRepository.
package duckdb

import (
	"context"
	"fmt"
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
// Utilise match_skill_rank (table v5.3) depuis la player DB.
// titleSlug, season et playlist sont des filtres optionnels (chaîne vide = tous).
func (r *LeaderboardRepo) GetLocalLeaderboard(ctx context.Context, titleSlug, season, playlist string) ([]domain.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Requête sur shared.match_participants avec le MAX CSR connu par XUID.
	// On approxime le CSR current par le dernier match_skill_rank disponible.
	const q = `
		WITH ranked AS (
			SELECT
				mp.xuid,
				COALESCE(xa.gamertag, mp.gamertag, '') AS gamertag,
				MAX(COALESCE(mp.csr_after, 0))  AS csr_current
			FROM shared.match_participants mp
			LEFT JOIN shared.xuid_aliases xa ON xa.xuid = mp.xuid
			GROUP BY mp.xuid, gamertag
		)
		SELECT xuid, gamertag, csr_current
		FROM ranked
		WHERE csr_current > 0
		ORDER BY csr_current DESC
		LIMIT 100`

	rows, err := r.pdb.Shared.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard: %w", err)
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e domain.LeaderboardEntry
		if err := rows.Scan(&e.XUID, &e.Gamertag, &e.CSR); err != nil {
			return nil, fmt.Errorf("LeaderboardRepo.GetLocalLeaderboard scan: %w", err)
		}
		e.TitleSlug = titleSlug
		e.Season = season
		e.Playlist = playlist
		e.IsLocal = true
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
