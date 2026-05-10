package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mocks ---

type mockCompareRepo struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
	xuid     string
	xuidErr  error
}

func (m *mockCompareRepo) GetLocalStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

func (m *mockCompareRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}

type mockStatsProvider struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
}

func (m *mockStatsProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

// --- tests ---

func TestCompareService_BothLocal(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  100,
		WinRate:  0.60,
		KDR:      1.5,
	}
	statsB := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerB",
		Matches:  80,
		WinRate:  0.50,
		KDR:      1.2,
	}

	callCount := 0
	repo := &mockCompareRepo{
		xuid: "xuid-b",
	}
	// GetLocalStats retourne A pour xuid "xuid-a", B pour xuid "xuid-b"
	repo.stats = statsA

	provider := &mockStatsProvider{statsErr: errors.New("not needed")}

	svc := NewCompareService(&mockCompareRepoAB{a: statsA, b: statsB}, provider, "xuid-a", "hi")
	_ = callCount

	resp, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PlayerA.Gamertag != "PlayerA" {
		t.Errorf("expected PlayerA, got %s", resp.PlayerA.Gamertag)
	}
	if len(resp.Metrics) == 0 {
		t.Error("expected metrics to be populated")
	}
}

func TestCompareService_PlayerBNotFound(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  50,
	}

	repo := &mockCompareRepoAB{
		a:       statsA,
		bErr:    errors.New("not found"),
		xuidErr: errors.New("not found"),
	}
	provider := &mockStatsProvider{statsErr: errors.New("not found")}

	svc := NewCompareService(repo, provider, "xuid-a", "hi")
	_, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "UnknownPlayer"})
	if err == nil {
		t.Error("expected error when player B not found, got nil")
	}
}

// mockCompareRepoAB — retourne stats différentes selon le xuid demandé.
type mockCompareRepoAB struct {
	a       *domain.NormalizedPlayerStats
	b       *domain.NormalizedPlayerStats
	bErr    error
	xuid    string
	xuidErr error
	mu      sync.Mutex
	calls   int
}

func (m *mockCompareRepoAB) GetLocalStats(_ context.Context, xuid, _ string) (*domain.NormalizedPlayerStats, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if xuid == "xuid-b" || (m.a != nil && xuid != "xuid-a") {
		if m.bErr != nil {
			return nil, m.bErr
		}
		return m.b, nil
	}
	return m.a, nil
}

func (m *mockCompareRepoAB) ResolveXUID(_ context.Context, _ string) (string, error) {
	if m.xuidErr != nil {
		return "", m.xuidErr
	}
	if m.xuid != "" {
		return m.xuid, nil
	}
	return "xuid-b", nil
}

func (m *mockCompareRepoAB) GetPlayerATH(_ context.Context) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (m *mockCompareRepoAB) GetPlayerATHFor(_ context.Context, _, _ string) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (m *mockCompareRepoAB) GetFavoriteWeapon(_ context.Context, _ string) (*domain.WeaponHighlight, error) {
	return nil, nil
}

// ─── F5 : Test de latence Compare P95 < 5s ───────────────────────────────────

// slowProvider simule une latence Waypoint configurable.
type slowProvider struct {
	delay    time.Duration
	stats    *domain.NormalizedPlayerStats
	statsErr error
}

func (s *slowProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	time.Sleep(s.delay)
	return s.stats, s.statsErr
}

// TestCompareService_Latency_P95 vérifie que GetPage s'exécute en < 5s
// même quand le provider Waypoint répond après un délai simulé.
// Le test est répété N fois pour estimer le P95.
func TestCompareService_Latency_P95(t *testing.T) {
	const N = 20
	const maxP95 = 5 * time.Second
	const simulatedDelay = 50 * time.Millisecond // latence mock — pas de vrai réseau

	statsA := &domain.NormalizedPlayerStats{Gamertag: "PlayerA", Matches: 100, WinRate: 0.6}
	statsB := &domain.NormalizedPlayerStats{Gamertag: "PlayerB", Matches: 80, WinRate: 0.5}

	repo := &mockCompareRepoAB{a: statsA, xuidErr: errors.New("not found")}
	provider := &slowProvider{delay: simulatedDelay, stats: statsB}
	svc := NewCompareService(repo, provider, "xuid-a", "halo_infinite")

	durations := make([]time.Duration, N)
	for i := range N {
		start := time.Now()
		_, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
		durations[i] = time.Since(start)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}

	// Calcul P95 : trier et prendre le 95e percentile.
	sorted := make([]time.Duration, N)
	copy(sorted, durations)
	sortDurations(sorted)
	p95idx := int(float64(N)*0.95) - 1
	if p95idx < 0 {
		p95idx = 0
	}
	p95 := sorted[p95idx]

	if p95 > maxP95 {
		t.Errorf("P95 latence Compare = %v, dépasse le seuil de %v", p95, maxP95)
	}
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}
