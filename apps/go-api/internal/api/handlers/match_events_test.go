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
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

type fakeMatchEventsSvc struct {
	tl   *canonical.MatchEventTimeline
	err  error
	seen *canonical.MatchEventOptions
}

func (f fakeMatchEventsSvc) GetMatchEvents(_ context.Context, _ string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	if f.seen != nil {
		*f.seen = opts
	}
	return f.tl, f.err
}

func newMatchEventsRouter(factory handlers.ServiceFactory[port.MatchEventsService]) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) {
		handlers.NewMatchEventsHandler(factory).Mount(sub)
	})
	return r
}

func matchEventsFactory(svc port.MatchEventsService, err error) handlers.ServiceFactory[port.MatchEventsService] {
	return func(context.Context, string) (port.MatchEventsService, error) { return svc, err }
}

func TestMatchEventsHandler_Happy(t *testing.T) {
	tl := &canonical.MatchEventTimeline{
		MatchID: "m1",
		Events:  []canonical.MatchEvent{{Type: canonical.MatchEventKill, TimeMs: 100}},
	}
	r := newMatchEventsRouter(matchEventsFactory(fakeMatchEventsSvc{tl: tl}, nil))
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/matches/m1/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got canonical.MatchEventTimeline
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.MatchID != "m1" || len(got.Events) != 1 {
		t.Errorf("payload inattendu: %+v", got)
	}
}

func TestMatchEventsHandler_CapabilityNotSupported503(t *testing.T) {
	r := newMatchEventsRouter(matchEventsFactory(fakeMatchEventsSvc{err: games.ErrCapabilityNotSupported}, nil))
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/matches/m1/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("capability non supportée → 503, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestMatchEventsHandler_InvalidType400(t *testing.T) {
	r := newMatchEventsRouter(matchEventsFactory(fakeMatchEventsSvc{tl: &canonical.MatchEventTimeline{MatchID: "m1"}}, nil))
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/matches/m1/events?types=bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("type d'event inconnu → 400, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestMatchEventsHandler_PlayerNotFound404(t *testing.T) {
	r := newMatchEventsRouter(matchEventsFactory(nil, errors.New("no such player")))
	req := httptest.NewRequest(http.MethodGet, "/players/unknown/matches/m1/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("joueur inconnu → 404, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestMatchEventsHandler_TypesFilterParsed(t *testing.T) {
	var seen canonical.MatchEventOptions
	svc := fakeMatchEventsSvc{tl: &canonical.MatchEventTimeline{MatchID: "m1"}, seen: &seen}
	r := newMatchEventsRouter(matchEventsFactory(svc, nil))
	req := httptest.NewRequest(http.MethodGet, "/players/JGtm/matches/m1/events?types=kill,medal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(seen.Types) != 2 || seen.Types[0] != canonical.MatchEventKill || seen.Types[1] != canonical.MatchEventMedal {
		t.Errorf("filtre types non transmis: %+v", seen.Types)
	}
}
