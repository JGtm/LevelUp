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

func TestAdminTokenHealthHandler_Get_OK(t *testing.T) {
	want := domain.TokenHealthResponse{
		GeneratedAt: "2026-06-10T00:00:00Z",
		Players: []domain.PlayerTokenHealth{
			{Gamertag: "GT", Refresh: "ok", Access: "expiring", XSTS: "ok"},
		},
	}
	h := NewAdminTokenHealthHandler(func(_ context.Context, _ string) (domain.TokenHealthResponse, error) {
		return want, nil
	})
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/token-health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got domain.TokenHealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Players) != 1 || got.Players[0].Access != "expiring" {
		t.Fatalf("got %+v", got)
	}
}

func TestAdminTokenHealthHandler_Get_Error(t *testing.T) {
	h := NewAdminTokenHealthHandler(func(_ context.Context, _ string) (domain.TokenHealthResponse, error) {
		return domain.TokenHealthResponse{}, errors.New("boom")
	})
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { h.Mount(r) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/token-health", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
