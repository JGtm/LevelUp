// Package handlers — progression_backfill_test.go : couverture handler
// POST /_admin/progression/backfill. Mock runner pour isoler la logique HTTP.
//
// Fix 2026-05-30 (timeout shared reader → tables progression vides).
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

type stubProgressionBackfiller struct {
	diag *domain.ProgressionDiag
	err  error
}

func (s *stubProgressionBackfiller) BackfillProgression(_ context.Context, slug string) (*domain.ProgressionDiag, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := *s.diag
	out.PlayerSlug = slug
	return &out, nil
}

func serveBackfill(t *testing.T, slug string, factory ProgressionBackfillFactory) *httptest.ResponseRecorder {
	t.Helper()
	h := NewProgressionBackfillHandler(factory)
	r := chi.NewRouter()
	r.Post("/api/v1/_admin/progression/backfill/{player_slug}", h.RunBackfill)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/_admin/progression/backfill/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestProgressionBackfill_Success : le backfill renvoie le diag post-exécution.
func TestProgressionBackfill_Success(t *testing.T) {
	diag := &domain.ProgressionDiag{
		StreakCount:           2,
		PlayerRecordsCount:    9,
		MilestoneEarnedCount:  6,
		MilestoneCatalogCount: 14,
	}
	factory := func(_ context.Context, _ string) (ProgressionBackfiller, error) {
		return &stubProgressionBackfiller{diag: diag}, nil
	}
	w := serveBackfill(t, "JGtm", factory)
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
	if got.MilestoneEarnedCount != 6 {
		t.Errorf("MilestoneEarnedCount: got %d want 6", got.MilestoneEarnedCount)
	}
}

// TestProgressionBackfill_PlayerNotFound : factory en erreur → 404.
func TestProgressionBackfill_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (ProgressionBackfiller, error) {
		return nil, errors.New("player_not_found")
	}
	w := serveBackfill(t, "Unknown", factory)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestProgressionBackfill_RunError : l'évaluation échoue → 500.
func TestProgressionBackfill_RunError(t *testing.T) {
	factory := func(_ context.Context, _ string) (ProgressionBackfiller, error) {
		return &stubProgressionBackfiller{err: errors.New("boom")}, nil
	}
	w := serveBackfill(t, "JGtm", factory)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

// TestProgressionBackfill_MissingSlug : route sans slug → 404 (chi route filter).
func TestProgressionBackfill_MissingSlug(t *testing.T) {
	factory := func(_ context.Context, _ string) (ProgressionBackfiller, error) {
		return &stubProgressionBackfiller{diag: &domain.ProgressionDiag{}}, nil
	}
	h := NewProgressionBackfillHandler(factory)
	r := chi.NewRouter()
	r.Post("/api/v1/_admin/progression/backfill/{player_slug}", h.RunBackfill)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/_admin/progression/backfill/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for missing slug", w.Code)
	}
}
