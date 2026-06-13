// Package api — registry_asset_names_sweep.go : balayage périodique des noms
// d'assets restés en UUID dans match_registry (filet de convergence pour la
// traîne — assets dont la résolution a échoué/été capée au sync ET qui ne sont
// jamais rejoués, donc jamais re-tentés par la résolution in-sync).
//
// Lancé par le cron catalogue hebdomadaire (cmd/server). Réutilise la chaîne
// token-free GameCMS (halo.NewAssetNameFetcher) + ops.UpsertAssetTranslation
// (ART-safe) — même mécanique que la résolution in-sync, mais sourcée depuis
// match_registry sur TOUS les assets non résolus (pas seulement ceux du cycle).
package api

import (
	"context"
	"fmt"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/platform/halo"
	syncpkg "levelup/go-api/internal/sync"
)

// ResolveUnresolvedAssetNames balaye match_registry et résout les noms d'assets
// restés en UUID vers asset_translations. Self-gated par le kill-switch
// LEVELUP_SYNC_RESOLVE_ASSETS (fetcher nil → no-op). Handles : shared RO +
// metadata RW partagé (dataQualityHandles). Best-effort.
func (r *ServiceRegistry) ResolveUnresolvedAssetNames(ctx context.Context, titleSlug string) (assetnames.Result, error) {
	fetcher := halo.NewAssetNameFetcherIfEnabled()
	if fetcher == nil {
		return assetnames.Result{}, nil // kill-switch actif
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(titleSlug)
	if err != nil {
		return assetnames.Result{}, err
	}
	defer closeAll()
	if metaSQL == nil {
		return assetnames.Result{}, fmt.Errorf("metadata indisponible pour %s", titleSlug)
	}
	return syncpkg.ResolveUnresolvedAssetNames(ctx, fetcher, metaSQL, sharedSQL, titleSlug)
}
