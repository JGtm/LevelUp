// Package api — registry_catalog_expand.go : découverte « A à Z » des couples map-mode
// ENFANTS d'une playlist (même jamais joués), via la config playlist discovery-infiniteugc
// (sync.GetPlaylistConfig). Le drain (RunCatalogUGCDrain) ne part que de match_registry
// (joué) ; cette étape enfile en plus les enfants déclarés dans la config de chaque
// playlist, pour que le drain les nomme. Stocke aussi les poids de rotation.
package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// playlistExpandRPS : débit du fetch config playlist (discovery-infiniteugc). Conservateur.
const playlistExpandRPS = 3

// ExpandPlaylistChildren lit la config de chaque playlist vue en match (avec sa version,
// requise par l'API) et enfile ses couples map-mode enfants dans catalog_fetch_queue
// (le drain les nommera ensuite), en stockant les poids de rotation. Best-effort par
// playlist : un fetch raté est loggé et n'interrompt pas les autres. Retourne le nombre
// de couples enfants enfilés. In-process (handles DB du serveur) → pas de lock.
func (r *ServiceRegistry) ExpandPlaylistChildren(ctx context.Context, titleSlug string) (int, error) {
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return 0, err
	}
	defer closeAll()
	if metaSQL == nil || sharedSQL == nil {
		return 0, fmt.Errorf("DBs indisponibles pour %s", titleSlug)
	}
	if err := ensurePlaylistWeightsTable(ctx, metaSQL); err != nil {
		return 0, err
	}

	tokens, err := r.haloTokensForDrain(ctx, titleSlug)
	if err != nil {
		return 0, err
	}
	client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, playlistExpandRPS)

	// Playlists distinctes vues en match, avec leur version (l'API la requiert :
	// 404 sans version, 400 sur "latest"). 216 lignes non vides observées 2026-06-12.
	rows, err := sharedSQL.QueryContext(ctx, `
		SELECT DISTINCT playlist_id, playlist_version_id
		FROM match_registry
		WHERE playlist_id IS NOT NULL AND playlist_id != ''
		  AND playlist_version_id IS NOT NULL AND playlist_version_id != ''`)
	if err != nil {
		return 0, fmt.Errorf("query playlists: %w", err)
	}
	type pl struct{ id, ver string }
	var playlists []pl
	for rows.Next() {
		var p pl
		if err := rows.Scan(&p.id, &p.ver); err == nil {
			playlists = append(playlists, p)
		}
	}
	_ = rows.Close()

	slog.InfoContext(ctx, "catalog_expand: démarré", "module", logging.ModuleCatalog,
		"title", titleSlug, "playlists", len(playlists))

	enqueued, failed := 0, 0
	for _, p := range playlists {
		if err := ctx.Err(); err != nil {
			return enqueued, err
		}
		cfg, ferr := client.GetPlaylistConfig(ctx, p.id, p.ver)
		if ferr != nil {
			failed++
			// Best-effort par-playlist : DEBUG. Le total est remonté une fois en fin
			// de cycle (WARN si failed>0), pas N WARN ici. Un fetch raté au boot est
			// souvent un simple 429 transitoire (cf. cooldown pool).
			slog.DebugContext(ctx, "catalog_expand: config playlist échouée (best-effort)",
				"module", logging.ModuleCatalog, "playlist", p.id, "err", ferr)
			continue
		}
		for _, e := range cfg.Entries {
			if e.MapModePairAssetID == "" {
				continue
			}
			// ART-safe / PK-less (#23046) : catalog_fetch_queue n'a plus de PK
			// (rebuild_catalog_fetch_queue_drop_art_indexes) → INSERT OR IGNORE
			// échouait EN DUR ("no UNIQUE/PRIMARY KEY ... specify ON CONFLICT columns")
			// → expansion playlists morte. SELECT-then-INSERT (NOT EXISTS), comme
			// ops/catalog_queue.go + enqueueCatalogChild.
			if _, eerr := metaSQL.ExecContext(ctx,
				`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id)
				 SELECT ?, 'pair', ?, ?
				 WHERE NOT EXISTS (
				   SELECT 1 FROM catalog_fetch_queue
				   WHERE title_slug = ? AND asset_type = 'pair' AND asset_id = ?
				 )`,
				titleSlug, e.MapModePairAssetID, e.VersionID,
				titleSlug, e.MapModePairAssetID,
			); eerr != nil {
				slog.WarnContext(ctx, "catalog_expand: enqueue enfant échoué",
					"module", logging.ModuleCatalog, "pair", e.MapModePairAssetID, "err", eerr)
				continue
			}
			enqueued++
			upsertPlaylistWeight(ctx, metaSQL, titleSlug, p.id, e.MapModePairAssetID, e.Weight)
		}
	}
	observability.IncCounter("catalog_playlist_expand_total")
	// Résumé unique de fin de cycle : WARN si au moins une playlist a échoué
	// (signal agrégé visible), INFO sinon.
	if failed > 0 {
		slog.WarnContext(ctx, "catalog_expand: terminé avec échecs", "module", logging.ModuleCatalog,
			"playlists", len(playlists), "children_enqueued", enqueued, "playlists_failed", failed)
	} else {
		slog.InfoContext(ctx, "catalog_expand: terminé", "module", logging.ModuleCatalog,
			"playlists", len(playlists), "children_enqueued", enqueued, "playlists_failed", failed)
	}
	return enqueued, nil
}

// ensurePlaylistWeightsTable crée la table des poids si absente (idempotent).
func ensurePlaylistWeightsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS playlist_map_mode_weights (
			title_slug        VARCHAR NOT NULL,
			playlist_asset_id VARCHAR NOT NULL,
			pair_asset_id     VARCHAR NOT NULL,
			weight            DOUBLE,
			updated_at        TIMESTAMP,
			PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
		)`)
	if err != nil {
		return fmt.Errorf("create playlist_map_mode_weights: %w", err)
	}
	return nil
}

// upsertPlaylistWeight : wrapper best-effort de l'upsert ART-safe canonique
// (duckdb.UpsertRowNoConflict — SELECT-then-write, JAMAIS d'ON CONFLICT sur metadata,
// bug ART #23046). L'erreur est délibérément ignorée (poids best-effort, recalculé au
// prochain cycle catalog_expand).
func upsertPlaylistWeight(ctx context.Context, db *sql.DB, titleSlug, playlistID, pairID string, weight float64) {
	now := time.Now().UTC()
	_ = duckdb.UpsertRowNoConflict(ctx, db,
		`SELECT 1 FROM playlist_map_mode_weights WHERE title_slug=? AND playlist_asset_id=? AND pair_asset_id=?`,
		[]any{titleSlug, playlistID, pairID},
		`UPDATE playlist_map_mode_weights SET weight=?, updated_at=? WHERE title_slug=? AND playlist_asset_id=? AND pair_asset_id=?`,
		[]any{weight, now, titleSlug, playlistID, pairID},
		`INSERT INTO playlist_map_mode_weights (title_slug, playlist_asset_id, pair_asset_id, weight, updated_at) VALUES (?,?,?,?,?)`,
		[]any{titleSlug, playlistID, pairID, weight, now},
	)
}
