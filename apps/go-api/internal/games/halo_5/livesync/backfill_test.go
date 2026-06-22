package livesync

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
)

// fakeCapture simule halo5.CapturePageAt : sert des pages de match_ids indexées par
// `start`, honore le skip-known (isKnown) SANS delta-stop, et signale hasMore=false sur
// page vide ou incomplète. Reproduit fidèlement le contrat de CapturePageAt (stats
// cumulées en place, seen partagé) afin que le test de RunBackfill valide la BOUCLE
// (pagination + persist-par-page + resume) sans dépendre de la vraie source h5 (dont
// le type events est non exporté → non mockable hors package).
type fakeCapture struct {
	pages    map[int][]string // start -> match_ids de la page
	pageSize int
	calls    []int // historique des `start` capturés (preuve d'arrêt)
}

func (f *fakeCapture) capture(_ context.Context, viewer canonical.PlayerIdentity,
	_ func(string) string, isKnown func(string) bool,
	start, _ int, seen map[string]struct{}, stats *halo5.CaptureStats,
) ([]*persist.MatchBatch, bool, error) {
	f.calls = append(f.calls, start)
	ids, ok := f.pages[start]
	if !ok || len(ids) == 0 {
		return nil, false, nil // page vide = fin d'historique
	}
	var batches []*persist.MatchBatch
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return batches, false, nil // pagination non avançante
		}
		seen[id] = struct{}{}
		stats.MatchesSeen++
		if isKnown(id) {
			stats.MatchesSkipped++ // skip-known SANS stop (backfill)
			continue
		}
		batches = append(batches, batchFor(id, viewer))
		stats.MatchesCollected++
	}
	hasMore := len(ids) >= f.pageSize // page incomplète = dernière
	return batches, hasMore, nil
}

// batchFor construit un MatchBatch minimal (registry seul) pour un match_id.
func batchFor(id string, viewer canonical.PlayerIdentity) *persist.MatchBatch {
	return persist.NewBatchBuilder(halo5.TitleSlug, viewer.Gamertag, viewer.XUID, "h5_backfill_test").
		SetMatch(&domain.MatchRegistryRow{
			MatchID:   id,
			StartTime: time.Date(2023, 4, 5, 12, 0, 0, 0, time.UTC),
		}).
		Build()
}

// collectingPersist capture les batches persistés page par page (preuve d'incrément).
type collectingPersist struct {
	pages [][]string // un []match_id par appel (= par page persistée)
}

func (c *collectingPersist) persist(_ context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
	ids := make([]string, 0, len(batches))
	for _, b := range batches {
		if b.Shared.Match != nil {
			ids = append(ids, b.Shared.Match.MatchID)
		}
	}
	c.pages = append(c.pages, ids)
	return batches, nil // tout persisté, aucune erreur
}

func backfillViewer() canonical.PlayerIdentity {
	return canonical.PlayerIdentity{Gamertag: "JGtm", XUID: "xJG"}
}

func idResolve(gt string) string {
	if gt == "JGtm" {
		return "xJG"
	}
	return ""
}

// TestRunBackfill_PaginatesThreeFullPagesThenStops — 3 pages PLEINES (pageSize=2) puis
// page vide → les 3 pages sont persistées (incrémental, 1 persist par page) et la 4e
// requête (page vide) arrête la boucle. Scénario nominal du backfill full-historique.
func TestRunBackfill_PaginatesThreeFullPagesThenStops(t *testing.T) {
	const pageSize = 2
	fc := &fakeCapture{pageSize: pageSize, pages: map[int][]string{
		0: {"m1", "m2"}, // page pleine -> continue
		2: {"m3", "m4"}, // page pleine -> continue
		4: {"m5", "m6"}, // page pleine -> continue
		// start=6 -> non mappé -> page vide -> arrêt
	}}
	cp := &collectingPersist{}
	deps := BackfillDeps{
		CapturePage: fc.capture,
		Viewer:      backfillViewer(),
		ResolveXUID: idResolve,
		LoadKnown:   func(context.Context) (map[string]bool, error) { return map[string]bool{}, nil },
		PersistPage: cp.persist,
	}

	stats, err := RunBackfill(context.Background(), deps, pageSize, nil)
	if err != nil {
		t.Fatalf("RunBackfill: %v", err)
	}

	// 3 pages persistées (1 appel persist par page non vide) — preuve de l'incrément.
	if len(cp.pages) != 3 {
		t.Fatalf("pages persistées = %d, want 3 (incrémental par page)", len(cp.pages))
	}
	for i, want := range [][]string{{"m1", "m2"}, {"m3", "m4"}, {"m5", "m6"}} {
		if len(cp.pages[i]) != 2 || cp.pages[i][0] != want[0] || cp.pages[i][1] != want[1] {
			t.Errorf("page %d persistée = %v, want %v", i, cp.pages[i], want)
		}
	}

	// La 4e requête (start=6) a bien eu lieu ET a renvoyé vide → arrêt.
	wantCalls := []int{0, 2, 4, 6}
	if len(fc.calls) != len(wantCalls) {
		t.Fatalf("requêtes capture = %v, want %v (4e vide arrête)", fc.calls, wantCalls)
	}
	for i := range wantCalls {
		if fc.calls[i] != wantCalls[i] {
			t.Errorf("call %d start = %d, want %d", i, fc.calls[i], wantCalls[i])
		}
	}

	if stats.Pages != 4 || stats.Inserted != 6 || stats.Skipped != 0 || stats.MatchesSeen != 6 {
		t.Errorf("stats = %+v, want pages4/inserted6/skipped0/seen6", stats)
	}
}

// TestRunBackfill_SkipKnownButPaginatesDeeper — known-set préchargé : les matchs connus
// sont SAUTÉS (Skipped) mais la boucle CONTINUE à paginer plus profond (PAS de
// delta-stop). Stratégie resume : relancer après interruption ne re-persiste pas les
// pages déjà faites mais creuse jusqu'à la fin de l'historique.
func TestRunBackfill_SkipKnownButPaginatesDeeper(t *testing.T) {
	const pageSize = 2
	fc := &fakeCapture{pageSize: pageSize, pages: map[int][]string{
		0: {"m1", "m2"}, // déjà connus -> sautés, mais on continue
		2: {"m3", "m4"}, // nouveaux -> persistés
	}}
	cp := &collectingPersist{}
	deps := BackfillDeps{
		CapturePage: fc.capture,
		Viewer:      backfillViewer(),
		ResolveXUID: idResolve,
		// m1, m2 déjà en base : un delta-stop s'arrêterait ici → on prouve qu'on continue.
		LoadKnown:   func(context.Context) (map[string]bool, error) { return map[string]bool{"m1": true, "m2": true}, nil },
		PersistPage: cp.persist,
	}

	stats, err := RunBackfill(context.Background(), deps, pageSize, nil)
	if err != nil {
		t.Fatalf("RunBackfill: %v", err)
	}

	// page 0 tout-connu = aucun batch → pas d'appel persist ; m3/m4 persistés (1 page).
	if len(cp.pages) != 1 {
		t.Fatalf("pages persistées = %d, want 1 (page 0 tout-connu = aucun batch)", len(cp.pages))
	}
	if len(cp.pages[0]) != 2 || cp.pages[0][0] != "m3" || cp.pages[0][1] != "m4" {
		t.Errorf("page persistée = %v, want [m3 m4]", cp.pages[0])
	}
	if stats.Skipped != 2 {
		t.Errorf("skipped = %d, want 2 (m1, m2 connus)", stats.Skipped)
	}
	if stats.Inserted != 2 {
		t.Errorf("inserted = %d, want 2 (m3, m4)", stats.Inserted)
	}
	// Preuve qu'on a paginé AU-DELÀ du 1er connu (delta-stop aurait arrêté à start=0).
	if len(fc.calls) < 2 || fc.calls[1] != 2 {
		t.Errorf("calls = %v, want pagination jusqu'à start=2 (pas de delta-stop)", fc.calls)
	}
}
