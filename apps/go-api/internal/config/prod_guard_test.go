// Package config — prod_guard_test.go : garde-fou de démarrage en production.
// Revue P0 2026-06-02 — Validate() refuse une configuration non sûre quand
// LEVELUP_ENV=production, SecurityWarnings() liste les réglages dangereux.
package config

import (
	"strings"
	"testing"
)

func TestValidate_ProductionRejectsUnsafeDefaults(t *testing.T) {
	cfg := &AppConfig{
		Environment:   "production",
		SessionSecret: DefaultSessionSecret,
		AuthMode:      "none",
		CORSOrigins:   []string{"http://localhost:5173"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate devrait refuser une config production non sûre")
	}
	for _, want := range []string{"SESSION_SECRET", "AUTH_MODE", "CORS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("erreur ne mentionne pas %q: %v", want, err)
		}
	}
}

func TestValidate_ProductionAcceptsSafeConfig(t *testing.T) {
	cfg := &AppConfig{
		Environment:   "production",
		SessionSecret: strings.Repeat("a", 40),
		AuthMode:      "xbox",
		CORSOrigins:   []string{"https://app.example.com"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config production sûre rejetée: %v", err)
	}
}

func TestValidate_NonProductionNeverFails(t *testing.T) {
	cfg := &AppConfig{
		Environment:   "", // développement
		SessionSecret: DefaultSessionSecret,
		AuthMode:      "none",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("hors production Validate ne doit jamais échouer: %v", err)
	}
	// Mais les avertissements restent émis (pour le log au boot).
	if len(cfg.SecurityWarnings()) == 0 {
		t.Error("SecurityWarnings devrait lister les réglages non sûrs même en dev")
	}
}

func TestValidate_DemoModeBypassesGuard(t *testing.T) {
	cfg := &AppConfig{
		Environment:   "production",
		DemoMode:      true,
		SessionSecret: DefaultSessionSecret,
		AuthMode:      "none",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DemoMode doit court-circuiter le garde-fou: %v", err)
	}
}

func TestSecurityWarnings_ShortSecretFlagged(t *testing.T) {
	cfg := &AppConfig{
		SessionSecret: "trop-court",
		AuthMode:      "xbox",
		CORSOrigins:   []string{"https://x.example.com"},
	}
	w := cfg.SecurityWarnings()
	if len(w) != 1 || !strings.Contains(w[0], "trop court") {
		t.Errorf("attendu 1 avertissement 'trop court', obtenu: %v", w)
	}
}

func TestSecurityWarnings_SafeConfigEmpty(t *testing.T) {
	cfg := &AppConfig{
		SessionSecret: strings.Repeat("x", 32),
		AuthMode:      "password",
		CORSOrigins:   []string{"https://app.example.com"},
	}
	if w := cfg.SecurityWarnings(); len(w) != 0 {
		t.Errorf("config sûre ne devrait émettre aucun avertissement, obtenu: %v", w)
	}
}

func TestCorsAllLocalhost(t *testing.T) {
	cases := []struct {
		name    string
		origins []string
		want    bool
	}{
		{"vide", nil, true},
		{"localhost seul", []string{"http://localhost:5173", "http://127.0.0.1:5174"}, true},
		{"prod present", []string{"http://localhost:5173", "https://app.example.com"}, false},
		{"prod seul", []string{"https://app.example.com"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &AppConfig{CORSOrigins: tc.origins}
			if got := cfg.corsAllLocalhost(); got != tc.want {
				t.Errorf("corsAllLocalhost(%v) = %v, attendu %v", tc.origins, got, tc.want)
			}
		})
	}
}
