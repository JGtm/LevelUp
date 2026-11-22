//go:build integration

// sync_v2_convergence_integration_test.go : le test qui manquait. Le rattrapage de l'historique livré
// en 8.1.2 n'était éprouvé que par le moteur V1, alors que l'ordonnanceur appelle le pipeline V2. Ici,
// des cycles successifs de l'ordonnanceur (RunOnceTrigger, orchestrateur construit par la fabrique de
// production, vraies bases DuckDB temporaires, faux service Halo : voir le banc) rattrapent un historique
// de 1 000 matchs, prennent les matchs neufs du dessus, rangent la fin, puis redeviennent des deltas
// d'une page ; la cadence de l'ordonnanceur suit (10 minutes pendant le rattrapage, l'intervalle réglé
// ensuite).
package main

import (
	"testing"
	"time"

	"levelup/go-api/internal/sync/historystate"
)

// catchUpCadence : intervalle de l'ordonnanceur pendant un rattrapage (catchUpCycleInterval, non exporté
// par le paquet de l'ordonnanceur ; la valeur est celle de la décision D-7).
const catchUpCadence = 10 * time.Minute

func assertNoUnexpectedRequests(t *testing.T, e *convergenceEnv) {
	t.Helper()
	if u := e.fake.unexpectedRequests(); len(u) > 0 {
		t.Logf("requêtes hors du faux Halo (réponse 404, sans conséquence sur le contrat) : %d, par exemple %q", len(u), u[0])
	}
}

// Historique de 1 000 matchs, pages de 25, des matchs neufs entre deux cycles : 200 matchs de plus par
// cycle (neufs du dessus compris), reprise au marque-page sans rebalayage, fin rangée au 6e cycle, puis
// une page, zéro rattrapage, plus rien en attente, cadence revenue à l'intervalle réglé.
func TestConvergence_V2_Ordonnanceur_Infinite_1000Matchs(t *testing.T) {
	e := newConvergenceEnv(t, 1000)
	const budget = 200
	slugBudget := historystate.PassBudget()
	if slugBudget != budget {
		t.Fatalf("budget de passe = %d, le scénario suppose %d", slugBudget, budget)
	}

	arrivals := map[int]int{2: 3, 4: 2} // cycle -> matchs neufs apparus en haut avant lui
	var (
		endCycle    int
		readsByPass []int
		totalReads  int
	)
	for cycle := 1; cycle <= 9 && endCycle == 0; cycle++ {
		fresh := e.fake.addOnTop(arrivals[cycle])
		before := e.registryIDs()
		missing := e.fake.listLen() - len(before)
		cursorBefore, hadCursorBefore := func() (historystate.Cursor, bool) {
			_, c := e.historyState()
			if c == nil {
				return historystate.Cursor{}, false
			}
			return *c, true
		}()

		out, reads := e.cycle()
		after := e.registryIDs()
		readsByPass = append(readsByPass, len(reads))
		totalReads += len(reads)

		if out.Status != "ok" {
			t.Fatalf("cycle %d : issue %q (%s)", cycle, out.Status, out.FirstError)
		}
		// +200 matchs par cycle, les neufs du dessus compris ; moins s'il n'en reste pas autant.
		if want := min(budget, missing); len(after)-len(before) != want {
			t.Fatalf("cycle %d : %d matchs de plus au registre, attendu %d (reste %d)", cycle, len(after)-len(before), want, missing)
		}
		for _, id := range fresh {
			if !after[id] {
				t.Fatalf("cycle %d : le match neuf du dessus %s n'a pas été pris", cycle, id)
			}
		}
		// Lectures de pages bornées : page 0 (le haut, jusqu'à l'ancre), la page de contrôle à la position
		// du marque-page translaté, puis budget/25 pages ; jamais un rebalayage de la liste.
		if bound := 2 + budget/convergencePageSize + 1; len(reads) > bound {
			t.Fatalf("cycle %d : %d pages d'historique lues %v, borne %d", cycle, len(reads), reads, bound)
		}
		if hadCursorBefore {
			// Les neufs du dessus sont dans la page 0 ; l'ancre donne leur nombre exact, la reprise est
			// à marque-page + neufs, lue d'une page dont la première entrée est le marque-page.
			resume := cursorBefore.Offset + len(fresh) - 1
			if len(reads) < 2 || reads[0] != 0 || reads[1] != resume {
				t.Fatalf("cycle %d : lectures %v, attendu la page 0 puis la reprise à %d (marque-page %+v, %d neufs)",
					cycle, reads, resume, cursorBefore, len(fresh))
			}
			for _, start := range reads[2:] {
				if start < resume {
					t.Fatalf("cycle %d : lecture à %d, en deçà de la reprise %d : rebalayage", cycle, start, resume)
				}
			}
		}

		complete, cursor := e.historyState()
		if complete == out.HistoryPending {
			t.Fatalf("cycle %d : clé de fin rangée=%v mais HistoryPending=%v (doivent être opposés)", cycle, complete, out.HistoryPending)
		}
		// La cadence de l'ordonnanceur suit l'issue : 10 minutes tant qu'il reste à rattraper, l'intervalle
		// réglé ensuite.
		if out.HistoryPending {
			e.waitCadence(catchUpCadence)
		} else {
			e.waitCadence(time.Duration(e.intervalH) * time.Hour)
		}

		if complete {
			endCycle = cycle
			if cursor != nil {
				t.Errorf("cycle %d : le marque-page doit être effacé quand la fin est rangée : %+v", cycle, *cursor)
			}
			continue
		}
		// Cycle intermédiaire : marque-page rangé sur le dernier match traité (bloc contigu depuis le haut).
		if cursor == nil {
			t.Fatalf("cycle %d : aucun marque-page rangé alors que la fin n'est pas vue", cycle)
		}
		if cursor.Offset != len(after) || cursor.LastMatchID != e.fake.idAt(cursor.Offset-1) {
			t.Fatalf("cycle %d : marque-page %+v, attendu {%d %s}", cycle, *cursor, len(after), e.fake.idAt(len(after)-1))
		}
	}
	if endCycle < 5 || endCycle > 7 {
		t.Fatalf("fin d'historique rangée au cycle %d, attendu le 5e ou le 6e (7e toléré avec les matchs neufs)", endCycle)
	}
	if got, want := len(e.registryIDs()), e.fake.listLen(); got != want {
		t.Fatalf("à la fin, %d matchs au registre sur %d listés", got, want)
	}
	if bound := e.fake.listLen()/convergencePageSize + endCycle*3; totalReads > bound {
		t.Errorf("%d pages lues au total, borne %d : le rattrapage rebalaye l'historique", totalReads, bound)
	}
	t.Logf("V2/ordonnanceur : fin rangée au cycle %d ; appels GetMatchHistory par cycle %v (total %d)", endCycle, readsByPass, totalReads)

	// Cycle suivant : une page, aucun rattrapage, rien en attente, cadence à l'intervalle réglé.
	out, reads := e.cycle()
	if len(reads) != 1 || reads[0] != 0 {
		t.Errorf("après la fin : lectures %v, attendu une seule page (la première)", reads)
	}
	if out.HistoryPending || out.MatchesInserted != 0 || out.Status != "ok" {
		t.Errorf("après la fin : %+v, attendu rien d'inséré ni d'en attente", out)
	}
	e.waitCadence(time.Duration(e.intervalH) * time.Hour)

	// Un match neuf après la fin : pris par le delta, toujours une page.
	fresh := e.fake.addOnTop(1)
	out, reads = e.cycle()
	if len(reads) != 1 || out.MatchesInserted != 1 || !e.registryIDs()[fresh[0]] || out.HistoryPending {
		t.Errorf("delta d'un match neuf : lectures %v, issue %+v", reads, out)
	}
	if complete, _ := e.historyState(); !complete {
		t.Error("la clé de fin d'historique ne doit pas disparaître après un delta")
	}
	assertNoUnexpectedRequests(t, e)
}

// Un match dont les stats sont toujours en erreur 500 ne garde pas le joueur en rattrapage à vie : il est
// retenté à chaque passe qui le liste, abandonné à la troisième, puis la fin d'historique est rangée.
func TestConvergence_V2_Ordonnanceur_Infinite_MatchEnEchecDurableAbandonne(t *testing.T) {
	e := newConvergenceEnv(t, 300)
	failing := e.fake.idAt(250)
	e.fake.setStatsStatus(failing, 500)

	var endCycle int
	var failedSeen []int
	for cycle := 1; cycle <= 8 && endCycle == 0; cycle++ {
		out, reads := e.cycle()
		complete, _ := e.historyState()
		if complete == out.HistoryPending {
			t.Fatalf("cycle %d : clé de fin rangée=%v mais HistoryPending=%v (lectures %v)", cycle, complete, out.HistoryPending, reads)
		}
		if e.fake.statsCallsFor(failing) > 0 {
			failedSeen = append(failedSeen, cycle)
		}
		if out.HistoryPending {
			e.waitCadence(catchUpCadence)
			continue
		}
		endCycle = cycle
		e.waitCadence(time.Duration(e.intervalH) * time.Hour)
	}
	if endCycle == 0 {
		t.Fatalf("fin non rangée en 8 cycles malgré l'abandon du match en échec (registre %d / %d)", len(e.registryIDs()), e.fake.listLen())
	}
	if len(failedSeen) != 3 {
		t.Errorf("le match en échec a été demandé aux cycles %v, attendu exactement trois passes (abandon à la 3e)", failedSeen)
	}
	ids := e.registryIDs()
	if ids[failing] || len(ids) != e.fake.listLen()-1 {
		t.Errorf("registre : %d matchs dont le match en échec=%v ; attendu tous sauf lui (%d)", len(ids), ids[failing], e.fake.listLen()-1)
	}
	t.Logf("match 500 durable : demandé aux cycles %v, fin rangée au cycle %d", failedSeen, endCycle)

	// Cycle suivant : delta d'une page, le match abandonné n'est plus demandé.
	out, reads := e.cycle()
	if len(reads) != 1 || out.HistoryPending || e.fake.statsCallsFor(failing) != 0 {
		t.Errorf("après la fin : lectures %v, issue %+v, requêtes du match abandonné %d ; attendu un delta d'une page qui ne le redemande pas",
			reads, out, e.fake.statsCallsFor(failing))
	}
	assertNoUnexpectedRequests(t, e)
}

// Un match que l'API déclare introuvable (404) n'est pas un échec : il ne bloque pas la fin d'historique,
// qui se range au cycle normal, sans l'attendre trois passes.
func TestConvergence_V2_Ordonnanceur_Infinite_MatchIntrouvable404NeBloquePas(t *testing.T) {
	e := newConvergenceEnv(t, 300)
	gone := e.fake.idAt(250)
	e.fake.setStatsStatus(gone, 404)

	var endCycle int
	for cycle := 1; cycle <= 6 && endCycle == 0; cycle++ {
		out, reads := e.cycle()
		complete, _ := e.historyState()
		if complete == out.HistoryPending {
			t.Fatalf("cycle %d : clé de fin rangée=%v mais HistoryPending=%v (lectures %v)", cycle, complete, out.HistoryPending, reads)
		}
		if !out.HistoryPending {
			endCycle = cycle
		}
	}
	// 300 matchs, budget 200 : 200 + 99 (le 404 n est jamais rangé), fin vue au 2e cycle.
	if endCycle != 2 {
		t.Fatalf("fin rangée au cycle %d, attendu le 2e (le 404 ne coûte aucune passe de plus)", endCycle)
	}
	ids := e.registryIDs()
	if ids[gone] || len(ids) != e.fake.listLen()-1 {
		t.Errorf("registre : %d matchs dont l'introuvable=%v ; attendu tous sauf lui (%d)", len(ids), ids[gone], e.fake.listLen()-1)
	}
	e.waitCadence(time.Duration(e.intervalH) * time.Hour)

	out, reads := e.cycle()
	if len(reads) != 1 || out.HistoryPending {
		t.Errorf("après la fin : lectures %v, issue %+v ; attendu un delta d'une page", reads, out)
	}
	assertNoUnexpectedRequests(t, e)
}
