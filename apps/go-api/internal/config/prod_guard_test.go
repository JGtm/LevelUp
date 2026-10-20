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

// TestIsExposedDeployment : l'instance est-elle réellement joignable ? Pilote le
// NIVEAU du journal de démarrage des réglages non sûrs (WARN si exposée, INFO
// sinon) — jamais le garde-fou fail-fast, qui ne dépend que de la production.
func TestIsExposedDeployment(t *testing.T) {
	cases := []struct {
		name string
		host string
		env  string
		want bool
	}{
		{"dev local par défaut", "127.0.0.1", "", false},
		{"dev local, env development", "127.0.0.1", "development", false},
		{"dev local, casse indifférente", "127.0.0.1", "Development", false},
		{"localhost", "localhost", "", false},
		{"IPv6 loopback", "::1", "", false},
		{"IPv6 loopback entre crochets", "[::1]", "", false},
		{"boucle 127.0.0.2", "127.0.0.2", "", false},
		{"hôte vide = toutes les interfaces", "", "", true},
		{"0.0.0.0", "0.0.0.0", "", true},
		{"IP de LAN", "192.168.1.20", "", true},
		{"IPv6 non loopback", "::", "", true},
		{"env production même en loopback", "127.0.0.1", "production", true},
		{"env staging même en loopback", "127.0.0.1", "staging", true},
		{"env avec espaces", "127.0.0.1", "  ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &AppConfig{APIHost: tc.host, Environment: tc.env}
			if got := cfg.IsExposedDeployment(); got != tc.want {
				t.Errorf("IsExposedDeployment(host=%q, env=%q) = %v, want %v", tc.host, tc.env, got, tc.want)
			}
		})
	}
}

// TestIsExposedDeployment_NAffectePasValidate — RATCHET. Le refus de démarrer en
// production ne dépend QUE de LEVELUP_ENV : l'hôte d'écoute ne doit jamais le
// relâcher (une prod qui écoute en loopback derrière un proxy reste une prod).
func TestIsExposedDeployment_NAffectePasValidate(t *testing.T) {
	unsafe := func(host string) *AppConfig {
		return &AppConfig{
			Environment:   "production",
			APIHost:       host,
			SessionSecret: DefaultSessionSecret,
			AuthMode:      "none",
			CORSOrigins:   []string{"http://localhost:5173"},
		}
	}
	for _, host := range []string{"127.0.0.1", "localhost", "::1", "", "0.0.0.0"} {
		if err := unsafe(host).Validate(); err == nil {
			t.Errorf("Validate accepte une prod non sûre avec APIHost=%q — le garde-fou a été relâché", host)
		}
	}
	// Et hors production, Validate reste permissif quel que soit l'hôte.
	for _, host := range []string{"127.0.0.1", "0.0.0.0"} {
		cfg := unsafe(host)
		cfg.Environment = ""
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate refuse hors production avec APIHost=%q: %v", host, err)
		}
	}
}
