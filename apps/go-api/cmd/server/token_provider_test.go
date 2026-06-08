// Package main — token_provider_test.go : sélection du TokenProvider (PR-D).
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/settings"
)

// writeAuthProviderSettings écrit un app_settings.json minimal avec auth_provider.
func newSettingsStoreWithProvider(t *testing.T, provider string) *settings.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	body := map[string]any{}
	if provider != "" {
		body["auth_provider"] = provider
	}
	data, _ := json.Marshal(body)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return settings.NewStore(path)
}

func TestBuildTokenProvider_DefaultIsSISU(t *testing.T) {
	cases := []struct {
		name     string
		provider string
	}{
		{"vide -> SISU", ""},
		{"sisu explicite -> SISU", "sisu"},
		{"inconnu -> SISU (défaut)", "banane"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := buildTokenProvider(newSettingsStoreWithProvider(t, c.provider))
			if _, ok := p.(*auth.SISUProvider); !ok {
				t.Errorf("attendu *SISUProvider, got %T", p)
			}
		})
	}
}

func TestBuildTokenProvider_MsalExplicit(t *testing.T) {
	p := buildTokenProvider(newSettingsStoreWithProvider(t, "msal"))
	if _, ok := p.(*auth.MSALProvider); !ok {
		t.Errorf("attendu *MSALProvider pour auth_provider=msal, got %T", p)
	}
}
