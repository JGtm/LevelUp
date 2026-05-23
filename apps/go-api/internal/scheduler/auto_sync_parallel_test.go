// Package scheduler — auto_sync_parallel_test.go : tests TDD pour la
// parallélisation de RunOnce (plan stabilisation 2026-05-22 §3.4).
//
// Contexte : sur 3 joueurs séquentiels, un cycle complet prend ~15 min
// (5 min/joueur). Avec errgroup.SetLimit(poolSize), les sync-par-joueur
// tournent en parallèle → cycle ~5-8 min.
//
// Audit Agent 1 confirmé : pas de risque deadlock car dblease.leaseMutex
// sérialise les writes shared au niveau Go applicatif, et les writes
// match_participants sont protégés par singleflight (phase 2.3).
//
// **Mode TDD strict** : tests écrits AVANT modif de RunOnce. Le test
// LatencyParallelFasterThanSequential doit ÉCHOUER sur le code séquentiel
// actuel et passer après l'errgroup.

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
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// latencyRunner simule un RunDelta avec latence configurable. Permet de
// mesurer le gain perf parallèle vs séquentiel.
type latencyRunner struct {
	latency time.Duration
	result  domain.SyncResult
	err     error
}

func (m *latencyRunner) RunDelta(ctx context.Context, _ domain.SyncOptions) (domain.SyncResult, error) {
	select {
	case <-time.After(m.latency):
		return m.result, m.err
	case <-ctx.Done():
		return domain.SyncResult{}, ctx.Err()
	}
}

// setup4PlayerParallel prépare un repo avec 4 joueurs configurés, tous
// présents dans le pool, tous avec DB locale. Pool size=4 pour permettre
// la parallélisation complète.
func setup4PlayerParallel(t *testing.T, runner *latencyRunner) *scheduler.AutoSyncScheduler {
	t.Helper()
	repoRoot := t.TempDir()

	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644)

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	_ = os.WriteFile(dbProfilesPath, []byte(`{
		"version":"3.0","admin":"P1","profiles":{"halo_infinite":{
			"P1":{"db_path":"data/titles/halo_infinite/players/P1/stats.duckdb","xuid":"x1","waypoint_player":"P1"},
			"P2":{"db_path":"data/titles/halo_infinite/players/P2/stats.duckdb","xuid":"x2","waypoint_player":"P2"},
			"P3":{"db_path":"data/titles/halo_infinite/players/P3/stats.duckdb","xuid":"x3","waypoint_player":"P3"},
			"P4":{"db_path":"data/titles/halo_infinite/players/P4/stats.duckdb","xuid":"x4","waypoint_player":"P4"}
		}}}`), 0o644)

	touchParallelDB(t, repoRoot, "P1")
	touchParallelDB(t, repoRoot, "P2")
	touchParallelDB(t, repoRoot, "P3")
	touchParallelDB(t, repoRoot, "P4")

	p := &fakePool{
		hasPlayerMap: map[string]bool{"P1": true, "P2": true, "P3": true, "P4": true},
		size:         4,
	}

	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  dbProfilesPath,
		AppSettingsPath: settingsPath,
	}
	s := scheduler.New(cfg, store, &fakeProvider{}, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return runner
	}
	return s
}

// TestRunOnce_Parallel_LatencyFasterThanSequential : 4 joueurs × 400ms latence
// chacun. Séquentiel : 1.6s. Parallèle (pool=4) : ~400ms + overhead.
//
// **Test perf qui ÉCHOUE sur le code séquentiel** et passe après l'errgroup.
func TestRunOnce_Parallel_LatencyFasterThanSequential(t *testing.T) {
	runner := &latencyRunner{
		latency: 400 * time.Millisecond,
		result:  domain.SyncResult{MatchesInserted: 1}, // OK, count Synced++
	}
	s := setup4PlayerParallel(t, runner)

	start := time.Now()
	res := s.RunOnce(context.Background())
	elapsed := time.Since(start)

	// 4 joueurs OK
	if res.Total != 4 {
		t.Errorf("Total = %d, want 4", res.Total)
	}
	if res.Synced != 4 {
		t.Errorf("Synced = %d, want 4 (4 joueurs OK)", res.Synced)
	}

	// Séquentiel théorique : 4 × 400ms = 1.6s
	// Parallèle (pool=4) attendu : ~400ms + overhead = <800ms.
	// Seuil 1000ms pour absorber variance CI.
	const maxAcceptable = 1000 * time.Millisecond
	if elapsed > maxAcceptable {
		t.Errorf("wall-time %v > %v (parallélisation absente ou défaillante — "+
			"séquentiel théorique = 1.6s pour 4×400ms, parallèle attendu < 1s)",
			elapsed, maxAcceptable)
	} else {
		t.Logf("wall-time = %v (gain parallélisation confirmé)", elapsed)
	}
}

// TestRunOnce_Parallel_CountersPreserved : version parallèle conserve les
// counters Total/Synced/Skipped/Failed exactement. Test du contrat.
func TestRunOnce_Parallel_CountersPreserved(t *testing.T) {
	// Latence faible : on teste les compteurs, pas la perf.
	runner := &latencyRunner{
		latency: 1 * time.Millisecond,
		result:  domain.SyncResult{MatchesInserted: 5},
	}
	s := setup4PlayerParallel(t, runner)

	res := s.RunOnce(context.Background())
	if res.Total != 4 {
		t.Errorf("Total = %d, want 4", res.Total)
	}
	if res.Synced != 4 {
		t.Errorf("Synced = %d, want 4", res.Synced)
	}
	if res.Failed != 0 {
		t.Errorf("Failed = %d, want 0", res.Failed)
	}
	if res.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", res.Skipped)
	}
}

// TestRunOnce_Parallel_MixedOutcomes_Counted : si certains joueurs OK, d'autres
// fail, d'autres skip — la version parallèle compte correctement (test race
// sur compteurs).
func TestRunOnce_Parallel_MixedOutcomes_Counted(t *testing.T) {
	repoRoot := t.TempDir()

	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644)

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	_ = os.WriteFile(dbProfilesPath, []byte(`{
		"version":"3.0","admin":"OK1","profiles":{"halo_infinite":{
			"OK1":{"db_path":"data/titles/halo_infinite/players/OK1/stats.duckdb","xuid":"xa","waypoint_player":"OK1"},
			"OK2":{"db_path":"data/titles/halo_infinite/players/OK2/stats.duckdb","xuid":"xb","waypoint_player":"OK2"},
			"FAIL":{"db_path":"data/titles/halo_infinite/players/FAIL/stats.duckdb","xuid":"xc","waypoint_player":"FAIL"},
			"SKIP_NOPOOL":{"db_path":"data/titles/halo_infinite/players/SKIP_NOPOOL/stats.duckdb","xuid":"xd","waypoint_player":"SKIP_NOPOOL"}
		}}}`), 0o644)

	touchParallelDB(t, repoRoot, "OK1")
	touchParallelDB(t, repoRoot, "OK2")
	touchParallelDB(t, repoRoot, "FAIL")
	// SKIP_NOPOOL n'a pas de DB → skip pour cause DB absente. Mais on veut
	// tester le skip via pool absent : on touche aussi sa DB et on l'omet
	// du pool.
	touchParallelDB(t, repoRoot, "SKIP_NOPOOL")

	p := &fakePool{
		hasPlayerMap: map[string]bool{"OK1": true, "OK2": true, "FAIL": true}, // SKIP_NOPOOL absent
		size:         3,
	}

	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: dbProfilesPath, AppSettingsPath: settingsPath}
	s := scheduler.New(cfg, store, &fakeProvider{}, p)

	s.RunnerFactory = func(_ context.Context, gt, _ string) scheduler.DeltaRunner {
		switch gt {
		case "FAIL":
			return &latencyRunner{latency: 5 * time.Millisecond, err: errors.New("forced fail")}
		default:
			return &latencyRunner{latency: 5 * time.Millisecond, result: domain.SyncResult{MatchesInserted: 1}}
		}
	}

	res := s.RunOnce(context.Background())
	if res.Total != 4 {
		t.Errorf("Total = %d, want 4", res.Total)
	}
	if res.Synced != 2 {
		t.Errorf("Synced = %d, want 2 (OK1+OK2)", res.Synced)
	}
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (FAIL)", res.Failed)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (SKIP_NOPOOL pas dans le pool)", res.Skipped)
	}
}

// TestRunOnce_Parallel_CtxCancelDrainsProperly : ctx.Cancel à mi-parcours →
// les joueurs en cours finissent / sont annulés sans crash. Pas de fuite
// goroutine.
func TestRunOnce_Parallel_CtxCancelDrainsProperly(t *testing.T) {
	runner := &latencyRunner{
		latency: 500 * time.Millisecond,
		result:  domain.SyncResult{MatchesInserted: 1},
	}
	s := setup4PlayerParallel(t, runner)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Ne crash pas, retourne un résultat partiel (ou tout failed/skipped).
	res := s.RunOnce(ctx)
	if res.Total != 4 {
		t.Errorf("Total = %d, want 4", res.Total)
	}
	// Pas d'assertion sur Synced/Failed — la race entre cancel et runner peut
	// produire différents états selon le timing. Le critère est : pas de panic,
	// pas de crash, Total reste correct.
}

// touchParallelDB : helper local pour créer un fichier stats.duckdb vide.
// Équivalent de touchPlayerDB du auto_sync_test.go ; on évite la dépendance
// directe pour pouvoir lancer ce fichier indépendamment si besoin.
func touchParallelDB(t *testing.T, repoRoot, gamertag string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stats.duckdb"), []byte{}, 0o644); err != nil {
		t.Fatalf("write stats.duckdb: %v", err)
	}
}
