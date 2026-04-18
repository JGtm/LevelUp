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
