// Package scheduler — spartan_customization_cron_test.go : tests unitaires
// du cron customization V2 Phase 8.
package scheduler_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/scheduler"
)

// mockSpartanFetcher compte les invocations, mémorise le dernier xuid SUJET reçu
// (finding ID4 : le sujet est passé explicitement) + retourne un identity bidon.
type mockSpartanFetcher struct {
	calls    atomic.Int32
	lastXUID atomic.Value // string ; refreshOne appelle le refresher séquentiellement
}

func (m *mockSpartanFetcher) GetSpartanIdentityFor(_ context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error) {
	m.calls.Add(1)
	m.lastXUID.Store(xuid)
	return &domain.HomeSpartanIdentityRow{}, nil
}

// fakeCronPool — minimal pour les tests du cron uniquement.
type fakeCronPool struct {
	hasPlayerMap map[string]bool
	leaseTokens  *domain.HaloTokens
	acquireErr   error
}

func (m *fakeCronPool) Acquire(_ context.Context, _ pool.AcquirePolicy, _ string) (*pool.Lease, error) {
	if m.acquireErr != nil {
		return nil, m.acquireErr
	}
	return &pool.Lease{Tokens: m.leaseTokens, Release: func() {}}, nil
}
func (m *fakeCronPool) Size() int                               { return len(m.hasPlayerMap) }
func (m *fakeCronPool) HasPlayer(gt string) bool                { return m.hasPlayerMap[gt] }
func (m *fakeCronPool) MarkUnhealthy(_ string, _ error)         {}
func (m *fakeCronPool) OnHTTPError(_ int, _ time.Duration)      {}
func (m *fakeCronPool) On429ForToken(_ string, _ time.Duration) {}
func (m *fakeCronPool) AddOrUpdateSource(_ context.Context, _ pool.CredentialSource) error {
	return nil
}
func (m *fakeCronPool) Close() {}

// writeTestProfiles crée un db_profiles.json minimal pour cfg.LoadPlayers.
func writeTestProfiles(t *testing.T, repoRoot, gamertag, xuid string) {
	t.Helper()
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	json := `{"version":"3.0","profiles":{"halo_infinite":{"` + gamertag +
		`":{"db_path":"unused","xuid":"` + xuid + `","waypoint_player":"` + gamertag + `"}}}}`
	if err := os.WriteFile(profilesPath, []byte(json), 0o644); err != nil {
		t.Fatalf("write profiles: %v", err)
	}
}

// writeTestProfilesMultiTitle écrit un db_profiles.json v3 avec un joueur PAR
// titre (le slug est la clé de premier niveau). Permet de vérifier le routage
// title-aware du cron (chaque titre charge SES joueurs).
func writeTestProfilesMultiTitle(t *testing.T, repoRoot string, byTitle map[string][2]string) {
	t.Helper()
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	var sb strings.Builder
	sb.WriteString(`{"version":"3.0","profiles":{`)
	first := true
	for slug, gtXuid := range byTitle {
		if !first {
			sb.WriteString(",")
		}
		first = false
		sb.WriteString(`"` + slug + `":{"` + gtXuid[0] +
			`":{"db_path":"unused","xuid":"` + gtXuid[1] + `","waypoint_player":"` + gtXuid[0] + `"}}`)
	}
	sb.WriteString(`}}`)
	if err := os.WriteFile(profilesPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write profiles: %v", err)
	}
}

// TestSpartanCron_NoOpOnNilProvider : pas de panic si provider nil.
func TestSpartanCron_NoOpOnNilProvider(t *testing.T) {
	cron := scheduler.NewSpartanCustomizationCron(nil, nil, nil, "halo_infinite", 0)
	cron.RunOnce(context.Background()) // no panic
}

// TestSpartanCron_DefaultInterval : interval=0 → 8h par défaut.
func TestSpartanCron_DefaultInterval(t *testing.T) {
	const want = 8 * time.Hour
	if scheduler.DefaultSpartanCustomizationInterval != want {
		t.Errorf("DefaultSpartanCustomizationInterval: got %v, want %v",
			scheduler.DefaultSpartanCustomizationInterval, want)
	}
}

// TestSpartanCron_RunOnce_NoPlayers : config sans player → RunOnce ne plante pas.
func TestSpartanCron_RunOnce_NoPlayers(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	cron := scheduler.NewSpartanCustomizationCron(cfg, nil, provider, "halo_infinite", 0)
	cron.RunOnce(context.Background())
}

// TestSpartanCron_RunOnce_WithPlayerInPool : 1 joueur dans le pool, fetcher
// est appelé exactement 1 fois.
func TestSpartanCron_RunOnce_WithPlayerInPool(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestProfiles(t, repoRoot, "TestGT", "1234567890123456")

	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: filepath.Join(repoRoot, "db_profiles.json")}
	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	pl := &fakeCronPool{
		hasPlayerMap: map[string]bool{"TestGT": true},
		leaseTokens:  &domain.HaloTokens{SpartanToken: "fake-token"},
	}
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0)
	cron.RunOnce(context.Background())

	if fetcher.calls.Load() != 1 {
		t.Errorf("fetcher.calls: got %d, want 1", fetcher.calls.Load())
	}
	// Finding ID4 : le SUJET est passé explicitement (p.XUID), pas lu du ctx ambiant.
	if got, _ := fetcher.lastXUID.Load().(string); got != "1234567890123456" {
		t.Errorf("sujet passé au fetcher: got %q, want 1234567890123456 (p.XUID explicite)", got)
	}
}

// TestSpartanCron_RunOnce_SkipPlayerNotInPool : joueur configuré mais absent
// du pool → fetcher non appelé.
func TestSpartanCron_RunOnce_SkipPlayerNotInPool(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestProfiles(t, repoRoot, "TestGT", "1234567890123456")

	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: filepath.Join(repoRoot, "db_profiles.json")}
	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	pl := &fakeCronPool{hasPlayerMap: map[string]bool{}} // joueur absent
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0)
	cron.RunOnce(context.Background())

	if fetcher.calls.Load() != 0 {
		t.Errorf("fetcher should not be called when player not in pool: %d", fetcher.calls.Load())
	}
}

// TestSpartanCron_RunOnce_AcquireFails : pool.Acquire échoue → fetcher non appelé,
// pas de panic.
func TestSpartanCron_RunOnce_AcquireFails(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestProfiles(t, repoRoot, "TestGT", "1234567890123456")

	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: filepath.Join(repoRoot, "db_profiles.json")}
	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	pl := &fakeCronPool{
		hasPlayerMap: map[string]bool{"TestGT": true},
		acquireErr:   errors.New("token unavailable"),
	}
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0)
	cron.RunOnce(context.Background())

	if fetcher.calls.Load() != 0 {
		t.Errorf("fetcher should not be called when acquire fails: %d", fetcher.calls.Load())
	}
}

// TestSpartanCron_RunOnce_SkipsAuthOnlyProfile : un profil auth_only (compte
// token-only sans player DB, ex. DankerGlue/QuiteSiren) est exclu du refresh à la
// source — le fetcher n'est appelé QUE pour le vrai joueur. Garde-rail contre le
// bruit "refresher failed: No such file or directory" en prod.
func TestSpartanCron_RunOnce_SkipsAuthOnlyProfile(t *testing.T) {
	repoRoot := t.TempDir()
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	json := `{"version":"3.0","profiles":{"halo_infinite":{` +
		`"RealGT":{"db_path":"unused","xuid":"1111111111111111","waypoint_player":"RealGT"},` +
		`"DankerGlue":{"db_path":"","xuid":"2222222222222222","waypoint_player":"DankerGlue","auth_only":true}` +
		`}}}`
	if err := os.WriteFile(profilesPath, []byte(json), 0o644); err != nil {
		t.Fatalf("write profiles: %v", err)
	}

	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: profilesPath}
	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	// Les DEUX sont dans le pool (le compte auth_only fournit bien un RT) : sans le
	// filtre SyncablePlayers, le cron tenterait de le rafraîchir et échouerait.
	pl := &fakeCronPool{
		hasPlayerMap: map[string]bool{"RealGT": true, "DankerGlue": true},
		leaseTokens:  &domain.HaloTokens{SpartanToken: "fake-token"},
	}
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0)
	cron.RunOnce(context.Background())

	if fetcher.calls.Load() != 1 {
		t.Errorf("fetcher.calls: got %d, want 1 (auth_only exclu)", fetcher.calls.Load())
	}
}

// twoTitleRegistry construit un registre avec halo_infinite (built-in) + un titre
// supplémentaire actif (ex. halo_5) pour les tests title-aware.
func twoTitleRegistry(extraSlug string) *titlePkg.Registry {
	reg := titlePkg.NewRegistry() // halo_infinite seul par défaut
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:   extraSlug,
		Name:   extraSlug,
		Status: titlePkg.StatusActive,
	})
	return reg
}

// TestSpartanCron_TitleAware_RoutesToPerTitleRefresher : un joueur HINF et un
// joueur H5 sont chacun rafraîchis par LEUR refresher (routage title-aware), et
// jamais par celui de l'autre titre.
func TestSpartanCron_TitleAware_RoutesToPerTitleRefresher(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestProfilesMultiTitle(t, repoRoot, map[string][2]string{
		"halo_infinite": {"InfGT", "1111111111111111"},
		"halo_5":        {"H5GT", "2222222222222222"},
	})
	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: filepath.Join(repoRoot, "db_profiles.json")}

	fetcher := &mockSpartanFetcher{} // refresher HINF (chemin career identity)
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}

	var h5Calls atomic.Int32
	var h5Gamertags []string
	h5Refresher := func(_ context.Context, p domain.PlayerSummary) error {
		h5Calls.Add(1)
		h5Gamertags = append(h5Gamertags, p.Gamertag)
		return nil
	}

	pl := &fakeCronPool{
		hasPlayerMap: map[string]bool{"InfGT": true, "H5GT": true},
		leaseTokens:  &domain.HaloTokens{SpartanToken: "fake-token"},
	}
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0).
		WithRegistry(twoTitleRegistry("halo_5")).
		WithRefresher("halo_5", h5Refresher)
	cron.RunOnce(context.Background())

	if fetcher.calls.Load() != 1 {
		t.Errorf("HINF refresher: got %d calls, want 1", fetcher.calls.Load())
	}
	if h5Calls.Load() != 1 {
		t.Errorf("H5 refresher: got %d calls, want 1", h5Calls.Load())
	}
	if len(h5Gamertags) != 1 || h5Gamertags[0] != "H5GT" {
		t.Errorf("H5 refresher reçu les mauvais joueurs: %v (want [H5GT])", h5Gamertags)
	}
}

// TestSpartanCron_TitleAware_SkipsTitleWithoutRefresher : un titre actif SANS
// refresher enregistré est ignoré proprement (pas de panic, pas d'appel), même
// s'il a des joueurs configurés.
func TestSpartanCron_TitleAware_SkipsTitleWithoutRefresher(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestProfilesMultiTitle(t, repoRoot, map[string][2]string{
		"halo_infinite": {"InfGT", "1111111111111111"},
		"halo_5":        {"H5GT", "2222222222222222"},
	})
	cfg := &config.AppConfig{RepoRoot: repoRoot, DBProfilesPath: filepath.Join(repoRoot, "db_profiles.json")}

	fetcher := &mockSpartanFetcher{}
	provider := func(_ context.Context, _ string) (scheduler.SpartanIdentityFetcher, error) {
		return fetcher, nil
	}
	pl := &fakeCronPool{
		hasPlayerMap: map[string]bool{"InfGT": true, "H5GT": true},
		leaseTokens:  &domain.HaloTokens{SpartanToken: "fake-token"},
	}
	// halo_5 actif dans le registre mais AUCUN refresher H5 enregistré → skip propre.
	cron := scheduler.NewSpartanCustomizationCron(cfg, pl, provider, "halo_infinite", 0).
		WithRegistry(twoTitleRegistry("halo_5"))
	cron.RunOnce(context.Background()) // no panic

	if fetcher.calls.Load() != 1 {
		t.Errorf("seul HINF doit être rafraîchi: got %d calls, want 1", fetcher.calls.Load())
	}
}

// TestSpartanCron_WithRefresher_NilSafe : WithRefresher ignore slug vide / refresher
// nil sans paniquer ni enregistrer une entrée fantôme.
func TestSpartanCron_WithRefresher_NilSafe(t *testing.T) {
	cron := scheduler.NewSpartanCustomizationCron(nil, nil, nil, "halo_infinite", 0)
	cron.WithRefresher("", func(context.Context, domain.PlayerSummary) error { return nil })
	cron.WithRefresher("halo_5", nil)
	cron.RunOnce(context.Background()) // cfg nil → no-op, no panic
}
