// Package validation — gate_test.go : tests unitaires pour les types Gate (sans DuckDB).
//
//go:build integration

package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGateReport_Format_AllPassed(t *testing.T) {
	report := &GateReport{
		GeneratedAt: time.Now(),
		AllPassed:   true,
		Items: []GateItem{
			{ID: "G1", Label: "DB accessible", Passed: true, Message: ""},
			{ID: "G2", Label: "Tables présentes", Passed: true, Message: ""},
		},
	}

	out := report.Format()
	if !strings.Contains(out, "✅") {
		t.Error("expected ✅ for passed items")
	}
	if !strings.Contains(out, "G1") || !strings.Contains(out, "G2") {
		t.Error("expected item IDs in output")
	}
}

func TestGateReport_Format_WithFailure(t *testing.T) {
	report := &GateReport{
		GeneratedAt: time.Now(),
		AllPassed:   false,
		Items: []GateItem{
			{ID: "G1", Label: "DB accessible", Passed: false, Message: "fichier introuvable"},
		},
	}

	out := report.Format()
	if !strings.Contains(out, "❌") {
		t.Error("expected ❌ for failed item")
	}
	if !strings.Contains(out, "fichier introuvable") {
		t.Error("expected failure message in output")
	}
}

func TestGateItem_Structure(t *testing.T) {
	item := GateItem{
		ID:      "TEST-1",
		Label:   "Test item",
		Passed:  true,
		Message: "tout va bien",
	}

	if item.ID != "TEST-1" {
		t.Errorf("ID mismatch: %s", item.ID)
	}
	if !item.Passed {
		t.Error("expected Passed=true")
	}
}

func TestTableComparison_StatusConstants(t *testing.T) {
	// Vérifier que les constantes de statut sont cohérentes
	validStatuses := map[string]bool{
		statusOK:      true,
		statusWarn:    true,
		statusMissGo:  true,
		statusMissPy:  true,
		statusDiverge: true,
	}
	for s := range validStatuses {
		if s == "" {
			t.Error("status constant should not be empty")
		}
	}
}

// ---------------------------------------------------------------------------
// checkBinary — pas de DuckDB requis
// ---------------------------------------------------------------------------

func TestCheckBinary_NotFound(t *testing.T) {
	ok, msg := checkBinary("/nonexistent/root")
	if ok {
		t.Error("expected false for nonexistent root")
	}
	if !strings.Contains(msg, "introuvable") {
		t.Errorf("expected 'introuvable' message, got: %s", msg)
	}
}

func TestCheckBinary_Found(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "apps", "go-api", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Créer un faux binaire
	if err := os.WriteFile(filepath.Join(binDir, "levelup.exe"), []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	ok, msg := checkBinary(dir)
	if !ok {
		t.Errorf("expected true, got false: %s", msg)
	}
}

// ---------------------------------------------------------------------------
// checkDBProfiles — pas de DuckDB requis
// ---------------------------------------------------------------------------

func TestCheckDBProfiles_Missing(t *testing.T) {
	ok, msg := checkDBProfiles("/nonexistent/db_profiles.json")
	if ok {
		t.Error("expected false for missing file")
	}
	if !strings.Contains(msg, "absent") {
		t.Errorf("expected 'absent' message, got: %s", msg)
	}
}

func TestCheckDBProfiles_Empty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, _ := checkDBProfiles(p)
	if ok {
		t.Error("expected false for empty profiles")
	}
}

func TestCheckDBProfiles_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(p, []byte(`{"players": [{"gamertag": "test"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	ok, _ := checkDBProfiles(p)
	if !ok {
		t.Error("expected true for valid profiles")
	}
}

// ---------------------------------------------------------------------------
// checkDiscordNotify — pas de DuckDB requis
// ---------------------------------------------------------------------------

func TestCheckDiscordNotify_NoConfig(t *testing.T) {
	// Sauvegarder et nettoyer l'env
	prev := os.Getenv("DISCORD_WEBHOOK_URL")
	os.Unsetenv("DISCORD_WEBHOOK_URL")
	defer os.Setenv("DISCORD_WEBHOOK_URL", prev)

	ok, _ := checkDiscordNotify("/nonexistent")
	if ok {
		t.Error("expected false with no config")
	}
}
