// Package ops — catalog_refresh.go : seed des tables catalog metadata
// (playlists_catalog, maps_catalog, game_variants_catalog,
// map_mode_pair_definitions + pair_mode_label_translations) depuis
// match_registry, SANS appel réseau.
//
// Extrait de cmd/populate-playlists-catalog (mode --from-match-registry) pour être
// appelable in-process par le dashboard admin ET au sync (résorption autonome des
// « playlists hors catalogue », sans flag).
//
// ART-safe : toutes les écritures passent par duckdb.UpsertRowNoConflict
// (SELECT-then-write), JAMAIS d'`INSERT ... ON CONFLICT DO UPDATE` — ce pattern
// FATAL-invalide metadata.duckdb (bug ART « Failed to delete all rows from index »),
// ce qui faisait retomber toute l'UI en anglais brut jusqu'au redémarrage. Même upsert
// canonique que le CatalogFetcherService (cf. ADR 0019). C'est ce qui permet de garder
// cette résolution TOUJOURS active (plus de LEVELUP_CATALOG_REFRESH).
//
// Spécifique Halo Infinite (rankedplaylists allowlist) — comme la source.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/platform/duckdb"
)

// CatalogSeedResult compte les upserts par table catalog.
type CatalogSeedResult struct {
	Playlists, Pairs, Maps, GameVariants int
}

// CatalogRefreshFromRegistry peuple les tables catalog depuis match_registry sans
// appel HTTP, en SELECT-then-write (ART-safe). Idempotent. Délègue chaque table à
// un helper dédié (≤80 lignes, SRP).
func CatalogRefreshFromRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (CatalogSeedResult, error) {
	var r CatalogSeedResult
	var err error
	if r.Playlists, err = refreshPlaylistsCatalog(ctx, metadataDB, sharedDB, titleSlug); err != nil {
		return r, err
	}
	if r.Maps, err = refreshMapsCatalog(ctx, metadataDB, sharedDB, titleSlug); err != nil {
		return r, err
	}
	if r.GameVariants, err = refreshGameVariantsCatalog(ctx, metadataDB, sharedDB, titleSlug); err != nil {
		return r, err
	}
	if r.Pairs, err = refreshPairsCatalog(ctx, metadataDB, sharedDB, titleSlug); err != nil {
		return r, err
	}
	return r, nil
}

// refreshPlaylistsCatalog : playlists_catalog depuis match_registry (ART-safe).
func refreshPlaylistsCatalog(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (int, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			playlist_id,
			COALESCE(MAX(playlist_version_id) FILTER (WHERE playlist_version_id IS NOT NULL AND playlist_version_id != ''), '') AS version_id,
			FIRST(playlist_name ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(is_ranked   ORDER BY start_time DESC NULLS LAST) AS is_ranked,
			MIN(start_time) AS first_seen,
			MAX(start_time) AS last_seen
		FROM match_registry
		WHERE playlist_id IS NOT NULL AND playlist_id != '' AND playlist_name IS NOT NULL
		GROUP BY playlist_id`)
	if err != nil {
		return 0, fmt.Errorf("query playlists: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, versionID, name string
		var isRanked bool
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&id, &versionID, &name, &isRanked, &firstSeen, &lastSeen); err != nil {
			return n, fmt.Errorf("scan playlist: %w", err)
		}
		// Allowlist : ne jamais rétrograder une playlist classée connue (classif
		// depuis match_registry peu fiable — source du bug récurrent is_ranked).
		if rankedplaylists.IsRanked(id) {
			isRanked = true
		}
		experience := classifyExperienceFromName(name, isRanked)
		if err := duckdb.UpsertRowNoConflict(ctx, metadataDB,
			`SELECT 1 FROM playlists_catalog WHERE title_slug = ? AND playlist_asset_id = ?`,
			[]any{titleSlug, id},
			`UPDATE playlists_catalog SET
				current_version_id = CASE WHEN ? != '' THEN ? ELSE current_version_id END,
				name_canonical = ?, experience = ?, is_ranked = ?, last_seen_at = ?,
				last_fetched_at = CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
			 WHERE title_slug = ? AND playlist_asset_id = ?`,
			[]any{versionID, versionID, name, experience, isRanked, lastSeen, titleSlug, id},
			`INSERT INTO playlists_catalog
				(title_slug, playlist_asset_id, current_version_id, name_canonical, experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
			 VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
			[]any{titleSlug, id, versionID, name, experience, isRanked, firstSeen, lastSeen},
		); err != nil {
			slog.WarnContext(ctx, "upsert playlist", "id", id, "err", err)
			continue
		}
		n++
	}
	return n, rows.Err()
}

// refreshMapsCatalog : maps_catalog depuis match_registry (ART-safe).
func refreshMapsCatalog(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (int, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			map_id,
			COALESCE(MAX(map_version_id) FILTER (WHERE map_version_id IS NOT NULL AND map_version_id != ''), '') AS version_id,
			FIRST(map_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE map_id IS NOT NULL AND map_id != '' AND map_name IS NOT NULL
		GROUP BY map_id`)
	if err != nil {
		return 0, fmt.Errorf("query maps: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			return n, fmt.Errorf("scan map: %w", err)
		}
		// Skip mode-variant names (> 3 mots) : les noms de map de base Halo Infinite
		// font ≤ 3 mots quelle que soit la langue ; les variantes embarquent le mode
		// (ex "Sentry Defense on Highpower", "du Lourd sur Aquarius").
		if len(strings.Fields(name)) > 3 {
			slog.DebugContext(ctx, "map skip: mode-variant name", "map_id", id, "name", name)
			continue
		}
		if err := duckdb.UpsertRowNoConflict(ctx, metadataDB,
			`SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ?`,
			[]any{titleSlug, id},
			`UPDATE maps_catalog SET
				current_version_id = CASE WHEN ? != '' THEN ? ELSE current_version_id END,
				name_canonical = ?, last_fetched_at = CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
			 WHERE title_slug = ? AND map_asset_id = ?`,
			[]any{versionID, versionID, name, titleSlug, id},
			`INSERT INTO maps_catalog
				(title_slug, map_asset_id, current_version_id, name_canonical, last_fetched_at)
			 VALUES (?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
			[]any{titleSlug, id, versionID, name},
		); err != nil {
			slog.WarnContext(ctx, "upsert map", "id", id, "err", err)
			continue
		}
		n++
	}
	return n, rows.Err()
}

// refreshGameVariantsCatalog : game_variants_catalog depuis match_registry (ART-safe).
func refreshGameVariantsCatalog(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (int, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			game_variant_id,
			COALESCE(MAX(game_variant_version_id) FILTER (WHERE game_variant_version_id IS NOT NULL AND game_variant_version_id != ''), '') AS version_id,
			FIRST(game_variant_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE game_variant_id IS NOT NULL AND game_variant_id != '' AND game_variant_name IS NOT NULL
		GROUP BY game_variant_id`)
	if err != nil {
		return 0, fmt.Errorf("query game_variants: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			return n, fmt.Errorf("scan game_variant: %w", err)
		}
		modeCanonical := classifyModeCanonicalFromName(name)
		if err := duckdb.UpsertRowNoConflict(ctx, metadataDB,
			`SELECT 1 FROM game_variants_catalog WHERE title_slug = ? AND game_variant_asset_id = ?`,
			[]any{titleSlug, id},
			`UPDATE game_variants_catalog SET
				current_version_id = CASE WHEN ? != '' THEN ? ELSE current_version_id END,
				name_canonical = ?, mode_canonical = ?, last_fetched_at = CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
			 WHERE title_slug = ? AND game_variant_asset_id = ?`,
			[]any{versionID, versionID, name, modeCanonical, titleSlug, id},
			`INSERT INTO game_variants_catalog
				(title_slug, game_variant_asset_id, current_version_id, name_canonical, mode_canonical, last_fetched_at)
			 VALUES (?, ?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
			[]any{titleSlug, id, versionID, name, modeCanonical},
		); err != nil {
			slog.WarnContext(ctx, "upsert game_variant", "id", id, "err", err)
			continue
		}
		n++
	}
	return n, rows.Err()
}

// refreshPairsCatalog : map_mode_pair_definitions + pair_mode_label_translations
// depuis match_registry (ART-safe). Tolère l'absence de pair_name_fr.
func refreshPairsCatalog(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (int, error) {
	const withFR = `
		SELECT pair_id,
			COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
			FIRST(pair_name    ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(pair_name_fr ORDER BY start_time DESC NULLS LAST) AS name_fr,
			FIRST(map_id          ORDER BY start_time DESC NULLS LAST) AS map_id,
			FIRST(game_variant_id ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
			FIRST(mode_category   ORDER BY start_time DESC NULLS LAST) AS mode_category
		FROM match_registry
		WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
		GROUP BY pair_id`
	const withoutFR = `
		SELECT pair_id,
			COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
			FIRST(pair_name   ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(map_id          ORDER BY start_time DESC NULLS LAST) AS map_id,
			FIRST(game_variant_id ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
			FIRST(mode_category   ORDER BY start_time DESC NULLS LAST) AS mode_category
		FROM match_registry
		WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
		GROUP BY pair_id`

	hasFR := true
	rows, err := sharedDB.QueryContext(ctx, withFR)
	if err != nil {
		hasFR = false
		slog.WarnContext(ctx, "pair_name_fr absent, fallback sans FR", "err", err)
		if rows, err = sharedDB.QueryContext(ctx, withoutFR); err != nil {
			return 0, fmt.Errorf("query pairs (fallback): %w", err)
		}
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, versionID, name string
		var nameFR, mapID, gameVariantID, modeCategory sql.NullString
		var scanErr error
		if hasFR {
			scanErr = rows.Scan(&id, &versionID, &name, &nameFR, &mapID, &gameVariantID, &modeCategory)
		} else {
			scanErr = rows.Scan(&id, &versionID, &name, &mapID, &gameVariantID, &modeCategory)
		}
		if scanErr != nil {
			return n, fmt.Errorf("scan pair: %w", scanErr)
		}
		if err := upsertPairDefinition(ctx, metadataDB, titleSlug, id, versionID, name, mapID.String, gameVariantID.String, modeCategory.String); err != nil {
			slog.WarnContext(ctx, "upsert pair", "id", id, "err", err)
			continue
		}
		n++
		if name != "" {
			if lblErr := upsertPairLabel(ctx, metadataDB, titleSlug, id, "en", name); lblErr != nil {
				slog.WarnContext(ctx, "upsert pair label (en)", "id", id, "err", lblErr)
			}
		}
		if nameFR.Valid && nameFR.String != "" {
			if lblErr := upsertPairLabel(ctx, metadataDB, titleSlug, id, "fr", nameFR.String); lblErr != nil {
				slog.WarnContext(ctx, "upsert pair label (fr)", "id", id, "err", lblErr)
			}
		}
	}
	return n, rows.Err()
}

// upsertPairDefinition : une ligne map_mode_pair_definitions (ART-safe).
func upsertPairDefinition(ctx context.Context, db *sql.DB, titleSlug, id, versionID, name, mapID, gvID, modeCategory string) error {
	return duckdb.UpsertRowNoConflict(ctx, db,
		`SELECT 1 FROM map_mode_pair_definitions WHERE title_slug = ? AND pair_asset_id = ?`,
		[]any{titleSlug, id},
		`UPDATE map_mode_pair_definitions SET
			current_version_id    = CASE WHEN ? != '' THEN ? ELSE current_version_id END,
			name_canonical        = ?,
			map_asset_id          = CASE WHEN ? != '' THEN ? ELSE map_asset_id END,
			game_variant_asset_id = CASE WHEN ? != '' THEN ? ELSE game_variant_asset_id END,
			mode_category         = CASE WHEN ? != '' THEN ? ELSE mode_category END,
			last_fetched_at       = CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		 WHERE title_slug = ? AND pair_asset_id = ?`,
		[]any{versionID, versionID, name, mapID, mapID, gvID, gvID, modeCategory, modeCategory, titleSlug, id},
		`INSERT INTO map_mode_pair_definitions
			(title_slug, pair_asset_id, current_version_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category, last_fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		[]any{titleSlug, id, versionID, name, mapID, gvID, modeCategory},
	)
}

// upsertPairLabel : une traduction pair_mode_label_translations (ART-safe).
func upsertPairLabel(ctx context.Context, db *sql.DB, titleSlug, id, lang, label string) error {
	return duckdb.UpsertRowNoConflict(ctx, db,
		`SELECT 1 FROM pair_mode_label_translations WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
		[]any{titleSlug, id, lang},
		`UPDATE pair_mode_label_translations SET label = ? WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
		[]any{label, titleSlug, id, lang},
		`INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label) VALUES (?, ?, ?, ?)`,
		[]any{titleSlug, id, lang, label},
	)
}

// classifyExperienceFromName dérive l'expérience depuis le nom de playlist et is_ranked.
func classifyExperienceFromName(name string, isRanked bool) string {
	if isRanked {
		return "ranked"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "big team") || strings.Contains(lower, "btb"):
		return "btb"
	case strings.Contains(lower, "firefight"):
		return "firefight"
	case strings.Contains(lower, "action sack"):
		return "action_sack"
	case strings.Contains(lower, "limited") || strings.Contains(lower, "event"):
		return "limited_time"
	default:
		return "social"
	}
}

// classifyModeCanonicalFromName dérive le mode canonique depuis le nom de game_variant.
func classifyModeCanonicalFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "slayer"):
		return "slayer"
	case strings.Contains(lower, "ctf") || strings.Contains(lower, "capture the flag"):
		return "ctf"
	case strings.Contains(lower, "oddball"):
		return "oddball"
	case strings.Contains(lower, "king of the hill") || strings.Contains(lower, "koth"):
		return "koth"
	case strings.Contains(lower, "stronghold"):
		return "strongholds"
	case strings.Contains(lower, "extraction"):
		return "extraction"
	case strings.Contains(lower, "fiesta"):
		return "fiesta"
	case strings.Contains(lower, "firefight") || strings.Contains(lower, "kotr"):
		return "firefight_kotr"
	case strings.Contains(lower, "attrition"):
		return "attrition"
	case strings.Contains(lower, "stockpile"):
		return "stockpile"
	case strings.Contains(lower, "total control"):
		return "total_control"
	default:
		return "unknown"
	}
}
