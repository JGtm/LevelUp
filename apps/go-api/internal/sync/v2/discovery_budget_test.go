// Package v2 — discovery_budget_test.go : le budget de rattrapage est celui du CYCLE, réparti entre les
// joueurs au rattrapage (budget.go), et les abandons échus sont retenus par identifiant (retry.go).
package v2

import (
	"context"
	"database/sql"
	"fmt"
	gosync "sync"
	"testing"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/knownset"
)

// budgetProvider note l'état d'historique reçu par chaque joueur et ne liste rien.
type budgetProvider struct {
	mu   gosync.Mutex
	seen map[string]HistoryState
}

func (b *budgetProvider) ListUnknownMatches(_ context.Context, p PlayerProfile, _ map[string]bool, h HistoryState) (Listing, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seen == nil {
		b.seen = map[string]HistoryState{}
	}
	b.seen[p.PlayerSlug] = h
	return Listing{}, nil
}

func slugs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("p%02d", i)
	}
	return out
}

// catchUpLoader : tous les joueurs sont au rattrapage.
func catchUpLoader(slugs []string) *mockLoader {
	hist := map[string]HistoryState{}
	for _, s := range slugs {
		hist[s] = HistoryState{Incomplete: true}
	}
	return &mockLoader{history: hist}
}

func discoverBudgets(t *testing.T, n int) (*budgetProvider, DiscoveryResult) {
	t.Helper()
	ids := slugs(n)
	prov := &budgetProvider{}
	res, err := RunDiscovery(context.Background(), mkPlayers(ids...), catchUpLoader(ids), prov)
	if err != nil {
		t.Fatal(err)
	}
	return prov, res
}

// Un seul joueur au rattrapage : le budget entier du cycle.
func TestRunDiscovery_Budget_UnJoueur(t *testing.T) {
	prov, _ := discoverBudgets(t, 1)
	if got := prov.seen["p00"].catchUpBudget(); got != historystate.PassBudget() {
		t.Errorf("budget %d, want %d", got, historystate.PassBudget())
	}
}

// Deux joueurs au rattrapage dans le même cycle : 100 chacun, 200 au total.
func TestRunDiscovery_Budget_DeuxJoueursSePartagent(t *testing.T) {
	prov, res := discoverBudgets(t, 2)
	total := 0
	for slug, h := range prov.seen {
		if !h.Incomplete || h.Deferred || h.Budget != historystate.PassBudget()/2 {
			t.Errorf("%s : %+v, want un rattrapage à %d", slug, h, historystate.PassBudget()/2)
		}
		total += h.Budget
		if res.History[slug].Budget != h.Budget {
			t.Errorf("%s : PlayerHistory.Budget %d, want %d", slug, res.History[slug].Budget, h.Budget)
		}
	}
	if total != historystate.PassBudget() {
		t.Errorf("total %d, want %d", total, historystate.PassBudget())
	}
}

// Dix joueurs : huit rattrapent (budget minimal), deux attendent en delta ordinaire et restent en attente.
func TestRunDiscovery_Budget_DixJoueurs_HuitRattrapentDeuxAttendent(t *testing.T) {
	prov, res := discoverBudgets(t, 10)
	catching, deferred, total := 0, 0, 0
	for slug, h := range prov.seen {
		switch {
		case h.Deferred:
			deferred++
			if h.Incomplete || h.Cursor != nil || !res.History[slug].Deferred {
				t.Errorf("%s en attente : %+v / %+v, want un delta ordinaire (Incomplete faux) marqué Deferred", slug, h, res.History[slug])
			}
		case h.Incomplete:
			catching++
			total += h.Budget
			if h.Budget != catchUpMinBudget {
				t.Errorf("%s : budget %d, want %d", slug, h.Budget, catchUpMinBudget)
			}
		}
	}
	if catching != 8 || deferred != 2 || total != historystate.PassBudget() {
		t.Errorf("%d rattrapent, %d attendent, %d de budget, want 8, 2 et %d", catching, deferred, total, historystate.PassBudget())
	}
	// Ordre stable : les deux derniers par identifiant attendent.
	for _, slug := range []string{"p08", "p09"} {
		if !prov.seen[slug].Deferred {
			t.Errorf("%s devait attendre (ordre stable par identifiant)", slug)
		}
	}
	// Reportés = en attente (HistoryPending) tant que leur tour n'est pas venu.
	tracker := newHistoryTracker(res)
	for _, slug := range []string{"p08", "p09"} {
		if !tracker.pending[slug] {
			t.Errorf("%s : HistoryPending faux alors que son rattrapage est reporté", slug)
		}
	}
}

// Les joueurs complets ne sont pas comptés dans le budget : trois joueurs au rattrapage et un complet.
func TestRunDiscovery_Budget_LesCompletsNeComptentPas(t *testing.T) {
	ids := slugs(4)
	loader := catchUpLoader(ids[:3])
	prov := &budgetProvider{}
	if _, err := RunDiscovery(context.Background(), mkPlayers(ids...), loader, prov); err != nil {
		t.Fatal(err)
	}
	for _, slug := range ids[:3] {
		if got := prov.seen[slug].catchUpBudget(); got != historystate.PassBudget()/3 {
			t.Errorf("%s : budget %d, want %d", slug, got, historystate.PassBudget()/3)
		}
	}
	if h := prov.seen[ids[3]]; h.Incomplete || h.Deferred || h.Budget != 0 {
		t.Errorf("joueur complet : %+v, want inchangé", h)
	}
}

// multiEnv : plusieurs joueurs dont l'historique se rattrape par cycles successifs (vrai chemin du cycle,
// bases `sync_meta` en mémoire).
type multiEnv struct {
	t       *testing.T
	dbs     map[string]*sql.DB
	clients map[string]*pagedClient
	known   map[string]bool // identifiants préfixés par joueur : « <slug>-m00001 »
	mu      gosync.Mutex
	fetcher *mockFetcher
	persist *mockCyclePersister
}

func newMultiEnv(t *testing.T, matches int, slugs ...string) *multiEnv {
	t.Helper()
	e := &multiEnv{
		t: t, dbs: map[string]*sql.DB{}, clients: map[string]*pagedClient{}, known: map[string]bool{},
		fetcher: &mockFetcher{perMatchErr: map[string]error{}},
	}
	for _, s := range slugs {
		single := newHistEnv(t, 0)
		e.dbs[s] = single.db
		e.clients[s] = newPagedClient(matchIDs(s+"-m", matches))
	}
	e.persist = &mockCyclePersister{hookBeforeReturn: func(_ context.Context, b CycleBatch) {
		e.mu.Lock()
		defer e.mu.Unlock()
		for id := range b.Matches {
			e.known[id] = true
		}
	}}
	return e
}

// LoadKnown (KnownLoader).
func (e *multiEnv) LoadKnown(ctx context.Context, p PlayerProfile) (KnownState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := map[string]bool{}
	for id := range e.known {
		if len(id) > len(p.PlayerSlug) && id[:len(p.PlayerSlug)+1] == p.PlayerSlug+"-" {
			k[id] = true
		}
	}
	return KnownState{Set: knownset.Set{Known: k}, History: readHistoryState(ctx, e.dbs[p.PlayerSlug], p.PlayerSlug)}, nil
}

func (e *multiEnv) cycle(players ...string) CycleResult {
	e.t.Helper()
	provider := NewMatchListProvider(func(gamertag, _ string) HaloClient { return e.clients[gamertag[:len(gamertag)-len("_GT")]] }, "matchmaking", 25, 20)
	orch := NewCycleOrchestrator(e, provider, e.fetcher, &mockEnrichmentFetcher{}, e.persist, &mockPostSyncRunner{}, CycleConfig{})
	orch.WithHistoryWriter(func(_ context.Context, gamertag string) (*sql.DB, func(), error) {
		return e.dbs[gamertag[:len(gamertag)-len("_GT")]], func() {}, nil
	})
	res, err := orch.Run(context.Background(), mkPlayers(players...))
	if err != nil {
		e.t.Fatal(err)
	}
	return res
}

func (e *multiEnv) knownOf(slug string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	n := 0
	for id := range e.known {
		if id[:len(slug)+1] == slug+"-" {
			n++
		}
	}
	return n
}

func (e *multiEnv) complete(slug string) bool {
	_, ok, err := syncMetaGet(e.dbs[slug], historystate.Key)
	return err == nil && ok
}

// Deux joueurs de 150 matchs dans le même cycle : 100 chacun au premier (200 au total), puis 50 chacun et
// les deux fins rangées.
func TestCycleHistory_DeuxJoueurs_BudgetPartageEtLesDeuxFinissent(t *testing.T) {
	e := newMultiEnv(t, 150, "alice", "bob")

	res := e.cycle("alice", "bob")
	if res.UniqueMatches != historystate.PassBudget() {
		t.Errorf("cycle 1 : %d matchs uniques, want %d (budget du cycle)", res.UniqueMatches, historystate.PassBudget())
	}
	for _, s := range []string{"alice", "bob"} {
		if got := e.knownOf(s); got != historystate.PassBudget()/2 {
			t.Errorf("cycle 1 : %s a %d matchs, want %d", s, got, historystate.PassBudget()/2)
		}
		if !res.PerPlayer[s].HistoryPending || e.complete(s) {
			t.Errorf("cycle 1 : %s HistoryPending %v, fin rangée %v, want vrai et non", s, res.PerPlayer[s].HistoryPending, e.complete(s))
		}
	}

	res = e.cycle("alice", "bob")
	for _, s := range []string{"alice", "bob"} {
		if e.knownOf(s) != 150 || !e.complete(s) || res.PerPlayer[s].HistoryPending {
			t.Errorf("cycle 2 : %s %d matchs, fin rangée %v, HistoryPending %v, want 150, oui, non", s, e.knownOf(s), e.complete(s), res.PerPlayer[s].HistoryPending)
		}
	}
}

// Dix joueurs : huit rattrapent au premier cycle (25 chacun, marque-page rangé), les deux autres font un delta
// borné sans marque-page, en attente, puis finissent quand leur tour vient.
func TestCycleHistory_DixJoueurs_HuitRattrapentDeuxAttendent(t *testing.T) {
	ids := slugs(10)
	e := newMultiEnv(t, 60, ids...)

	res := e.cycle(ids...)
	for _, s := range ids[:8] {
		if c, ok := historystate.LoadCursor(context.Background(), e.dbs[s]); !ok || c.Offset != catchUpMinBudget {
			t.Errorf("cycle 1 : %s marque-page %+v (%v), want un rattrapage à %d", s, c, ok, catchUpMinBudget)
		}
	}
	for _, s := range ids[8:] {
		if _, ok := historystate.LoadCursor(context.Background(), e.dbs[s]); ok || e.knownOf(s) > catchUpMinBudget {
			t.Errorf("cycle 1 : %s marque-page %v, %d matchs, want aucun marque-page et un delta borné à %d", s, ok, e.knownOf(s), catchUpMinBudget)
		}
		if !res.PerPlayer[s].HistoryPending {
			t.Errorf("cycle 1 : %s HistoryPending faux alors que son rattrapage est reporté", s)
		}
	}
	for cycle := 2; cycle <= 8; cycle++ {
		e.cycle(ids...)
	}
	for _, s := range ids {
		if e.knownOf(s) != 60 || !e.complete(s) {
			t.Errorf("%s : %d matchs, fin rangée %v, want 60 et oui après quelques cycles", s, e.knownOf(s), e.complete(s))
		}
	}
}
