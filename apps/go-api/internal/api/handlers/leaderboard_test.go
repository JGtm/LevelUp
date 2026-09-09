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
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// strictLeaderboardRepo reproduit le contrat INTERNE du repo DuckDB : le couple
// (saison, playlist) est obligatoire côté lecture, sinon erreur. C'est justement
// cette erreur qui ne doit plus atteindre la couche HTTP.
//
// Sa valeur ZÉRO sert aussi de fixture « repo sans aucune ligne » : entries nil et
// catalogue à la valeur zéro, exactement ce que produit un scan DuckDB vide (cf.
// TestLeaderboardPage_EmptyCollectionsOnTheWire).
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

// newLeaderboardRouter monte les routes du classement. Les middlewares optionnels
// sont installés AVANT les routes (contrainte chi) ; ils servent à reproduire ce
// que fait middleware.TitleExtractor en production, cf. withTitleSlug.
func newLeaderboardRouter(repo port.LeaderboardRepository, mw ...func(http.Handler) http.Handler) *chi.Mux {
	factory := func(_ context.Context, _ string) (port.LeaderboardService, string, string, error) {
		return service.NewLeaderboardService(repo), testXUID1, testGamertag, nil
	}
	r := chi.NewRouter()
	for _, m := range mw {
		r.Use(m)
	}
	h := handlers.NewLeaderboardHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

// withTitleSlug reproduit ce qu'injecte middleware.TitleExtractor. Sans lui,
// ctxkeys.TitleSlug retombe sur « halo_infinite » et le chemin « titre sans la
// capability » de GetCatalog — qui lit le titre dans le CONTEXTE, et non dans un
// query param comme GetPage — reste inatteignable depuis un test HTTP.
func withTitleSlug(slug string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ctxkeys.WithTitleSlug(r.Context(), slug)))
		})
	}
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

// TestLeaderboardPage_EmptyCollectionsOnTheWire (constat du lot 4) : `entries`,
// `seasons` et `playlists` sont des champs SANS omitempty — le contrat promet que
// le champ est présent, donc `[]` et jamais `null`. Un scan DuckDB sans aucune
// ligne rend une slice Go NIL : le ratchet côté service
// (TestDTOs_NoNilSlicesOnEmptyInput) n'exerçait que la sortie anticipée « couple
// incomplet » de GetPage, laissant sans garde le chemin NOMINAL (le service
// affectait la slice nil du repo par-dessus sa garantie de construction) ainsi
// que tout GetCatalog.
//
// Ce test lit le JSON RÉELLEMENT émis, pas la structure : c'est la sérialisation
// qui transforme un nil en `null`, et c'est ce `null` qui atteint le frontend.
func TestLeaderboardPage_EmptyCollectionsOnTheWire(t *testing.T) {
	const pageWithCouple = "/players/test-player/pages/leaderboard?season=csrseason13-3&playlist=pl-a"
	const catalogPath = "/players/test-player/pages/leaderboard/catalog"

	cases := []struct {
		name string
		// ctxTitleSlug : titre injecté dans le CONTEXTE (vide = pas de middleware,
		// donc ctxkeys retombe sur halo_infinite). Seul GetCatalog le lit ; GetPage
		// prend le sien dans le query param `title_slug`.
		ctxTitleSlug string
		path         string
		fields       []string
	}{
		{
			name:   "page servie, repo sans aucune ligne",
			path:   pageWithCouple,
			fields: []string{"entries"},
		},
		{
			name:   "catalogue sans aucun snapshot",
			path:   catalogPath,
			fields: []string{"seasons", "playlists"},
		},
		{
			// Le scénario que le constat du lot 4 NOMME, exercé sur le fil.
			name:   "page, titre sans la capability",
			path:   pageWithCouple + "&title_slug=unknown_title_no_cap",
			fields: []string{"entries"},
		},
		{
			name:         "catalogue, titre sans la capability",
			ctxTitleSlug: "unknown_title_no_cap",
			path:         catalogPath,
			fields:       []string{"seasons", "playlists"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Repo sans données : entries nil ET un catalogue à la valeur zéro (slices nil).
			var mw []func(http.Handler) http.Handler
			if tc.ctxTitleSlug != "" {
				mw = append(mw, withTitleSlug(tc.ctxTitleSlug))
			}
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			newLeaderboardRouter(&strictLeaderboardRepo{}, mw...).ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
				t.Fatalf("corps non décodable: %v", err)
			}
			for _, field := range tc.fields {
				v, ok := raw[field]
				if !ok {
					t.Errorf("champ %q absent du corps alors qu'il n'a pas d'omitempty: %s", field, w.Body.String())
					continue
				}
				if string(v) != "[]" {
					t.Errorf("%s = %s, want [] (un `null` casse le consommateur typé)", field, v)
				}
			}
		})
	}
}
