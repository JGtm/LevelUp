package wire

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games/halo_5/livesync"
)

// TestNewLiveRunnerBuilder : Halo 5 (suivi live seul) reçoit son runner `livesync` ; Halo Infinite
// reçoit une interface nil (pas un *Runner nil typé), donc le moteur de sync.
func TestNewLiveRunnerBuilder(t *testing.T) {
	build := NewLiveRunnerBuilder(&config.AppConfig{RepoRoot: t.TempDir()})

	r := build("halo_5", "JGtm", "xJG")
	if _, ok := r.(*livesync.Runner); !ok {
		t.Errorf("halo_5 → *livesync.Runner attendu, got %T", r)
	}
	if r2 := build("halo_infinite", "Madina", "x"); r2 != nil {
		t.Errorf("halo_infinite → nil attendu, got %T", r2)
	}
}
