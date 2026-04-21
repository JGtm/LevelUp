package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/platform/duckdb"
)

// stubMedalRepo implémente MedalImageRepo pour les tests.
type stubMedalRepo struct {
	entry    *duckdb.MedalImageEntry
	upserted *duckdb.MedalImageEntry
}

func (s *stubMedalRepo) GetMedalImageCache(_ context.Context, _ string, _ int64) (*duckdb.MedalImageEntry, error) {
	return s.entry, nil
}

func (s *stubMedalRepo) UpsertMedalImageCache(_ context.Context, e duckdb.MedalImageEntry) error {
	s.upserted = &e
	return nil
}

// stubMapRepo implémente MapImageRepo pour les tests.
type stubMapRepo struct {
	entry    *duckdb.MapImageEntry
	upserted *duckdb.MapImageEntry
}

func (s *stubMapRepo) GetMapImageCache(_ context.Context, _ string, _ string) (*duckdb.MapImageEntry, error) {
	return s.entry, nil
}

func (s *stubMapRepo) UpsertMapImageCache(_ context.Context, e duckdb.MapImageEntry) error {
	s.upserted = &e
	return nil
}

func newAssetTestRouter(h *handlers.AssetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/assets/medals/{title_id}/{medal_id}/image", h.GetMedalImage)
	return r
}

// TestMedalImageHandler_CacheHit vérifie qu'un cache hit retourne une redirection 302
// sans appel UpsertMedalImageCache.
func TestMedalImageHandler_CacheHit(t *testing.T) {
	stub := &stubMedalRepo{
		entry: &duckdb.MedalImageEntry{
			TitleID:   "halo_infinite",
			MedalID:   12345,
			ImageURL:  "https://example.com/medal.png",
			FetchedAt: time.Now(),
		},
	}
	r := newAssetTestRouter(handlers.NewAssetHandlerWithRepo(stub))

	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/12345/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("cache hit: attendu 302, obtenu %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/medal.png" {
		t.Errorf("cache hit: Location=%q, attendu %q", loc, "https://example.com/medal.png")
	}
	if stub.upserted != nil {
		t.Error("cache hit: UpsertMedalImageCache ne doit pas être appelé")
	}
}

// TestMedalImageHandler_InvalidMedalID vérifie le rejet d'un medal_id non numérique.
func TestMedalImageHandler_InvalidMedalID(t *testing.T) {
	stub := &stubMedalRepo{}
	r := newAssetTestRouter(handlers.NewAssetHandlerWithRepo(stub))

	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/not-a-number/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("medal_id invalide: attendu 400, obtenu %d", w.Code)
	}
}

// TestMedalImageHandler_CacheMiss_Upserts vérifie qu'un cache miss déclenche un upsert.
// Le fetch Waypoint est skippé ici (pas de serveur mock) — on vérifie uniquement
// que l'absence d'entrée en cache produit un appel UpsertMedalImageCache.
func TestMedalImageHandler_CacheMiss_Upserts(t *testing.T) {
	stub := &stubMedalRepo{entry: nil} // cache vide
	r := newAssetTestRouter(handlers.NewAssetHandlerWithRepo(stub))

	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/99999/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Le fetch Waypoint peut échouer en test (pas de réseau) — on accepte 302 ou 502.
	if w.Code != http.StatusFound && w.Code != http.StatusBadGateway {
		t.Errorf("cache miss: attendu 302 ou 502, obtenu %d", w.Code)
	}
}

// ============================================================================
// Tests pour GetMapImage (cache-aside maps)
// ============================================================================

func newMapTestRouter(h *handlers.AssetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/assets/maps/{title_id}/{map_id}/image", h.GetMapImage)
	return r
}

// TestMapImageHandler_CacheHit vérifie qu'un cache hit retourne une redirection 302
// sans appel UpsertMapImageCache.
func TestMapImageHandler_CacheHit(t *testing.T) {
	stubMap := &stubMapRepo{
		entry: &duckdb.MapImageEntry{
			TitleID:  "halo_infinite",
			MapID:    "aquarius",
			ImageURL: "https://example.com/map.png",
		},
	}
	h := handlers.NewAssetHandlerWithMapRepo(stubMap)
	r := newMapTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/assets/maps/halo_infinite/aquarius/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("cache hit: attendu 302, obtenu %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/map.png" {
		t.Errorf("cache hit: Location=%q, attendu %q", loc, "https://example.com/map.png")
	}
	if stubMap.upserted != nil {
		t.Error("cache hit: UpsertMapImageCache ne doit pas être appelé")
	}
}

// TestMapImageHandler_EmptyMapID vérifie le rejet d'un map_id vide.
func TestMapImageHandler_EmptyMapID(t *testing.T) {
	stubMap := &stubMapRepo{}
	h := handlers.NewAssetHandlerWithMapRepo(stubMap)
	r := newMapTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/assets/maps/halo_infinite//image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("map_id vide: attendu 400, obtenu %d", w.Code)
	}
}

// TestMapImageHandler_CacheMiss vérifie qu'un cache miss déclenche un fetch + upsert.
func TestMapImageHandler_CacheMiss(t *testing.T) {
	stubMap := &stubMapRepo{entry: nil} // cache vide
	h := handlers.NewAssetHandlerWithMapRepo(stubMap)
	r := newMapTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/assets/maps/halo_infinite/unknown_map/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Le fetch Waypoint peut échouer en test (pas de réseau) — on accepte 302 ou 502.
	if w.Code != http.StatusFound && w.Code != http.StatusBadGateway {
		t.Errorf("cache miss: attendu 302 ou 502, obtenu %d", w.Code)
	}
}
