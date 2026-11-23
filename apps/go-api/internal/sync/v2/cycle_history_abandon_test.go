// Package v2 — cycle_history_abandon_test.go : les matchs en échec de rattrapage comptent leurs
// passes dans l'état rangé avec le marque-page ; à maxCatchUpAttempts passes, le match est abandonné
// et ne retient plus ni le budget, ni la fin d'historique, ni le marque-page.
package v2

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

func (e *histEnv) failures() []historystate.Failure {
	return historystate.LoadCatchUp(context.Background(), e.db).Failures
}

// Un match qui échoue toujours : une passe de plus à chaque cycle, abandonné à la maxCatchUpAttempts-ième
// (journalisé et compté une fois), la fin d'historique se range dans cette même passe, l'état est effacé.
func TestCycleHistory_MatchQuiEchoueToujours_AbandonneEtFinRangee(t *testing.T) {
	budget := historystate.PassBudget()
	size := budget + 30
	e := newHistEnv(t, size)
	bad := e.ids[size-5]
	e.fetcher.perMatchErr[bad] = serverErr()
	title := ctxkeys.TitleSlug(context.Background())
	before := observability.LoadCounterT(title, counterHistoryAbandoned)

	if _, err := e.cycle(); err != nil { // passe 1 : les 200 premiers, sans échec
		t.Fatal(err)
	}
	for attempt := 1; attempt < maxCatchUpAttempts; attempt++ {
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		if e.complete() || !out.HistoryPending {
			t.Fatalf("échec %d : fin rangée %v, HistoryPending %v, want non et vrai (pas encore abandonné)", attempt, e.complete(), out.HistoryPending)
		}
		got := e.failures()
		if len(got) != 1 || got[0] != (historystate.Failure{MatchID: bad, Attempts: attempt}) {
			t.Fatalf("échec %d : états rangés %v, want [{%s %d}]", attempt, got, bad, attempt)
		}
		if c, ok := e.cursor(); !ok || c.Offset != budget {
			t.Errorf("échec %d : le marque-page ne doit pas avancer (%+v, %v)", attempt, c, ok)
		}
		if observability.LoadCounterT(title, counterHistoryAbandoned) != before {
			t.Fatalf("échec %d : abandon compté trop tôt", attempt)
		}
	}

	out, err := e.cycle() // l'échec de seuil : abandonné, fin rangée
	if err != nil {
		t.Fatal(err)
	}
	if !e.complete() || out.HistoryPending {
		t.Fatalf("passe d'abandon : fin rangée %v, HistoryPending %v, want oui et non", e.complete(), out.HistoryPending)
	}
	if got := observability.LoadCounterT(title, counterHistoryAbandoned) - before; got != 1 {
		t.Errorf("compteur des abandons +%d, want +1", got)
	}
	if len(e.failures()) != 0 {
		t.Errorf("états rangés %v, want vide après la fin d'historique", e.failures())
	}
	if e.known[bad] {
		t.Error("le match abandonné ne doit pas être rangé")
	}
}

// Un match abandonné ne retient plus le marque-page ni la fin : il est abandonné à la
// maxCatchUpAttempts-ième passe (le marque-page avance alors), la passe suivante ne le liste plus et
// range la fin d'historique.
func TestCycleHistory_MatchAbandonne_NeRetientPlusRien(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)
	bad := e.ids[3]
	e.fetcher.perMatchErr[bad] = serverErr()

	for i := 1; i <= maxCatchUpAttempts; i++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
		_, hasCursor := e.cursor()
		if wantCursor := i == maxCatchUpAttempts; hasCursor != wantCursor {
			t.Fatalf("passe %d : marque-page présent %v, want %v", i, hasCursor, wantCursor)
		}
	}
	if got := e.failures(); len(got) != 1 || got[0].MatchID != bad || !isAbandoned(got[0].Attempts) {
		t.Fatalf("échecs rangés %v, want le match abandonné conservé", got)
	}

	e.fetcher.calledWith.Delete(bad)
	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if _, retried := e.fetcher.calledWith.Load(bad); retried {
		t.Error("le match abandonné est encore retenté")
	}
	if !e.complete() || out.HistoryPending {
		t.Errorf("fin rangée %v, HistoryPending %v, want oui et non : un abandonné ne retient pas la fin", e.complete(), out.HistoryPending)
	}
}

// Un match qui échoue une fois puis réussit sort de la liste des échecs.
func TestCycleHistory_EchecPuisSucces_SortDeLaListe(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)
	bad := e.ids[budget+4] // dans la deuxième tranche
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	e.fetcher.perMatchErr[bad] = serverErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := e.failures(); len(got) != 1 || got[0] != (historystate.Failure{MatchID: bad, Attempts: 1}) {
		t.Fatalf("échecs rangés %v, want [{%s 1}]", got, bad)
	}
	delete(e.fetcher.perMatchErr, bad)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := e.failures(); len(got) != 0 {
		t.Errorf("échecs rangés %v, want vide après le succès", got)
	}
	if c, ok := e.cursor(); !ok || c.Offset <= 2*budget {
		t.Errorf("marque-page %+v (%v), want un offset au-delà de %d (avancé une fois tout résolu)", c, ok, 2*budget)
	}
}

// Un 404 reste traité comme avant : jamais rangé comme échec, ne retient rien.
func TestCycleHistory_404_JamaisUnEchec(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 2*budget)
	e.fetcher.perMatchErr[e.ids[7]] = notFoundErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := e.failures(); len(got) != 0 {
		t.Errorf("échecs rangés %v, want aucun pour un 404", got)
	}
	if c, ok := e.cursor(); !ok || c.Offset != budget {
		t.Errorf("marque-page %+v (%v), want offset %d", c, ok, budget)
	}
}

// Borne : au plus MaxFailures échecs rangés, les plus anciens sortent.
func TestCycleHistory_BorneDesEchecs(t *testing.T) {
	e := newHistEnv(t, 100)
	var prev []historystate.Failure
	for i := 0; i < historystate.MaxFailures-2; i++ {
		prev = append(prev, historystate.Failure{MatchID: fmt.Sprintf("ancien%03d", i), Attempts: 1})
	}
	if !historystate.SaveFailures(context.Background(), e.db, prev) {
		t.Fatal("SaveFailures")
	}
	const failing = 5 // sous la part d'une passe de masse : les échecs comptent
	for _, id := range e.ids[:failing] {
		e.fetcher.perMatchErr[id] = errors.New("décodage")
	}
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	got := e.failures()
	if len(got) != historystate.MaxFailures {
		t.Fatalf("%d échecs rangés, want %d", len(got), historystate.MaxFailures)
	}
	over := len(prev) + failing - historystate.MaxFailures
	if got[0].MatchID != prev[over].MatchID || got[len(got)-1].MatchID != e.ids[failing-1] {
		t.Errorf("entrées gardées de %s à %s, want de %s (les %d plus anciennes sorties) à %s",
			got[0].MatchID, got[len(got)-1].MatchID, prev[over].MatchID, over, e.ids[failing-1])
	}
}
