package handlers_test

import (
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
	page domain.RelationsPageResponse
	err  error
}

func (m *mockRelationsService) GetRelationsPage(_ context.Context) (domain.RelationsPageResponse, error) {
	return m.page, m.err
}

func newRelationsRouter(factory handlers.RelationsFactory) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewRelationsHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func TestRelationsHandler_OK_Empty(t *testing.T) {
	factory := func(_ context.Context, slug string) (port.RelationsService, error) {
		if slug != testPlayerSlug {
			t.Fatalf("unexpected slug %q", slug)
		}
		return &mockRelationsService{page: domain.RelationsPageResponse{Relations: []domain.RelationInsight{}}}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/"+testPlayerSlug+"/pages/palmares/relations", nil)
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
}

func TestRelationsHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return nil, errors.New("not found")
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/"+testPlayerSlug+"/pages/palmares/relations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func TestRelationsHandler_ServiceError(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.RelationsService, error) {
		return &mockRelationsService{err: errors.New("boom")}, nil
	}
	r := newRelationsRouter(factory)
	req := httptest.NewRequest(http.MethodGet, "/players/"+testPlayerSlug+"/pages/palmares/relations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500", w.Code)
	}
}
