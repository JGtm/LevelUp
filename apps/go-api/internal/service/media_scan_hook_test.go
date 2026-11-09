package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/ops"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// NewLiveMediaScanHook lit le dossier configuré dans le store à l'appel : un dossier absent y est
// vu comme une racine indisponible.
func TestNewLiveMediaScanHook_LitLesReglagesDuStore(t *testing.T) {
	root := t.TempDir()
	const gamertag = "LiveHookRootGone"
	settingsPath := filepath.Join(root, "app_settings.json")
	if err := os.WriteFile(settingsPath,
		[]byte(`{"media_captures_base_dir": "`+filepath.ToSlash(filepath.Join(root, "disque-debranche"))+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{RepoRoot: root}
	hook := NewLiveMediaScanHook(cfg, settings_platform.NewStore(settingsPath), gamertag)
	hook(ctxkeys.WithTitleSlug(context.Background(), rootStateTitle))

	if st := ops.GetRootState(rootStateTitle, gamertag); !st.Unavailable {
		t.Errorf("état = %+v, attendu racine indisponible (dossier du store absent)", st)
	}
}

// Sans store : aucun dossier configuré, aucun signal d'indisponibilité, pas de panique.
func TestNewLiveMediaScanHook_SansStore(t *testing.T) {
	root := t.TempDir()
	const gamertag = "LiveHookNoStore"
	hook := NewLiveMediaScanHook(&config.AppConfig{RepoRoot: root}, nil, gamertag)
	hook(ctxkeys.WithTitleSlug(context.Background(), rootStateTitle))
	if st := ops.GetRootState(rootStateTitle, gamertag); st.Unavailable {
		t.Errorf("état = %+v, attendu aucun signal sans dossier configuré", st)
	}
}
