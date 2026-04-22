//go:build cgo

// Package ops — healthcheck_cgo_test.go : tests de checkDuckDB et RunHealthcheck
// avec une DB DuckDB temporaire / répertoire vide.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// createDBAt crée une DuckDB vide à un chemin précis et la ferme.
func createDBAt(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("createDBAt open %s: %v", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("createDBAt ping %s: %v", path, err)
	}
	db.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// checkDuckDB
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckDuckDB_FileAbsent(t *testing.T) {
	check := checkDuckDB("test_absent", "/nonexistent/db.duckdb")
	if check.OK {
		t.Error("expected OK=false pour fichier absent")
	}
	if check.Name != "test_absent" {
		t.Errorf("Name = %q, want test_absent", check.Name)
	}
}

func TestCheckDuckDB_ValidDB(t *testing.T) {
	path := openTempDB(t) // helper défini dans seed_cgo_test.go
	check := checkDuckDB("test_valid", path)
	if !check.OK {
		t.Errorf("expected OK=true pour DB valide, got msg=%q", check.Message)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkFileExists
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckFileExists_Absent(t *testing.T) {
	check := checkFileExists("cfg", "/nonexistent/file.json")
	if check.OK {
		t.Error("expected OK=false pour fichier absent")
	}
	if check.Name != "cfg" {
		t.Errorf("Name = %q, want cfg", check.Name)
	}
}

func TestCheckFileExists_Present(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/config.json"
	if err := os.WriteFile(p, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	check := checkFileExists("config.json", p)
	if !check.OK {
		t.Errorf("expected OK=true, got msg=%q", check.Message)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RunHealthcheck — avec un répertoire temporaire vide
// ─────────────────────────────────────────────────────────────────────────────

func TestRunHealthcheck_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	report := RunHealthcheck(HealthcheckOptions{RepoRoot: dir})
	// Le rapport doit exister et contenir au moins le check "os"
	if len(report.Checks) == 0 {
		t.Fatal("expected au moins 1 check dans le rapport")
	}
	// Le premier check doit être "os" (toujours OK)
	if report.Checks[0].Name != "os" {
		t.Errorf("premier check = %q, want os", report.Checks[0].Name)
	}
	if !report.Checks[0].OK {
		t.Error("check os doit être OK")
	}
	// Avec un dir vide, le rapport global doit être KO (pas de DB ni config)
	if report.OK {
		t.Error("expected OK=false avec répertoire vide (pas de DB)")
	}
	// Summary() ne doit pas paniquer
	summary := report.Summary()
	if summary == "" {
		t.Error("Summary() ne doit pas être vide")
	}
}

func TestRunHealthcheck_WithDBs(t *testing.T) {
	dir := t.TempDir()
	// Créer la structure title-aware : data/titles/halo_infinite/warehouse/
	warehouseDir := filepath.Join(dir, "data", "titles", "halo_infinite", "warehouse")
	if err := os.MkdirAll(warehouseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Créer les deux DB DuckDB critiques
	createDBAt(t, filepath.Join(warehouseDir, "shared_matches_v2.duckdb"))
	createDBAt(t, filepath.Join(warehouseDir, "metadata.duckdb"))

	report := RunHealthcheck(HealthcheckOptions{RepoRoot: dir})
	if len(report.Checks) == 0 {
		t.Fatal("expected checks dans le rapport")
	}
	// Vérifier que les checks DuckDB sont présents et OK
	var foundShared bool
	for _, c := range report.Checks {
		if c.Name == "shared_matches_v2" {
			foundShared = true
			if !c.OK {
				t.Errorf("check shared_matches_v2 KO: %q", c.Message)
			}
		}
	}
	if !foundShared {
		t.Error("check shared_matches_v2 absent du rapport")
	}
}
