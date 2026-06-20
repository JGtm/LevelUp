package handlers

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_5/livesync"
	go_sync "levelup/go-api/internal/sync"
)

// TestSyncHandler_runnerFor : la sélection registry-driven du runner de /sync/initial.
// Halo 5 (live-only) → runner livesync dédié + ctx portant le SpartanToken (l'adapter
// le lit du ctx) ; Halo Infinite → SyncEngine, ctx non muté.
func TestSyncHandler_runnerFor(t *testing.T) {
	h := &SyncHandler{cfg: &config.AppConfig{RepoRoot: t.TempDir()}}

	r, ctx := h.runnerFor(context.Background(), "halo_5", "JGtm", "xJG",
		&domain.HaloTokens{SpartanToken: "v4=token"})
	if _, ok := r.(*livesync.Runner); !ok {
		t.Errorf("halo_5 → *livesync.Runner attendu, got %T", r)
	}
	if tok := ctxkeys.HaloTokens(ctx); tok == nil || tok.SpartanToken != "v4=token" {
		t.Error("ctx h5 doit porter le SpartanToken (lu par l'adapter live)")
	}

	r2, ctx2 := h.runnerFor(context.Background(), "halo_infinite", "Madina", "x",
		&domain.HaloTokens{})
	if _, ok := r2.(*go_sync.SyncEngine); !ok {
		t.Errorf("halo_infinite → *go_sync.SyncEngine attendu, got %T", r2)
	}
	if ctxkeys.HaloTokens(ctx2) != nil {
		t.Error("ctx Infinite ne doit PAS être muté (tokens passés en argument)")
	}
}
