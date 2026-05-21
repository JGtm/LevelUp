// Package validation — gate_extra_test.go : tests des helpers et du rapport gate.
//
// Couvre les branches accessibles sans DuckDB :
//   - GateReport.Format() (toutes branches)
//   - RunGateCheck4 avec répertoire temporaire (checkBinary, checkDBProfiles,
//     checkDiscordNotify, checkDBAccessible via path absent)
//   - checkBinary, checkDBProfiles, checkDiscordNotify
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GateReport.Format() — cas supplémentaires (basiques couverts dans gate_unit_test.go)

func TestGateReport_Format_ItemWithMessage(t *testing.T) {
	// Vérifie que le message d'un item échoué apparaît dans le rapport
	r := &GateReport{
		GeneratedAt: time.Now(),
		AllPassed:   false,
		Items: []GateItem{
			{ID: "shared-db", Label: "shared DB accessible", Passed: false, Message: "fichier .duckdb absent"},
		},
	}
	out := r.Format()
	if !strings.Contains(out, "fichier .duckdb absent") {
		t.Error("attendu: message de l'item dans le rapport")
	}
	if !strings.Contains(out, "shared-db") {
		t.Error("attendu: ID de l'item")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkBinary
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckBinary_NotFound(t *testing.T) {
	dir := t.TempDir()
	passed, msg := checkBinary(dir)
	if passed {
		t.Error("expected false (binaire absent)")
	}
	if !strings.Contains(msg, "introuvable") {
		t.Errorf("message inattendu: %q", msg)
	}
}

func TestCheckBinary_Found(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "apps", "go-api", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(binDir, "levelup.exe")
	if err := os.WriteFile(binPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	passed, msg := checkBinary(dir)
	if !passed {
		t.Errorf("expected true (binaire présent), msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkDBProfiles
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckDBProfiles_Absent(t *testing.T) {
	passed, msg := checkDBProfiles("/nonexistent/db_profiles.json")
	if passed {
		t.Error("expected false (fichier absent)")
	}
	if !strings.Contains(msg, "absent") {
		t.Errorf("message inattendu: %q", msg)
	}
}

func TestCheckDBProfiles_Empty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(p, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	passed, msg := checkDBProfiles(p)
	if passed {
		t.Error("expected false (fichier vide)")
	}
	if !strings.Contains(msg, "vide") {
		t.Errorf("message inattendu: %q", msg)
	}
}

func TestCheckDBProfiles_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "db_profiles.json")
	content := `{"version":"3.0","profiles":{"halo-infinite":{"p1":{"db_path":"x","xuid":"y"}}}}`
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	passed, msg := checkDBProfiles(p)
	if !passed {
		t.Errorf("expected true (fichier valide), msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// checkDiscordNotify
// ─────────────────────────────────────────────────────────────────────────────

func TestCheckDiscordNotify_ValidEnvVar(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123/abc")
	passed, msg := checkDiscordNotify(t.TempDir())
	if !passed {
		t.Errorf("expected true, msg=%q", msg)
	}
}

func TestCheckDiscordNotify_InvalidEnvVar(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "http://invalid.url")
	passed, msg := checkDiscordNotify(t.TempDir())
	if passed {
		t.Error("expected false (URL invalide)")
	}
	if !strings.Contains(msg, "invalide") {
		t.Errorf("message inattendu: %q", msg)
	}
}

func TestCheckDiscordNotify_NoEnvNoFile(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "")
	dir := t.TempDir()
	passed, msg := checkDiscordNotify(dir)
	if passed {
		t.Error("expected false (pas de config)")
	}
	_ = msg
}

func TestCheckDiscordNotify_InAppSettings(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "")
	dir := t.TempDir()
	content := `{"discord_webhook_url":"https://discord.com/api/webhooks/456/xyz"}`
	if err := os.WriteFile(filepath.Join(dir, "app_settings.json"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	passed, msg := checkDiscordNotify(dir)
	if !passed {
		t.Errorf("expected true (configuré dans app_settings.json), msg=%q", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RunGateCheck4 — exécution complète avec dir temporaire
// ─────────────────────────────────────────────────────────────────────────────

func TestRunGateCheck4_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	dbProfiles := filepath.Join(dir, "db_profiles.json")
	// Pas de binaire, pas de DB, pas de db_profiles.json
	report := RunGateCheck4(t.Context(), GateCheckConfig{
		RepoRoot:       dir,
		DBProfilesPath: dbProfiles,
		Gamertag:       "",
	})
	if report == nil {
		t.Fatal("rapport nil")
	}
	// Doit avoir plusieurs items
	if len(report.Items) == 0 {
		t.Error("expected at least 1 gate item")
	}
	// AllPassed doit être false (aucune DB, aucun binaire)
	if report.AllPassed {
		t.Error("expected AllPassed=false dans un dir vide")
	}
}

func TestRunGateCheck4_WithBinaryAndProfiles(t *testing.T) {
	dir := t.TempDir()

	// Créer un faux binaire
	binDir := filepath.Join(dir, "apps", "go-api", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "levelup.exe"), []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	// Créer un db_profiles.json valide
	dbProfiles := filepath.Join(dir, "db_profiles.json")
	content := `{"version":"3.0","profiles":{"halo-infinite":{"p1":{"db_path":"x","xuid":"y"}}}}`
	if err := os.WriteFile(dbProfiles, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_WEBHOOK_URL", "")
	report := RunGateCheck4(t.Context(), GateCheckConfig{
		RepoRoot:       dir,
		DBProfilesPath: dbProfiles,
		Gamertag:       "",
	})
	if report == nil {
		t.Fatal("rapport nil")
	}
	// Le check sync-binary et db-profiles sont passés
	synced := false
	for _, item := range report.Items {
		if item.ID == "sync-binary" && item.Passed {
			synced = true
		}
	}
	if !synced {
		t.Error("expected sync-binary=passed avec binaire présent")
	}
}
