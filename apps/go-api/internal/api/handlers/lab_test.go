package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

type stubLabProvider struct {
	resources   *domain.LabResourcesResponse
	contracts   *domain.LabContractsResponse
	diagnostics *domain.LabDiagnosticsResponse
}

func (s *stubLabProvider) GetResources(
	_ context.Context,
	_ string,
	_ domain.LabResourcesQuery,
) (*domain.LabResourcesResponse, error) {
	return s.resources, nil
}

func (s *stubLabProvider) GetContracts(_ context.Context) (*domain.LabContractsResponse, error) {
	return s.contracts, nil
}

func (s *stubLabProvider) GetDiagnostics(
	_ context.Context,
	_ string,
) (*domain.LabDiagnosticsResponse, error) {
	return s.diagnostics, nil
}

var _ port.LabProvider = (*stubLabProvider)(nil)

func newLabHandlerForTest(t *testing.T, cfg *config.AppConfig) *handlers.LabHandler {
	t.Helper()
	provider := &stubLabProvider{
		resources: &domain.LabResourcesResponse{TitleSlug: testTitleSlug, MetadataDBPath: "metadata.duckdb"},
		contracts: &domain.LabContractsResponse{
			Summary: domain.LabOpenAPISummary{Status: "OK"},
		},
		diagnostics: &domain.LabDiagnosticsResponse{TitleSlug: testTitleSlug},
	}
	return handlers.NewLabHandler(service.NewLabService(cfg, provider))
}

func withTitle(req *http.Request) *http.Request {
	ctx := ctxkeys.WithTitleSlug(req.Context(), testTitleSlug)
	return req.WithContext(ctx)
}

func TestLabHandler_GetResources_OK(t *testing.T) {
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/resources?limit=5", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLabHandler_GetResources_InvalidMedalID(t *testing.T) {
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/resources?medal_id=oops", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLabHandler_GetContracts_OK(t *testing.T) {
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/contracts", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLabHandler_GetDiagnostics_OK(t *testing.T) {
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/diagnostics", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLabHandler_GetWaypoint_OK(t *testing.T) {
	explore := func(_ context.Context, q domain.LabWaypointQuery) (*domain.LabWaypointResponse, error) {
		return &domain.LabWaypointResponse{
			Segment: q.Segment, AssetID: q.AssetID, VersionID: q.VersionID,
			ResolvedOK: true, AssetName: "Live Fire",
		}, nil
	}
	// Routes Lab montées via Huma (h.Mount), comme les autres tests Lab post-migration
	// Phase 3b — l'ancien r.Get(h.GetWaypoint) chi-style n'existe plus.
	h := handlers.NewLabHandler(
		service.NewLabService(&config.AppConfig{}, &stubLabProvider{}).WithWaypointExplorer(explore))
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/waypoint?segment=map&asset_id=abc&version_id=1", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLabHandler_GetWaypoint_InvalidSegment(t *testing.T) {
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/waypoint?segment=bogus&asset_id=abc&version_id=1", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLabHandler_GetWaypoint_Unavailable(t *testing.T) {
	// Requête valide mais explorateur non câblé (newLabHandlerForTest n'injecte
	// pas WithWaypointExplorer) → 503.
	h := newLabHandlerForTest(t, &config.AppConfig{})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/waypoint?segment=map&asset_id=abc&version_id=1", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestLabHandler_Forbidden(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"can_manage_instance": false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newLabHandlerForTest(t, &config.AppConfig{AppSettingsPath: settingsPath})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/contracts", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
