// Package handlers_test — media_audience_test.go : l'audience des médias (auteurs
// visibles) appliquée par la galerie, la liste des auteurs, les fichiers et l'onglet
// Médias d'un match.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// staticAudience rend toujours la même audience, quelle que soit la page.
func staticAudience(names ...string) handlers.MediaAudienceResolver {
	return func(context.Context, string) (domain.MediaAudience, error) {
		return domain.NewMediaAudience(names...), nil
	}
}

func mediaRouterWith(mock *mockMediaService, resolve handlers.MediaAudienceResolver) *chi.Mux {
	factory := func(context.Context, string) (port.MediaService, error) { return mock, nil }
	h := handlers.NewMediaHandler(factory, nil, "").WithMediaAudience(resolve)
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

func TestMediaAudience_GalleryPassesAudienceToService(t *testing.T) {
	mock := &mockMediaService{page: &domain.MediaPageResponse{}}
	r := mediaRouterWith(mock, staticAudience(testPlayerSlug, "Madina"))
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/media",
		strings.NewReader(`{"author_slugs":["Etranger"]}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	if mock.pageReq == nil || mock.pageReq.Audience == nil {
		t.Fatal("audience absente de la requête transmise au service")
	}
	a := mock.pageReq.Audience
	if !a.Allows("madina") || !a.Allows(testPlayerSlug) || a.Allows("Etranger") {
		t.Fatalf("audience = %v", a.Authors())
	}
}

func TestMediaAudience_AuthorsFiltered(t *testing.T) {
	authors := []domain.MediaAuthor{
		{PlayerSlug: testPlayerSlug, IsSelf: true}, {PlayerSlug: "Madina"}, {PlayerSlug: "Etranger"},
	}
	cases := []struct {
		name    string
		resolve handlers.MediaAudienceResolver
		want    []string
	}{
		{"groupe", staticAudience(testPlayerSlug, "Madina"), []string{testPlayerSlug, "Madina"}},
		{"sans résolveur : la page seule", nil, []string{testPlayerSlug}},
		{"résolveur en échec : la page seule", func(context.Context, string) (domain.MediaAudience, error) {
			return domain.MediaAudience{}, errors.New("groups.json illisible")
		}, []string{testPlayerSlug}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := mediaRouterWith(&mockMediaService{authors: authors}, tc.resolve)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/media/authors", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("statut %d : %s", w.Code, w.Body.String())
			}
			var resp domain.MediaAuthorsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(resp.Authors))
			for _, a := range resp.Authors {
				got = append(got, a.PlayerSlug)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("auteurs = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("auteurs = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// TestMediaAudience_ServeFileRefusesOutsider : le fichier d'un auteur hors audience ne
// se sert pas par la racine commune, celui d'un co-membre si.
func TestMediaAudience_ServeFileRefusesOutsider(t *testing.T) {
	base := t.TempDir()
	for _, owner := range []string{"Madina", "Etranger"} {
		if err := os.MkdirAll(filepath.Join(base, owner), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, owner, "shot.png"), []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	factory := func(context.Context, string) (port.MediaService, error) { return nil, nil }
	h := handlers.NewMediaHandler(factory, nil, "").
		WithSettingsStore(writeSettingsJSON(t, base)).
		WithMediaAudience(staticAudience(testPlayerSlug, "Madina"))
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { r.Get("/media/files/*", h.ServeMediaFile) })

	for owner, want := range map[string]int{"Madina": http.StatusOK, "Etranger": http.StatusNotFound} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/media/files/"+owner+"/shot.png", nil))
		if w.Code != want {
			t.Errorf("%s : statut %d, want %d", owner, w.Code, want)
		}
	}
}

func TestMediaAudience_MatchViewDropsOutsiders(t *testing.T) {
	mine, friend, stranger := testPlayerSlug, "Madina", "Etranger"
	resp := domain.MatchViewResponse{
		Header: domain.MatchViewHeader{MatchID: "m1"},
		MediaTab: domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{
			{FileID: "1", FilePath: "a.png", OwnerGamertag: &mine},
			{FileID: "2", FilePath: "b.png", OwnerGamertag: &friend},
			{FileID: "3", FilePath: "c.png", OwnerGamertag: &stranger},
			{FileID: "4", FilePath: "d.png"},
		}},
	}
	factory := func(context.Context, string) (port.MatchViewService, error) {
		return &mockMatchViewService{resp: resp}, nil
	}
	h := handlers.NewMatchViewHandler(factory).WithMediaAudience(staticAudience(testPlayerSlug, "Madina"))
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	var got domain.MatchViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, it := range got.MediaTab.MediaItems {
		ids = append(ids, it.FileID)
	}
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Fatalf("médias = %v, want [1 2]", ids)
	}
}
