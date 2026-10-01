// background_tasks.go — lancement des tâches de fond du serveur, et leur sort en mode démo
// (backlog 2026-09-26, lot B5.3 ; décision D-7 : « mode démo hermétique côté fichiers »).
//
// Extrait de main() pour être TESTABLE (cf. background_tasks_test.go) : chaque tâche DÉCLARE
// son statut démo. Une tâche « coupée » n'est pas lancée en démo : elle écrit, supprime ou lit
// comme état des fichiers hors de la racine démo (le harnais visuel lance la démo sur le VRAI
// checkout). Une tâche « gardée » ne touche que la racine démo, la mémoire, ou rien.
// Inventaire de référence : tableau B5.0 du journal de .ai/PLAN_BACKLOG_2026-09-26.md.
//
// Hors démo, rien ne change : toutes les tâches sont lancées, dans le même ordre, sur les
// mêmes contextes et groupes d'attente qu'avant l'extraction.
package main

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/friendstore"
	"levelup/go-api/internal/platform/groupstore"
	syncpkg "levelup/go-api/internal/sync"
)

// demoPolicy : sort d'une tâche de fond en mode démo.
type demoPolicy int

const (
	// demoKeep : la tâche tourne en démo (elle n'écrit que sous la racine démo, en mémoire,
	// ou nulle part).
	demoKeep demoPolicy = iota
	// demoCut : la tâche est coupée en démo.
	demoCut
)

// Noms des tâches de fond et des étapes de boot (log de boot, tests).
const (
	taskJanitor        = "janitor (sync_cache, WAL)"
	taskWALRecovery    = "recovery WAL périodique"
	taskSocialCkpt     = "checkpoint shared_social"
	taskPoolRescan     = "re-scan du pool de tokens"
	taskAutoSync       = "scheduler d'auto-sync"
	taskDataHealth     = "santé données"
	taskDetectionFlush = "flush des détections (monitoring)"
	taskDiskWatch      = "surveillance disque"
	taskReplayNotify   = "notification des rejeux prêts"
	taskCatalogCron    = "cron catalogue"
	taskAssetNameSweep = "balayage des noms d'assets"
	taskSpartanCron    = "cron Spartan"
	taskWorldLeaderbd  = "cron classement mondial"
	taskReplayPurge    = "purge des rejeux"
	taskAppRelease     = "notification de version (app_release)"
	taskHeartbeat      = "heartbeat"
	stepFriendsMigr    = "migration des amis par joueur"
	stepGroupMigr      = "migration du groupe par défaut"
	stepTokenPool      = "pool de tokens (auto-sync)"
	stepPersistQueue   = "file persist asynchrone (WAL, RecoverPending)"
	stepWatcher        = "watcher de présence"
	monitoringInMemory = "monitoring.duckdb (magasin en mémoire)"
	heartbeatPeriod    = 30 * time.Second
	janitorPeriod      = 24 * time.Hour
	janitorMaxAge      = 7 * 24 * time.Hour
	walRecoveryPeriod  = 10 * time.Minute
	socialCkptPeriod   = 5 * time.Minute
	socialCkptTimeout  = 10 * time.Second
	poolRescanPeriod   = 15 * time.Minute
)

// demoPolicies : LA déclaration du statut démo de chaque tâche de fond et de chaque étape de
// boot (tableau B5.0 du plan). Une tâche absente de la table est traitée comme coupée en
// démo (défaut sûr) — background_tasks_test.go exige que chaque nom y figure.
var demoPolicies = map[string]demoPolicy{
	taskJanitor:        demoCut,
	taskWALRecovery:    demoCut,
	taskSocialCkpt:     demoKeep,
	taskPoolRescan:     demoCut,
	taskAutoSync:       demoCut,
	taskDataHealth:     demoCut,
	taskDetectionFlush: demoKeep,
	taskDiskWatch:      demoCut,
	taskReplayNotify:   demoCut,
	taskCatalogCron:    demoCut,
	taskAssetNameSweep: demoCut,
	taskSpartanCron:    demoCut,
	taskWorldLeaderbd:  demoCut,
	taskReplayPurge:    demoCut,
	taskAppRelease:     demoKeep,
	taskHeartbeat:      demoKeep,
	stepFriendsMigr:    demoCut,
	stepGroupMigr:      demoCut,
	stepTokenPool:      demoCut,
	stepPersistQueue:   demoCut,
	stepWatcher:        demoCut,
}

// policyOf rend le statut démo déclaré d'une tâche ; inconnue → coupée (défaut sûr).
func policyOf(name string) demoPolicy {
	if p, ok := demoPolicies[name]; ok {
		return p
	}
	return demoCut
}

// backgroundTask : une tâche de fond lancée au boot. Son statut démo se lit dans
// demoPolicies, par son nom.
type backgroundTask struct {
	name string
	// tracked : ajoutée au groupe d'attente du scheduler (drainé avant duckdb.CloseAll).
	tracked bool
	run     func()
}

// bootTasks lance les tâches de fond et les étapes de boot, et tient la liste de ce qui est
// coupé en démo (un seul log Info au boot, cf. logCut).
type bootTasks struct {
	demo bool
	wg   *sync.WaitGroup
	cut  []string
}

func newBootTasks(demo bool, wg *sync.WaitGroup) *bootTasks {
	return &bootTasks{demo: demo, wg: wg}
}

// launch lance la tâche dans sa goroutine et rend true si elle est lancée (usage :
// `if b.launch(t) { slog.Info("… scheduled") }`). En démo, une tâche coupée n'est PAS
// lancée : elle est ajoutée à la liste du log de boot.
func (b *bootTasks) launch(t backgroundTask) bool {
	if b.cutInDemo(t.name) {
		return false
	}
	if t.tracked {
		b.wg.Add(1)
	}
	go func() {
		if t.tracked {
			defer b.wg.Done()
		}
		t.run()
	}()
	return true
}

// cutInDemo rend true si la tâche ou l'étape `name` est coupée : en démo ET déclarée coupée
// dans demoPolicies. Hors démo, toujours false (rien ne change). Une étape coupée est
// ajoutée à la liste du log de boot.
func (b *bootTasks) cutInDemo(name string) bool {
	if !b.demo || policyOf(name) != demoCut {
		return false
	}
	b.cut = append(b.cut, name)
	return true
}

// logCut émet, en démo, LE log Info de boot qui liste ce qui est coupé.
func (b *bootTasks) logCut(ctx context.Context) {
	if !b.demo {
		return
	}
	slog.InfoContext(ctx, "demo_mode: tâches de fond coupées (écritures et lectures hors de la racine démo)",
		"coupees", strings.Join(b.cut, " ; "), "en_memoire", monitoringInMemory)
}

// loopTask : une boucle de fond déjà construite (cron, scheduler, registre).
func loopTask(name string, tracked bool, ctx context.Context, run func(context.Context)) backgroundTask {
	return backgroundTask{name: name, tracked: tracked, run: func() { run(ctx) }}
}

// janitorTask : purge 1× / 24 h `data/sync_cache` > 7 jours et les WAL ACKés > 7 jours,
// après avoir retenté les WAL pending (PLAN_PERSIST_ROBUSTNESS Phase 1). Premier passage
// immédiat. Coupée en démo : elle purge le vrai dépôt.
func janitorTask(ctx context.Context, pr *title.PathResolver, queue *persist.BatchQueue) backgroundTask {
	runJanitor := func() {
		if n, err := syncpkg.PurgeOldFetchCache(pr.SyncCacheDir(), janitorMaxAge); err != nil {
			slog.WarnContext(ctx, "janitor: PurgeOldFetchCache échoué (non-bloquant)", "err", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "janitor: fetch_cache purgé", "dirs_removed", n)
		}
		if queue == nil {
			return
		}
		// Garde-fou anti-perte : re-tenter les WAL pending AVANT de purger, sinon on
		// pourrait effacer un batch qu'on aurait pu rejouer.
		if rerr := queue.RecoverPending(); rerr != nil {
			slog.WarnContext(ctx, "janitor: RecoverPending échoué (non-bloquant)",
				"module", logging.ModulePersist, "err", rerr)
		}
		if n, err := queue.PurgeOldWAL(janitorMaxAge); err != nil {
			slog.WarnContext(ctx, "janitor: PurgeOldWAL échoué (non-bloquant)",
				"module", logging.ModulePersist, "err", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "janitor: WAL purgé", "module", logging.ModulePersist, "files_removed", n)
		}
	}
	return backgroundTask{name: taskJanitor, run: func() {
		ticker := time.NewTicker(janitorPeriod)
		defer ticker.Stop()
		runJanitor() // passage au boot (données périmées d'un run précédent)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runJanitor()
			}
		}
	}}
}

// walRecoveryTask re-soumet toutes les 10 min les WAL pending (un batch échoué restait
// bloqué jusqu'au reboot). Le dédup inFlight de la file évite de re-pousser un batch en vol.
func walRecoveryTask(ctx context.Context, queue *persist.BatchQueue) backgroundTask {
	return backgroundTask{name: taskWALRecovery, run: func() {
		recTicker := time.NewTicker(walRecoveryPeriod)
		defer recTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-recTicker.C:
				if rerr := queue.RecoverPending(); rerr != nil {
					slog.WarnContext(ctx, "persist: recovery périodique échouée (non-bloquant)",
						"module", logging.ModulePersist, "err", rerr)
				}
			}
		}
	}}
}

// socialCheckpointTask vide le WAL de shared_social toutes les 5 min (ADR 0022) sans ouvrir
// de connexion propre : il réutilise le handle du pool process-wide (LookupCachedDB) et
// saute le tick tant que la base n'est pas ouverte. Gardée en démo : la base est celle de la
// fixture.
func socialCheckpointTask(ctx context.Context, sharedSocialPath string) backgroundTask {
	return backgroundTask{name: taskSocialCkpt, run: func() {
		ckptTicker := time.NewTicker(socialCkptPeriod)
		defer ckptTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ckptTicker.C:
				socialDB, ok := duckdb.LookupCachedDB(sharedSocialPath)
				if !ok {
					continue // DB pas encore ouverte, skip
				}
				ckptStart := time.Now()
				ckptCtx, ckptCancel := context.WithTimeout(context.Background(), socialCkptTimeout)
				if _, err := socialDB.SQLDb().ExecContext(ckptCtx, "CHECKPOINT"); err != nil {
					slog.WarnContext(ckptCtx, "shared_social: periodic checkpoint failed", "err", err)
				} else {
					slog.DebugContext(ckptCtx, "shared_social: periodic checkpoint",
						"duration_ms", time.Since(ckptStart).Milliseconds())
				}
				ckptCancel()
			}
		}
	}}
}

// poolRescanTask re-scanne toutes les 15 min le MultiUserTokenStore pour ajouter à chaud les
// nouveaux tokens (SSO, token-capture) sans reboot.
func poolRescanTask(ctx context.Context, cfg *config.AppConfig, p pool.Pool) backgroundTask {
	runRescan := func() {
		resolver := title.NewPathResolver(cfg.RepoRoot)
		multiUserStore := auth.NewMultiUserTokenStore(cfg.WatcherTokensDir())
		discovery := pool.NewDiscoveryWithStore(cfg, resolver, title.DefaultSlug, multiUserStore)
		sources, err := discovery.Scan(ctx)
		if err != nil {
			slog.WarnContext(ctx, "pool: re-scan échoué (non-bloquant)", "err", err)
			return
		}
		added := 0
		for _, src := range sources {
			if err := p.AddOrUpdateSource(ctx, src); err != nil {
				slog.DebugContext(ctx, "pool: re-scan AddOrUpdateSource skip",
					"gamertag", src.Gamertag, "err", err)
				continue
			}
			added++
		}
		if added > 0 {
			slog.InfoContext(ctx, "pool: re-scan terminé",
				"sources_scanned", len(sources), "sources_processed", added, "pool_size", p.Size())
		}
	}
	return backgroundTask{name: taskPoolRescan, run: func() {
		rescanTicker := time.NewTicker(poolRescanPeriod)
		defer rescanTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-rescanTicker.C:
				runRescan()
			}
		}
	}}
}

// heartbeatTask : sentinelle de vie du process toutes les 30 s (incident 2026-05-22 :
// silence total sans trace). Log seul — gardée en démo.
func heartbeatTask(ctx context.Context, startedAt time.Time) backgroundTask {
	return backgroundTask{name: taskHeartbeat, run: func() {
		ticker := time.NewTicker(heartbeatPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				slog.InfoContext(ctx, "heartbeat: alive",
					"uptime_s", int(time.Since(startedAt).Seconds()),
					"goroutines", runtime.NumGoroutine(),
				)
			}
		}
	}}
}

// runBootMigrations : migrations de boot des amis par joueur puis du groupe par défaut
// (ordre imposé : la seconde lit le store de la première). Coupées en démo : ce sont des
// migrations de données d'utilisateurs réels, sans objet pour la fixture.
func runBootMigrations(ctx context.Context, b *bootTasks, cfg *config.AppConfig,
	fs *friendstore.FriendStore, gs *groupstore.GroupStore) {
	if !b.cutInDemo(stepFriendsMigr) {
		migratePlayerFriendsAtBoot(ctx, cfg, fs)
	}
	if !b.cutInDemo(stepGroupMigr) {
		migrateDefaultGroupAtBoot(ctx, cfg, fs, gs)
	}
}

// openMonitoringStore ouvre le magasin monitoring : la base globale
// `data/global/monitoring.duckdb` hors démo, une base EN MÉMOIRE en démo (décision D-7 :
// la démo ne doit pas ouvrir en écriture la base de monitoring réelle).
func openMonitoringStore(ctx context.Context, cfg *config.AppConfig, pr *title.PathResolver) (*ops.MonitoringStore, error) {
	if cfg.DemoMode {
		return ops.NewMonitoringStore(ctx, ops.MonitoringStoreInMemory)
	}
	return ops.NewMonitoringStore(ctx, pr.GlobalMonitoringDB())
}

// bootFriendStore rend le store des amis par joueur (data/global/player_friends.json) : sous
// `<démo>/runtime/` en démo (lot B5.5), à sa place d'avant sinon.
func bootFriendStore(cfg *config.AppConfig) *friendstore.FriendStore {
	return friendstore.NewFriendStore(cfg.RuntimePaths().PlayerFriendsPath())
}
