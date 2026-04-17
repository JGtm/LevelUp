// Package validation — gate_test.go : tests unitaires pour les types Gate (sans DuckDB).
//
//go:build integration

package validation

import (
	"os"
	"strings"
	"testing"
	"time"
)

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
