package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlan_LegacyExists(t *testing.T) {
	root := t.TempDir()

	// Créer les répertoires legacy.
	if err := os.MkdirAll(filepath.Join(root, "data", "warehouse"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "data", "players", "TestGT"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "data", "players", "OtherGT"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, "halo_infinite")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	// 1 warehouse + 2 players = 3 ops
	if len(plan.Operations) != 3 {
		t.Errorf("expected 3 operations, got %d", len(plan.Operations))
	}
}

func TestBuildPlan_NoLegacy(t *testing.T) {
	root := t.TempDir()
	// Pas de data/ → que des warnings

	plan, err := BuildPlan(root, "halo_infinite")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	if len(plan.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(plan.Operations))
	}
	if len(plan.Warnings) == 0 {
		t.Error("expected warnings for missing dirs")
	}
}

func TestApplyAndRollback(t *testing.T) {
	root := t.TempDir()

	// Créer structure legacy avec un fichier.
	warehouseDir := filepath.Join(root, "data", "warehouse")
	if err := os.MkdirAll(warehouseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(warehouseDir, "test.duckdb"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	playerDir := filepath.Join(root, "data", "players", "TestGT")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "stats.duckdb"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, "halo_infinite")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	// Apply
	manifest, err := Apply(root, plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(manifest.Operations) != 2 {
		t.Errorf("expected 2 ops, got %d", len(manifest.Operations))
	}

	// Vérifier que les fichiers sont à la bonne place.
	newWarehouse := filepath.Join(root, "data", "titles", "halo_infinite", "warehouse", "test.duckdb")
	if _, err := os.Stat(newWarehouse); os.IsNotExist(err) {
		t.Error("expected test.duckdb in new warehouse dir")
	}

	newPlayer := filepath.Join(root, "data", "titles", "halo_infinite", "players", "TestGT", "stats.duckdb")
	if _, err := os.Stat(newPlayer); os.IsNotExist(err) {
		t.Error("expected stats.duckdb in new player dir")
	}

	// Rollback
	if err := Rollback(root, "halo_infinite"); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// Vérifier que les fichiers sont restaurés.
	if _, err := os.Stat(filepath.Join(warehouseDir, "test.duckdb")); os.IsNotExist(err) {
		t.Error("expected test.duckdb back in legacy warehouse")
	}
	if _, err := os.Stat(filepath.Join(playerDir, "stats.duckdb")); os.IsNotExist(err) {
		t.Error("expected stats.duckdb back in legacy player dir")
	}
}

func TestRollback_AlreadyRolledBack(t *testing.T) {
	root := t.TempDir()

	// Simulate: apply then rollback twice.
	warehouseDir := filepath.Join(root, "data", "warehouse")
	if err := os.MkdirAll(warehouseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	plan, _ := BuildPlan(root, "halo_infinite")
	if _, err := Apply(root, plan); err != nil {
		t.Fatal(err)
	}
	if err := Rollback(root, "halo_infinite"); err != nil {
		t.Fatal(err)
	}

	// Second rollback should fail.
	err := Rollback(root, "halo_infinite")
	if err == nil {
		t.Error("expected error on double rollback")
	}
}
