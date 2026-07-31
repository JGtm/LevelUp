// Package handlers_test — replay_test.go : tests unitaires ReplayHandler.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/port"
)

// mockReplayService implémente port.ReplayService.
type mockReplayService struct {
	doc replay.ReplayDocument
	err error
}

func (m *mockReplayService) GetReplay(_ context.Context, _ string) (replay.ReplayDocument, error) {
	return m.doc, m.err
}

func newReplayRouter(factory handlers.ServiceFactory[port.ReplayService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewReplayHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/matches/{match_id}/replay", h.GetReplay)
	})
	return r
}

// doReplayGet émet une requête depuis la BOUCLE LOCALE.
//
// C'EST NÉCESSAIRE, ET C'EST LE SUJET : le rejeu n'est servi qu'en local (cf.
// replay_local_gate.go). `httptest.NewRequest` pose `192.0.2.1:1234` comme adresse d'appel —
// une adresse de documentation, donc non locale — et le garde répondait 404 à tous ces tests.
// Ils testaient le garde sans le savoir, au lieu de tester le handler.
func doReplayGet(r *chi.Mux, slug, matchID string) *httptest.ResponseRecorder {
	return doReplayGetFrom(r, slug, matchID, "127.0.0.1:54321")
}

// doReplayGetFrom permet de choisir l'adresse d'appel, pour éprouver le garde lui-même.
func doReplayGetFrom(r *chi.Mux, slug, matchID, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/players/"+slug+"/matches/"+matchID+"/replay", nil)
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestReplayHandler_OK(t *testing.T) {
	mock := &mockReplayService{doc: replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: "000d5950", TitleSlug: "halo_infinite",
		FrameCount: 2,
		Tracks:     []replay.Track{{Slot: 665, Team: -1, Points: []replay.Point{{T: 0, X: 1, Y: 2}}}},
	}}
	factory := func(_ context.Context, slug string) (port.ReplayService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "000d5950")
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var got replay.ReplayDocument
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}
	if got.MatchID != "000d5950" || len(got.Tracks) != 1 || got.Tracks[0].Slot != 665 {
		t.Errorf("document inattendu: %+v", got)
	}
}

func TestReplayHandler_NotAvailable(t *testing.T) {
	mock := &mockReplayService{err: port.ErrReplayNotAvailable}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "sans-artefact")
	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "replay_not_available") {
		t.Errorf("code d'erreur attendu replay_not_available, body=%s", w.Body.String())
	}
}

func TestReplayHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.ReplayService, error) {
		return nil, errors.New("player_not_found")
	}
	w := doReplayGet(newReplayRouter(factory), "inconnu", "000d5950")
	if w.Code != http.StatusNotFound {
		t.Fatalf("attendu 404, obtenu %d", w.Code)
	}
}

func TestReplayHandler_ServiceError(t *testing.T) {
	mock := &mockReplayService{err: errors.New("boom")}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGet(newReplayRouter(factory), testPlayerSlug, "000d5950")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d", w.Code)
	}
}

// TestReplayHandler_RefusesRemoteCaller vérifie que LA ROUTE applique le garde local.
//
// CE QU'IL AJOUTE AUX TESTS DU GARDE. `TestReplayGate_*` éprouve la RÈGLE (quelle adresse est
// locale, et qu'un en-tête ne peut rien contre elle) ; celui-ci éprouve son BRANCHEMENT — si
// l'appel à `allowReplay` disparaissait du handler, les tests de la règle resteraient verts et
// le rejeu serait servi à tout le monde.
//
// Il répare aussi le défaut qui l'a fait écrire : le garde a été posé sans que les tests du
// handler soient adaptés, et ceux-ci tombaient dessus en annonçant un handler cassé.
func TestReplayHandler_RefusesRemoteCaller(t *testing.T) {
	mock := &mockReplayService{doc: replay.ReplayDocument{MatchID: "000d5950"}}
	factory := func(_ context.Context, _ string) (port.ReplayService, error) { return mock, nil }
	w := doReplayGetFrom(newReplayRouter(factory), testPlayerSlug, "000d5950", "203.0.113.7:443")
	if w.Code != http.StatusNotFound {
		t.Fatalf("un appelant distant doit recevoir 404, obtenu %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "replay_not_available") {
		t.Errorf("code d'erreur attendu replay_not_available, body=%s", w.Body.String())
	}
}
