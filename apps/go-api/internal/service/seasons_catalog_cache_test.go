// seasons_catalog_cache_test.go — cache du catalogue des saisons et mémoire de
// l'échec du fetch live (plan perf 2026-09-23, lot L5b, D5b.1).
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// testClock est une horloge pilotée : le catalogue lit c.now().
type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)}
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// seasonsCtxWithTokens : contexte porteur de jetons Halo, comme une requête
// authentifiée — seul ce cas peut faire un appel réseau, donc ouvrir l'attente.
func seasonsCtxWithTokens() context.Context {
	return ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "spartan", ClearanceToken: "clearance"}, "1234")
}

// switchableSeasonProvider échoue tant que failing est vrai, réussit ensuite.
// Compteur atomique et latence optionnelle pour les tests de concurrence.
type switchableSeasonProvider struct {
	failing atomic.Bool
	calls   atomic.Int32
	delay   time.Duration
	result  []domain.SeasonCalendar
}

func (p *switchableSeasonProvider) FetchSeasonCalendar(_ context.Context, _ string) ([]domain.SeasonCalendar, []byte, error) {
	p.calls.Add(1)
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
	if p.failing.Load() {
		return nil, nil, errors.New("HTTP 403")
	}
	return p.result, nil, nil
}

// syncMetadataRepo : base de saisons en mémoire, sûre en concurrence. Un
// UpsertSeason la peuple, ListSeasons compte ses lectures.
type syncMetadataRepo struct {
	fakeMetadataRepo
	mu      sync.Mutex
	rows    []domain.SeasonCalendar
	lists   int
	listErr error
}

func (r *syncMetadataRepo) ListSeasons(_ context.Context, _ string) ([]domain.SeasonCalendar, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lists++
	if r.listErr != nil {
		return nil, r.listErr
	}
	return append([]domain.SeasonCalendar(nil), r.rows...), nil
}

func (r *syncMetadataRepo) UpsertSeason(_ context.Context, s domain.SeasonCalendar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, s)
	return nil
}

func (r *syncMetadataRepo) listCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lists
}

func TestSeasonsCatalog_CachedUntilTTL(t *testing.T) {
	end := time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)
	repo := &syncMetadataRepo{rows: []domain.SeasonCalendar{
		dbSeason("season1", "Heroes of Reach", time.Date(2021, 12, 8, 0, 0, 0, 0, time.UTC), &end),
	}}
	clock := newTestClock()
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, nil, nil)
	cat.now = clock.now

	first := cat.Load(context.Background(), "halo_infinite")
	clock.advance(seasonsCatalogTTL - time.Minute)
	second := cat.Load(context.Background(), "halo_infinite")
	if got := repo.listCount(); got != 1 {
		t.Fatalf("lectures de la base avant le TTL = %d, want 1 (le 2e Load doit venir du cache)", got)
	}
	if len(first) != len(second) || len(second) != 2 {
		t.Fatalf("catalogues = %d puis %d entrées, want 2 et 2", len(first), len(second))
	}
	clock.advance(2 * time.Minute) // TTL dépassé
	cat.Load(context.Background(), "halo_infinite")
	if got := repo.listCount(); got != 2 {
		t.Errorf("lectures de la base après le TTL = %d, want 2", got)
	}
}

// TestSeasonsCatalog_FetchFailureMemorized_ThenSuccess : un échec du fetch live
// (requête authentifiée) n'est plus retenté pendant l'attente — aucun appel
// réseau, aucune relecture de base —, puis la requête qui suit l'attente retente
// et réussit : le catalogue porte alors la saison fetchée.
func TestSeasonsCatalog_FetchFailureMemorized_ThenSuccess(t *testing.T) {
	repo := &syncMetadataRepo{}
	provider := &switchableSeasonProvider{
		result: []domain.SeasonCalendar{dbSeason("season14", "Skyfall", time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), nil)},
	}
	provider.failing.Store(true)
	clock := newTestClock()
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, provider, nil)
	cat.now = clock.now
	ctx := seasonsCtxWithTokens()

	if got := cat.Load(ctx, "halo_infinite"); len(got) != 2 {
		t.Fatalf("échec live : %d entrées, want 2 (repli TOML)", len(got))
	}
	provider.failing.Store(false) // Waypoint répondrait désormais : l'attente doit primer
	clock.advance(29 * time.Minute)
	if got := cat.Load(ctx, "halo_infinite"); len(got) != 2 {
		t.Fatalf("pendant l'attente : %d entrées, want 2 (repli caché)", len(got))
	}
	if got := provider.calls.Load(); got != 1 {
		t.Fatalf("appels live pendant l'attente = %d, want 1 (échec mémorisé)", got)
	}
	if got := repo.listCount(); got != 1 {
		t.Fatalf("lectures de base pendant l'attente = %d, want 1", got)
	}

	clock.advance(2 * time.Minute) // attente de 30 min écoulée
	got := cat.Load(ctx, "halo_infinite")
	if provider.calls.Load() != 2 {
		t.Fatalf("appels live après l'attente = %d, want 2", provider.calls.Load())
	}
	if len(got) != 3 {
		t.Fatalf("après la réussite : %d entrées, want 3 (TOML + season14)", len(got))
	}
	clock.advance(time.Minute)
	cat.Load(ctx, "halo_infinite")
	if provider.calls.Load() != 2 {
		t.Errorf("appels live après une réussite = %d, want 2 (catalogue caché)", provider.calls.Load())
	}
}

// TestSeasonsCatalog_FetchFailureWithoutTokens_NotMemorized : sans jeton, le
// provider refuse sans appel réseau ; l'échec n'ouvre pas d'attente (une requête
// authentifiée suivante doit pouvoir réussir) et le repli n'est pas caché.
func TestSeasonsCatalog_FetchFailureWithoutTokens_NotMemorized(t *testing.T) {
	repo := &syncMetadataRepo{}
	provider := &switchableSeasonProvider{
		result: []domain.SeasonCalendar{dbSeason("season14", "Skyfall", time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), nil)},
	}
	provider.failing.Store(true)
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, provider, nil)
	cat.now = newTestClock().now

	cat.Load(context.Background(), "halo_infinite")
	provider.failing.Store(false)
	got := cat.Load(seasonsCtxWithTokens(), "halo_infinite")
	if provider.calls.Load() != 2 {
		t.Fatalf("appels live = %d, want 2 (l'échec sans jeton ne bloque rien)", provider.calls.Load())
	}
	if len(got) != 3 {
		t.Errorf("requête authentifiée suivante : %d entrées, want 3", len(got))
	}
}

func TestSeasonsCatalog_ListSeasonsError_NotCached(t *testing.T) {
	repo := &syncMetadataRepo{listErr: errors.New("database is locked")}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, nil, nil)
	cat.now = newTestClock().now

	cat.Load(context.Background(), "halo_infinite")
	cat.Load(context.Background(), "halo_infinite")
	if got := repo.listCount(); got != 2 {
		t.Errorf("lectures de base = %d, want 2 (un échec de lecture n'est pas caché)", got)
	}
}

// TestSeasonsCatalog_ReturnsCopies : muter le résultat (Extra, End, Label) d'un
// Load — qu'il vienne d'une résolution ou du cache — ne touche ni le cache ni les
// appels suivants.
func TestSeasonsCatalog_ReturnsCopies(t *testing.T) {
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), nil, nil, nil)
	mutate := func(entries []SeasonCatalogEntry) {
		entries[0].Extra["csr_season_id"] = "muté"
		*entries[0].End = time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)
		entries[0].Label = "muté"
	}
	check := func(step string, entries []SeasonCatalogEntry) {
		t.Helper()
		if entries[0].Extra["csr_season_id"] != "CsrSeason1" {
			t.Errorf("%s : Extra partagé : %q", step, entries[0].Extra["csr_season_id"])
		}
		if entries[0].End.Year() != 2022 {
			t.Errorf("%s : End partagé : %v", step, entries[0].End)
		}
		if entries[0].Label != "Heroes of Reach" {
			t.Errorf("%s : Label partagé : %q", step, entries[0].Label)
		}
	}

	resolved := cat.Load(context.Background(), "halo_infinite") // résolution
	mutate(resolved)
	hit := cat.Load(context.Background(), "halo_infinite") // cache
	check("après mutation du résultat résolu", hit)
	mutate(hit)
	check("après mutation d'un résultat caché", cat.Load(context.Background(), "halo_infinite"))
}

// TestSeasonsCatalog_ConcurrentMisses_OneResolution : des Load concurrents sur un
// cache froid ne déclenchent qu'une lecture de base et qu'un fetch live.
func TestSeasonsCatalog_ConcurrentMisses_OneResolution(t *testing.T) {
	repo := &syncMetadataRepo{}
	provider := &switchableSeasonProvider{delay: 50 * time.Millisecond}
	provider.failing.Store(true)
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, provider, nil)
	ctx := seasonsCtxWithTokens()

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if got := cat.Load(ctx, "halo_infinite"); len(got) != 2 {
				t.Errorf("Load concurrent : %d entrées, want 2", len(got))
			}
		}()
	}
	wg.Wait()
	if got := provider.calls.Load(); got != 1 {
		t.Errorf("fetchs live = %d, want 1", got)
	}
	if got := repo.listCount(); got != 1 {
		t.Errorf("lectures de base = %d, want 1", got)
	}
}

// TestSeasonsCatalog_TimingMarkers : le chargement pose un marqueur miss puis hit
// (sections de durée nulle : la durée reste portée par la section appelante).
func TestSeasonsCatalog_TimingMarkers(t *testing.T) {
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), nil, nil, nil)
	ctx, tm := timing.WithTimings(context.Background())
	cat.Load(ctx, "halo_infinite")
	cat.Load(ctx, "halo_infinite")
	cat.Load(ctx, "halo_infinite")

	calls := map[string]int{}
	for _, s := range tm.Snapshot() {
		calls[s.Name] = s.Calls
	}
	if calls["seasons_catalog_miss"] != 1 || calls["seasons_catalog_hit"] != 2 {
		t.Errorf("marqueurs = %v, want miss=1 hit=2", calls)
	}
}

// ctxAwareSeasonProvider : comme un vrai client HTTP, rend l'erreur du contexte quand la
// requête appelante a pris fin pendant l'appel ; sinon errOverride s'il est posé.
type ctxAwareSeasonProvider struct {
	calls       atomic.Int32
	errOverride error
	result      []domain.SeasonCalendar
}

func (p *ctxAwareSeasonProvider) FetchSeasonCalendar(ctx context.Context, _ string) ([]domain.SeasonCalendar, []byte, error) {
	p.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if p.errOverride != nil {
		return nil, nil, p.errOverride
	}
	return p.result, nil, nil
}

// TestSeasonsCatalog_ContextEndNotMemorized (lot perf L9-go, revue adversariale B) : une
// requête authentifiée annulée pendant le fetch live, ou un fetch tombé sur une échéance,
// n'est pas un verdict de Waypoint — l'attente de 30 min ne s'ouvre pas, la requête vivante
// suivante retente et sert la saison fetchée (avant : repli TOML servi 30 min à tout le titre).
func TestSeasonsCatalog_ContextEndNotMemorized(t *testing.T) {
	end := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	for nom, prepare := range map[string]func(*ctxAwareSeasonProvider) context.Context{
		"requête annulée": func(*ctxAwareSeasonProvider) context.Context {
			ctx, cancel := context.WithCancel(seasonsCtxWithTokens())
			cancel() // le client est parti pendant l'appel
			return ctx
		},
		"échéance de l'appel": func(p *ctxAwareSeasonProvider) context.Context {
			p.errOverride = fmt.Errorf("GET season calendar: %w", context.DeadlineExceeded)
			return seasonsCtxWithTokens()
		},
	} {
		t.Run(nom, func(t *testing.T) {
			repo := &syncMetadataRepo{}
			prov := &ctxAwareSeasonProvider{result: []domain.SeasonCalendar{
				dbSeason("season-new", "Operation Nouvelle", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), &end),
			}}
			clock := newTestClock()
			cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, prov, nil)
			cat.now = clock.now

			_ = cat.Load(prepare(prov), "halo_infinite")
			prov.errOverride = nil
			clock.advance(time.Minute)
			found := false
			for _, e := range cat.Load(seasonsCtxWithTokens(), "halo_infinite") {
				found = found || e.ID == "season-new"
			}
			if !found || prov.calls.Load() != 2 {
				t.Errorf("requête vivante 1 min après : saison Waypoint servie=%v, %d appel(s) live — want servie, 2 appels (fin de contexte non mémorisée)",
					found, prov.calls.Load())
			}
		})
	}
}
