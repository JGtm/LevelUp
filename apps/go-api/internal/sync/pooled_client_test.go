package sync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
)

// mockPool implémente pool.Pool pour les tests.
type mockPool struct {
	tokens           map[string]*domain.HaloTokens // gamertag → tokens
	err              error
	onHTTPErrorCalls []int         // Track statusCode values for OnHTTPError calls
	lastRetryAfter   time.Duration // dernier retryAfter passé à OnHTTPError
	slotLimiter      *rate.Limiter // Si non-nil, populé dans Lease.Limiter (Option 2)
	markUnhealthy    []string      // gamertags passés à MarkUnhealthy (RC-1 401/403)
	on429Gamertags   []string      // gamertags passés à On429ForToken (429 per-token)
}

func (m *mockPool) Acquire(ctx context.Context, policy pool.AcquirePolicy, pinnedGamertag string) (*pool.Lease, error) {
	if m.err != nil {
		return nil, m.err
	}

	var gt string
	var tokens *domain.HaloTokens

	switch policy {
	case pool.PolicyAnyPublic:
		// Retourner le premier gamertag disponible.
		for g, t := range m.tokens {
			gt = g
			tokens = t
			break
		}
	case pool.PolicyPinnedPlayer:
		if pinnedGamertag == "" {
			return nil, errors.New("pinnedGamertag vide")
		}
		var ok bool
		tokens, ok = m.tokens[pinnedGamertag]
		if !ok {
			return nil, errors.New("gamertag not found")
		}
		gt = pinnedGamertag
	}

	if tokens == nil {
		return nil, errors.New("no tokens available")
	}

	return &pool.Lease{
		Tokens:   tokens,
		Gamertag: gt,
		Release:  func() {},
		Limiter:  m.slotLimiter,
	}, nil
}

func (m *mockPool) Size() int {
	return len(m.tokens)
}

func (m *mockPool) HasPlayer(gamertag string) bool {
	_, ok := m.tokens[gamertag]
	return ok
}

func (m *mockPool) MarkUnhealthy(gamertag string, reason error) {
	m.markUnhealthy = append(m.markUnhealthy, gamertag)
}

func (m *mockPool) OnHTTPError(statusCode int, retryAfter time.Duration) {
	m.onHTTPErrorCalls = append(m.onHTTPErrorCalls, statusCode)
	m.lastRetryAfter = retryAfter
}

func (m *mockPool) On429ForToken(gamertag string, retryAfter time.Duration) {
	m.on429Gamertags = append(m.on429Gamertags, gamertag)
	m.lastRetryAfter = retryAfter
}

func (m *mockPool) AddOrUpdateSource(_ context.Context, _ pool.CredentialSource) error {
	return nil
}

func (m *mockPool) Close() {
	// no-op for tests
}

// testTokens crée des tokens de test pour un gamertag.
func testTokens(gamertag string) *domain.HaloTokens {
	return &domain.HaloTokens{
		SpartanToken:   "spartan_" + gamertag,
		ClearanceToken: "clearance_" + gamertag,
	}
}

// TestPooledHaloClientGetMatchHistory teste GetMatchHistory avec PolicyAnyPublic.
func TestPooledHaloClientGetMatchHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: fait de vrais appels réseau avec backoff (>120s en -short)")
	}
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Alice": testTokens("Alice"),
		},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	ctx := context.Background()
	// Appel va utiliser PolicyAnyPublic et acquérir le token "Alice".
	// La requête échouera probablement (pas de vrai API), mais on teste juste que Acquire fonctionne.
	_, err := client.GetMatchHistory(ctx, "Bob", "all", 0, 25)
	// Erreur attendue car pas de vrai API, mais le pool fonctionne.
	if err == nil {
		t.Fatal("expected error (no real API), got nil")
	}
	// Vérifier que l'erreur ne vient pas du pool.
	if err.Error() == "pooled: Acquire failed: no tokens available" {
		t.Fatalf("unexpected pool error: %v", err)
	}
}

// TestPooledHaloClientGetMatchStats teste GetMatchStats avec PolicyAnyPublic.
func TestPooledHaloClientGetMatchStats(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Bob": testTokens("Bob"),
		},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	ctx := context.Background()
	_, err := client.GetMatchStats(ctx, "match-id-123")
	if err == nil {
		t.Fatal("expected error (no real API), got nil")
	}
	if err.Error() == "pooled: Acquire failed: no tokens available" {
		t.Fatalf("unexpected pool error: %v", err)
	}
}

// TestPooledHaloClientGetCareerRank_PinnedToken vérifie que la branche pinned
// est bien empruntée (Acquire avec PolicyPinnedPlayer + gamertag) sans faire
// d'I/O réelle vers economy.svc.halowaypoint.com. On utilise un context déjà
// annulé pour court-circuiter doGet : l'erreur attendue est une erreur réseau
// (context canceled) — surtout PAS une erreur du pool, ce qui confirme que
// Acquire a réussi et que la requête HTTP a été tentée.
func TestPooledHaloClientGetCareerRank_PinnedToken(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Alice": testTokens("Alice"),
		},
	}
	client := NewPooledHaloClient(mp, "Alice", "xuid_alice", 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetCareerRank(ctx, "xuid_alice")
	if err == nil {
		t.Fatal("expected network error (context canceled), got nil")
	}
	if err.Error() == "pooled: Acquire failed: no tokens available" {
		t.Fatalf("unexpected pool error (Acquire devait réussir): %v", err)
	}
}

// TestPooledHaloClientGetCareerRank_NoPinnedToken teste GetCareerRank sans pinned token.
func TestPooledHaloClientGetCareerRank_NoPinnedToken(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Alice": testTokens("Alice"),
		},
	}
	// Pas de pinned token.
	client := NewPooledHaloClient(mp, "", "", 0)

	ctx := context.Background()
	result, err := client.GetCareerRank(ctx, "xuid_alice")
	// Doit retourner (nil, nil) sans appeler le pool.
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// TestPooledHaloClientGetCareerRank_PoolError teste GetCareerRank avec erreur pool.
func TestPooledHaloClientGetCareerRank_PoolError(t *testing.T) {
	mp := &mockPool{
		tokens: make(map[string]*domain.HaloTokens),
		err:    errors.New("no healthy tokens"),
	}
	client := NewPooledHaloClient(mp, "Alice", "xuid_alice", 0)

	ctx := context.Background()
	result, err := client.GetCareerRank(ctx, "xuid_alice")
	// Doit retourner (nil, nil) si le pool échoue.
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if err != nil {
		t.Errorf("expected nil error on pool failure, got %v", err)
	}
}

// TestPooledHaloClientAcquireFailure teste le comportement en cas d'erreur Acquire.
func TestPooledHaloClientAcquireFailure(t *testing.T) {
	mp := &mockPool{
		tokens: make(map[string]*domain.HaloTokens),
		err:    errors.New("no tokens available"),
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	ctx := context.Background()
	_, err := client.GetMatchHistory(ctx, "Bob", "all", 0, 25)
	if err == nil {
		t.Fatal("expected error from Acquire failure, got nil")
	}
	if err.Error() != "pooled: Acquire failed: no tokens available" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestPooledHaloClientInterface vérifie que PooledHaloClient implémente HaloClient.
func TestPooledHaloClientInterface(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	// Vérifier que client implémente HaloClient.
	var _ HaloClient = client
}

// TestPooledHaloClientNotifyHTTPError_429 teste qu'un 429 route un cooldown
// PER-TOKEN (On429ForToken sur le gamertag du lease fautif), PAS un cooldown
// global (OnHTTPError) — fini le scorched-earth.
func TestPooledHaloClientNotifyHTTPError_429(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
		onHTTPErrorCalls: []int{},
	}
	client := NewPooledHaloClient(mp, "test", "xuid_test", 0)

	// Simuler un 429 HTTP error sur le token "Alice".
	err := &HTTPError{
		StatusCode: 429,
		URL:        "https://example.com/api",
		Err:        errors.New("rate limited"),
	}
	client.notifyPoolOnError(&pool.Lease{Gamertag: "Alice"}, err)

	// Cooldown PER-TOKEN sur Alice, aucun cooldown global.
	if len(mp.on429Gamertags) != 1 || mp.on429Gamertags[0] != "Alice" {
		t.Errorf("expected On429ForToken(Alice), got %v", mp.on429Gamertags)
	}
	if len(mp.onHTTPErrorCalls) != 0 {
		t.Errorf("expected 0 OnHTTPError (global) call sur un 429 par-token, got %d", len(mp.onHTTPErrorCalls))
	}
}

// TestPooledHaloClientNotifyHTTPError_503 teste que 503 signal le pool.
func TestPooledHaloClientNotifyHTTPError_503(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
		onHTTPErrorCalls: []int{},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	// Simuler un 503 HTTP error.
	err := &HTTPError{
		StatusCode: 503,
		URL:        "https://example.com/api",
		Err:        errors.New("service unavailable"),
	}
	client.notifyPoolOnError(nil, err)

	// Vérifier que pool.OnHTTPError a été appelée avec 503.
	if len(mp.onHTTPErrorCalls) != 1 {
		t.Errorf("expected 1 OnHTTPError call, got %d", len(mp.onHTTPErrorCalls))
	}
	if len(mp.onHTTPErrorCalls) > 0 && mp.onHTTPErrorCalls[0] != 503 {
		t.Errorf("expected statusCode 503, got %d", mp.onHTTPErrorCalls[0])
	}
}

// TestPooledHaloClientNotifyHTTPError_OtherStatus teste que d'autres codes sont ignorés.
func TestPooledHaloClientNotifyHTTPError_OtherStatus(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
		onHTTPErrorCalls: []int{},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	// Simuler un 500 HTTP error (non-429/503).
	err := &HTTPError{
		StatusCode: 500,
		URL:        "https://example.com/api",
		Err:        errors.New("internal server error"),
	}
	client.notifyPoolOnError(nil, err)

	// Vérifier que pool.OnHTTPError n'a PAS été appelée.
	if len(mp.onHTTPErrorCalls) != 0 {
		t.Errorf("expected 0 OnHTTPError calls for status 500, got %d", len(mp.onHTTPErrorCalls))
	}
}

// makeLease forge un *pool.Lease pour les tests internes de newAPIClient.
func makeLease(spartan, clearance string, limiter *rate.Limiter) *pool.Lease {
	return &pool.Lease{
		Tokens:   &domain.HaloTokens{SpartanToken: spartan, ClearanceToken: clearance},
		Gamertag: "test",
		Release:  func() {},
		Limiter:  limiter,
	}
}

// TestPooledHaloClient_FallbackLimiter valide le path fallback : quand le
// Lease ne porte pas de Limiter (cas legacy / mock minimal), newAPIClient
// utilise fallbackLimiter — tous les HaloAPIClient le partagent.
func TestPooledHaloClient_FallbackLimiter(t *testing.T) {
	mp := &mockPool{tokens: map[string]*domain.HaloTokens{"alice": testTokens("alice")}}
	const rps = 20
	const n = 10
	pc := NewPooledHaloClient(mp, "", "", rps)

	// Sanity : Lease.Limiter nil → newAPIClient retombe sur fallbackLimiter.
	leaseNil := makeLease("s1", "c1", nil)
	c1 := pc.newAPIClient(leaseNil)
	c2 := pc.newAPIClient(makeLease("s2", "c2", nil))
	if c1.limiter != c2.limiter || c1.limiter != pc.fallbackLimiter {
		t.Fatal("avec Lease.Limiter nil, les HaloAPIClient doivent partager pc.fallbackLimiter")
	}

	// Concurrence : N goroutines via fallback. Throughput ≤ rps.
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			c := pc.newAPIClient(makeLease("s", "c", nil))
			c.rateWait(context.Background())
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	minExpected := time.Duration(n-1) * time.Second / time.Duration(rps) * 80 / 100
	if elapsed < minExpected {
		t.Fatalf("fallback limiter inopérant : %d goroutines à %d RPS terminées en %v (attendu ≥ %v)",
			n, rps, elapsed, minExpected)
	}
}

// TestPooledHaloClient_LeaseLimiterPriority valide l'invariant Option 2 : si
// le Lease porte un Limiter (pool en prod), newAPIClient l'utilise au lieu du
// fallback. Régression du bug pré-Option 1+2 où chaque newAPIClient créait un
// limiter neuf (burst=1 plein) → rate-limit inopérant.
func TestPooledHaloClient_LeaseLimiterPriority(t *testing.T) {
	const slotRPS = 20
	slotLim := rate.NewLimiter(rate.Limit(slotRPS), 1)
	mp := &mockPool{
		tokens:      map[string]*domain.HaloTokens{"alice": testTokens("alice")},
		slotLimiter: slotLim,
	}
	pc := NewPooledHaloClient(mp, "", "", 100) // fallback 100 RPS - doit NE PAS être utilisé

	// Sanity : Lease.Limiter prend le pas sur fallbackLimiter.
	lease, err := mp.Acquire(context.Background(), pool.PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("mockPool.Acquire: %v", err)
	}
	c := pc.newAPIClient(lease)
	if c.limiter != slotLim {
		t.Fatal("newAPIClient doit utiliser Lease.Limiter (slot) et pas le fallback")
	}
	if c.limiter == pc.fallbackLimiter {
		t.Fatal("newAPIClient ne doit PAS utiliser fallbackLimiter quand Lease.Limiter est non-nil")
	}

	// Concurrence : N goroutines via slotLimiter. Throughput borné par slotRPS,
	// pas par le fallback (100 RPS).
	const n = 10
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			lease, _ := mp.Acquire(context.Background(), pool.PolicyAnyPublic, "")
			c := pc.newAPIClient(lease)
			c.rateWait(context.Background())
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	minExpected := time.Duration(n-1) * time.Second / time.Duration(slotRPS) * 80 / 100
	if elapsed < minExpected {
		t.Fatalf("lease.Limiter inopérant : %d goroutines à %d RPS slot terminées en %v (attendu ≥ %v) — le fallback à 100 RPS a peut-être été utilisé",
			n, slotRPS, elapsed, minExpected)
	}
}

// TestPooledHaloClientNotifyHTTPError_NilError teste qu'une erreur nil est ignorée.
func TestPooledHaloClientNotifyHTTPError_NilError(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
		onHTTPErrorCalls: []int{},
	}
	client := NewPooledHaloClient(mp, "", "", 0)

	// Passer nil comme erreur.
	client.notifyPoolOnError(nil, nil)

	// Vérifier que pool.OnHTTPError n'a pas été appelée.
	if len(mp.onHTTPErrorCalls) != 0 {
		t.Errorf("expected 0 OnHTTPError calls for nil error, got %d", len(mp.onHTTPErrorCalls))
	}
}

// TestPooledHaloClientNotifyError_401_MarksUnhealthy : RC-1 — un 401/403 marque
// unhealthy le slot du LEASE (et non pinnedGamertag, vide en round-robin), sans
// déclencher de cooldown global (réservé 429/503).
func TestPooledHaloClientNotifyError_401_MarksUnhealthy(t *testing.T) {
	for _, code := range []int{401, 403} {
		mp := &mockPool{
			tokens:           map[string]*domain.HaloTokens{"slotA": testTokens("slotA")},
			onHTTPErrorCalls: []int{},
		}
		client := NewPooledHaloClient(mp, "", "", 0)
		lease := &pool.Lease{Gamertag: "slotA", Tokens: testTokens("slotA"), Release: func() {}}
		err := &HTTPError{StatusCode: code, URL: "https://example.com/api", Err: errors.New("auth refused")}

		client.notifyPoolOnError(lease, err)

		if len(mp.markUnhealthy) != 1 || mp.markUnhealthy[0] != "slotA" {
			t.Errorf("code %d : MarkUnhealthy attendu sur 'slotA', got %v", code, mp.markUnhealthy)
		}
		if len(mp.onHTTPErrorCalls) != 0 {
			t.Errorf("code %d : aucun cooldown global attendu sur 401/403, got %v", code, mp.onHTTPErrorCalls)
		}
		if !isAuthError(err) {
			t.Errorf("isAuthError(%d) devrait être true", code)
		}
	}
}
