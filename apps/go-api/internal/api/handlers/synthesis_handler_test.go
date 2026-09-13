// Package handlers — synthesis_handler_test.go : tests du SynthesisHandler.
// Sprint 55 D9.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

var errSynthHandlerTest = errors.New("synthesis test error")

// --- mock SynthesisService ---

type mockSynthesisService struct {
	resp *domain.SynthesisPageV2Response
	err  error
}

func (m *mockSynthesisService) GetSynthesisPage(
	_ context.Context,
	_ string,
	_ domain.SynthesisRequest,
) (*domain.SynthesisPageV2Response, error) {
	return m.resp, m.err
}

func newSynthesisTestRouter(factory handlers.ContextFactory[port.SynthesisService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSynthesisHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func synthesisContextFactory(svc port.SynthesisService, err error) handlers.ContextFactory[port.SynthesisService] {
	return func(_ context.Context, _ string) (port.SynthesisService, string, string, error) {
		return svc, "test-xuid", testGamertag, err
	}
}

func TestSynthesisHandler_OK_NoBody(t *testing.T) {
	resp := &domain.SynthesisPageV2Response{
		Scope: domain.SynthesisScope{
			Period:     "all",
			MatchCount: 5,
			ComputedAt: time.Now().UTC(),
		},
	}
	mock := &mockSynthesisService{resp: resp}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var got domain.SynthesisPageV2Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if got.Scope.Period != "all" {
		t.Errorf("want period=all, got %q", got.Scope.Period)
	}
}

func TestSynthesisHandler_OK_WithBody(t *testing.T) {
	resp := &domain.SynthesisPageV2Response{
		Scope: domain.SynthesisScope{Period: "1m"},
	}
	mock := &mockSynthesisService{resp: resp}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	body := `{"period":"1m"}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSynthesisHandler_ServiceError(t *testing.T) {
	mock := &mockSynthesisService{err: errSynthHandlerTest}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestSynthesisHandler_PlayerNotFound(t *testing.T) {
	router := newSynthesisTestRouter(synthesisContextFactory(nil, errSynthHandlerTest))

	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/synthesis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestSynthesisHandler_InvalidBody(t *testing.T) {
	mock := &mockSynthesisService{resp: &domain.SynthesisPageV2Response{}}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	badBody := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", badBody)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(badBody.Len())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for invalid JSON, got %d", w.Code)
	}
}

func TestSynthesisHandler_HighlightsInResponse(t *testing.T) {
	kda := 3.0
	resp := &domain.SynthesisPageV2Response{
		Scope: domain.SynthesisScope{Period: "all"},
		HighlightsPreview: domain.SynthesisHighlightsPreview{
			TopByKills: []domain.SynthesisMatchHighlight{
				{MatchID: "m1", Kills: 20, KDA: &kda, Outcome: 2},
			},
		},
	}
	mock := &mockSynthesisService{resp: resp}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "highlights_preview") {
		t.Error("response should contain highlights_preview field")
	}
	if !strings.Contains(body, "m1") {
		t.Error("response should contain match id m1")
	}
}

// --- D9 : scope.period propagé dans la réponse JSON ---

// TestSynthesisHandler_ScopePeriodInResponse vérifie que la période du body
// est retournée dans scope.period de la réponse JSON.
func TestSynthesisHandler_ScopePeriodInResponse(t *testing.T) {
	resp := &domain.SynthesisPageV2Response{
		Scope: domain.SynthesisScope{Period: "1m", MatchCount: 42},
	}
	mock := &mockSynthesisService{resp: resp}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	body := `{"period":"1m"}`
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	var scope struct {
		Period     string `json:"period"`
		MatchCount int    `json:"match_count"`
	}
	if err := json.Unmarshal(out["scope"], &scope); err != nil {
		t.Fatalf("cannot parse scope: %v", err)
	}
	if scope.Period != "1m" {
		t.Errorf("scope.period = %q, want %q", scope.Period, "1m")
	}
	if scope.MatchCount != 42 {
		t.Errorf("scope.match_count = %d, want 42", scope.MatchCount)
	}
}

// TestSynthesisHandler_OverviewInResponse vérifie que le bloc overview
// est présent et non-nul dans la réponse JSON.
func TestSynthesisHandler_OverviewInResponse(t *testing.T) {
	resp := &domain.SynthesisPageV2Response{
		Scope: domain.SynthesisScope{Period: "all", MatchCount: 5},
		Overview: domain.SynthesisOverview{
			TotalMatches: 5,
			TotalWins:    3,
			WinRate:      0.6,
		},
	}
	mock := &mockSynthesisService{resp: resp}
	router := newSynthesisTestRouter(synthesisContextFactory(mock, nil))

	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/synthesis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if _, ok := out["overview"]; !ok {
		t.Error("response should contain overview field")
	}
	var overview struct {
		TotalMatches int     `json:"total_matches"`
		WinRate      float64 `json:"win_rate"`
	}
	if err := json.Unmarshal(out["overview"], &overview); err != nil {
		t.Fatalf("cannot parse overview: %v", err)
	}
	if overview.TotalMatches != 5 {
		t.Errorf("overview.total_matches = %d, want 5", overview.TotalMatches)
	}
}
