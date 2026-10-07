// Package api — registry_asset_names_sweep.go : balayage périodique des noms
// d'assets restés en UUID dans match_registry (filet de convergence pour la
// traîne — assets dont la résolution a échoué au sync ET qui ne sont jamais
// rejoués, donc jamais re-tentés par la résolution in-sync), puis CONVERGENCE
// du registre vers les traductions obtenues.
//
// Lancé par le cron hebdomadaire de noms d'assets (cmd/server) avec le POOL
// UNIFIÉ de tokens (la même source que tous les syncs — l'API d'assets exige un
// token Spartan). Écrit asset_translations via ops.UpsertAssetTranslation
// (ART-safe), puis match_registry via sync.BackfillRegistryNames →
// persist.RegistryNamesPersister, sous le writer shared.
package wire

import (
	"context"
	"fmt"
	"os"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// ResolveUnresolvedAssetNames balaye match_registry et résout les noms d'assets
// restés en UUID vers asset_translations, via le pool unifié p, PUIS réinscrit
// dans match_registry les noms désormais connus (convergeRegistryNames) : sans
// cette seconde passe, une traduction obtenue après l'insertion d'un match ne
// remonte pas dans son registre. Le gate LEVELUP_SYNC_RESOLVE_ASSETS est
// appliqué côté sync. p nil → no-op. Best-effort : un échec de convergence est
// journalisé, le résultat du balayage reste rendu.
func (r *ServiceRegistry) ResolveUnresolvedAssetNames(ctx context.Context, titleSlug string, p pool.Pool) (assetnames.Result, error) {
	if p == nil {
		return assetnames.Result{}, nil
	}
	res, err := r.sweepAssetNames(ctx, titleSlug, p)
	if err != nil {
		return res, err
	}
	stats, cErr := r.convergeRegistryNames(ctx, titleSlug, "asset_name_sweep_converge")
	if cErr != nil {
		monitoringLog.WarnContext(ctx, "asset_name_sweep: convergence du registre non faite",
			"title", titleSlug, "err", cErr)
		return res, nil
	}
	monitoringLog.InfoContext(ctx, "asset_name_sweep: registre convergé",
		"title", titleSlug, "colonnes", stats.Total(), "paires_construites", stats.PairsConstructed,
		"errors", stats.Errors)
	return res, nil
}

// sweepAssetNames : la passe de résolution des traductions. Handles : shared RO + metadata RW
// partagé (dataQualityHandles), RELÂCHÉS avant la convergence qui prend le writer shared.
func (r *ServiceRegistry) sweepAssetNames(ctx context.Context, titleSlug string, p pool.Pool) (assetnames.Result, error) {
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

// convergeRegistryNames exécute sync.BackfillRegistryNames sous le writer shared du titre
// (SharedProvider, sérialisé avec les syncs). Refuse un titre dont la base partagée n'est pas
// celle que tient le provider : écrire ailleurs que dans SA base serait mêler deux titres.
// label nomme l'écrivain dans les journaux du lease.
func (r *ServiceRegistry) convergeRegistryNames(ctx context.Context, titleSlug, label string) (syncpkg.BackfillRegistryStats, error) {
	var stats syncpkg.BackfillRegistryStats
	if r.cfg.SharedProvider == nil {
		return stats, fmt.Errorf("shared provider non câblé (mode legacy) — convergence des noms indisponible")
	}
	pr := titlePkg.NewPathResolver(r.cfg.RepoRoot)
	if r.cfg.SharedProvider.Path() != pr.SharedDBPath(titleSlug) {
		return stats, fmt.Errorf("base partagée de %s non tenue par le provider — convergence des noms indisponible", titleSlug)
	}
	metaPath := pr.MetadataDBPath(titleSlug)
	if _, statErr := os.Stat(metaPath); statErr != nil {
		return stats, fmt.Errorf("metadata absente pour %s: %w", titleSlug, statErr)
	}

	acquireCtx, cancel := context.WithTimeout(ctx, acquireWriterTimeout)
	defer cancel()
	writer, err := r.cfg.SharedProvider.AcquireWriter(ctxkeys.WithDBWriterLabel(acquireCtx, label))
	if err != nil {
		return stats, fmt.Errorf("acquisition writer shared (sync en cours ?): %w", err)
	}
	defer writer.Release()

	metaDB, err := duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return stats, fmt.Errorf("open metadata: %w", err)
	}
	defer metaDB.Close() //nolint:errcheck // ref-count

	return syncpkg.BackfillRegistryNames(ctx, writer.DB(), metaDB.SQLDb(), syncpkg.RegistryNamesOptions{})
}
