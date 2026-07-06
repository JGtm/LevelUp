// Package api — registry_actions.go : runners des actions correctives du
// dashboard monitoring (backfill registry names, traductions metadata,
// convergence joueur).
//
// Toutes in-process (JAMAIS de spawn de CLI — conflit DuckDB mono-process).
// Single-flight par action lourde : mutex TryLock → erreur "déjà en cours"
// (pattern hlsSweepMu). Écriture shared via SharedProvider.AcquireWriter
// (sérialisée dblease avec les syncs, timeout court → le front réessaie).
package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	gosync "sync"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	sync_pkg "levelup/go-api/internal/sync"
)

// ErrActionBusy : une exécution de la même action est déjà en vol (409).
var ErrActionBusy = errors.New("action déjà en cours")

// ErrSyncInFlight : le joueur est déjà en cours de sync via une autre source
// (watcher/HTTP/scheduler) — l'action de convergence cède (re-tenter après).
var ErrSyncInFlight = errors.New("sync déjà en vol pour ce joueur")

// registryNamesMu sérialise le backfill registry names (single-flight).
var registryNamesMu gosync.Mutex

// catalogRefreshMu sérialise le refresh catalog (single-flight).
var catalogRefreshMu gosync.Mutex

// lyingBitsResetMu sérialise le reset des bits menteurs (single-flight).
var lyingBitsResetMu gosync.Mutex

// acquireWriterTimeout borne l'attente du writer shared (un sync long tient
// le lease — on rend la main au front plutôt que de pendre la requête).
const acquireWriterTimeout = 60 * time.Second

// RunRegistryNamesBackfill résout les assets UUID bruts de match_registry via
// metadata.asset_translations (sync.BackfillRegistryNames, idempotent).
// dryRun → scan seul en RO, aucun UPDATE.
func (r *ServiceRegistry) RunRegistryNamesBackfill(ctx context.Context, titleSlug string, dryRun bool) (domain.RegistryNamesBackfillResult, error) {
	res := domain.RegistryNamesBackfillResult{DryRun: dryRun}
	if dryRun {
		counts, err := r.DataQualityCounts(ctx, titleSlug)
		if err != nil {
			return res, err
		}
		res.PlaylistsScanned = counts.RawUUIDPlaylists
		res.MapsScanned = counts.RawUUIDMaps
		res.PairsScanned = counts.RawUUIDPairs
		res.VariantsScanned = counts.RawUUIDVariants
		return res, nil
	}

	if !registryNamesMu.TryLock() {
		return res, ErrActionBusy
	}
	defer registryNamesMu.Unlock()

	if r.cfg.SharedProvider == nil {
		return res, fmt.Errorf("shared provider non câblé (mode legacy) — backfill indisponible")
	}

	acquireCtx, cancel := context.WithTimeout(ctx, acquireWriterTimeout)
	defer cancel()
	writer, err := r.cfg.SharedProvider.AcquireWriter(ctxkeys.WithDBWriterLabel(acquireCtx, "admin_registry_names"))
	if err != nil {
		return res, fmt.Errorf("acquisition writer shared (sync en cours ?): %w", err)
	}
	defer writer.Release()

	metaPath := titlePkg.NewPathResolver(r.cfg.RepoRoot).MetadataDBPath(titleSlug)
	if _, statErr := os.Stat(metaPath); statErr != nil {
		return res, fmt.Errorf("metadata absente pour %s: %w", titleSlug, statErr)
	}
	metaDB, err := duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return res, fmt.Errorf("open metadata: %w", err)
	}
	defer metaDB.Close() //nolint:errcheck // ref-count

	stats, err := sync_pkg.BackfillRegistryNames(ctx, writer.DB(), metaDB.SQLDb())
	if err != nil {
		return res, err
	}
	res.PlaylistsScanned, res.PlaylistsFixed = stats.PlaylistsScanned, stats.PlaylistsFixed
	res.MapsScanned, res.MapsFixed = stats.MapsScanned, stats.MapsFixed
	res.PairsScanned, res.PairsFixed = stats.PairsScanned, stats.PairsFixed
	res.VariantsScanned, res.VariantsFixed = stats.VariantsScanned, stats.VariantsFixed
	res.TotalFixed = stats.Total()

	monitoringLog.InfoContext(ctx, "admin_actions: backfill registry names terminé",
		"title", titleSlug, "total_fixed", res.TotalFixed)
	observability.IncCounter("admin_action_registry_names_total")
	return res, nil
}

// RunLyingBitsReset clear les bits backfill_completed menteurs de
// match_registry (events/weapons posés mais tables vides) + le flag
// events_loaded menteur. Débloque le heal events/weapons au prochain sync.
// dryRun → COUNT seul en RO (shared cache), aucun UPDATE.
func (r *ServiceRegistry) RunLyingBitsReset(ctx context.Context, titleSlug string, dryRun bool) (domain.LyingBitsResetResult, error) {
	res := domain.LyingBitsResetResult{DryRun: dryRun}

	if dryRun {
		sharedSQL, _, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
		if err != nil {
			return res, err
		}
		defer closeAll()
		opsRes, err := ops.ResetLyingBits(ctx, sharedSQL, true)
		if err != nil {
			return res, err
		}
		return lyingBitsToDomain(opsRes), nil
	}

	if !lyingBitsResetMu.TryLock() {
		return res, ErrActionBusy
	}
	defer lyingBitsResetMu.Unlock()

	if r.cfg.SharedProvider == nil {
		return res, fmt.Errorf("shared provider non câblé (mode legacy) — reset indisponible")
	}
	acquireCtx, cancel := context.WithTimeout(ctx, acquireWriterTimeout)
	defer cancel()
	writer, err := r.cfg.SharedProvider.AcquireWriter(ctxkeys.WithDBWriterLabel(acquireCtx, "admin_lying_bits_reset"))
	if err != nil {
		return res, fmt.Errorf("acquisition writer shared (sync en cours ?): %w", err)
	}
	defer writer.Release()

	opsRes, err := ops.ResetLyingBits(ctx, writer.DB(), false)
	if err != nil {
		return res, err
	}
	res = lyingBitsToDomain(opsRes)
	monitoringLog.InfoContext(ctx, "admin_actions: reset lying bits terminé",
		"title", titleSlug, "events", res.EventsBitsCleared,
		"weapons", res.WeaponsBitsCleared, "events_loaded", res.EventsLoadedCleared)
	observability.IncCounter("admin_action_lying_bits_reset_total")
	return res, nil
}

// lyingBitsToDomain mappe le résultat ops vers le DTO API.
func lyingBitsToDomain(o ops.LyingBitsResetResult) domain.LyingBitsResetResult {
	return domain.LyingBitsResetResult{
		DryRun:              o.DryRun,
		EventsBitsCleared:   o.EventsBitsCleared,
		WeaponsBitsCleared:  o.WeaponsBitsCleared,
		EventsLoadedCleared: o.EventsLoadedCleared,
		Total:               o.Total(),
	}
}

// ResolveModeTranslation upserte mode_name_tr[fr] pour un mode (clé
// normalisée via NormalizeModeLabel — idempotent si déjà normalisée).
func (r *ServiceRegistry) ResolveModeTranslation(ctx context.Context, titleSlug, modeEN, nameFR string) (domain.ResolveResult, error) {
	out := domain.ResolveResult{}
	normalized := analysis.NormalizeModeLabel(modeEN)
	if normalized == "" {
		return out, fmt.Errorf("mode_en vide après normalisation")
	}
	metaDB, closeMeta, err := r.metadataRWHandle(titleSlug)
	if err != nil {
		return out, err
	}
	defer closeMeta()

	action, err := ops.UpsertModeTranslation(ctx, metaDB.SQLDb(), normalized, "fr", nameFR)
	if err != nil {
		return out, err
	}
	out.Action = action
	out.ModeEN = normalized
	monitoringLog.InfoContext(ctx, "admin_actions: traduction mode écrite",
		"title", titleSlug, "mode_en", normalized, "action", action)
	observability.IncCounter("admin_action_mode_translation_total")
	return out, nil
}

// validAssetKinds : asset_type acceptés par translations/asset (alignés sur
// asset_translations + BackfillRegistryNames).
var validAssetKinds = map[string]struct{}{
	games.AssetKindPlaylist:    {},
	games.AssetKindMap:         {},
	games.AssetKindPair:        {},
	games.AssetKindGameVariant: {},
}

// ResolveAssetTranslation upserte asset_translations pour un asset (en-US
// et/ou fr-FR). C'est la résolution effective d'un asset UUID inconnu : la
// chaîne d'affichage et le backfill registry names lisent cette table.
func (r *ServiceRegistry) ResolveAssetTranslation(ctx context.Context, titleSlug string, req domain.AssetTranslationRequest) (domain.ResolveResult, error) {
	out := domain.ResolveResult{}
	if _, ok := validAssetKinds[req.AssetKind]; !ok {
		return out, fmt.Errorf("asset_kind invalide %q (playlist|map|pair|game_variant)", req.AssetKind)
	}
	if req.NameEN == "" && req.NameFR == "" {
		return out, fmt.Errorf("au moins un nom (name_en ou name_fr) est requis")
	}
	metaDB, closeMeta, err := r.metadataRWHandle(titleSlug)
	if err != nil {
		return out, err
	}
	defer closeMeta()

	type langName struct{ lang, name string }
	writes := []langName{}
	if req.NameEN != "" {
		writes = append(writes, langName{duckdb.LangCodeEN, req.NameEN})
	}
	if req.NameFR != "" {
		writes = append(writes, langName{duckdb.LangCodeFR, req.NameFR})
	}
	for _, w := range writes {
		action, err := ops.UpsertAssetTranslation(ctx, metaDB.SQLDb(), req.AssetKind, req.AssetID, w.lang, w.name)
		if err != nil {
			return out, err
		}
		out.Action = action
		out.Langs = append(out.Langs, w.lang)
	}
	monitoringLog.InfoContext(ctx, "admin_actions: traduction asset écrite",
		"title", titleSlug, "asset_kind", req.AssetKind, "asset_id", req.AssetID, "langs", out.Langs)
	observability.IncCounter("admin_action_asset_translation_total")
	return out, nil
}

// RunCatalogRefresh seed les tables catalog metadata depuis match_registry
// (zéro réseau — le drain DiscoveryUGC reste CLI-only). Résout les playlists
// « hors catalogue » de la page Qualité données.
func (r *ServiceRegistry) RunCatalogRefresh(ctx context.Context, titleSlug string) (domain.CatalogRefreshResult, error) {
	out := domain.CatalogRefreshResult{}
	if !catalogRefreshMu.TryLock() {
		return out, ErrActionBusy
	}
	defer catalogRefreshMu.Unlock()

	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return out, err
	}
	defer closeAll()
	if metaSQL == nil {
		return out, fmt.Errorf("metadata indisponible pour %s", titleSlug)
	}

	res, err := ops.CatalogRefreshFromRegistry(ctx, metaSQL, sharedSQL, titleSlug)
	if err != nil {
		return out, err
	}
	out.Playlists, out.Pairs, out.Maps, out.GameVariants = res.Playlists, res.Pairs, res.Maps, res.GameVariants
	monitoringLog.InfoContext(ctx, "admin_actions: catalog refresh terminé",
		"title", titleSlug, "playlists", out.Playlists, "pairs", out.Pairs,
		"maps", out.Maps, "game_variants", out.GameVariants)
	observability.IncCounter("admin_action_catalog_refresh_total")
	return out, nil
}

// metadataRWHandle ouvre metadata en RW partagé (ref-count, jamais de second
// sql.Open concurrent). closeMeta décrémente — à defer.
func (r *ServiceRegistry) metadataRWHandle(titleSlug string) (*duckdb.DB, func(), error) {
	metaPath := titlePkg.NewPathResolver(r.cfg.RepoRoot).MetadataDBPath(titleSlug)
	if _, err := os.Stat(metaPath); err != nil {
		return nil, nil, fmt.Errorf("metadata absente pour %s: %w", titleSlug, err)
	}
	metaDB, err := duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open metadata: %w", err)
	}
	return metaDB, func() { _ = metaDB.Close() }, nil //nolint:errcheck // ref-count
}

// RunPlayerConvergence relance un RunDelta complet pour un joueur (la
// convergence est intrinsèque au cycle : post-sync déclenché via
// hasConvergenceBacklog même à 0 insert). Claim du SyncGate AVANT le travail
// — ErrSyncInFlight si une autre source sync déjà ce joueur.
func (r *ServiceRegistry) RunPlayerConvergence(ctx context.Context, titleSlug, playerSlug string) (map[string]any, error) {
	if r.autoSyncScheduler == nil {
		return nil, fmt.Errorf("scheduler auto-sync non câblé")
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, err
	}
	var gamertag, xuid string
	for _, p := range players {
		if p.PlayerSlug == playerSlug || p.Gamertag == playerSlug {
			gamertag, xuid = p.Gamertag, p.XUID
			break
		}
	}
	if gamertag == "" || xuid == "" {
		return nil, fmt.Errorf("joueur inconnu %q pour %s", playerSlug, titleSlug)
	}

	release, ok := r.autoSyncScheduler.Gate().TryClaimT(titleSlug, gamertag)
	if !ok {
		return nil, ErrSyncInFlight
	}
	defer release()

	// MT-11 / PMT-3 : porter le titre dans le ctx pour que BuildEngine écrive
	// dans les DB du bon titre (le moteur lit ctxkeys.TitleSlug).
	ctx = ctxkeys.WithTitleSlug(ctx, titleSlug)
	engine := r.autoSyncScheduler.BuildEngine(ctx, gamertag, xuid)
	syncRes, err := engine.RunDelta(ctx, domain.DefaultSyncOptions())
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"gamertag":         gamertag,
		"matches_inserted": syncRes.MatchesInserted,
		"matches_skipped":  syncRes.MatchesSkipped,
		"status":           syncRes.Status(),
		"error_count":      len(syncRes.Errors),
	}
	if syncRes.PostSync != nil {
		result["post_sync"] = *syncRes.PostSync
	}
	monitoringLog.InfoContext(ctx, "admin_actions: convergence joueur terminée",
		"title", titleSlug, "gamertag", gamertag,
		"inserted", syncRes.MatchesInserted, "status", syncRes.Status())
	observability.IncCounter("admin_action_player_convergence_total")
	return result, nil
}
