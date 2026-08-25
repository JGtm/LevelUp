//go:build cgo
// +build cgo

// Package scheduler_test — auto_sync_e2e_test.go : test end-to-end du pipeline
// complet AutoSyncScheduler avec vrai Pool/Resolver/Discovery + un
// MultiUserTokenStore temp.
//
// Couvre :
//   - Discovery lit le refresh_token depuis le MultiUserTokenStore (source
//     unique ADR 0023 Phase 5 — plus aucune lecture sync_meta)
//   - Resolver fait OAuth refresh (mocké) → Exchange (mocké) avec rotation
//   - Le callback onRotated persiste le rotated RT dans le store (write réel)
//   - Pool.HasPlayer reflète les sources découvertes
//   - Scheduler.syncPlayer câble PooledHaloClient et compte les outcomes
//   - L'erreur Microsoft (invalid_grant) propage à PlayerOutcomeDetail
//
// Le RunnerFactory est mocké pour court-circuiter le SyncEngine réel
// (qui dépend du shared_matches DB qu'on ne veut pas instancier ici).
package scheduler_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"

	_ "github.com/duckdb/duckdb-go/v2"
)

// e2eProvider est un TokenProvider stub configurable pour le test E2E.
type e2eProvider struct {
	oauthAccess   string
	oauthRotated  string
	oauthErr      error
	spartanToken  string
	exchangeErr   error
	oauthCalls    int
	exchangeCalls int
}

func (p *e2eProvider) InitDeviceFlow(_ context.Context) (auth.DeviceFlow, error) {
	return nil, nil
}
func (p *e2eProvider) TryOAuthRefresh(_ context.Context, _ string) (string, error) {
	p.oauthCalls++
	return p.oauthAccess, p.oauthErr
}
func (p *e2eProvider) TryOAuthRefreshWithRotation(_ context.Context, _ string) (string, string, error) {
	p.oauthCalls++
	return p.oauthAccess, p.oauthRotated, p.oauthErr
}
func (p *e2eProvider) Exchange(_ context.Context, _ string) (*auth.ExchangeResult, error) {
	p.exchangeCalls++
	if p.exchangeErr != nil {
		return nil, p.exchangeErr
	}
	return &auth.ExchangeResult{
		Tokens: &domain.HaloTokens{SpartanToken: p.spartanToken, ClearanceToken: "fake_clearance"},
	}, nil
}

// xuidForTestGamertag produit un xuid NUMÉRIQUE stable pour un gamertag de test.
// Le MultiUserTokenStore refuse les xuid non numériques (xuidIsSafe, protection
// path-traversal) : les fixtures « xuid-Alice » de l'ère sync_meta ne passent plus.
func xuidForTestGamertag(gamertag string) string {
	var sum int
	for _, r := range gamertag {
		sum = sum*31 + int(r)
		sum %= 100000
	}
	return "25332748" + fmt.Sprintf("%05d", sum)
}

// seedEmptyPlayerDB crée la stats.duckdb du joueur (le scheduler skippe un
// joueur sans DB). Depuis ADR 0023 Phase 5 elle ne porte PLUS aucun credential :
// sync_meta n'existe que pour la sync, pas pour l'auth.
func seedEmptyPlayerDB(t *testing.T, repoRoot, gamertag string) {
	t.Helper()
	dbPath := titlePkg.NewPathResolver(repoRoot).PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb %s: %v", dbPath, err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)`); err != nil {
		t.Fatalf("create sync_meta: %v", err)
	}
}

// seedTokenStore crée le MultiUserTokenStore du repo temp avec un refresh token
// initial pour le joueur (source unique ADR 0023).
func seedTokenStore(t *testing.T, repoRoot, xuid, gamertag, initialRT string) *auth.MultiUserTokenStore {
	t.Helper()
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(repoRoot).WatcherTokensDir())
	if err := store.Upsert(&auth.UserTokens{
		XUID: xuid, Gamertag: gamertag, OAuthRefreshToken: initialRT,
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	return store
}

// readStoreRT lit la valeur courante du refresh_token dans le store.
func readStoreRT(t *testing.T, store *auth.MultiUserTokenStore, xuid string) string {
	t.Helper()
	u, err := store.Load(xuid)
	if err != nil {
		if errors.Is(err, auth.ErrUserTokensNotFound) {
			return ""
		}
		t.Fatalf("store.Load: %v", err)
	}
	return u.OAuthRefreshToken
}

// storeRotationCallback reproduit le callback onRotated de production
// (main.go::buildAutoSyncPool) : écriture dans le MultiUserTokenStore.
func storeRotationCallback(store *auth.MultiUserTokenStore, xuid string) pool.TokenRotationCallback {
	return func(_ context.Context, _, newRT string) error {
		return store.UpdateOAuthRefreshToken(xuid, newRT)
	}
}

// buildE2EHarness assemble Discovery + Resolver + Pool + scheduler avec un
// MultiUserTokenStore temp et un provider stub.
// Le scheduler reçoit un RunnerFactory mockable (RunDelta est court-circuité).
func buildE2EHarness(t *testing.T, gamertag, initialRT string, prov *e2eProvider, runnerErr error) (*scheduler.AutoSyncScheduler, *auth.MultiUserTokenStore, string) {
	t.Helper()
	repoRoot := t.TempDir()
	xuid := xuidForTestGamertag(gamertag)

	// db_profiles.json minimal avec 1 joueur.
	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	profilesJSON := `{"version":"3.0","admin":"` + gamertag + `","profiles":{"halo_infinite":{"` + gamertag + `":{"db_path":"data/titles/halo_infinite/players/` + gamertag + `/stats.duckdb","xuid":"` + xuid + `","waypoint_player":"` + gamertag + `"}}}}`
	if err := os.WriteFile(dbProfilesPath, []byte(profilesJSON), 0o644); err != nil {
		t.Fatalf("write db_profiles: %v", err)
	}
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644)

	seedEmptyPlayerDB(t, repoRoot, gamertag)
	tokenStore := seedTokenStore(t, repoRoot, xuid, gamertag, initialRT)

	cfg := &config.AppConfig{
		RepoRoot:        repoRoot,
		DBProfilesPath:  dbProfilesPath,
		AppSettingsPath: settingsPath,
	}

	// Build le pipeline Discovery → Resolver → Pool comme en production.
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	discovery := pool.NewDiscoveryWithStore(cfg, pr, titlePkg.DefaultSlug, tokenStore)
	sources, err := discovery.Scan(context.Background())
	if err != nil {
		t.Fatalf("Discovery.Scan: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("Discovery n'a trouvé aucune source — le MultiUserTokenStore n'a peut-être pas été lu correctement")
	}

	resolver := pool.NewResolver(prov, 0, storeRotationCallback(tokenStore, xuid))
	p, err := pool.NewPool(context.Background(), resolver, sources, pool.PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	store := settings_platform.NewStore(settingsPath)
	s := scheduler.New(cfg, store, prov, p)

	// Mock le RunnerFactory : on ne veut pas dépendre du vrai SyncEngine
	// dans ce test E2E (le SyncEngine nécessite shared_matches.duckdb etc.).
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{
			result: domain.SyncResult{MatchesInserted: 1},
			err:    runnerErr,
		}
	}

	return s, tokenStore, xuid
}

// =============================================================================
// Tests E2E
// =============================================================================

// TestE2E_PipelineComplet_RotationPersistedAtBoot vérifie que le pipeline
// complet (Discovery → Resolver → Pool → scheduler) persiste bien le RT rotaté
// par Microsoft dans le MultiUserTokenStore. La rotation se produit
// au boot du Pool (qui Resolve toutes les sources à NewPool), pas pendant
// RunOnce car le Resolver cache les tokens TTL ~3h30.
//
// Sequence :
//  1. Seed le store watcher_tokens avec oauth_refresh_token = "rt_v1_initial"
//  2. Provider stub configuré : oauthRotated = "rt_v2_rotated_by_MS"
//  3. buildE2EHarness construit Pool → Resolver.Resolve au boot → callback
//     onRotated → UpdateOAuthRefreshToken
//  4. Relire le store : doit valoir "rt_v2_rotated_by_MS"
//  5. RunOnce → utilise le token cached, n'invoque pas un nouveau Resolve
func TestE2E_PipelineComplet_RotationPersistedAtBoot(t *testing.T) {
	prov := &e2eProvider{
		oauthAccess:  "access_v1",
		oauthRotated: "rt_v2_rotated_by_MS",
		spartanToken: "spartan_v1",
	}
	s, tokenStore, xuid := buildE2EHarness(t, "Alice", "rt_v1_initial", prov, nil)

	// Après le boot du Pool (dans buildE2EHarness), le Resolver a déjà été
	// invoqué et le callback onRotated a déjà persisté le rotated RT.
	if got := readStoreRT(t, tokenStore, xuid); got != "rt_v2_rotated_by_MS" {
		t.Errorf("store après boot Pool = %q, want rt_v2_rotated_by_MS (callback onRotated non câblé ?)", got)
	}

	// Vérifier que le provider a bien été appelé pendant le boot.
	if prov.oauthCalls == 0 {
		t.Error("provider.TryOAuthRefreshWithRotation jamais appelé pendant le boot du Pool")
	}
	if prov.exchangeCalls == 0 {
		t.Error("provider.Exchange jamais appelé — le pipeline Resolver a échoué avant Exchange")
	}

	// Lancer un cycle scheduler. Comme le Resolver cache, on n'a pas de
	// nouvel appel oauth, et le RT en DB reste rotaté.
	oauthCallsBeforeRun := prov.oauthCalls
	res := s.RunOnce(context.Background())
	if res.Synced != 1 {
		t.Errorf("Synced = %d, want 1 (Alice OK via pipeline complet)", res.Synced)
	}
	if got := readStoreRT(t, tokenStore, xuid); got != "rt_v2_rotated_by_MS" {
		t.Errorf("store après RunOnce = %q, want rt_v2_rotated_by_MS (inchangé)", got)
	}
	if prov.oauthCalls != oauthCallsBeforeRun {
		t.Errorf("oauthCalls = %d après RunOnce, want %d (cache TTL devrait éviter un nouvel appel)",
			prov.oauthCalls, oauthCallsBeforeRun)
	}

	// Vérifier le snapshot scheduler.
	snap := s.Snapshot()
	if len(snap.Players) != 1 || snap.Players[0].Outcome != "ok" {
		t.Errorf("snapshot Players = %+v", snap.Players)
	}
	if snap.PoolSize != 1 {
		t.Errorf("PoolSize = %d, want 1", snap.PoolSize)
	}
}

// TestE2E_ProviderInvalidGrant_PoolNotCreated vérifie que si Microsoft retourne
// invalid_grant pour TOUTES les sources au boot, NewPool refuse de démarrer
// (aucun slot vivant). Le RT initial dans le store reste intact (pas de
// rotation réussie). En production, `buildAutoSyncPool` capte cette erreur,
// retourne nil, et le scheduler skip tous les joueurs avec un reason explicite.
func TestE2E_ProviderInvalidGrant_PoolNotCreated(t *testing.T) {
	prov := &e2eProvider{
		oauthErr: errors.New("oauth_refresh: invalid_grant — AADSTS70000"),
	}
	repoRoot := t.TempDir()
	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	_ = os.WriteFile(dbProfilesPath, []byte(`{"version":"3.0","admin":"Bob","profiles":{"halo_infinite":{"Bob":{"db_path":"data/titles/halo_infinite/players/Bob/stats.duckdb","xuid":"2533274800002","waypoint_player":"Bob"}}}}`), 0o644)
	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: dbProfilesPath}
	tokenStore := seedTokenStore(t, repoRoot, "2533274800002", "Bob", "rt_v1_revoked_by_microsoft")

	pr := titlePkg.NewPathResolver(repoRoot)
	discovery := pool.NewDiscoveryWithStore(cfg, pr, titlePkg.DefaultSlug, tokenStore)
	sources, _ := discovery.Scan(context.Background())
	if len(sources) != 1 {
		t.Fatalf("Discovery devrait trouver 1 source, got %d", len(sources))
	}

	resolver := pool.NewResolver(prov, 0, storeRotationCallback(tokenStore, "2533274800002"))
	_, err := pool.NewPool(context.Background(), resolver, sources, pool.PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err == nil {
		t.Error("NewPool devrait échouer quand toutes les sources sont invalid_grant")
	}

	// Le RT initial ne doit PAS être modifié (aucune rotation réussie).
	if got := readStoreRT(t, tokenStore, "2533274800002"); got != "rt_v1_revoked_by_microsoft" {
		t.Errorf("store modifié alors que tout a échoué : got %q", got)
	}
}

// TestE2E_PoolNil_SchedulerSkipsAll vérifie le scénario où buildAutoSyncPool
// retourne nil (par exemple Discovery vide ou tous les RT révoqués) : le
// scheduler tourne quand même mais skip tous les joueurs avec un reason
// explicite (en production via le hint affiché dans le snapshot).
func TestE2E_PoolNil_SchedulerSkipsAll(t *testing.T) {
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	_ = os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":1}`), 0o644)

	dbProfilesPath := filepath.Join(repoRoot, "db_profiles.json")
	_ = os.WriteFile(dbProfilesPath, []byte(`{
		"version":"3.0","admin":"Alice","profiles":{"halo_infinite":{
			"Alice":{"db_path":"data/titles/halo_infinite/players/Alice/stats.duckdb","xuid":"xa","waypoint_player":"Alice"},
			"Bob":{"db_path":"data/titles/halo_infinite/players/Bob/stats.duckdb","xuid":"2533274800002","waypoint_player":"Bob"}
		}}}`), 0o644)

	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: dbProfilesPath, AppSettingsPath: settingsPath}
	store := settings_platform.NewStore(settingsPath)
	s := scheduler.New(cfg, store, &e2eProvider{}, nil) // pool nil

	res := s.RunOnce(context.Background())
	if res.Total != 2 {
		t.Errorf("Total = %d, want 2", res.Total)
	}
	if res.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2 (pool nil → tous skipped)", res.Skipped)
	}
	if res.Synced != 0 || res.Failed != 0 {
		t.Errorf("Synced/Failed devraient être 0 ; got %d/%d", res.Synced, res.Failed)
	}
}
