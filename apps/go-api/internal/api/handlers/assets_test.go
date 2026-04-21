package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
)

// stubResolver implements assets.Resolver for tests.
type stubResolver struct {
	result assets.Resolved
	err    error
	gotRef assets.Ref
}

func (s *stubResolver) Get(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
	s.gotRef = ref
	return s.result, s.err
}

func (s *stubResolver) Refresh(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
	return s.result, s.err
}

func (s *stubResolver) Warm(_ context.Context, _ ...assets.Ref) {}
func (s *stubResolver) RegisterLocalFile(_ context.Context, _ assets.Ref, _ string) error {
	return nil
}
func (s *stubResolver) Close(_ context.Context) error { return nil }

var _ assets.Resolver = (*stubResolver)(nil)
var _ = errors.Is(nil, assets.ErrNotFound)

func newMedalTestRouter(h *handlers.AssetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/assets/medals/{title_id}/{medal_id}/image", h.GetMedalImage)
	return r
}

func TestMedalImageHandler_URLPayload_Redirect(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.URLPayload{URL: "https://example.com/medal.png"}},
	}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/12345/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("URLPayload: attendu 302, obtenu %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/medal.png" {
		t.Errorf("URLPayload: Location=%q", loc)
	}
	if stub.gotRef.Kind != assets.KindMedalImage {
		t.Errorf("Ref.Kind=%v, attendu KindMedalImage", stub.gotRef.Kind)
	}
}

func TestMedalImageHandler_BinaryPayload_Serve(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.BinaryPayload{
			ContentType: "image/png",
			Bytes:       []byte{0x89, 0x50, 0x4e, 0x47},
			ETag:        "abc123",
		}},
	}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/12345/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("BinaryPayload: attendu 200, obtenu %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type=%q", ct)
	}
}

func TestMedalImageHandler_InvalidMedalID(t *testing.T) {
	stub := &stubResolver{}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/not-a-number/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("medal_id invalide: attendu 400, obtenu %d", w.Code)
	}
}

func TestMedalImageHandler_NotFound(t *testing.T) {
	stub := &stubResolver{err: assets.ErrNotFound}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/99999/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrNotFound: attendu 404, obtenu %d", w.Code)
	}
}

func TestMedalImageHandler_Upstream(t *testing.T) {
	stub := &stubResolver{err: assets.ErrUpstreamUnavailable}
	r := newMedalTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/medals/halo_infinite/99999/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("ErrUpstream: attendu 502, obtenu %d", w.Code)
	}
}

func newMapTestRouter(h *handlers.AssetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/assets/maps/{title_id}/{map_id}/image", h.GetMapImage)
	return r
}

func TestMapImageHandler_URLPayload_Redirect(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.URLPayload{URL: "https://example.com/map.png"}},
	}
	r := newMapTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/maps/halo_infinite/aquarius/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("URLPayload: attendu 302, obtenu %d", w.Code)
	}
	if stub.gotRef.Kind != assets.KindMapImage {
		t.Errorf("Ref.Kind=%v, attendu KindMapImage", stub.gotRef.Kind)
	}
	if stub.gotRef.ID != "aquarius" {
		t.Errorf("Ref.ID=%q, attendu aquarius", stub.gotRef.ID)
	}
}

func TestMapImageHandler_NotFound(t *testing.T) {
	stub := &stubResolver{err: assets.ErrNotFound}
	r := newMapTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/maps/halo_infinite/unknown_map/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("ErrNotFound: attendu 404, obtenu %d", w.Code)
	}
}

func newBPTestRouter(h *handlers.AssetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/assets/battlepass/{subdir}/*", h.GetBattlePassImage)
	return r
}

func TestBattlePassImage_URLPayload(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.URLPayload{URL: "https://example.com/bp.png"}},
	}
	r := newBPTestRouter(handlers.NewAssetHandler(stub))
	req := httptest.NewRequest(http.MethodGet, "/assets/battlepass/track/season1/track.png", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("BP URLPayload: attendu 302, obtenu %d", w.Code)
	}
}

func TestChallengeBadgeHandler_URLPayload(t *testing.T) {
	stub := &stubResolver{
		result: assets.Resolved{Payload: assets.URLPayload{URL: "https://example.com/badge.png"}},
	}
	r := chi.NewRouter()
	r.Get("/assets/challenge-badge/{title_id}/{badge_id}", handlers.NewAssetHandler(stub).GetChallengeBadge)
	req := httptest.NewRequest(http.MethodGet, "/assets/challenge-badge/halo_infinite/daily-easy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("badge URLPayload: attendu 302, obtenu %d", w.Code)
	}
	if stub.gotRef.Kind != assets.KindChallengeBadge {
		t.Errorf("Ref.Kind=%v, attendu KindChallengeBadge", stub.gotRef.Kind)
	}
	if stub.gotRef.ID != "daily-easy" {
		t.Errorf("Ref.ID=%q, attendu daily-easy", stub.gotRef.ID)
	}
}
