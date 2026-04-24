// Package scheduler_test — auto_sync_test.go : tests unitaires du scheduler de sync automatique.
//
// Tous les tests utilisent des mocks injectés (tokenReader + engineFactory) pour éviter
// toute dépendance DuckDB, réseau ou MSAL. Les mocks sont déclarés dans ce fichier.
//
// Stratégie des tests :
//   - Les champs AutoSyncScheduler.tokenReader et .engineFactory sont injectables
//     directement après New() → tests totalement déterministes.
//   - Les tests de compteurs (Synced/Skipped/Failed) vérifient la logique de RunOnce.
//   - TestRun_CancelCtxStops vérifie que Run() se termine sans deadlock à l'annulation.
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
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockProvider implémente auth.TokenProvider de façon configurable.
type mockProvider struct {
	silentToken string
	silentErr   error
	oauthToken  string
	oauthErr    error
	exchangeRes *auth.ExchangeResult
	exchangeErr error
}

// Vérification compile-time : mockProvider satisfait l'interface.
var _ auth.TokenProvider = (*mockProvider)(nil)

func (m *mockProvider) InitDeviceFlow(_ context.Context) (*auth.DeviceCodeFlow, error) {
	return nil, nil
}

func (m *mockProvider) TrySilentRefresh(_ context.Context, _ string) (string, error) {
	return m.silentToken, m.silentErr
}

func (m *mockProvider) TryOAuthRefresh(_ context.Context, _ string) (string, error) {
	return m.oauthToken, m.oauthErr
}

func (m *mockProvider) AcquireAccessToken(_ context.Context, _, _ string) (string, error) {
	if m.silentToken != "" && m.silentErr == nil {
		return m.silentToken, nil
	}
	return m.oauthToken, m.oauthErr
}

func (m *mockProvider) Exchange(_ context.Context, _ string) (*auth.ExchangeResult, error) {
	return m.exchangeRes, m.exchangeErr
}

// mockRunner implémente DeltaRunner de façon configurable.
type mockRunner struct {
	result domain.SyncResult
	err    error
	called bool
}

func (m *mockRunner) RunDelta(_ context.Context, _ domain.SyncOptions) (domain.SyncResult, error) {
	m.called = true
	return m.result, m.err
}

// ---------------------------------------------------------------------------
// Helpers de construction
// ---------------------------------------------------------------------------

// newTestScheduler crée un AutoSyncScheduler avec :
//   - un tokenReader injectable (peut être nil → remplacé par le défaut)
//   - un engineFactory injectable (peut être nil → remplacé par le défaut)
//   - un AppConfig pointant sur repoRoot (temp dir fourni par le test)
//   - un Store de settings pointant sur un fichier de settings minimal
func newTestScheduler(
	t *testing.T,
	repoRoot string,
	provider auth.TokenProvider,
	tokenReader scheduler.TokenReader,
	factory scheduler.EngineFactory,
) *scheduler.AutoSyncScheduler {
	t.Helper()

	// Créer un fichier app_settings.json minimal dans repoRoot
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	settingsJSON := `{
		"spnkr_auto_sync_enabled": true,
		"spnkr_auto_sync_interval_hours": 1
	}`
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o644); err != nil {
		t.Fatalf("écriture settings: %v", err)
	}

	// Créer un fichier db_profiles.json vide (aucun joueur par défaut)
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{"version":"2.1","profiles":{}}`), 0o644); err != nil {
		t.Fatalf("écriture db_profiles: %v", err)
	}

	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  profilesPath,
		AppSettingsPath: settingsPath,
	}
	store := settings_platform.NewStore(settingsPath)

	s := scheduler.New(cfg, store, provider)
	if tokenReader != nil {
		s.TokenReader = tokenReader
	}
	if factory != nil {
		s.EngineFactory = factory
	}
	return s
}

// addPlayer crée la structure data/titles/halo_infinite/players/{gamertag}/ dans repoRoot et enregistre
// le joueur dans db_profiles.json (format v2.1).
func addPlayer(t *testing.T, repoRoot, gamertag string, withDB bool) {
	t.Helper()

	playerDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", playerDir, err)
	}
	if withDB {
		dbPath := filepath.Join(playerDir, "stats.duckdb")
		if err := os.WriteFile(dbPath, []byte(""), 0o644); err != nil {
			t.Fatalf("création stats.duckdb: %v", err)
		}
	}

	// Mettre à jour db_profiles.json avec le joueur
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	content := `{"version":"2.1","profiles":{"` + gamertag + `":{"xuid":"xuid_` + gamertag + `"}}}`
	if err := os.WriteFile(profilesPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile db_profiles: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests : intervalFromHours (fonction non exportée, testée indirectement)
// ---------------------------------------------------------------------------

func TestIntervalFromHours_Zero_ReturnsDefault(t *testing.T) {
	// L'intervalle par défaut est 6h. settings avec 0 → doit utiliser le défaut.
	dir := t.TempDir()
	provider := &mockProvider{}
	s := newTestScheduler(t, dir, provider, nil, nil)

	// Écrire settings avec interval = 0
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":0}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Vérification indirecte via CurrentInterval()
	d := s.CurrentInterval()
	if d != 6*time.Hour {
		t.Errorf("intervalle attendu 6h, obtenu %v", d)
	}
}

func TestIntervalFromHours_Custom(t *testing.T) {
	dir := t.TempDir()
	provider := &mockProvider{}
	s := newTestScheduler(t, dir, provider, nil, nil)

	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":12}`), 0o644); err != nil {
		t.Fatal(err)
	}

	d := s.CurrentInterval()
	if d != 12*time.Hour {
		t.Errorf("intervalle attendu 12h, obtenu %v", d)
	}
}

// ---------------------------------------------------------------------------
// Tests : RunOnce
// ---------------------------------------------------------------------------

func TestRunOnce_EmptyPlayerList(t *testing.T) {
	dir := t.TempDir()
	provider := &mockProvider{}
	s := newTestScheduler(t, dir, provider, nil, nil)
	// db_profiles.json est vide ({}) → aucun joueur

	res := s.RunOnce(context.Background())
	if res.Total != 0 {
		t.Errorf("Total: attendu 0, obtenu %d", res.Total)
	}
	if res.Synced != 0 || res.Skipped != 0 || res.Failed != 0 {
		t.Errorf("compteurs non nuls sur liste vide: %+v", res)
	}
}

func TestRunOnce_PlayerDBAbsent(t *testing.T) {
	// Le joueur est dans db_profiles mais stats.duckdb n'existe pas → Skipped
	dir := t.TempDir()
	provider := &mockProvider{}
	s := newTestScheduler(t, dir, provider, nil, nil)
	addPlayer(t, dir, "TestGT", false /* sans DB */)

	res := s.RunOnce(context.Background())
	if res.Total != 1 {
		t.Errorf("Total: attendu 1, obtenu %d", res.Total)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped: attendu 1, obtenu %d", res.Skipped)
	}
	if res.Failed != 0 || res.Synced != 0 {
		t.Errorf("compteurs inattendus: %+v", res)
	}
}

func TestRunOnce_NoToken(t *testing.T) {
	// tokenReader retourne ("", nil) → aucun token → Skipped
	dir := t.TempDir()
	provider := &mockProvider{}

	noToken := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "", nil
	}
	s := newTestScheduler(t, dir, provider, noToken, nil)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Skipped != 1 {
		t.Errorf("Skipped: attendu 1, obtenu %d", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Failed inattendu: %+v", res)
	}
}

func TestRunOnce_TokenReadError(t *testing.T) {
	// tokenReader retourne une erreur → Failed
	dir := t.TempDir()
	provider := &mockProvider{}

	tokenErr := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "", errors.New("DuckDB: base corrompue")
	}
	s := newTestScheduler(t, dir, provider, tokenErr, nil)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Failed != 1 {
		t.Errorf("Failed: attendu 1, obtenu %d", res.Failed)
	}
	if res.Skipped != 0 || res.Synced != 0 {
		t.Errorf("compteurs inattendus: %+v", res)
	}
}

func TestRunOnce_ExchangeError(t *testing.T) {
	// tokenReader retourne un token valide, mais Exchange échoue → Failed
	dir := t.TempDir()
	provider := &mockProvider{
		exchangeErr: errors.New("XSTS: token expiré"),
	}

	validToken := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "access_token_valide", nil
	}
	s := newTestScheduler(t, dir, provider, validToken, nil)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Failed != 1 {
		t.Errorf("Failed: attendu 1, obtenu %d", res.Failed)
	}
}

func TestRunOnce_DeltaError(t *testing.T) {
	// Exchange OK, RunDelta retourne une erreur → Failed
	dir := t.TempDir()
	runner := &mockRunner{err: errors.New("API Halo: rate limit")}
	provider := &mockProvider{
		exchangeRes: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "spartan_tok", ClearanceToken: "clearance_tok"},
		},
	}

	validToken := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "access_token_valide", nil
	}
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		return runner
	}
	s := newTestScheduler(t, dir, provider, validToken, factory)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Failed != 1 {
		t.Errorf("Failed: attendu 1, obtenu %d", res.Failed)
	}
	if !runner.called {
		t.Error("RunDelta aurait dû être appelé")
	}
}

func TestRunOnce_DeltaPartialSuccess(t *testing.T) {
	// RunDelta réussit mais retourne des erreurs partielles → compté comme Synced
	dir := t.TempDir()
	runner := &mockRunner{
		result: domain.SyncResult{
			MatchesInserted: 3,
			Errors:          []string{"timeout sur 1 match"},
		},
	}
	provider := &mockProvider{
		exchangeRes: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "s", ClearanceToken: "c"},
		},
	}

	validToken := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "tok", nil
	}
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		return runner
	}
	s := newTestScheduler(t, dir, provider, validToken, factory)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced: attendu 1 (succès partiel compte quand même), obtenu %d", res.Synced)
	}
	if res.Failed != 0 {
		t.Errorf("Failed inattendu sur succès partiel: %+v", res)
	}
}

func TestRunOnce_FullSuccess(t *testing.T) {
	// Scénario nominal : token MSAL valide + Exchange OK + RunDelta OK → Synced=1
	dir := t.TempDir()
	runner := &mockRunner{
		result: domain.SyncResult{MatchesInserted: 10, MatchesSkipped: 2},
	}
	provider := &mockProvider{
		exchangeRes: &auth.ExchangeResult{
			Tokens:   &domain.HaloTokens{SpartanToken: "spartan", ClearanceToken: "clearance"},
			Gamertag: "TestGT",
			XUID:     "xuid_TestGT",
		},
	}

	msalToken := func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "msal_access_token", nil
	}
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		return runner
	}
	s := newTestScheduler(t, dir, provider, msalToken, factory)
	addPlayer(t, dir, "TestGT", true)

	res := s.RunOnce(context.Background())
	if res.Total != 1 {
		t.Errorf("Total: attendu 1, obtenu %d", res.Total)
	}
	if res.Synced != 1 {
		t.Errorf("Synced: attendu 1, obtenu %d", res.Synced)
	}
	if res.Skipped != 0 || res.Failed != 0 {
		t.Errorf("compteurs inattendus: %+v", res)
	}
	if res.Duration == 0 {
		t.Error("Duration devrait être > 0")
	}
}

func TestRunOnce_MultiPlayer_MixedOutcomes(t *testing.T) {
	// 3 joueurs : 1 DB absente (Skipped), 1 token absent (Skipped), 1 succès (Synced)
	dir := t.TempDir()

	runner := &mockRunner{result: domain.SyncResult{MatchesInserted: 5}}
	provider := &mockProvider{
		exchangeRes: &auth.ExchangeResult{
			Tokens: &domain.HaloTokens{SpartanToken: "s", ClearanceToken: "c"},
		},
	}

	tokenReader := func(_ context.Context, dbPath string, _ string, _ auth.TokenProvider) (string, error) {
		if filepath.Base(filepath.Dir(dbPath)) == "PlayerNoToken" {
			return "", nil // pas de token
		}
		return "access_token", nil
	}
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		return runner
	}
	// Créer le scheduler d'abord (écrit db_profiles.json vide)
	s := newTestScheduler(t, dir, provider, tokenReader, factory)

	// Joueur 1 : DB absente → Skipped par le scheduler (avant même le tokenReader)
	playerDir1 := filepath.Join(dir, "data", "titles", "halo_infinite", "players", "PlayerNoDb")
	if err := os.MkdirAll(playerDir1, 0o755); err != nil {
		t.Fatal(err)
	}

	// Joueur 2 : DB présente mais tokenReader retourne ""
	playerDir2 := filepath.Join(dir, "data", "titles", "halo_infinite", "players", "PlayerNoToken")
	if err := os.MkdirAll(playerDir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir2, "stats.duckdb"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Joueur 3 : DB présente, token OK, sync réussie
	playerDir3 := filepath.Join(dir, "data", "titles", "halo_infinite", "players", "PlayerOK")
	if err := os.MkdirAll(playerDir3, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir3, "stats.duckdb"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Écrire db_profiles.json avec les 3 joueurs (écrase le {} créé par newTestScheduler)
	profiles := `{"version":"2.1","profiles":{
		"PlayerNoDb":    {"xuid": "xuid1"},
		"PlayerNoToken": {"xuid": "xuid2"},
		"PlayerOK":      {"xuid": "xuid3"}
	}}`
	profilesPath := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	res := s.RunOnce(context.Background())
	if res.Total != 3 {
		t.Errorf("Total: attendu 3, obtenu %d", res.Total)
	}
	if res.Synced != 1 {
		t.Errorf("Synced: attendu 1, obtenu %d", res.Synced)
	}
	if res.Skipped != 2 {
		t.Errorf("Skipped: attendu 2, obtenu %d", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Failed inattendu: %+v", res)
	}
}

// ---------------------------------------------------------------------------
// Test : Run() s'arrête proprement à l'annulation du contexte
// ---------------------------------------------------------------------------

func TestRun_CancelCtxStops(t *testing.T) {
	dir := t.TempDir()
	provider := &mockProvider{}
	s := newTestScheduler(t, dir, provider, nil, nil)

	// Forcer un intervalle très long pour que le ticker ne fire pas
	// → on teste uniquement que le case <-ctx.Done() fonctionne.
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":99}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	// Annuler après un court délai
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Run() s'est terminé correctement
	case <-time.After(2 * time.Second):
		t.Error("Run() ne s'est pas terminé dans les 2 secondes après annulation du contexte")
	}
}

// ---------------------------------------------------------------------------
// Test : ActivityChecker — le scheduler saute un joueur actif
// ---------------------------------------------------------------------------

// mockActivityChecker implémente PlayerActivityChecker.
type mockActivityChecker struct {
	activePlayers map[string]bool
}

func (m *mockActivityChecker) IsPlayerActive(gamertag string) bool {
	return m.activePlayers[gamertag]
}

func TestRunOnce_ActivityChecker_SkipsActivePlayer(t *testing.T) {
	dir := t.TempDir()
	provider := &mockProvider{}
	syncCalled := false
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		syncCalled = true
		return &mockRunner{}
	}
	s := newTestScheduler(t, dir, provider, nil, factory)

	// Configurer un joueur avec DB + token valide
	addPlayer(t, dir, "ActivePlayer", true)
	provider.exchangeRes = &auth.ExchangeResult{}
	s.TokenReader = func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "token123", nil
	}

	// ActivityChecker dit que le joueur est actif
	s.ActivityChecker = &mockActivityChecker{activePlayers: map[string]bool{"ActivePlayer": true}}

	res := s.RunOnce(context.Background())

	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", res.Skipped)
	}
	if res.Synced != 0 {
		t.Errorf("Synced = %d, want 0 (joueur actif ne doit pas être syncé)", res.Synced)
	}
	if syncCalled {
		t.Error("RunDelta ne doit pas être appelé quand le watcher est actif sur ce joueur")
	}
}

func TestRunOnce_ActivityChecker_SyncsIdlePlayer(t *testing.T) {
	dir := t.TempDir()
	provider := &mockProvider{
		exchangeRes: &auth.ExchangeResult{
			Tokens:   &domain.HaloTokens{SpartanToken: "s", ClearanceToken: "c"},
			Gamertag: "IdlePlayer",
			XUID:     "xuid_IdlePlayer",
		},
	}
	syncCalled := false
	factory := func(_, _, _ string, _ *domain.HaloTokens, _ auth.TokenProvider) scheduler.DeltaRunner {
		syncCalled = true
		return &mockRunner{}
	}
	s := newTestScheduler(t, dir, provider, nil, factory)

	addPlayer(t, dir, "IdlePlayer", true)
	s.TokenReader = func(_ context.Context, _ string, _ string, _ auth.TokenProvider) (string, error) {
		return "token123", nil
	}

	// ActivityChecker dit que le joueur est Idle (non actif)
	s.ActivityChecker = &mockActivityChecker{activePlayers: map[string]bool{"IdlePlayer": false}}

	res := s.RunOnce(context.Background())

	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (joueur idle doit être syncé normalement)", res.Synced)
	}
	if !syncCalled {
		t.Error("RunDelta doit être appelé pour un joueur Idle")
	}
}
