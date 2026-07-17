// Package handlers — diag_prestige_telemetry_test.go : couverture handler
// /_diag/prestige/telemetry. Provider mock pour isoler la logique HTTP.
//
// ADR 0020.
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

type stubPrestigeTelemetryDiagProvider struct {
	diag *domain.PrestigeTelemetryDiag
	err  error
}

func (s *stubPrestigeTelemetryDiagProvider) GetPrestigeTelemetryDiag(_ context.Context, slug string) (*domain.PrestigeTelemetryDiag, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.diag == nil {
		return nil, nil
	}
	out := *s.diag
	out.PlayerSlug = slug
	return &out, nil
}

func servePrestigeTelemetryDiag(t *testing.T, slug string, factory PrestigeTelemetryDiagFactory) *httptest.ResponseRecorder {
	t.Helper()
	h := NewDiagPrestigeTelemetryHandler(factory)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/_diag/prestige/telemetry/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestDiagPrestigeTelemetryHandler_BySource : agrégats par origine renvoyés en 200.
func TestDiagPrestigeTelemetryHandler_BySource(t *testing.T) {
	diag := &domain.PrestigeTelemetryDiag{
		TotalEvents: 9,
		BySource: []domain.PrestigeTelemetrySourceStats{
			{Source: "coach", Created: 3, Completed: 2, CompletionRate: 2.0 / 3.0, AcceptanceRate: 1, AbandonRate: 0},
			{Source: "unknown", Created: 2, Completed: 0, CompletionRate: 0, AcceptanceRate: 1, AbandonRate: 0.5, Abandoned: 1},
		},
	}
	factory := func(ctx context.Context, slug string) (PrestigeTelemetryDiagProvider, error) {
		return &stubPrestigeTelemetryDiagProvider{diag: diag}, nil
	}
	w := servePrestigeTelemetryDiag(t, "JGtm", factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got domain.PrestigeTelemetryDiag
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.PlayerSlug != "JGtm" {
		t.Errorf("slug: got %q want JGtm", got.PlayerSlug)
	}
	if len(got.BySource) != 2 || got.BySource[0].Source != "coach" {
		t.Fatalf("by_source mismatch: %+v", got.BySource)
	}
	if got.BySource[0].Completed != 2 {
		t.Errorf("coach completed: got %d want 2", got.BySource[0].Completed)
	}
}

// TestDiagPrestigeTelemetryHandler_PlayerNotFound : factory err → 404.
func TestDiagPrestigeTelemetryHandler_PlayerNotFound(t *testing.T) {
	factory := func(context.Context, string) (PrestigeTelemetryDiagProvider, error) {
		return nil, errors.New("player_not_found")
	}
	w := servePrestigeTelemetryDiag(t, "Unknown", factory)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestDiagPrestigeTelemetryHandler_MissingSlug : route sans slug → 404 (chi filter).
func TestDiagPrestigeTelemetryHandler_MissingSlug(t *testing.T) {
	factory := func(context.Context, string) (PrestigeTelemetryDiagProvider, error) {
		return &stubPrestigeTelemetryDiagProvider{}, nil
	}
	h := NewDiagPrestigeTelemetryHandler(factory)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/_diag/prestige/telemetry/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for missing slug", w.Code)
	}
}
