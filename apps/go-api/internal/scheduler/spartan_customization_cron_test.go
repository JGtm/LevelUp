// Package scheduler — spartan_customization_cron_test.go : tests unitaires
// du cron customization V2 Phase 8.
package scheduler_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/scheduler"
)

// mockSpartanFetcher compte les invocations + retourne un identity bidon.
type mockSpartanFetcher struct {
	calls atomic.Int32
}

func (m *mockSpartanFetcher) GetSpartanIdentity(_ context.Context) (*domain.HomeSpartanIdentityRow, error) {
	m.calls.Add(1)
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
func (m *fakeCronPool) Size() int                          { return len(m.hasPlayerMap) }
func (m *fakeCronPool) HasPlayer(gt string) bool           { return m.hasPlayerMap[gt] }
func (m *fakeCronPool) MarkUnhealthy(_ string, _ error)    {}
func (m *fakeCronPool) OnHTTPError(_ int, _ time.Duration) {}
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
