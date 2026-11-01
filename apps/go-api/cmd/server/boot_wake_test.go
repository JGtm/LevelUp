// boot_wake_test.go — la séquence de réveil n'est armée que sous la capacité WakeSequence (profil
// desktop) : un serveur hébergé redémarre à chaque déploiement et ne doit voir aucun changement.
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// TestLaunchWakeSequence_SansLaCapaciteRienNeDemarre : capacités nulles (profil hébergé) — si la
// séquence démarrait, elle déréférencerait l'ordonnanceur et le contexte, absents ici.
func TestLaunchWakeSequence_SansLaCapaciteRienNeDemarre(t *testing.T) {
	st := &bootState{cfg: &config.AppConfig{}}
	launchWakeSequence(st)
	st.schedulerWG.Wait()
}

// TestLaunchWakeSequence_SousLaCapaciteRattrapeUneSyncEnRetard : au lancement, sans aucune synchronisation
// connue, la séquence demande un cycle immédiat : la boucle de l'ordonnanceur, dont l'intervalle est
// d'une heure, date un cycle sans attendre. L'annulation du contexte des tâches de fond arrête la
// séquence et libère le groupe d'attente.
func TestLaunchWakeSequence_SousLaCapaciteRattrapeUneSyncEnRetard(t *testing.T) {
	root := t.TempDir()
	settingsPath := filepath.Join(root, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled": true, "spnkr_auto_sync_interval_hours": 1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{
		RepoRoot:        root,
		AppSettingsPath: settingsPath,
		DBProfilesPath:  filepath.Join(root, "db_profiles.json"), // absent : aucun profil suivi
	}
	cfg.Capabilities.WakeSequence = true
	sched := scheduler.New(cfg, settings.NewStore(settingsPath), nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	st := &bootState{cfg: cfg, autoScheduler: sched, schedulerCtx: ctx}
	loopDone := make(chan struct{})
	go func() { sched.Run(ctx); close(loopDone) }()

	launchWakeSequence(st)

	deadline := time.Now().Add(10 * time.Second)
	for sched.Snapshot().LastCycleAt.IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("la séquence du lancement n'a déclenché aucun cycle de sync")
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-loopDone
	done := make(chan struct{})
	go func() { st.schedulerWG.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("la séquence ne s'arrête pas avec le contexte des tâches de fond")
	}
}
