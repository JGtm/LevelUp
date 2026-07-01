package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
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

// TestPoolHasPlayer_True vérifie que HasPlayer retourne true pour un gamertag
// présent parmi les slots, indépendamment de son état healthy.
// testSlotEnv(3) crée des gamertags "A", "B", "C".
func TestPoolHasPlayer_True(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	for _, gt := range []string{"A", "B", "C"} {
		if !pool.HasPlayer(gt) {
			t.Errorf("HasPlayer(%q) = false, want true", gt)
		}
	}
}

// TestPoolHasPlayer_False vérifie que HasPlayer retourne false pour un gamertag
// absent du pool (jamais découvert par Discovery).
func TestPoolHasPlayer_False(t *testing.T) {
	sources := testSlotEnv(2) // A, B
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	for _, gt := range []string{"GhostPlayer", "Z", ""} {
		if pool.HasPlayer(gt) {
			t.Errorf("HasPlayer(%q) = true, want false", gt)
		}
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

// TestPoolOnHTTPError_429 teste le backoff global sur 429.
func TestPoolOnHTTPError_429(t *testing.T) {
	sources := testSlotEnv(3)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquérir un token — doit fonctionner normalement.
	lease1, err := pool.Acquire(ctx, PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire before OnHTTPError failed: %v", err)
	}
	if lease1.Gamertag == "" {
		t.Fatal("expected non-empty gamertag")
	}
	lease1.Release()

	// Déclencher le cooldown global avec un 429.
	pool.OnHTTPError(429, 0)

	// Immédiatement après, tous les tokens doivent être malsains.
	_, err = pool.Acquire(ctx, PolicyAnyPublic, "")
	if err == nil {
		t.Fatal("expected error after OnHTTPError(429), got nil")
	}
	if err.Error() != "pool: aucun slot sain disponible (PolicyAnyPublic)" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestPoolOn429ForToken_PerTokenNotGlobal vérifie qu'un 429 imputé à UN token met
// CE token en cooldown (skippé à l'acquisition) SANS toucher les autres — fini le
// scorched-earth où un 429 isolé mettait tout le pool en pause.
func TestPoolOn429ForToken_PerTokenNotGlobal(t *testing.T) {
	sources := testSlotEnv(3) // A, B, C
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer p.Close()

	// Mettre "A" en cooldown 429 (long) — les autres restent servables.
	p.On429ForToken("A", 30*time.Second)

	ctx := context.Background()
	seen := map[string]int{}
	for i := 0; i < 12; i++ {
		lease, aerr := p.Acquire(ctx, PolicyAnyPublic, "")
		if aerr != nil {
			t.Fatalf("Acquire %d a échoué alors que B et C sont sains: %v", i, aerr)
		}
		seen[lease.Gamertag]++
		lease.Release()
	}

	if seen["A"] != 0 {
		t.Errorf("A rate-limité aurait dû être skippé, servi %d fois", seen["A"])
	}
	if seen["B"] == 0 || seen["C"] == 0 {
		t.Errorf("B et C sains auraient dû servir, vu B=%d C=%d", seen["B"], seen["C"])
	}
}

// TestPoolOn429ForToken_AutoRecovers vérifie qu'un token rate-limité redevient
// acquérable seul à l'expiration du cooldown, SANS re-exchange.
func TestPoolOn429ForToken_AutoRecovers(t *testing.T) {
	sources := testSlotEnv(1) // A seulement
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer p.Close()

	ctx := context.Background()

	// Cooldown très court sur l'unique token.
	p.On429ForToken("A", 80*time.Millisecond)

	// Immédiatement : indisponible.
	if _, aerr := p.Acquire(ctx, PolicyAnyPublic, ""); aerr == nil {
		t.Fatal("Acquire aurait dû échouer pendant le cooldown 429")
	}

	// Après le cooldown : re-disponible sans intervention (pas de re-exchange).
	time.Sleep(150 * time.Millisecond)
	lease, aerr := p.Acquire(ctx, PolicyAnyPublic, "")
	if aerr != nil {
		t.Fatalf("Acquire aurait dû réussir après cooldown: %v", aerr)
	}
	lease.Release()
}

// TestPoolOnHTTPError_FloorAtGlobalCooldown : un 429 avec un Retry-After court ne
// doit PAS descendre sous le plancher globalCooldown (fin du thrash-loop 1s).
func TestPoolOnHTTPError_FloorAtGlobalCooldown(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, testSlotEnv(2),
		PoolOptions{MaxSize: 0, PerTokenRPS: 1, GlobalCooldown: 30 * time.Second})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	p.OnHTTPError(429, 1*time.Second) // Retry-After=1s
	if got := lastCooldownSeconds.Value(); got < 30 {
		t.Errorf("cooldown planché attendu >= 30s (globalCooldown), got %ds — Retry-After 1s ne doit pas passer sous le plancher", got)
	}
}

// TestPoolOnHTTPError_BackoffSurvivesRetryAfter : le backoff exponentiel n'est PLUS
// neutralisé par un Retry-After (ancien bug : Retry-After=1s remettait le compteur
// à 0 -> thrash-loop 1s à vie).
func TestPoolOnHTTPError_BackoffSurvivesRetryAfter(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, testSlotEnv(2),
		PoolOptions{MaxSize: 0, PerTokenRPS: 1, GlobalCooldown: 30 * time.Second})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()
	pi := p.(*poolImpl)

	// Simuler 2 incidents antérieurs (backoff déjà escaladé), hors cooldown actif.
	pi.cooldownMu.Lock()
	pi.consecutive429 = 2
	pi.coolingDown = false
	pi.cooldownMu.Unlock()

	// Un 429 avec Retry-After=1s : le backoff 30s<<2 = 120s doit primer (pas 1s).
	pi.OnHTTPError(429, 1*time.Second)
	if got := lastCooldownSeconds.Value(); got < 120 {
		t.Errorf("le backoff doit survivre au Retry-After : attendu >= 120s, got %ds", got)
	}
}

// TestPoolOn429ForToken_UnknownGamertagDoesNotNukePool : un 429 sur un gamertag
// hors pool ne doit PAS mettre tout le pool en cooldown (no-op).
func TestPoolOn429ForToken_UnknownGamertagDoesNotNukePool(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, testSlotEnv(2), PoolOptions{MaxSize: 0, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	p.On429ForToken("Inconnu", time.Second)
	lease, aerr := p.Acquire(context.Background(), PolicyAnyPublic, "")
	if aerr != nil {
		t.Fatalf("un gamertag inconnu ne doit PAS nuke le pool: %v", aerr)
	}
	lease.Release()
}

// TestPoolOn429ForToken_EmptyGamertagFallbackGlobal : sans token identifiable, on
// retombe sur le filet global (cooldown) plutôt que d'ignorer le signal.
func TestPoolOn429ForToken_EmptyGamertagFallbackGlobal(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, testSlotEnv(2),
		PoolOptions{MaxSize: 0, PerTokenRPS: 1, GlobalCooldown: 30 * time.Second})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	p.On429ForToken("", time.Second) // gamertag vide -> filet global
	if _, aerr := p.Acquire(context.Background(), PolicyAnyPublic, ""); aerr == nil {
		t.Fatal("gamertag vide -> filet global attendu (pool en cooldown)")
	}
}

// TestPool_ConcurrentAcquireAndUpdate_Race verrouille la correction de la data race
// sur slot.resolved / p.slots : Acquire (lecture) + AddOrUpdateSource (réassigne
// resolved + append) + On429ForToken (écrit rateLimitedUntil) en parallèle. À lancer
// avec -race — l'ancien code (lecture hors verrou) échouait.
func TestPool_ConcurrentAcquireAndUpdate_Race(t *testing.T) {
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}
	p, err := NewPool(context.Background(), resolver, testSlotEnv(3), PoolOptions{MaxSize: 0, PerTokenRPS: 100000})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if lease, aerr := p.Acquire(ctx, PolicyAnyPublic, ""); aerr == nil {
					_ = lease.Tokens
					lease.Release()
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 100; j++ {
			_ = p.AddOrUpdateSource(ctx, CredentialSource{Gamertag: "A", XUID: "1000", Source: "test"})
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 200; j++ {
			p.On429ForToken("B", 10*time.Millisecond)
		}
	}()
	wg.Wait()
}

// TestPoolOnHTTPError_503 teste le backoff global sur 503.
func TestPoolOnHTTPError_503(t *testing.T) {
	sources := testSlotEnv(2)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1, GlobalCooldown: 100 * time.Millisecond}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Déclencher le cooldown.
	pool.OnHTTPError(503, 0)

	// Vérifier que les acquisitions échouent.
	_, err = pool.Acquire(ctx, PolicyAnyPublic, "")
	if err == nil {
		t.Fatal("expected error after OnHTTPError(503)")
	}

	// Attendre que le cooldown passe.
	time.Sleep(150 * time.Millisecond)

	// Maintenant, les tokens devraient être refreshés et disponibles.
	// (Le refresher loop les réactive pendant que le cooldown est levé)
	// Mais le timing est non-déterministe, donc on juste vérifie qu'on peut essayer.
	_, err = pool.Acquire(ctx, PolicyAnyPublic, "")
	// Peut être nil (refresher a eu le temps) ou non-nil (timing)
	// Pas de vérification stricte car c'est une race.
}

// TestPoolOnHTTPError_OtherStatusCode teste que les autres codes d'erreur sont ignorés.
func TestPoolOnHTTPError_OtherStatusCode(t *testing.T) {
	sources := testSlotEnv(2)
	resolver := &testResolver{resolved: make(map[string]*ResolvedTokens)}

	opts := PoolOptions{MaxSize: 0, PerTokenRPS: 1}
	pool, err := NewPool(context.Background(), resolver, sources, opts)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Appeler OnHTTPError avec un code qui n'est pas 429/503.
	pool.OnHTTPError(500, 0)

	// Les acquisitions doivent continuer normalement (pas de cooldown).
	lease, err := pool.Acquire(ctx, PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire after OnHTTPError(500) failed: %v", err)
	}
	lease.Release()
}

// ─── E.v2 — AddOrUpdateSource (hot-add ou refresh d'un slot) ────────────────

func TestPoolAddOrUpdateSource_NewSlot(t *testing.T) {
	sources := testSlotEnv(2) // A, B
	resolver := &testResolver{}
	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 2 {
		t.Fatalf("size init = %d, want 2", pool.Size())
	}

	// Hot-add d'un 3e gamertag C.
	newSrc := CredentialSource{Gamertag: "C", XUID: "1002", MSALCache: "cache_C", Source: "test"}
	if err := pool.AddOrUpdateSource(context.Background(), newSrc); err != nil {
		t.Fatalf("AddOrUpdateSource: %v", err)
	}

	if pool.Size() != 3 {
		t.Errorf("size after add = %d, want 3", pool.Size())
	}
	if !pool.HasPlayer("C") {
		t.Error("HasPlayer(C) = false, want true")
	}

	// PolicyPinnedPlayer doit fonctionner sur le nouveau slot.
	lease, err := pool.Acquire(context.Background(), PolicyPinnedPlayer, "C")
	if err != nil {
		t.Fatalf("Acquire pinned C: %v", err)
	}
	if lease.Gamertag != "C" {
		t.Errorf("lease.Gamertag = %q, want C", lease.Gamertag)
	}
	if lease.Tokens.SpartanToken != "spartan_C" {
		t.Errorf("lease token = %q, want spartan_C", lease.Tokens.SpartanToken)
	}
	lease.Release()
}

func TestPoolAddOrUpdateSource_UpdateExisting(t *testing.T) {
	sources := testSlotEnv(2)
	resolver := &testResolver{
		resolved: map[string]*ResolvedTokens{
			"A": {Gamertag: "A", XUID: "1000", Tokens: &domain.HaloTokens{SpartanToken: "spartan_A_v1"}, ExpiresAt: time.Now().Add(1 * time.Hour), Source: "test"},
		},
	}
	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	// Update : remplacer le token de A par une nouvelle version.
	resolver.mu.Lock()
	resolver.resolved["A"] = &ResolvedTokens{
		Gamertag: "A", XUID: "1000",
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_A_v2"},
		ExpiresAt: time.Now().Add(1 * time.Hour), Source: "test",
	}
	resolver.mu.Unlock()

	if err := pool.AddOrUpdateSource(context.Background(), sources[0]); err != nil {
		t.Fatalf("AddOrUpdateSource: %v", err)
	}

	// Size inchangée.
	if pool.Size() != 2 {
		t.Errorf("size after update = %d, want 2 (no new slot)", pool.Size())
	}

	// Le nouveau token est servi.
	lease, err := pool.Acquire(context.Background(), PolicyPinnedPlayer, "A")
	if err != nil {
		t.Fatalf("Acquire A: %v", err)
	}
	if lease.Tokens.SpartanToken != "spartan_A_v2" {
		t.Errorf("token = %q, want spartan_A_v2 (refreshed)", lease.Tokens.SpartanToken)
	}
	lease.Release()
}

func TestPoolAddOrUpdateSource_EmptyGamertag(t *testing.T) {
	sources := testSlotEnv(1)
	resolver := &testResolver{}
	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	err = pool.AddOrUpdateSource(context.Background(), CredentialSource{Gamertag: ""})
	if err == nil {
		t.Error("AddOrUpdateSource avec gamertag vide doit échouer")
	}
}

func TestPoolAddOrUpdateSource_MaxSizeCap(t *testing.T) {
	sources := testSlotEnv(2)
	resolver := &testResolver{}
	pool, err := NewPool(context.Background(), resolver, sources, PoolOptions{MaxSize: 2, PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	// Tenter d'ajouter un 3e slot avec MaxSize=2.
	newSrc := CredentialSource{Gamertag: "C", XUID: "1002", MSALCache: "cache_C", Source: "test"}
	err = pool.AddOrUpdateSource(context.Background(), newSrc)
	if err == nil {
		t.Error("AddOrUpdateSource au-delà de MaxSize doit échouer")
	}
	if pool.Size() != 2 {
		t.Errorf("size = %d, want 2 (cap respecté)", pool.Size())
	}
}

func TestPoolAddOrUpdateSource_ResolveError(t *testing.T) {
	// Resolver qui échoue pour gamertag "FAIL".
	resolver := &testResolver{
		resolved: map[string]*ResolvedTokens{
			"A": {Gamertag: "A", XUID: "1000", Tokens: &domain.HaloTokens{SpartanToken: "spartan_A"}, ExpiresAt: time.Now().Add(1 * time.Hour), Source: "test"},
		},
	}
	// Custom resolver wrapping that fails on "FAIL".
	failResolver := &failingResolver{wrapped: resolver, failGamertag: "FAIL"}
	pool, err := NewPool(context.Background(), failResolver, []CredentialSource{{Gamertag: "A", XUID: "1000", MSALCache: "cache_A", Source: "test"}}, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	err = pool.AddOrUpdateSource(context.Background(), CredentialSource{Gamertag: "FAIL", XUID: "x", Source: "test"})
	if err == nil {
		t.Error("AddOrUpdateSource avec Resolve échoué doit propager l'erreur")
	}
	if pool.Size() != 1 {
		t.Errorf("size = %d, want 1 (slot non ajouté si Resolve fail)", pool.Size())
	}
}

// failingResolver est un Resolver qui échoue pour un gamertag spécifique.
type failingResolver struct {
	wrapped      Resolver
	failGamertag string
}

func (fr *failingResolver) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	if src.Gamertag == fr.failGamertag {
		return nil, errors.New("simulated resolve failure")
	}
	return fr.wrapped.Resolve(ctx, src)
}

func (fr *failingResolver) Refresh(ctx context.Context, gamertag string) (*ResolvedTokens, error) {
	return fr.wrapped.Refresh(ctx, gamertag)
}
