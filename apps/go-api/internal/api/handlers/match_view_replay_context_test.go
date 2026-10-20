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

func newReplayContextRouter(svc *mockMatchViewService) *chi.Mux {
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return svc, nil
	}
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(factory).WithMediaURLs(nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.MountReplayContext(r)
	})
	return r
}

func getReplayContext(t *testing.T, svc *mockMatchViewService) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123/replay/context", nil)
	w := httptest.NewRecorder()
	newReplayContextRouter(svc).ServeHTTP(w, req)
	return w
}

// TestReplayContextHandler_OK : 200, les quatre clés du contrat, les chemins des médias
// réécrits en URLs servables comme sur la vue complète, aucun Cache-Control sur un contexte
// complet.
func TestReplayContextHandler_OK(t *testing.T) {
	thumb := "captures/clip.webp"
	gt := "Moi"
	svc := &mockMatchViewService{replayCtx: domain.MatchReplayContext{
		Header:          domain.MatchViewHeader{MatchID: "abc123", T0Ms: 15_000},
		Scoreboard:      []domain.MatchScoreboardRow{{XUID: "me", Gamertag: "Moi", IsMe: true}},
		HighlightEvents: []domain.MatchHighlightEvent{{EventType: "kill", ActorGamertag: &gt}},
		Media: domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{
			{FileID: "f1", FilePath: "captures/clip.mp4", ThumbnailURL: &thumb},
		}},
	}}
	w := getReplayContext(t, svc)
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); cc != "" {
		t.Errorf("Cache-Control = %q sur un contexte complet, attendu absent", cc)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("JSON : %v", err)
	}
	for _, key := range []string{"header", "scoreboard", "highlight_events", "media"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("clé %q absente : %s", key, w.Body.String())
		}
	}
	if len(raw) != 4 {
		t.Errorf("%d clés, attendu 4 (SourceUnavailable n'est jamais sérialisé) : %s", len(raw), w.Body.String())
	}
	var body domain.MatchReplayContext
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("JSON typé : %v", err)
	}
	item := body.Media.MediaItems[0]
	if item.FilePath != "/api/v1/players/test-player/media/files/captures/clip.mp4" ||
		item.ThumbnailURL == nil || *item.ThumbnailURL != "/api/v1/players/test-player/media/files/captures/clip.webp" {
		t.Errorf("médias non réécrits : %+v", item)
	}
	if body.Header.T0Ms != 15_000 || len(body.Scoreboard) != 1 || len(body.HighlightEvents) != 1 {
		t.Errorf("corps : %+v", body)
	}
}

// TestReplayContextHandler_Refus : mêmes refus que la vue complète.
func TestReplayContextHandler_Refus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
		want string
	}{
		{"match inconnu", domain.ErrNotFound("match", "abc123"), http.StatusNotFound, "match_not_found"},
		{"participation refusée", &domain.APIError{Code: "match_not_participant", Message: "non"}, http.StatusNotFound, "match_not_participant"},
		{"panne", errors.New("verrou"), http.StatusInternalServerError, "replay_context_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := getReplayContext(t, &mockMatchViewService{err: tc.err})
			if w.Code != tc.code {
				t.Fatalf("statut %d, attendu %d : %s", w.Code, tc.code, w.Body.String())
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != tc.want {
				t.Errorf("code %q, attendu %q (%v) : %s", body.Code, tc.want, err, w.Body.String())
			}
		})
	}
}

// TestReplayContextHandler_NoStore : un contexte construit avec une lecture en échec, ou pendant
// l'analyse du film, n'est rejoué par aucun cache HTTP.
func TestReplayContextHandler_NoStore(t *testing.T) {
	for name, ctx := range map[string]domain.MatchReplayContext{
		"source en panne":  {SourceUnavailable: true},
		"analyse en cours": {Header: domain.MatchViewHeader{AnalysisPending: true}},
	} {
		t.Run(name, func(t *testing.T) {
			w := getReplayContext(t, &mockMatchViewService{replayCtx: ctx})
			if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("statut %d, Cache-Control %q, attendu 200 + no-store", w.Code, w.Header().Get("Cache-Control"))
			}
		})
	}
}
