// Package service — fanout_bootstrap_extra_test.go : tests des branches pures
// de FanoutService et BootstrapService (sans accès DuckDB).
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// ─────────────────────────────────────────────────────────────────────────────
// FanoutService — branches sans DB
// ─────────────────────────────────────────────────────────────────────────────

func TestFanoutBuildPlan_EmptyMatchIDs(t *testing.T) {
	cfg := &config.AppConfig{DBProfilesPath: "/nonexistent/db_profiles.json"}
	svc := NewFanoutService(cfg)

	plan, err := svc.BuildPlan(context.Background(), "PlayerA", nil)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if plan == nil {
		t.Fatal("plan nil")
	}
	if plan.SourceGamertag != "PlayerA" {
		t.Errorf("SourceGamertag = %q, want %q", plan.SourceGamertag, "PlayerA")
	}
	if len(plan.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(plan.Targets))
	}
}

func TestFanoutBuildPlan_EmptySlice(t *testing.T) {
	cfg := &config.AppConfig{DBProfilesPath: "/nonexistent/db_profiles.json"}
	svc := NewFanoutService(cfg)

	plan, err := svc.BuildPlan(context.Background(), "PlayerB", []string{})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(plan.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(plan.Targets))
	}
}

func TestFanoutExecute_NilPlan(t *testing.T) {
	cfg := &config.AppConfig{}
	svc := NewFanoutService(cfg)

	result := svc.Execute(context.Background(), nil)
	if result.TargetsProcessed != 0 {
		t.Errorf("expected 0 processed, got %d", result.TargetsProcessed)
	}
	if result.MatchesEnriched != 0 {
		t.Errorf("expected 0 enriched, got %d", result.MatchesEnriched)
	}
}

func TestFanoutExecute_EmptyPlan(t *testing.T) {
	cfg := &config.AppConfig{}
	svc := NewFanoutService(cfg)

	plan := &domain.FanoutPlan{SourceGamertag: "P1"}
	result := svc.Execute(context.Background(), plan)
	if result.TargetsProcessed != 0 {
		t.Errorf("expected 0 processed, got %d", result.TargetsProcessed)
	}
}

func TestFanoutExecute_TargetWithResolveError(t *testing.T) {
	// Le chemin db_profiles.json n'existe pas → ResolvePlayer échoue → Errors
	cfg := &config.AppConfig{
		DBProfilesPath: "/nonexistent/path/db_profiles.json",
		RepoRoot:       "/nonexistent",
	}
	svc := NewFanoutService(cfg)

	plan := &domain.FanoutPlan{
		SourceGamertag: "P1",
		MatchIDs:       []string{"match-001"},
		Targets: []domain.FanoutTarget{
			{Gamertag: "P2", XUID: "xuid-p2", CommonCount: 1, MissingCount: 1},
		},
	}

	result := svc.Execute(context.Background(), plan)
	// enrichTarget échoue → result.Errors non vide, TargetsProcessed = 0
	if result.TargetsProcessed != 0 {
		t.Errorf("expected 0 targets processed (all fail), got %d", result.TargetsProcessed)
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors, got none")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BootstrapService — Build + BuildPlayersList
// ─────────────────────────────────────────────────────────────────────────────

// newTestAppConfig crée un AppConfig minimal avec un db_profiles.json vide.
func newTestAppConfig(t *testing.T) *config.AppConfig {
	t.Helper()
	dir := t.TempDir()
	dbProfiles := filepath.Join(dir, "db_profiles.json")
	appSettings := filepath.Join(dir, "app_settings.json")

	// db_profiles vide → LoadPlayers retourne []
	if err := os.WriteFile(dbProfiles, []byte(`{"version":"3.0","profiles":{"halo-infinite":{}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	// app_settings minimal
	if err := os.WriteFile(appSettings, []byte(`{"lang":"fr"}`), 0600); err != nil {
		t.Fatal(err)
	}

	return &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  dbProfiles,
		AppSettingsPath: appSettings,
		DemoMode:        false,
	}
}

func TestBootstrapService_Build_NilSession(t *testing.T) {
	cfg := newTestAppConfig(t)
	svc := NewBootstrapService(cfg, nil)

	resp, err := svc.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	// Pas de joueurs → SetupRequired = true
	if !resp.SetupRequired {
		t.Error("expected SetupRequired=true (aucun joueur)")
	}
	if resp.AuthState != "missing" {
		t.Errorf("AuthState = %q, want %q", resp.AuthState, "missing")
	}
}

func TestBootstrapService_Build_WithSession(t *testing.T) {
	cfg := newTestAppConfig(t)
	svc := NewBootstrapService(cfg, nil)

	sess := &domain.SessionData{
		AuthReady:          true,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT1", XUID: "x1"},
		CurrentTitleSlug:   "halo_infinite",
	}
	resp, err := svc.Build(context.Background(), sess)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if resp.AuthState != "ready" {
		t.Errorf("AuthState = %q, want %q", resp.AuthState, "ready")
	}
	if resp.CurrentTitleSlug != "halo_infinite" {
		t.Errorf("CurrentTitleSlug = %q, want halo_infinite", resp.CurrentTitleSlug)
	}
}

func TestBootstrapService_Build_WithPlayers(t *testing.T) {
	dir := t.TempDir()
	dbProfiles := filepath.Join(dir, "db_profiles.json")
	appSettings := filepath.Join(dir, "app_settings.json")

	// Un joueur configuré
	profilesJSON := `{"version":"3.0","profiles":{"halo_infinite":{"testplayer":{"db_path":"data/players/testplayer/stats.duckdb","xuid":"xuid-tp"}}}}`
	if err := os.WriteFile(dbProfiles, []byte(profilesJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appSettings, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  dbProfiles,
		AppSettingsPath: appSettings,
	}
	svc := NewBootstrapService(cfg, nil)

	resp, err := svc.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	// Avec 1 joueur → SetupRequired = false
	if resp.SetupRequired {
		t.Error("expected SetupRequired=false avec 1 joueur")
	}
	if resp.CurrentPlayer == nil {
		t.Error("expected CurrentPlayer non-nil")
	}
}

func TestBootstrapService_BuildPlayersList_Empty(t *testing.T) {
	cfg := newTestAppConfig(t)
	svc := NewBootstrapService(cfg, nil)

	resp, err := svc.BuildPlayersList(context.Background(), nil)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if resp == nil {
		t.Fatal("resp nil")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	if resp.DefaultPlayerSlug != nil {
		t.Error("expected nil DefaultPlayerSlug")
	}
}

func TestBootstrapService_BuildPlayersList_WithPlayer(t *testing.T) {
	dir := t.TempDir()
	dbProfiles := filepath.Join(dir, "db_profiles.json")
	appSettings := filepath.Join(dir, "app_settings.json")

	profilesJSON := `{"version":"3.0","profiles":{"halo_infinite":{"player1":{"db_path":"data/players/player1/stats.duckdb","xuid":"xuid-1"}}}}`
	if err := os.WriteFile(dbProfiles, []byte(profilesJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appSettings, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  dbProfiles,
		AppSettingsPath: appSettings,
	}
	svc := NewBootstrapService(cfg, nil)

	resp, err := svc.BuildPlayersList(context.Background(), nil)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.DefaultPlayerSlug == nil {
		t.Error("expected non-nil DefaultPlayerSlug")
	}
}
