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
	maxRetries int
}

// NewCatalogFetcherService construit le service avec la DB metadata + resolver.
func NewCatalogFetcherService(metadataDB *sql.DB, resolver games.Resolver, maxRetries int) *CatalogFetcherService {
	if maxRetries <= 0 {
		maxRetries = 5
	}
	return &CatalogFetcherService{
		metadataDB: metadataDB,
		resolver:   resolver,
		maxRetries: maxRetries,
	}
}

// DrainResult agrège les compteurs après un drain.
type DrainResult struct {
	Playlists    int
	Pairs        int
	Maps         int
	GameVariants int
	Errors       int
	Skipped      int // attempts >= maxRetries
}

// Drain retire les entrées de catalog_fetch_queue, les hydrate via les adapters,
// upsert dans les tables catalogue, et supprime la ligne sur succès.
//
// Les entrées sont traitées par titleSlug (1 SELECT par titre). Les nouveaux
// asset IDs découverts (pairs/maps/variants référencés par une playlist) sont
// ré-enqueués pour drain ultérieur.
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

	rows, err := s.metadataDB.QueryContext(ctx,
		`SELECT asset_type, asset_id, COALESCE(version_id, ''), attempts
		 FROM catalog_fetch_queue
		 WHERE title_slug = ? AND attempts < ?
		 ORDER BY attempts ASC, enqueued_at ASC`,
		titleSlug, s.maxRetries,
	)
	if err != nil {
		return DrainResult{}, fmt.Errorf("select queue: %w", err)
	}
	type queueEntry struct {
		assetType, assetID, versionID string
		attempts                      int
	}
	var entries []queueEntry
	for rows.Next() {
		var e queueEntry
		if err := rows.Scan(&e.assetType, &e.assetID, &e.versionID, &e.attempts); err != nil {
			rows.Close()
			return DrainResult{}, fmt.Errorf("scan queue: %w", err)
		}
		entries = append(entries, e)
	}
	rows.Close()

	var res DrainResult
	for _, e := range entries {
		if err := s.processEntry(ctx, adapter, titleSlug, e.assetType, e.assetID, e.versionID); err != nil {
			s.markError(ctx, titleSlug, e.assetType, e.assetID, err)
			res.Errors++
			continue
		}
		s.deleteFromQueue(ctx, titleSlug, e.assetType, e.assetID)
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
	_, err := s.metadataDB.ExecContext(ctx, `
		INSERT INTO playlists_catalog
		  (title_slug, playlist_asset_id, current_version_id, name_canonical, experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?)
		ON CONFLICT (title_slug, playlist_asset_id) DO UPDATE SET
		  current_version_id = excluded.current_version_id,
		  name_canonical     = excluded.name_canonical,
		  experience         = excluded.experience,
		  is_ranked          = excluded.is_ranked,
		  last_seen_at       = excluded.last_seen_at,
		  last_fetched_at    = excluded.last_fetched_at
		`,
		titleSlug, pl.AssetID, pl.VersionID, pl.NameCanonical, experience, isRanked, now, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert playlist: %w", err)
	}
	// Re-enqueue les pairs liés s'ils ne sont pas dans le catalogue.
	for _, link := range pl.PairLinks {
		_, _ = s.metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO playlist_pair_links (title_slug, playlist_asset_id, pair_asset_id, weight)
			 VALUES (?, ?, ?, ?)`,
			titleSlug, pl.AssetID, link.PairAssetID, link.Weight,
		)
		_, _ = s.metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES (?, 'pair', ?)`,
			titleSlug, link.PairAssetID,
		)
	}
	return nil
}

func (s *CatalogFetcherService) upsertPair(ctx context.Context, titleSlug string, p canonical.CanonicalPair) error {
	now := time.Now().UTC()
	_, err := s.metadataDB.ExecContext(ctx, `
		INSERT INTO map_mode_pair_definitions
		  (title_slug, pair_asset_id, current_version_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category, last_fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_slug, pair_asset_id) DO UPDATE SET
		  current_version_id    = excluded.current_version_id,
		  name_canonical        = excluded.name_canonical,
		  map_asset_id          = excluded.map_asset_id,
		  game_variant_asset_id = excluded.game_variant_asset_id,
		  mode_category         = excluded.mode_category,
		  last_fetched_at       = excluded.last_fetched_at
		`,
		titleSlug, p.AssetID, p.VersionID, p.NameCanonical, p.MapAssetID, p.GameVariantAssetID, p.ModeCategory, now,
	)
	if err != nil {
		return fmt.Errorf("upsert pair: %w", err)
	}
	// Upsert les labels normalisés multi-langues.
	for lang, label := range p.ModeLabels {
		_, _ = s.metadataDB.ExecContext(ctx, `
			INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (title_slug, pair_asset_id, lang) DO UPDATE SET label = excluded.label
		`, titleSlug, p.AssetID, lang, label)
	}
	// Re-enqueue map et game_variant si inconnus.
	if p.MapAssetID != "" {
		_, _ = s.metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES (?, 'map', ?)`,
			titleSlug, p.MapAssetID,
		)
	}
	if p.GameVariantAssetID != "" {
		_, _ = s.metadataDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES (?, 'game_variant', ?)`,
			titleSlug, p.GameVariantAssetID,
		)
	}
	return nil
}

func (s *CatalogFetcherService) upsertMap(ctx context.Context, titleSlug string, m canonical.CanonicalMap) error {
	now := time.Now().UTC()
	_, err := s.metadataDB.ExecContext(ctx, `
		INSERT INTO maps_catalog (title_slug, map_asset_id, current_version_id, name_canonical, image_url, last_fetched_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_slug, map_asset_id) DO UPDATE SET
		  current_version_id = excluded.current_version_id,
		  name_canonical     = excluded.name_canonical,
		  image_url          = excluded.image_url,
		  last_fetched_at    = excluded.last_fetched_at
		`,
		titleSlug, m.AssetID, m.VersionID, m.NameCanonical, m.ImageURL, now,
	)
	if err != nil {
		return fmt.Errorf("upsert map: %w", err)
	}
	return nil
}

func (s *CatalogFetcherService) upsertGameVariant(ctx context.Context, titleSlug string, gv canonical.CanonicalGameVariant) error {
	now := time.Now().UTC()
	_, err := s.metadataDB.ExecContext(ctx, `
		INSERT INTO game_variants_catalog (title_slug, game_variant_asset_id, current_version_id, name_canonical, mode_canonical, game_variant_category, last_fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (title_slug, game_variant_asset_id) DO UPDATE SET
		  current_version_id    = excluded.current_version_id,
		  name_canonical        = excluded.name_canonical,
		  mode_canonical        = excluded.mode_canonical,
		  game_variant_category = excluded.game_variant_category,
		  last_fetched_at       = excluded.last_fetched_at
		`,
		titleSlug, gv.AssetID, gv.VersionID, gv.NameCanonical, string(gv.ModeCanonical), gv.GameVariantCategory, now,
	)
	if err != nil {
		return fmt.Errorf("upsert game variant: %w", err)
	}
	return nil
}

// markError incrémente attempts et stocke last_error pour retry ultérieur.
func (s *CatalogFetcherService) markError(ctx context.Context, titleSlug, assetType, assetID string, err error) {
	_, dbErr := s.metadataDB.ExecContext(ctx,
		`UPDATE catalog_fetch_queue SET attempts = attempts + 1, last_error = ?
		 WHERE title_slug = ? AND asset_type = ? AND asset_id = ?`,
		err.Error(), titleSlug, assetType, assetID,
	)
	if dbErr != nil {
		slog.WarnContext(ctx, "catalog drain: markError failed", "err", dbErr,
			"title_slug", titleSlug, "asset_type", assetType, "asset_id", assetID)
	}
}

// deleteFromQueue supprime une ligne après upsert réussi.
func (s *CatalogFetcherService) deleteFromQueue(ctx context.Context, titleSlug, assetType, assetID string) {
	_, err := s.metadataDB.ExecContext(ctx,
		`DELETE FROM catalog_fetch_queue WHERE title_slug = ? AND asset_type = ? AND asset_id = ?`,
		titleSlug, assetType, assetID,
	)
	if err != nil {
		slog.WarnContext(ctx, "catalog drain: delete failed", "err", err,
			"title_slug", titleSlug, "asset_type", assetType, "asset_id", assetID)
	}
}
