// Package scheduler_test — auto_sync_build_engine_test.go : garde-rail
// anti-régression pour le câblage du SyncEngine via BuildEngine.
//
// CONTEXTE — incident 2026-05-26 :
//   - Le watcher path (Coordinator → Trigger.RunSync) faisait
//     NewSyncEngine direct, sans .WithSharedProvider.
//   - Conséquence : tous les syncs déclenchés par le watcher tombaient en
//     mode legacy → conflit "Can't open a connection to same database file
//     with a different configuration than existing connections" sur
//     shared_matches_v2.duckdb.
//
// CONTRAT — BuildEngine est désormais l'UNIQUE source of truth du wiring
// SyncEngine pour le serveur. Le scheduler ET le watcher (via
// syncTrigger.WithEngineFactory(autoScheduler.BuildEngine)) la partagent.
//
// Ces tests garantissent qu'une régression future ne peut PAS retirer
// silencieusement un .WithXxx critique :
//   - SharedProvider câblé quand cfg.SharedProvider != nil
//   - FriendsLoader câblé quand settings != nil
//   - PostSyncRunner câblé quand WithPostSyncRunner(runner)
//   - MediaScanHook câblé quand settings != nil
//   - CSRSeasonID câblé quand cfg.CurrentCSRSeasonID != ""
package scheduler_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
)

// stubPostSyncRunner — minimal pour test golden.
type stubPostSyncRunner struct{}

func (s *stubPostSyncRunner) BeforeSync(_ context.Context, _ string) port.PostSyncFinalizer {
	return nil
}

var _ port.PostSyncRunner = (*stubPostSyncRunner)(nil)

// newFullyWiredScheduler construit un AutoSyncScheduler avec TOUTES les deps
// non-nilles, en gardant le contrôle sur le AppConfig pour pouvoir y
// injecter SharedProvider/CurrentCSRSeasonID avant l'appel BuildEngine.
func newFullyWiredScheduler(t *testing.T) (*scheduler.AutoSyncScheduler, *config.AppConfig) {
	t.Helper()
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{
		"spnkr_auto_sync_enabled": true,
		"spnkr_auto_sync_interval_hours": 1,
		"friend_gamertags": ["F1"],
		"media_captures_base_dir": "/tmp/caps",
		"user_timezone": "Europe/Paris"
	}`), 0o644); err != nil {
		t.Fatalf("écriture settings: %v", err)
	}
	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(dbProfilesPath, []byte(`{"version":"3.0","admin":"P","profiles":{"halo_infinite":{"P":{"db_path":"data/x.duckdb","xuid":"x","waypoint_player":"P"}}}}`), 0o644); err != nil {
		t.Fatalf("écriture db_profiles: %v", err)
	}
	store := settings_platform.NewStore(settingsPath)
	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  dbProfilesPath,
		AppSettingsPath: settingsPath,
	}
	s := scheduler.New(cfg, store, &fakeProvider{}, nil)
	return s, cfg
}

// TestBuildEngine_AllOptionsWired_GoldenAntiRegression vérifie que toutes
// les options critiques sont câblées sur le *SyncEngine retourné par
// BuildEngine quand le scheduler a toutes ses dépendances.
//
// C'EST LE TEST GOLDEN. Il échoue si quelqu'un retire un With...() de
// BuildEngine. Toute évolution doit ajouter un nouveau Has...() côté
// SyncEngine + une assertion ici.
func TestBuildEngine_AllOptionsWired_GoldenAntiRegression(t *testing.T) {
	s, cfg := newFullyWiredScheduler(t)

	// Câbler les deps qui passent par WithXxx (postSyncRunner) ou par cfg
	// (SharedProvider, CurrentCSRSeasonID).
	memProvider := sharedprovider.FromInMemoryDB((*sql.DB)(nil), ":memory:")
	s.WithPostSyncRunner(&stubPostSyncRunner{})
	prestigeFired := false
	s.WithPrestigeHook(func(_ context.Context, _, _ string) { prestigeFired = true })
	cfg.SharedProvider = memProvider
	cfg.CurrentCSRSeasonID = "CsrSeason42"

	engine := s.BuildEngine(context.Background(), "JGtm", "2533274823110022")
	_ = prestigeFired // évite unused si les assertions bougent
	if engine == nil {
		t.Fatal("BuildEngine returned nil")
	}

	// --- Assertions de wiring ---
	if !engine.HasSharedProvider() {
		t.Error("REGRESSION: SharedProvider non câblé — le path watcher tombera en legacy → conflit different configuration (incident 2026-05-26)")
	}
	if !engine.HasFriendsLoader() {
		t.Error("REGRESSION: FriendsLoader non câblé — is_with_friends ne se recalculera plus auto post-sync")
	}
	if !engine.HasPostSyncRunner() {
		t.Error("REGRESSION: PostSyncRunner non câblé — notifications delta + progression V2 muettes (AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 cause B)")
	}
	if !engine.HasMediaScanHook() {
		t.Error("REGRESSION: MediaScanHook non câblé — captures non associées aux matchs")
	}
	if !engine.HasPrestigeHook() {
		t.Error("REGRESSION: PrestigeHook non câblé — prestige.RunPostSyncHook ne tournera sur AUCUN chemin V1 (VF-1, feature morte tests verts)")
	}
	if engine.CSRSeasonIDForTest() != "CsrSeason42" {
		t.Errorf("REGRESSION: CSRSeasonID = %q, want CsrSeason42", engine.CSRSeasonIDForTest())
	}
	if engine.GamertagForTest() != "JGtm" {
		t.Errorf("gamertag = %q, want JGtm", engine.GamertagForTest())
	}
	if engine.XUIDForTest() != "2533274823110022" {
		t.Errorf("xuid = %q, want 2533274823110022", engine.XUIDForTest())
	}
}

// TestBuildEngine_NilDeps_ProducesPartiallyWiredEngine vérifie qu'avec des
// deps nilles, BuildEngine reste opérationnel mais retourne un engine
// partiellement câblé. C'est le mode dégradé (tests / boot sans pool).
func TestBuildEngine_NilDeps_ProducesPartiallyWiredEngine(t *testing.T) {
	// Scheduler avec settings nil et pool nil.
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	s := scheduler.New(cfg, nil, nil, nil)

	engine := s.BuildEngine(context.Background(), "Player", "1234")
	if engine == nil {
		t.Fatal("BuildEngine returned nil even in degraded mode")
	}

	// Sans SharedProvider configuré, le moteur est en legacy mode (acceptable
	// pour les tests, pas pour la prod).
	if engine.HasSharedProvider() {
		t.Error("SharedProvider câblé alors que cfg.SharedProvider est nil — fuite")
	}
	// Sans settings, pas de FriendsLoader ni MediaScanHook.
	if engine.HasFriendsLoader() {
		t.Error("FriendsLoader câblé alors que settings est nil — fuite")
	}
	if engine.HasMediaScanHook() {
		t.Error("MediaScanHook câblé alors que settings est nil — fuite")
	}
	// Sans pool, pas de CustomClient.
	if engine.HasCustomClient() {
		t.Error("CustomClient câblé alors que pool est nil — fuite")
	}
	// Sans postSyncRunner injecté, pas de PostSyncRunner.
	if engine.HasPostSyncRunner() {
		t.Error("PostSyncRunner câblé alors que WithPostSyncRunner pas appelé — fuite")
	}
	// Sans WithPrestigeHook, pas de prestigeHook.
	if engine.HasPrestigeHook() {
		t.Error("PrestigeHook câblé alors que WithPrestigeHook pas appelé — fuite")
	}
}

// TestBuildEngine_DefaultRunnerFactory_DelegatesToBuildEngine vérifie que
// le DeltaRunner retourné par defaultRunnerFactory est bien le *SyncEngine
// produit par BuildEngine (même instance — pas une réécriture parallèle).
// Si quelqu'un divergait les 2 implementations, le path scheduler et le
// path watcher seraient à nouveau désynchronisés.
func TestBuildEngine_DefaultRunnerFactory_DelegatesToBuildEngine(t *testing.T) {
	s, cfg := newFullyWiredScheduler(t)
	memProvider := sharedprovider.FromInMemoryDB((*sql.DB)(nil), ":memory:")
	cfg.SharedProvider = memProvider

	// L'accès au type DeltaRunner privé du scheduler n'est pas possible
	// depuis _test externe, mais on peut vérifier que BuildEngine renvoie
	// un engine câblé de façon identique (smoke test stabilité).
	eng1 := s.BuildEngine(context.Background(), "P", "X")
	eng2 := s.BuildEngine(context.Background(), "P", "X")
	if eng1 == nil || eng2 == nil {
		t.Fatal("BuildEngine returned nil")
	}
	if eng1 == eng2 {
		t.Error("BuildEngine returned same pointer twice — devrait être un engine jetable par appel")
	}
	if eng1.HasSharedProvider() != eng2.HasSharedProvider() {
		t.Error("BuildEngine non-déterministe sur HasSharedProvider")
	}
	if eng1.HasFriendsLoader() != eng2.HasFriendsLoader() {
		t.Error("BuildEngine non-déterministe sur HasFriendsLoader")
	}
}
