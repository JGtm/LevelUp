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
	syncv2 "levelup/go-api/internal/sync/v2"
)

// recordingCycleOrch capture les profils reçus par le pipeline V2 (pour vérifier
// qu'un joueur live-only n'y est JAMAIS envoyé).
type recordingCycleOrch struct{ got []syncv2.PlayerProfile }

func (o *recordingCycleOrch) Run(_ context.Context, profiles []syncv2.PlayerProfile) (syncv2.CycleResult, error) {
	o.got = append(o.got, profiles...)
	return syncv2.CycleResult{}, nil
}

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

// TestRunOnceTrigger_V2_RoutesH5ToLiveRunner_NotOrchestrator (D1c étape 1) : sous V2
// (orchestrator câblé), un joueur live-only (Halo 5) est syncé via liveRunner
// (syncPlayer), JAMAIS envoyé à l'orchestrator V2 mono-titre (Infinite). Le joueur
// Infinite, lui, va bien à l'orchestrator. Garde-fou du split de dispatch runOnceV2.
func TestRunOnceTrigger_V2_RoutesH5ToLiveRunner_NotOrchestrator(t *testing.T) {
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// db_profiles.json v3 : le même joueur JGtm sur 2 titres (Infinite + Halo 5).
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{
		"version": "3.0",
		"profiles": {
			"halo_infinite": {"JGtm": {"db_path": "hi", "xuid": "1111111111111111", "waypoint_player": "JGtm"}},
			"halo_5": {"JGtm": {"db_path": "h5", "xuid": "1111111111111111", "waypoint_player": "JGtm"}}
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{RepoRoot: repoRoot, AppSettingsPath: settingsPath, DBProfilesPath: profilesPath}
	s := New(cfg, settings_platform.NewStore(settingsPath), nil, &h5FakePool{has: map[string]bool{"JGtm": true}})

	orch := &recordingCycleOrch{}
	s.cycleOrchestrator = orch

	var liveSlugs []string
	s.liveRunner = func(_ context.Context, slug, gt, xuid string) (DeltaRunner, context.Context, func(), error) {
		liveSlugs = append(liveSlugs, slug)
		return &stubDeltaRunner{}, context.Background(), func() {}, nil
	}
	s.RunnerFactory = func(context.Context, string, string) DeltaRunner {
		t.Fatal("RunnerFactory (engine Infinite V1) ne doit PAS être appelée : V2 gère l'Infinite via l'orchestrator, H5 via liveRunner")
		return nil
	}

	_ = s.RunOnceTrigger(context.Background(), "manual")

	// L'orchestrator V2 ne doit recevoir QUE le joueur Infinite (jamais Halo 5).
	if len(orch.got) != 1 {
		t.Fatalf("orchestrator a reçu %d profils, want 1 (Infinite seul) : %+v", len(orch.got), orch.got)
	}
	if orch.got[0].TitleSlug != "halo_infinite" {
		t.Errorf("orchestrator a reçu %q, want halo_infinite", orch.got[0].TitleSlug)
	}
	for _, p := range orch.got {
		if p.TitleSlug == halo5.TitleSlug {
			t.Errorf("orchestrator V2 a reçu un joueur Halo 5 %q — doit être exclu et routé vers liveRunner", p.Gamertag)
		}
	}
	// Le joueur H5 doit avoir été routé vers liveRunner.
	foundH5 := false
	for _, slug := range liveSlugs {
		if slug == halo5.TitleSlug {
			foundH5 = true
		}
	}
	if !foundH5 {
		t.Errorf("liveRunner non appelé pour Halo 5, liveSlugs=%v", liveSlugs)
	}
}
