package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

type mockCatalogRepo struct {
	playlists map[string][]domain.CatalogPlaylist // titleSlug → playlists
	pairs     map[string][]domain.CatalogPair     // playlistAssetID → pairs
	maps      map[string][]domain.CatalogMap
}

func (m *mockCatalogRepo) PlaylistsByTitle(_ context.Context, titleSlug, _ string, _ bool) ([]domain.CatalogPlaylist, error) {
	return m.playlists[titleSlug], nil
}
func (m *mockCatalogRepo) PairsByPlaylist(_ context.Context, _, playlistID string) ([]domain.CatalogPair, error) {
	return m.pairs[playlistID], nil
}
func (m *mockCatalogRepo) MapsByTitle(_ context.Context, titleSlug, _ string, _ bool) ([]domain.CatalogMap, error) {
	return m.maps[titleSlug], nil
}
func (m *mockCatalogRepo) CountCatalogEntries(_ context.Context, titleSlug string) (int, error) {
	return len(m.playlists[titleSlug]), nil
}

func TestCatalogHandler_Playlists(t *testing.T) {
	repo := &mockCatalogRepo{
		playlists: map[string][]domain.CatalogPlaylist{
			"halo_infinite": {
				{TitleSlug: "halo_infinite", PlaylistAssetID: "pl-1", Name: "Quick Play", Experience: "social"},
				{TitleSlug: "halo_infinite", PlaylistAssetID: "pl-2", Name: "Ranked Arena", Experience: "ranked", IsRanked: true},
			},
		},
	}
	h := NewCatalogHandler(repo)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/titles/halo_infinite/catalog/playlists?only_played=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}

	var body struct {
		TitleSlug string                   `json:"title_slug"`
		Playlists []domain.CatalogPlaylist `json:"playlists"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.TitleSlug != "halo_infinite" {
		t.Errorf("title_slug = %q", body.TitleSlug)
	}
	if len(body.Playlists) != 2 {
		t.Errorf("playlists len = %d, want 2", len(body.Playlists))
	}
}

func TestCatalogHandler_Pairs(t *testing.T) {
	repo := &mockCatalogRepo{
		pairs: map[string][]domain.CatalogPair{
			"pl-1": {
				{TitleSlug: "halo_infinite", PairAssetID: "pa-1", Name: "Slayer on Bazaar", ModeCategory: "Assassin"},
			},
		},
	}
	h := NewCatalogHandler(repo)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/titles/halo_infinite/catalog/pairs?playlist_asset_id=pl-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var body struct {
		Pairs []domain.CatalogPair `json:"pairs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Pairs) != 1 {
		t.Errorf("pairs len = %d, want 1", len(body.Pairs))
	}
	if body.Pairs[0].ModeCategory != "Assassin" {
		t.Errorf("mode_category = %q", body.Pairs[0].ModeCategory)
	}
}

func TestCatalogHandler_Maps(t *testing.T) {
	repo := &mockCatalogRepo{
		maps: map[string][]domain.CatalogMap{
			"halo_infinite": {
				{TitleSlug: "halo_infinite", MapAssetID: "m-1", Name: "Bazaar"},
			},
		},
	}
	h := NewCatalogHandler(repo)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/titles/halo_infinite/catalog/maps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var body struct {
		Maps []domain.CatalogMap `json:"maps"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if len(body.Maps) != 1 || body.Maps[0].Name != "Bazaar" {
		t.Errorf("maps = %+v", body.Maps)
	}
}
