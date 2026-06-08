// Package duckdb — leaderboard_world_repo.go : lecture du classement CSR mondial
// (snapshots scrapés depuis Halo Waypoint) et des classements de stats
// communautaires (agrégation de shared.match_participants).
//
// Écriture des snapshots : InsertWorldCSRSnapshot (INSERT pur, règle ART).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// logModuleLeaderboard route les logs de lecture du classement vers
// logs/leaderboard.log (cf. observability/logging.ModuleLeaderboard).
const logModuleLeaderboard = "leaderboard"

// statLeaderboardMinMatches : nombre minimal de matchs pour figurer dans un
// classement de stats (évite les flukes sur 1-2 parties).
const statLeaderboardMinMatches = 10

// GetCSRWorldLeaderboard lit le dernier snapshot du classement CSR mondial pour
// une saison + playlist depuis world_csr_leaderboard_latest (shared).
// Le tier/sous-palier sont re-dérivés du CSR (source unique domain.DeriveCSRTier).
// is_local = true si le xuid correspond au joueur courant.
func (r *LeaderboardRepo) GetCSRWorldLeaderboard(
	ctx context.Context, season, playlist string, limit int,
) ([]domain.LeaderboardEntry, error) {
	if strings.TrimSpace(season) == "" || strings.TrimSpace(playlist) == "" {
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: season et playlist requis")
	}
	if limit <= 0 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: shared reader: %w", err)
	}
	defer release()

	const q = `
		SELECT rank, COALESCE(gamertag, ''), COALESCE(xuid, ''), csr_value
		FROM world_csr_leaderboard_latest
		WHERE season_id = ? AND playlist_id = ?
		ORDER BY rank ASC
		LIMIT ?`
	rows, err := sharedDB.QueryContext(ctx, q, season, playlist, limit)
	if err != nil {
		slog.WarnContext(ctx, "lecture classement CSR mondial échouée", "module", logModuleLeaderboard,
			"season", season, "playlist", playlist, "err", err)
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: query: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LeaderboardEntry, 0, limit)
	for rows.Next() {
		var rank, csr int
		var gamertag, xuid string
		if err := rows.Scan(&rank, &gamertag, &xuid, &csr); err != nil {
			return nil, fmt.Errorf("GetCSRWorldLeaderboard: scan: %w", err)
		}
		tier, subTier := domain.DeriveCSRTier(csr)
		out = append(out, domain.LeaderboardEntry{
			Rank:     rank,
			Gamertag: gamertag,
			XUID:     xuid,
			CSR:      csr,
			CSRValue: csr,
			Tier:     tier,
			SubTier:  subTier,
			Season:   season,
			Playlist: playlist,
			Category: string(domain.LeaderboardCSRWorld),
			Value:    float64(csr),
			IsLocal:  r.isLocalXUID(xuid),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, "classement CSR mondial lu", "module", logModuleLeaderboard,
		"season", season, "playlist", playlist, "entries", len(out))
	return out, nil
}

// statMetric décrit l'expression SQL d'agrégation et l'unité d'une catégorie.
type statMetric struct {
	expr string
	unit string
}

// statMetrics mappe chaque catégorie de stat à son agrégat (pas de magic string).
// GREATEST(...,1) / NULLIF évitent les divisions par zéro.
var statMetrics = map[domain.LeaderboardCategory]statMetric{
	domain.LeaderboardKills:         {"SUM(mp.kills)", ""},
	domain.LeaderboardDeaths:        {"SUM(mp.deaths)", ""},
	domain.LeaderboardAssists:       {"SUM(mp.assists)", ""},
	domain.LeaderboardKillsPerGame:  {"SUM(mp.kills) * 1.0 / COUNT(DISTINCT mp.match_id)", ""},
	domain.LeaderboardKDR:           {"SUM(mp.kills) * 1.0 / GREATEST(SUM(mp.deaths), 1)", ""},
	domain.LeaderboardKDA:           {"(SUM(mp.kills) + SUM(mp.assists) / 3.0) / GREATEST(SUM(mp.deaths), 1)", ""},
	domain.LeaderboardAccuracy:      {"SUM(mp.shots_hit) * 100.0 / NULLIF(SUM(mp.shots_fired), 0)", "%"},
	domain.LeaderboardDamage:        {"SUM(mp.damage_dealt)", ""},
	domain.LeaderboardDamagePerGame: {"SUM(mp.damage_dealt) * 1.0 / COUNT(DISTINCT mp.match_id)", ""},
}

// GetStatLeaderboard agrège shared.match_participants par xuid pour une catégorie
// de stat (joueurs réellement croisés). playlist : filtre optionnel (ILIKE sur
// match_registry.playlist_name). Bots exclus, seuil min de matchs appliqué.
func (r *LeaderboardRepo) GetStatLeaderboard(
	ctx context.Context, category domain.LeaderboardCategory, playlist string, limit int,
) ([]domain.LeaderboardEntry, error) {
	metric, ok := statMetrics[category]
	if !ok {
		return nil, fmt.Errorf("GetStatLeaderboard: catégorie inconnue %q", category)
	}
	if limit <= 0 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetStatLeaderboard: shared reader: %w", err)
	}
	defer release()

	args := []any{}
	playlistJoin, playlistWhere := "", ""
	if strings.TrimSpace(playlist) != "" {
		playlistJoin = "JOIN match_registry r ON r.match_id = mp.match_id"
		playlistWhere = "AND lower(COALESCE(r.playlist_name, '')) LIKE '%' || lower(?) || '%'"
		args = append(args, playlist)
	}
	// #nosec G201 -- metric.expr provient d'une allowlist interne (statMetrics), pas d'entrée utilisateur.
	q := fmt.Sprintf(`
		SELECT mp.xuid,
		       COALESCE(vg.gamertag, 'Joueur ' || RIGHT(mp.xuid, 4)) AS gamertag,
		       COUNT(DISTINCT mp.match_id) AS matches,
		       %s AS value
		FROM match_participants mp
		LEFT JOIN v_gamertag_lookup vg ON vg.xuid = mp.xuid
		%s
		WHERE mp.xuid NOT LIKE 'bid(%%'
		%s
		GROUP BY mp.xuid, vg.gamertag
		HAVING COUNT(DISTINCT mp.match_id) >= ? AND value IS NOT NULL
		ORDER BY value DESC
		LIMIT ?`, metric.expr, playlistJoin, playlistWhere)
	args = append(args, statLeaderboardMinMatches, limit)

	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "lecture classement de stats échouée", "module", logModuleLeaderboard,
			"category", string(category), "playlist", playlist, "err", err)
		return nil, fmt.Errorf("GetStatLeaderboard(%s): query: %w", category, err)
	}
	defer rows.Close()

	out := make([]domain.LeaderboardEntry, 0, limit)
	rank := 0
	for rows.Next() {
		var xuid, gamertag string
		var matches int
		var value float64
		if err := rows.Scan(&xuid, &gamertag, &matches, &value); err != nil {
			return nil, fmt.Errorf("GetStatLeaderboard(%s): scan: %w", category, err)
		}
		rank++
		out = append(out, domain.LeaderboardEntry{
			Rank:          rank,
			XUID:          xuid,
			Gamertag:      gamertag,
			Category:      string(category),
			Value:         value,
			Unit:          metric.unit,
			MatchesPlayed: matches,
			IsLocal:       r.isLocalXUID(xuid),
		})
	}
	return out, rows.Err()
}

// isLocalXUID indique si le xuid est celui du joueur courant (mise en évidence).
func (r *LeaderboardRepo) isLocalXUID(xuid string) bool {
	return xuid != "" && xuid == r.pdb.XUID
}

// InsertWorldCSRSnapshot persiste un lot d'entrées du classement CSR mondial en
// INSERT pur (règle ART — jamais d'UPDATE) dans world_csr_leaderboard_snapshots.
// `db` est une connexion shared en écriture (fournie par le job CLI). Retourne le
// nombre de lignes insérées.
func InsertWorldCSRSnapshot(ctx context.Context, db *sql.DB, entries []domain.LeaderboardEntry) (int, error) {
	const ins = `
		INSERT INTO world_csr_leaderboard_snapshots
			(season_id, playlist_id, rank, gamertag, csr_value, tier_derived, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	inserted := 0
	for _, e := range entries {
		if _, err := db.ExecContext(ctx, ins,
			e.Season, e.Playlist, e.Rank, e.Gamertag, e.CSRValue, e.Tier, e.FetchedAt,
		); err != nil {
			return inserted, fmt.Errorf("InsertWorldCSRSnapshot (rank %d): %w", e.Rank, err)
		}
		inserted++
	}
	return inserted, nil
}
