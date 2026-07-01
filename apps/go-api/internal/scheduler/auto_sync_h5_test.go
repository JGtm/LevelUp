// Package scheduler — auto_sync_h5_test.go : tests internes du routage des titres
// live-only (Halo 5+) dans le scheduler. Test INTERNE (package scheduler) car il
// injecte le champ non-exporté s.liveRunner (parité avec RunnerFactory côté engine)
// pour exercer syncPlayer sans pool réel ni réseau.
package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/auth/pool"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// h5FakePool : pool.Pool minimal. Acquire n'est jamais appelé (on injecte
// s.liveRunner directement) — il retourne nil pour révéler tout chemin imprévu.
type h5FakePool struct{ has map[string]bool }

func (p *h5FakePool) Acquire(context.Context, pool.AcquirePolicy, string) (*pool.Lease, error) {
	return nil, nil
}
func (p *h5FakePool) Size() int                                                      { return len(p.has) }
func (p *h5FakePool) HasPlayer(gt string) bool                                       { return p.has[gt] }
func (p *h5FakePool) MarkUnhealthy(string, error)                                    {}
func (p *h5FakePool) OnHTTPError(int, time.Duration)                                 {}
func (p *h5FakePool) On429ForToken(string, time.Duration)                            {}
func (p *h5FakePool) AddOrUpdateSource(context.Context, pool.CredentialSource) error { return nil }
func (p *h5FakePool) Close()                                                         {}

func newH5Scheduler(t *testing.T) *AutoSyncScheduler {
	t.Helper()
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: repoRoot, AppSettingsPath: settingsPath}
	return New(cfg, settings_platform.NewStore(settingsPath), nil, &h5FakePool{has: map[string]bool{"JGtm": true}})
}

// stubDeltaRunner enregistre l'appel à RunDelta (preuve que le runner live a tourné).
type stubDeltaRunner struct{ called bool }

func (r *stubDeltaRunner) RunDelta(context.Context, domain.SyncOptions) (domain.SyncResult, error) {
	r.called = true
	return domain.SyncResult{MatchesInserted: 3}, nil
}

// Les titres live-only n'ont pas de player DB → la précondition os.Stat doit être
// court-circuitée (sinon le joueur h5 serait skippé à vie). Le titre Infinite, lui,
// reste soumis au contrôle (DB absente → skip), servant de contrôle négatif.
func TestCheckSyncPreconditions_LiveTitleSkipsPlayerDBCheck(t *testing.T) {
	s := newH5Scheduler(t)
	ctx := context.Background()

	reason, ok := s.checkSyncPreconditions(ctx, domain.PlayerSummary{Gamertag: "JGtm", XUID: "x", TitleSlug: halo5.TitleSlug})
	if !ok {
		t.Fatalf("h5 : précondition devrait passer (skip player-DB), got skip=%q", reason)
	}

	reason, ok = s.checkSyncPreconditions(ctx, domain.PlayerSummary{Gamertag: "JGtm", XUID: "x", TitleSlug: "halo_infinite"})
	if ok {
		t.Fatal("infinite : précondition devrait skip (player DB absente), got ok=true")
	}
	if reason == "" {
		t.Error("infinite : raison de skip vide")
	}
}

// Un joueur h5 doit être routé vers s.liveRunner (pipeline dédié), JAMAIS vers
// l'engine Infinite (RunnerFactory) — sinon fetch Infinite dans le store h5.
func TestSyncPlayer_H5RoutesToLiveRunner(t *testing.T) {
	s := newH5Scheduler(t)
	stub := &stubDeltaRunner{}
	var liveCalled, released bool
	var gotSlug string
	s.liveRunner = func(ctx context.Context, slug, gt, xuid string) (DeltaRunner, context.Context, func(), error) {
		liveCalled, gotSlug = true, slug
		return stub, ctx, func() { released = true }, nil
	}
	s.RunnerFactory = func(context.Context, string, string) DeltaRunner {
		t.Fatal("RunnerFactory (engine Infinite) ne doit PAS être appelée pour un titre live")
		return nil
	}

	out := s.syncPlayer(context.Background(), domain.PlayerSummary{Gamertag: "JGtm", XUID: "x", TitleSlug: halo5.TitleSlug})
	if out != outcomeOK {
		t.Fatalf("outcome = %v, want outcomeOK", out)
	}
	if !liveCalled {
		t.Error("liveRunner non appelé")
	}
	if gotSlug != halo5.TitleSlug {
		t.Errorf("slug transmis = %q, want %q", gotSlug, halo5.TitleSlug)
	}
	if !stub.called {
		t.Error("RunDelta du runner live non appelé")
	}
	if !released {
		t.Error("release du lease pool non différé")
	}
}

// Si la résolution du runner live échoue (pool indispo), le tick ÉCHOUE — pas de
// fallback silencieux vers l'engine (qui corromprait le store h5).
func TestSyncPlayer_H5LiveRunnerError_Failed(t *testing.T) {
	s := newH5Scheduler(t)
	s.liveRunner = func(ctx context.Context, slug, gt, xuid string) (DeltaRunner, context.Context, func(), error) {
		return nil, ctx, func() {}, errors.New("pool indisponible")
	}
	s.RunnerFactory = func(context.Context, string, string) DeltaRunner {
		t.Fatal("engine ne doit PAS être appelé en repli")
		return nil
	}

	out := s.syncPlayer(context.Background(), domain.PlayerSummary{Gamertag: "JGtm", XUID: "x", TitleSlug: halo5.TitleSlug})
	if out != outcomeFailed {
		t.Fatalf("outcome = %v, want outcomeFailed", out)
	}
}
