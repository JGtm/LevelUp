// Package duckdb — catalog_repo_write.go : côté ÉCRITURE du catalogue (drain + upserts).
// Extrait de service.CatalogFetcherService (K1j, 2026-07-06) pour que la couche service ne
// tienne plus de *sql.DB brut (ADR 0025 D-MV2) et n'ait plus sa copie locale de l'upsert
// ART-safe (le repo utilise UpsertRowNoConflict canonique). Toutes les écritures sont
// ART-safe (SELECT-then-write, JAMAIS d'ON CONFLICT sur metadata.duckdb, bug #23046).
//
// Tient un *sql.DB brut (comportement identique à l'ex-service : ni recovery ni reopen —
// le CLI populate-playlists-catalog l'ouvre via sql.Open, le serveur via dataQualityHandles).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// CatalogWriterDB implémente port.CatalogWriter sur un *sql.DB metadata brut.
type CatalogWriterDB struct {
	db *sql.DB
}

// NewCatalogWriter construit le writer catalogue depuis un *sql.DB metadata.
func NewCatalogWriter(db *sql.DB) *CatalogWriterDB {
	return &CatalogWriterDB{db: db}
}

// SelectPending retourne les entrées de catalog_fetch_queue dont l'asset n'est PAS
// encore présent dans la table catalogue de son type (append-only : la file n'est
// jamais mutée ; un asset résolu en sort naturellement via ce NOT EXISTS).
func (r *CatalogWriterDB) SelectPending(ctx context.Context, titleSlug string) ([]domain.CatalogQueueEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT q.asset_type, q.asset_id, COALESCE(q.version_id, '')
		 FROM catalog_fetch_queue q
		 WHERE q.title_slug = ?
		   AND CASE q.asset_type
		         WHEN 'playlist'     THEN NOT EXISTS (SELECT 1 FROM playlists_catalog c         WHERE c.title_slug = q.title_slug AND c.playlist_asset_id     = q.asset_id)
		         WHEN 'pair'         THEN NOT EXISTS (SELECT 1 FROM map_mode_pair_definitions c WHERE c.title_slug = q.title_slug AND c.pair_asset_id         = q.asset_id)
		         WHEN 'map'          THEN NOT EXISTS (SELECT 1 FROM maps_catalog c              WHERE c.title_slug = q.title_slug AND c.map_asset_id          = q.asset_id)
		         WHEN 'game_variant' THEN NOT EXISTS (SELECT 1 FROM game_variants_catalog c     WHERE c.title_slug = q.title_slug AND c.game_variant_asset_id = q.asset_id)
		         ELSE TRUE
		       END
		 ORDER BY q.enqueued_at ASC`,
		titleSlug,
	)
	if err != nil {
		return nil, fmt.Errorf("select queue: %w", err)
	}
	defer rows.Close()
	var out []domain.CatalogQueueEntry
	for rows.Next() {
		var e domain.CatalogQueueEntry
		if err := rows.Scan(&e.AssetType, &e.AssetID, &e.VersionID); err != nil {
			return nil, fmt.Errorf("scan queue: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// enqueueChild ajoute un asset enfant à catalog_fetch_queue s'il n'y est pas déjà.
// ART-safe : INSERT … SELECT … WHERE NOT EXISTS (catalog_fetch_queue n'a plus de PK →
// INSERT OR IGNORE ne dédupliquerait plus). version_id=” : le drain le résoudra.
// Best-effort (erreur loggée en debug).
func (r *CatalogWriterDB) enqueueChild(ctx context.Context, titleSlug, assetType, assetID string) {
	if assetID == "" {
		return
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id)
		 SELECT ?, ?, ?, ''
		 WHERE NOT EXISTS (
		   SELECT 1 FROM catalog_fetch_queue
		   WHERE title_slug = ? AND asset_type = ? AND asset_id = ?
		 )`,
		titleSlug, assetType, assetID, titleSlug, assetType, assetID,
	); err != nil {
		slog.DebugContext(ctx, "catalog re-enqueue: INSERT échoué",
			"asset_type", assetType, "asset_id", assetID, "err", err)
	}
}

// UpsertPlaylist persiste une playlist + ses pair_links + ré-enqueue les pairs enfants.
// isRanked/experience sont déjà corrigés par le caller (gate rankedplaylists côté service).
func (r *CatalogWriterDB) UpsertPlaylist(ctx context.Context, titleSlug string, pl canonical.CanonicalPlaylist, isRanked bool, experience string) error {
	now := time.Now().UTC()
	err := UpsertRowNoConflict(ctx, r.db,
		`SELECT 1 FROM playlists_catalog WHERE title_slug = ? AND playlist_asset_id = ?`,
		[]any{titleSlug, pl.AssetID},
		`UPDATE playlists_catalog SET
		   current_version_id = ?, name_canonical = ?, experience = ?,
		   is_ranked = ?, last_seen_at = ?, last_fetched_at = ?
		 WHERE title_slug = ? AND playlist_asset_id = ?`,
		[]any{pl.VersionID, pl.NameCanonical, experience, isRanked, now, now, titleSlug, pl.AssetID},
		`INSERT INTO playlists_catalog
		   (title_slug, playlist_asset_id, current_version_id, name_canonical, experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?)`,
		[]any{titleSlug, pl.AssetID, pl.VersionID, pl.NameCanonical, experience, isRanked, now, now, now},
	)
	if err != nil {
		return fmt.Errorf("upsert playlist: %w", err)
	}
	// Re-enqueue les pairs liés s'ils ne sont pas dans le catalogue. ART-safe :
	// playlist_pair_links GARDE sa PK → SELECT-then-INSERT (NOT EXISTS), pas de conflit
	// (comportement OR IGNORE préservé : pas d'update du weight si la ligne existe).
	for _, link := range pl.PairLinks {
		_, _ = r.db.ExecContext(ctx,
			`INSERT INTO playlist_pair_links (title_slug, playlist_asset_id, pair_asset_id, weight)
			 SELECT ?, ?, ?, ?
			 WHERE NOT EXISTS (
			   SELECT 1 FROM playlist_pair_links
			   WHERE title_slug = ? AND playlist_asset_id = ? AND pair_asset_id = ?
			 )`,
			titleSlug, pl.AssetID, link.PairAssetID, link.Weight,
			titleSlug, pl.AssetID, link.PairAssetID,
		)
		r.enqueueChild(ctx, titleSlug, "pair", link.PairAssetID)
	}
	return nil
}

// UpsertPair persiste un pair + ses labels normalisés multi-langues + ré-enqueue
// map/game_variant enfants.
func (r *CatalogWriterDB) UpsertPair(ctx context.Context, titleSlug string, p canonical.CanonicalPair) error {
	now := time.Now().UTC()
	err := UpsertRowNoConflict(ctx, r.db,
		`SELECT 1 FROM map_mode_pair_definitions WHERE title_slug = ? AND pair_asset_id = ?`,
		[]any{titleSlug, p.AssetID},
		`UPDATE map_mode_pair_definitions SET
		   current_version_id = ?, name_canonical = ?, map_asset_id = ?,
		   game_variant_asset_id = ?, mode_category = ?, last_fetched_at = ?
		 WHERE title_slug = ? AND pair_asset_id = ?`,
		[]any{p.VersionID, p.NameCanonical, p.MapAssetID, p.GameVariantAssetID, p.ModeCategory, now, titleSlug, p.AssetID},
		`INSERT INTO map_mode_pair_definitions
		   (title_slug, pair_asset_id, current_version_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category, last_fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{titleSlug, p.AssetID, p.VersionID, p.NameCanonical, p.MapAssetID, p.GameVariantAssetID, p.ModeCategory, now},
	)
	if err != nil {
		return fmt.Errorf("upsert pair: %w", err)
	}
	for lang, label := range p.ModeLabels {
		_ = UpsertRowNoConflict(ctx, r.db,
			`SELECT 1 FROM pair_mode_label_translations WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
			[]any{titleSlug, p.AssetID, lang},
			`UPDATE pair_mode_label_translations SET label = ? WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
			[]any{label, titleSlug, p.AssetID, lang},
			`INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label) VALUES (?, ?, ?, ?)`,
			[]any{titleSlug, p.AssetID, lang, label},
		)
	}
	r.enqueueChild(ctx, titleSlug, "map", p.MapAssetID)
	r.enqueueChild(ctx, titleSlug, "game_variant", p.GameVariantAssetID)
	return nil
}

// UpsertMap persiste une map du catalogue.
func (r *CatalogWriterDB) UpsertMap(ctx context.Context, titleSlug string, m canonical.CanonicalMap) error {
	now := time.Now().UTC()
	err := UpsertRowNoConflict(ctx, r.db,
		`SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ?`,
		[]any{titleSlug, m.AssetID},
		`UPDATE maps_catalog SET
		   current_version_id = ?, name_canonical = ?, image_url = ?, last_fetched_at = ?
		 WHERE title_slug = ? AND map_asset_id = ?`,
		[]any{m.VersionID, m.NameCanonical, m.ImageURL, now, titleSlug, m.AssetID},
		`INSERT INTO maps_catalog (title_slug, map_asset_id, current_version_id, name_canonical, image_url, last_fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		[]any{titleSlug, m.AssetID, m.VersionID, m.NameCanonical, m.ImageURL, now},
	)
	if err != nil {
		return fmt.Errorf("upsert map: %w", err)
	}
	return nil
}

// UpsertGameVariant persiste un game_variant du catalogue.
func (r *CatalogWriterDB) UpsertGameVariant(ctx context.Context, titleSlug string, gv canonical.CanonicalGameVariant) error {
	now := time.Now().UTC()
	err := UpsertRowNoConflict(ctx, r.db,
		`SELECT 1 FROM game_variants_catalog WHERE title_slug = ? AND game_variant_asset_id = ?`,
		[]any{titleSlug, gv.AssetID},
		`UPDATE game_variants_catalog SET
		   current_version_id = ?, name_canonical = ?, mode_canonical = ?,
		   game_variant_category = ?, last_fetched_at = ?
		 WHERE title_slug = ? AND game_variant_asset_id = ?`,
		[]any{gv.VersionID, gv.NameCanonical, string(gv.ModeCanonical), gv.GameVariantCategory, now, titleSlug, gv.AssetID},
		`INSERT INTO game_variants_catalog (title_slug, game_variant_asset_id, current_version_id, name_canonical, mode_canonical, game_variant_category, last_fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		[]any{titleSlug, gv.AssetID, gv.VersionID, gv.NameCanonical, string(gv.ModeCanonical), gv.GameVariantCategory, now},
	)
	if err != nil {
		return fmt.Errorf("upsert game variant: %w", err)
	}
	return nil
}
