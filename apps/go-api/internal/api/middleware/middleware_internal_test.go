// Package middleware — middleware_internal_test.go : tests internes des fonctions privées.
//
// Couvre : contractResponseWriter, validateErrorShape, resolveTitleSlug.
//
// NOTE : tests errorTrackWriter, discordSimplePayload, checkWindow, notifyError,
// postDiscord SUPPRIMÉS en revue 2026-04-29 P8.3 (ADR 0009 — error_tracker
// retiré, alerting Discord 500 « pas souhaité »).
package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// ---------------------------------------------------------------------------
// contractResponseWriter
// ---------------------------------------------------------------------------

func TestContractResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &contractResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
	}
	cw.WriteHeader(http.StatusCreated)
	if cw.status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", cw.status)
	}
	// Second call should not overwrite.
	cw.WriteHeader(http.StatusNotFound)
	if cw.status != http.StatusCreated {
		t.Fatalf("expected status 201 (unchanged), got %d", cw.status)
	}
}

func TestContractResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &contractResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
	}
	body := []byte(`{"ok":true}`)
	n, err := cw.Write(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(body) {
		t.Fatalf("expected %d bytes, got %d", len(body), n)
	}
	if cw.status != http.StatusOK {
		t.Fatalf("expected default status 200, got %d", cw.status)
	}
	if cw.buf.String() != string(body) {
		t.Fatalf("buffer mismatch: %q", cw.buf.String())
	}
}

// ---------------------------------------------------------------------------
// validateErrorShape
// ---------------------------------------------------------------------------

func TestValidateErrorShape_ValidError(t *testing.T) {
	parsed := map[string]any{
		"code":      "not_found",
		"message":   "Player not found",
		"retryable": false,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Should not panic — all fields present.
	validateErrorShape(parsed, req, 404)
}

func TestValidateErrorShape_MissingFields(t *testing.T) {
	parsed := map[string]any{
		"code": "bad_request",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	// Should log warning but not panic.
	validateErrorShape(parsed, req, 400)
}

func TestValidateErrorShape_NonObject(t *testing.T) {
	parsed := "just a string"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Should log warning but not panic.
	validateErrorShape(parsed, req, 500)
}

// Tests errorTrackWriter / discordSimplePayload / checkWindow / notifyError /
// postDiscord supprimés en revue 2026-04-29 P8.3 (ADR 0009).

// ---------------------------------------------------------------------------

func TestResolveTitleSlug_NoHeader_NoSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	reg := titlePkg.NewRegistry()
	slug := resolveTitleSlug(req, reg)
	// No header, no session → fallback to default slug.
	if slug == "" {
		t.Fatal("expected non-empty default slug")
	}
}
