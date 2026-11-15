package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	go_sync "levelup/go-api/internal/sync"
)

// TestSyncJobService_runnerFor : la sélection du runner d'une sync initiale d'après le constructeur de
// runners de suivi en direct que le câblage injecte. Un titre dont le constructeur rend un runner (suivi
// live seul) → ce runner + contexte portant le SpartanToken (l'adapter le lit du contexte) ; un titre
// dont il rend nil → SyncEngine, contexte non muté. Sans constructeur, tous les titres reçoivent le
// moteur de sync.
func TestSyncJobService_runnerFor(t *testing.T) {
	live := &fakeDeltaRunner{}
	builder := func(titleSlug, _, _ string) DeltaRunner {
		if titleSlug == "titre_live" {
			return live
		}
		return nil
	}
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil).WithLiveRunnerBuilder(builder)

	r, ctx := s.runnerFor(context.Background(), "titre_live", "JGtm", "xJG", &domain.HaloTokens{SpartanToken: "v4=token"})
	if r != DeltaRunner(live) {
		t.Errorf("titre_live → le runner du constructeur attendu, got %T", r)
	}
	if tok := ctxkeys.HaloTokens(ctx); tok == nil || tok.SpartanToken != "v4=token" {
		t.Error("le contexte d'un titre en suivi live doit porter le SpartanToken (lu par l'adapter live)")
	}

	r2, ctx2 := s.runnerFor(context.Background(), "halo_infinite", "Madina", "x", &domain.HaloTokens{})
	if _, ok := r2.(*go_sync.SyncEngine); !ok {
		t.Errorf("halo_infinite → *go_sync.SyncEngine attendu, got %T", r2)
	}
	if ctxkeys.HaloTokens(ctx2) != nil {
		t.Error("le contexte d'un titre servi par le moteur ne doit PAS être muté (jetons passés en argument)")
	}

	// Câblage absent : Halo 5 reçoit le moteur de sync, et
	// son contexte reste intact — jamais le runner ni les jetons d'un autre titre.
	sans := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)
	r3, ctx3 := sans.runnerFor(context.Background(), "halo_5", "JGtm", "xJG", &domain.HaloTokens{SpartanToken: "v4=token"})
	if _, ok := r3.(*go_sync.SyncEngine); !ok {
		t.Errorf("sans constructeur → *go_sync.SyncEngine attendu, got %T", r3)
	}
	if ctxkeys.HaloTokens(ctx3) != nil {
		t.Error("sans constructeur, le contexte ne doit PAS être muté")
	}
}
