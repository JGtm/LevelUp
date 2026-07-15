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
