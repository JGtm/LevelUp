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
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
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

	// NB : le snapshot Halo Waypoint ne publie pas de xuid (seulement gamertag) —
	// la table world_csr_leaderboard_snapshots n'a donc pas de colonne xuid.
	// On sélectionne '' pour rester compatible avec le scan (xuid vide → is_local
	// toujours false sur ce classement mondial, attendu).
	const q = `
		SELECT rank, COALESCE(gamertag, ''), '' AS xuid, csr_value
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

// GetWorldLeaderboardCatalog liste les saisons et playlists pour lesquelles des
// snapshots CSR mondiaux existent réellement (distinct sur la vue _latest).
// Alimente les sélecteurs dynamiques de la page Classement. Les saisons sont
// triées du plus récent au plus ancien (ordre lexicographique inverse, cohérent
// avec le format "csrseasonNN-M"). Les playlists reçoivent un libellé via la
// référence rankedplaylists (FR si dispo, sinon EN, sinon l'asset_id brut).
func (r *LeaderboardRepo) GetWorldLeaderboardCatalog(ctx context.Context) (domain.LeaderboardCatalog, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: shared reader: %w", err)
	}
	defer release()

	seasons, err := scanCatalogColumn(ctx, sharedDB,
		`SELECT DISTINCT season_id FROM world_csr_leaderboard_latest
		 WHERE season_id <> '' ORDER BY season_id DESC`, nil)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: seasons: %w", err)
	}
	playlists, err := scanCatalogColumn(ctx, sharedDB,
		`SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest
		 WHERE playlist_id <> '' ORDER BY playlist_id`, playlistDisplayName)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: playlists: %w", err)
	}
	slog.DebugContext(ctx, "catalogue classement mondial lu", "module", logModuleLeaderboard,
		"seasons", len(seasons), "playlists", len(playlists))
	return domain.LeaderboardCatalog{Seasons: seasons, Playlists: playlists}, nil
}

// scanCatalogColumn lit une colonne d'IDs et construit des refs. displayFn
// dérive le libellé depuis l'ID (nil → libellé = ID).
func scanCatalogColumn(ctx context.Context, db *sql.DB, q string, displayFn func(string) string) ([]domain.LeaderboardCatalogRef, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LeaderboardCatalogRef
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		display := id
		if displayFn != nil {
			display = displayFn(id)
		}
		out = append(out, domain.LeaderboardCatalogRef{ID: id, DisplayName: display})
	}
	return out, rows.Err()
}

// playlistDisplayName résout un asset_id de playlist en libellé (FR > EN > id).
func playlistDisplayName(assetID string) string {
	if pl, ok := rankedplaylists.Lookup(assetID); ok {
		if pl.NameFR != "" {
			return pl.NameFR
		}
		if pl.NameEN != "" {
			return pl.NameEN
		}
	}
	return assetID
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
// de stat (joueurs réellement croisés). Filtres optionnels :
//   - playlist : ILIKE sur match_registry.playlist_name.
//   - season   : égalité exacte sur match_registry.season_id (format interne
//     "CsrSeasonN", PAS le format Waypoint du classement mondial — les deux
//     domaines de saison sont distincts).
//
// Bots exclus, seuil min de matchs appliqué. Le JOIN match_registry n'est ajouté
// que si au moins un des deux filtres est actif (évite un JOIN inutile sinon).
func (r *LeaderboardRepo) GetStatLeaderboard(
	ctx context.Context, category domain.LeaderboardCategory, playlist, season string, limit int,
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
	registryWhere := strings.Builder{}
	playlist, season = strings.TrimSpace(playlist), strings.TrimSpace(season)
	if playlist != "" {
		registryWhere.WriteString("AND lower(COALESCE(r.playlist_name, '')) LIKE '%' || lower(?) || '%'\n")
		args = append(args, playlist)
	}
	if season != "" {
		registryWhere.WriteString("AND r.season_id = ?\n")
		args = append(args, season)
	}
	registryJoin := ""
	if registryWhere.Len() > 0 {
		registryJoin = "JOIN match_registry r ON r.match_id = mp.match_id"
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
		LIMIT ?`, metric.expr, registryJoin, registryWhere.String())
	args = append(args, statLeaderboardMinMatches, limit)

	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "lecture classement de stats échouée", "module", logModuleLeaderboard,
			"category", string(category), "playlist", playlist, "season", season, "err", err)
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

// WorldCSRSnapshotAge retourne l'âge du snapshot le plus récent pour une saison
// (toutes playlists confondues). ok=false si aucun snapshot. Utilisé par le cron
// comme garde-fou de fraîcheur : évite de re-scraper Halo Waypoint à chaque boot /
// hot-reload Air si un snapshot récent existe déjà.
//
// L'âge est calculé ENTIÈREMENT en SQL (CURRENT_TIMESTAMP - max(written_at)) :
// les deux timestamps partagent l'horloge/zone de la DB, ce qui évite le piège
// TZ d'un timestamp DuckDB naïf relu comme UTC puis comparé à time.Now() local
// (cf. reference_timezone_canonical_pattern). `db` peut être RO ou RW.
func WorldCSRSnapshotAge(ctx context.Context, db *sql.DB, seasonID string) (time.Duration, bool, error) {
	const q = `
		SELECT date_part('epoch', CURRENT_TIMESTAMP - max(written_at))
		FROM world_csr_leaderboard_snapshots
		WHERE season_id = ?`
	var ageSeconds sql.NullFloat64
	if err := db.QueryRowContext(ctx, q, seasonID).Scan(&ageSeconds); err != nil {
		return 0, false, fmt.Errorf("WorldCSRSnapshotAge: %w", err)
	}
	if !ageSeconds.Valid {
		return 0, false, nil
	}
	return time.Duration(ageSeconds.Float64 * float64(time.Second)), true, nil
}

// InsertWorldCSRSnapshot persiste un lot d'entrées du classement CSR mondial en
// INSERT pur (règle ART — jamais d'UPDATE) dans world_csr_leaderboard_snapshots.
// `db` est une connexion shared en écriture (cron ou job CLI). Retourne le nombre
// de lignes insérées.
//
// ATOMIQUE : tout le lot est inséré dans une seule transaction → en cas d'échec
// en cours de route, rien n'est commité (pas de demi-snapshot). Garantit aussi un
// `fetched_at` cohérent sur tout le lot (déjà fixé en amont par le scraper), ce
// qui permet à la vue _latest de grouper par batch de scrape.
func InsertWorldCSRSnapshot(ctx context.Context, db *sql.DB, entries []domain.LeaderboardEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("InsertWorldCSRSnapshot: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op si Commit a réussi ; rollback sinon

	const ins = `
		INSERT INTO world_csr_leaderboard_snapshots
			(season_id, playlist_id, rank, gamertag, csr_value, tier_derived, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	for _, e := range entries {
		if _, err := tx.ExecContext(ctx, ins,
			e.Season, e.Playlist, e.Rank, e.Gamertag, e.CSRValue, e.Tier, e.FetchedAt,
		); err != nil {
			return 0, fmt.Errorf("InsertWorldCSRSnapshot (rank %d): %w", e.Rank, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("InsertWorldCSRSnapshot: commit: %w", err)
	}
	return len(entries), nil
}
