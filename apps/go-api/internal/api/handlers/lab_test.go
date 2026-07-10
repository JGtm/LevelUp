// Package handlers_test — lab_test.go : contrat HTTP du diagnostic d'instance
// (ex-Lab, A3.5/DC-9 : seul GET /lab/diagnostics survit).
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
	diagnostics *domain.LabDiagnosticsResponse
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
		diagnostics: &domain.LabDiagnosticsResponse{TitleSlug: testTitleSlug},
	}
	return handlers.NewLabHandler(service.NewLabService(cfg, provider))
}

func withTitle(req *http.Request) *http.Request {
	ctx := ctxkeys.WithTitleSlug(req.Context(), testTitleSlug)
	return req.WithContext(ctx)
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

func TestLabHandler_Forbidden(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"can_manage_instance": false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newLabHandlerForTest(t, &config.AppConfig{AppSettingsPath: settingsPath})
	r := chi.NewRouter()
	h.Mount(r)

	req := withTitle(httptest.NewRequest(http.MethodGet, "/lab/diagnostics", nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
