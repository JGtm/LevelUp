package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

func TestAdminDBContentionHandler_Get(t *testing.T) {
	want := domain.DBContentionResponse{State: "ro", Swaps: 3, AvgAcquireMs: 12, ReadsRejected: 1}
	h := NewAdminDBContentionHandler(func(_ context.Context) domain.DBContentionResponse { return want })
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/db-contention", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got domain.DBContentionResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.State != "ro" || got.Swaps != 3 || got.AvgAcquireMs != 12 || got.ReadsRejected != 1 {
		t.Fatalf("got %+v", got)
	}
}
