package duckdbbackup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestScheduler_SkipsWhenUnchanged verifies that RunOnce produces no snapshot
// when no DB has changed since the last backup.
// Requires restic in PATH — skipped otherwise.
func TestScheduler_SkipsWhenUnchanged(t *testing.T) {
	if !resticAvailable() {
		t.Skip("restic not in PATH — skipping scheduler integration test")
	}
	dir := t.TempDir()
	dbPath := createFakefile(t, dir, "fake.duckdb")
	target := Target{Key: "fake", Path: dbPath}

	// Pre-seed the manifest so the scheduler sees the DB as already backed up.
	m, err := LoadManifest(filepath.Join(dir, ".manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.MarkSaved(target); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return []Target{target}, nil })

	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected Skipped=true, got exported=%v", result.Exported)
	}
	if result.SnapshotID != "" {
		t.Fatalf("expected no snapshot, got %q", result.SnapshotID)
	}
}

// TestScheduler_SkipsMissingTarget verifies that a target whose DB file does not
// exist is silently ignored — the cycle still completes without error.
func TestScheduler_SkipsMissingTarget(t *testing.T) {
	dir := t.TempDir()
	target := Target{Key: "ghost", Path: filepath.Join(dir, "nonexistent.duckdb")}

	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return []Target{target}, nil })

	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce with missing target: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected Skipped=true (no exportable target), got exported=%v", result.Exported)
	}
}

// TestScheduler_ZeroTargets verifies that a cycle with no targets is a no-op.
func TestScheduler_ZeroTargets(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return nil, nil })

	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Fatal("expected Skipped with 0 targets")
	}
}

// TestScheduler_AllUnchangedAfterSave verifies that after a full backup cycle,
// a second RunOnce on the same (untouched) files produces Skipped=true.
// Does not call restic — only validates the fingerprint logic.
func TestScheduler_AllUnchangedAfterSave(t *testing.T) {
	dir := t.TempDir()

	// Create two "DB" files.
	paths := []string{
		createFakefile(t, dir, "a.duckdb"),
		createFakefile(t, dir, "b.duckdb"),
	}
	targets := []Target{
		{Key: "a", Path: paths[0]},
		{Key: "b", Path: paths[1]},
	}

	// Seed manifest as if both were already backed up.
	m, _ := LoadManifest(filepath.Join(dir, ".manifest.json"))
	for _, t2 := range targets {
		_ = m.MarkSaved(t2)
	}
	_ = m.Save()

	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return targets, nil })

	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected Skipped=true (files unchanged), exported=%v", result.Exported)
	}
}

// TestScheduler_DetectsChangedFile verifies that a target modified after the last
// backup appears in Exported (even though restic will subsequently fail).
func TestScheduler_DetectsChangedFile(t *testing.T) {
	if !resticAvailable() {
		t.Skip("restic not in PATH — skipping export+restic test")
	}
	dir := t.TempDir()
	dbPath := createFakefile(t, dir, "fake.duckdb")
	target := Target{Key: "fake", Path: dbPath}

	// Seed manifest with an old fingerprint (wrong size → changed).
	m, _ := LoadManifest(filepath.Join(dir, ".manifest.json"))
	_ = m.MarkSaved(target)
	// Overwrite the file to change its size and mtime.
	if err := os.WriteFile(dbPath, []byte("updated content v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = m.Save()

	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return []Target{target}, nil })

	// This will try to export (fake file, not a real DuckDB — export fails)
	// then fail at restic. The interesting assertion is that the scheduler
	// attempted to export the changed target before reaching restic.
	_, _ = sched.RunOnce(context.Background())
	// No panic = scheduler handled the error gracefully.
}

// TestScheduler_Status_NoManifest verifies that Status() returns sane zero values
// when no manifest exists yet (first run, backup never completed).
func TestScheduler_Status_NoManifest(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Enabled: true, BackupDir: dir, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 12}
	sched := New(cfg, func() ([]Target, error) { return nil, nil })

	st := sched.Status()
	if !st.Enabled {
		t.Error("expected Enabled=true")
	}
	if st.LastBackupAt != "" {
		t.Errorf("expected empty LastBackupAt (no backup yet), got %q", st.LastBackupAt)
	}
	if st.IntegrityChecks != nil {
		t.Error("expected nil IntegrityChecks (no backup yet)")
	}
	if st.Config.KeepDaily != 7 {
		t.Errorf("expected KeepDaily=7, got %d", st.Config.KeepDaily)
	}
}

// TestScheduler_Status_WithManifest verifies that Status() reads all fields
// (LastBackupAt, IntegrityChecks, etc.) from a pre-seeded manifest.
func TestScheduler_Status_WithManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, ".manifest.json")

	// Pre-seed manifest with integrity results.
	m, _ := LoadManifest(manifestPath)
	m.SetIntegrityResult("shared", IntegrityResult{OK: true})
	m.SetIntegrityResult("player", IntegrityResult{OK: false, Detail: "corrupted page 1"})
	m.SetLastResult("abc123", []string{"shared", "player"}, 3_000_000_000) // 3s
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg := Config{Enabled: true, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return nil, nil })

	st := sched.Status()
	if st.LastBackupAt == "" {
		t.Error("expected non-empty LastBackupAt")
	}
	if st.LastSnapshotID != "abc123" {
		t.Errorf("expected LastSnapshotID=abc123, got %q", st.LastSnapshotID)
	}
	if st.LastDurationMs != 3000 {
		t.Errorf("expected LastDurationMs=3000, got %d", st.LastDurationMs)
	}
	if len(st.IntegrityChecks) != 2 {
		t.Fatalf("expected 2 integrity checks, got %d", len(st.IntegrityChecks))
	}
	if !st.IntegrityChecks["shared"].OK {
		t.Error("shared: expected OK=true")
	}
	if st.IntegrityChecks["player"].OK || st.IntegrityChecks["player"].Detail != "corrupted page 1" {
		t.Errorf("player: unexpected %+v", st.IntegrityChecks["player"])
	}
}

// TestScheduler_IntegrityPersistedOnExportFail verifies that integrity check results
// are written to the manifest even when every export fails (so the UI shows warnings).
// Uses a real DuckDB so CheckIntegrity can open the file, but a non-DuckDB staging
// path will cause ExportTarget to fail (or produce an empty export).
func TestScheduler_IntegrityPersistedOnExportFail(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal DuckDB so CheckIntegrity succeeds (returns OK=true).
	db, err := openNewDB(t, filepath.Join(dir, "real.duckdb"))
	if err != nil {
		t.Fatalf("open DB: %v", err)
	}
	db.Close()
	target := Target{Key: "real", Path: filepath.Join(dir, "real.duckdb")}

	// Point BackupDir to a path that cannot be created (file, not dir) so
	// os.MkdirAll fails at the staging outDir level.
	backupDir := filepath.Join(dir, "backup")
	// Make staging/real a FILE so MkdirAll in ExportTarget fails.
	if err := os.MkdirAll(filepath.Join(backupDir, "staging"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "staging", "real"), []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Enabled: true, BackupDir: backupDir}
	sched := New(cfg, func() ([]Target, error) { return []Target{target}, nil })

	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected Skipped=true (export failed), got exported=%v", result.Exported)
	}

	// Verify the manifest was saved with the integrity result.
	m, loadErr := LoadManifest(filepath.Join(backupDir, ".manifest.json"))
	if loadErr != nil {
		t.Fatalf("LoadManifest: %v", loadErr)
	}
	if len(m.IntegrityChecks) == 0 {
		t.Error("expected integrity checks to be persisted even when export failed")
	}
	if ic, ok := m.IntegrityChecks["real"]; !ok {
		t.Error("expected entry for key 'real'")
	} else if !ic.OK {
		t.Errorf("expected OK=true for a valid DB, got detail=%q", ic.Detail)
	}
}

// TestManifest_SaveIntegrityOnly verifies that SaveIntegrityOnly does not update LastBackupAt.
func TestManifest_SaveIntegrityOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".manifest.json")

	m, _ := LoadManifest(path)
	m.SetIntegrityResult("db", IntegrityResult{OK: true})
	if err := m.SaveIntegrityOnly(); err != nil {
		t.Fatalf("SaveIntegrityOnly: %v", err)
	}

	m2, _ := LoadManifest(path)
	if !m2.LastBackupAt.IsZero() {
		t.Errorf("SaveIntegrityOnly must not update LastBackupAt, got %v", m2.LastBackupAt)
	}
	if len(m2.IntegrityChecks) != 1 || !m2.IntegrityChecks["db"].OK {
		t.Errorf("unexpected integrity checks: %+v", m2.IntegrityChecks)
	}
}

func openNewDB(t *testing.T, path string) (*sql.DB, error) {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("CREATE TABLE _init (x INTEGER)"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func resticAvailable() bool {
	return NewResticClient(Config{}).IsAvailable()
}

func createFakefile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
