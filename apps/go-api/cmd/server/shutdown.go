package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/coalesce"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/watcher"
)

// shutdownDeps réunit ce que la séquence d'arrêt arrête, vide ou ferme.
type shutdownDeps struct {
	srv              *http.Server
	cancelScheduler  context.CancelFunc
	watcherDaemon    *watcher.Daemon // nil : pas de watcher (démo)
	autoScheduler    *scheduler.AutoSyncScheduler
	schedulerWG      *sync.WaitGroup
	autoBatchQueue   *persist.BatchQueue // nil : persistance synchrone
	workerWG         *sync.WaitGroup
	sharedSocialPath string
	reg              *wire.ServiceRegistry // nil : routeur jamais monté
	closeShared      func() error
	metaDB           *duckdb.DB
	multiHandler     *logging.MultiModuleHandler // nil : journalisation simple
}

// runShutdown exécute la séquence d'arrêt du serveur, une fois l'arrêt demandé.
//
// Elle tient dans shutdownBudget, mesuré depuis son premier instant : chaque phase
// d'attente (shutdownPhases) reçoit la plus petite de sa borne et du budget restant, et une
// phase qui dépasse est nommée dans un Warn avant que la suivante ne commence. Les
// fermetures (closeStores) viennent toujours après, sur la réserve shutdownCloseReserve
// qu'aucune phase n'entame : duckdb.CloseAll est atteint même si toutes les phases ont
// débordé. docker-compose.yml laisse stop_grace_period au-delà du budget ; air (kill_delay
// de 5 s, ignoré sous Windows où il tue l'arbre de process) n'attend pas la séquence
// entière.
//
// Ordre tenu par shutdown_budget_test.go : les requêtes HTTP finissent avant les
// recalculs, qui finissent avant les écrivains, et les bases ne se ferment qu'après tous.
func runShutdown(d shutdownDeps) {
	fmt.Fprint(os.Stderr, "\n  [..] Arret en cours...")
	ctx := context.Background()
	clock := newShutdownClock(shutdownBudget, shutdownCloseReserve)

	d.cancelScheduler()
	clock.runAll(ctx, shutdownPhases(d))
	closeStores(ctx, d)

	slog.Info("shutdown_total_duration_ms", "ms", time.Since(clock.start).Milliseconds())

	// Fermer proprement les fichiers logs/{module}.log (flush + close des handles).
	// Idempotent.
	if d.multiHandler != nil {
		if err := d.multiHandler.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "fermeture logging multi-module: %v\n", err)
		}
	}
	fmt.Fprintln(os.Stderr, " terminé.")
}

// shutdownPhases liste les phases d'attente, dans l'ordre où elles s'exécutent.
func shutdownPhases(d shutdownDeps) []shutdownPhase {
	var phases []shutdownPhase
	if d.watcherDaemon != nil {
		phases = append(phases, phaseWatcher(d))
	}
	phases = append(phases, phaseHTTP(d), phaseRecomputes(), phaseScheduler(d))
	if d.watcherDaemon != nil {
		phases = append(phases, phaseGate(d))
	}
	if d.autoBatchQueue != nil {
		phases = append(phases, phaseBatchDrain(d), phaseBatchClose(d))
	}
	return append(phases, phaseWorker(d), phaseCheckpoint(d))
}

// phaseWatcher arrête le daemon du watcher : ses syncs en vol abandonnent et libèrent leur
// lease, ses claims du gate sont rendus avant la phase du gate.
func phaseWatcher(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"watcher", phaseWatcherCap, blocking(d.watcherDaemon.Stop)}
}

// phaseHTTP arrête le serveur HTTP : plus aucune requête n'est acceptée, celles en vol
// finissent.
func phaseHTTP(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"http", phaseHTTPCap, func(ctx context.Context) bool {
		if err := d.srv.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "shutdown", "err", err)
		}
		return ctx.Err() == nil
	}}
}

// phaseRecomputes refuse les recalculs coalescés, annule ceux en vol et les attend
// (drainCoalescedRecomputes). Après phaseHTTP : plus aucune requête n'en lance.
func phaseRecomputes() shutdownPhase {
	return shutdownPhase{"recomputes", coalesce.DrainWait, drainCoalescedRecomputes}
}

// phaseScheduler attend la goroutine du scheduler, pour que les connexions DuckDB qu'elle
// tient soient libérées avant CloseAll.
func phaseScheduler(d shutdownDeps) shutdownPhase {
	wait := blocking(d.schedulerWG.Wait)
	return shutdownPhase{"scheduler", phaseSchedulerCap, func(ctx context.Context) bool {
		if !wait(ctx) {
			return false
		}
		slog.DebugContext(ctx, "scheduler terminé")
		return true
	}}
}

// phaseGate attend que les syncs HTTP pris sous le gate cross-source finissent, avant
// duckdb.CloseAll. cancelScheduler a annulé schedulerCtx, dont les contextes des syncs HTTP
// dérivent : les RunDelta en cours abandonnent, libèrent leur lease et leur claim. Sans
// cette attente, un sync détaché pourrait écrire une player DB après CloseAll (handle
// orphelin, WAL #7659). Les claims du watcher sont déjà rendus (phaseWatcher), ceux de
// l'auto-sync par phaseScheduler.
//
// BeginShutdown AVANT WaitInFlight : il fige le gate pour qu'aucun TryClaim (donc
// gateWG.Add) ne survienne pendant l'attente — sinon data race WaitGroup et retour
// prématuré de Wait. Un sync HTTP arrivé après ce point est refusé (409) sans claim.
func phaseGate(d shutdownDeps) shutdownPhase {
	gate := d.autoScheduler.Gate()
	return shutdownPhase{"gate", phaseGateCap, func(ctx context.Context) bool {
		gate.BeginShutdown()
		if !blocking(gate.WaitInFlight)(ctx) {
			return false
		}
		slog.DebugContext(ctx, "gate: tous les claims cross-source libérés")
		return true
	}}
}

// phaseBatchDrain attend les persists en cours de la BatchQueue. Avant la fermeture des
// bases, sinon les workers écriraient sur des DBs fermées.
func phaseBatchDrain(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"persist.drain", phaseBatchDrainCap, func(ctx context.Context) bool {
		err := d.autoBatchQueue.Drain(ctx)
		switch {
		case err == nil:
			return true
		case ctx.Err() != nil:
			return false
		default:
			slog.WarnContext(ctx, "persist: BatchQueue.Drain échoué (non-bloquant)", "err", err)
			return true
		}
	}}
}

// phaseBatchClose ferme la BatchQueue : les Submit en vol finissent, le canal se ferme et
// le worker se termine (phaseWorker).
func phaseBatchClose(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"persist.close", phaseBatchCloseCap, blocking(func() {
		if err := d.autoBatchQueue.Close(); err != nil {
			slog.Warn("persist: BatchQueue.Close échoué (non-bloquant)", "err", err)
		}
	})}
}

// phaseWorker attend la terminaison du worker de persistance après la fermeture du canal.
func phaseWorker(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"persist.worker", phaseWorkerCap, blocking(d.workerWG.Wait)}
}

// phaseCheckpoint vide le WAL de shared_social.duckdb avant la fermeture des connexions :
// essentiel au hot-reload (bug DuckDB #7659 : le rejeu du WAL à la réouverture peut échouer
// s'il contient des opérations non checkpointées). Sans risque ici : les workers sont
// arrêtés (phaseWorker), aucune écriture concurrente.
func phaseCheckpoint(d shutdownDeps) shutdownPhase {
	return shutdownPhase{"checkpoint", phaseCheckpointCap, func(ctx context.Context) bool {
		db, ok := duckdb.LookupCachedDB(d.sharedSocialPath)
		if !ok {
			return true
		}
		if _, err := db.SQLDb().ExecContext(ctx, "CHECKPOINT"); err != nil {
			slog.WarnContext(ctx, "shared_social: CHECKPOINT final au shutdown échoué (non-fatal — WAL sera rejoué au prochain boot)",
				"err", err)
			return ctx.Err() == nil
		}
		slog.InfoContext(ctx, "shared_social: CHECKPOINT final au shutdown terminé")
		return true
	}}
}

// closeStores ferme les bases et les registres. Toujours exécuté, hors budget des phases :
// la fermeture ne se borne pas, elle vit sur shutdownCloseReserve.
func closeStores(ctx context.Context, d shutdownDeps) {
	duckdb.CloseAll()

	// Fermer le ServiceRegistry AVANT metaDB.Close() pour décrémenter le refCount de
	// metadata porté par PrestigeBundle. Sans cela, refCount=2 au metaDB.Close() →
	// décrément à 1 → handle Windows tenu jusqu'à la fin du process → verrou metadata au
	// prochain hot-reload (docs/INCIDENT_2026-05-21_metadata_duckdb_lock_air_hot_reload.md
	// §3.2).
	if d.reg != nil {
		d.reg.Close()
	}
	if err := d.closeShared(); err != nil {
		slog.WarnContext(ctx, "fermeture shared DB", "err", err)
	}
	if err := d.metaDB.Close(); err != nil {
		slog.WarnContext(ctx, "fermeture metadata DB", "err", err)
	}

	// Une fuite de refCount sur le cache openDBs laisse un HANDLE Windows tenu jusqu'à la
	// fin du process : verrou metadata.duckdb au prochain hot-reload.
	leaks := duckdb.DumpCachedLeaks()
	for k, refs := range leaks {
		slog.WarnContext(ctx, "shutdown_db_leak", "cache_key", k, "refCount", refs)
	}
	if len(leaks) > 0 {
		slog.WarnContext(ctx, "shutdown_db_leak aggregate", "leaks_count", len(leaks))
	}
}
