package sync

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// mockLiveRunner compte les RunDelta — preuve que le pipeline live a tourné.
type mockLiveRunner struct{ called atomic.Int32 }

func (m *mockLiveRunner) RunDelta(context.Context, domain.SyncOptions) (domain.SyncResult, error) {
	m.called.Add(1)
	return domain.SyncResult{MatchesInserted: 2}, nil
}

// Un titre live (factory handled=true) doit être routé vers le runner live, SANS
// toucher au tokenProvider ni à l'engineFactory (sinon fetch Infinite → store h5).
func TestTrigger_RunSync_LiveTitle_RoutesToLiveRunner(t *testing.T) {
	// failingTokenProvider : si GetTokens était appelé, RunSync remonterait son
	// erreur — le succès prouve que le path live court-circuite l'auth engine.
	tp := &failingTokenProvider{err: errors.New("GetTokens ne doit pas être appelé")}
	runner := &mockLiveRunner{}
	var gotSlug, gotGT, gotXUID string
	released := false
	var engineCalls atomic.Int32

	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{}).
		WithEngineFactory(func(context.Context, string, string) *SyncEngine { engineCalls.Add(1); return nil }).
		WithLiveRunnerFactory(func(ctx context.Context, slug, gt, xuid string) (LiveTitleRunner, context.Context, func(), bool, error) {
			gotSlug, gotGT, gotXUID = slug, gt, xuid
			return runner, ctx, func() { released = true }, true, nil
		})

	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_5")
	if err := trigger.RunSync(ctx, "JGtm", "2533", []string{"m1"}); err != nil {
		t.Fatalf("RunSync live: err = %v, want nil", err)
	}
	if runner.called.Load() != 1 {
		t.Errorf("RunDelta live appelé %d fois, want 1", runner.called.Load())
	}
	if engineCalls.Load() != 0 {
		t.Errorf("engineFactory appelée %d fois — ne doit pas l'être pour un titre live", engineCalls.Load())
	}
	if !released {
		t.Error("release du lease non différé")
	}
	if gotSlug != "halo_5" || gotGT != "JGtm" || gotXUID != "2533" {
		t.Errorf("factory args = (%q,%q,%q), want (halo_5,JGtm,2533)", gotSlug, gotGT, gotXUID)
	}
}

// Titre live mais résolution échouée (handled=true, err) → RunSync ÉCHOUE, sans
// repli vers l'engine Infinite (qui corromprait le store h5).
func TestTrigger_RunSync_LiveTitle_FactoryErrorNoEngineFallback(t *testing.T) {
	tp := &mockTokenProvider{}
	var engineCalls atomic.Int32

	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{}).
		WithEngineFactory(func(context.Context, string, string) *SyncEngine { engineCalls.Add(1); return nil }).
		WithLiveRunnerFactory(func(ctx context.Context, slug, gt, xuid string) (LiveTitleRunner, context.Context, func(), bool, error) {
			return nil, ctx, func() {}, true, errors.New("pool indisponible")
		})

	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_5")
	err := trigger.RunSync(ctx, "JGtm", "2533", nil)
	if err == nil {
		t.Fatal("attendu une erreur quand la résolution du runner live échoue")
	}
	if engineCalls.Load() != 0 {
		t.Errorf("aucun fallback engine attendu, engineCalls=%d", engineCalls.Load())
	}
}

// Titre non-live (handled=false) → fall-through vers le path engine inchangé.
// engineFactory retourne nil → erreur explicite : preuve qu'on a bien atteint
// l'engine et que le runner live n'a PAS tourné.
func TestTrigger_RunSync_NonLiveTitle_FallsThroughToEngine(t *testing.T) {
	tp := &mockTokenProvider{}
	runner := &mockLiveRunner{}
	var engineCalls atomic.Int32

	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{}).
		WithEngineFactory(func(context.Context, string, string) *SyncEngine { engineCalls.Add(1); return nil }).
		WithLiveRunnerFactory(func(ctx context.Context, slug, gt, xuid string) (LiveTitleRunner, context.Context, func(), bool, error) {
			return nil, ctx, func() {}, false, nil
		})

	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	err := trigger.RunSync(ctx, "Player1", "1234", nil)
	if err == nil {
		t.Fatal("attendu erreur (engineFactory nil) prouvant le fall-through engine")
	}
	if engineCalls.Load() != 1 {
		t.Errorf("engineFactory devrait être appelée (handled=false), calls=%d", engineCalls.Load())
	}
	if runner.called.Load() != 0 {
		t.Error("le runner live ne doit pas tourner pour un titre non-live")
	}
}
