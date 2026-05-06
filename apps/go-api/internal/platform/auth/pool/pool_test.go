package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// testSlotEnv crée un set de sources pour les tests.
func testSlotEnv(count int) []CredentialSource {
	sources := make([]CredentialSource, count)
	for i := 0; i < count; i++ {
		sources[i] = CredentialSource{
			Gamertag:  string([]byte{byte('A') + byte(i)}),
			XUID:      string(rune(1000 + i)),
			MSALCache: "cache_" + string(rune('A'+i)),
			Source:    "test",
		}
	}
	return sources
}

// testResolver implémente Resolver avec une map mock de tokens.
type testResolver struct {
	resolved map[string]*ResolvedTokens
	mu       sync.Mutex
}

func (tr *testResolver) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if r, ok := tr.resolved[src.Gamertag]; ok {
		return r, nil
	}
	// Créer un token de test.
	return &ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_" + src.Gamertag},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Source:    src.Source,
	}, nil
}

func (tr *testResolver) Refresh(ctx context.Context, gamertag string) (*ResolvedTokens, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	// Retourner un token "rafraîchi" avec un nouveau timestamp.
	return &ResolvedTokens{
		Gamertag:  gamertag,
		XUID:      "xuid_" + gamertag,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_refreshed_" + gamertag},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Source:    "test",
	}, nil
}

// TestPoolNewPool_Success teste la création d'un pool.
func TestPoolNewPool_Success(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 3 {
		t.Errorf("expected pool size 3, got %d", pool.Size())
	}
}

// TestPoolNewPool_EmptySources teste la création avec aucune source.
func TestPoolNewPool_EmptySources(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{}
	_, err := NewPool(context.Background(), resolver, []CredentialSource{}, opts)
	if err == nil {
		t.Fatal("expected error for empty sources, got nil")
	}
}

// TestPoolNewPool_MaxSize teste la limitation de taille.
func TestPoolNewPool_MaxSize(t *testing.T) {
	sources := testSlotEnv(5)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 2, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 2 {
		t.Errorf("expected pool size 2 (MaxSize=2), got %d", pool.Size())
	}
}

// TestPoolAcquireAnyPublic teste la politique round-robin.
func TestPoolAcquireAnyPublic(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquérir 3 leases — doivent correspondre à 3 gamertags différents.
	acquired := make(map[string]bool)
	for i := 0; i < 3; i++ {
		lease, err := pool.Acquire(ctx, PolicyAnyPublic, "")
		if err != nil {
			t.Fatalf("Acquire %d failed: %v", i, err)
		}
		acquired[lease.Gamertag] = true
		lease.Release()
	}

	if len(acquired) != 3 {
		t.Errorf("expected 3 distinct gamertags, got %d", len(acquired))
	}
}

// TestPoolAcquirePinnedPlayer teste la politique pinned.
func TestPoolAcquirePinnedPlayer(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquérir le token pinné pour 'A'.
	lease, err := pool.Acquire(ctx, PolicyPinnedPlayer, "A")
	if err != nil {
		t.Fatalf("Acquire pinned failed: %v", err)
	}
	if lease.Gamertag != "A" {
		t.Errorf("expected gamertag A, got %s", lease.Gamertag)
	}
	lease.Release()

	// Acquérir un gamertag inexistant doit échouer.
	_, err = pool.Acquire(ctx, PolicyPinnedPlayer, "Unknown")
	if err == nil {
		t.Fatal("expected error for unknown gamertag, got nil")
	}
}

// TestPoolMarkUnhealthy teste l'invalidation et la marque malsain.
func TestPoolMarkUnhealthy(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquérir un token et le marquer malsain.
	lease1, err := pool.Acquire(ctx, PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	gamertag1 := lease1.Gamertag
	lease1.Release()

	// Marquer comme malsain.
	pool.MarkUnhealthy(gamertag1, errors.New("401 unauthorized"))

	// Acquérir à nouveau — doit obtenir un autre token (pas celui-ci).
	lease2, err := pool.Acquire(ctx, PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire 2 failed: %v", err)
	}
	defer lease2.Release()

	if lease2.Gamertag == gamertag1 {
		t.Errorf("expected different gamertag after MarkUnhealthy, got %s again", gamertag1)
	}

	// Tenter d'acquérir le gamertag malsain avec PolicyPinnedPlayer doit échouer.
	_, err = pool.Acquire(ctx, PolicyPinnedPlayer, gamertag1)
	if err == nil {
		t.Fatal("expected error for unhealthy pinned player, got nil")
	}
}

// TestPoolConcurrentAcquire teste les accès concurrents sans race conditions.
func TestPoolConcurrentAcquire(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	errCount := atomic.Int32{}

	// 20 goroutines qui acquièrent/release en parallèle.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := pool.Acquire(ctx, PolicyAnyPublic, "")
			if err != nil {
				errCount.Add(1)
				return
			}
			time.Sleep(1 * time.Millisecond) // Simule un peu de travail
			lease.Release()
		}()
	}

	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("expected 0 acquire errors, got %d", errCount.Load())
	}
}

// TestPoolAcquireWithContextTimeout teste le timeout du contexte.
func TestPoolAcquireWithContextTimeout(t *testing.T) {
	sources := testSlotEnv(1)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	// Acquérir et garder le lease.
	lease1, err := pool.Acquire(context.Background(), PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire 1 failed: %v", err)
	}

	// Tenter d'acquérir avec timeout court (sera bloqué car 1 seul slot).
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx, PolicyAnyPublic, "")
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}

	lease1.Release()
}

// TestPoolSize teste la méthode Size().
func TestPoolSize(t *testing.T) {
	sources := testSlotEnv(5)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 3, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 3 {
		t.Errorf("expected Size() = 3, got %d", pool.Size())
	}
}

// TestPoolRoundRobinDistribution teste la distribution équitable du round-robin.
func TestPoolRoundRobinDistribution(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	counts := make(map[string]int)
	countMu := sync.Mutex{}

	// Acquérir 30 leases et compter la distribution.
	for i := 0; i < 30; i++ {
		lease, err := pool.Acquire(ctx, PolicyAnyPublic, "")
		if err != nil {
			t.Fatalf("Acquire failed: %v", err)
		}
		countMu.Lock()
		counts[lease.Gamertag]++
		countMu.Unlock()
		lease.Release()
	}

	// Chaque gamertag doit avoir environ 10 acquisitions.
	// Accepter une petite variance due au timing.
	for gt, count := range counts {
		if count < 8 || count > 12 {
			t.Errorf("gamertag %s: expected ~10, got %d (variance too high)", gt, count)
		}
	}
}

// TestPoolDefaultOptions teste les valeurs par défaut des options.
func TestPoolDefaultOptions(t *testing.T) {
	sources := testSlotEnv(1)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	// Passer des options vides.
	opts := PoolOptions{}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	// Vérifier que les defaults ont été appliqués.
	if pool.Size() != 1 {
		t.Errorf("expected size 1, got %d", pool.Size())
	}
}
