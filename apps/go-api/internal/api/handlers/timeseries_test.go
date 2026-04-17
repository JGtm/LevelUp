// Package handlers_test — timeseries_test.go : tests unitaires TimeseriesHandler.
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

// mockTimeseriesService implémente port.TimeseriesService.
type mockTimeseriesService struct {
	page    domain.TimeseriesPageResponse
	pageErr error
}

func (m *mockTimeseriesService) GetPage(_ context.Context, _ domain.TimeseriesQueryRequest) (domain.TimeseriesPageResponse, error) {
	return m.page, m.pageErr
}

func newTimeseriesRouter(factory handlers.ServiceFactory[port.TimeseriesService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewTimeseriesHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/timeseries", h.GetPage)
	})
	return r
}

func TestTimeseriesHandler_OK(t *testing.T) {
	mock := &mockTimeseriesService{page: domain.TimeseriesPageResponse{}}
	factory := func(_ context.Context, slug string) (port.TimeseriesService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newTimeseriesRouter(factory)
	body, _ := json.Marshal(domain.TimeseriesQueryRequest{Granularity: "day"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/timeseries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTimeseriesHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.TimeseriesService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newTimeseriesRouter(factory)
	body, _ := json.Marshal(domain.TimeseriesQueryRequest{})
	req := httptest.NewRequest(http.MethodPost, "/players/unknown/pages/timeseries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTimeseriesHandler_ServiceError(t *testing.T) {
	mock := &mockTimeseriesService{pageErr: errors.New("db_error")}
	factory := func(_ context.Context, _ string) (port.TimeseriesService, error) {
		return mock, nil
	}
	r := newTimeseriesRouter(factory)
	body, _ := json.Marshal(domain.TimeseriesQueryRequest{Granularity: "week"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/timeseries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestTimeseriesHandler_InvalidBody(t *testing.T) {
	mock := &mockTimeseriesService{}
	factory := func(_ context.Context, _ string) (port.TimeseriesService, error) {
		return mock, nil
	}
	r := newTimeseriesRouter(factory)
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/timeseries",
		bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
