// Package scheduler — spartan_customization_cron_test.go : tests unitaires
// du cron customization V2 Phase 8.
package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
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
	// fetcher.calls peut être 0 ou plus selon les players retournés (souvent 0 ici).
}
