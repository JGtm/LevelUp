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
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
)

// mockAssetService implémente port.AssetService pour les tests du handler.
type mockAssetService struct {
	maps    []canonical.AssetMeta
	weapons []canonical.AssetMeta
	err     error
}

func (m *mockAssetService) ListMaps(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.maps, m.err
}

func (m *mockAssetService) ListWeapons(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.weapons, m.err
}

func (m *mockAssetService) ListMedals(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.maps, m.err
}

func newAssetMetaRouter(svc *mockAssetService, capEnabled bool) *chi.Mux {
	h := handlers.NewAssetMetadataHandler(svc, func(_ string, _ titlePkg.Capability) bool {
		return capEnabled
	})
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func TestAssetMetadataHandler_ListMaps_OK(t *testing.T) {
	svc := &mockAssetService{
		maps: []canonical.AssetMeta{
			{ID: "map-001", NameEN: "Aquarius", NameFR: "Aquarius", ImageURL: "/api/v1/assets/maps/halo_infinite/map-001/image"},
			{ID: "map-002", NameEN: "Breaker", NameFR: "Breaker", ImageURL: "/api/v1/assets/maps/halo_infinite/map-002/image"},
		},
	}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/maps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var items []canonical.AssetMeta
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len=%d, want 2", len(items))
	}
	if items[0].ID != "map-001" {
		t.Errorf("ID=%q, want map-001", items[0].ID)
	}
}

func TestAssetMetadataHandler_ListMaps_EmptySearch_ReturnsArray(t *testing.T) {
	svc := &mockAssetService{maps: nil}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/maps?q=zzz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	var items []canonical.AssetMeta
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if items == nil {
		t.Error("attendu slice vide (non null), obtenu null")
	}
	if len(items) != 0 {
		t.Errorf("len=%d, want 0", len(items))
	}
}

func TestAssetMetadataHandler_ListMaps_CapabilityMissing_404(t *testing.T) {
	svc := &mockAssetService{}
	r := newAssetMetaRouter(svc, false)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo2/maps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("attendu 404 (capability absente), obtenu %d", w.Code)
	}
}

func TestAssetMetadataHandler_ListMaps_ServiceError_500(t *testing.T) {
	svc := &mockAssetService{err: errors.New("db fail")}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/maps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("attendu 500, obtenu %d", w.Code)
	}
}

func TestAssetMetadataHandler_ListWeapons_OK(t *testing.T) {
	svc := &mockAssetService{
		weapons: []canonical.AssetMeta{
			{ID: "100", NameEN: "BR75 Battle Rifle", NameFR: "Fusil BR75"},
		},
	}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/weapons", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d: %s", w.Code, w.Body.String())
	}
	var items []canonical.AssetMeta
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len=%d, want 1", len(items))
	}
	if items[0].NameEN != "BR75 Battle Rifle" {
		t.Errorf("NameEN=%q", items[0].NameEN)
	}
}

func TestAssetMetadataHandler_ListWeapons_CapabilityMissing_404(t *testing.T) {
	svc := &mockAssetService{}
	r := newAssetMetaRouter(svc, false)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo2/weapons", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("attendu 404 (capability absente), obtenu %d", w.Code)
	}
}

func TestAssetMetadataHandler_ListWeapons_ServiceError_500(t *testing.T) {
	svc := &mockAssetService{err: errors.New("db fail")}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/weapons", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("attendu 500, obtenu %d", w.Code)
	}
}

func TestAssetMetadataHandler_ListWeapons_EmptyResult_ReturnsArray(t *testing.T) {
	svc := &mockAssetService{weapons: nil}
	r := newAssetMetaRouter(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/weapons?q=zzz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d", w.Code)
	}
	var items []canonical.AssetMeta
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if items == nil {
		t.Error("attendu slice vide (non null), obtenu null")
	}
	if len(items) != 0 {
		t.Errorf("len=%d, want 0", len(items))
	}
}

func TestAssetMetadataHandler_QueryParam_Forwarded(t *testing.T) {
	svc := &captureSearchService{}
	h := handlers.NewAssetMetadataHandler(svc, func(_ string, _ titlePkg.Capability) bool { return true })
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/assets/halo_infinite/maps?q=aqu", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if svc.capturedSearch != "aqu" {
		t.Errorf("q=%q, want aqu", svc.capturedSearch)
	}
	if w.Code != http.StatusOK {
		t.Errorf("attendu 200, obtenu %d", w.Code)
	}
}

// captureSearchService capture le paramètre search transmis par le handler.
type captureSearchService struct {
	capturedSearch string
}

func (c *captureSearchService) ListMaps(_ context.Context, _, search string) ([]canonical.AssetMeta, error) {
	c.capturedSearch = search
	return []canonical.AssetMeta{{ID: "map-001", NameEN: "Aquarius"}}, nil
}

func (c *captureSearchService) ListWeapons(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}

func (c *captureSearchService) ListMedals(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, nil
}
