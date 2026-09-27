// background_tasks_test.go — en démo, le lancement des tâches de fond ne touche à AUCUN
// fichier hors de la racine démo (backlog 2026-09-26, lot B5.7-2 ; décision D-7).
//
// Le harnais visuel lance la démo avec LEVELUP_REPO_ROOT = le VRAI checkout. Avant B5, le
// janitor y purgeait data/sync_cache et le WAL, le monitoring ouvrait
// data/global/monitoring.duckdb en écriture, la migration des amis écrivait
// data/global/player_friends.json, la purge des rejeux supprimait dans data/cache/replays…
// Le test pose un dépôt LEURRE garni de fichiers sentinelles, lance les tâches et étapes de
// boot RÉELLES (celles qui ne demandent pas le registre complet ; les autres par des
// doublures qui crient si on les lance), puis compare un manifeste avant/après
// (chemin, taille, date).
package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/scheduler"
)

// entreeManifeste : ce qui doit rester identique d'un fichier du dépôt leurre.
type entreeManifeste struct {
	taille int64
	date   time.Time
	dir    bool
}

func manifeste(t *testing.T, racine string) map[string]entreeManifeste {
	t.Helper()
	out := map[string]entreeManifeste{}
	err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		rel, _ := filepath.Rel(racine, p)
		out[filepath.ToSlash(rel)] = entreeManifeste{taille: info.Size(), date: info.ModTime(), dir: d.IsDir()}
		return nil
	})
	if err != nil {
		t.Fatalf("manifeste %s: %v", racine, err)
	}
	return out
}

func ecrire(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", chemin, err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatalf("write %s: %v", chemin, err)
	}
}

// garnirLeurre pose les sentinelles que les tâches de fond d'avant B5 touchaient.
func garnirLeurre(t *testing.T, depot string) {
	t.Helper()
	pr := title.NewPathResolver(depot)
	vieuxCycle := filepath.Join(pr.SyncCacheDir(), "sync.RunDelta_vieux")
	ecrire(t, filepath.Join(vieuxCycle, "lot.json"), `{"sentinelle":true}`)
	ilYA30Jours := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(vieuxCycle, ilYA30Jours, ilYA30Jours); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	ecrire(t, filepath.Join(pr.WALDir(), "pending.json"), `{"sentinelle":true}`)
	ecrire(t, pr.ActionJournalPath(), `{"sentinelle":true}`)
	ecrire(t, pr.ReplayArtifactPath(title.DefaultSlug, "deadbeef-0000-0000-0000-000000000000"), `{"sentinelle":true}`)
	ecrire(t, filepath.Join(depot, "logs", "sync.log"), "sentinelle\n")
	ecrire(t, pr.AppSettingsPath(), `{"lang":"fr"}`)
}

// garnirDemo pose une fixture minimale : réglages et un db_profiles qui désigne un admin.
func garnirDemo(t *testing.T, demo string) {
	t.Helper()
	l := title.NewDemoLayout(demo)
	ecrire(t, l.AppSettingsPath(), `{"lang":"fr"}`)
	ecrire(t, l.DBProfilesPath(), `{"version":"3.0","admin":"DemoPlayer","profiles":{}}`)
}

// doublure : une boucle de fond qui compte ses lancements.
type doublure struct{ lancee atomic.Int32 }

func (d *doublure) run(ctx context.Context) {
	d.lancee.Add(1)
	<-ctx.Done()
}

// tachesDeTest construit les tâches de boot : réelles quand elles ne demandent que la config
// (janitor, santé données, purge des rejeux, checkpoint, heartbeat), doublures sinon.
func tachesDeTest(ctx context.Context, cfg *config.AppConfig, doublures map[string]*doublure) []backgroundTask {
	pr := title.NewPathResolver(cfg.RepoRoot)
	taches := []backgroundTask{
		janitorTask(ctx, pr, nil),
		socialCheckpointTask(ctx, cfg.DemoLayout().SharedSocialDBPath(title.DefaultSlug)),
		poolRescanTask(ctx, cfg, nil),
		loopTask(taskDataHealth, true, ctx, scheduler.NewDataHealthScheduler(cfg.RepoRoot).Run),
		loopTask(taskReplayPurge, true, ctx, scheduler.NewReplayPurgeCron(cfg.RepoRoot, func() int { return 1 }, 0).Run),
		heartbeatTask(ctx, time.Now()),
	}
	for _, nom := range []string{taskAutoSync, taskDetectionFlush, taskDiskWatch, taskReplayNotify,
		taskCatalogCron, taskAssetNameSweep, taskSpartanCron, taskWorldLeaderbd, taskAppRelease} {
		d := &doublure{}
		doublures[nom] = d
		taches = append(taches, loopTask(nom, true, ctx, d.run))
	}
	return taches
}

// TestBackgroundTasks_Demo_ManifesteDuDepotInchange : en démo, sur un dépôt leurre, aucune
// tâche ni étape de boot ne crée, ne modifie ni ne supprime un fichier du dépôt — et
// monitoring.duckdb n'y est jamais créé.
func TestBackgroundTasks_Demo_ManifesteDuDepotInchange(t *testing.T) {
	depot := t.TempDir()
	demo := t.TempDir()
	garnirLeurre(t, depot)
	garnirDemo(t, demo)
	cfg := chargerCfg(t, depot, demo, true)
	avant := manifeste(t, depot)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	b := newBootTasks(cfg.DemoMode, &wg)

	runBootMigrations(ctx, b, cfg, bootFriendStore(cfg), groupstore.NewGroupStore(filepath.Join(cfg.AuthDir, "groups.json")))
	mon, err := openMonitoringStore(ctx, cfg, title.NewPathResolver(cfg.RepoRoot))
	if err != nil {
		t.Fatalf("openMonitoringStore: %v", err)
	}
	if err := mon.RecordCronRun(ctx, "server_boot", time.Now(), true, "", 0); err != nil {
		t.Fatalf("marqueur server_boot : %v", err)
	}
	doublures := map[string]*doublure{}
	for _, tache := range tachesDeTest(ctx, cfg, doublures) {
		b.launch(tache)
	}
	time.Sleep(1500 * time.Millisecond) // premier passage des tâches qui tirent au boot
	cancel()
	wg.Wait()
	if err := mon.Close(); err != nil {
		t.Errorf("fermeture du magasin monitoring : %v", err)
	}

	apres := manifeste(t, depot)
	var ecarts []string
	for p, e := range avant {
		a, ok := apres[p]
		switch {
		case !ok:
			ecarts = append(ecarts, "SUPPRIMÉ "+p)
		case !e.dir && (a.taille != e.taille || !a.date.Equal(e.date)):
			ecarts = append(ecarts, "MODIFIÉ "+p)
		}
	}
	for p := range apres {
		if _, ok := avant[p]; !ok {
			ecarts = append(ecarts, "CRÉÉ "+p)
		}
	}
	sort.Strings(ecarts)
	for _, e := range ecarts {
		t.Errorf("dépôt leurre touché en démo : %s", e)
	}
	coupees := map[string]bool{}
	for _, nom := range b.cut {
		coupees[nom] = true
	}
	for _, nom := range []string{stepFriendsMigr, stepGroupMigr, taskJanitor, taskPoolRescan, taskDataHealth, taskReplayPurge} {
		if !coupees[nom] {
			t.Errorf("%q absent de la liste des coupées du log de boot : %v", nom, b.cut)
		}
	}
	for nom, d := range doublures {
		if policyOf(nom) == demoCut && d.lancee.Load() > 0 {
			t.Errorf("tâche coupée en démo LANCÉE : %s", nom)
		}
		if policyOf(nom) == demoKeep && d.lancee.Load() == 0 {
			t.Errorf("tâche gardée en démo NON lancée : %s", nom)
		}
	}
}

// TestBackgroundTasks_HorsDemo_ToutLance : hors démo, rien n'est coupé — chaque tâche est
// lancée et aucune étape de boot n'est sautée (le comportement hors démo ne change pas).
func TestBackgroundTasks_HorsDemo_ToutLance(t *testing.T) {
	var wg sync.WaitGroup
	b := newBootTasks(false, &wg)
	ctx, cancel := context.WithCancel(context.Background())
	var lancees []*doublure
	for nom := range demoPolicies {
		if b.cutInDemo(nom) {
			t.Errorf("étape %q sautée hors démo", nom)
		}
		d := &doublure{}
		lancees = append(lancees, d)
		if !b.launch(loopTask(nom, true, ctx, d.run)) {
			t.Errorf("tâche %q non lancée hors démo", nom)
		}
	}
	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()
	for i, d := range lancees {
		if d.lancee.Load() != 1 {
			t.Errorf("doublure %d lancée %d fois, attendu 1", i, d.lancee.Load())
		}
	}
	if len(b.cut) != 0 {
		t.Errorf("hors démo, liste des coupées = %v, attendu vide", b.cut)
	}
}

// TestBackgroundTasks_DeclarationDemo : la déclaration des statuts démo suit le tableau B5.0
// du plan — chaque nom de tâche ou d'étape y figure, avec le statut attendu.
func TestBackgroundTasks_DeclarationDemo(t *testing.T) {
	gardees := map[string]bool{
		taskSocialCkpt: true, taskDetectionFlush: true, taskAppRelease: true, taskHeartbeat: true,
	}
	tous := []string{
		taskJanitor, taskWALRecovery, taskSocialCkpt, taskPoolRescan, taskAutoSync, taskDataHealth,
		taskDetectionFlush, taskDiskWatch, taskReplayNotify, taskCatalogCron, taskAssetNameSweep,
		taskSpartanCron, taskWorldLeaderbd, taskReplayPurge, taskAppRelease, taskHeartbeat,
		stepFriendsMigr, stepGroupMigr, stepTokenPool, stepPersistQueue, stepWatcher,
	}
	if len(tous) != len(demoPolicies) {
		t.Errorf("%d noms déclarés dans demoPolicies, %d attendus", len(demoPolicies), len(tous))
	}
	for _, nom := range tous {
		p, ok := demoPolicies[nom]
		if !ok {
			t.Errorf("%q absent de demoPolicies", nom)
			continue
		}
		if want := map[bool]demoPolicy{true: demoKeep, false: demoCut}[gardees[nom]]; p != want {
			t.Errorf("%q : statut démo %v, attendu %v (tableau B5.0)", nom, p, want)
		}
	}
	if policyOf("tâche inconnue") != demoCut {
		t.Error("une tâche non déclarée doit être coupée en démo (défaut sûr)")
	}
}
