// Package ops — catalog_refresh.go : seed des tables catalog metadata
// (playlists_catalog, maps_catalog, game_variants_catalog,
// map_mode_pair_definitions + pair_mode_label_translations) depuis
// match_registry, SANS appel réseau.
//
// Extrait de cmd/populate-playlists-catalog (mode --from-match-registry) pour
// être appelable in-process par le dashboard admin (action « rafraîchir le
// catalogue ») — le CLI reste le wrapper du même code. Le mode drain
// DiscoveryUGC (réseau, rate-limité) reste CLI-only.
//
// Écritures metadata basse fréquence : les ON CONFLICT d'origine sont
// conservés tels quels (tables catalog créées par les migrations Go avec PK —
// pas le piège des tables prebuilt sans contrainte).
//
// Spécifique Halo Infinite (rankedplaylists allowlist) — comme la source.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// CatalogSeedResult compte les upserts par table catalog.
type CatalogSeedResult struct {
	Playlists, Pairs, Maps, GameVariants int
}

// CatalogRefreshEnabled : interrupteur MAÎTRE des écritures catalogue (drain réseau
// hebdo ET refresh in-sync). LEVELUP_CATALOG_REFRESH=1|true active. OFF par défaut :
// les UPSERT catalogue (ON CONFLICT DO UPDATE sur metadata) peuvent taper le bug ART
// « Failed to delete all rows from index » qui invalide metadata.duckdb (cf. mémoire
// projet : l'hypothèse « ces tables ont une PK donc c'est sûr » a été démentie en prod
// 2026-05-24). À réactiver quand les écritures catalogue seront migrées append-only.
// Indépendant de LEVELUP_SYNC_RESOLVE_ASSETS (résolution des NOMS, ART-safe, autonome).
func CatalogRefreshEnabled() bool {
	v := strings.TrimSpace(os.Getenv("LEVELUP_CATALOG_REFRESH"))
	return v == "1" || strings.EqualFold(v, "true")
}

// CatalogRefreshFromRegistry peuple les tables catalog depuis match_registry
// sans appel HTTP. Les noms en/fr sont pris directement des colonnes de match_registry.
func CatalogRefreshFromRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (CatalogSeedResult, error) {
	var r CatalogSeedResult

	// --- Playlists ---
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
		GROUP BY playlist_id
	`)
	if err != nil {
		return r, fmt.Errorf("query playlists: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		var isRanked bool
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&id, &versionID, &name, &isRanked, &firstSeen, &lastSeen); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan playlist: %w", err)
		}
		// Conformité allowlist : ne jamais rétrograder une playlist classée connue
		// (la classif depuis match_registry est peu fiable — source du bug récurrent).
		if rankedplaylists.IsRanked(id) {
			isRanked = true
		}
		experience := classifyExperienceFromName(name, isRanked)
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO playlists_catalog
				(title_slug, playlist_asset_id, current_version_id, name_canonical, experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, playlist_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				experience         = excluded.experience,
				is_ranked          = excluded.is_ranked,
				last_seen_at       = excluded.last_seen_at,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, experience, isRanked, firstSeen, lastSeen)
		if err != nil {
			slog.WarnContext(ctx, "upsert playlist", "id", id, "err", err)
			continue
		}
		r.Playlists++
	}
	rows.Close()

	// --- Maps ---
	rows, err = sharedDB.QueryContext(ctx, `
		SELECT
			map_id,
			COALESCE(MAX(map_version_id) FILTER (WHERE map_version_id IS NOT NULL AND map_version_id != ''), '') AS version_id,
			FIRST(map_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE map_id IS NOT NULL AND map_id != '' AND map_name IS NOT NULL
		GROUP BY map_id
	`)
	if err != nil {
		return r, fmt.Errorf("query maps: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan map: %w", err)
		}
		// Skip mode-variant names (≥ 4 words): Halo Infinite base map names are ≤ 3 words
		// regardless of locale. Mode variants embed the mode in the name, e.g.
		// "du Lourd sur Aquarius" (FR) or "Sentry Defense on Highpower" (EN).
		if len(strings.Fields(name)) > 3 {
			slog.DebugContext(ctx, "map skip: mode-variant name", "map_id", id, "name", name)
			continue
		}
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO maps_catalog
				(title_slug, map_asset_id, current_version_id, name_canonical, last_fetched_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, map_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name)
		if err != nil {
			slog.WarnContext(ctx, "upsert map", "id", id, "err", err)
			continue
		}
		r.Maps++
	}
	rows.Close()

	// --- Game Variants ---
	rows, err = sharedDB.QueryContext(ctx, `
		SELECT
			game_variant_id,
			COALESCE(MAX(game_variant_version_id) FILTER (WHERE game_variant_version_id IS NOT NULL AND game_variant_version_id != ''), '') AS version_id,
			FIRST(game_variant_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE game_variant_id IS NOT NULL AND game_variant_id != '' AND game_variant_name IS NOT NULL
		GROUP BY game_variant_id
	`)
	if err != nil {
		return r, fmt.Errorf("query game_variants: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan game_variant: %w", err)
		}
		modeCanonical := classifyModeCanonicalFromName(name)
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO game_variants_catalog
				(title_slug, game_variant_asset_id, current_version_id, name_canonical, mode_canonical, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, game_variant_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				mode_canonical     = excluded.mode_canonical,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, modeCanonical)
		if err != nil {
			slog.WarnContext(ctx, "upsert game_variant", "id", id, "err", err)
			continue
		}
		r.GameVariants++
	}
	rows.Close()

	// --- Pairs (+ pair_mode_label_translations) ---
	// Essai avec pair_name_fr ; fallback sans si la colonne n'existe pas.
	pairQuery := `
		SELECT
			pair_id,
			COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
			FIRST(pair_name    ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(pair_name_fr ORDER BY start_time DESC NULLS LAST) AS name_fr,
			FIRST(map_id           ORDER BY start_time DESC NULLS LAST) AS map_id,
			FIRST(game_variant_id  ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
			FIRST(mode_category    ORDER BY start_time DESC NULLS LAST) AS mode_category
		FROM match_registry
		WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
		GROUP BY pair_id`
	rows, err = sharedDB.QueryContext(ctx, pairQuery)
	hasFR := true
	if err != nil {
		// pair_name_fr absent → fallback sans cette colonne
		hasFR = false
		slog.WarnContext(ctx, "pair_name_fr absent, fallback sans FR", "err", err)
		pairQuery = `
			SELECT
				pair_id,
				COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
				FIRST(pair_name   ORDER BY start_time DESC NULLS LAST) AS name,
				FIRST(map_id          ORDER BY start_time DESC NULLS LAST) AS map_id,
				FIRST(game_variant_id ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
				FIRST(mode_category   ORDER BY start_time DESC NULLS LAST) AS mode_category
			FROM match_registry
			WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
			GROUP BY pair_id`
		rows, err = sharedDB.QueryContext(ctx, pairQuery)
		if err != nil {
			return r, fmt.Errorf("query pairs (fallback): %w", err)
		}
	}
	for rows.Next() {
		var id, versionID, name string
		var nameFR sql.NullString
		var mapID, gameVariantID, modeCategory sql.NullString
		var scanErr error
		if hasFR {
			scanErr = rows.Scan(&id, &versionID, &name, &nameFR, &mapID, &gameVariantID, &modeCategory)
		} else {
			scanErr = rows.Scan(&id, &versionID, &name, &mapID, &gameVariantID, &modeCategory)
		}
		if scanErr != nil {
			rows.Close()
			return r, fmt.Errorf("scan pair: %w", scanErr)
		}
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO map_mode_pair_definitions
				(title_slug, pair_asset_id, current_version_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, pair_asset_id) DO UPDATE SET
				current_version_id    = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical        = excluded.name_canonical,
				map_asset_id          = CASE WHEN excluded.map_asset_id != '' THEN excluded.map_asset_id ELSE map_asset_id END,
				game_variant_asset_id = CASE WHEN excluded.game_variant_asset_id != '' THEN excluded.game_variant_asset_id ELSE game_variant_asset_id END,
				mode_category         = CASE WHEN excluded.mode_category != '' THEN excluded.mode_category ELSE mode_category END,
				last_fetched_at       = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, mapID.String, gameVariantID.String, modeCategory.String)
		if err != nil {
			slog.WarnContext(ctx, "upsert pair", "id", id, "err", err)
			continue
		}
		r.Pairs++

		// pair_mode_label_translations — EN
		if name != "" {
			metadataDB.ExecContext(ctx, `
				INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label)
				VALUES (?, ?, 'en', ?)
				ON CONFLICT (title_slug, pair_asset_id, lang) DO UPDATE SET label = excluded.label
			`, titleSlug, id, name) //nolint:errcheck
		}
		// pair_mode_label_translations — FR
		if nameFR.Valid && nameFR.String != "" {
			metadataDB.ExecContext(ctx, `
				INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label)
				VALUES (?, ?, 'fr', ?)
				ON CONFLICT (title_slug, pair_asset_id, lang) DO UPDATE SET label = excluded.label
			`, titleSlug, id, nameFR.String) //nolint:errcheck
		}
	}
	rows.Close()

	return r, nil
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
