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

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

const (
	// DegradedBatchMinRowRatio : part MINIMALE des lignes du lot servi qu'un candidat
	// doit atteindre. En dessous (< 50 %), on considère la capture tronquée. Le
	// classement mondial ne perd pas la moitié de ses joueurs classés d'un jour à
	// l'autre — mais une page à moitié rendue, si.
	DegradedBatchMinRowRatio = 0.5
	// ServedXUIDCoverageFloor : couverture xuid à partir de laquelle le lot de
	// référence est considéré comme massivement identifié. Un candidat à 0 xuid face
	// à un tel lot signale un parsing cassé (le xuid vient du même payload que le
	// gamertag), pas une évolution réelle du classement.
	ServedXUIDCoverageFloor = 0.9
)

// DegradedBatchReason applique la décision D1 et retourne la cause du refus en clair
// (chaîne vide = lot acceptable). Fonction PURE : toute la règle métier est ici, donc
// testable sans base ni réseau. Appelée par le cron, avant de persister un lot
// fraîchement scrapé (reference = lot servi, candidate = lot scrapé).
//
// candidateDepthLimit = profondeur MAXIMALE que le candidat pouvait atteindre (la
// limite de son appelant) ; 0 = aucun plafond (échelle complète demandée).
//
// À quoi sert ce plafond : un lot déjà servi peut être plus profond que ce que le
// cron capture (200 entrées par playlist, contre jusqu'à 2500 pour un lot archivé
// sans limite). Comparer brutalement les volumes gèlerait le cron pour toujours dès
// qu'un lot profond a été servi : 200 lignes face à 2500 tombent sous la moitié, donc
// chaque cycle quotidien serait refusé alors qu'il a fait exactement le travail
// demandé. On borne donc la référence à ce que le candidat avait le DROIT de
// ramener : un lot au plafond de sa propre limite n'est jamais « effondré ». La
// règle d'identification (xuid), elle, ne dépend pas de la profondeur et s'applique
// telle quelle.
func DegradedBatchReason(reference, candidate WorldCSRBatchStats, candidateDepthLimit int) string {
	expected := reference.Rows
	if candidateDepthLimit > 0 && candidateDepthLimit < expected {
		expected = candidateDepthLimit
	}
	if float64(candidate.Rows) < DegradedBatchMinRowRatio*float64(expected) {
		return "effondrement du volume (moins de la moitié des lignes attendues)"
	}
	if reference.XUIDCoverage() >= ServedXUIDCoverageFloor && candidate.WithXUID == 0 {
		return "effondrement de l'identification (référence identifiée, candidat sans aucun xuid)"
	}
	return ""
}

// WorldCSRBatchStats décrit la QUALITÉ d'un lot du classement mondial : nombre de
// lignes et nombre d'entrées portant un xuid.
type WorldCSRBatchStats struct {
	Rows     int // nombre de lignes du lot
	WithXUID int // dont celles ayant un xuid non vide
}

// WorldCSRStatsOfEntries mesure un lot EN MÉMOIRE (lot fraîchement scrapé, pas encore
// écrit) avec EXACTEMENT la même définition que WorldCSRServedBatchStats côté base —
// sans quoi comparer les deux jeux de chiffres n'aurait aucun sens. Appelée par le
// cron.
func WorldCSRStatsOfEntries(entries []domain.LeaderboardEntry) WorldCSRBatchStats {
	stats := WorldCSRBatchStats{Rows: len(entries)}
	for _, e := range entries {
		if e.XUID != "" {
			stats.WithXUID++
		}
	}
	return stats
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
