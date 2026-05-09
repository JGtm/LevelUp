// Package duckdb — CompareRepo : stats normalisées depuis shared.match_participants.
//
// Sprint 54 C : CompareRepository.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// CompareRepo implémente port.CompareRepository.
type CompareRepo struct {
	pdb *PlayerDB
}

// NewCompareRepo crée un CompareRepo.
func NewCompareRepo(pdb *PlayerDB) *CompareRepo {
	return &CompareRepo{pdb: pdb}
}

// GetLocalStats calcule les stats normalisées d'un joueur local depuis shared.match_participants.
func (r *CompareRepo) GetLocalStats(ctx context.Context, xuid, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Résolveur canonique : v_gamertag_lookup (bots + xuid_aliases shared +
	// match_participants) en priorité, fallback shared.xuid_aliases pour les
	// xuid jamais croisés directement, puis '' si vraiment inconnu.
	const q = `
		SELECT
			mp.xuid,
			COALESCE(vg.gamertag, xa.gamertag, '') AS gamertag,
			COUNT(*)                               AS matches,
			AVG(CASE WHEN mp.outcome = 2 THEN 1.0 ELSE 0.0 END) AS win_rate,
			AVG(COALESCE(mp.kills, 0) + 0.33 * COALESCE(mp.assists, 0)) /
				NULLIF(AVG(COALESCE(mp.deaths, 0)), 0)            AS kda,
			AVG(COALESCE(mp.kills, 0)) /
				NULLIF(AVG(COALESCE(mp.deaths, 0)), 0)            AS kdr,
			AVG(COALESCE(mp.kills, 0))                            AS kills_per_game,
			AVG(COALESCE(mp.deaths, 0))                          AS deaths_per_game,
			AVG(COALESCE(mp.assists, 0))                         AS assists_per_game,
			AVG(COALESCE(mp.accuracy, 0.0)) / 100.0              AS accuracy,
			AVG(COALESCE(mp.damage_dealt, 0.0))                  AS damage_per_game
		FROM shared.match_participants mp
		LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = mp.xuid
		LEFT JOIN shared.xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE mp.xuid = ?
		GROUP BY mp.xuid, COALESCE(vg.gamertag, xa.gamertag, '')`

	row := r.pdb.Player.QueryRow(ctx, q, xuid)

	var s domain.NormalizedPlayerStats
	var kda, kdr sql.NullFloat64
	err := row.Scan(
		&s.XUID, &s.Gamertag, &s.Matches,
		&s.WinRate,
		&kda, &kdr,
		&s.KillsPerGame, &s.DeathsPerGame, &s.AssistsPerGame,
		&s.Accuracy, &s.DamagePerGame,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CompareRepo.GetLocalStats: joueur %s non trouvé", xuid)
	}
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetLocalStats: %w", err)
	}
	if kda.Valid {
		s.KDA = kda.Float64
	}
	if kdr.Valid {
		s.KDR = kdr.Float64
	}
	s.TitleSlug = titleSlug
	return &s, nil
}

// ResolveXUID retourne le XUID correspondant à un gamertag dans le registre partagé.
func (r *CompareRepo) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
		SELECT xuid FROM shared.xuid_aliases
		WHERE lower(gamertag) = lower(?)
		LIMIT 1`

	var xuid string
	err := r.pdb.Player.QueryRow(ctx, q, gamertag).Scan(&xuid)
	if err == sql.ErrNoRows {
		return "", nil // non trouvé localement — pas une erreur fatale
	}
	if err != nil {
		return "", fmt.Errorf("CompareRepo.ResolveXUID: %w", err)
	}
	return xuid, nil
}
