// Package api — registry_asset_names_sweep.go : balayage périodique des noms
// d'assets restés en UUID dans match_registry (filet de convergence pour la
// traîne — assets dont la résolution a échoué/été capée au sync ET qui ne sont
// jamais rejoués, donc jamais re-tentés par la résolution in-sync).
//
// Lancé par le cron catalogue hebdomadaire (cmd/server) avec le POOL UNIFIÉ de
// tokens (la même source que tous les syncs — GameCMS exige un token Spartan).
// Écrit asset_translations via ops.UpsertAssetTranslation (ART-safe).
package wire

import (
	"context"
	"fmt"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/platform/auth/pool"
	syncpkg "levelup/go-api/internal/sync"
)

// ResolveUnresolvedAssetNames balaye match_registry et résout les noms d'assets
// restés en UUID vers asset_translations, via le pool unifié p. Le gate
// LEVELUP_SYNC_RESOLVE_ASSETS est appliqué côté sync. Handles : shared RO +
// metadata RW partagé (dataQualityHandles). p nil → no-op. Best-effort.
func (r *ServiceRegistry) ResolveUnresolvedAssetNames(ctx context.Context, titleSlug string, p pool.Pool) (assetnames.Result, error) {
	if p == nil {
		return assetnames.Result{}, nil
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return assetnames.Result{}, err
	}
	defer closeAll()
	if metaSQL == nil {
		return assetnames.Result{}, fmt.Errorf("metadata indisponible pour %s", titleSlug)
	}
	return syncpkg.ResolveUnresolvedAssetNames(ctx, p, metaSQL, sharedSQL, titleSlug), nil
}
