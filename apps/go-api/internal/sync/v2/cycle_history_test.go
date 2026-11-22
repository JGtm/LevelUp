// Package v2 — cycle_history_test.go : le cycle range la fin d'historique et le marque-page d'un
// joueur seulement sous les conditions de la règle (history.go) : fin vue, découverte sans erreur,
// persistance réussie, chaque match listé persisté (relu) ou introuvable (404/410).
//
// Le cycle est joué en entier : vraie découverte (matchListProviderV2 sur une liste d'API
// factice), vraie lecture et écriture de `sync_meta` (DuckDB en mémoire, par historystate), faux
// récupérateur et faux persisteur qui rangent les matchs dans l'ensemble connu.
package v2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	gosync "sync"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/knownset"

	_ "github.com/duckdb/duckdb-go/v2"
)

// histEnv est un joueur (alice) dont l'historique se rattrape par cycles successifs.
type histEnv struct {
	t      *testing.T
	db     *sql.DB // sync_meta du joueur
	client *pagedClient
	ids    []string

	mu    gosync.Mutex
	known map[string]bool

	fetcher       *mockFetcher
	persister     *mockCyclePersister
	dropOnPersist map[string]bool // matchs que la persistance « oublie » en silence
	writer        PlayerDBOpener  // nil = écrivain non câblé
}

func newHistEnv(t *testing.T, n int) *histEnv {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
		t.Fatalf("create sync_meta: %v", err)
	}
	e := &histEnv{
		t: t, db: db, ids: matchIDs("m", n), known: map[string]bool{},
		fetcher:       &mockFetcher{perMatchErr: map[string]error{}},
		dropOnPersist: map[string]bool{},
	}
	e.client = newPagedClient(e.ids)
	e.persister = &mockCyclePersister{hookBeforeReturn: func(_ context.Context, b CycleBatch) {
		e.mu.Lock()
		defer e.mu.Unlock()
		for id := range b.Matches {
			if !e.dropOnPersist[id] {
				e.known[id] = true
			}
		}
	}}
	e.writer = func(context.Context, string) (*sql.DB, func(), error) { return db, func() {}, nil }
	return e
}

// LoadKnown (KnownLoader) : l'ensemble connu courant et l'état d'historique lu sur la base du joueur.
func (e *histEnv) LoadKnown(ctx context.Context, _ PlayerProfile) (KnownState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := make(map[string]bool, len(e.known))
	for id := range e.known {
		k[id] = true
	}
	return KnownState{Set: knownset.Set{Known: k}, History: readHistoryState(ctx, e.db)}, nil
}

// cycle joue un cycle complet pour alice et rend son issue.
func (e *histEnv) cycle() (PlayerOutcome, error) {
	e.t.Helper()
	e.client.starts = nil
	provider := NewMatchListProvider(func(string, string) HaloClient { return e.client }, "matchmaking", 25, 20)
	orch := NewCycleOrchestrator(e, provider, e.fetcher, &mockEnrichmentFetcher{}, e.persister, &mockPostSyncRunner{}, CycleConfig{})
	if e.writer != nil {
		orch.WithHistoryWriter(e.writer)
	}
	res, err := orch.Run(context.Background(), mkPlayers("alice"))
	return res.PerPlayer["alice"], err
}

func (e *histEnv) complete() bool {
	_, ok, err := syncMetaGet(e.db, historystate.Key)
	return err == nil && ok
}

func (e *histEnv) cursor() (historystate.Cursor, bool) {
	return historystate.LoadCursor(context.Background(), e.db)
}

func (e *histEnv) knownCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.known)
}

func syncMetaGet(db *sql.DB, key string) (string, bool, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return v, err == nil, err
}

// not404 / err500 : erreurs de récupération d'un match, telles que le client Halo les rend.
func notFoundErr() error {
	return fmt.Errorf("GetMatchStats: %w", &syncpkg.HTTPError{StatusCode: 404, URL: "u"})
}
func goneErr() error {
	return fmt.Errorf("GetMatchStats: %w", &syncpkg.HTTPError{StatusCode: 410, URL: "u"})
}
func serverErr() error {
	return fmt.Errorf("GetMatchStats: %w", &syncpkg.HTTPError{StatusCode: 500, URL: "u"})
}

// Convergence sur le chemin du cycle : 450 matchs rangés en trois passes (budget de 200), la fin
// rangée à la troisième, puis un delta d'une seule page.
func TestCycleHistory_ConvergeEnTroisPassesPuisDelta(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 2*budget+50)

	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if e.knownCount() != budget || e.complete() {
		t.Fatalf("passe 1 : %d connus, fin rangée %v, want %d et non", e.knownCount(), e.complete(), budget)
	}
	if c, ok := e.cursor(); !ok || c.Offset != budget || c.LastMatchID != e.ids[budget-1] {
		t.Errorf("passe 1 : marque-page %+v (%v), want offset %d sur %s", c, ok, budget, e.ids[budget-1])
	}
	if !out.HistoryPending {
		t.Error("passe 1 : HistoryPending = false, l'historique n'est pas fini")
	}

	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if e.knownCount() != 2*budget {
		t.Fatalf("passe 2 : %d connus, want %d", e.knownCount(), 2*budget)
	}
	if got := e.client.startsBetween(25, budget-25); got != 0 {
		t.Errorf("passe 2 : %d lectures dans la zone déjà rangée, le marque-page devait l'éviter", got)
	}
	if c, ok := e.cursor(); !ok || c.Offset != 2*budget {
		t.Errorf("passe 2 : marque-page %+v (%v), want offset %d", c, ok, 2*budget)
	}

	out, err = e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if e.knownCount() != 2*budget+50 || !e.complete() {
		t.Fatalf("passe 3 : %d connus, fin rangée %v, want %d et oui", e.knownCount(), e.complete(), 2*budget+50)
	}
	if _, ok := e.cursor(); ok {
		t.Error("passe 3 : le marque-page doit être effacé avec la fin d'historique")
	}
	if out.HistoryPending {
		t.Error("passe 3 : HistoryPending = true alors que la fin est rangée")
	}

	out, err = e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.client.starts) != 1 || out.HistoryPending {
		t.Errorf("passe 4 : %d lectures et HistoryPending %v, want une seule page (delta) et faux", len(e.client.starts), out.HistoryPending)
	}
}

// La fin d'historique se range seulement quand tout est résolu ; chaque cas laisse la clé absente.
func TestCycleHistory_FinRangeeSeulementSousD5(t *testing.T) {
	budget := historystate.PassBudget()
	size := budget + 30 // deux passes : 200 puis 30 avec la fin vue

	// prepare joue la première passe (200 matchs, marque-page rangé) et rend l'environnement.
	prepare := func(t *testing.T) *histEnv {
		e := newHistEnv(t, size)
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
		if c, ok := e.cursor(); !ok || c.Offset != budget {
			t.Fatalf("préparation : marque-page %+v (%v), want offset %d", c, ok, budget)
		}
		return e
	}
	last := func(e *histEnv) string { return e.ids[size-1] }

	cases := []struct {
		name         string
		arrange      func(e *histEnv)
		wantComplete bool
		wantErr      bool
		wantPending  bool
		wantStatus   string
	}{
		{
			name:         "tout réussi",
			arrange:      func(*histEnv) {},
			wantComplete: true, wantStatus: "ok",
		},
		{
			name:        "erreur de découverte (liste d'API en échec)",
			arrange:     func(e *histEnv) { e.client.failAt, e.client.failErr = budget-1, errors.New("api 500") },
			wantPending: true, wantStatus: "failed",
		},
		{
			name:        "erreur de récupération (500)",
			arrange:     func(e *histEnv) { e.fetcher.perMatchErr[last(e)] = serverErr() },
			wantPending: true, wantStatus: "ok",
		},
		{
			name:         "match introuvable (404) : ne bloque pas",
			arrange:      func(e *histEnv) { e.fetcher.perMatchErr[last(e)] = notFoundErr() },
			wantComplete: true, wantStatus: "ok",
		},
		{
			name:         "match introuvable (410) : ne bloque pas",
			arrange:      func(e *histEnv) { e.fetcher.perMatchErr[last(e)] = goneErr() },
			wantComplete: true, wantStatus: "ok",
		},
		{
			name: "404 et 500 mêlés : le 500 bloque",
			arrange: func(e *histEnv) {
				e.fetcher.perMatchErr[last(e)] = notFoundErr()
				e.fetcher.perMatchErr[e.ids[budget]] = serverErr()
			},
			wantPending: true, wantStatus: "ok",
		},
		{
			name:        "persistance en échec",
			arrange:     func(e *histEnv) { e.persister.err = errors.New("disque plein") },
			wantErr:     true,
			wantPending: true, wantStatus: "failed",
		},
		{
			name:        "persistance qui oublie un match en silence",
			arrange:     func(e *histEnv) { e.dropOnPersist[last(e)] = true },
			wantPending: true, wantStatus: "ok",
		},
		{
			name:        "écrivain non câblé",
			arrange:     func(e *histEnv) { e.writer = nil },
			wantPending: true, wantStatus: "ok",
		},
		{
			name: "base joueur indisponible en écriture",
			arrange: func(e *histEnv) {
				e.writer = func(context.Context, string) (*sql.DB, func(), error) { return nil, nil, errors.New("bail tenu") }
			},
			wantPending: true, wantStatus: "ok",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := prepare(t)
			tc.arrange(e)
			out, err := e.cycle()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got := e.complete(); got != tc.wantComplete {
				t.Errorf("fin d'historique rangée = %v, want %v", got, tc.wantComplete)
			}
			if out.HistoryPending != tc.wantPending {
				t.Errorf("HistoryPending = %v, want %v", out.HistoryPending, tc.wantPending)
			}
			if out.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", out.Status, tc.wantStatus)
			}
			if !tc.wantComplete {
				// Rien n'a avancé : le marque-page de la première passe reste en place.
				if c, ok := e.cursor(); !ok || c.Offset != budget {
					t.Errorf("marque-page %+v (%v), want l'ancien (offset %d)", c, ok, budget)
				}
			}
		})
	}
}

// Les passes suivantes reprennent les échecs : un 500 transitoire bloque une passe, la suivante
// (erreur disparue) range la fin.
func TestCycleHistory_EchecTransitoire_LaPasseSuivanteReprend(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, budget+30)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	e.fetcher.perMatchErr[e.ids[budget+5]] = serverErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if e.complete() {
		t.Fatal("fin rangée malgré un match en échec")
	}
	delete(e.fetcher.perMatchErr, e.ids[budget+5])
	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if !e.complete() || out.HistoryPending {
		t.Errorf("fin rangée %v, HistoryPending %v, want vrai et faux", e.complete(), out.HistoryPending)
	}
	if e.knownCount() != budget+30 {
		t.Errorf("%d matchs rangés, want %d", e.knownCount(), budget+30)
	}
}

// Le marque-page n'avance que si tous les inconnus listés sont résolus : un 404 passe, un 500 non.
func TestCycleHistory_MarquePageSeulementSiToutEstResolu(t *testing.T) {
	budget := historystate.PassBudget()
	t.Run("404 : avance", func(t *testing.T) {
		e := newHistEnv(t, 3*budget)
		e.fetcher.perMatchErr[e.ids[3]] = notFoundErr()
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
		if c, ok := e.cursor(); !ok || c.Offset != budget {
			t.Errorf("marque-page %+v (%v), want offset %d", c, ok, budget)
		}
	})
	t.Run("500 : pas de marque-page", func(t *testing.T) {
		e := newHistEnv(t, 3*budget)
		e.fetcher.perMatchErr[e.ids[3]] = serverErr()
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := e.cursor(); ok {
			t.Error("marque-page rangé malgré un match en échec")
		}
		if !out.HistoryPending {
			t.Error("HistoryPending = false")
		}
	})
}

// Historique complet au départ : aucune lecture ni écriture de l'état, HistoryPending faux, delta. Cinq matchs
// neufs en haut : le delta lit la page 0 puis la page 1, qui complète la suite de 25 connus (D-23 ; un delta
// arrêté au premier connu ne lisait que la page 0).
func TestCycleHistory_HistoriqueComplet_Delta(t *testing.T) {
	e := newHistEnv(t, 300)
	if !historystate.MarkComplete(context.Background(), e.db, "alice") {
		t.Fatal("MarkComplete")
	}
	for _, id := range e.ids[5:] {
		e.known[id] = true
	}
	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if out.HistoryPending || len(e.client.starts) != 2 || e.knownCount() != 300 {
		t.Errorf("HistoryPending %v, %d lectures, %d connus : want faux, 2 pages, 300", out.HistoryPending, len(e.client.starts), e.knownCount())
	}
}

// Les introuvables sont comptés (compteur expvar titré).
func TestCycleHistory_IntrouvablesComptes(t *testing.T) {
	title := ctxkeys.TitleSlug(context.Background())
	before := observability.LoadCounterT(title, counterHistoryNotFound)
	e := newHistEnv(t, 50)
	e.fetcher.perMatchErr[e.ids[1]] = notFoundErr()
	e.fetcher.perMatchErr[e.ids[2]] = goneErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := observability.LoadCounterT(title, counterHistoryNotFound) - before; got != 2 {
		t.Errorf("compteur des introuvables +%d, want +2", got)
	}
	if !e.complete() {
		t.Error("fin non rangée alors que seuls des 404/410 restent")
	}
}
