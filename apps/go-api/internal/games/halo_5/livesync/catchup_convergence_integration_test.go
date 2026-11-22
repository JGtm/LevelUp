//go:build integration

// catchup_convergence_integration_test.go : convergence de bout en bout du rattrapage de l'historique
// Halo 5 par Runner.RunDelta (le chemin que l'ordonnanceur et le suivi en direct appellent) sur un
// faux historique de 1 000 matchs : cycles successifs, budget par passe, marque-page, matchs neufs en
// haut de la liste, fin rangée, puis delta d'une page. L'état de l'historique (clé de fin, marque-page,
// compte d'échecs) vit dans une VRAIE base joueur DuckDB temporaire ; seules la capture des pages et la
// persistance des matchs sont simulées (la base partagée Halo 5 n'a pas de rôle dans le contrat).
package livesync

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/historystate"
)

const convergenceH5Total = 1000

// convergenceH5 : le monde simulé, l'état réel du joueur et les compteurs de lecture de pages.
type convergenceH5 struct {
	t          *testing.T
	w          *historyWorld
	h          playerHistory
	runner     *Runner
	deltaPages int // pages lues par le delta du dessus (Capture)
	nextFresh  int
}

func newConvergenceH5(t *testing.T, total int) *convergenceH5 {
	t.Helper()
	h, _ := tempPlayerHistory(t)
	c := &convergenceH5{t: t, w: newHistoryWorld(total), h: h}
	deps := c.w.deps()
	deps.History = h
	deps.Capture = c.countingCapture
	c.runner = NewRunner(deps, nil)
	return c
}

// countingCapture est le delta simulé du monde (historyWorld.capture) avec le compte des pages lues :
// une page par tranche de 25 entrées vues, plus la page vide qui constate la fin quand elle tombe pile.
func (c *convergenceH5) countingCapture(ctx context.Context, src halo5.CaptureSource, viewer canonical.PlayerIdentity,
	resolve func(string) string, isKnown func(string) bool, opts halo5.CaptureOptions) ([]*persist.MatchBatch, halo5.CaptureStats, error) {
	batches, stats, err := c.w.capture(ctx, src, viewer, resolve, isKnown, opts)
	pages := (stats.MatchesSeen + backfillDefaultPageSize - 1) / backfillDefaultPageSize
	if stats.HistoryEnd && stats.MatchesSeen%backfillDefaultPageSize == 0 {
		pages++
	}
	c.deltaPages += pages
	return batches, stats, err
}

// addOnTop place n matchs neufs, plus récents que tous les autres, en haut de la liste de l'API.
func (c *convergenceH5) addOnTop(n int) []string {
	fresh := make([]string, 0, n)
	for i := 0; i < n; i++ {
		fresh = append([]string{fmt.Sprintf("t%04d", c.nextFresh)}, fresh...)
		c.nextFresh++
	}
	c.w.ids = append(append([]string{}, fresh...), c.w.ids...)
	return fresh
}

// addMixedOnTop place en haut de la liste des entrées neuves, plus récentes que toutes les autres, dont
// warzone parties Warzone (écartées : vues, jamais rangées) entremêlées à arena matchs Arena.
func (c *convergenceH5) addMixedOnTop(arena, warzone int) []string {
	total := arena + warzone
	fresh := make([]string, 0, total)
	for i := 0; i < total; i++ {
		if warzone > 0 && (i%3 == 1 || arena == 0) {
			warzone--
			fresh = append(fresh, fmt.Sprintf("w9%03d", c.nextFresh))
		} else {
			arena--
			fresh = append(fresh, fmt.Sprintf("t%04d", c.nextFresh))
		}
		c.nextFresh++
	}
	c.w.prependIDs(fresh...)
	return fresh
}

// pass exécute une passe et rend son résultat, avec les pages lues (delta du dessus et reprise).
func (c *convergenceH5) pass(opts domain.SyncOptions) (res domain.SyncResult, pages int, resumeStarts []int) {
	c.t.Helper()
	c.w.pageStarts, c.w.listStarts, c.deltaPages = nil, nil, 0
	res, err := c.runner.RunDelta(context.Background(), opts)
	if err != nil {
		c.t.Fatalf("RunDelta: %v", err)
	}
	return res, c.deltaPages + len(c.w.listStarts) + len(c.w.pageStarts), append([]int(nil), c.w.pageStarts...)
}

// Un historique de 1 000 matchs, des matchs neufs qui arrivent en haut entre deux cycles : les cycles
// successifs de RunDelta rattrapent 200 matchs au plus (page entière en plus), reprennent au
// marque-page sans rebalayer, prennent les nouveaux du dessus, rangent la fin, puis redeviennent des
// deltas d'une page.
func TestConvergence_Halo5_RunDelta_RattrapeParBudgetMarquePageFinPuisDelta(t *testing.T) {
	c := newConvergenceH5(t, convergenceH5Total)
	ctx := context.Background()
	budget := historystate.PassBudget()
	const pageSize = backfillDefaultPageSize

	// Cycle n° -> matchs neufs apparus en haut AVANT ce cycle.
	arrivals := map[int]int{2: 3, 4: 5}
	var (
		endCycle     int
		totalReads   int
		readsByCycle []int
	)
	for cycle := 1; cycle <= 12 && endCycle == 0; cycle++ {
		fresh := c.addOnTop(arrivals[cycle])
		cursorBefore, hadCursor := c.h.LoadCursor(ctx)
		storedBefore := len(c.w.stored)
		missing := len(c.w.ids) - storedBefore

		res, pages, resumeStarts := c.pass(domain.SyncOptions{})
		totalReads += pages
		readsByCycle = append(readsByCycle, pages)

		// Budget : au moins min(budget, reste) matchs, au plus le budget plus la page entière rangée.
		minIns := min(budget, missing)
		if res.MatchesInserted < minIns || res.MatchesInserted > budget+pageSize-1 {
			t.Fatalf("cycle %d : %d insérés, attendu entre %d et %d (reste %d)", cycle, res.MatchesInserted, minIns, budget+pageSize-1, missing)
		}
		if got := len(c.w.stored); got != storedBefore+res.MatchesInserted {
			t.Fatalf("cycle %d : %d matchs rangés, attendu %d", cycle, got, storedBefore+res.MatchesInserted)
		}
		// Les matchs neufs du dessus sont pris dès ce cycle.
		for _, id := range fresh {
			if !c.w.stored[id] {
				t.Fatalf("cycle %d : le match neuf du dessus %s n'a pas été pris", cycle, id)
			}
		}
		// Pages lues bornées : delta du dessus (<= 2 pages), reprise (recouvrement + budget/25 pages + 1).
		if bound := 2 + 1 + budget/pageSize + 2; pages > bound {
			t.Fatalf("cycle %d : %d pages lues, borne %d", cycle, pages, bound)
		}
		// Reprise au marque-page, jamais depuis le début : la première page de la reprise est la page de
		// recouvrement (décalage du marque-page + neufs - une page).
		if hadCursor {
			wantStart := max(0, cursorBefore.Offset+len(fresh)-pageSize)
			if len(resumeStarts) == 0 || resumeStarts[0] != wantStart {
				t.Fatalf("cycle %d : la reprise commence à %v, attendu la page de recouvrement %d (marque-page %+v, %d neufs)",
					cycle, resumeStarts, wantStart, cursorBefore, len(fresh))
			}
			if wantStart == 0 {
				t.Fatalf("cycle %d : le scénario ne teste plus la reprise au marque-page (départ à 0)", cycle)
			}
		}

		complete, err := c.h.Complete(ctx)
		if err != nil {
			t.Fatalf("cycle %d : lecture de la clé de fin : %v", cycle, err)
		}
		if complete == res.HistoryPending {
			t.Fatalf("cycle %d : clé de fin rangée=%v mais HistoryPending=%v (doivent être opposés)", cycle, complete, res.HistoryPending)
		}
		if complete {
			endCycle = cycle
			if _, ok := c.h.LoadCursor(ctx); ok {
				t.Errorf("cycle %d : le marque-page doit être effacé quand la fin est rangée", cycle)
			}
			continue
		}
		// Cycle intermédiaire : un marque-page est rangé, sur le dernier match traité.
		cur, ok := c.h.LoadCursor(ctx)
		if !ok {
			t.Fatalf("cycle %d : aucun marque-page rangé alors que la fin n'est pas vue", cycle)
		}
		if cur.Offset != len(c.w.stored) || cur.LastMatchID != c.w.ids[cur.Offset-1] {
			t.Fatalf("cycle %d : marque-page %+v, attendu {%d %s}", cycle, cur, len(c.w.stored), c.w.ids[len(c.w.stored)-1])
		}
	}
	if endCycle == 0 || endCycle > 7 {
		t.Fatalf("fin d'historique rangée au cycle %d, attendu entre le 5e et le 7e (1 000 matchs, budget %d)", endCycle, budget)
	}
	if got, want := len(c.w.stored), len(c.w.ids); got != want {
		t.Fatalf("à la fin, %d matchs rangés sur %d", got, want)
	}
	// Pas de rebalayage : la lecture totale reste proche de la taille de la liste, pas quadratique.
	if bound := len(c.w.ids)/pageSize + endCycle*4; totalReads > bound {
		t.Errorf("%d pages lues au total, borne %d : la reprise rebalaye l'historique", totalReads, bound)
	}
	t.Logf("Halo 5 : fin rangée au cycle %d, pages lues par cycle %v (total %d)", endCycle, readsByCycle, totalReads)

	// Cycle suivant : un delta d'une page, aucun rattrapage, rien d'en attente.
	deltaBefore := c.w.deltaCalls
	res, pages, resumeStarts := c.pass(domain.SyncOptions{MaxMatches: 25})
	if c.w.deltaCalls != deltaBefore+1 || len(resumeStarts) != 0 || pages != 1 {
		t.Errorf("après la fin : %d delta, reprise %v, %d pages lues ; attendu 1 delta, aucune reprise, 1 page", c.w.deltaCalls-deltaBefore, resumeStarts, pages)
	}
	if res.MatchesInserted != 0 || res.HistoryPending {
		t.Errorf("après la fin : inséré=%d pending=%v, attendu 0 / faux", res.MatchesInserted, res.HistoryPending)
	}
	// Un match neuf après la fin : pris par le delta, une page.
	fresh := c.addOnTop(1)
	res, pages, resumeStarts = c.pass(domain.SyncOptions{MaxMatches: 25})
	if res.MatchesInserted != 1 || !c.w.stored[fresh[0]] || pages != 1 || len(resumeStarts) != 0 {
		t.Errorf("delta d'un match neuf : %d insérés, pris=%v, %d pages, reprise %v ; attendu 1, vrai, 1, aucune",
			res.MatchesInserted, c.w.stored[fresh[0]], pages, resumeStarts)
	}
	if complete, err := c.h.Complete(ctx); err != nil || !complete {
		t.Errorf("la clé de fin doit rester rangée après un delta : complet=%v err=%v", complete, err)
	}
}

// Une page de l'API qui échoue toujours ne garde pas le joueur en rattrapage à vie : elle est
// abandonnée à la troisième passe qui la rencontre, le reste de l'historique est rattrapé, la fin est
// rangée, puis le joueur redevient un delta.
func TestConvergence_Halo5_RunDelta_PageEnEchecDurableAbandonneeLaFinSeRange(t *testing.T) {
	c := newConvergenceH5(t, convergenceH5Total)
	ctx := context.Background()
	const failingStart = 500 // page des matchs 500 à 524, jamais lisible
	c.w.errAtStart[failingStart] = fmt.Errorf("GetPlayerMatches: 500 interne")

	var endCycle int
	var failedCycles []int
	for cycle := 1; cycle <= 12 && endCycle == 0; cycle++ {
		res, _, _ := c.pass(domain.SyncOptions{})
		complete, err := c.h.Complete(ctx)
		if err != nil {
			t.Fatalf("cycle %d : lecture de la clé : %v", cycle, err)
		}
		if complete == res.HistoryPending {
			t.Fatalf("cycle %d : clé de fin rangée=%v mais HistoryPending=%v", cycle, complete, res.HistoryPending)
		}
		if len(res.Errors) > 0 && !complete {
			failedCycles = append(failedCycles, cycle)
			// Le compte d'échecs de la page survit entre les passes (base joueur réelle).
			cur, ok := c.h.LoadCursor(ctx)
			if !ok {
				t.Fatalf("cycle %d : page en échec mais aucun marque-page rangé avant elle", cycle)
			}
			if got := c.h.LoadStreak(ctx, cur.LastMatchID); got.Attempts != len(failedCycles) {
				t.Fatalf("cycle %d : %d échecs comptés pour la page, attendu %d", cycle, got.Attempts, len(failedCycles))
			}
		}
		if complete {
			endCycle = cycle
		}
	}
	if endCycle == 0 {
		t.Fatalf("fin non rangée en 12 cycles malgré l'abandon (rangés %d / %d)", len(c.w.stored), len(c.w.ids))
	}
	if len(failedCycles) != 2 {
		t.Errorf("cycles en échec = %v, attendu 2 passes en échec puis l'abandon à la 3e", failedCycles)
	}
	// Seule la page abandonnée manque.
	for i, id := range c.w.ids {
		inFailing := i >= failingStart && i < failingStart+backfillDefaultPageSize
		if c.w.stored[id] == inFailing {
			t.Fatalf("match %d (%s) : rangé=%v, la page abandonnée (500-524) est seule à manquer", i, id, c.w.stored[id])
		}
	}
	if _, ok := c.h.LoadCursor(ctx); ok {
		t.Error("le marque-page doit être effacé une fois la fin rangée")
	}
	t.Logf("Halo 5 (page durable en échec) : fin rangée au cycle %d, passes en échec %v", endCycle, failedCycles)

	// Cycle suivant : delta d'une page.
	deltaBefore := c.w.deltaCalls
	res, pages, resumeStarts := c.pass(domain.SyncOptions{MaxMatches: 25})
	if c.w.deltaCalls != deltaBefore+1 || pages != 1 || len(resumeStarts) != 0 || res.HistoryPending || res.MatchesInserted != 0 {
		t.Errorf("après la fin : delta=%d pages=%d reprise=%v pending=%v inséré=%d ; attendu un delta d'une page sans attente",
			c.w.deltaCalls-deltaBefore, pages, resumeStarts, res.HistoryPending, res.MatchesInserted)
	}
}

// Quarante parties Warzone restent en tête de liste pendant tout le rattrapage (écartées : jamais
// rangées, jamais connues), et des entrées neuves, Arena et Warzone mêlées, arrivent entre deux cycles :
// chaque cycle reprend exactement au marque-page (décalage + entrées arrivées - une page), aucun ne
// repart de 0, la fin se range. Avant l'ancre, la règle « jusqu'au premier connu » recomptait les
// quarante Warzone à chaque passe : au-delà d'une page, la reprise repartait de 0.
func TestConvergence_Halo5_RunDelta_WarzoneEnTeteNeFaitJamaisRepartirDe0(t *testing.T) {
	c := newConvergenceH5(t, convergenceH5Total)
	ctx := context.Background()
	c.w.prependIDs(idsOf("w", 40)...)
	const pageSize = backfillDefaultPageSize
	budget := historystate.PassBudget()
	missedBefore := observability.LoadCounter(metricCursorMissed)

	arrivals := map[int][2]int{2: {2, 1}, 4: {3, 2}} // cycle -> {Arena, Warzone} arrivés avant lui
	var endCycle, totalReads int
	var readsByCycle []int
	for cycle := 1; cycle <= 12 && endCycle == 0; cycle++ {
		arrived := 0
		if a, ok := arrivals[cycle]; ok {
			arrived = len(c.addMixedOnTop(a[0], a[1]))
		}
		cursorBefore, hadCursor := c.h.LoadCursor(ctx)

		res, pages, resumeStarts := c.pass(domain.SyncOptions{})
		totalReads += pages
		readsByCycle = append(readsByCycle, pages)
		if res.MatchesInserted > budget+pageSize-1 {
			t.Errorf("cycle %d : %d insérés, au plus %d attendus", cycle, res.MatchesInserted, budget+pageSize-1)
		}
		if hadCursor {
			want := max(0, cursorBefore.Offset+arrived-pageSize)
			if len(resumeStarts) == 0 || resumeStarts[0] != want || want == 0 {
				t.Errorf("cycle %d : la reprise commence à %v, attendu la page de recouvrement %d (marque-page %d + %d arrivées)",
					cycle, resumeStarts, want, cursorBefore.Offset, arrived)
			}
		}
		complete, err := c.h.Complete(ctx)
		if err != nil {
			t.Fatalf("cycle %d : lecture de la clé de fin : %v", cycle, err)
		}
		if complete {
			endCycle = cycle
		} else if cur, ok := c.h.LoadCursor(ctx); !ok || cur.LastMatchID != c.w.ids[cur.Offset-1] || cur.HeadID != c.w.ids[0] {
			t.Fatalf("cycle %d : marque-page %+v (rangé=%v), attendu une position cohérente avec la liste et l'ancre %s", cycle, cur, ok, c.w.ids[0])
		}
	}
	if endCycle == 0 {
		t.Fatalf("fin non rangée en 12 cycles (rangés %d)", len(c.w.stored))
	}
	for _, id := range c.w.ids {
		if c.w.stored[id] == c.w.excluded[id] {
			t.Fatalf("match %s : rangé=%v, écarté=%v (un match écarté n'est jamais rangé, tout autre l'est)", id, c.w.stored[id], c.w.excluded[id])
		}
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 0 {
		t.Errorf("%d reprises depuis 0 (compteur h5_history_cursor_missed_total), attendu 0", got)
	}
	t.Logf("Halo 5 (40 Warzone en tête) : fin rangée au cycle %d, pages lues par cycle %v (total %d)", endCycle, readsByCycle, totalReads)
}
