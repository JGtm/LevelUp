// Package handlers — diag_progression_test.go : couverture handler
// /_diag/progression. Test avec provider mock pour isoler la logique HTTP.
//
// Phase 4 plan stabilisation 2026-05-22.
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

type stubProgressionDiagProvider struct {
	diag *domain.ProgressionDiag
	err  error
}

func (s *stubProgressionDiagProvider) GetProgressionDiag(_ context.Context, slug string) (*domain.ProgressionDiag, error) {
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

func serveProgressionDiag(t *testing.T, slug string, factory ProgressionDiagFactory) *httptest.ResponseRecorder {
	t.Helper()
	h := NewDiagProgressionHandler(factory)
	r := chi.NewRouter()
	r.Get("/api/v1/_diag/progression/{player_slug}", h.GetDiag)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/_diag/progression/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestDiagProgressionHandler_PipelineRunning : counts non-zéro = pipeline OK.
func TestDiagProgressionHandler_PipelineRunning(t *testing.T) {
	diag := &domain.ProgressionDiag{
		StreakCount:           3,
		PlayerRecordsCount:    8,
		RecordHistoryCount:    24,
		MilestoneEarnedCount:  5,
		MilestoneCatalogCount: 13,
		PipelineWiredAt:       "2026-05-22T10:00:00Z",
	}
	factory := func(ctx context.Context, slug string) (ProgressionDiagProvider, error) {
		return &stubProgressionDiagProvider{diag: diag}, nil
	}
	w := serveProgressionDiag(t, "JGtm", factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got domain.ProgressionDiag
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.PlayerSlug != "JGtm" {
		t.Errorf("slug: got %q want JGtm", got.PlayerSlug)
	}
	if got.MilestoneCatalogCount != 13 {
		t.Errorf("MilestoneCatalogCount: got %d want 13", got.MilestoneCatalogCount)
	}
	if got.StreakCount != 3 {
		t.Errorf("StreakCount: got %d want 3", got.StreakCount)
	}
}

// TestDiagProgressionHandler_PipelineNotWired : tout à zéro = pipeline pas câblé.
func TestDiagProgressionHandler_PipelineNotWired(t *testing.T) {
	diag := &domain.ProgressionDiag{
		MilestoneCatalogCount: 13, // catalog seedé (Phase 4.2)
		// streak/records/milestones = 0 (pipeline pas câblé)
	}
	factory := func(ctx context.Context, slug string) (ProgressionDiagProvider, error) {
		return &stubProgressionDiagProvider{diag: diag}, nil
	}
	w := serveProgressionDiag(t, "JGtm", factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got domain.ProgressionDiag
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.MilestoneCatalogCount != 13 {
		t.Errorf("catalog count: got %d want 13", got.MilestoneCatalogCount)
	}
	if got.StreakCount != 0 || got.MilestoneEarnedCount != 0 {
		t.Errorf("expected pipeline-empty signature, got streak=%d milestone=%d",
			got.StreakCount, got.MilestoneEarnedCount)
	}
}

// TestDiagProgressionHandler_PlayerNotFound : factory retourne err → 404.
func TestDiagProgressionHandler_PlayerNotFound(t *testing.T) {
	factory := func(ctx context.Context, slug string) (ProgressionDiagProvider, error) {
		return nil, errors.New("player_not_found")
	}
	w := serveProgressionDiag(t, "Unknown", factory)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestDiagProgressionHandler_MissingSlug : route sans slug → 404 (chi route filter).
func TestDiagProgressionHandler_MissingSlug(t *testing.T) {
	factory := func(_ context.Context, _ string) (ProgressionDiagProvider, error) {
		return &stubProgressionDiagProvider{}, nil
	}
	h := NewDiagProgressionHandler(factory)
	r := chi.NewRouter()
	r.Get("/api/v1/_diag/progression/{player_slug}", h.GetDiag)

	// Empty path segment → chi ne matche pas la route → 404
	req := httptest.NewRequest(http.MethodGet, "/api/v1/_diag/progression/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for missing slug", w.Code)
	}
}
