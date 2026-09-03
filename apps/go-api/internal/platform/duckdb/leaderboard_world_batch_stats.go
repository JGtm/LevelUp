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
	"errors"
	"fmt"
	"strings"
	"time"

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
// testable sans base ni réseau.
//
// Deux appelants, une seule définition (règle « ≤ 2 copies ») :
//   - le cron, avant de persister un lot fraîchement scrapé (reference = lot servi) ;
//   - le CLI -restore-best, pour décider si le lot SERVI mérite d'être remplacé par
//     un meilleur lot historique (reference = meilleur lot, candidate = lot servi).
func DegradedBatchReason(reference, candidate WorldCSRBatchStats) string {
	if float64(candidate.Rows) < DegradedBatchMinRowRatio*float64(reference.Rows) {
		return "effondrement du volume (moins de la moitié des lignes de référence)"
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
// cron et par le CLI de capture.
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

// WorldCSRBatchKey identifie le couple (titre, saison, playlist) auquel appartient
// un lot de classement.
type WorldCSRBatchKey struct {
	TitleSlug  string
	SeasonID   string
	PlaylistID string
}

// WorldCSRBatch identifie un lot capturé — son instant de capture, qui le distingue
// des autres lots du même couple — et porte sa qualité mesurée.
type WorldCSRBatch struct {
	FetchedAt time.Time
	Stats     WorldCSRBatchStats
}

// WorldCSRSeasonPlaylistPairs liste les couples (saison, playlist) présents dans les
// snapshots d'un titre. Sert au balayage de restauration (CLI -restore-best) : on ne
// peut pas deviner ces couples, ils dépendent de ce qui a été capturé au fil du temps.
func WorldCSRSeasonPlaylistPairs(ctx context.Context, db *sql.DB, titleSlug string) ([]WorldCSRBatchKey, error) {
	if strings.TrimSpace(titleSlug) == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	const q = `
		SELECT DISTINCT season_id, playlist_id
		FROM world_csr_leaderboard_snapshots
		WHERE title_slug = ? AND season_id <> '' AND playlist_id <> ''
		ORDER BY season_id, playlist_id`
	rows, err := db.QueryContext(ctx, q, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("WorldCSRSeasonPlaylistPairs: %w", err)
	}
	defer rows.Close()
	var out []WorldCSRBatchKey
	for rows.Next() {
		k := WorldCSRBatchKey{TitleSlug: titleSlug}
		if err := rows.Scan(&k.SeasonID, &k.PlaylistID); err != nil {
			return nil, fmt.Errorf("WorldCSRSeasonPlaylistPairs: scan: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// WorldCSRBestBatch retourne le MEILLEUR lot historique d'un couple, tous batches
// confondus (un batch = les lignes partageant un même fetched_at).
//
// Ordre de mérite : d'abord le nombre d'entrées IDENTIFIÉES (xuid), car c'est ce qui
// rend un relevé exploitable — un classement sans xuid ne se joint à rien ; puis le
// nombre de lignes (profondeur du top) ; puis la fraîcheur, qui ne départage que des
// lots de qualité égale. ok=false si le couple n'a aucun snapshot.
func WorldCSRBestBatch(ctx context.Context, db *sql.DB, key WorldCSRBatchKey) (WorldCSRBatch, bool, error) {
	slug := key.TitleSlug
	if strings.TrimSpace(slug) == "" {
		slug = titlePkg.DefaultSlug
	}
	const q = `
		SELECT fetched_at,
		       COUNT(*) AS rows_count,
		       COALESCE(SUM(CASE WHEN COALESCE(xuid, '') <> '' THEN 1 ELSE 0 END), 0) AS with_xuid
		FROM world_csr_leaderboard_snapshots
		WHERE title_slug = ? AND season_id = ? AND playlist_id = ?
		GROUP BY fetched_at
		ORDER BY with_xuid DESC, rows_count DESC, fetched_at DESC
		LIMIT 1`
	var b WorldCSRBatch
	err := db.QueryRowContext(ctx, q, slug, key.SeasonID, key.PlaylistID).
		Scan(&b.FetchedAt, &b.Stats.Rows, &b.Stats.WithXUID)
	if errors.Is(err, sql.ErrNoRows) {
		return WorldCSRBatch{}, false, nil
	}
	if err != nil {
		return WorldCSRBatch{}, false, fmt.Errorf("WorldCSRBestBatch: %w", err)
	}
	return b, true, nil
}

// WorldCSRBatchEntries relit les entrées d'un lot précis (identifié par son
// fetched_at) pour permettre sa RÉ-INSERTION à l'identique — rank, gamertag, xuid,
// CSR et palier préservés. Le FetchedAt des entrées retournées est laissé à zéro :
// c'est à l'appelant de poser un instant frais commun au lot restauré (sans quoi la
// vue _latest, qui sert le dernier fetched_at, ne servirait pas la restauration).
func WorldCSRBatchEntries(ctx context.Context, db *sql.DB, key WorldCSRBatchKey, fetchedAt time.Time) ([]domain.LeaderboardEntry, error) {
	slug := key.TitleSlug
	if strings.TrimSpace(slug) == "" {
		slug = titlePkg.DefaultSlug
	}
	const q = `
		SELECT rank, COALESCE(gamertag, ''), COALESCE(xuid, ''), csr_value, COALESCE(tier_derived, '')
		FROM world_csr_leaderboard_snapshots
		WHERE title_slug = ? AND season_id = ? AND playlist_id = ? AND fetched_at = ?
		ORDER BY rank`
	rows, err := db.QueryContext(ctx, q, slug, key.SeasonID, key.PlaylistID, fetchedAt)
	if err != nil {
		return nil, fmt.Errorf("WorldCSRBatchEntries: %w", err)
	}
	defer rows.Close()
	var out []domain.LeaderboardEntry
	for rows.Next() {
		e := domain.LeaderboardEntry{Season: key.SeasonID, Playlist: key.PlaylistID}
		if err := rows.Scan(&e.Rank, &e.Gamertag, &e.XUID, &e.CSRValue, &e.Tier); err != nil {
			return nil, fmt.Errorf("WorldCSRBatchEntries: scan: %w", err)
		}
		e.CSR = e.CSRValue
		e.Value = float64(e.CSRValue)
		e.Category = string(domain.LeaderboardCSRWorld)
		out = append(out, e)
	}
	return out, rows.Err()
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
