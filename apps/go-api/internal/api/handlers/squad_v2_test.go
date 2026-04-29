// Package handlers — squad_v2_test.go : tests unitaires SquadV2Handler avec
// mock service. Couvre 200/400/404/503 + parsing query params (teammates,
// period).
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// mockSquadV2Service est un stub de port.SquadV2Service. Il enregistre les
// arguments reçus pour les assertions et peut renvoyer un payload ou une
// erreur configurés par le test.
type mockSquadV2Service struct {
	resp *domain.SquadPageV2Response
	err  error

	// Capture des appels pour assertions
	calledTitleSlug       string
	calledMainGT          string
	calledTeammates       []string
	calledPeriod          temporal.Period
	calledExperienceTypes []string
	calledPlaylists       []string
	calledMaps            []string
	calledModes           []string
}

func (m *mockSquadV2Service) GetSquadPage(
	_ context.Context,
	titleSlug string,
	mainGT string,
	teammates []string,
	period temporal.Period,
	experienceTypes []string,
	playlists []string,
	maps []string,
	modes []string,
) (*domain.SquadPageV2Response, error) {
	m.calledTitleSlug = titleSlug
	m.calledMainGT = mainGT
	m.calledTeammates = teammates
	m.calledPeriod = period
	m.calledExperienceTypes = experienceTypes
	m.calledPlaylists = playlists
	m.calledMaps = maps
	m.calledModes = modes
	return m.resp, m.err
}

func newSquadV2Router(factory handlers.ContextFactory[port.SquadV2Service]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewSquadV2Handler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/squad/v2", h.GetSquadPage)
	})
	return r
}

func makeSquadV2Factory(svc port.SquadV2Service) handlers.ContextFactory[port.SquadV2Service] {
	return func(_ context.Context, slug string) (port.SquadV2Service, string, string, error) {
		if slug == "unknown" {
			return nil, "", "", errors.New("player_not_found")
		}
		return svc, "xuid-test", "MainGT", nil
	}
}

func TestSquadV2Handler_GetSquadPage_OK(t *testing.T) {
	t.Parallel()

	expected := &domain.SquadPageV2Response{
		MainPlayer:         "MainGT",
		Teammates:          []string{"Bob", "Carol"},
		Period:             "1y",
		SharedMatchesCount: 0,
	}
	svc := &mockSquadV2Service{resp: expected}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?teammates=Bob,Carol&period=1y", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.SquadPageV2Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.MainPlayer != "MainGT" {
		t.Errorf("MainPlayer = %q, want MainGT", resp.MainPlayer)
	}

	// Vérifier le passage des params au service
	if svc.calledMainGT != "MainGT" {
		t.Errorf("service called with mainGT=%q, want MainGT", svc.calledMainGT)
	}
	if got, want := svc.calledTeammates, []string{"Bob", "Carol"}; !equalStringSlice(got, want) {
		t.Errorf("teammates=%v, want %v", got, want)
	}
	if svc.calledPeriod != temporal.Period1Y {
		t.Errorf("period=%q, want 1y", svc.calledPeriod)
	}
}

func TestSquadV2Handler_GetSquadPage_DefaultPeriodAll(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/squad/v2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.calledPeriod != temporal.PeriodAll {
		t.Errorf("default period=%q, want all", svc.calledPeriod)
	}
	if len(svc.calledTeammates) != 0 {
		t.Errorf("default teammates=%v, want empty", svc.calledTeammates)
	}
}

func TestSquadV2Handler_GetSquadPage_TooManyTeammates(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	// 4 coéquipiers > max 3
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?teammates=A,B,C,D", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "invalid_teammates" {
		t.Errorf("error code=%v, want invalid_teammates", body["code"])
	}
}

func TestSquadV2Handler_GetSquadPage_InvalidPeriod(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?period=42d", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "invalid_period" {
		t.Errorf("error code=%v, want invalid_period", body["code"])
	}
}

func TestSquadV2Handler_GetSquadPage_PlayerNotFound(t *testing.T) {
	t.Parallel()

	r := newSquadV2Router(makeSquadV2Factory(&mockSquadV2Service{}))

	req := httptest.NewRequest(http.MethodGet, "/players/unknown/pages/squad/v2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSquadV2Handler_GetSquadPage_CapabilityNotSupported(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{err: games.ErrCapabilityNotSupported}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/squad/v2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "capability_not_supported" {
		t.Errorf("error code=%v, want capability_not_supported", body["code"])
	}
}

func TestSquadV2Handler_GetSquadPage_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{err: errors.New("db boom")}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/squad/v2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSquadV2Handler_GetSquadPage_TrimsTeammates(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?teammates=%20Bob%20%2C%20Carol%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got, want := svc.calledTeammates, []string{"Bob", "Carol"}; !equalStringSlice(got, want) {
		t.Errorf("teammates=%v, want %v", got, want)
	}
}

func TestSquadV2Handler_GetSquadPage_CascadeFilters(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?experience_types=PVP+class%C3%A9%2CPVE&playlists=Ranked+Arena&maps=D%C3%A9charge%2CBazar&modes=Assassin%2CCTF", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	wantExp := []string{"PVP classé", "PVE"}
	if !equalStringSlice(svc.calledExperienceTypes, wantExp) {
		t.Errorf("experienceTypes=%v, want %v", svc.calledExperienceTypes, wantExp)
	}
	if !equalStringSlice(svc.calledPlaylists, []string{"Ranked Arena"}) {
		t.Errorf("playlists=%v, want [Ranked Arena]", svc.calledPlaylists)
	}
	wantMaps := []string{"Décharge", "Bazar"}
	if !equalStringSlice(svc.calledMaps, wantMaps) {
		t.Errorf("maps=%v, want %v", svc.calledMaps, wantMaps)
	}
	wantModes := []string{"Assassin", "CTF"}
	if !equalStringSlice(svc.calledModes, wantModes) {
		t.Errorf("modes=%v, want %v", svc.calledModes, wantModes)
	}
}

func TestSquadV2Handler_GetSquadPage_ModesFilter(t *testing.T) {
	t.Parallel()

	svc := &mockSquadV2Service{resp: &domain.SquadPageV2Response{}}
	r := newSquadV2Router(makeSquadV2Factory(svc))

	// Vérifier que modes seul (sans les autres filtres) est bien transmis.
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/squad/v2?modes=Slayer", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !equalStringSlice(svc.calledModes, []string{"Slayer"}) {
		t.Errorf("modes=%v, want [Slayer]", svc.calledModes)
	}
	if len(svc.calledMaps) != 0 || len(svc.calledPlaylists) != 0 {
		t.Errorf("autres filtres ne doivent pas être pollués: maps=%v playlists=%v",
			svc.calledMaps, svc.calledPlaylists)
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
