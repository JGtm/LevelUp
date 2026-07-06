// Package service — catalog_fetcher_service.go : Phase F du plan PLAN_PLAYLISTS_CATALOG.md.
//
// CatalogFetcherService draine la queue catalog_fetch_queue en appelant
// TitleCatalogAdapter pour chaque entrée, et persiste les résultats via le
// port.CatalogWriter (K1j : plus de *sql.DB brut ni de SQL ici — ADR 0025 D-MV2 ;
// la persistance ART-safe vit dans platform/duckdb.CatalogRepo).
//
// Pas de worker auto : exposé via CLI (Phase G) ou refresh mensuel (Phase J).

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/port"
)

// CatalogFetcherService draine la queue catalog_fetch_queue.
type CatalogFetcherService struct {
	repo     port.CatalogWriter
	resolver games.Resolver
}

// NewCatalogFetcherService construit le service avec le writer catalogue (persistance
// D-MV2 côté platform/duckdb, K1j) + le resolver d'adapters.
func NewCatalogFetcherService(repo port.CatalogWriter, resolver games.Resolver) *CatalogFetcherService {
	return &CatalogFetcherService{repo: repo, resolver: resolver}
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
// présentes dans les tables catalogue, et upsert le résultat via le repo.
//
// APPEND-ONLY (fix ART 2026-06-19) : la file n'est JAMAIS mutée (ni DELETE ni UPDATE) —
// un DELETE/UPDATE sur cette table ART-indexée FATAL-invaliderait metadata.duckdb. « Reste
// à résoudre » = entrée absente de la table catalogue de son type (repo.SelectPending) :
// un asset résolu en sort naturellement ; un asset non résolvable (404 Discovery) y reste
// et sera re-tenté (rate-limité). Les nouveaux IDs découverts sont ré-enqueués par le repo
// (INSERT OR IGNORE — insert pur, sûr).
func (s *CatalogFetcherService) Drain(ctx context.Context, titleSlug string) (DrainResult, error) {
	if s.repo == nil {
		return DrainResult{}, errors.New("CatalogFetcherService: repo nil")
	}
	if s.resolver == nil {
		return DrainResult{}, errors.New("CatalogFetcherService: resolver nil")
	}

	adapter, err := s.resolver.Catalog(titleSlug)
	if err != nil {
		return DrainResult{}, fmt.Errorf("resolve adapter: %w", err)
	}

	entries, err := s.repo.SelectPending(ctx, titleSlug)
	if err != nil {
		return DrainResult{}, fmt.Errorf("select queue: %w", err)
	}

	var res DrainResult
	for _, e := range entries {
		if err := s.processEntry(ctx, adapter, titleSlug, e.AssetType, e.AssetID, e.VersionID); err != nil {
			// Append-only : pas de markError (UPDATE interdit sur table ART-indexée).
			// L'entrée reste "pending" (absente du catalogue) → re-tentée au prochain drain.
			slog.WarnContext(ctx, "catalog drain: process entry échoué (sera re-tenté)", "err", err,
				"title_slug", titleSlug, "asset_type", e.AssetType, "asset_id", e.AssetID)
			res.Errors++
			continue
		}
		// Append-only : pas de deleteFromQueue (DELETE interdit). L'entrée sort du
		// périmètre "pending" car elle est désormais présente dans la table catalogue.
		switch e.AssetType {
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

// processEntry hydrate une entrée queue via l'adapter et upsert via le repo.
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
		return s.repo.UpsertPair(ctx, titleSlug, pair)
	case games.AssetKindMap:
		m, err := adapter.FetchMap(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.repo.UpsertMap(ctx, titleSlug, m)
	case games.AssetKindGameVariant:
		gv, err := adapter.FetchGameVariant(ctx, assetID, versionID)
		if err != nil {
			return err
		}
		return s.repo.UpsertGameVariant(ctx, titleSlug, gv)
	}
	return fmt.Errorf("asset_type inconnu: %q", assetType)
}

// upsertPlaylist applique la POLICY is_ranked (la référence rankedplaylists est la source
// de vérité : si DiscoveryUGC classe à tort une playlist classée en 'social', on rétablit
// ici plutôt que de propager une valeur fausse) puis délègue la persistance au repo.
func (s *CatalogFetcherService) upsertPlaylist(ctx context.Context, titleSlug string, pl canonical.CanonicalPlaylist) error {
	isRanked := pl.IsRanked
	experience := string(pl.Experience)
	if rankedplaylists.IsRanked(pl.AssetID) {
		isRanked = true
		experience = "ranked"
	}
	return s.repo.UpsertPlaylist(ctx, titleSlug, pl, isRanked, experience)
}
