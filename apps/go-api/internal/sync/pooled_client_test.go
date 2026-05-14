package sync

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
)

// mockPool implémente pool.Pool pour les tests.
type mockPool struct {
	tokens           map[string]*domain.HaloTokens // gamertag → tokens
	err              error
	onHTTPErrorCalls []int // Track statusCode values for OnHTTPError calls
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
	// no-op for tests
}

func (m *mockPool) OnHTTPError(statusCode int) {
	m.onHTTPErrorCalls = append(m.onHTTPErrorCalls, statusCode)
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
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Alice": testTokens("Alice"),
		},
	}
	client := NewPooledHaloClient(mp, "", "")

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
	client := NewPooledHaloClient(mp, "", "")

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
	client := NewPooledHaloClient(mp, "Alice", "xuid_alice")

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
	client := NewPooledHaloClient(mp, "", "")

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
	client := NewPooledHaloClient(mp, "Alice", "xuid_alice")

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
	client := NewPooledHaloClient(mp, "", "")

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
	client := NewPooledHaloClient(mp, "", "")

	// Vérifier que client implémente HaloClient.
	var _ HaloClient = client
}

// TestPooledHaloClientNotifyHTTPError_429 teste que 429 signal le pool.
func TestPooledHaloClientNotifyHTTPError_429(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"test": testTokens("test"),
		},
		onHTTPErrorCalls: []int{},
	}
	client := NewPooledHaloClient(mp, "test", "xuid_test")

	// Simuler un 429 HTTP error.
	err := &HTTPError{
		StatusCode: 429,
		URL:        "https://example.com/api",
		Err:        errors.New("rate limited"),
	}
	client.notifyPoolOnHTTPError(err)

	// Vérifier que pool.OnHTTPError a été appelée avec 429.
	if len(mp.onHTTPErrorCalls) != 1 {
		t.Errorf("expected 1 OnHTTPError call, got %d", len(mp.onHTTPErrorCalls))
	}
	if len(mp.onHTTPErrorCalls) > 0 && mp.onHTTPErrorCalls[0] != 429 {
		t.Errorf("expected statusCode 429, got %d", mp.onHTTPErrorCalls[0])
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
	client := NewPooledHaloClient(mp, "", "")

	// Simuler un 503 HTTP error.
	err := &HTTPError{
		StatusCode: 503,
		URL:        "https://example.com/api",
		Err:        errors.New("service unavailable"),
	}
	client.notifyPoolOnHTTPError(err)

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
	client := NewPooledHaloClient(mp, "", "")

	// Simuler un 500 HTTP error (non-429/503).
	err := &HTTPError{
		StatusCode: 500,
		URL:        "https://example.com/api",
		Err:        errors.New("internal server error"),
	}
	client.notifyPoolOnHTTPError(err)

	// Vérifier que pool.OnHTTPError n'a PAS été appelée.
	if len(mp.onHTTPErrorCalls) != 0 {
		t.Errorf("expected 0 OnHTTPError calls for status 500, got %d", len(mp.onHTTPErrorCalls))
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
	client := NewPooledHaloClient(mp, "", "")

	// Passer nil comme erreur.
	client.notifyPoolOnHTTPError(nil)

	// Vérifier que pool.OnHTTPError n'a pas été appelée.
	if len(mp.onHTTPErrorCalls) != 0 {
		t.Errorf("expected 0 OnHTTPError calls for nil error, got %d", len(mp.onHTTPErrorCalls))
	}
}
