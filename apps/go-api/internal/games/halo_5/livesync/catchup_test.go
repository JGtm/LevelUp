package livesync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/historystate"
)

// historyWorld simule un historique Halo 5 (liste de match_ids, du plus récent au plus ancien), la
// base partagée (matchs rangés) et l'état du rattrapage du joueur (HistoryStore). Ses captures
// reproduisent le contrat de halo5 : la page connus sautés sans arrêt, HistoryEnd sur page vide ou
// incomplète, LastMatchID ; le delta s'arrête après domain.DeltaKnownRun connus d'affilée (CollectRecentMatches).
type historyWorld struct {
	ids        []string
	stored     map[string]bool
	errAtStart map[int]error   // décalage d'une page de la reprise -> erreur de capture
	persistErr map[string]bool // match_id -> la persistance échoue
	excluded   map[string]bool // match_id écartés à la collecte (Warzone, campagne) : vus, jamais rangés ni connus

	// état du joueur
	complete  bool
	stateErr  error
	markFails bool
	marked    int
	cursor    *historystate.Cursor
	streak    historystate.Streak
	saves     int

	// observations
	deltaCalls int
	deltaMax   int
	pageStarts []int
	listStarts []int // pages relues par la lecture légère de la liste (ancre)
}

func newHistoryWorld(n int) *historyWorld {
	w := &historyWorld{stored: map[string]bool{}, errAtStart: map[int]error{}, persistErr: map[string]bool{}, excluded: map[string]bool{}}
	for i := 0; i < n; i++ {
		w.ids = append(w.ids, fmt.Sprintf("m%04d", i))
	}
	return w
}

// prepend ajoute k matchs neufs en haut de la liste.
func (w *historyWorld) prepend(k int) {
	fresh := make([]string, 0, k+len(w.ids))
	for i := k - 1; i >= 0; i-- {
		fresh = append(fresh, fmt.Sprintf("n%04d", i))
	}
	w.ids = append(fresh, w.ids...)
}

func (w *historyWorld) page(start, size int) []string {
	if start >= len(w.ids) {
		return nil
	}
	return w.ids[start:min(start+size, len(w.ids))]
}

func (w *historyWorld) capturePage(_ context.Context, viewer canonical.PlayerIdentity, _ func(string) string,
	isKnown func(string) bool, start, pageSize int, seen map[string]struct{}, stats *halo5.CaptureStats,
) ([]*persist.MatchBatch, bool, error) {
	w.pageStarts = append(w.pageStarts, start)
	if err := w.errAtStart[start]; err != nil {
		return nil, false, err
	}
	page := w.page(start, pageSize)
	if len(page) == 0 {
		stats.HistoryEnd = true
		return nil, false, nil
	}
	var batches []*persist.MatchBatch
	for i, id := range page {
		seen[id] = struct{}{}
		stats.LastMatchID = id
		stats.MatchesSeen++
		noteEntry(stats, 0, start+i, id)
		if isKnown(id) {
			stats.MatchesSkipped++
			continue
		}
		if w.excluded[id] {
			stats.ExcludedWarzone++
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

// capture simule CollectRecentMatches : delta-stop après domain.DeltaKnownRun connus d'affilée (les entrées
// écartées ne comptent ni ne remettent à zéro), plafond MaxMatches.
func (w *historyWorld) capture(_ context.Context, _ halo5.CaptureSource, viewer canonical.PlayerIdentity, _ func(string) string,
	isKnown func(string) bool, opts halo5.CaptureOptions) ([]*persist.MatchBatch, halo5.CaptureStats, error) {
	w.deltaCalls++
	w.deltaMax = opts.MaxMatches
	var (
		batches []*persist.MatchBatch
		stats   halo5.CaptureStats
	)
	for start := 0; ; start += backfillDefaultPageSize {
		page := w.page(start, backfillDefaultPageSize)
		if len(page) == 0 {
			stats.HistoryEnd = true
			return batches, stats, nil
		}
		for i, id := range page {
			stats.MatchesSeen++
			noteEntry(&stats, opts.TopLimit, start+i, id)
			if isKnown(id) {
				stats.MatchesSkipped++
				if stats.KnownRun++; stats.KnownRun >= domain.DeltaKnownRun {
					stats.StoppedOnKnown = true
					return batches, stats, nil
				}
				continue
			}
			stats.TopSpan = stats.MatchesSeen
			if w.excluded[id] {
				stats.ExcludedWarzone++
				continue
			}
			stats.KnownRun = 0
			batches = append(batches, batchFor(id, viewer))
			stats.MatchesCollected++
			if opts.MaxMatches > 0 && stats.MatchesCollected >= opts.MaxMatches {
				return batches, stats, nil
			}
		}
		if len(page) < backfillDefaultPageSize {
			stats.HistoryEnd = true
			return batches, stats, nil
		}
	}
}

// noteEntry reproduit halo5.CaptureStats.HeadID / TopIDs de la capture réelle.
func noteEntry(stats *halo5.CaptureStats, topLimit, position int, id string) {
	if position == 0 {
		stats.HeadID = id
	}
	if position == len(stats.TopIDs) && len(stats.TopIDs) < topLimit {
		stats.TopIDs = append(stats.TopIDs, id)
	}
}

// listIDs simule halo5.ListMatchIDs : une page d'identifiants de la liste, sans rien capturer.
func (w *historyWorld) listIDs(_ context.Context, _ canonical.PlayerIdentity, start, pageSize int) ([]string, error) {
	w.listStarts = append(w.listStarts, start)
	return append([]string(nil), w.page(start, pageSize)...), nil
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

// HistoryStore.
func (w *historyWorld) Complete(context.Context) (bool, error) { return w.complete, w.stateErr }

func (w *historyWorld) MarkComplete(context.Context) bool {
	if w.markFails {
		return false
	}
	w.complete = true
	w.marked++
	w.cursor, w.streak = nil, historystate.Streak{}
	return true
}

func (w *historyWorld) LoadCursor(context.Context) (historystate.Cursor, bool) {
	if w.cursor == nil {
		return historystate.Cursor{}, false
	}
	return *w.cursor, true
}

func (w *historyWorld) SaveCursor(_ context.Context, c historystate.Cursor) bool {
	w.cursor = &c
	w.saves++
	return true
}

func (w *historyWorld) LoadStreak(_ context.Context, subject string) historystate.Streak {
	if w.streak.Subject != subject {
		return historystate.Streak{Subject: subject}
	}
	return w.streak
}

func (w *historyWorld) SaveStreak(_ context.Context, s historystate.Streak) bool {
	w.streak = s
	return true
}

func (w *historyWorld) ClearStreak(context.Context) bool {
	w.streak = historystate.Streak{}
	return true
}

func (w *historyWorld) deps() Deps {
	return Deps{
		NewSource: func(context.Context) (halo5.CaptureSource, error) { return nil, nil },
		Capture:   w.capture,
		Viewer:    backfillViewer(),
		LoadKnown: func(context.Context) (map[string]bool, error) {
			known := make(map[string]bool, len(w.stored))
			for id := range w.stored {
				known[id] = true
			}
			return known, nil
		},
		PersistAll:  w.persistAll,
		CapturePage: w.capturePage,
		ListIDs:     w.listIDs,
		History:     w,
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

// Historique incomplet sans marque-page : la passe est bornée au budget (page entière rangée, donc au
// plus budget + taille de page - 1 matchs), n'appelle pas le delta, range le marque-page et pas la fin.
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
		t.Errorf("le delta a tourné %d fois sans marque-page", w.deltaCalls)
	}
	if w.marked != 0 || !res.HistoryPending {
		t.Errorf("fin rangée=%d pending=%v, attendu 0/true", w.marked, res.HistoryPending)
	}
	if w.cursor == nil || w.cursor.Offset != 50 || w.cursor.LastMatchID != "m0049" {
		t.Errorf("marque-page = %+v, attendu {50 m0049}", w.cursor)
	}
	if len(res.InsertedMatchIDs) != res.MatchesInserted {
		t.Errorf("InsertedMatchIDs = %d, MatchesInserted = %d : les hooks doivent recevoir les matchs de la passe",
			len(res.InsertedMatchIDs), res.MatchesInserted)
	}
}

// Trois passes rangent 450 matchs avec le budget par défaut (200) ; la deuxième reprend au marque-page
// (page de recouvrement 175, pas la page 0), la fin n'est rangée qu'à la troisième, la quatrième est un delta.
func TestRunDelta_Rattrapage_ReprendSurTroisPassesJusquALaFin(t *testing.T) {
	w := newHistoryWorld(450)
	if historystate.PassBudget() != 200 {
		t.Fatalf("budget de passe = %d, le scénario suppose 200", historystate.PassBudget())
	}

	want := []struct {
		inserted int
		marked   int
		pending  bool
	}{{200, 0, true}, {200, 0, true}, {50, 1, false}}
	for i, wnt := range want {
		w.pageStarts = nil
		res := w.run(t, domain.SyncOptions{})
		if res.MatchesInserted != wnt.inserted || w.marked != wnt.marked || res.HistoryPending != wnt.pending {
			t.Fatalf("passe %d : insérés=%d fin rangée=%d pending=%v, attendu %d/%d/%v",
				i+1, res.MatchesInserted, w.marked, res.HistoryPending, wnt.inserted, wnt.marked, wnt.pending)
		}
		if len(res.Errors) != 0 || res.Status() != "success" {
			t.Fatalf("passe %d : erreurs %v statut %s", i+1, res.Errors, res.Status())
		}
		if i > 0 && w.pageStarts[0] != 175+(i-1)*200 {
			t.Errorf("passe %d : première page lue = %d, attendu la page de recouvrement %d", i+1, w.pageStarts[0], 175+(i-1)*200)
		}
	}
	if len(w.stored) != 450 {
		t.Fatalf("matchs rangés = %d, attendu 450", len(w.stored))
	}
	if w.cursor != nil {
		t.Errorf("la fin rangée efface le marque-page : %+v", w.cursor)
	}

	// Historique complet : delta ordinaire, la liste de l'API n'est plus parcourue.
	w.pageStarts, w.deltaCalls = nil, 0
	res := w.run(t, domain.SyncOptions{MaxMatches: 25})
	if w.deltaCalls != 1 || w.deltaMax != 25 || len(w.pageStarts) != 0 || res.HistoryPending {
		t.Errorf("delta=%d (MaxMatches %d), pages de reprise=%v pending=%v, attendu 1 delta à 25 sans page", w.deltaCalls, w.deltaMax, w.pageStarts, res.HistoryPending)
	}
}

// Des matchs neufs arrivent entre deux passes : le delta du dessus les prend en premier, compte N, et
// la reprise commence à marque-page + N moins une page de recouvrement.
func TestRunDelta_Rattrapage_ReprendAuMarquePageAvecDesMatchsNeufs(t *testing.T) {
	w := newHistoryWorld(450)
	w.run(t, domain.SyncOptions{}) // 200 matchs, marque-page {200 m0199}
	if w.cursor == nil || w.cursor.Offset != 200 {
		t.Fatalf("marque-page après la première passe = %+v, attendu {200 ...}", w.cursor)
	}

	const fresh = 7
	w.prepend(fresh)
	w.pageStarts = nil
	res := w.run(t, domain.SyncOptions{})

	for i := 0; i < fresh; i++ {
		if !w.stored[fmt.Sprintf("n%04d", i)] {
			t.Errorf("match neuf n%04d non rangé par le delta du dessus", i)
		}
	}
	// m0199 est maintenant à l'entrée 206 : la page de recouvrement commence à 207 - 25 = 182.
	if len(w.pageStarts) == 0 || w.pageStarts[0] != 182 {
		t.Fatalf("pages de reprise = %v, attendu une première page à 182 (marque-page 200 + N 7 - 25)", w.pageStarts)
	}
	for _, s := range w.pageStarts {
		if s < 182 {
			t.Errorf("page %d relue : la reprise ne repasse pas sous la page de recouvrement", s)
		}
	}
	if res.MatchesInserted < 200 {
		t.Errorf("insérés = %d, attendu au moins le budget (7 neufs + la reprise)", res.MatchesInserted)
	}
	if w.deltaCalls != 1 || w.marked != 0 || !res.HistoryPending {
		t.Errorf("delta=%d fin rangée=%d pending=%v", w.deltaCalls, w.marked, res.HistoryPending)
	}

	// Les passes suivantes finissent le travail : tout est rangé, la fin aussi.
	for i := 0; i < 3 && w.marked == 0; i++ {
		w.run(t, domain.SyncOptions{})
	}
	if len(w.stored) != 450+fresh || w.marked != 1 {
		t.Errorf("rangés=%d fin rangée=%d, attendu %d/1", len(w.stored), w.marked, 450+fresh)
	}
}

// Le marque-page n'est pas sur la page de recouvrement : la passe repart du début (journalisé, compté),
// et finit par ranger la fin.
func TestRunDelta_Rattrapage_RecouvrementRateRepartDuDebut(t *testing.T) {
	w := newHistoryWorld(100)
	w.run(t, domain.SyncOptions{MaxMatches: 50}) // 50 matchs, marque-page {50 m0049}
	w.cursor = &historystate.Cursor{Offset: 75, LastMatchID: "m0010"}
	w.pageStarts = nil
	missedBefore := observability.LoadCounter(metricCursorMissed)

	res := w.run(t, domain.SyncOptions{})

	if len(w.pageStarts) < 2 || w.pageStarts[0] != 50 || w.pageStarts[1] != 0 {
		t.Fatalf("pages = %v, attendu la page de recouvrement 50 puis la reprise depuis 0", w.pageStarts)
	}
	if got := observability.LoadCounter(metricCursorMissed) - missedBefore; got != 1 {
		t.Errorf("compteur h5_history_cursor_missed_total +%d, attendu +1", got)
	}
	if w.marked != 1 || res.HistoryPending || len(w.stored) != 100 {
		t.Errorf("fin rangée=%d pending=%v rangés=%d, attendu 1/false/100", w.marked, res.HistoryPending, len(w.stored))
	}
}

// Une page dont la capture échoue : pas de fin rangée, l'erreur est dans le résultat, la page d'avant
// reste rangée et le marque-page s'arrête avant la page en erreur ; la passe suivante reprend là.
func TestRunDelta_Rattrapage_ErreurDePageNeRangePasLaFin(t *testing.T) {
	w := newHistoryWorld(60)
	w.errAtStart[25] = errors.New("GetPlayerMatches: 500")
	res := w.run(t, domain.SyncOptions{})

	if w.marked != 0 || !res.HistoryPending || len(res.Errors) == 0 {
		t.Errorf("fin rangée=%d pending=%v erreurs=%v, attendu 0/true/une erreur", w.marked, res.HistoryPending, res.Errors)
	}
	if res.MatchesInserted != 25 || w.cursor == nil || w.cursor.Offset != 25 {
		t.Errorf("insérés=%d marque-page=%+v, attendu 25 et {25 ...} (avant la page en erreur)", res.MatchesInserted, w.cursor)
	}
	if w.streak.Attempts != 1 || w.streak.Subject != "m0024" {
		t.Errorf("compte d'échecs = %+v, attendu 1 échec au point m0024", w.streak)
	}

	delete(w.errAtStart, 25)
	res = w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending || res.MatchesInserted != 35 {
		t.Errorf("passe de reprise : fin rangée=%d pending=%v insérés=%d, attendu 1/false/35", w.marked, res.HistoryPending, res.MatchesInserted)
	}
	if w.streak.Attempts != 0 {
		t.Errorf("la page a réussi : le compte d'échecs doit être remis à zéro, %+v", w.streak)
	}
}

// Une page en échec durable est abandonnée à la troisième passe (journalisée, comptée), puis la fin se range.
func TestRunDelta_Rattrapage_PageEnErreurDurableAbandonneeALaTroisiemePasse(t *testing.T) {
	w := newHistoryWorld(100)
	w.errAtStart[50] = errors.New("GetPlayerMatches: décodage")
	abandonedBefore := observability.LoadCounter(metricAbandonedPages)

	for pass := 1; pass <= 2; pass++ {
		res := w.run(t, domain.SyncOptions{})
		if w.marked != 0 || !res.HistoryPending || len(res.Errors) == 0 {
			t.Fatalf("passe %d : fin rangée=%d pending=%v erreurs=%v, attendu une passe en échec", pass, w.marked, res.HistoryPending, res.Errors)
		}
		if w.streak.Attempts != pass || w.streak.Subject != "m0049" {
			t.Fatalf("passe %d : compte d'échecs %+v, attendu %d échec(s) au point m0049", pass, w.streak, pass)
		}
	}
	if got := observability.LoadCounter(metricAbandonedPages) - abandonedBefore; got != 0 {
		t.Fatalf("pages abandonnées +%d avant la troisième passe, attendu 0", got)
	}

	res := w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending || len(res.Errors) != 0 || res.Status() != "success" {
		t.Fatalf("troisième passe : fin rangée=%d pending=%v erreurs=%v statut=%s, attendu la fin rangée", w.marked, res.HistoryPending, res.Errors, res.Status())
	}
	if got := observability.LoadCounter(metricAbandonedPages) - abandonedBefore; got != 1 {
		t.Errorf("pages abandonnées +%d, attendu +1", got)
	}
	if len(w.stored) != 75 || w.stored["m0050"] || !w.stored["m0075"] {
		t.Errorf("rangés=%d : la page abandonnée (50 à 74) manque, le reste est rangé", len(w.stored))
	}
	if w.streak.Attempts != 0 {
		t.Errorf("compte d'échecs après la fin : %+v", w.streak)
	}
}

// Même abandon pour un match que la base refuse durablement : la fin se range à la troisième passe.
func TestRunDelta_Rattrapage_ErreurDePersistanceDurableAbandonnee(t *testing.T) {
	w := newHistoryWorld(60)
	w.persistErr["m0030"] = true

	for pass := 1; pass <= 2; pass++ {
		res := w.run(t, domain.SyncOptions{})
		if w.marked != 0 || !res.HistoryPending || (pass == 1 && res.Status() != "partial_success") {
			t.Fatalf("passe %d : fin rangée=%d pending=%v statut=%s", pass, w.marked, res.HistoryPending, res.Status())
		}
	}
	res := w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending {
		t.Fatalf("troisième passe : fin rangée=%d pending=%v, attendu la fin rangée (page abandonnée)", w.marked, res.HistoryPending)
	}
	if w.stored["m0030"] || len(w.stored) != 59 {
		t.Errorf("rangés=%d : m0030 reste absent, les 59 autres sont rangés", len(w.stored))
	}
}

// Fin de liste vue mais un match échoue à la persistance : pas de fin rangée (il resterait un trou) ;
// la passe suivante, une fois le match rangé, la range.
func TestRunDelta_Rattrapage_ErreurDePersistanceNeRangePasLaFin(t *testing.T) {
	w := newHistoryWorld(30)
	w.persistErr["m0007"] = true
	res := w.run(t, domain.SyncOptions{})

	if w.marked != 0 || !res.HistoryPending || res.Status() != "partial_success" {
		t.Errorf("fin rangée=%d pending=%v statut=%s, attendu 0/true/partial_success", w.marked, res.HistoryPending, res.Status())
	}
	if w.cursor != nil {
		t.Errorf("la page 0 est en erreur : aucun marque-page ne doit être rangé, %+v", w.cursor)
	}
	delete(w.persistErr, "m0007")
	res = w.run(t, domain.SyncOptions{})
	if w.marked != 1 || res.HistoryPending || res.MatchesInserted != 1 {
		t.Errorf("passe de reprise : fin rangée=%d pending=%v insérés=%d, attendu 1/false/1", w.marked, res.HistoryPending, res.MatchesInserted)
	}
}

// Un match rangé sans timeline ni carnage n'est pas une erreur : la fin se range quand même.
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

// Sans magasin d'historique (runner nu) : comportement d'avant, rien en attente.
func TestRunDelta_SansMagasinResteUnDelta(t *testing.T) {
	w := newHistoryWorld(10)
	deps := w.deps()
	deps.History = nil
	res, err := NewRunner(deps, nil).RunDelta(context.Background(), domain.SyncOptions{})
	if err != nil || w.deltaCalls != 1 || res.HistoryPending {
		t.Errorf("err=%v delta=%d pending=%v", err, w.deltaCalls, res.HistoryPending)
	}
}

// RunBackfill sans budget ni reprise (la commande h5-backfill) parcourt tout l'historique d'un trait ;
// avec un budget il s'arrête et ne déclare pas la fin.
func TestRunBackfill_BudgetOptionnel(t *testing.T) {
	run := func(budget int) BackfillStats {
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
		return stats
	}

	if stats := run(0); stats.Inserted != 450 || !stats.HistoryEnd || stats.BudgetReached || stats.Pages != 19 {
		t.Errorf("sans budget : %+v, attendu 450 insérés, fin vue, 19 pages", stats)
	}
	if stats := run(100); stats.Inserted != 100 || stats.HistoryEnd || !stats.BudgetReached || stats.Pages != 4 {
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
