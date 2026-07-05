// Package service — catalog_fetcher_service.go : Phase F du plan PLAN_PLAYLISTS_CATALOG.md.
//
// CatalogFetcherService draine la queue catalog_fetch_queue en appelant
// TitleCatalogAdapter pour chaque entrée, et persiste les résultats dans les
// tables catalogue (playlists_catalog, maps_catalog, etc.).
//
// Pas de worker auto : exposé via CLI (Phase G) ou refresh mensuel (Phase J).

package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// CatalogFetcherService draine la queue catalog_fetch_queue.
type CatalogFetcherService struct {
	metadataDB *sql.DB
	resolver   games.Resolver
}

// NewCatalogFetcherService construit le service avec la DB metadata + resolver.
func NewCatalogFetcherService(metadataDB *sql.DB, resolver games.Resolver) *CatalogFetcherService {
	return &CatalogFetcherService{
		metadataDB: metadataDB,
		resolver:   resolver,
	}
}

// DrainResult agrège les compteurs après un drain.
type DrainResult struct {
	Playlists    int
	Pairs        int
	Maps         int
	GameVariants int
	Errors       int
}

// Drain hydrate via les adapters les entrées de catalog_fetch_queue PAS ENCORE
// présentes dans les tables catalogue, et upsert le résultat dans ces tables.
//
// APPEND-ONLY (fix ART 2026-06-19) : la file n'est JAMAIS mutée (ni DELETE ni
// UPDATE). Un DELETE/UPDATE sur cette table ART-indexée FATAL-invalide
// metadata.duckdb (même bug que les UPSERT, cf. upsertRowNoConflict ci-dessous).
// « Reste à résoudre » = entrée de la file absente de la table catalogue de son
// type (NOT EXISTS) : un asset résolu en sort naturellement ; un asset non
// résolvable (404 Discovery) y reste et sera re-tenté aux drains suivants
// (acceptable, rate-limité). Les nouveaux IDs découverts sont ré-enqueués via
// INSERT OR IGNORE (insert pur, sûr).
func (s *CatalogFetcherService) Drain(ctx context.Context, titleSlug string) (DrainResult, error) {
	if s.metadataDB == nil {
		return DrainResult{}, errors.New("CatalogFetcherService: metadataDB nil")
	}
	if s.resolver == nil {
		return DrainResult{}, errors.New("CatalogFetcherService: resolver nil")
	}

	adapter, err := s.resolver.Catalog(titleSlug)
	if err != nil {
		return DrainResult{}, fmt.Errorf("resolve adapter: %w", err)
	}

	// Pending = entrée de la file dont l'asset n'est PAS déjà dans la table
	// catalogue de son type (append-only : aucune mutation de la file).
	rows, err := s.metadataDB.QueryContext(ctx,
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
		return DrainResult{}, fmt.Errorf("select queue: %w", err)
	}
	type queueEntry struct{ assetType, assetID, versionID string }
	var entries []queueEntry
	for rows.Next() {
		var e queueEntry
		if err := rows.Scan(&e.assetType, &e.assetID, &e.versionID); err != nil {
			rows.Close()
			return DrainResult{}, fmt.Errorf("scan queue: %w", err)
		}
		entries = append(entries, e)
	}
	rows.Close()

	var res DrainResult
	for _, e := range entries {
		if err := s.processEntry(ctx, adapter, titleSlug, e.assetType, e.assetID, e.versionID); err != nil {
			// Append-only : pas de markError (UPDATE interdit sur table ART-indexée).
			// L'entrée reste "pending" (absente du catalogue) → re-tentée au prochain drain.
			slog.WarnContext(ctx, "catalog drain: process entry échoué (sera re-tenté)", "err", err,
				"title_slug", titleSlug, "asset_type", e.assetType, "asset_id", e.assetID)
			res.Errors++
			continue
		}
		// Append-only : pas de deleteFromQueue (DELETE interdit). L'entrée sort du
		// périmètre "pending" car elle est désormais présente dans la table catalogue.
		switch e.assetType {
		case games.AssetKindPlaylist:
			res.Playlists++
		case games.AssetKindPair:
			res.Pairs++
		case games.AssetKindMap:
			res.Maps++
		case games.AssetKindGameVariant:
			res.GameVariants++
		}
	}
	return res, nil
}

// processEntry hydrate une entrée queue via l'adapter et upsert dans les tables catalogue.
func (s *CatalogFetcherService) processEntry(ctx context.Context, adapter games.TitleCatalogAdapter,
	titleSlug, assetType, assetID, versionID string,
) error {
	switch assetType {
	case games.AssetKindPlaylist:
		pl, err := adapter.FetchPlaylist(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.upsertPlaylist(ctx, titleSlug, pl)
	case games.AssetKindPair:
		pair, err := adapter.FetchPair(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.upsertPair(ctx, titleSlug, pair)
	case games.AssetKindMap:
		m, err := adapter.FetchMap(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.upsertMap(ctx, titleSlug, m)
	case games.AssetKindGameVariant:
		gv, err := adapter.FetchGameVariant(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.upsertGameVariant(ctx, titleSlug, gv)
	}
	return fmt.Errorf("asset_type inconnu: %q", assetType)
}

// upsertRowNoConflict fait un SELECT d'existence puis UPDATE ou INSERT.
//
// Copie locale VOLONTAIRE de duckdb.UpsertRowNoConflict : la couche service ne peut
// pas importer internal/platform/duckdb (ADR 0025 D-MV2, verrou
// TestServicesDoNotImportDuckDB). Allowlistée par le garde-rail dédup #6
// archlint/no_local_upsert_helper_test.go ; toute AUTRE copie hors duckdb est interdite.
// À supprimer quand un CatalogRepository (port) remplacera le *sql.DB brut tenu ici.
//
// Évite délibérément `INSERT ... ON CONFLICT DO UPDATE` : sur metadata.duckdb
// ce pattern déclenche le bug ART DuckDB « Failed to delete all rows from index.
// Only deleted 0 out of 1 rows. » qui FATAL-invalide la connexion partagée pour
// TOUT le process — toutes les lectures metadata suivantes (mode_name_tr,
// asset_translations, map_images_registry, weapon_labels…) échouent alors
// silencieusement et l'UI retombe en anglais brut jusqu'au redémarrage.
// Cf. ADR 0019 + thought_log 2026-05-30 (incident filtres EN page Synthesis).
func (s *CatalogFetcherService) upsertRowNoConflict(
	ctx context.Context,
	existsQuery string, existsArgs []any,
	updateQuery string, updateArgs []any,
	insertQuery string, insertArgs []any,
) error {
	var dummy int
	err := s.metadataDB.QueryRowContext(ctx, existsQuery, existsArgs...).Scan(&dummy)
	switch {
	case err == nil:
		_, execErr := s.metadataDB.ExecContext(ctx, updateQuery, updateArgs...)
		return execErr
	case errors.Is(err, sql.ErrNoRows):
		_, execErr := s.metadataDB.ExecContext(ctx, insertQuery, insertArgs...)
		return execErr
	default:
		return err
	}
}

func (s *CatalogFetcherService) upsertPlaylist(ctx context.Context, titleSlug string, pl canonical.CanonicalPlaylist) error {
	now := time.Now().UTC()
	// Conformité allowlist : la référence rankedplaylists est la source de vérité
	// de is_ranked. Si DiscoveryUGC classe à tort une playlist classée en 'social'
	// (bug récurrent), on rétablit ici plutôt que de propager une valeur fausse.
	isRanked := pl.IsRanked
	experience := string(pl.Experience)
	if rankedplaylists.IsRanked(pl.AssetID) {
		isRanked = true
		experience = "ranked"
	}
	err := s.upsertRowNoConflict(ctx,
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
	// Re-enqueue les pairs liés s'ils ne sont pas dans le catalogue.
	for _, link := range pl.PairLinks {
		// ART-safe (metadata.duckdb, #23046) : playlist_pair_links GARDE sa PK
		// (title_slug, playlist_asset_id, pair_asset_id) → un INSERT OR IGNORE
		// sonderait/maintiendrait l'index ART de la PK au conflit (vecteur). SELECT-then-
		// INSERT (NOT EXISTS) : pas de résolution de conflit. Comportement OR IGNORE
		// préservé (pas d'update du weight si la ligne existe).
		_, _ = s.metadataDB.ExecContext(ctx,
			`INSERT INTO playlist_pair_links (title_slug, playlist_asset_id, pair_asset_id, weight)
			 SELECT ?, ?, ?, ?
			 WHERE NOT EXISTS (
			   SELECT 1 FROM playlist_pair_links
			   WHERE title_slug = ? AND playlist_asset_id = ? AND pair_asset_id = ?
			 )`,
			titleSlug, pl.AssetID, link.PairAssetID, link.Weight,
			titleSlug, pl.AssetID, link.PairAssetID,
		)
		s.enqueueCatalogChild(ctx, titleSlug, "pair", link.PairAssetID)
	}
	return nil
}

// enqueueCatalogChild ajoute un asset enfant à catalog_fetch_queue s'il n'y est pas
// déjà. ART-safe : INSERT … SELECT … WHERE NOT EXISTS (catalog_fetch_queue n'a plus de
// PK depuis rebuild_catalog_fetch_queue_drop_art_indexes → INSERT OR IGNORE ne
// dédupliquerait plus ET échouait silencieusement sur la colonne version_id NOT NULL).
// version_id=” : le drain le résoudra. Best-effort (erreur loggée en debug).
func (s *CatalogFetcherService) enqueueCatalogChild(ctx context.Context, titleSlug, assetType, assetID string) {
	if assetID == "" {
		return
	}
	if _, err := s.metadataDB.ExecContext(ctx,
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

func (s *CatalogFetcherService) upsertPair(ctx context.Context, titleSlug string, p canonical.CanonicalPair) error {
	now := time.Now().UTC()
	err := s.upsertRowNoConflict(ctx,
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
	// Upsert les labels normalisés multi-langues (SELECT-then-write, cf. upsertRowNoConflict).
	for lang, label := range p.ModeLabels {
		_ = s.upsertRowNoConflict(ctx,
			`SELECT 1 FROM pair_mode_label_translations WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
			[]any{titleSlug, p.AssetID, lang},
			`UPDATE pair_mode_label_translations SET label = ? WHERE title_slug = ? AND pair_asset_id = ? AND lang = ?`,
			[]any{label, titleSlug, p.AssetID, lang},
			`INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label) VALUES (?, ?, ?, ?)`,
			[]any{titleSlug, p.AssetID, lang, label},
		)
	}
	// Re-enqueue map et game_variant si inconnus (ART-safe via enqueueCatalogChild).
	s.enqueueCatalogChild(ctx, titleSlug, "map", p.MapAssetID)
	s.enqueueCatalogChild(ctx, titleSlug, "game_variant", p.GameVariantAssetID)
	return nil
}

func (s *CatalogFetcherService) upsertMap(ctx context.Context, titleSlug string, m canonical.CanonicalMap) error {
	now := time.Now().UTC()
	err := s.upsertRowNoConflict(ctx,
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

func (s *CatalogFetcherService) upsertGameVariant(ctx context.Context, titleSlug string, gv canonical.CanonicalGameVariant) error {
	now := time.Now().UTC()
	err := s.upsertRowNoConflict(ctx,
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
