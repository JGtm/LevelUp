package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

func TestAdminInvariantsHandler_Get_OK(t *testing.T) {
	run := func(_ context.Context, titleSlug string) (domain.AdminInvariantsResponse, error) {
		return domain.AdminInvariantsResponse{
			TitleSlug:   titleSlug,
			GeneratedAt: "2026-06-10T18:00:00Z",
			Reports: []domain.PlayerInvariantsReport{
				{
					PlayerSlug: "jgtm",
					Gamertag:   "JGtm",
					XUID:       "123",
					Violations: []domain.InvariantViolation{
						{Key: "psa_missing", Severity: "warn", Count: 2, Sample: []string{"m1", "m2"}, Description: "d"},
					},
					WarnCount: 1,
				},
			},
		}, nil
	}
	h := NewAdminInvariantsHandler(run)
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/invariants", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got domain.AdminInvariantsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Sans ?title= → slug par défaut transmis au runner.
	if got.TitleSlug == "" {
		t.Error("TitleSlug vide — le défaut doit être transmis au runner")
	}
	if len(got.Reports) != 1 || got.Reports[0].WarnCount != 1 {
		t.Errorf("reports inattendus : %+v", got.Reports)
	}
}

func TestAdminInvariantsHandler_Get_RunnerError(t *testing.T) {
	run := func(_ context.Context, _ string) (domain.AdminInvariantsResponse, error) {
		return domain.AdminInvariantsResponse{}, errors.New("boom")
	}
	h := NewAdminInvariantsHandler(run)
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/invariants?title=halo_infinite", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d (attendu 500)", rec.Code)
	}
}
