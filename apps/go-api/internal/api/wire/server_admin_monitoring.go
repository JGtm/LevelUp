// Package api — server_admin_monitoring.go : montage des routes du dashboard
// monitoring admin. Extrait de server.go (déjà >1000 L) — server.go n'ajoute
// qu'un appel à MountAdminMonitoringRoutes dans le groupe /admin existant.
package wire

import (
	"context"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/observability/logging"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// MountAdminMonitoringRoutes monte les endpoints monitoring + actions sous le
// groupe /admin (déjà gardé RequireAuth + RequireAdmin). NoStore sur les GET :
// état courant, jamais de cache.
//
//   - sched peut être nil (watcher/scheduler désactivés) : sections dégradées.
//   - serverCtx : parent des goroutines de jobs (annulé au shutdown, avant
//     duckdb.CloseAll) — pattern syncH.WithServerContext.
//
// Le HealthScheduler n'est PAS passé ici : il est créé par main.go APRÈS
// NewRouter et attaché via reg.WithHealthScheduler (lecture lazy par les
// runners à chaque requête — ordre de boot sûr).
func MountAdminMonitoringRoutes(
	r chi.Router,
	reg *ServiceRegistry,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs_platform.Store,
	serverCtx context.Context,
	apiOpt humacore.MountOption,
) {
	reg.WithAutoSyncScheduler(sched)

	monitoringH := handlers.NewAdminMonitoringHandler(
		reg.MonitoringOverview, reg.ConvergenceReport, reg.PerfStats, reg.ErrorStats,
		reg.DetectionsReport, reg.SetDetectionStatus, reg.FreshnessReport, reg.ResourcesReport, reg.CronsReport, sched, jobStore).
		// File de construction + ouvriers : l'état vit côté web, donc le dashboard
		// voit le travail distant sans jamais interroger l'ouvrier (piste F §4bis).
		WithBuildQueue(reg.BuildQueueReport)
	// 6 GET /monitoring/* migrés vers Huma (Phase 3b), NoStore.
	monitoringH.Mount(r.With(middleware.NoStore), apiOpt)

	// Couverture de résolution d'arme (registre vs weapon_labels vs non résolu),
	// par titre. GET /monitoring/weapon-coverage?title=.
	coverageH := handlers.NewAdminWeaponCoverageHandler(reg.WeaponCoverage)
	coverageH.Mount(r.With(middleware.NoStore), apiOpt)

	// Trous d'intérieur LUSR (garde-fou notes LUSR) + action replay par joueur.
	// GET /monitoring/lusr-gaps?title= ; POST /monitoring/lusr-gaps/{player}/recompute.
	lusrH := handlers.NewAdminLUSRGapsHandler(reg.LUSRGapsReport, reg.RecomputeLUSRGapsForPlayer)
	lusrH.Mount(r.With(middleware.NoStore), apiOpt)

	actionsH := handlers.NewAdminActionsHandler(
		reg.RunDataHealthNow, reg.ActionJournalReport, sched, jobStore, serverCtx)
	actionsH.Mount(r, apiOpt) // POST /actions/data-health/run, /actions/auto-sync/run ; GET /actions/journal

	// Qualité données : compteurs/listes d'inconnus + actions de résolution
	// (backfill registry names, traductions metadata) — Phase 2.
	// Mount gère le Cache-Control:no-store des 2 GET en interne (champ header).
	dqH := handlers.NewAdminDataQualityHandler(
		reg.DataQualityCounts, reg.DataQualityIssues,
		reg.RunRegistryNamesBackfill, reg.ResolveModeTranslation, reg.ResolveAssetTranslation,
		reg.RunCatalogRefresh, reg.RunLyingBitsReset,
		ErrActionBusy)
	dqH.Mount(r, apiOpt)

	// Convergence ciblée d'un joueur (job asynchrone, claim SyncGate).
	convH := handlers.NewAdminConvergenceActionHandler(
		reg.RunPlayerConvergence, jobStore, serverCtx, ErrSyncInFlight)
	convH.Mount(r, apiOpt) // POST /actions/convergence/run

	// Synchronisation initiale (re-import complet, RunFull) d'un joueur au choix
	// (job asynchrone, claim SyncGate, tokens via le pool unifié).
	initialSyncH := handlers.NewAdminInitialSyncActionHandler(
		reg.RunPlayerInitialSync, jobStore, serverCtx, ErrSyncInFlight)
	initialSyncH.Mount(r, apiOpt) // POST /actions/initial-sync/run

	// Drain DiscoveryUGC (job asynchrone — réseau, rate-limité). Complète le
	// catalog/refresh zéro-réseau en hydratant les assets absents de match_registry.
	drainH := handlers.NewAdminCatalogDrainHandler(reg.RunCatalogUGCDrain, jobStore, serverCtx)
	drainH.Mount(r, apiOpt) // POST /actions/catalog/ugc-drain

	// Construction du rejeu 2D d'un match (job asynchrone — décodage hors ligne du film
	// en cache via la librairie replaybuild, sérialisé par le verrou process filmdec).
	replayBuildH := handlers.NewAdminReplayBuildActionHandler(reg.RunReplayBuild, jobStore, serverCtx).
		WithEnqueuer(reg.EnqueueReplayBuild)
	replayBuildH.Mount(r, apiOpt) // POST /actions/replay-build/{run,enqueue}

	// Viewer de logs : modules + tail filtré (lecture par la fin chunkée).
	logsH := handlers.NewAdminLogsHandler(logging.LoadConfig(reg.cfg.RepoRoot).LogsDir)
	logsH.Mount(r.With(middleware.NoStore), apiOpt) // GET /monitoring/logs/{modules,tail}
}
