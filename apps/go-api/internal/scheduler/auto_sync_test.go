// Package scheduler_test — auto_sync_test.go : tests unitaires du scheduler.
//
// Suite à la migration AutoSyncScheduler→Pool, les tests qui injectaient un
// TokenReader/EngineFactory custom (mock provider → mock RunDelta) ont été
// supprimés : ils dépendaient d'une couche d'abstraction qui n'existe plus.
// Le scheduler délègue maintenant entièrement au Pool, et les tests
// d'intégration du pipeline complet vivent dans cmd/levelup/cmd_sync.go
// (tests live) et internal/sync/pooled_client_test.go.
//
// Ce qui reste testable unitairement et qui est couvert ici :
//   - Helpers d'intervalle (resolveInterval / intervalFromHours).
//   - syncPlayer chemins de skip (pool nil, pool.HasPlayer=false, watcher actif,
//     DB joueur absente).
//   - Snapshot — récupération thread-safe du dernier cycle.
//   - Run() — arrêt propre sur ctx.Done().
package scheduler_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// fakePool implémente pool.Pool minimalement pour tester les chemins skip
// du scheduler. Acquire n'est pas appelé dans les tests skip (le scheduler
// court-circuite via HasPlayer avant) — on retourne nil pour éviter de
// faux positifs si jamais le test passe quand même par là.
type fakePool struct {
	hasPlayerMap map[string]bool
	size         int
}

func (m *fakePool) Acquire(_ context.Context, _ pool.AcquirePolicy, _ string) (*pool.Lease, error) {
	return nil, nil
}
func (m *fakePool) Size() int                               { return m.size }
func (m *fakePool) HasPlayer(gt string) bool                { return m.hasPlayerMap[gt] }
func (m *fakePool) MarkUnhealthy(_ string, _ error)         {}
func (m *fakePool) OnHTTPError(_ int, _ time.Duration)      {}
func (m *fakePool) On429ForToken(_ string, _ time.Duration) {}
func (m *fakePool) AddOrUpdateSource(_ context.Context, _ pool.CredentialSource) error {
	return nil
}
func (m *fakePool) Close() {}

// fakeActivityChecker implémente PlayerActivityChecker pour les tests.
type fakeActivityChecker struct {
	activeGamertags map[string]bool
}

func (f *fakeActivityChecker) IsPlayerActive(gt string) bool {
	return f.activeGamertags[gt]
}

// fakeProvider est utilisé uniquement comme placeholder pour scheduler.New.
// Le scheduler ne l'appelle jamais directement — c'est le SyncEngine
// (non-mockable ici) qui pourrait l'utiliser pour les achievements.
type fakeProvider struct{}

var _ auth.TokenProvider = (*fakeProvider)(nil)

func (f *fakeProvider) InitDeviceFlow(_ context.Context) (auth.DeviceFlow, error) {
	return nil, nil
}
func (f *fakeProvider) TryOAuthRefresh(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (f *fakeProvider) TryOAuthRefreshWithRotation(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (f *fakeProvider) Exchange(_ context.Context, _ string) (*auth.ExchangeResult, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newSchedulerForTest construit un AutoSyncScheduler isolé : repoRoot temp,
// settings minimaux, pool optionnel, provider stub.
func newSchedulerForTest(
	t *testing.T,
	repoRoot string,
	tokenPool pool.Pool,
) *scheduler.AutoSyncScheduler {
	t.Helper()

	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	settingsJSON := `{
		"spnkr_auto_sync_enabled": true,
		"spnkr_auto_sync_interval_hours": 1
	}`
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o644); err != nil {
		t.Fatalf("écriture settings: %v", err)
	}

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(dbProfilesPath, []byte(`{
		"version":"3.0",
		"admin":"Player1",
		"profiles":{"halo_infinite":{"Player1":{"db_path":"data/titles/halo_infinite/players/Player1/stats.duckdb","xuid":"xuid-1","waypoint_player":"Player1"}}}
	}`), 0o644); err != nil {
		t.Fatalf("écriture db_profiles: %v", err)
	}

	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  dbProfilesPath,
		AppSettingsPath: settingsPath,
	}
	return scheduler.New(cfg, store, &fakeProvider{}, tokenPool)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRunOnce_PoolNil_AllSkipped(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil)
	res := s.RunOnce(context.Background())
	if res.Total != 1 {
		t.Errorf("Total = %d, want 1", res.Total)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (pool nil → tous skipped)", res.Skipped)
	}
	if res.Synced != 0 || res.Failed != 0 {
		t.Errorf("Synced=%d Failed=%d, want 0/0", res.Synced, res.Failed)
	}

	snap := s.Snapshot()
	if len(snap.Players) != 1 {
		t.Fatalf("snapshot.Players len = %d, want 1", len(snap.Players))
	}
	if snap.Players[0].Outcome != "skipped" {
		t.Errorf("Outcome = %q, want skipped", snap.Players[0].Outcome)
	}
	if snap.Players[0].Reason == "" {
		t.Error("Reason vide — devrait expliquer pourquoi skipped")
	}
	if snap.PoolSize != 0 {
		t.Errorf("PoolSize = %d, want 0 (pool nil)", snap.PoolSize)
	}
}

func TestRunOnce_PlayerNotInPool_Skipped(t *testing.T) {
	p := &fakePool{hasPlayerMap: map[string]bool{}, size: 0}
	s := newSchedulerForTest(t, t.TempDir(), p)
	res := s.RunOnce(context.Background())
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (joueur absent du pool)", res.Skipped)
	}

	snap := s.Snapshot()
	if snap.Players[0].Outcome != "skipped" || snap.Players[0].Reason == "" {
		t.Errorf("snapshot player = %+v, want skipped+reason non vide", snap.Players[0])
	}
}

func TestRunOnce_PlayerDBAbsent_Skipped(t *testing.T) {
	// Player1 est dans le pool mais sa DB n'existe pas → skip.
	repoRoot := t.TempDir()
	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	res := s.RunOnce(context.Background())
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (DB absente)", res.Skipped)
	}

	snap := s.Snapshot()
	if snap.Players[0].Reason == "" {
		t.Error("Reason vide")
	}
}

func TestRunOnce_ActivityChecker_SkipsActivePlayer(t *testing.T) {
	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, t.TempDir(), p)
	s.ActivityChecker = &fakeActivityChecker{activeGamertags: map[string]bool{"Player1": true}}

	res := s.RunOnce(context.Background())
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (watcher actif)", res.Skipped)
	}

	snap := s.Snapshot()
	if snap.Players[0].Reason == "" || snap.Players[0].Outcome != "skipped" {
		t.Errorf("snapshot player = %+v", snap.Players[0])
	}
}

func TestSnapshot_EmptyBeforeRun(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil)
	snap := s.Snapshot()
	if snap.LastCycleResult != nil {
		t.Error("LastCycleResult devrait être nil avant tout RunOnce")
	}
	if len(snap.Players) != 0 {
		t.Errorf("Players len = %d, want 0", len(snap.Players))
	}
}

func TestRun_CancelCtxStops(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
		// OK : Run s'est terminé proprement
	case <-time.After(2 * time.Second):
		t.Error("Run n'a pas retourné dans le délai")
	}
}

func TestIntervalDefaults_AppliedFromSettings(t *testing.T) {
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_interval_minutes":15}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{RepoRoot: repoRoot, AppSettingsPath: settingsPath}
	s := scheduler.New(cfg, store, &fakeProvider{}, nil)

	got := s.CurrentInterval()
	want := 15 * time.Minute
	if got != want {
		t.Errorf("CurrentInterval = %v, want %v", got, want)
	}
}

// =============================================================================
// Tests outcome=ok/failed/mixed via RunnerFactory mockable
//
// Pour tester les chemins après les préconditions (skip), on injecte un
// RunnerFactory qui retourne un mockRunner avec un SyncResult / err configurés.
// On crée aussi un fichier stats.duckdb vide pour que la précondition
// "DB joueur présente" passe (le mock court-circuite tout accès DuckDB réel).
// =============================================================================

// mockRunner satisfait scheduler.DeltaRunner pour les tests.
type mockRunner struct {
	result domain.SyncResult
	err    error
}

func (m *mockRunner) RunDelta(_ context.Context, _ domain.SyncOptions) (domain.SyncResult, error) {
	return m.result, m.err
}

// touchPlayerDB crée un fichier stats.duckdb vide pour le gamertag indiqué
// (uniquement pour passer la précondition os.Stat dans syncPlayer ; le mock
// RunnerFactory n'utilise pas du tout le fichier).
func touchPlayerDB(t *testing.T, repoRoot, gamertag string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	dbPath := filepath.Join(dir, "stats.duckdb")
	if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
		t.Fatalf("write %s: %v", dbPath, err)
	}
}

func TestRunOnce_RunnerOK_Synced(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 3, MatchesSkipped: 2, MedalsInserted: 12}}
	}

	res := s.RunOnce(context.Background())
	if res.Synced != 1 || res.Failed != 0 || res.Skipped != 0 {
		t.Errorf("counters = (Synced=%d Failed=%d Skipped=%d), want (1, 0, 0)", res.Synced, res.Failed, res.Skipped)
	}

	snap := s.Snapshot()
	if len(snap.Players) != 1 {
		t.Fatalf("snapshot Players len = %d", len(snap.Players))
	}
	d := snap.Players[0]
	if d.Outcome != "ok" {
		t.Errorf("Outcome = %q, want ok", d.Outcome)
	}
	if d.MatchesInserted != 3 || d.MatchesSkipped != 2 || d.MedalsInserted != 12 {
		t.Errorf("compteurs propagés incorrects: %+v", d)
	}
}

func TestRunOnce_RunnerFailed_Counted(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{err: errors.New("API Halo 500 — service unavailable")}
	}

	res := s.RunOnce(context.Background())
	if res.Failed != 1 || res.Synced != 0 || res.Skipped != 0 {
		t.Errorf("counters = (Synced=%d Failed=%d Skipped=%d), want (0, 1, 0)", res.Synced, res.Failed, res.Skipped)
	}

	snap := s.Snapshot()
	d := snap.Players[0]
	if d.Outcome != "failed" {
		t.Errorf("Outcome = %q, want failed", d.Outcome)
	}
	if d.FirstError == "" {
		t.Error("FirstError vide alors que RunDelta a renvoyé une erreur")
	}
	if d.Reason == "" {
		t.Error("Reason vide")
	}
}

func TestRunOnce_RunnerPartialWarnings_OkWithErrors(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{
			MatchesInserted: 1,
			Errors:          []string{"fetchMatchData(x): timeout", "fetchMatchData(y): 500"},
		}}
	}

	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (partial succès reste outcome=ok)", res.Synced)
	}

	snap := s.Snapshot()
	d := snap.Players[0]
	if d.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", d.ErrorCount)
	}
	if d.FirstError == "" {
		t.Error("FirstError vide alors qu'il y a 2 erreurs partielles")
	}
}

func TestRunOnce_RunnerOK_ZeroInserted_OkWithDifferentReason(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0, MatchesSkipped: 25}}
	}

	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1", res.Synced)
	}

	snap := s.Snapshot()
	d := snap.Players[0]
	if d.Outcome != "ok" {
		t.Errorf("Outcome = %q, want ok", d.Outcome)
	}
	// La raison doit différencier le cas "0 nouveau match" du cas "N insérés".
	if d.Reason == "" {
		t.Error("Reason vide")
	}
}

func TestRunOnce_MultiPlayer_MixedOutcomes(t *testing.T) {
	repoRoot := t.TempDir()

	// db_profiles.json avec 3 joueurs : Alice, Bob, Carol.
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644)

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	_ = os.WriteFile(dbProfilesPath, []byte(`{
		"version":"3.0","admin":"Alice","profiles":{"halo_infinite":{
			"Alice":{"db_path":"data/titles/halo_infinite/players/Alice/stats.duckdb","xuid":"xa","waypoint_player":"Alice"},
			"Bob":{"db_path":"data/titles/halo_infinite/players/Bob/stats.duckdb","xuid":"xb","waypoint_player":"Bob"},
			"Carol":{"db_path":"data/titles/halo_infinite/players/Carol/stats.duckdb","xuid":"xc","waypoint_player":"Carol"}
		}}}`), 0o644)

	touchPlayerDB(t, repoRoot, "Alice")
	touchPlayerDB(t, repoRoot, "Bob")
	// Carol n'a pas de DB → skip

	p := &fakePool{
		hasPlayerMap: map[string]bool{"Alice": true, "Bob": true}, // Carol absente du pool
		size:         2,
	}

	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: dbProfilesPath, AppSettingsPath: settingsPath}
	s := scheduler.New(cfg, store, &fakeProvider{}, p)

	// Alice → OK, Bob → fail. Carol → skip (pas dans le pool).
	s.RunnerFactory = func(_ context.Context, gt, _ string) scheduler.DeltaRunner {
		if gt == "Alice" {
			return &mockRunner{result: domain.SyncResult{MatchesInserted: 5}}
		}
		return &mockRunner{err: errors.New("Bob API 500")}
	}

	res := s.RunOnce(context.Background())
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (Alice)", res.Synced)
	}
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (Bob)", res.Failed)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (Carol pas dans le pool)", res.Skipped)
	}

	snap := s.Snapshot()
	if len(snap.Players) != 3 {
		t.Fatalf("snapshot Players len = %d, want 3", len(snap.Players))
	}

	byGT := map[string]scheduler.PlayerOutcomeDetail{}
	for _, d := range snap.Players {
		byGT[d.Gamertag] = d
	}
	if byGT["Alice"].Outcome != "ok" {
		t.Errorf("Alice outcome = %q, want ok", byGT["Alice"].Outcome)
	}
	if byGT["Bob"].Outcome != "failed" {
		t.Errorf("Bob outcome = %q, want failed", byGT["Bob"].Outcome)
	}
	if byGT["Carol"].Outcome != "skipped" {
		t.Errorf("Carol outcome = %q, want skipped", byGT["Carol"].Outcome)
	}
}

func TestRunOnce_ActivityChecker_SyncsIdlePlayer(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.ActivityChecker = &fakeActivityChecker{activeGamertags: map[string]bool{}} // aucun actif
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 1}}
	}

	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (Player1 idle → sync OK)", res.Synced)
	}
}

func TestRunOnce_RunnerFactoryNil_Failed(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	// Forcer la factory à retourner nil pour exercer le cas dégénéré.
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return nil
	}

	res := s.RunOnce(context.Background())
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (factory retourne nil)", res.Failed)
	}

	snap := s.Snapshot()
	d := snap.Players[0]
	if d.Outcome != "failed" {
		t.Errorf("Outcome = %q, want failed", d.Outcome)
	}
}

// Vérifier que les types domain.PlayerSummary + SyncResult restent compatibles
// (compile-time check qui assure que le scheduler peut être lié au reste).
var _ = domain.PlayerSummary{}
var _ = domain.SyncResult{}
