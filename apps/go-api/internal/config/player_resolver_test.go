// Tests resolveDemoPlayer — couvre le cas fixture absente / présente.
//
// Régression 2026-04-29 : un process API démarré en LEVELUP_DEMO_MODE=true
// sans fixture stats.duckdb installée renvoyait un IO Error opaque côté
// client (« could not connect to database… cannot open file… »). Le test
// vérifie que :
//  1. fixture absente → erreur explicite avec hint corrective
//  2. fixture présente → resolveDemoPlayer ouvre sans erreur (smoke)
//
// Build tag cgo car GetOrOpen ouvre une connexion DuckDB réelle.

//go:build cgo

package config

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestResolveDemoPlayer_FixtureMissing_ReturnsExplicitError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &AppConfig{
		RepoRoot:        tmpDir,
		DemoMode:        true,
		DemoFixturesDir: filepath.Join(tmpDir, "tests", "fixtures", "ref_player"),
	}

	_, err := ResolvePlayer(context.Background(), cfg, "demo-player", "halo_infinite")
	if err == nil {
		t.Fatal("expected error when demo fixture is missing, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "fixture démo manquante") {
		t.Errorf("error message should mention missing fixture, got: %s", msg)
	}
	if !strings.Contains(msg, "LEVELUP_DEMO_MODE") {
		t.Errorf("error message should mention LEVELUP_DEMO_MODE corrective, got: %s", msg)
	}
	if !strings.Contains(msg, "LEVELUP_DEMO_FIXTURES_DIR") {
		t.Errorf("error message should mention LEVELUP_DEMO_FIXTURES_DIR corrective, got: %s", msg)
	}
}

func TestResolveDemoPlayer_FixturePresent_OpensWithoutError(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureDir := filepath.Join(tmpDir, "tests", "fixtures", "ref_player")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}

	// Créer une fixture stats.duckdb minimale (DuckDB crée le fichier au open).
	statsPath := filepath.Join(fixtureDir, "stats.duckdb")
	db, err := sql.Open("duckdb", statsPath)
	if err != nil {
		t.Fatalf("create fixture stats.duckdb: %v", err)
	}
	if err := db.PingContext(t.Context()); err != nil {
		t.Fatalf("ping fixture: %v", err)
	}
	_ = db.Close()

	// xuid.txt optionnel (fallback hardcodé si absent — pas testé ici,
	// le but est juste de valider que stats.duckdb manquante émet une erreur claire).

	cfg := &AppConfig{
		RepoRoot:        tmpDir,
		DemoMode:        true,
		DemoFixturesDir: fixtureDir,
	}

	// On n'attend pas un succès complet (les autres DBs shared/metadata/aliases
	// sont absentes), mais on attend que le check fixture stats.duckdb passe.
	// L'erreur retournée ne doit PAS être celle de "fixture démo manquante".
	_, err = ResolvePlayer(context.Background(), cfg, "demo-player", "halo_infinite")
	if err != nil && strings.Contains(err.Error(), "fixture démo manquante") {
		t.Errorf("fixture stats.duckdb is present, but resolver still says fixture missing: %s", err.Error())
	}
}
