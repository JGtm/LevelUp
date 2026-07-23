// Package handlers — admin_appearance_diag_test.go : contrats HTTP du diagnostic
// apparence (Lot F, F5). 200 nominal, 404 slug inconnu (sentinel
// service.ErrProfileNotFound). Montage Huma sous /admin, comme la prod.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

func serveAppearanceDiag(h *AdminAppearanceDiagHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestAdminAppearanceDiag_OK : 200 + payload du runner.
func TestAdminAppearanceDiag_OK(t *testing.T) {
	h := NewAdminAppearanceDiagHandler(func(_ context.Context, slug string) (domain.AppearanceDiagnosisResponse, error) {
		return domain.AppearanceDiagnosisResponse{
			PlayerSlug: slug,
			Gamertag:   "JGtm",
			XUID:       "2535",
			TitleSlug:  "halo_infinite",
			Components: []domain.AppearanceComponentDiagnosis{
				{Component: "banner", Verdict: "upstream_missing", ServedFrom: "carry"},
			},
		}, nil
	})
	rec := serveAppearanceDiag(h, httptest.NewRequest(http.MethodGet, "/admin/diag/appearance/JGtm", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var got domain.AppearanceDiagnosisResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.PlayerSlug != "JGtm" || len(got.Components) != 1 || got.Components[0].Verdict != "upstream_missing" {
		t.Fatalf("payload inattendu : %+v", got)
	}
}

// TestAdminAppearanceDiag_NotFound : sentinel ErrProfileNotFound → 404.
func TestAdminAppearanceDiag_NotFound(t *testing.T) {
	h := NewAdminAppearanceDiagHandler(func(_ context.Context, slug string) (domain.AppearanceDiagnosisResponse, error) {
		return domain.AppearanceDiagnosisResponse{}, fmt.Errorf("%w: slug=%q", service.ErrProfileNotFound, slug)
	})
	rec := serveAppearanceDiag(h, httptest.NewRequest(http.MethodGet, "/admin/diag/appearance/Inconnu", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d (attendu 404) body=%s", rec.Code, rec.Body.String())
	}
}
