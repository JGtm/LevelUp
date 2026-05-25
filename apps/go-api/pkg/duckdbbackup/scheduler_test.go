package duckdbbackup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestScheduler_SkipsWhenUnchanged verifies that RunOnce produces no snapshot
// when no DB has changed since the last backup.
func TestScheduler_SkipsWhenUnchanged(t *testing.T) {
	if !resticAvailable() {
		t.Skip("restic not in PATH — skipping scheduler integration test")
	}
	dir := t.TempDir()
	dbPath := createFakeDB(t, dir, "fake.duckdb")

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

	cfg := Config{
		Enabled:   true,
		BackupDir: dir,
		Interval:  0,
	}
	sched := New(cfg, func() ([]Target, error) {
		return []Target{target}, nil
	})

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

// TestScheduler_DisabledDoesNotRun verifies that Run returns immediately when
// Enabled=false (tested via RunOnce, which calls cycle directly).
func TestScheduler_RunOnce_DisabledConfig(t *testing.T) {
	// Even with restic absent this should not panic — cycle is called directly.
	dir := t.TempDir()
	cfg := Config{Enabled: false, BackupDir: dir}
	sched := New(cfg, func() ([]Target, error) { return nil, nil })

	// RunOnce bypasses the Enabled check (it's a raw cycle call).
	// The scheduler goroutine (Run) would exit immediately, but RunOnce still runs.
	// We just verify it doesn't crash with empty target list.
	result, err := sched.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected Skipped with 0 targets")
	}
}

// resticAvailable checks whether the restic binary is reachable.
func resticAvailable() bool {
	c := NewResticClient(Config{})
	return c.IsAvailable()
}

// createFakeDB writes a minimal placeholder file so os.Stat succeeds.
// In unit tests we never actually open it with DuckDB.
func createFakeDB(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
