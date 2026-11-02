package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// TestPeriodicCheckpointTask_EmptiesTheWALOfAHeldDatabase : la tâche vide le WAL d'une base
// tenue en RW, laisse celle de skip, et s'arrête à l'annulation du contexte.
func TestPeriodicCheckpointTask_EmptiesTheWALOfAHeldDatabase(t *testing.T) {
	dir := t.TempDir()
	held := filepath.Join(dir, "held.duckdb")
	skipped := filepath.Join(dir, "skipped.duckdb")
	for _, p := range []string{held, skipped} {
		db, err := duckdb.OpenReadWrite(p)
		if err != nil {
			t.Fatalf("OpenReadWrite: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if _, err := db.SQLDb().Exec(`CREATE TABLE t (id INTEGER)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.SQLDb().Exec(`INSERT INTO t VALUES (1)`); err != nil {
			t.Fatal(err)
		}
	}
	wal := func(p string) int64 {
		info, err := os.Stat(p + ".wal")
		if err != nil {
			return 0
		}
		return info.Size()
	}
	skippedBefore := wal(skipped)

	ctx, cancel := context.WithCancel(context.Background())
	task := periodicCheckpointTask(ctx, 20*time.Millisecond, map[string]bool{skipped: true})
	if !task.tracked || task.name != taskPeriodicCkpt {
		t.Fatalf("tâche = %+v ; attendue suivie et nommée %q", task, taskPeriodicCkpt)
	}
	done := make(chan struct{})
	go func() { task.run(); close(done) }()

	deadline := time.Now().Add(10 * time.Second)
	for wal(held) != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("la tâche ne s'arrête pas à l'annulation du contexte")
	}
	if got := wal(held); got != 0 {
		t.Errorf("WAL de la base tenue = %d octets ; attendu vidé", got)
	}
	if got := wal(skipped); got != skippedBefore {
		t.Errorf("WAL de la base à sauter changé (%d -> %d)", skippedBefore, got)
	}
}

// TestPeriodicCheckpoint_DeclaredLikeEveryTask : la tâche est déclarée par profil (tournée sauf en
// démo, où elle écrirait hors de la racine démo).
func TestPeriodicCheckpoint_DeclaredLikeEveryTask(t *testing.T) {
	if policyOf(taskPeriodicCkpt, config.DeploymentDesktop) != taskKeep ||
		policyOf(taskPeriodicCkpt, config.DeploymentDemo) != taskCut {
		t.Error("taskPeriodicCkpt : desktop doit la lancer, la démo la couper")
	}
}

// openHeldWithWAL ouvre une base en RW, y écrit une ligne (le WAL la porte) et rend son chemin.
func openHeldWithWAL(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{`CREATE TABLE t (id INTEGER)`, `INSERT INTO t VALUES (1)`} {
		if _, err := db.SQLDb().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func walBytes(p string) int64 {
	info, err := os.Stat(p + ".wal")
	if err != nil {
		return 0
	}
	return info.Size()
}

// TestSharedSocialPaths_EveryTitleThroughThePathResolver : le shared_social de CHAQUE titre (pas
// seulement celui du boot) vient du PathResolver, sans doublon ; celui du boot (la fixture en démo)
// y figure toujours.
func TestSharedSocialPaths_EveryTitleThroughThePathResolver(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	boot := pr.SharedSocialDBPath(title.DefaultSlug)
	got := sharedSocialPaths(pr, []string{title.DefaultSlug, "halo_5"}, boot)
	want := map[string]bool{boot: true, pr.SharedSocialDBPath("halo_5"): true}
	if len(got) != len(want) {
		t.Fatalf("chemins = %v ; attendu %d sans doublon", got, len(want))
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("chemin inattendu %q", p)
		}
	}
	demo := filepath.Join(t.TempDir(), "demo", "shared_social.duckdb")
	if withDemo := sharedSocialPaths(pr, []string{"halo_5"}, demo); len(withDemo) != 2 {
		t.Errorf("chemins = %v ; le chemin du boot s'ajoute à celui des titres", withDemo)
	}
}

// TestPeriodicCheckpointFor_SkipsTheSharedSocialOfEveryTitle : le CHECKPOINT périodique des autres
// bases laisse le shared_social du titre par défaut ET celui de Halo 5 (ils ont leur tâche) et
// vide le WAL d'une autre base tenue.
func TestPeriodicCheckpointFor_SkipsTheSharedSocialOfEveryTitle(t *testing.T) {
	root := t.TempDir()
	pr := title.NewPathResolver(root)
	social := sharedSocialPaths(pr, []string{title.DefaultSlug, "halo_5"}, pr.SharedSocialDBPath(title.DefaultSlug))
	for _, p := range social {
		openHeldWithWAL(t, p)
	}
	held := openHeldWithWAL(t, filepath.Join(root, "held.duckdb"))
	socialBefore := map[string]int64{}
	for _, p := range social {
		socialBefore[p] = walBytes(p)
	}

	if walBytes(held) == 0 || walBytes(social[0]) == 0 {
		t.Skip("DuckDB a déjà vidé le WAL de lui-même : rien à démontrer")
	}
	cfg := &config.AppConfig{}
	cfg.CheckpointInterval = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	task, ok := periodicCheckpointFor(ctx, cfg, social)
	if !ok {
		t.Fatal("intervalle > 0 : la tâche doit être lancée")
	}
	done := make(chan struct{})
	go func() { task.run(); close(done) }()
	deadline := time.Now().Add(10 * time.Second)
	for walBytes(held) != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done
	if walBytes(held) != 0 {
		t.Error("la base tenue n'a pas été checkpointée")
	}
	for _, p := range social {
		if walBytes(p) != socialBefore[p] {
			t.Errorf("le shared_social %s a été checkpointé par la tâche périodique", p)
		}
	}
}

// TestPeriodicCheckpointFor_ZeroIntervalLaunchesNothing : sans la capacité FrequentCheckpoint
// l'intervalle est nul et la tâche n'est pas lancée.
func TestPeriodicCheckpointFor_ZeroIntervalLaunchesNothing(t *testing.T) {
	cfg := &config.AppConfig{}
	if _, ok := periodicCheckpointFor(context.Background(), cfg, nil); ok {
		t.Error("intervalle nul : aucune tâche attendue")
	}
	cfg.CheckpointInterval = time.Minute
	task, ok := periodicCheckpointFor(context.Background(), cfg, nil)
	if !ok || task.name != taskPeriodicCkpt || !task.tracked {
		t.Errorf("intervalle posé : tâche = %+v, %v ; attendue suivie et nommée %q", task, ok, taskPeriodicCkpt)
	}
}

// TestCheckpointSharedSocial_EmptiesEveryOpenedTitleAndIgnoresTheOthers : la tâche sociale vide le
// WAL de chaque shared_social ouvert (Halo 5 compris) et passe son chemin quand la base n'est pas ouverte.
func TestCheckpointSharedSocial_EmptiesEveryOpenedTitleAndIgnoresTheOthers(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	infinite := openHeldWithWAL(t, pr.SharedSocialDBPath(title.DefaultSlug))
	halo5 := openHeldWithWAL(t, pr.SharedSocialDBPath("halo_5"))
	if walBytes(infinite) == 0 || walBytes(halo5) == 0 {
		t.Skip("DuckDB a déjà vidé le WAL de lui-même : rien à démontrer")
	}
	for _, p := range []string{infinite, halo5, filepath.Join(t.TempDir(), "jamais_ouverte.duckdb")} {
		checkpointSharedSocial(p)
	}
	if walBytes(infinite) != 0 || walBytes(halo5) != 0 {
		t.Errorf("WAL restants : %d / %d octets ; attendus vidés", walBytes(infinite), walBytes(halo5))
	}
}

// TestRegisteredSlugs_ListsEveryRegisteredTitle : le registre sert de source aux titres dont le
// shared_social est checkpointé par la tâche sociale.
func TestRegisteredSlugs_ListsEveryRegisteredTitle(t *testing.T) {
	slugs := registeredSlugs(title.DefaultRegistry())
	found := false
	for _, s := range slugs {
		if s == title.DefaultSlug {
			found = true
		}
	}
	if !found || len(slugs) != len(title.DefaultRegistry().All()) {
		t.Errorf("slugs = %v ; attendu tous les titres du registre dont %q", slugs, title.DefaultSlug)
	}
}
