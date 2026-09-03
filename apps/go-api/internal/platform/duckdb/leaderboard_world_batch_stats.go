// Package duckdb — leaderboard_world_batch_stats.go : mesure de la QUALITÉ d'un lot
// du classement CSR mondial (nombre de lignes, nombre d'entrées identifiées par un
// xuid), et lecture de ces chiffres pour le lot ACTUELLEMENT SERVI.
//
// À quoi ça sert : le cron de capture compare le lot qu'il vient de scraper au lot
// servi avant de l'écrire (garde-fou D1, cf. scheduler/world_leaderboard_quality.go).
// Sans cette comparaison, un cycle dégradé remplace silencieusement un relevé sain —
// la vue world_csr_leaderboard_latest sert le DERNIER batch par (titre, saison,
// playlist), donc écrire 86 lignes sans xuid suffit à masquer 200 lignes
// intégralement identifiées (incident du 2026-07-07).
//
// Pourquoi c'est irréversible en pratique : ces snapshots sont la SEULE archive du
// classement mondial. Halo Waypoint retire les saisons passées de son site
// (csrseason13-2 a disparu en 2026-09) — ce qui n'a pas été capturé proprement, ou
// ce qui n'est plus servi, ne peut pas être re-scrapé.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
)

// WorldCSRBatchStats décrit la QUALITÉ d'un lot du classement mondial : nombre de
// lignes et nombre d'entrées portant un xuid.
type WorldCSRBatchStats struct {
	Rows     int // nombre de lignes du lot
	WithXUID int // dont celles ayant un xuid non vide
}

// XUIDCoverage retourne la part d'entrées portant un xuid (0..1). Lot vide → 0.
func (s WorldCSRBatchStats) XUIDCoverage() float64 {
	if s.Rows <= 0 {
		return 0
	}
	return float64(s.WithXUID) / float64(s.Rows)
}

// WorldCSRServedBatchStats lit la qualité du lot ACTUELLEMENT SERVI pour
// (titre, saison, playlist), c'est-à-dire les lignes que la vue
// world_csr_leaderboard_latest expose aujourd'hui (dernier fetched_at du couple).
//
// ok=false avec des stats à zéro quand aucun lot n'est servi (première capture de
// cette playlist) : il n'y a alors rien à protéger, l'appelant doit accepter le lot.
// `db` peut être RO ou RW (lecture seule, aucune écriture).
func WorldCSRServedBatchStats(ctx context.Context, db *sql.DB, titleSlug, seasonID, playlistID string) (WorldCSRBatchStats, bool, error) {
	if strings.TrimSpace(titleSlug) == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	const q = `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(xuid, '') <> '' THEN 1 ELSE 0 END), 0)
		FROM world_csr_leaderboard_latest
		WHERE title_slug = ? AND season_id = ? AND playlist_id = ?`
	var rows, withXUID int
	if err := db.QueryRowContext(ctx, q, titleSlug, seasonID, playlistID).Scan(&rows, &withXUID); err != nil {
		return WorldCSRBatchStats{}, false, fmt.Errorf("WorldCSRServedBatchStats: %w", err)
	}
	if rows == 0 {
		return WorldCSRBatchStats{}, false, nil
	}
	return WorldCSRBatchStats{Rows: rows, WithXUID: withXUID}, true, nil
}
