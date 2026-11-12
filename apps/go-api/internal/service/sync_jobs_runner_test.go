package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_5/livesync"
	go_sync "levelup/go-api/internal/sync"
)

// TestSyncJobService_runnerFor : la sélection du runner d'une sync initiale d'après le registre des
// titres. Halo 5 (suivi live seul) → runner livesync dédié + contexte portant le SpartanToken (l'adapter
// le lit du contexte) ; Halo Infinite → SyncEngine, contexte non muté.
func TestSyncJobService_runnerFor(t *testing.T) {
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)

	r, ctx := s.runnerFor(context.Background(), "halo_5", "JGtm", "xJG", &domain.HaloTokens{SpartanToken: "v4=token"})
	if _, ok := r.(*livesync.Runner); !ok {
		t.Errorf("halo_5 → *livesync.Runner attendu, got %T", r)
	}
	if tok := ctxkeys.HaloTokens(ctx); tok == nil || tok.SpartanToken != "v4=token" {
		t.Error("le contexte de halo_5 doit porter le SpartanToken (lu par l'adapter live)")
	}

	r2, ctx2 := s.runnerFor(context.Background(), "halo_infinite", "Madina", "x", &domain.HaloTokens{})
	if _, ok := r2.(*go_sync.SyncEngine); !ok {
		t.Errorf("halo_infinite → *go_sync.SyncEngine attendu, got %T", r2)
	}
	if ctxkeys.HaloTokens(ctx2) != nil {
		t.Error("le contexte d'Infinite ne doit PAS être muté (jetons passés en argument)")
	}
}
