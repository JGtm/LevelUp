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
	"levelup/go-api/internal/service"
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
	owner := "test-player"
	svc := &mockMatchViewService{replayCtx: domain.MatchReplayContext{
		Header:          domain.MatchViewHeader{MatchID: "abc123", T0Ms: 15_000},
		Scoreboard:      []domain.MatchScoreboardRow{{XUID: "me", Gamertag: "Moi", IsMe: true}},
		HighlightEvents: []domain.MatchHighlightEvent{{EventType: "kill", ActorGamertag: &gt}},
		Media: domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{
			{FileID: "f1", FilePath: "captures/clip.mp4", ThumbnailURL: &thumb, OwnerGamertag: &owner},
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
	for _, key := range []string{"header", "scoreboard", "highlight_events", "media", "source_unavailable"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("clé %q absente : %s", key, w.Body.String())
		}
	}
	if len(raw) != 5 {
		t.Errorf("%d clés, attendu 5 : %s", len(raw), w.Body.String())
	}
	if string(raw["source_unavailable"]) != "false" {
		t.Errorf("source_unavailable = %s sur un contexte complet, attendu false", raw["source_unavailable"])
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
// l'analyse du film, n'est rejoué par aucun cache HTTP ; la panne est dite au client
// (`source_unavailable`, qui pilote son nouvel essai toutes les 30 s), l'analyse non.
func TestReplayContextHandler_NoStore(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx             domain.MatchReplayContext
		wantUnavailable string
	}{
		"source en panne":  {ctx: domain.MatchReplayContext{SourceUnavailable: true}, wantUnavailable: "true"},
		"analyse en cours": {ctx: domain.MatchReplayContext{Header: domain.MatchViewHeader{AnalysisPending: true}}, wantUnavailable: "false"},
	} {
		t.Run(name, func(t *testing.T) {
			w := getReplayContext(t, &mockMatchViewService{replayCtx: tc.ctx})
			if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("statut %d, Cache-Control %q, attendu 200 + no-store", w.Code, w.Header().Get("Cache-Control"))
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
				t.Fatalf("JSON : %v", err)
			}
			if got := string(raw["source_unavailable"]); got != tc.wantUnavailable {
				t.Errorf("source_unavailable = %q, attendu %q : %s", got, tc.wantUnavailable, w.Body.String())
			}
		})
	}
}

// repoTableauEnPanne : le repo de bout en bout de la vue (repoDuelsEnPanne), dont la lecture
// du tableau de score — source à raison dédiée, chargeur `goLoad` — échoue.
type repoTableauEnPanne struct {
	repoDuelsEnPanne
	boardErr error
}

func (r *repoTableauEnPanne) GetMatchScoreboard(ctx context.Context, id string) ([]domain.ScoreboardRaw, error) {
	if r.boardErr != nil {
		return nil, r.boardErr
	}
	return r.repoDuelsEnPanne.GetMatchScoreboard(ctx, id)
}

// TestReplayContextHandler_TableauDeScoreEnPanne_NoStore : par le vrai service et le vrai
// handler, une lecture du tableau de score en échec donne 200, `source_unavailable` vrai et
// `no-store` ; la source rétablie, la réponse suivante est complète et sans en-tête de cache.
func TestReplayContextHandler_TableauDeScoreEnPanne_NoStore(t *testing.T) {
	repo := &repoTableauEnPanne{boardErr: errors.New("tableau HS")}
	factory := func(context.Context, string) (port.MatchViewService, error) {
		return service.NewMatchViewService(repo, "xuid1"), nil
	}
	get := func() (*httptest.ResponseRecorder, map[string]json.RawMessage) {
		r := chi.NewRouter()
		h := handlers.NewMatchViewHandler(factory)
		r.Route("/players/{player_slug}", func(r chi.Router) { h.MountReplayContext(r) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m1/replay/context", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("200 attendu, obtenu %d : %s", w.Code, w.Body.String())
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
			t.Fatalf("JSON : %v", err)
		}
		return w, raw
	}

	w, raw := get()
	if string(raw["source_unavailable"]) != "true" || w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("panne : source_unavailable %s, Cache-Control %q — attendu true + no-store",
			raw["source_unavailable"], w.Header().Get("Cache-Control"))
	}

	repo.boardErr = nil
	w, raw = get()
	if string(raw["source_unavailable"]) != "false" || w.Header().Get("Cache-Control") != "" {
		t.Errorf("rétabli : source_unavailable %s, Cache-Control %q — attendu false, aucun en-tête",
			raw["source_unavailable"], w.Header().Get("Cache-Control"))
	}
}
