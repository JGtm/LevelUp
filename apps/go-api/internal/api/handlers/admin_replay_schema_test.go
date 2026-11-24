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

func TestAdminReplaySchemaHandler_Get_OK(t *testing.T) {
	want := domain.AdminReplaySchema{
		GeneratedAt:   "2026-11-24T10:00:00Z",
		CurrentSchema: 127,
		Titles:        []domain.AdminReplaySchemaTitle{{TitleSlug: "halo_infinite", Total: 3, Current: 1, Behind: 1, Unreadable: 1}},
	}
	h := NewAdminReplaySchemaHandler(func(context.Context) (domain.AdminReplaySchema, error) { return want, nil })
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/monitoring/replay-schema", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got domain.AdminReplaySchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.CurrentSchema != 127 || len(got.Titles) != 1 || got.Titles[0].Behind != 1 || got.Titles[0].Unreadable != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestAdminReplaySchemaHandler_Get_Error(t *testing.T) {
	h := NewAdminReplaySchemaHandler(func(context.Context) (domain.AdminReplaySchema, error) {
		return domain.AdminReplaySchema{}, errors.New("boom")
	})
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/monitoring/replay-schema", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
