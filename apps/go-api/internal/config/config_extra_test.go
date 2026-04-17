// Package config — config_extra_test.go : tests de Load() et des fonctions
// pure/filesystem non couvertes (SharedDBPath, readXUIDFile, autoDetectRepoRoot).
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Load — chargement depuis variables d'environnement
// ─────────────────────────────────────────────────────────────────────────────

func TestLoad_DefaultValues(t *testing.T) {
	// Effacer les variables d'environnement LevelUp pour ne pas polluer le test
	for _, k := range []string{
		"LEVELUP_REPO_ROOT", "LEVELUP_DEMO_MODE", "LEVELUP_API_HOST",
		"LEVELUP_API_PORT", "LEVELUP_LANG", "LEVELUP_SESSION_SECRET",
		"LEVELUP_CORS_ORIGINS", "LEVELUP_DB_PROFILES", "LEVELUP_APP_SETTINGS",
	} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() inattendu: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg nil")
	}
	// Valeurs par défaut
	if cfg.APIHost != "127.0.0.1" {
		t.Errorf("APIHost = %q, want 127.0.0.1", cfg.APIHost)
	}
	if cfg.APIPort != 8000 {
		t.Errorf("APIPort = %d, want 8000", cfg.APIPort)
	}
	if cfg.Lang != "fr" {
		t.Errorf("Lang = %q, want fr", cfg.Lang)
	}
	if cfg.DemoMode {
		t.Error("DemoMode should be false by default")
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LEVELUP_REPO_ROOT", dir)
	t.Setenv("LEVELUP_API_HOST", "0.0.0.0")
	t.Setenv("LEVELUP_API_PORT", "9090")
	t.Setenv("LEVELUP_LANG", "en")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() inattendu: %v", err)
	}
	if cfg.RepoRoot != dir {
		t.Errorf("RepoRoot = %q, want %q", cfg.RepoRoot, dir)
	}
	if cfg.APIHost != "0.0.0.0" {
		t.Errorf("APIHost = %q, want 0.0.0.0", cfg.APIHost)
	}
	if cfg.APIPort != 9090 {
		t.Errorf("APIPort = %d, want 9090", cfg.APIPort)
	}
	if cfg.Lang != "en" {
		t.Errorf("Lang = %q, want en", cfg.Lang)
	}
	if !cfg.DemoMode {
		t.Error("DemoMode devrait être true")
	}
}

func TestLoad_CORSOrigins(t *testing.T) {
	t.Setenv("LEVELUP_CORS_ORIGINS", "http://localhost:3000,http://localhost:4000")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() inattendu: %v", err)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("CORSOrigins len = %d, want 2", len(cfg.CORSOrigins))
	}
}

func TestLoad_ServerAddr(t *testing.T) {
	t.Setenv("LEVELUP_API_HOST", "127.0.0.1")
	t.Setenv("LEVELUP_API_PORT", "8080")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	addr := cfg.ServerAddr()
	if addr != "127.0.0.1:8080" {
		t.Errorf("ServerAddr() = %q, want 127.0.0.1:8080", addr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SharedDBPath — pure (pas de DuckDB requis)
// ─────────────────────────────────────────────────────────────────────────────

func TestSharedDBPath_DemoMode(t *testing.T) {
	fixturesDir := t.TempDir()
	cfg := &AppConfig{
		DemoMode:        true,
		DemoFixturesDir: fixturesDir,
	}
	p := SharedDBPath(cfg, "")
	if !strings.Contains(p, "shared_matches_v2.duckdb") {
		t.Errorf("SharedDBPath demo = %q, doit contenir shared_matches_v2.duckdb", p)
	}
	if !strings.HasPrefix(p, fixturesDir) {
		t.Errorf("SharedDBPath demo = %q, doit commencer par %q", p, fixturesDir)
	}
}

func TestSharedDBPath_DefaultTitle(t *testing.T) {
	cfg := &AppConfig{
		RepoRoot: t.TempDir(),
		DemoMode: false,
	}
	p := SharedDBPath(cfg, "")
	if !strings.Contains(p, "shared_matches") {
		t.Errorf("SharedDBPath = %q, doit contenir shared_matches", p)
	}
}

func TestSharedDBPath_KnownTitle(t *testing.T) {
	cfg := &AppConfig{
		RepoRoot: t.TempDir(),
		DemoMode: false,
	}
	p := SharedDBPath(cfg, "halo_infinite")
	if p == "" {
		t.Error("SharedDBPath ne doit pas être vide")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// readXUIDFile — filesystem pur
// ─────────────────────────────────────────────────────────────────────────────

func TestReadXUIDFile_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "xuid.txt")
	if err := os.WriteFile(p, []byte("2535469190789936\n"), 0600); err != nil {
		t.Fatal(err)
	}
	xuid, err := readXUIDFile(p)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if xuid != "2535469190789936" {
		t.Errorf("xuid = %q, want 2535469190789936", xuid)
	}
}

func TestReadXUIDFile_Absent(t *testing.T) {
	_, err := readXUIDFile("/nonexistent/xuid.txt")
	if err == nil {
		t.Error("expected error (fichier absent)")
	}
}

func TestReadXUIDFile_Empty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "xuid.txt")
	if err := os.WriteFile(p, []byte("   \n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readXUIDFile(p)
	if err == nil {
		t.Error("expected error (contenu vide)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// autoDetectRepoRoot — teste la branche "marqueur introuvable → fallback"
// ─────────────────────────────────────────────────────────────────────────────

func TestAutoDetectRepoRoot_ReturnsString(t *testing.T) {
	// Ne peut pas contrôler os.Executable() dans les tests,
	// mais on vérifie que la fonction retourne toujours une string non-vide.
	result := autoDetectRepoRoot()
	if result == "" {
		t.Error("autoDetectRepoRoot ne doit pas retourner une chaîne vide")
	}
}
