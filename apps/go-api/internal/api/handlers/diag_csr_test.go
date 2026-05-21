// Package handlers — diag_csr_test.go : couverture handler /_diag/csr-coverage.
//
// Phase 9 du plan pipeline CSR. Test handler avec provider mock pour isoler la
// logique HTTP (routing, error codes, JSON shape) de l'accès DB.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

type stubCoverageProvider struct {
	cov *domain.CSRCoverage
	err error
}

func (s *stubCoverageProvider) GetCoverage(_ context.Context, slug, xuid string) (*domain.CSRCoverage, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.cov == nil {
		return nil, nil
	}
	out := *s.cov
	out.PlayerSlug = slug
	out.XUID = xuid
	return &out, nil
}

func serveDiag(t *testing.T, slug string, factory CSRCoverageFactory) *httptest.ResponseRecorder {
	t.Helper()
	h := NewDiagCSRHandler(factory)
	r := chi.NewRouter()
	r.Get("/api/v1/_diag/csr-coverage/{player_slug}", h.GetCoverage)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/_diag/csr-coverage/"+slug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDiagCSRHandler_FullCoverage_NoBackfillNeeded(t *testing.T) {
	cov := &domain.CSRCoverage{
		Snapshots: domain.CSRSnapshotsCoverage{Total: 4, WithAlltimeValue: 4},
		MatchSkillRankCSR: domain.MSRCSRCoverage{
			Total: 12, Matured: 5, Placement: 7, RankedMatchesInRegistry: 12, CoverageGap: 0,
		},
		NeedsBackfill: false,
	}
	factory := func(ctx context.Context, slug string) (CSRCoverageProvider, string, error) {
		return &stubCoverageProvider{cov: cov}, "xuid-123", nil
	}
	w := serveDiag(t, "JGtm", factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got domain.CSRCoverage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if got.PlayerSlug != "JGtm" || got.XUID != "xuid-123" {
		t.Errorf("slug/xuid: got %q/%q", got.PlayerSlug, got.XUID)
	}
	if got.NeedsBackfill {
		t.Error("NeedsBackfill: want false")
	}
	if got.MatchSkillRankCSR.Total != 12 {
		t.Errorf("MatchSkillRankCSR.Total: got %d, want 12", got.MatchSkillRankCSR.Total)
	}
}

func TestDiagCSRHandler_GapDetected_NeedsBackfillTrue(t *testing.T) {
	cov := &domain.CSRCoverage{
		Snapshots: domain.CSRSnapshotsCoverage{Total: 0},
		MatchSkillRankCSR: domain.MSRCSRCoverage{
			Total: 2, RankedMatchesInRegistry: 5, CoverageGap: 3,
		},
		NeedsBackfill: true,
	}
	factory := func(ctx context.Context, slug string) (CSRCoverageProvider, string, error) {
		return &stubCoverageProvider{cov: cov}, "xuid-456", nil
	}
	w := serveDiag(t, "Chocoboflor", factory)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"needs_backfill":true`) {
		t.Errorf("needs_backfill=true absent du body : %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"coverage_gap":3`) {
		t.Errorf("coverage_gap=3 absent du body : %s", w.Body.String())
	}
}

func TestDiagCSRHandler_PlayerNotFound_Returns404(t *testing.T) {
	factory := func(ctx context.Context, slug string) (CSRCoverageProvider, string, error) {
		return nil, "", errors.New("player " + slug + " not found")
	}
	w := serveDiag(t, "unknown", factory)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "player_not_found") {
		t.Errorf("error code player_not_found absent : %s", w.Body.String())
	}
}

func TestDiagCSRHandler_ProviderError_Returns500(t *testing.T) {
	factory := func(ctx context.Context, slug string) (CSRCoverageProvider, string, error) {
		return &stubCoverageProvider{err: errors.New("db read failure")}, "xuid-789", nil
	}
	w := serveDiag(t, "JGtm", factory)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "coverage_error") {
		t.Errorf("error code coverage_error absent : %s", w.Body.String())
	}
}
