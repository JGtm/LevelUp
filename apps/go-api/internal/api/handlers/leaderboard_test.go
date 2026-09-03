// Package handlers_test — leaderboard_test.go : contrat HTTP de la page Classement.
//
// Lot 4.1 (chantier « classement mondial : reprise du scrape ») : un GET sans
// saison ni playlist rendait un 500 `leaderboard_error` (l'erreur du repo
// remontait telle quelle). Un paramètre manquant n'est pas une panne serveur :
// le contrat est 200 + corps vide structuré. Test de bout en bout (routeur chi +
// Huma + VRAI LeaderboardService), pas un stub de service : c'est le câblage
// handler↔service qui produisait le 500.
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

// strictLeaderboardRepo reproduit le contrat INTERNE du repo DuckDB : le couple
// (saison, playlist) est obligatoire côté lecture, sinon erreur. C'est justement
// cette erreur qui ne doit plus atteindre la couche HTTP.
type strictLeaderboardRepo struct {
	entries []domain.LeaderboardEntry
	calls   int
}

func (r *strictLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func (r *strictLeaderboardRepo) GetCSRWorldLeaderboard(_ context.Context, _, season, playlist string, _ int) ([]domain.LeaderboardEntry, error) {
	r.calls++
	if season == "" || playlist == "" {
		return nil, errors.New("GetCSRWorldLeaderboard: season et playlist requis")
	}
	return r.entries, nil
}

func (r *strictLeaderboardRepo) GetStatLeaderboard(_ context.Context, _ string, _ domain.LeaderboardCategory, _, _ string, _ int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func (r *strictLeaderboardRepo) GetWorldLeaderboardCatalog(_ context.Context, _ string) (domain.LeaderboardCatalog, error) {
	return domain.LeaderboardCatalog{}, nil
}

func newLeaderboardRouter(repo port.LeaderboardRepository) *chi.Mux {
	factory := func(_ context.Context, _ string) (port.LeaderboardService, string, string, error) {
		return service.NewLeaderboardService(repo), testXUID1, testGamertag, nil
	}
	r := chi.NewRouter()
	h := handlers.NewLeaderboardHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func getLeaderboard(t *testing.T, r *chi.Mux, query string) (int, domain.LeaderboardResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/leaderboard"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body domain.LeaderboardResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("corps non décodable (%s): %v", w.Body.String(), err)
		}
	}
	return w.Code, body
}

// TestLeaderboardPage_MissingParams_200Empty : sans saison/playlist → 200 + vide,
// jamais 500. Les 3 combinaisons incomplètes sont couvertes.
func TestLeaderboardPage_MissingParams_200Empty(t *testing.T) {
	cases := []struct{ name, query string }{
		{"aucun paramètre", ""},
		{"saison seule", "?season=csrseason13-3"},
		{"playlist seule", "?playlist=edfef3ac-9cbe-4fa2-b949-8f29deafd483"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &strictLeaderboardRepo{}
			code, body := getLeaderboard(t, newLeaderboardRouter(repo), tc.query)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			if len(body.Entries) != 0 {
				t.Errorf("entries = %d, want 0", len(body.Entries))
			}
			if body.TotalLocal != 0 {
				t.Errorf("total = %d, want 0", body.TotalLocal)
			}
			if body.Category != string(domain.LeaderboardCSRWorld) {
				t.Errorf("category = %q, want csr-world", body.Category)
			}
			if repo.calls != 0 {
				t.Errorf("repo appelé %d fois alors que le couple est incomplet", repo.calls)
			}
		})
	}
}

// TestLeaderboardPage_CompleteParams_ServesEntries : le couple complet passe bien
// au repo — la garde 4.1 ne coupe pas le chemin nominal.
func TestLeaderboardPage_CompleteParams_ServesEntries(t *testing.T) {
	repo := &strictLeaderboardRepo{entries: []domain.LeaderboardEntry{
		{Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180},
	}}
	code, body := getLeaderboard(t, newLeaderboardRouter(repo),
		"?season=csrseason13-3&playlist=edfef3ac-9cbe-4fa2-b949-8f29deafd483")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if len(body.Entries) != 1 || body.Entries[0].Gamertag != "Twissted Mindss" {
		t.Fatalf("entries inattendues: %+v", body.Entries)
	}
	if body.TotalLocal != 1 {
		t.Errorf("total = %d, want 1", body.TotalLocal)
	}
	if repo.calls != 1 {
		t.Errorf("repo appelé %d fois, want 1", repo.calls)
	}
}

// TestLeaderboardPage_TotalFieldNameOnTheWire (Lot 4.4) : le compteur rempli par
// le service (domain.LeaderboardResponse.TotalLocal) voyage sous le nom `total` —
// il n'y a jamais eu de champ `total_local` sur le fil, le tag JSON est présent et
// sans omitempty. Ce test fige le nom : le contrat OpenAPI et generated.ts en
// dérivent, un renommage silencieux casserait le front sans erreur de compilation.
func TestLeaderboardPage_TotalFieldNameOnTheWire(t *testing.T) {
	repo := &strictLeaderboardRepo{entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "A", CSRValue: 1500}}}
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/leaderboard?season=csrseason13-3&playlist=pl-a", nil)
	w := httptest.NewRecorder()
	newLeaderboardRouter(repo).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("corps non décodable: %v", err)
	}
	if _, unexpected := raw["total_local"]; unexpected {
		t.Errorf("le corps expose `total_local` — le nom de fil attendu est `total`: %s", w.Body.String())
	}
	total, ok := raw["total"]
	if !ok {
		t.Fatalf("champ `total` absent du corps: %s", w.Body.String())
	}
	if total != float64(1) {
		t.Errorf("total = %v, want 1", total)
	}
}
