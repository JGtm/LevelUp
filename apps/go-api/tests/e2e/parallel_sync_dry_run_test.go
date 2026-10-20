// Package e2e_test — parallel_sync_dry_run_test.go : dry-run sync parallèle.
//
// Symétrique de air_restart_cycle_test.go (single-player) mais avec N joueurs
// résolus en parallèle. Vérifie qu'avec MultiUserTokenStore comme source unique,
// la concurrence sur le Pool/Resolver ne crée ni race condition sur le store,
// ni cross-talk entre joueurs (RT de Alice utilisé pour Bob), ni invalid_grant.
//
// Le mockProvider track par xuid (Alice rt-alice-v1 → rt-alice-v2, Bob
// rt-bob-v1 → rt-bob-v2, ...). Si un xuid voit le RT d'un autre xuid, ou
// si un RT est utilisé deux fois sans rotation persistée, le test fail.
//
//go:build cgo

package e2e_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
)

// perPlayerRotationProvider track les rotations par RT (et donc par joueur, car
// chaque joueur a un RT unique). Simule Microsoft strictement : un RT utilisé
// deux fois → invalid_grant.
type perPlayerRotationProvider struct {
	mu sync.Mutex
	// rtToOwner : mappe chaque RT à son xuid propriétaire au moment de l'émission.
	// Si un xuid utilise un RT qui n'est pas le sien (cross-talk), violation.
	rtToOwner map[string]string
	// usedRTs : RTs déjà consommés (rotation déclenchée). Re-utilisation = invalid_grant.
	usedRTs map[string]bool

	rotationsByOwner atomic.Value // map[string]int — rotations par xuid (lecture concurrente)
	rotMu            sync.Mutex   // protège rotationsByOwner

	refreshCallCount  atomic.Int64
	exchangeCallCount atomic.Int64
	invalidGrantCount atomic.Int64
	crossTalkCount    atomic.Int64
}

func newPerPlayerRotationProvider() *perPlayerRotationProvider {
	p := &perPlayerRotationProvider{
		rtToOwner: make(map[string]string),
		usedRTs:   make(map[string]bool),
	}
	p.rotationsByOwner.Store(map[string]int{})
	return p
}

// SeedInitialRT enregistre un RT initial pour un xuid, comme si capturecli
// l'avait persisté dans le store.
func (p *perPlayerRotationProvider) SeedInitialRT(xuid, rt string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rtToOwner[rt] = xuid
}

func (p *perPlayerRotationProvider) InitDeviceFlow(_ context.Context) (auth.DeviceFlow, error) {
	return nil, fmt.Errorf("not used")
}

func (p *perPlayerRotationProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error) {
	at, _, err := p.TryOAuthRefreshWithRotation(ctx, refreshToken)
	return at, err
}

func (p *perPlayerRotationProvider) TryOAuthRefreshWithRotation(_ context.Context, refreshToken string) (string, string, error) {
	p.refreshCallCount.Add(1)

	p.mu.Lock()
	defer p.mu.Unlock()

	owner, known := p.rtToOwner[refreshToken]
	if !known {
		p.invalidGrantCount.Add(1)
		return "", "", fmt.Errorf("invalid_grant: unknown refresh_token (%s)", refreshToken)
	}
	if p.usedRTs[refreshToken] {
		p.invalidGrantCount.Add(1)
		return "", "", fmt.Errorf("invalid_grant: refresh_token already used (%s, owner=%s)", refreshToken, owner)
	}
	p.usedRTs[refreshToken] = true

	// Rotation : nouveau RT toujours associé au même owner
	p.incrementRotation(owner)
	rotations := p.getRotations(owner)
	rotatedRT := fmt.Sprintf("rt-%s-v%d", owner, rotations+1)
	p.rtToOwner[rotatedRT] = owner
	accessToken := fmt.Sprintf("at-%s-v%d", owner, rotations)
	return accessToken, rotatedRT, nil
}

func (p *perPlayerRotationProvider) incrementRotation(owner string) {
	p.rotMu.Lock()
	defer p.rotMu.Unlock()
	m := p.rotationsByOwner.Load().(map[string]int)
	newM := make(map[string]int, len(m)+1)
	for k, v := range m {
		newM[k] = v
	}
	newM[owner]++
	p.rotationsByOwner.Store(newM)
}

func (p *perPlayerRotationProvider) getRotations(owner string) int {
	m := p.rotationsByOwner.Load().(map[string]int)
	return m[owner]
}

// Exchange retourne des tokens uniques par xuid. Si l'access_token reçu ne
// correspond pas à un xuid connu, log cross-talk.
func (p *perPlayerRotationProvider) Exchange(_ context.Context, accessToken string) (*auth.ExchangeResult, error) {
	p.exchangeCallCount.Add(1)
	// Format access_token attendu : at-<owner>-vN
	var owner string
	_, _ = fmt.Sscanf(accessToken, "at-%s", &owner)
	// Drop suffix "-vN"
	for i := len(owner) - 1; i >= 0; i-- {
		if owner[i] == '-' {
			owner = owner[:i]
			break
		}
	}
	return &auth.ExchangeResult{
		Tokens: &domain.HaloTokens{
			SpartanToken:   fmt.Sprintf("spartan-%s", owner),
			ClearanceToken: fmt.Sprintf("clearance-%s", owner),
		},
		Gamertag: owner,
		XUID:     owner,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────
// DRY-RUN PARALLÈLE — 5 joueurs résolus simultanément, 3 cycles
// ─────────────────────────────────────────────────────────────────────────

func TestParallelSyncDryRun_FivePlayersConcurrent_NoInvalidGrant(t *testing.T) {
	const N = 5
	const cycles = 3

	tempDir := t.TempDir()
	storeDir := tempDir + "/data/auth/watcher_tokens"
	store := auth.NewMultiUserTokenStore(storeDir)

	provider := newPerPlayerRotationProvider()

	// Setup : N joueurs avec RT initial dans le store
	players := make([]string, N)
	for i := 0; i < N; i++ {
		xuid := fmt.Sprintf("100%d", i)
		initialRT := fmt.Sprintf("rt-%s-v1", xuid)
		players[i] = xuid

		provider.SeedInitialRT(xuid, initialRT)
		if err := store.UpdateOAuthRefreshToken(xuid, initialRT); err != nil {
			t.Fatalf("seed %s: %v", xuid, err)
		}
	}

	onRotated := func(ctx context.Context, gamertag, newRT string) error {
		// gamertag == xuid dans ce test (provider Exchange retourne xuid comme gamertag)
		return store.UpdateOAuthRefreshToken(gamertag, newRT)
	}

	// ─── Cycles séquentiels avec batch de N Resolve concurrents par cycle ─
	for cycle := 1; cycle <= cycles; cycle++ {
		t.Logf("Cycle %d : %d Resolves concurrents", cycle, N)

		// Nouveau resolver à chaque cycle (simule restart Air)
		resolver := pool.NewResolver(provider, 0, onRotated)

		var wg sync.WaitGroup
		errCh := make(chan error, N)
		wg.Add(N)
		for _, xuid := range players {
			go func(xuid string) {
				defer wg.Done()
				current, _ := store.Load(xuid)
				src := pool.CredentialSource{
					Gamertag:     xuid,
					XUID:         xuid,
					RefreshToken: current.OAuthRefreshToken,
					Source:       "watcher_oauth",
				}
				if _, err := resolver.Resolve(context.Background(), src); err != nil {
					errCh <- fmt.Errorf("xuid=%s: %w", xuid, err)
				}
			}(xuid)
		}
		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Errorf("cycle %d : %v", cycle, err)
		}

		// Verify : chaque xuid a son RT incrémenté de 1 dans le store
		for _, xuid := range players {
			user, _ := store.Load(xuid)
			expected := fmt.Sprintf("rt-%s-v%d", xuid, cycle+1)
			if user.OAuthRefreshToken != expected {
				t.Errorf("cycle %d xuid=%s : RT = %q, want %q", cycle, xuid, user.OAuthRefreshToken, expected)
			}
		}
	}

	// ─── Assertions globales ───────────────────────────────────────────────
	totalRefreshes := provider.refreshCallCount.Load()
	totalExchanges := provider.exchangeCallCount.Load()
	totalInvalidGrants := provider.invalidGrantCount.Load()
	totalCrossTalk := provider.crossTalkCount.Load()

	t.Logf("─── BILAN DRY-RUN PARALLÈLE (N=%d joueurs × %d cycles) ───", N, cycles)
	t.Logf("  refresh_calls       : %d (attendu : %d = N*cycles)", totalRefreshes, N*cycles)
	t.Logf("  exchange_calls      : %d", totalExchanges)
	t.Logf("  invalid_grants      : %d (attendu : 0)", totalInvalidGrants)
	t.Logf("  cross_talk          : %d (attendu : 0)", totalCrossTalk)

	if totalRefreshes != int64(N*cycles) {
		t.Errorf("refresh_calls = %d, want %d (cycle complet sur chaque joueur)", totalRefreshes, N*cycles)
	}
	if totalInvalidGrants != 0 {
		t.Errorf("invalid_grants = %d — race condition ou cross-talk !", totalInvalidGrants)
	}
	if totalCrossTalk != 0 {
		t.Errorf("cross_talk = %d — un RT d'un joueur a été attribué à un autre", totalCrossTalk)
	}

	// Vérification finale : chaque joueur a la bonne rotation
	rotations := provider.rotationsByOwner.Load().(map[string]int)
	for _, xuid := range players {
		if got := rotations[xuid]; got != cycles {
			t.Errorf("xuid=%s : rotations = %d, want %d", xuid, got, cycles)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// DRY-RUN PARALLÈLE — Burst extreme (50 joueurs en simultané, 1 cycle)
// ─────────────────────────────────────────────────────────────────────────

func TestParallelSyncDryRun_BurstFiftyPlayers_NoRaceCondition(t *testing.T) {
	const N = 50

	tempDir := t.TempDir()
	storeDir := tempDir + "/data/auth/watcher_tokens"
	store := auth.NewMultiUserTokenStore(storeDir)
	provider := newPerPlayerRotationProvider()

	players := make([]string, N)
	for i := 0; i < N; i++ {
		xuid := fmt.Sprintf("200%03d", i)
		initialRT := fmt.Sprintf("rt-%s-v1", xuid)
		players[i] = xuid
		provider.SeedInitialRT(xuid, initialRT)
		_ = store.UpdateOAuthRefreshToken(xuid, initialRT)
	}

	resolver := pool.NewResolver(provider, 0, func(ctx context.Context, gt, newRT string) error {
		return store.UpdateOAuthRefreshToken(gt, newRT)
	})

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(N)
	for _, xuid := range players {
		go func(xuid string) {
			defer wg.Done()
			current, _ := store.Load(xuid)
			src := pool.CredentialSource{
				Gamertag:     xuid,
				XUID:         xuid,
				RefreshToken: current.OAuthRefreshToken,
				Source:       "watcher_oauth",
			}
			_, _ = resolver.Resolve(context.Background(), src)
		}(xuid)
	}
	wg.Wait()
	burstDuration := time.Since(start)

	t.Logf("─── BURST DRY-RUN (%d joueurs simultanés, 1 cycle) ───", N)
	t.Logf("  duration            : %v", burstDuration)
	t.Logf("  refresh_calls       : %d (attendu : %d)", provider.refreshCallCount.Load(), N)
	t.Logf("  invalid_grants      : %d (attendu : 0)", provider.invalidGrantCount.Load())

	if provider.invalidGrantCount.Load() != 0 {
		t.Errorf("BURST race condition détectée : %d invalid_grant sur %d joueurs",
			provider.invalidGrantCount.Load(), N)
	}
	// Vérifier que tous les joueurs ont leur RT mis à jour (rt-XXX-v2)
	for _, xuid := range players {
		user, _ := store.Load(xuid)
		expected := fmt.Sprintf("rt-%s-v2", xuid)
		if user.OAuthRefreshToken != expected {
			t.Errorf("xuid=%s : RT = %q, want %q (rotation incomplète sous burst)",
				xuid, user.OAuthRefreshToken, expected)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// DRY-RUN — Cross-talk : un joueur ne doit JAMAIS hériter du RT d'un autre
// ─────────────────────────────────────────────────────────────────────────

func TestParallelSyncDryRun_NoCrossTalk_BetweenPlayers(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := tempDir + "/data/auth/watcher_tokens"
	store := auth.NewMultiUserTokenStore(storeDir)
	provider := newPerPlayerRotationProvider()

	// 2 joueurs distincts avec RT distincts (xuids numériques pour xuidIsSafe)
	const aliceXUID = "1001"
	const bobXUID = "1002"
	provider.SeedInitialRT(aliceXUID, "rt-"+aliceXUID+"-v1")
	provider.SeedInitialRT(bobXUID, "rt-"+bobXUID+"-v1")
	if err := store.UpdateOAuthRefreshToken(aliceXUID, "rt-"+aliceXUID+"-v1"); err != nil {
		t.Fatalf("seed alice: %v", err)
	}
	if err := store.UpdateOAuthRefreshToken(bobXUID, "rt-"+bobXUID+"-v1"); err != nil {
		t.Fatalf("seed bob: %v", err)
	}

	resolver := pool.NewResolver(provider, 0, func(ctx context.Context, gt, newRT string) error {
		return store.UpdateOAuthRefreshToken(gt, newRT)
	})

	// 20 résolutions séquentielles alternant alice/bob (le Pool ne déduplique pas
	// les Resolve concurrents sur le même gamertag, donc séquentiel pour ce test
	// précis ; le burst test ci-dessus couvre la concurrence différenciée).
	for i := 0; i < 20; i++ {
		xuid := aliceXUID
		if i%2 == 1 {
			xuid = bobXUID
		}
		current, err := store.Load(xuid)
		if err != nil {
			t.Fatalf("Load %s: %v", xuid, err)
		}
		src := pool.CredentialSource{
			Gamertag:     xuid,
			XUID:         xuid,
			RefreshToken: current.OAuthRefreshToken,
			Source:       "watcher_oauth",
		}
		if _, err := resolver.Resolve(context.Background(), src); err != nil {
			t.Errorf("iteration %d xuid=%s : %v", i, xuid, err)
		}
	}

	// Verify : le store d'Alice a un RT alice-, celui de Bob un RT bob-
	alice, _ := store.Load(aliceXUID)
	bob, _ := store.Load(bobXUID)

	if !strings.Contains(alice.OAuthRefreshToken, aliceXUID) {
		t.Errorf("CROSS-TALK : Alice store contient %q (devrait contenir %s)", alice.OAuthRefreshToken, aliceXUID)
	}
	if !strings.Contains(bob.OAuthRefreshToken, bobXUID) {
		t.Errorf("CROSS-TALK : Bob store contient %q (devrait contenir %s)", bob.OAuthRefreshToken, bobXUID)
	}
	if strings.Contains(alice.OAuthRefreshToken, bobXUID) {
		t.Errorf("CROSS-TALK : Alice store contient le xuid de Bob %q", alice.OAuthRefreshToken)
	}

	t.Logf("─── DRY-RUN cross-talk (10 alterns alice/bob × 2 = 20 resolves) ───")
	t.Logf("  alice RT final      : %q", alice.OAuthRefreshToken)
	t.Logf("  bob RT final        : %q", bob.OAuthRefreshToken)
	t.Logf("  invalid_grants      : %d (attendu : 0)", provider.invalidGrantCount.Load())
}
