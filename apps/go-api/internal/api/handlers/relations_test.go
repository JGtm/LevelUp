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

type mockRelationsService struct {
	page     domain.RelationsPageResponse
	moments  domain.RelationsMomentsResponse
	err      error
	gotInput domain.FilterContextInput
}

func (m *mockRelationsService) GetRelationsPage(_ context.Context, in domain.FilterContextInput) (domain.RelationsPageResponse, error) {
	m.gotInput = in
	return m.page, m.err
}

func (m *mockRelationsService) GetRelationsMoments(_ context.Context, in domain.FilterContextInput) (domain.RelationsMomentsResponse, error) {
	m.gotInput = in
	return m.moments, m.err
}

func newRelationsRouter(factory handlers.RelationsFactory) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewRelationsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

const relationsPath = "/players/" + testPlayerSlug + "/pages/palmares/relations"

// Corps absent → 200, sélection zéro-valeur (= tout) transmise au service.
func TestRelationsHandler_OK_Empty(t *testing.T) {
	mock := &mockRelationsService{page: domain.RelationsPageResponse{Relations: []domain.RelationInsight{}}}
	factory := func(_ context.Context, slug string) (port.RelationsService, error) {
		if slug != testPlayerSlug {
			t.Fatalf("unexpected slug %q", slug)
		}
		return mock, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", w.Code, w.Body.String())
	}
	var resp domain.RelationsPageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Overview.DistinctPlayers != 0 {
		t.Fatalf("distinct=%d want 0", resp.Overview.DistinctPlayers)
	}
	if mock.gotInput.MatchContext != "" || len(mock.gotInput.Cascade.Playlists) != 0 {
		t.Fatalf("empty body should yield zero-value input, got %+v", mock.gotInput)
	}
}

// Corps avec FilterContextInput → décodé et transmis au service.
func TestRelationsHandler_FilterBodyForwarded(t *testing.T) {
	mock := &mockRelationsService{page: domain.RelationsPageResponse{Relations: []domain.RelationInsight{}}}
	factory := func(_ context.Context, _ string) (port.RelationsService, error) { return mock, nil }
	r := newRelationsRouter(factory)

	body := `{"filter_mode":"period","match_context":"squad","cascade":{"playlists":["Ranked Arena"],"modes":["Slayer"]}}`
	req := httptest.NewRequest(http.MethodPost, relationsPath, bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", w.Code, w.Body.String())
	}
	if mock.gotInput.MatchContext != "squad" {
		t.Fatalf("match_context=%q want squad", mock.gotInput.MatchContext)
	}
	if len(mock.gotInput.Cascade.Playlists) != 1 || mock.gotInput.Cascade.Playlists[0] != "Ranked Arena" {
		t.Fatalf("playlists=%v want [Ranked Arena]", mock.gotInput.Cascade.Playlists)
	}
}

// JSON invalide → 400 invalid_body.
func TestRelationsHandler_InvalidBody(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return &mockRelationsService{}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsPath, bytes.NewBufferString("{not json"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", w.Code, w.Body.String())
	}
}

// match_context invalide → 400 (Validate).
func TestRelationsHandler_InvalidMatchContext(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return &mockRelationsService{}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsPath, bytes.NewBufferString(`{"match_context":"duo"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestRelationsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return nil, errors.New("not found")
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

const relationsMomentsPath = "/players/" + testPlayerSlug + "/pages/palmares/relations/moments"

// Sous-endpoint Moments : corps absent → 200, réponse {heatmap, rivalries}.
func TestRelationsHandler_Moments_OK(t *testing.T) {
	mock := &mockRelationsService{moments: domain.RelationsMomentsResponse{
		Heatmap:      []domain.RelationHeatmapCell{{XUID: "x1", Gamertag: "Foe", Hour: 19, Count: 7}},
		Rivalries:    []domain.RelationRivalry{{XUID: "x1", Gamertag: "Foe", EnemyMatches: 12}},
		TopRelations: 8,
	}}
	factory := func(_ context.Context, _ string) (port.RelationsService, error) { return mock, nil }
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsMomentsPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", w.Code, w.Body.String())
	}
	var resp domain.RelationsMomentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Heatmap) != 1 || resp.Heatmap[0].Count != 7 {
		t.Fatalf("heatmap=%+v", resp.Heatmap)
	}
	if len(resp.Rivalries) != 1 || resp.Rivalries[0].EnemyMatches != 12 {
		t.Fatalf("rivalries=%+v", resp.Rivalries)
	}
}

// Sous-endpoint Moments : corps invalide → 400.
func TestRelationsHandler_Moments_InvalidBody(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return &mockRelationsService{}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsMomentsPath, bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestRelationsHandler_ServiceError(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return &mockRelationsService{err: errors.New("boom")}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodPost, relationsPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500", w.Code)
	}
}
