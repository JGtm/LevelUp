package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// mockTrendsService implémente port.TrendsService.
type mockTrendsService struct {
	page    domain.TrendsPageResponse
	pageErr error
}

func (m *mockTrendsService) GetPage(_ context.Context, _ domain.TrendsQueryRequest) (domain.TrendsPageResponse, error) {
	return m.page, m.pageErr
}

// mockSquadTrendsService implémente port.SquadTrendsService et note ses appels.
type mockSquadTrendsService struct {
	page    domain.TrendsPageResponse
	err     error
	calls   int
	gotXUID string
	gotReq  domain.TrendsQueryRequest
}

func (m *mockSquadTrendsService) GetSquadTrends(_ context.Context, xuid string, req domain.TrendsQueryRequest) (domain.TrendsPageResponse, error) {
	m.calls++
	m.gotXUID, m.gotReq = xuid, req
	return m.page, m.err
}

func postTrends(t *testing.T, svc port.TrendsService, factoryErr error, body string) *httptest.ResponseRecorder {
	t.Helper()
	return postTrendsBoth(t, svc, factoryErr, &mockSquadTrendsService{}, nil, body)
}

// postTrendsBoth branche les deux fabriques (Solo et Escouade).
func postTrendsBoth(
	t *testing.T, svc port.TrendsService, factoryErr error,
	squad *mockSquadTrendsService, squadFactoryErr error, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	factory := func(_ context.Context, _ string) (port.TrendsService, error) {
		if factoryErr != nil {
			return nil, factoryErr
		}
		return svc, nil
	}
	squadFactory := func(_ context.Context, _ string) (port.SquadTrendsService, string, string, error) {
		if squadFactoryErr != nil {
			return nil, "", "", squadFactoryErr
		}
		return squad, "xuid-principal", "Principal", nil
	}
	r := chi.NewRouter()
	h := handlers.NewTrendsHandler(factory, squadFactory)
	r.Route("/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/trends", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTrendsHandler_OK(t *testing.T) {
	for name, body := range map[string]string{"corps {}": "{}", "corps vide": ""} {
		t.Run(name, func(t *testing.T) {
			w := postTrends(t, &mockTrendsService{}, nil, body)
			if w.Code != http.StatusOK {
				t.Fatalf("attendu 200, reçu %d : %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTrendsHandler_InvalidJSON(t *testing.T) {
	w := postTrends(t, &mockTrendsService{}, nil, "{bad")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_json") {
		t.Fatalf("attendu 400 invalid_json, reçu %d : %s", w.Code, w.Body.String())
	}
}

func TestTrendsHandler_InvalidRequest(t *testing.T) {
	for name, body := range map[string]string{
		"vue inconnue": `{"view":"duo"}`,
		"vue squad":    `{"view":"squad"}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := postTrends(t, &mockTrendsService{}, nil, body)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") {
				t.Fatalf("attendu 400 invalid_request, reçu %d : %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestTrendsHandler_PlayerNotFound(t *testing.T) {
	w := postTrends(t, nil, errors.New("player_not_found"), "{}")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "player_not_found") {
		t.Fatalf("attendu 404 player_not_found, reçu %d : %s", w.Code, w.Body.String())
	}
}

func TestTrendsHandler_CapabilityNotSupported(t *testing.T) {
	w := postTrends(t, &mockTrendsService{pageErr: games.ErrCapabilityNotSupported}, nil, "{}")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("attendu 503, reçu %d : %s", w.Code, w.Body.String())
	}
}

const squadBody = `{"view":"squad","selected_gamertags":["Madina97294"],"exact_composition":true}`

func TestTrendsHandler_SquadRoutedToSquadService(t *testing.T) {
	solo := &mockTrendsService{}
	squad := &mockSquadTrendsService{}
	w := postTrendsBoth(t, solo, errors.New("la vue Escouade ne doit pas construire le service Solo"), squad, nil, squadBody)
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, reçu %d : %s", w.Code, w.Body.String())
	}
	if squad.calls != 1 || squad.gotXUID != "xuid-principal" {
		t.Fatalf("appels %d, xuid %q", squad.calls, squad.gotXUID)
	}
	if squad.gotReq.View != domain.TrendsViewSquad || !squad.gotReq.ExactComposition || len(squad.gotReq.SelectedGamertags) != 1 {
		t.Errorf("requête transmise : %+v", squad.gotReq)
	}
}

func TestTrendsHandler_SoloDoesNotBuildSquadService(t *testing.T) {
	squad := &mockSquadTrendsService{}
	factoryCalled := false
	r := chi.NewRouter()
	h := handlers.NewTrendsHandler(
		func(_ context.Context, _ string) (port.TrendsService, error) { return &mockTrendsService{}, nil },
		func(_ context.Context, _ string) (port.SquadTrendsService, string, string, error) {
			factoryCalled = true
			return squad, "x", "g", nil
		})
	r.Route("/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	req := httptest.NewRequest(http.MethodPost, "/players/p/pages/trends", strings.NewReader(`{"view":"solo"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || factoryCalled || squad.calls != 0 {
		t.Fatalf("code %d, fabrique d'escouade appelée : %v", w.Code, factoryCalled)
	}
}

func TestTrendsHandler_SquadPlayerNotFound(t *testing.T) {
	w := postTrendsBoth(t, &mockTrendsService{}, nil, &mockSquadTrendsService{}, errors.New("player_not_found"), squadBody)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "player_not_found") {
		t.Fatalf("attendu 404 player_not_found, reçu %d : %s", w.Code, w.Body.String())
	}
}

func TestTrendsHandler_SquadWithoutGamertag(t *testing.T) {
	squad := &mockSquadTrendsService{}
	w := postTrendsBoth(t, &mockTrendsService{}, nil, squad, nil, `{"view":"squad"}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_request") || squad.calls != 0 {
		t.Fatalf("attendu 400 invalid_request, reçu %d : %s", w.Code, w.Body.String())
	}
}

func TestTrendsHandler_SquadCapabilityNotSupported(t *testing.T) {
	squad := &mockSquadTrendsService{err: games.ErrCapabilityNotSupported}
	w := postTrendsBoth(t, &mockTrendsService{}, nil, squad, nil, squadBody)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("attendu 503, reçu %d : %s", w.Code, w.Body.String())
	}
}
