// Package v2 — cycle_history_systemic_test.go : une panne de l'API ne fait pas abandonner des matchs
// sains (historyfail), et un match abandonné est retenté après historystate.AbandonRetryAfter, tentatives
// remises à zéro (retry.go).
package v2

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

// setClock remplace l'horloge de historystate le temps du test ; le rendu avance d'une durée.
func setClock(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	old := historystate.Now
	historystate.Now = func() time.Time { return now }
	t.Cleanup(func() { historystate.Now = old })
	return func(d time.Duration) { now = now.Add(d) }
}

func abandonedCount() int64 {
	return observability.LoadCounterT(ctxkeys.TitleSlug(context.Background()), counterHistoryAbandoned)
}

func tooManyRequestsErr() error {
	return fmt.Errorf("GetMatchStats: %w", &syncpkg.HTTPError{StatusCode: 429, URL: "u"})
}

// Toutes les statistiques en 500 pendant plus de maxCatchUpAttempts passes, puis l'API saine : les matchs
// sont tous rangés, aucun n'a été abandonné.
func TestCycleHistory_PanneDeLAPI_AucunAbandon(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, budget+30)
	for _, id := range e.ids {
		e.fetcher.perMatchErr[id] = serverErr()
	}
	before := abandonedCount()
	for pass := 1; pass <= maxCatchUpAttempts+1; pass++ {
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		if !out.HistoryPending || e.knownCount() != 0 || len(e.failures()) != 0 {
			t.Fatalf("passe %d en panne : HistoryPending %v, %d connus, échecs %v, want vrai, 0, aucun", pass, out.HistoryPending, e.knownCount(), e.failures())
		}
	}
	if got := abandonedCount() - before; got != 0 {
		t.Fatalf("%d abandons pendant la panne, want 0", got)
	}

	clear(e.fetcher.perMatchErr) // l'API est saine
	for pass := 0; pass < 3 && !e.complete(); pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
	}
	if e.knownCount() != budget+30 || !e.complete() {
		t.Errorf("après la panne : %d connus, fin rangée %v, want %d et oui", e.knownCount(), e.complete(), budget+30)
	}
	if got := abandonedCount() - before; got != 0 {
		t.Errorf("%d abandons, want 0", got)
	}
}

// 5 % de matchs en échec durable (défaut du match, pas une panne) : abandonnés à la maxCatchUpAttempts-ième passe.
func TestCycleHistory_CinqPourCentDEchecsDurables_AbandonALaTroisiemePasse(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, budget+30)
	bad := e.ids[:budget/20]
	for _, id := range bad {
		e.fetcher.perMatchErr[id] = errors.New("décodage du match")
	}
	before := abandonedCount()
	for pass := 1; pass < maxCatchUpAttempts; pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
		if got := abandonedCount() - before; got != 0 {
			t.Fatalf("passe %d : %d abandons, want 0 avant la %de passe", pass, got, maxCatchUpAttempts)
		}
	}
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := abandonedCount() - before; got != int64(len(bad)) {
		t.Errorf("%d abandons à la %de passe, want %d", got, maxCatchUpAttempts, len(bad))
	}
	if e.knownCount() != budget+30-len(bad) {
		t.Errorf("%d matchs rangés, want %d (les sains)", e.knownCount(), budget+30-len(bad))
	}
}

// Plus de 10 % des matchs listés en échec, même sans être des 5xx : passe de masse, aucune tentative.
func TestCycleHistory_PasseDeMasse_AucuneTentative(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, budget+30)
	for _, id := range e.ids[:budget/4] {
		e.fetcher.perMatchErr[id] = errors.New("écriture refusée")
	}
	before := abandonedCount()
	for pass := 1; pass <= maxCatchUpAttempts+1; pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
	}
	if got := abandonedCount() - before; got != 0 || len(e.failures()) != 0 {
		t.Errorf("%d abandons, échecs %v : une passe de masse ne compte aucune tentative", got, e.failures())
	}
}

// timeoutErr : une erreur de liaison (net.Error) qui a dépassé son délai.
type timeoutErr struct{}

func (*timeoutErr) Error() string   { return "i/o timeout" }
func (*timeoutErr) Timeout() bool   { return true }
func (*timeoutErr) Temporary() bool { return true }

// Un 429 ou une erreur réseau ne comptent jamais, même seuls ; un dernier match isolé en échec durable,
// lui, est bien abandonné (la part d'une passe de quelques matchs ne dit rien).
func TestCycleHistory_EchecPassagerNeCompteJamais_DernierMatchIsoleAbandonne(t *testing.T) {
	for name, transient := range map[string]error{
		"429":    tooManyRequestsErr(),
		"réseau": fmt.Errorf("doGet: %w", &timeoutErr{}),
	} {
		t.Run(name, func(t *testing.T) {
			e := newHistEnv(t, 1)
			e.fetcher.perMatchErr[e.ids[0]] = transient
			before := abandonedCount()
			for pass := 0; pass < maxCatchUpAttempts+2; pass++ {
				if _, err := e.cycle(); err != nil {
					t.Fatal(err)
				}
			}
			if got := abandonedCount() - before; got != 0 || e.complete() {
				t.Errorf("%d abandons, fin rangée %v, want 0 et non", got, e.complete())
			}
		})
	}
	e := newHistEnv(t, 1)
	e.fetcher.perMatchErr[e.ids[0]] = errors.New("décodage du match")
	before := abandonedCount()
	for pass := 0; pass < maxCatchUpAttempts; pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
	}
	if got := abandonedCount() - before; got != 1 || !e.complete() {
		t.Errorf("%d abandons, fin rangée %v, want 1 et oui", got, e.complete())
	}
}

// abandonEnv rend un environnement dont le match `bad` a été abandonné puis la fin rangée.
func abandonEnv(t *testing.T) (e *histEnv, bad string) {
	t.Helper()
	e = newHistEnv(t, 60)
	bad = e.ids[40]
	e.fetcher.perMatchErr[bad] = errors.New("décodage du match")
	for pass := 0; pass < maxCatchUpAttempts; pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
	}
	if !e.complete() || !e.skipSet()[bad] {
		t.Fatalf("préparation : fin rangée %v, abandon dans l'ensemble %v, want oui et oui", e.complete(), e.skipSet()[bad])
	}
	return e, bad
}

// Un match abandonné n'est pas redemandé avant AbandonRetryAfter ; passé ce délai il est retenté par
// identifiant (le delta ne descend pas jusqu'à lui) et, rangé, devient connu.
func TestCycleHistory_AbandonRetenteApres24h(t *testing.T) {
	advance := setClock(t)
	e, bad := abandonEnv(t)

	advance(historystate.AbandonRetryAfter - time.Minute)
	e.fetcher.calledWith.Delete(bad)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if _, retried := e.fetcher.calledWith.Load(bad); retried {
		t.Fatal("retenté avant le délai de 24 h")
	}

	advance(2 * time.Minute)
	delete(e.fetcher.perMatchErr, bad) // le match est maintenant lisible
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if !e.known[bad] {
		t.Error("le match abandonné n'a pas été retenté après 24 h")
	}
}

// Retenté en vain après 24 h : abandonné de nouveau pour 24 h (date renouvelée), pas redemandé le cycle suivant.
func TestCycleHistory_AbandonRetenteEnVain_AbandonneDeNouveau(t *testing.T) {
	advance := setClock(t)
	e, bad := abandonEnv(t)

	advance(historystate.AbandonRetryAfter + time.Hour)
	e.fetcher.calledWith.Delete(bad)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if _, retried := e.fetcher.calledWith.Load(bad); !retried {
		t.Fatal("abandon échu non retenté")
	}
	if got := historystate.LoadSkipSet(context.Background(), e.db); !got.Active[bad] || len(got.Due) != 0 {
		t.Fatalf("après l'échec du retour : actifs %v, échus %v, want %s de nouveau actif", got.Active, got.Due, bad)
	}
	e.fetcher.calledWith.Delete(bad)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if _, retried := e.fetcher.calledWith.Load(bad); retried {
		t.Error("redemandé au cycle suivant : la date d'abandon n'a pas été renouvelée")
	}
}

// Retenté pendant une panne de l'API : l'abandon échu le reste, rien n'est renouvelé.
func TestCycleHistory_AbandonRetentePendantUnePanne_ResteEchu(t *testing.T) {
	advance := setClock(t)
	e, bad := abandonEnv(t)

	advance(historystate.AbandonRetryAfter + time.Hour)
	e.fetcher.perMatchErr[bad] = tooManyRequestsErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := historystate.LoadSkipSet(context.Background(), e.db); len(got.Due) != 1 || got.Due[0] != bad {
		t.Errorf("échus %v, want [%s] : un échec passager ne renouvelle pas l'abandon", got.Due, bad)
	}
}

// Introuvable (404) au retour : définitif, plus jamais redemandé.
func TestCycleHistory_AbandonRetenteIntrouvable_Definitif(t *testing.T) {
	advance := setClock(t)
	e, bad := abandonEnv(t)

	advance(historystate.AbandonRetryAfter + time.Hour)
	e.fetcher.perMatchErr[bad] = notFoundErr()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	advance(10 * historystate.AbandonRetryAfter)
	e.fetcher.calledWith.Delete(bad)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if _, retried := e.fetcher.calledWith.Load(bad); retried {
		t.Error("un match introuvable (404) est redemandé : il doit rester exclu pour de bon")
	}
}

// Au rattrapage, un abandon échu repart de zéro : maxCatchUpAttempts nouvelles passes avant un nouvel abandon.
func TestCycleHistory_AbandonEchuAuRattrapage_TentativesRemisesAZero(t *testing.T) {
	advance := setClock(t)
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)
	bad := e.ids[3]
	e.fetcher.perMatchErr[bad] = errors.New("décodage du match")
	for pass := 0; pass < maxCatchUpAttempts; pass++ {
		if _, err := e.cycle(); err != nil {
			t.Fatal(err)
		}
	}
	if got := e.failures(); len(got) != 1 || !isAbandoned(got[0].Attempts) {
		t.Fatalf("préparation : échecs %v, want un abandonné", got)
	}
	advance(historystate.AbandonRetryAfter + time.Hour)
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if got := e.failures(); len(got) != 1 || got[0] != (historystate.Failure{MatchID: bad, Attempts: 1}) {
		t.Errorf("échecs %v, want [{%s 1}] : tentatives remises à zéro puis une passe comptée", got, bad)
	}
}

// CatchUpInserted : parmi les matchs rangés d'un cycle, ceux que le rattrapage ramène, hors nouveaux matchs
// arrivés au-dessus de l'ancre du marque-page (le delta du dessus) ; le budget réparti du joueur ne change pas la règle.
func TestCycleHistory_CatchUpInserted_HorsDeltaDuDessus(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)

	out, err := e.cycle() // première passe, sans marque-page : tout est du rattrapage
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted != budget || out.CatchUpInserted != budget {
		t.Fatalf("passe 1 : insérés %d, dont rattrapage %d, want %d et %d", out.MatchesInserted, out.CatchUpInserted, budget, budget)
	}

	const fresh = 7
	e.client.ids = append(matchIDs("n", fresh), e.client.ids...)
	out, err = e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted != budget || out.CatchUpInserted != budget-fresh {
		t.Errorf("passe 2 : insérés %d, dont rattrapage %d, want %d et %d (les %d neufs du dessus sont des nouveaux)",
			out.MatchesInserted, out.CatchUpInserted, budget, budget-fresh, fresh)
	}

	// Historique complet : tout est nouveau, rien n'est du rattrapage.
	e.markDeltaOnly()
	e.client.ids = append(matchIDs("p", 3), e.client.ids...)
	out, err = e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted == 0 || out.CatchUpInserted != 0 {
		t.Errorf("delta : insérés %d, dont rattrapage %d, want des matchs et 0", out.MatchesInserted, out.CatchUpInserted)
	}
}

// Un abandon retenté puis rangé n'est pas un nouveau match : il compte du côté du rattrapage, même à l'historique complet.
func TestCycleHistory_CatchUpInserted_AbandonRetenteNEstPasNouveau(t *testing.T) {
	advance := setClock(t)
	e, bad := abandonEnv(t)
	advance(historystate.AbandonRetryAfter + time.Hour)
	delete(e.fetcher.perMatchErr, bad)
	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if !e.known[bad] || out.MatchesInserted != 1 || out.CatchUpInserted != 1 {
		t.Errorf("rangé %v, insérés %d, dont rattrapage %d, want oui, 1 et 1", e.known[bad], out.MatchesInserted, out.CatchUpInserted)
	}
}
