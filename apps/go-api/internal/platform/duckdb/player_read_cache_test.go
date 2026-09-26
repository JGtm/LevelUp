// player_read_cache_test.go — cache process-wide des lectures joueur (plan perf
// 2026-09-23, lot L5b, D5b.3) : coalescence, copie à la lecture, TTL, invalidation
// par (xuid, titre) et course invalidation / chargement. Aucune base : chargeurs
// factices.
package duckdb

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// countingFilterRows est un chargeur de lignes de filtres qui compte ses appels.
type countingFilterRows struct {
	calls atomic.Int32
	delay time.Duration
	err   error
	rows  []domain.FilterMatchRow
}

func (f *countingFilterRows) LoadMatchesForFilters(_ context.Context) ([]domain.FilterMatchRow, error) {
	f.calls.Add(1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.FilterMatchRow(nil), f.rows...), nil
}

func filterRowsFixture() []domain.FilterMatchRow {
	return []domain.FilterMatchRow{
		{MatchID: "m1", IsWithFriends: false},
		{MatchID: "m2", IsWithFriends: true},
		{MatchID: "m3", IsWithFriends: false},
	}
}

func testScope(xuid, title, path string) playerCacheScope {
	return playerCacheScope{playerIdentity: playerIdentity{xuid: xuid, titleSlug: title}, dbPath: path}
}

// newTestFiltersRepo : CachedFiltersRepo sur un chargeur factice et un cache isolé.
func newTestFiltersRepo(loader *countingFilterRows, scope playerCacheScope) (*CachedFiltersRepo, *playerReadCache[[]domain.FilterMatchRow]) {
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	return &CachedFiltersRepo{rows: loader, cache: cache, scope: scope}, cache
}

// TestCachedFiltersRepo_ThreeResolvesOneLoad : les trois /filters/resolve d'une page
// (solo, escouade, aperçu) ne lisent la base qu'une fois — FiltersService.Resolve
// appelle LoadMatchesForFilters sans paramètre, le contexte est appliqué en Go.
func TestCachedFiltersRepo_ThreeResolvesOneLoad(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture()}
	repo, _ := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	for i := 0; i < 3; i++ {
		rows, err := repo.LoadMatchesForFilters(context.Background())
		if err != nil || len(rows) != 3 || rows[1].MatchID != "m2" {
			t.Fatalf("appel %d : %d lignes, err=%v", i, len(rows), err)
		}
	}
	if got := loader.calls.Load(); got != 1 {
		t.Errorf("lectures = %d, want 1", got)
	}
}

// TestCachedFiltersRepo_ReturnsCopies : trier ou modifier les lignes rendues (comme
// FilteredMatchIDs, qui trie ses lignes filtrées) ne touche pas au cache.
func TestCachedFiltersRepo_ReturnsCopies(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture()}
	repo, _ := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	for i := 0; i < 2; i++ { // 1re lecture = chargement, 2e = cache
		rows, _ := repo.LoadMatchesForFilters(context.Background())
		rows[0], rows[2] = rows[2], rows[0]
		rows[1].MatchID = "muté"
	}
	again, _ := repo.LoadMatchesForFilters(context.Background())
	if again[0].MatchID != "m1" || again[1].MatchID != "m2" || again[2].MatchID != "m3" {
		t.Errorf("cache modifié par un appelant : %v %v %v", again[0].MatchID, again[1].MatchID, again[2].MatchID)
	}
}

func TestPlayerReadCache_TTL(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture()}
	repo, cache := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	_, _ = repo.LoadMatchesForFilters(context.Background())
	now = now.Add(playerReadCacheTTL - time.Second)
	_, _ = repo.LoadMatchesForFilters(context.Background())
	if got := loader.calls.Load(); got != 1 {
		t.Fatalf("avant le TTL : %d lectures, want 1", got)
	}
	now = now.Add(2 * time.Second)
	_, _ = repo.LoadMatchesForFilters(context.Background())
	if got := loader.calls.Load(); got != 2 {
		t.Errorf("après le TTL : %d lectures, want 2", got)
	}
}

// TestPlayerReadCache_InvalidateScopes : l'invalidation retire (xuid, titre) — toutes
// bases confondues — et rien d'autre ; un titre vide vise tous les titres du xuid.
func TestPlayerReadCache_InvalidateScopes(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	loaders := map[string]*countingFilterRows{}
	repos := map[string]*CachedFiltersRepo{}
	for name, scope := range map[string]playerCacheScope{
		"x1/hi/a": testScope("x1", "halo_infinite", "a"),
		"x1/hi/b": testScope("x1", "halo_infinite", "b"),
		"x1/h5":   testScope("x1", "halo_5", "c"),
		"x2/hi":   testScope("x2", "halo_infinite", "d"),
	} {
		loaders[name] = &countingFilterRows{rows: filterRowsFixture()}
		repos[name] = &CachedFiltersRepo{rows: loaders[name], cache: cache, scope: scope}
		_, _ = repos[name].LoadMatchesForFilters(context.Background())
	}
	reloadAll := func() {
		for _, r := range repos {
			_, _ = r.LoadMatchesForFilters(context.Background())
		}
	}
	expect := func(step string, want map[string]int32) {
		t.Helper()
		for name, n := range want {
			if got := loaders[name].calls.Load(); got != n {
				t.Errorf("%s : %s lu %d fois, want %d", step, name, got, n)
			}
		}
	}

	if removed := cache.invalidate("x1", "halo_infinite"); removed != 2 {
		t.Errorf("entrées retirées = %d, want 2 (deux bases du même joueur/titre)", removed)
	}
	reloadAll()
	expect("après invalidate(x1, halo_infinite)", map[string]int32{"x1/hi/a": 2, "x1/hi/b": 2, "x1/h5": 1, "x2/hi": 1})

	cache.invalidate("x1", "")
	reloadAll()
	expect("après invalidate(x1, tous titres)", map[string]int32{"x1/hi/a": 3, "x1/hi/b": 3, "x1/h5": 2, "x2/hi": 1})
}

// TestPlayerReadCache_InvalidationDuringLoad : un chargement commencé avant une
// invalidation rend son résultat mais ne le met pas en cache, et une requête
// arrivée après l'invalidation ne se greffe pas sur lui.
func TestPlayerReadCache_InvalidationDuringLoad(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	scope := testScope("x1", "halo_infinite", "p")
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	// Chaque chargement étiquette ses lignes : le 1er (commencé avant l'écriture) rend
	// « périmé », les suivants « frais ».
	load := func(context.Context) ([]domain.FilterMatchRow, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release // premier chargement : bloqué jusqu'après l'invalidation
			return []domain.FilterMatchRow{{MatchID: "périmé"}}, nil
		}
		return []domain.FilterMatchRow{{MatchID: "frais"}}, nil
	}

	staleDone := make(chan error, 1)
	go func() {
		_, err := cache.load(context.Background(), scope, "", load)
		staleDone <- err
	}()
	<-started
	cache.invalidate("x1", "halo_infinite")

	freshDone := make(chan []domain.FilterMatchRow, 1)
	go func() {
		rows, _ := cache.load(context.Background(), scope, "", load)
		freshDone <- rows
	}()
	select {
	case rows := <-freshDone:
		if len(rows) != 1 || rows[0].MatchID != "frais" {
			t.Errorf("requête post-invalidation : %v, want [frais]", rows)
		}
	case <-time.After(2 * time.Second):
		t.Error("la requête post-invalidation attend le chargement périmé (vol partagé)")
	}
	close(release)
	if err := <-staleDone; err != nil {
		t.Fatalf("chargement pré-invalidation : %v", err)
	}
	rows, err := cache.load(context.Background(), scope, "", load)
	if err != nil || len(rows) != 1 || rows[0].MatchID != "frais" {
		t.Errorf("après le retour du chargement périmé : %v (err=%v), want [frais] depuis le cache", rows, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("chargements = %d, want 2", got)
	}
}

func TestPlayerReadCache_ConcurrentMissesCoalesce(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture(), delay: 50 * time.Millisecond}
	repo, _ := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	const n = 25
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if rows, err := repo.LoadMatchesForFilters(context.Background()); err != nil || len(rows) != 3 {
				t.Errorf("appel concurrent : %d lignes, err=%v", len(rows), err)
			}
		}()
	}
	wg.Wait()
	if got := loader.calls.Load(); got != 1 {
		t.Errorf("lectures = %d, want 1 (coalescence)", got)
	}
}

func TestPlayerReadCache_ErrorNotCached(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{err: errors.New("shared reader: timeout")}
	repo, _ := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	for i := 0; i < 2; i++ {
		if _, err := repo.LoadMatchesForFilters(context.Background()); err == nil {
			t.Fatalf("appel %d : erreur attendue", i)
		}
	}
	if got := loader.calls.Load(); got != 2 {
		t.Errorf("lectures = %d, want 2 (une erreur n'est pas cachée)", got)
	}
}

// TestPlayerReadCache_CanceledLeaderDoesNotFailWaiter : la requête qui porte le
// chargement partagé est annulée ; celle qui l'attendait, toujours vivante,
// recharge pour son compte au lieu d'hériter de l'annulation.
func TestPlayerReadCache_CanceledLeaderDoesNotFailWaiter(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	scope := testScope("x1", "halo_infinite", "p")
	var calls atomic.Int32
	started := make(chan struct{})
	load := func(ctx context.Context) ([]domain.FilterMatchRow, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return filterRowsFixture(), nil
	}

	leaderCtx, cancel := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := cache.load(leaderCtx, scope, "", load)
		leaderErr <- err
	}()
	<-started
	waiterDone := make(chan error, 1)
	var waiterRows []domain.FilterMatchRow
	go func() {
		rows, err := cache.load(context.Background(), scope, "", load)
		waiterRows = rows
		waiterDone <- err
	}()
	time.Sleep(50 * time.Millisecond) // l'attente se greffe sur le vol du meneur
	cancel()

	if err := <-leaderErr; !errors.Is(err, context.Canceled) {
		t.Errorf("meneur : err=%v, want context.Canceled", err)
	}
	if err := <-waiterDone; err != nil || len(waiterRows) != 3 {
		t.Errorf("requête en attente : err=%v, %d lignes — elle ne doit pas hériter de l'annulation", err, len(waiterRows))
	}
}

// TestInvalidatePlayerReadCaches_Global : la fonction publique (appelée par le
// post-sync) vide le cache process-wide des lignes de filtres du joueur.
func TestInvalidatePlayerReadCaches_Global(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture()}
	repo := &CachedFiltersRepo{rows: loader, cache: filterRowsReadCache,
		scope: testScope("l5b-test-global-invalidate", "halo_infinite", "p")}
	_, _ = repo.LoadMatchesForFilters(context.Background())
	_, _ = repo.LoadMatchesForFilters(context.Background())
	InvalidatePlayerReadCaches(context.Background(), "l5b-test-global-invalidate", "halo_infinite")
	_, _ = repo.LoadMatchesForFilters(context.Background())
	if got := loader.calls.Load(); got != 2 {
		t.Errorf("lectures = %d, want 2 (1 chargement, 1 hit, 1 rechargement après invalidation)", got)
	}
}

func TestPlayerReadCache_TimingMarkers(t *testing.T) {
	t.Parallel()
	loader := &countingFilterRows{rows: filterRowsFixture()}
	repo, _ := newTestFiltersRepo(loader, testScope("x1", "halo_infinite", "p"))
	ctx, tm := timing.WithTimings(context.Background())
	for i := 0; i < 3; i++ {
		_, _ = repo.LoadMatchesForFilters(ctx)
	}
	calls := map[string]int{}
	for _, s := range tm.Snapshot() {
		calls[s.Name] = s.Calls
	}
	if calls["filter_rows_cache_miss"] != 1 || calls["filter_rows_cache_hit"] != 2 {
		t.Errorf("marqueurs = %v, want miss=1 hit=2", calls)
	}
}

// TestPlayerReadCache_DegradedLoadNotStored (lot L9-go, revue adversariale B, P0) : une
// étape best-effort en échec (noteDegraded) rend ses lignes à la requête qui les a
// chargées, sans erreur (best-effort, comme sans cache), mais ne les met pas en cache ;
// le chargement complet suivant, lui, l'est. Marqueur timing `<cache>_degraded`.
func TestPlayerReadCache_DegradedLoadNotStored(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	scope := testScope("x1", "halo_infinite", "p")
	var calls atomic.Int32
	load := func(ctx context.Context) ([]domain.FilterMatchRow, error) {
		if calls.Add(1) == 1 {
			noteDegraded(ctx, "traductions")
			return []domain.FilterMatchRow{{MatchID: "non-traduit"}}, nil
		}
		return []domain.FilterMatchRow{{MatchID: "traduit"}}, nil
	}
	ctx, tm := timing.WithTimings(context.Background())
	for i, want := range []string{"non-traduit", "traduit", "traduit"} {
		rows, err := cache.load(ctx, scope, "", load)
		if err != nil || len(rows) != 1 || rows[0].MatchID != want {
			t.Fatalf("appel %d : %v (err=%v), want [%s]", i, rows, err, want)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("chargements = %d, want 2 (le dégradé n'est pas caché, le complet l'est)", got)
	}
	marqueurs := map[string]int{}
	for _, s := range tm.Snapshot() {
		marqueurs[s.Name] = s.Calls
	}
	if marqueurs["filter_rows_cache_degraded"] != 1 || marqueurs["filter_rows_cache_miss"] != 2 ||
		marqueurs["filter_rows_cache_hit"] != 1 {
		t.Errorf("marqueurs = %v, want degraded=1 miss=2 hit=1", marqueurs)
	}
}

// TestPlayerReadCache_RequestEndedDuringLoadNotStored : la requête qui porte le
// chargement prend fin pendant qu'il tourne, et une étape best-effort avale
// l'annulation (lignes non traduites, AUCUNE erreur, rien de consigné). Elles ne sont
// pas mises en cache : la requête vivante suivante recharge (revue B : 25 empoisonnements
// sur 400 annulations réparties avant le correctif).
func TestPlayerReadCache_RequestEndedDuringLoadNotStored(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	scope := testScope("x1", "halo_infinite", "p")
	var calls atomic.Int32
	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	load := func(context.Context) ([]domain.FilterMatchRow, error) {
		if calls.Add(1) == 1 {
			cancel() // le client part pendant la traduction, que l'étape avale
			return []domain.FilterMatchRow{{MatchID: "non-traduit"}}, nil
		}
		return []domain.FilterMatchRow{{MatchID: "traduit"}}, nil
	}
	rows, err := cache.load(reqCtx, scope, "", load)
	if err != nil || len(rows) != 1 || rows[0].MatchID != "non-traduit" {
		t.Fatalf("requête annulée : %v (err=%v), want ses lignes sans erreur", rows, err)
	}
	rows, err = cache.load(context.Background(), scope, "", load)
	if err != nil || len(rows) != 1 || rows[0].MatchID != "traduit" {
		t.Errorf("requête vivante suivante : %v (err=%v), want [traduit] (rechargé, pas servi du cache)", rows, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("chargements = %d, want 2", got)
	}
}

// TestPlayerReadCache_DegradedLoadNotSharedWithWaiters : une requête greffée sur le
// vol d'un chargement qui se révèle dégradé ne reçoit pas ses lignes : elle recharge
// pour son compte, et c'est son chargement complet qui entre en cache.
func TestPlayerReadCache_DegradedLoadNotSharedWithWaiters(t *testing.T) {
	t.Parallel()
	cache := newPlayerReadCache("filter_rows_cache", cloneFilterRows)
	scope := testScope("x1", "halo_infinite", "p")
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	load := func(ctx context.Context) ([]domain.FilterMatchRow, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
			noteDegraded(ctx, "traductions")
			return []domain.FilterMatchRow{{MatchID: "dégradé"}}, nil
		}
		return []domain.FilterMatchRow{{MatchID: "complet"}}, nil
	}
	porteur, attente := make(chan []domain.FilterMatchRow, 1), make(chan []domain.FilterMatchRow, 1)
	go func() {
		rows, _ := cache.load(context.Background(), scope, "", load)
		porteur <- rows
	}()
	<-started
	go func() {
		rows, _ := cache.load(context.Background(), scope, "", load)
		attente <- rows
	}()
	time.Sleep(50 * time.Millisecond) // la seconde requête se greffe sur le vol du porteur
	close(release)
	if rows := <-porteur; len(rows) != 1 || rows[0].MatchID != "dégradé" {
		t.Errorf("porteur : %v, want [dégradé] (best-effort : ses propres lignes)", rows)
	}
	if rows := <-attente; len(rows) != 1 || rows[0].MatchID != "complet" {
		t.Errorf("requête en attente : %v, want [complet] — le chargement dégradé ne se partage pas", rows)
	}
	rows, err := cache.load(context.Background(), scope, "", load)
	if err != nil || len(rows) != 1 || rows[0].MatchID != "complet" || calls.Load() != 2 {
		t.Errorf("après : %v (err=%v, %d chargements), want [complet] depuis le cache, 2 chargements",
			rows, err, calls.Load())
	}
}

// TestBestEffortFailed_Consignation : un échec consigne l'étape dans la sonde du
// chargement en cours ; une table absente (base non migrée, même résultat à chaque
// lecture) non ; hors chargement mis en cache, rien à consigner ni panique.
func TestBestEffortFailed_Consignation(t *testing.T) {
	t.Parallel()
	ctx, probe := withDegradationProbe(context.Background())
	bestEffortFailed(ctx, "absente", errors.New("Catalog Error: Table with name x does not exist"))
	if cause := probe.cause(ctx); cause != nil {
		t.Errorf("table absente consignée : %v", cause)
	}
	bestEffortFailed(ctx, "asset_fr_translations", errors.New("IO Error: lecture interrompue"))
	if cause := probe.cause(ctx); cause == nil || !strings.Contains(cause.Error(), "asset_fr_translations") {
		t.Errorf("échec non consigné : %v", cause)
	}
	bestEffortFailed(context.Background(), "hors_cache", errors.New("boom")) // aucune sonde : sans effet
}
