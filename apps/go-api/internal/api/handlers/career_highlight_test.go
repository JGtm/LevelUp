package handlers

// career_highlight_test.go — highlight-matches en UNE requête d'enrichissement
// (plan perf 2026-09-23, lot L5b, D5b.6) : parité ligne à ligne avec l'ancien
// enrichissement section par section, et un seul appel à MatchHistoryService.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeHighlightHistory imite MatchHistoryService.GetPage : liste blanche MatchIDs sur
// un historique complet (rendu en ordre chronologique inverse, pas dans l'ordre
// demandé), pagination, et compte ses appels.
type fakeHighlightHistory struct {
	calls atomic.Int32
	rows  []domain.MatchHistoryRow
	err   error
}

func (f *fakeHighlightHistory) GetPage(_ context.Context, req domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error) {
	f.calls.Add(1)
	if f.err != nil {
		return domain.MatchHistoryPageResponse{}, f.err
	}
	keep := map[string]bool{}
	for _, id := range req.MatchIDs {
		keep[id] = true
	}
	items := []domain.MatchHistoryRow{}
	for _, r := range f.rows {
		if keep[r.MatchID] {
			items = append(items, r)
		}
	}
	if size := req.Pagination.PageSize; size > 0 && len(items) > size {
		items = items[:size]
	}
	return domain.MatchHistoryPageResponse{Table: domain.MatchHistoryTable{Items: items}}, nil
}

func (f *fakeHighlightHistory) ExportCSV(context.Context, domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error) {
	return nil, nil
}

func (f *fakeHighlightHistory) OutcomeText(context.Context, int) string { return "" }

// highlightHistoryFixture : 24 matchs enrichis (m23 le plus récent … m00), champs
// variés pour que la projection Explorer ait de quoi différer d'une ligne à l'autre.
func highlightHistoryFixture() []domain.MatchHistoryRow {
	base := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	rows := make([]domain.MatchHistoryRow, 0, 24)
	for i := 23; i >= 0; i-- {
		perf := 30 + i*2
		mapName := fmt.Sprintf("Carte %d", i%5)
		rows = append(rows, domain.MatchHistoryRow{
			MatchID:                  fmt.Sprintf("m%02d", i),
			StartTime:                base.Add(time.Duration(i) * time.Hour),
			OutcomeCode:              2 + i%2,
			Outcome:                  []string{"win", "loss"}[i%2],
			MapUI:                    &mapName,
			PerformanceScoreRelative: &perf,
			Kills:                    10 + i,
			Deaths:                   5 + i%7,
			IsWithFriends:            i%3 == 0,
		})
	}
	return rows
}

// highlightSections : 15 meilleurs + 15 pires dans l'ordre Q9b (ni chronologique ni
// alphabétique), trois matchs dans les deux sections (historique court), un match
// absent de l'historique (exclu entre-temps), des HadBotTeammate.
func highlightSections() (best, worst []domain.HighlightMatchIDRow) {
	bestIDs := []string{"m17", "m03", "m22", "m09", "m11", "m00", "m20", "m05", "m14", "m07", "m19", "m02", "m12", "m99", "m15"}
	worstIDs := []string{"m08", "m13", "m01", "m21", "m06", "m16", "m04", "m18", "m10", "m23", "m17", "m03", "m22", "m09", "m11"}
	for i, id := range bestIDs {
		best = append(best, domain.HighlightMatchIDRow{MatchID: id, Section: 1, HadBotTeammate: i%4 == 0})
	}
	for i, id := range worstIDs {
		worst = append(worst, domain.HighlightMatchIDRow{MatchID: id, Section: 2, HadBotTeammate: i%5 == 0})
	}
	return best, worst
}

// legacyEnrichHighlightMatches est l'enrichissement d'AVANT (une requête par
// section), recopié tel quel comme oracle de parité.
func legacyEnrichHighlightMatches(ctx context.Context, mhSvc port.MatchHistoryService, rows []domain.HighlightMatchIDRow) ([]domain.ExplorerMatchesRow, error) {
	if len(rows) == 0 {
		return []domain.ExplorerMatchesRow{}, nil
	}
	matchIDs := make([]string, len(rows))
	for i, r := range rows {
		matchIDs[i] = r.MatchID
	}
	req := domain.MatchHistoryQueryRequest{
		MatchIDs:   matchIDs,
		Pagination: domain.PaginationRequest{Page: 1, PageSize: len(matchIDs)},
	}
	resp, err := mhSvc.GetPage(ctx, req)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.MatchHistoryRow, len(resp.Table.Items))
	for _, item := range resp.Table.Items {
		byID[item.MatchID] = item
	}
	out := make([]domain.ExplorerMatchesRow, 0, len(rows))
	for _, src := range rows {
		item, ok := byID[src.MatchID]
		if !ok {
			continue
		}
		row := BuildExplorerRowFromMatchHistory(item)
		row.HadBotTeammate = src.HadBotTeammate
		out = append(out, row)
	}
	return out, nil
}

func TestEnrichHighlightSections_ParityWithPerSection(t *testing.T) {
	cases := map[string]func() ([]domain.HighlightMatchIDRow, []domain.HighlightMatchIDRow){
		"deux sections": highlightSections,
		"meilleurs seuls": func() ([]domain.HighlightMatchIDRow, []domain.HighlightMatchIDRow) {
			b, _ := highlightSections()
			return b, nil
		},
		"pires seuls": func() ([]domain.HighlightMatchIDRow, []domain.HighlightMatchIDRow) {
			_, w := highlightSections()
			return nil, w
		},
		"aucune ligne": func() ([]domain.HighlightMatchIDRow, []domain.HighlightMatchIDRow) { return nil, nil },
		"sections sans matchs connus": func() ([]domain.HighlightMatchIDRow, []domain.HighlightMatchIDRow) {
			return []domain.HighlightMatchIDRow{{MatchID: "absent-1"}}, []domain.HighlightMatchIDRow{{MatchID: "absent-2"}}
		},
	}
	for name, sections := range cases {
		t.Run(name, func(t *testing.T) {
			best, worst := sections()
			legacySvc := &fakeHighlightHistory{rows: highlightHistoryFixture()}
			wantBest, err := legacyEnrichHighlightMatches(context.Background(), legacySvc, best)
			if err != nil {
				t.Fatal(err)
			}
			wantWorst, err := legacyEnrichHighlightMatches(context.Background(), legacySvc, worst)
			if err != nil {
				t.Fatal(err)
			}

			svc := &fakeHighlightHistory{rows: highlightHistoryFixture()}
			gotBest, gotWorst, err := enrichHighlightSections(context.Background(), svc, best, worst)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotBest, wantBest) {
				t.Errorf("meilleurs : %d lignes, différentes de l'ancien enrichissement (%d lignes)", len(gotBest), len(wantBest))
			}
			if !reflect.DeepEqual(gotWorst, wantWorst) {
				t.Errorf("pires : %d lignes, différentes de l'ancien enrichissement (%d lignes)", len(gotWorst), len(wantWorst))
			}
			if gotBest == nil || gotWorst == nil {
				t.Errorf("section nil (JSON null) : best=%v worst=%v", gotBest == nil, gotWorst == nil)
			}
			wantCalls := int32(0)
			if len(best)+len(worst) > 0 {
				wantCalls = 1
			}
			if got := svc.calls.Load(); got != wantCalls {
				t.Errorf("GetPage appelé %d fois, want %d (l'ancien chemin : %d)", got, wantCalls, legacySvc.calls.Load())
			}
		})
	}

	// La fixture principale rend bien 14 meilleurs (m99 exclu) et 15 pires, dans
	// l'ordre Q9b, pas dans celui de GetPage.
	best, worst := highlightSections()
	gotBest, gotWorst, _ := enrichHighlightSections(context.Background(), &fakeHighlightHistory{rows: highlightHistoryFixture()}, best, worst)
	if len(gotBest) != 14 || len(gotWorst) != 15 || gotBest[0].MatchID != "m17" || gotWorst[0].MatchID != "m08" {
		t.Errorf("sections : %d/%d lignes, tête %s/%s", len(gotBest), len(gotWorst), gotBest[0].MatchID, gotWorst[0].MatchID)
	}
}

// highlightCareerStub : CareerService minimal qui ne sert que les identifiants.
type highlightCareerStub struct {
	port.CareerService
	rows []domain.HighlightMatchIDRow
}

func (s *highlightCareerStub) GetHighlightMatchIDs(context.Context, domain.HighlightFilterInput) (domain.HighlightMatchesData, error) {
	return domain.HighlightMatchesData{Rows: s.rows}, nil
}

func serveHighlightMatches(t *testing.T, rows []domain.HighlightMatchIDRow, mh *fakeHighlightHistory) *httptest.ResponseRecorder {
	t.Helper()
	h := NewCareerHandler(
		func(context.Context, string) (port.CareerService, error) {
			return &highlightCareerStub{rows: rows}, nil
		},
		func(context.Context, string) (port.MatchHistoryService, string, string, error) {
			return mh, "x", "gt", nil
		},
	)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/p1/pages/career/highlight-matches", nil))
	return w
}

// TestHighlightMatches_OneEnrichmentRequest : l'endpoint ne charge plus l'historique
// qu'une fois pour les deux sections.
func TestHighlightMatches_OneEnrichmentRequest(t *testing.T) {
	best, worst := highlightSections()
	mh := &fakeHighlightHistory{rows: highlightHistoryFixture()}
	w := serveHighlightMatches(t, append(append([]domain.HighlightMatchIDRow{}, best...), worst...), mh)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d : %s", w.Code, w.Body.String())
	}
	var resp domain.CareerHighlightMatchesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.BestMatches) != 14 || len(resp.WorstMatches) != 15 {
		t.Errorf("sections : %d/%d, want 14/15", len(resp.BestMatches), len(resp.WorstMatches))
	}
	if got := mh.calls.Load(); got != 1 {
		t.Errorf("GetPage appelé %d fois, want 1", got)
	}
}

// TestHighlightMatches_ErrorCodes : le code d'erreur reste celui de la première
// section non vide, comme avec une requête par section.
func TestHighlightMatches_ErrorCodes(t *testing.T) {
	best, worst := highlightSections()
	for name, tc := range map[string]struct {
		rows []domain.HighlightMatchIDRow
		code string
	}{
		"meilleurs présents": {rows: append(append([]domain.HighlightMatchIDRow{}, best...), worst...), code: "highlight_best_enrich_error"},
		"pires seuls":        {rows: worst, code: "highlight_worst_enrich_error"},
	} {
		t.Run(name, func(t *testing.T) {
			w := serveHighlightMatches(t, tc.rows, &fakeHighlightHistory{err: errors.New("boom")})
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", w.Code)
			}
			var body struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if body.Code != tc.code {
				t.Errorf("code = %q, want %q (corps : %s)", body.Code, tc.code, w.Body.String())
			}
		})
	}
}
