package livesync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/historystate"
)

// historyWorld simule un historique Halo 5 (liste de match_ids, du plus récent au plus ancien), la
// base partagée (matchs rangés) et l'état de l'historique du joueur. Sa capture de page reproduit le
// contrat de halo5.CapturePageAt : connus sautés sans arrêt, HistoryEnd sur page vide ou incomplète.
type historyWorld struct {
	ids        []string
	stored     map[string]bool
	errAtStart map[int]error   // start -> erreur de capture de la page
	persistErr map[string]bool // match_id -> la persistance échoue

	complete   bool
	stateErr   error
	markFails  bool
	marked     int
	deltaCalls int
	deltaMax   int
	pageStarts []int
}

func newHistoryWorld(n int) *historyWorld {
	w := &historyWorld{stored: map[string]bool{}, errAtStart: map[int]error{}, persistErr: map[string]bool{}}
	for i := 0; i < n; i++ {
		w.ids = append(w.ids, fmt.Sprintf("m%04d", i))
	}
	return w
}

func (w *historyWorld) capturePage(_ context.Context, viewer canonical.PlayerIdentity, _ func(string) string,
	isKnown func(string) bool, start, pageSize int, seen map[string]struct{}, stats *halo5.CaptureStats,
) ([]*persist.MatchBatch, bool, error) {
	w.pageStarts = append(w.pageStarts, start)
	if err := w.errAtStart[start]; err != nil {
		return nil, false, err
	}
	if start >= len(w.ids) {
		stats.HistoryEnd = true
		return nil, false, nil
	}
	page := w.ids[start:min(start+pageSize, len(w.ids))]
	var batches []*persist.MatchBatch
	for _, id := range page {
		seen[id] = struct{}{}
		stats.MatchesSeen++
		if isKnown(id) {
			stats.MatchesSkipped++
			continue
		}
		batches = append(batches, batchFor(id, viewer))
		stats.MatchesCollected++
	}
	if len(page) < pageSize {
		stats.HistoryEnd = true
		return batches, false, nil
	}
	return batches, true, nil
}

func (w *historyWorld) persistAll(_ context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
	var done []*persist.MatchBatch
	var errs []string
	for _, b := range batches {
		id := b.Shared.Match.MatchID
		if w.persistErr[id] {
			errs = append(errs, "persist "+id+" échec")
			continue
		}
		w.stored[id] = true
		done = append(done, b)
	}
	return done, errs
}

func (w *historyWorld) deps() Deps {
	return Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
		Capture: func(_ context.Context, _ halo5.CaptureSource, _ canonical.PlayerIdentity, _ func(string) string,
			_ func(string) bool, opts halo5.CaptureOptions) ([]*persist.MatchBatch, halo5.CaptureStats, error) {
			w.deltaCalls++
			w.deltaMax = opts.MaxMatches
			return nil, halo5.CaptureStats{}, nil
		},
		Viewer: backfillViewer(),
		LoadKnown: func(context.Context) (map[string]bool, error) {
			known := make(map[string]bool, len(w.stored))
			for id := range w.stored {
				known[id] = true
			}
			return known, nil
		},
		PersistAll:  w.persistAll,
		CapturePage: w.capturePage,
		HistoryComplete: func(context.Context) (bool, error) {
			return w.complete, w.stateErr
		},
		MarkHistoryComplete: func(context.Context) bool {
			if w.markFails {
				return false
			}
			w.complete = true
			w.marked++
			return true
		},
	}
}

func (w *historyWorld) run(t *testing.T, opts domain.SyncOptions) domain.SyncResult {
	t.Helper()
	res, err := NewRunner(w.deps(), nil).RunDelta(context.Background(), opts)
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	return res
}

// Historique incomplet : la passe est bornée au budget (page entière rangée, donc au plus
// budget + taille de page - 1 matchs), n'appelle pas le delta, et ne range pas la fin.
func TestRunDelta_Rattrapage_PasseBorneeAuBudget(t *testing.T) {
	w := newHistoryWorld(400)
	res := w.run(t, domain.SyncOptions{MaxMatches: 30})

	const pageSize = 25
	if res.MatchesInserted < 30 || res.MatchesInserted > 30+pageSize-1 {
		t.Fatalf("insérés = %d, attendu entre 30 et %d (budget 30, pages de %d)", res.MatchesInserted, 30+pageSize-1, pageSize)
	}
	if res.MatchesInserted != 50 || len(w.pageStarts) != 2 {
		t.Errorf("insérés=%d pages=%v, attendu 50 matchs sur 2 pages (arrêt après la page qui atteint 30)", res.MatchesInserted, w.pageStarts)
	}
	if w.deltaCalls != 0 {
		t.Errorf("le delta a tourné %d fois pendant un rattrapage", w.deltaCalls)
	}
	if w.marked != 0 || !res.HistoryPending || res.HistoryExhausted {
		t.Errorf("fin rangée=%d pending=%v exhausted=%v, attendu 0/true/false", w.marked, res.HistoryPending, res.HistoryExhausted)
	}
	if len(res.InsertedMatchIDs) != res.MatchesInserted {
		t.Errorf("InsertedMatchIDs = %d, MatchesInserted = %d : les hooks doivent recevoir les matchs de la passe",
			len(res.InsertedMatchIDs), res.MatchesInserted)
	}
}

// Le budget par défaut est celui de historystate ; trois passes rangent 450 matchs, la fin est
// rangée à la troisième seulement, la quatrième est un delta ordinaire.
func TestRunDelta_Rattrapage_ReprendSurTroisPassesJusquALaFin(t *testing.T) {
	w := newHistoryWorld(450)
	budget := historystate.PassBudget()
	if budget != 200 {
		t.Fatalf("budget de passe = %d, le scénario suppose 200", budget)
	}

	want := []struct {
		inserted int
		marked   int
		pending  bool
	}{{200, 0, true}, {200, 0, true}, {50, 1, false}}
	for i, wnt := range want {
		res := w.run(t, domain.SyncOptions{})
		if res.MatchesInserted != wnt.inserted || w.marked != wnt.marked || res.HistoryPending != wnt.pending {
			t.Fatalf("passe %d : insérés=%d fin rangée=%d pending=%v, attendu %d/%d/%v",
				i+1, res.MatchesInserted, w.marked, res.HistoryPending, wnt.inserted, wnt.marked, wnt.pending)
		}
		if len(res.Errors) != 0 {
			t.Fatalf("passe %d : erreurs inattendues %v", i+1, res.Errors)
		}
		if res.Status() != "success" {
			t.Errorf("passe %d : statut %s", i+1, res.Status())
		}
	}
	if len(w.stored) != 450 {
		t.Fatalf("matchs rangés = %d, attendu 450", len(w.stored))
	}

	// Historique complet : delta ordinaire, la liste de l'API n'est plus parcourue.
	pagesBefore := len(w.pageStarts)
	res := w.run(t, domain.SyncOptions{MaxMatches: 25})
	if w.deltaCalls != 1 || w.deltaMax != 25 || len(w.pageStarts) != pagesBefore {
		t.Errorf("delta=%d (MaxMatches %d), pages parcourues en plus=%d, attendu 1 delta à 25 et aucune page",
			w.deltaCalls, w.deltaMax, len(w.pageStarts)-pagesBefore)
	}
	if res.HistoryPending {
		t.Error("historique complet : HistoryPending doit être faux")
	}
}

// Une page qui échoue (puis la fin n'est jamais atteinte) : pas de fin rangée, l'erreur est dans le
// résultat, HistoryPending reste vrai, et les pages d'avant sont rangées.
func TestRunDelta_Rattrapage_ErreurDePageNeRangePasLaFin(t *testing.T) {
	w := newHistoryWorld(60)
	w.errAtStart[25] = errors.New("GetPlayerMatches: 500")
	res := w.run(t, domain.SyncOptions{})

	if w.marked != 0 || !res.HistoryPending {
		t.Errorf("fin rangée=%d pending=%v, attendu 0/true", w.marked, res.HistoryPending)
	}
	if len(res.Errors) == 0 {
		t.Error("l'erreur de page doit être remontée dans le résultat")
	}
	if res.MatchesInserted != 25 || len(w.stored) != 25 {
		t.Errorf("insérés=%d rangés=%d, attendu 25 (la page 0 reste rangée)", res.MatchesInserted, len(w.stored))
	}

	// La passe suivante saute les 25 connus, reprend et range la fin.
	delete(w.errAtStart, 25)
	res = w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending || res.MatchesInserted != 35 {
		t.Errorf("passe de reprise : fin rangée=%d pending=%v insérés=%d, attendu 1/false/35", w.marked, res.HistoryPending, res.MatchesInserted)
	}
}

// Fin de liste vue mais un match échoue à la persistance : pas de fin rangée (il resterait un trou).
func TestRunDelta_Rattrapage_ErreurDePersistanceNeRangePasLaFin(t *testing.T) {
	w := newHistoryWorld(30)
	w.persistErr["m0007"] = true
	res := w.run(t, domain.SyncOptions{})

	if w.marked != 0 || !res.HistoryPending || res.Status() != "partial_success" {
		t.Errorf("fin rangée=%d pending=%v statut=%s, attendu 0/true/partial_success", w.marked, res.HistoryPending, res.Status())
	}
	if !res.HistoryExhausted {
		t.Error("la fin de liste a été vue : HistoryExhausted doit être vrai même si la fin n'est pas rangée")
	}
	delete(w.persistErr, "m0007")
	res = w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending || res.MatchesInserted != 1 {
		t.Errorf("passe de reprise : fin rangée=%d pending=%v insérés=%d, attendu 1/false/1", w.marked, res.HistoryPending, res.MatchesInserted)
	}
}

// Un match rangé sans timeline ni carnage n'est pas une erreur : la fin se range quand même. Ici le
// persisteur ne voit que des batches ; le compteur EventsFailed vient de la capture.
func TestRunDelta_Rattrapage_TimelineManquanteNEstPasUneErreur(t *testing.T) {
	w := newHistoryWorld(10)
	deps := w.deps()
	deps.CapturePage = func(ctx context.Context, v canonical.PlayerIdentity, r func(string) string, k func(string) bool,
		start, size int, seen map[string]struct{}, stats *halo5.CaptureStats) ([]*persist.MatchBatch, bool, error) {
		batches, more, err := w.capturePage(ctx, v, r, k, start, size, seen, stats)
		stats.EventsFailed += len(batches)
		stats.CarnageFailed += len(batches)
		return batches, more, err
	}
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if w.marked != 1 || res.HistoryPending || len(res.Errors) != 0 {
		t.Errorf("fin rangée=%d pending=%v erreurs=%v, attendu 1/false/aucune", w.marked, res.HistoryPending, res.Errors)
	}
}

// Une page pleine qui reprend des matchs déjà vus (liste qui ignore `start`) n'est pas une fin.
func TestRunDelta_Rattrapage_AntiBoucleNEstPasUneFin(t *testing.T) {
	w := newHistoryWorld(50)
	deps := w.deps()
	deps.CapturePage = func(ctx context.Context, v canonical.PlayerIdentity, r func(string) string, k func(string) bool,
		start, size int, seen map[string]struct{}, stats *halo5.CaptureStats) ([]*persist.MatchBatch, bool, error) {
		if start > 0 { // l'API rend toujours la première page : arrêt défensif, sans HistoryEnd
			return nil, false, nil
		}
		return w.capturePage(ctx, v, r, k, 0, size, seen, stats)
	}
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if w.marked != 0 || !res.HistoryPending {
		t.Errorf("fin rangée=%d pending=%v, attendu 0/true", w.marked, res.HistoryPending)
	}
}

// Fin vue sans erreur mais écriture de la clé impossible (bail tenu) : HistoryPending reste vrai.
func TestRunDelta_Rattrapage_EcritureDeLaFinEchouee(t *testing.T) {
	w := newHistoryWorld(10)
	w.markFails = true
	res := w.run(t, domain.SyncOptions{})
	if !res.HistoryPending || w.complete {
		t.Errorf("pending=%v complete=%v, attendu true/false", res.HistoryPending, w.complete)
	}
}

// Historique complet : delta inchangé (MaxMatches transmis, aucune page parcourue), rien en attente.
func TestRunDelta_HistoriqueCompletResteUnDelta(t *testing.T) {
	w := newHistoryWorld(100)
	w.complete = true
	res := w.run(t, domain.SyncOptions{MaxMatches: 40})
	if w.deltaCalls != 1 || w.deltaMax != 40 || len(w.pageStarts) != 0 || res.HistoryPending || w.marked != 0 {
		t.Errorf("delta=%d max=%d pages=%v pending=%v marked=%d", w.deltaCalls, w.deltaMax, w.pageStarts, res.HistoryPending, w.marked)
	}
}

// Base joueur illisible : delta ordinaire (jamais de rattrapage à l'aveugle), mais en attente.
func TestRunDelta_EtatIllisibleFaitUnDeltaEnAttente(t *testing.T) {
	w := newHistoryWorld(100)
	w.stateErr = errors.New("open player DB")
	res := w.run(t, domain.SyncOptions{})
	if w.deltaCalls != 1 || len(w.pageStarts) != 0 || !res.HistoryPending {
		t.Errorf("delta=%d pages=%v pending=%v, attendu 1/aucune/true", w.deltaCalls, w.pageStarts, res.HistoryPending)
	}
}

// Source indisponible avec un historique incomplet : le cycle échoue mais reste en attente.
func TestRunDelta_SourceIndisponibleRestePending(t *testing.T) {
	w := newHistoryWorld(100)
	deps := w.deps()
	deps.NewSource = func(context.Context) (halo5.CaptureSource, error) { return nil, errors.New("re-auth") }
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil || !res.HistoryPending || res.Status() != "failure" {
		t.Errorf("err=%v pending=%v statut=%s, attendu nil/true/failure", err, res.HistoryPending, res.Status())
	}
}

// Known-set illisible : pas de rattrapage (il compterait des matchs déjà rangés), delta sans arrêt
// sur le connu comme avant, historique encore en attente.
func TestRunDelta_KnownSetIllisibleNeLancePasLeRattrapage(t *testing.T) {
	w := newHistoryWorld(100)
	deps := w.deps()
	deps.LoadKnown = func(context.Context) (map[string]bool, error) { return nil, errors.New("DB KO") }
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if w.deltaCalls != 1 || len(w.pageStarts) != 0 || !res.HistoryPending || w.marked != 0 {
		t.Errorf("delta=%d pages=%v pending=%v marked=%d", w.deltaCalls, w.pageStarts, res.HistoryPending, w.marked)
	}
}

// Sans hooks d'historique (runner nu) : comportement d'avant, rien en attente.
func TestRunDelta_SansHooksHistoriqueResteUnDelta(t *testing.T) {
	w := newHistoryWorld(10)
	deps := w.deps()
	deps.HistoryComplete, deps.MarkHistoryComplete = nil, nil
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil || w.deltaCalls != 1 || res.HistoryPending {
		t.Errorf("err=%v delta=%d pending=%v", err, w.deltaCalls, res.HistoryPending)
	}
}

// RunBackfill sans budget (la commande h5-backfill) parcourt tout l'historique d'un trait ; avec un
// budget il s'arrête et ne déclare pas la fin.
func TestRunBackfill_BudgetOptionnel(t *testing.T) {
	run := func(budget int) (BackfillStats, *historyWorld) {
		w := newHistoryWorld(450)
		deps := BackfillDeps{
			CapturePage: w.capturePage,
			Viewer:      backfillViewer(),
			ResolveXUID: idResolve,
			LoadKnown:   func(context.Context) (map[string]bool, error) { return map[string]bool{}, nil },
			PersistPage: w.persistAll,
			Budget:      budget,
		}
		stats, err := RunBackfill(context.Background(), deps, 0, nil)
		if err != nil {
			t.Fatalf("RunBackfill: %v", err)
		}
		return stats, w
	}

	stats, _ := run(0)
	if stats.Inserted != 450 || !stats.HistoryEnd || stats.BudgetReached || stats.Pages != 19 {
		t.Errorf("sans budget : %+v, attendu 450 insérés, fin vue, 19 pages", stats)
	}
	stats, _ = run(100)
	if stats.Inserted != 100 || stats.HistoryEnd || !stats.BudgetReached || stats.Pages != 4 {
		t.Errorf("budget 100 : %+v, attendu 100 insérés, fin non vue, budget atteint, 4 pages", stats)
	}
}

// Fin de liste et budget sur la même page : la fin gagne (page incomplète). Budget atteint pile sur
// une page pleine : la fin n'est pas encore vue, la passe suivante la verra sans rien insérer.
func TestRunBackfill_BudgetEtFin(t *testing.T) {
	run := func(n, budget int) BackfillStats {
		w := newHistoryWorld(n)
		deps := BackfillDeps{
			CapturePage: w.capturePage,
			Viewer:      backfillViewer(),
			LoadKnown:   func(context.Context) (map[string]bool, error) { return map[string]bool{}, nil },
			PersistPage: w.persistAll,
			Budget:      budget,
		}
		stats, err := RunBackfill(context.Background(), deps, 25, nil)
		if err != nil {
			t.Fatalf("RunBackfill: %v", err)
		}
		return stats
	}
	if s := run(40, 40); s.Inserted != 40 || !s.HistoryEnd || s.BudgetReached {
		t.Errorf("40 matchs, budget 40 : %+v, attendu fin vue (page incomplète) sans budget atteint", s)
	}
	if s := run(50, 50); s.Inserted != 50 || s.HistoryEnd || !s.BudgetReached {
		t.Errorf("50 matchs, budget 50 : %+v, attendu budget atteint, fin pas encore vue", s)
	}
}
