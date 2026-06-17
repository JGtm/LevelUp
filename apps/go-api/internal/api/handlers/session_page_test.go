package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type mockSessionPageService struct {
	resp domain.SessionPageResponse
	err  error
}

func (m *mockSessionPageService) GetPage(_ context.Context, _ domain.SessionPageRequest) (domain.SessionPageResponse, error) {
	return m.resp, m.err
}

func newSessionPageRouter(factory handlers.ServiceFactory[port.SessionPageService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSessionPageHandler(factory)
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		h.Mount(sub)
	})
	return r
}

func TestSessionPageHandler_OK(t *testing.T) {
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{AvailableSessions: []string{"S1"}}}
	factory := func(_ context.Context, slug string) (port.SessionPageService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	body, _ := json.Marshal(domain.SessionPageRequest{EnableCompare: true})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionPageHandler_OKWithResolvedSession(t *testing.T) {
	sessionLabel := "S1"
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{
		CurrentSession:    &domain.SessionCompareEntry{SessionLabel: sessionLabel},
		AvailableSessions: []string{sessionLabel},
	}}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	body, _ := json.Marshal(domain.SessionPageRequest{SessionLabel: &sessionLabel})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionPageHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/sessions/detail", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSessionPageHandler_ServiceError(t *testing.T) {
	mock := &mockSessionPageService{err: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ADR 0024 Couche B : une session demandée inexistante (APIError
// session_not_found) est mappée en 404 avec le code distinct, pas en 500.
func TestSessionPageHandler_SessionNotFound_404(t *testing.T) {
	mock := &mockSessionPageService{
		err: &domain.APIError{Code: "session_not_found", Message: "session introuvable"},
	}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	body, _ := json.Marshal(domain.SessionPageRequest{SessionLabel: strptrSess("S-MISSING")})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["code"] != "session_not_found" {
		t.Errorf("code = %v, want session_not_found", resp["code"])
	}
}

func strptrSess(s string) *string { return &s }

func TestSessionPageHandler_InvalidBody(t *testing.T) {
	mock := &mockSessionPageService{}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSessionPageHandler_InvalidRequest(t *testing.T) {
	mock := &mockSessionPageService{}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail", bytes.NewReader([]byte(`{"session_label":""}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestSessionPageHandler_SerializesDrawerFields vérifie que les nouveaux champs
// du drawer compare (P3) sortent bien dans le JSON : compare_matches,
// previous_session_label, next_session_label. Sans ça le drawer côté front
// n'a aucun moyen d'alimenter ses charts ni de naviguer entre sessions.
func TestSessionPageHandler_SerializesDrawerFields(t *testing.T) {
	prev := "S-prev"
	next := "S-next"
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{
		CurrentSession:       &domain.SessionCompareEntry{SessionLabel: "S-current"},
		AvailableSessions:    []string{"S-prev", "S-current", "S-next"},
		Matches:              []domain.SessionDetailMatchRow{{MatchID: "m1"}},
		CompareEnabled:       true,
		CompareSession:       &domain.SessionCompareEntry{SessionLabel: "S-prev"},
		CompareMatches:       []domain.SessionDetailMatchRow{{MatchID: "m2"}, {MatchID: "m3"}},
		CompareMetrics:       []domain.SessionCompareMetricRow{},
		PreviousSessionLabel: &prev,
		NextSessionLabel:     &next,
	}}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail",
		bytes.NewReader([]byte(`{"enable_compare":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		CompareMatches       []domain.SessionDetailMatchRow `json:"compare_matches"`
		PreviousSessionLabel *string                        `json:"previous_session_label,omitempty"`
		NextSessionLabel     *string                        `json:"next_session_label,omitempty"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(body.CompareMatches) != 2 {
		t.Fatalf("expected 2 compare_matches in JSON, got %d", len(body.CompareMatches))
	}
	if body.PreviousSessionLabel == nil || *body.PreviousSessionLabel != "S-prev" {
		t.Fatalf("expected previous_session_label=S-prev, got %v", body.PreviousSessionLabel)
	}
	if body.NextSessionLabel == nil || *body.NextSessionLabel != "S-next" {
		t.Fatalf("expected next_session_label=S-next, got %v", body.NextSessionLabel)
	}
}

// TestSessionPageHandler_OmitsNeighborsAtBoundaries vérifie que `omitempty`
// joue son rôle quand prev/next sont nil (bornes de la liste de sessions).
func TestSessionPageHandler_OmitsNeighborsAtBoundaries(t *testing.T) {
	mock := &mockSessionPageService{resp: domain.SessionPageResponse{
		CurrentSession:    &domain.SessionCompareEntry{SessionLabel: "only"},
		AvailableSessions: []string{"only"},
		Matches:           []domain.SessionDetailMatchRow{},
		CompareMatches:    []domain.SessionDetailMatchRow{},
		CompareMetrics:    []domain.SessionCompareMetricRow{},
		// Pas de PreviousSessionLabel ni NextSessionLabel : pointeurs nil.
	}}
	factory := func(_ context.Context, _ string) (port.SessionPageService, error) {
		return mock, nil
	}
	r := newSessionPageRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/sessions/detail",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Inspection brute du JSON : les clés omitempty ne doivent PAS apparaître.
	raw := w.Body.String()
	if bytes.Contains([]byte(raw), []byte("previous_session_label")) {
		t.Fatalf("expected previous_session_label to be omitted, got: %s", raw)
	}
	if bytes.Contains([]byte(raw), []byte("next_session_label")) {
		t.Fatalf("expected next_session_label to be omitted, got: %s", raw)
	}
}
