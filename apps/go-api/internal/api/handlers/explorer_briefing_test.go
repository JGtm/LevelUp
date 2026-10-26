package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// capturingMatchHist capture la MatchHistoryQueryRequest reçue et sert une page
// fixe, pour vérifier la propagation du flag include_briefing.
type capturingMatchHist struct {
	got  domain.MatchHistoryQueryRequest
	page domain.MatchHistoryPageResponse
}

func (m *capturingMatchHist) GetPage(_ context.Context, req domain.MatchHistoryQueryRequest) (domain.MatchHistoryPageResponse, error) {
	m.got = req
	return m.page, nil
}

func (m *capturingMatchHist) ExportCSV(_ context.Context, _ domain.MatchHistoryQueryRequest) ([]domain.MatchHistoryRow, error) {
	return nil, nil
}

func (m *capturingMatchHist) OutcomeText(_ context.Context, _ int) string {
	return ""
}

func postMatchesQuery(t *testing.T, mock port.MatchHistoryService, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	explorerF := func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
		return &mockExplorerService{}, ctx, testXUID, "gt", nil
	}
	matchHistF := func(_ context.Context, _ string) (port.MatchHistoryService, string, string, error) {
		return mock, testXUID, "gt", nil
	}
	r := newExplorerRouter(explorerF, matchHistF)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/explorer/matches-query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestExplorerHandler_MatchesQuery_BriefingPropagatedAndReturned(t *testing.T) {
	mock := &capturingMatchHist{
		page: domain.MatchHistoryPageResponse{
			Briefing: &domain.ExplorerBriefing{LowSample: true},
		},
	}
	body, _ := json.Marshal(domain.ExplorerMatchesQueryRequest{IncludeBriefing: true})
	w := postMatchesQuery(t, mock, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !mock.got.IncludeExplorerBriefing {
		t.Error("include_briefing:true must propagate to IncludeExplorerBriefing")
	}
	var resp domain.ExplorerMatchesQueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Briefing == nil {
		t.Fatal("briefing should be carried through to the Explorer response")
	}
	if !resp.Briefing.LowSample {
		t.Error("briefing content should be preserved (LowSample)")
	}
}

func TestExplorerHandler_MatchesQuery_NoBriefingByDefault(t *testing.T) {
	// Pas de briefing servi par le service (flag absent) → réponse sans briefing.
	mock := &capturingMatchHist{page: domain.MatchHistoryPageResponse{}}
	body, _ := json.Marshal(domain.ExplorerMatchesQueryRequest{}) // include_briefing omis
	w := postMatchesQuery(t, mock, body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mock.got.IncludeExplorerBriefing {
		t.Error("include_briefing absent must yield IncludeExplorerBriefing=false")
	}
	var resp domain.ExplorerMatchesQueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Briefing != nil {
		t.Error("no briefing expected when flag absent")
	}
}

// Une carte « Par… » ou « Par contexte » en état vide traverse le handler avec
// `insufficient`, `entries: []` et le seuil `min_group_matches` servi au front.
func TestExplorerHandler_MatchesQuery_BriefingInsufficientFields(t *testing.T) {
	mock := &capturingMatchHist{
		page: domain.MatchHistoryPageResponse{
			Briefing: &domain.ExplorerBriefing{
				MinGroupMatches: 5,
				Dimensions: []domain.ExplorerBriefingDimension{
					{Dimension: "map", Entries: []domain.ExplorerBriefingDimensionEntry{}, Insufficient: true},
				},
				ContextSplit: &domain.ExplorerBriefingContextSplit{
					Insufficient: true,
					Solo:         domain.ExplorerBriefingContextGroup{Matches: 3},
					Squad:        domain.ExplorerBriefingContextGroup{Matches: 63},
				},
			},
		},
	}
	body, _ := json.Marshal(domain.ExplorerMatchesQueryRequest{IncludeBriefing: true})
	w := postMatchesQuery(t, mock, body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Briefing struct {
			MinGroupMatches int `json:"min_group_matches"`
			Dimensions      []struct {
				Dimension    string            `json:"dimension"`
				Entries      []json.RawMessage `json:"entries"`
				Insufficient bool              `json:"insufficient"`
			} `json:"dimensions"`
			ContextSplit struct {
				Insufficient bool `json:"insufficient"`
				Solo         struct {
					Matches int `json:"matches"`
				} `json:"solo"`
				Squad struct {
					Matches int `json:"matches"`
				} `json:"squad"`
			} `json:"context_split"`
		} `json:"briefing"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := resp.Briefing
	if b.MinGroupMatches != 5 {
		t.Errorf("min_group_matches = %d, want 5", b.MinGroupMatches)
	}
	if len(b.Dimensions) != 1 || !b.Dimensions[0].Insufficient || b.Dimensions[0].Entries == nil {
		t.Fatalf("dimension insufficient attendue avec entries [] : %+v", b.Dimensions)
	}
	if !b.ContextSplit.Insufficient || b.ContextSplit.Solo.Matches != 3 || b.ContextSplit.Squad.Matches != 63 {
		t.Errorf("context_split insufficient 3/63 attendu : %+v", b.ContextSplit)
	}
	for _, key := range []string{`"min_group_matches":5`, `"insufficient":true`, `"entries":[]`} {
		if !bytes.Contains(w.Body.Bytes(), []byte(key)) {
			t.Errorf("JSON sans %s : %s", key, w.Body.String())
		}
	}
}
