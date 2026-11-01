//go:build cgo

// boot_shared_migrations_test.go — migrateSharedAtBoot : étape en échec, verrou repris puis
// refusé, schéma plus récent, copie avant montée présente / absente / refusée. DuckDB en
// fichiers temporaires.
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const bootTestSlug = migration.DefaultSlug

// bootTestDB crée un fichier de base sous un dossier temporaire et rend son chemin.
func bootTestDB(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("création de %s: %v", name, err)
	}
	if _, err := db.SQLDb().Exec("CREATE TABLE seed (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// registerBootStep enregistre une étape de test (une fois : le registre est global au processus).
func registerBootStep(name string, target migration.TargetDB) {
	if migration.IsRegistered(name) {
		return
	}
	migration.Register(migration.Migration{
		Name:        name,
		TargetDB:    target,
		Description: "étape de test du démarrage",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INTEGER)")
			return err
		},
	})
}

// runRW rend une fonction qui monte le schéma de target sur path (runMigrations réduit).
func runRW(path string, target migration.TargetDB, calls *atomic.Int32) func() error {
	return func() error {
		calls.Add(1)
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			return err
		}
		defer db.Close()
		return migration.RunForDB(db.SQLDb(), target)
	}
}

func testDeps(path string, target migration.TargetDB, backups string, run func() error) bootMigrationDeps {
	d := newBootMigrationDeps(bootTestSlug, backups, "", []bootMigrationTarget{{path, target}}, run)
	d.prepare = func() error { return nil }
	d.lockAttempts, d.lockDelay = 3, time.Millisecond
	return d
}

func TestMigrateSharedAtBoot_EtapeEnEchec_MigrationFailedPuisRefusDeDemarrer(t *testing.T) {
	logs := captureSlog(t)
	target := migration.TargetDB("boot_test_failing")
	path := bootTestDB(t, "failing.duckdb")
	failing := migration.Migration{
		Name: "boot_test_failing_step", TargetDB: target,
		ApplySchema: func(*sql.DB) error { return errors.New("boom d'étape") },
	}
	var calls atomic.Int32
	run := func() error {
		calls.Add(1)
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			return err
		}
		defer db.Close()
		return migration.RunSteps(db.SQLDb(), target, []migration.Migration{failing})
	}

	state, err := migrateSharedAtBoot(t.Context(), testDeps(path, target, "", run))
	if err == nil || state != bootFailureMigrationFailed {
		t.Fatalf("état %q, err %v : migration_failed attendu", state, err)
	}
	if calls.Load() != 1 {
		t.Errorf("run appelé %d fois, une seule attendue (échec non lié au verrou : pas de reprise)", calls.Load())
	}

	// Le refus de démarrer garde la porte, journalise en Error et sort en erreur au signal.
	b, sigCh, _ := startSignalWatch(t)
	done := make(chan int, 1)
	go func() { done <- b.refuseStart(t.Context(), state, err) }()
	waitFailed(t, b.gate)
	sigCh <- os.Interrupt
	select {
	case code := <-done:
		if code == 0 {
			t.Error("code de sortie nul")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("refuseStart ne rend pas la main")
	}
	assertFailed(t, b.gate, bootFailureMigrationFailed)
	for _, want := range []string{"level=ERROR", "migration: échec, démarrage refusé", "state=migration_failed", "boom d'étape"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("journal sans %q", want)
		}
	}
}

func TestMigrateSharedAtBoot_VerrouTenu_RepriseEpuiseePuisLocked(t *testing.T) {
	target := migration.TargetDB("boot_test_locked")
	path := bootTestDB(t, "locked.duckdb")
	holder, err := sql.Open("duckdb", path) // second handle : tient le fichier en écriture
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Close() })
	if _, err := holder.Exec("CREATE TABLE held (id INTEGER)"); err != nil {
		t.Fatal(err)
	}

	var attempts, runs atomic.Int32
	d := testDeps(path, target, "", runRW(path, target, &runs))
	d.openRead = func(p string) (*sql.DB, func(), error) {
		attempts.Add(1)
		return duckdb.OpenReadForQuery(p)
	}

	state, err := migrateSharedAtBoot(t.Context(), d)
	if state != bootFailureLocked || err == nil {
		t.Fatalf("état %q, err %v : locked attendu", state, err)
	}
	if attempts.Load() != 3 {
		t.Errorf("%d essais d'ouverture, 3 attendus (reprise bornée puis abandon)", attempts.Load())
	}
	if runs.Load() != 0 {
		t.Error("migrations lancées alors que la base était tenue")
	}
}

func TestMigrateSharedAtBoot_VerrouLibereEnCours_RepriseReussit(t *testing.T) {
	target := migration.TargetDB("boot_test_released")
	registerBootStep("boot_test_released_step", target)
	path := bootTestDB(t, "released.duckdb")
	holder, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := holder.Exec("CREATE TABLE held (id INTEGER)"); err != nil {
		t.Fatal(err)
	}

	var attempts, runs atomic.Int32
	d := testDeps(path, target, "", runRW(path, target, &runs))
	d.lockAttempts = 5
	d.openRead = func(p string) (*sql.DB, func(), error) {
		if attempts.Add(1) == 2 {
			_ = holder.Close() // le détenteur s'en va entre deux essais
		}
		return duckdb.OpenReadForQuery(p)
	}

	state, err := migrateSharedAtBoot(t.Context(), d)
	if err != nil || state != "" {
		t.Fatalf("état %q, err %v : succès attendu après libération", state, err)
	}
	if attempts.Load() != 2 || runs.Load() != 1 {
		t.Errorf("essais=%d runs=%d, attendu 2 essais (1 refusé, 1 réussi) et 1 montée", attempts.Load(), runs.Load())
	}
}

func TestMigrateSharedAtBoot_BaseN1_SchemaNewerSansEcriture(t *testing.T) {
	target := migration.TargetDB("boot_test_newer")
	registerBootStep("boot_test_newer_step", target)
	path := bootTestDB(t, "newer.duckdb")
	var runs atomic.Int32
	if err := runRW(path, target, &runs)(); err != nil {
		t.Fatal(err)
	}
	runs.Store(0)
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQLDb().Exec("UPDATE title_schema_version SET version = version + 1 WHERE target = ?", string(target)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	backups := filepath.Join(t.TempDir(), "backups")
	state, err := migrateSharedAtBoot(t.Context(), testDeps(path, target, backups, runRW(path, target, &runs)))
	if state != bootFailureSchemaNewer || !errors.Is(err, migration.ErrSchemaNewerThanBinary) {
		t.Fatalf("état %q, err %v : schema_newer attendu", state, err)
	}
	if runs.Load() != 0 {
		t.Error("migrations lancées sur une base plus récente que le binaire")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("la base plus récente a été modifiée")
	}
	if _, statErr := os.Stat(path + ".wal"); statErr == nil {
		t.Error("un WAL est apparu : la base a été ouverte en écriture")
	}
	if _, statErr := os.Stat(backups); statErr == nil {
		t.Error("une copie a été prise pour une base refusée")
	}
}

func TestMigrateSharedAtBoot_Copie_PresenteAbsenteRefusee(t *testing.T) {
	target := migration.TargetDB("boot_test_backup")
	registerBootStep("boot_test_backup_a", target)
	var runs atomic.Int32

	t.Run("étapes en attente : copie AVANT la montée", func(t *testing.T) {
		path := bootTestDB(t, "metadata.duckdb")
		if err := runRW(path, target, &runs)(); err != nil { // base déjà à niveau
			t.Fatal(err)
		}
		registerBootStep(uniqueStepName("boot_test_backup_b"), target) // une étape de plus au binaire
		backups := filepath.Join(t.TempDir(), "backups")
		sawCopy := false
		run := func() error {
			entries, _ := os.ReadDir(backups)
			sawCopy = len(entries) > 0
			return runRW(path, target, &runs)()
		}
		state, err := migrateSharedAtBoot(t.Context(), testDeps(path, target, backups, run))
		if err != nil || state != "" {
			t.Fatalf("état %q, err %v", state, err)
		}
		if !sawCopy {
			t.Error("aucune copie présente quand la montée a commencé")
		}
		entries, _ := os.ReadDir(backups)
		if len(entries) == 0 || filepath.Ext(entries[0].Name()) != ".duckdb" {
			t.Errorf("copie attendue en .duckdb, trouvé %v", entries)
		}
	})

	t.Run("rien en attente : pas de copie", func(t *testing.T) {
		path := bootTestDB(t, "metadata.duckdb")
		if err := runRW(path, target, &runs)(); err != nil {
			t.Fatal(err)
		}
		backups := filepath.Join(t.TempDir(), "backups")
		if state, err := migrateSharedAtBoot(t.Context(), testDeps(path, target, backups, runRW(path, target, &runs))); err != nil || state != "" {
			t.Fatalf("état %q, err %v", state, err)
		}
		if _, err := os.Stat(backups); err == nil {
			t.Error("copie prise alors que rien n'était en attente")
		}
	})

	t.Run("base absente : rien à copier, la montée la crée", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fresh.duckdb")
		backups := filepath.Join(t.TempDir(), "backups")
		if state, err := migrateSharedAtBoot(t.Context(), testDeps(path, target, backups, runRW(path, target, &runs))); err != nil || state != "" {
			t.Fatalf("état %q, err %v", state, err)
		}
		if _, err := os.Stat(backups); err == nil {
			t.Error("copie prise d'une base qui n'existait pas")
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("base non créée par la montée: %v", err)
		}
	})

	t.Run("espace insuffisant : refus, migration_failed, aucune montée", func(t *testing.T) {
		path := bootTestDB(t, "metadata.duckdb")
		registerBootStep(uniqueStepName("boot_test_backup_c"), target)
		backups := filepath.Join(t.TempDir(), "backups")
		var local atomic.Int32
		d := testDeps(path, target, backups, runRW(path, target, &local))
		d.freeBytes = func(string) (uint64, error) { return 0, nil }
		state, err := migrateSharedAtBoot(t.Context(), d)
		if state != bootFailureMigrationFailed || !errors.Is(err, migration.ErrInsufficientBackupSpace) {
			t.Fatalf("état %q, err %v : migration_failed (espace) attendu", state, err)
		}
		if local.Load() != 0 {
			t.Error("migrations lancées sans copie de sécurité")
		}
	})
}

// uniqueStepName rend un nom d'étape neuf à chaque appel : une étape « ajoutée au binaire »
// doit être en attente même quand le test est rejoué dans le même processus (-count).
func uniqueStepName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
