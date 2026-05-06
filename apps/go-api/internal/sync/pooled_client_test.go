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
	tokens map[string]*domain.HaloTokens // gamertag → tokens
	err    error
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

func (m *mockPool) MarkUnhealthy(gamertag string, reason error) {
	// no-op for tests
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

// TestPooledHaloClientGetCareerRank_PinnedToken teste GetCareerRank avec pinned token.
func TestPooledHaloClientGetCareerRank_PinnedToken(t *testing.T) {
	mp := &mockPool{
		tokens: map[string]*domain.HaloTokens{
			"Alice": testTokens("Alice"),
		},
	}
	client := NewPooledHaloClient(mp, "Alice", "xuid_alice")

	ctx := context.Background()
	_, err := client.GetCareerRank(ctx, "xuid_alice")
	// Erreur attendue car pas de vrai API.
	if err == nil {
		t.Fatal("expected error (no real API), got nil")
	}
	if err.Error() == "pooled: Acquire failed: no tokens available" {
		t.Fatalf("unexpected pool error: %v", err)
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
