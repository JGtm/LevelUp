package livesync

import (
	"testing"

	"levelup/go-api/internal/config"
	halo5 "levelup/go-api/internal/games/halo_5"
)

func TestRunnerForTitle_Dispatch(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	// halo_5 = runner live-sync dédié (réseau différé → construction sans I/O).
	if r := RunnerForTitle(halo5.TitleSlug, cfg, "JGtm", "xJG"); r == nil {
		t.Errorf("halo_5 doit avoir un runner live-sync dédié")
	}
	// halo_infinite passe par le SyncEngine → pas de runner ici.
	if r := RunnerForTitle("halo_infinite", cfg, "Madina", "x"); r != nil {
		t.Errorf("halo_infinite → RunnerForTitle doit être nil (SyncEngine)")
	}
	if r := RunnerForTitle("titre_inconnu", cfg, "X", "y"); r != nil {
		t.Errorf("titre inconnu → nil")
	}
}

func TestHandlesTitle(t *testing.T) {
	if !HandlesTitle(halo5.TitleSlug) {
		t.Error("HandlesTitle(halo_5) doit être true")
	}
	if HandlesTitle("halo_infinite") {
		t.Error("HandlesTitle(halo_infinite) doit être false (SyncEngine)")
	}
}
