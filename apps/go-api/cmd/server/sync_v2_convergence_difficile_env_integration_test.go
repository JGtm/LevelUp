//go:build integration

// sync_v2_convergence_difficile_env_integration_test.go : extension du banc de convergence
// (sync_v2_convergence_env_integration_test.go) pour les cas difficiles du rattrapage de l'historique :
// arrêt du processus en plein cycle puis redémarrage sur les mêmes bases, deux joueurs suivis dans un
// même cycle, matchs introuvables mêlés aux matchs neufs.
//
// Réutilisé tel quel du banc de base : le faux client Halo (fakeHalo : liste d'un joueur et lectures de
// pages), les aides du faux réseau (reply, replyJSON, between), le corps de stats (convergenceMatchStats),
// le pool de jetons (fakeTokenPool), le recorder d'orchestrateur (recordingOrchestrator) et la lecture de
// cadence (convergenceEnv.waitCadence). Ce qui est NEUF, parce que le banc de base ne le permet pas :
//   - plusieurs joueurs et une liste par joueur (difficileMonde, un fakeHalo par joueur) ;
//   - le démarrage répété sur les mêmes bases (difficileEnv.start) : orchestrateur, ordonnanceur, file de
//     persistance et worker NEUFS à chaque démarrage, bases migrées à chaque fois comme au démarrage du serveur ;
//   - l'arrêt : orchestré par la séquence de production (runShutdown, shutdown.go) ou brutal (le journal
//     d'écriture de la file reste sur disque, comme à la mort du processus) ;
//   - la porte du worker de persistance (difficileGate), pour couper pendant l'écriture de la file.
//
// La file de persistance d'un démarrage ordinaire est celle de production (openPersistQueue, RecoverPending
// compris). Seul le démarrage « à porte » monte la file à la main, avec le même câblage que
// startPersistWorker (CombinedPersister) enveloppé par la porte.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/historyrepo"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	settingsplatform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
	syncpkg "levelup/go-api/internal/sync"
)

// ---------------------------------------------------------------------------------------------
// Le banc : racine, bases et démarrages successifs.
// ---------------------------------------------------------------------------------------------

type difficileEnv struct {
	t         *testing.T
	root      string
	pr        *titlePkg.PathResolver
	slug      string
	monde     *difficileMonde
	intervalH int

	settingsPath, profilesPath string

	boots int
}

// newDifficileEnv prépare la racine temporaire (rien sous data/ du dépôt), les réglages, le faux réseau et
// les bases des joueurs `all` (créées à l'intégration puis migrées) ; `active` est la liste des profils
// suivis au départ (setActive la change entre deux cycles).
func newDifficileEnv(t *testing.T, monde *difficileMonde, all, active []difficileJoueur) *difficileEnv {
	t.Helper()
	duckdbpkg.CloseAll()
	t.Cleanup(duckdbpkg.CloseAll)

	root := t.TempDir()
	e := &difficileEnv{
		t: t, root: root, pr: titlePkg.NewPathResolver(root), slug: titlePkg.DefaultSlug, monde: monde, intervalH: 6,
		settingsPath: filepath.Join(root, "app_settings.json"), profilesPath: filepath.Join(root, "db_profiles.json"),
	}
	writeTestFile(t, e.settingsPath, fmt.Sprintf(`{"spnkr_auto_sync_enabled": true, "spnkr_auto_sync_interval_hours": %d}`, e.intervalH))
	e.setActive(active...)

	if err := ensureWarehouseDir(e.pr, e.slug); err != nil {
		t.Fatalf("ensureWarehouseDir: %v", err)
	}
	for _, j := range all {
		path := e.pr.PlayerDBPath(e.slug, j.gamertag)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir joueur: %v", err)
		}
		created, err := syncpkg.OpenPlayerDB(path)
		if err != nil {
			t.Fatalf("OpenPlayerDB: %v", err)
		}
		_ = created.Close()
	}

	// Faux réseau : toutes les requêtes HTTP du client de production passent par le faux monde.
	previous := http.DefaultTransport
	http.DefaultTransport = monde.transport()
	t.Cleanup(func() { http.DefaultTransport = previous })
	return e
}

// setActive réécrit db_profiles.json : les joueurs suivis. Le cycle suivant les relit (LoadPlayers).
func (e *difficileEnv) setActive(js ...difficileJoueur) {
	e.t.Helper()
	entries := make([]string, 0, len(js))
	for _, j := range js {
		entries = append(entries, fmt.Sprintf(`%q:{"db_path":%q,"xuid":%q,"waypoint_player":%q}`,
			j.gamertag, "data/titles/"+e.slug+"/players/"+j.gamertag+"/stats.duckdb", j.xuid, j.gamertag))
	}
	writeTestFile(e.t, e.profilesPath, fmt.Sprintf(`{"version":"3.0","admin":%q,"profiles":{%q:{%s}}}`,
		convergenceGamertag, e.slug, strings.Join(entries, ",")))
}

func (e *difficileEnv) playerPath(j difficileJoueur) string {
	return e.pr.PlayerDBPath(e.slug, j.gamertag)
}

// difficileBoot est un démarrage du serveur : orchestrateur, ordonnanceur, file et worker NEUFS.
type difficileBoot struct {
	env      *difficileEnv
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      *config.AppConfig
	sched    *scheduler.AutoSyncScheduler
	rec      *recordingOrchestrator
	st       *bootState
	queue    *persist.BatchQueue
	gate     *difficileGate // nil : file de production
	mgr      *sharedprovider.Manager
	provider sharedprovider.Provider
	metaDB   *duckdbpkg.DB
	stopped  bool
}

// start monte un démarrage complet sur les bases existantes. gated : la file est montée à la main avec la
// porte du worker (sinon c'est openPersistQueue, la fonction de démarrage, RecoverPending compris).
func (e *difficileEnv) start(all []difficileJoueur, gated bool) *difficileBoot {
	t := e.t
	t.Helper()
	e.boots++
	duckdbpkg.CloseAll()

	cfg := &config.AppConfig{RepoRoot: e.root, DBProfilesPath: e.profilesPath, AppSettingsPath: e.settingsPath, PersistBatchAsync: true}
	slug, pr := e.slug, e.pr
	// Migrations comme au démarrage du serveur, à chaque démarrage (elles doivent supporter la reprise).
	if err := runMigrations(pr.MetadataDBPath(slug), pr.SharedDBPath(slug), pr.SharedSocialDBPath(slug), pr.SharedPVEDBPath(slug), ""); err != nil {
		t.Fatalf("runMigrations (démarrage %d): %v", e.boots, err)
	}
	for _, j := range all {
		if err := RunPlayerMigrations(e.playerPath(j), slug); err != nil {
			t.Fatalf("RunPlayerMigrations %s (démarrage %d): %v", j.gamertag, e.boots, err)
		}
	}
	mgr := sharedprovider.NewManager()
	provider, err := mgr.For(pr.SharedDBPath(slug))
	if err != nil {
		t.Fatalf("sharedprovider: %v", err)
	}
	cfg.SharedProvider, cfg.SharedManager = provider, mgr
	metaDB, err := duckdbpkg.OpenReadWriteShared(pr.MetadataDBPath(slug))
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sched := scheduler.New(cfg, settingsplatform.NewStore(e.settingsPath), nil, fakeTokenPool{}).
		WithHistoryReader(historyrepo.New(func(titleSlug, gamertag string) string {
			return config.PlayerDBPath(cfg, titleSlug, gamertag)
		}, nil))
	st := &bootState{
		ctx: ctx, cfg: cfg, pr: pr, titleSlug: slug, autoScheduler: sched, cancelScheduler: cancel,
		dbPaths: bootDBPaths{shared: pr.SharedDBPath(slug), metadata: pr.MetadataDBPath(slug)},
	}
	st.boot = newBootTasks(config.DeploymentDev, &st.schedulerWG)
	b := &difficileBoot{env: e, ctx: ctx, cancel: cancel, cfg: cfg, sched: sched, st: st, mgr: mgr, provider: provider, metaDB: metaDB}

	if gated {
		b.mountGatedQueue()
	} else {
		openPersistQueue(st)
	}
	if st.autoBatchQueue == nil {
		t.Fatal("la file de persistance n'est pas ouverte")
	}
	b.queue = st.autoBatchQueue

	inner := buildSyncV2Orchestrator(SyncV2WiringDeps{
		Cfg: cfg, PathResolver: pr, TitleSlug: slug, TokenPool: fakeTokenPool{},
		BatchQueue: st.autoBatchQueue, MetaDB: metaDB.SQLDb(),
	})
	if inner == nil {
		t.Fatal("buildSyncV2Orchestrator a rendu nil")
	}
	b.rec = &recordingOrchestrator{inner: inner}
	sched.WithCycleOrchestrator(b.rec)

	// Boucle Run, suivie par le schedulerWG comme au serveur (la séquence d'arrêt l'attend).
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		sched.Run(ctx)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for sched.NextCycleAt().IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("la boucle Run de l'ordonnanceur n'a pas démarré")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Cleanup(func() {
		if !b.stopped {
			if b.gate != nil {
				b.gate.letGo()
			}
			b.shutdownOrderly()
		}
	})
	return b
}

// mountGatedQueue monte la file et le worker avec le câblage de startPersistWorker, le persister enveloppé
// par la porte, puis rejoue le journal d'écriture (RecoverPending) comme au démarrage.
func (b *difficileBoot) mountGatedQueue() {
	st, cfg, pr := b.st, b.cfg, b.env.pr
	q, err := persist.NewBatchQueue(persist.BatchQueueConfig{WALDir: pr.WALDir(), ChanBufSize: 1000})
	if err != nil {
		b.env.t.Fatalf("NewBatchQueue: %v", err)
	}
	st.autoBatchQueue = q
	st.autoScheduler.WithBatchQueue(q)
	sharedPath := st.dbPaths.shared
	combined := persist.NewCombinedPersister(
		func(workerCtx context.Context) (*sql.DB, func(), error) {
			return syncpkg.AcquireSharedWriterStandalone(workerCtx, cfg.SharedProvider, sharedPath)
		},
		func(batchTitleSlug, gamertag string) string { return pr.PlayerDBPath(batchTitleSlug, gamertag) },
	)
	b.gate = newDifficileGate(combined)
	worker := persist.NewWorker("combined", q, persist.TargetShared, b.gate)
	st.workerWG.Add(1)
	go func() {
		defer st.workerWG.Done()
		_ = worker.Run(context.Background())
	}()
	if err := q.RecoverPending(); err != nil {
		b.env.t.Fatalf("RecoverPending: %v", err)
	}
}

// shutdownOrderly joue la séquence d'arrêt de production (runShutdown : annulation de l'ordonnanceur,
// attente du scheduler, vidage et fermeture de la file, attente du worker, fermeture des bases). Rend
// vrai pour un arrêt propre.
func (b *difficileBoot) shutdownOrderly() bool {
	d := shutdownDeps{
		reason: "test", srv: &http.Server{}, cancelScheduler: b.cancel, autoScheduler: b.sched,
		schedulerWG: &b.st.schedulerWG, autoBatchQueue: b.queue, workerWG: &b.st.workerWG,
		closeShared: b.mgr.Close, metaDB: b.metaDB,
	}
	clean := runShutdown(d)
	b.stopped = true
	return clean
}

// kill coupe comme la mort du processus : plus de cycle, le worker n'écrit plus (la porte tue), la file
// se ferme SANS vidage, le journal d'écriture reste sur disque, les bases se ferment. writeInFlight : le
// batch en cours d'écriture s'achève en base sans être acquitté.
func (b *difficileBoot) kill(writeInFlight bool) {
	b.cancel()
	b.st.schedulerWG.Wait()
	if b.gate != nil {
		b.gate.kill(writeInFlight)
	}
	_ = b.queue.Close()
	b.st.workerWG.Wait()
	duckdbpkg.CloseAll()
	_ = b.mgr.Close()
	_ = b.metaDB.Close()
	b.stopped = true
}

// recoveryTick joue un passage de la reprise périodique du journal d'écriture (walRecoveryTask : toutes les
// 10 minutes, soit entre deux cycles de rattrapage) : les journaux bloqués sont re-soumis.
func (b *difficileBoot) recoveryTick() {
	if err := b.queue.RecoverPending(); err != nil {
		b.env.t.Fatalf("RecoverPending: %v", err)
	}
}
