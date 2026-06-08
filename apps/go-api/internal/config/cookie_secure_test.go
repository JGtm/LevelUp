// Package config — cookie_secure_test.go : parsing LEVELUP_COOKIE_SECURE +
// avertissement de sécurité associé (fix onboarding 2026-06-08).
package config

import (
	"strings"
	"testing"
)

func TestParseCookieSecureMode(t *testing.T) {
	cases := map[string]string{
		"":       "auto",
		"auto":   "auto",
		"banane": "auto", // valeur inconnue → auto (sûr)
		"true":   "true",
		"1":      "true",
		"yes":    "true",
		"on":     "true",
		"TRUE":   "true",
		"false":  "false",
		"0":      "false",
		"no":     "false",
		"off":    "false",
		" False": "false", // trim + casse
	}
	for in, want := range cases {
		if got := parseCookieSecureMode(in); got != want {
			t.Errorf("parseCookieSecureMode(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestSecurityWarnings_CookieSecureFalse(t *testing.T) {
	cfg := &AppConfig{
		SessionSecret: strings.Repeat("a", 40),
		AuthMode:      "xbox",
		CORSOrigins:   []string{"https://app.example.com"},
		CookieSecure:  "false",
	}
	var found bool
	for _, w := range cfg.SecurityWarnings() {
		if strings.Contains(w, "LEVELUP_COOKIE_SECURE=false") {
			found = true
		}
	}
	if !found {
		t.Errorf("SecurityWarnings devrait avertir sur COOKIE_SECURE=false, got %v", cfg.SecurityWarnings())
	}
}

func TestSecurityWarnings_CookieSecureAutoNoWarning(t *testing.T) {
	cfg := &AppConfig{
		SessionSecret: strings.Repeat("a", 40),
		AuthMode:      "xbox",
		CORSOrigins:   []string{"https://app.example.com"},
		CookieSecure:  "auto",
	}
	for _, w := range cfg.SecurityWarnings() {
		if strings.Contains(w, "LEVELUP_COOKIE_SECURE") {
			t.Errorf("aucun warning COOKIE_SECURE attendu en mode auto, got %q", w)
		}
	}
}
