// handlers_internal_test.go — tests internes (package handlers) pour les helpers privés.
//
// encode/decodeExportToken, formatOptFloat, optStr,
// filterCitationsByCategory, filterCommendationsByCategory, fileExists,
// resolveCapturesDir, deviceFlowStartResponse, deviceFlowStatusResponse,
// writeJSONCached.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
)

// ---------------------------------------------------------------------------
// encodeExportToken / decodeExportToken
// ---------------------------------------------------------------------------

func TestEncodeDecodeExportToken_RoundTrip(t *testing.T) {
	req := domain.MatchHistoryQueryRequest{
		SortField: "start_time",
		SortDir:   "desc",
		Pagination: domain.PaginationRequest{
			Page:     2,
			PageSize: 25,
		},
	}
	token, err := encodeExportToken(req)
	if err != nil {
		t.Fatalf("encodeExportToken: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	got, err := decodeExportToken(token)
	if err != nil {
		t.Fatalf("decodeExportToken: %v", err)
	}
	if got.SortField != req.SortField {
		t.Errorf("SortField = %q, want %q", got.SortField, req.SortField)
	}
	if got.Pagination.Page != req.Pagination.Page {
		t.Errorf("Page = %d, want %d", got.Pagination.Page, req.Pagination.Page)
	}
}

func TestDecodeExportToken_InvalidBase64(t *testing.T) {
	_, err := decodeExportToken("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeExportToken_InvalidJSON(t *testing.T) {
	// Valid base64 but not JSON.
	_, err := decodeExportToken("bm90LWpzb24=") // "not-json" in base64
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// formatOptFloat
// ---------------------------------------------------------------------------

func TestFormatOptFloat_Nil(t *testing.T) {
	got := formatOptFloat(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFormatOptFloat_Value(t *testing.T) {
	v := 1234.5678
	got := formatOptFloat(&v)
	if got != "1234.57" {
		t.Errorf("got %q, want %q", got, "1234.57")
	}
}

func TestFormatOptFloat_Zero(t *testing.T) {
	v := 0.0
	got := formatOptFloat(&v)
	if got != "0.00" {
		t.Errorf("got %q, want %q", got, "0.00")
	}
}

// ---------------------------------------------------------------------------
// optStr
// ---------------------------------------------------------------------------

func TestOptStr_Nil(t *testing.T) {
	got := optStr(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestOptStr_Value(t *testing.T) {
	s := "hello"
	got := optStr(&s)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// ---------------------------------------------------------------------------
// filterCitationsByCategory
// ---------------------------------------------------------------------------

func TestFilterCitationsByCategory(t *testing.T) {
	page := &domain.CitationsPageResponse{
		Citations: []domain.CitationItem{
			{NameNorm: "a", Category: "Multikill"},
			{NameNorm: "b", Category: "Style"},
			{NameNorm: "c", Category: "Multikill"},
		},
		Categories: []string{"Multikill", "Style"},
		TotalCount: 3,
	}
	got := filterCitationsByCategory(page, "Multikill")
	if len(got.Citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(got.Citations))
	}
	if got.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", got.TotalCount)
	}
	// Categories preserved.
	if len(got.Categories) != 2 {
		t.Errorf("Categories should be preserved, got %d", len(got.Categories))
	}
}

func TestFilterCitationsByCategory_NoMatch(t *testing.T) {
	page := &domain.CitationsPageResponse{
		Citations:  []domain.CitationItem{{NameNorm: "a", Category: "X"}},
		Categories: []string{"X"},
		TotalCount: 1,
	}
	got := filterCitationsByCategory(page, "Y")
	if len(got.Citations) != 0 {
		t.Errorf("expected 0 citations, got %d", len(got.Citations))
	}
}

// ---------------------------------------------------------------------------
// filterCommendationsByCategory
// ---------------------------------------------------------------------------

func TestFilterCommendationsByCategory(t *testing.T) {
	page := &domain.CommendationsPageResponse{
		Categories: []domain.CommendationCategory{
			{Category: "Combat", Total: 10},
			{Category: "Objective", Total: 5},
			{Category: "Combat", Total: 3},
		},
		TotalCount: 18,
	}
	got := filterCommendationsByCategory(page, "Combat")
	if len(got.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(got.Categories))
	}
	if got.TotalCount != 13 {
		t.Errorf("TotalCount = %d, want 13", got.TotalCount)
	}
}

func TestFilterCommendationsByCategory_NoMatch(t *testing.T) {
	page := &domain.CommendationsPageResponse{
		Categories: []domain.CommendationCategory{{Category: "X", Total: 1}},
		TotalCount: 1,
	}
	got := filterCommendationsByCategory(page, "Z")
	if len(got.Categories) != 0 {
		t.Errorf("expected 0, got %d", len(got.Categories))
	}
	if got.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", got.TotalCount)
	}
}

// ---------------------------------------------------------------------------
// fileExists
// ---------------------------------------------------------------------------

func TestFileExists_Exists(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(tmp) {
		t.Error("expected true for existing file")
	}
}

func TestFileExists_NotExists(t *testing.T) {
	if fileExists(filepath.Join(t.TempDir(), "no-such-file")) {
		t.Error("expected false for non-existing file")
	}
}

// ---------------------------------------------------------------------------
// resolveCapturesDir
// ---------------------------------------------------------------------------

func TestResolveCapturesDir_WithTitle(t *testing.T) {
	got := resolveCapturesDir("", "halo-infinite", "Player1")
	want := filepath.Join("data", "titles", "halo-infinite", "players", "Player1", "captures")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCapturesDir_WithoutTitle(t *testing.T) {
	got := resolveCapturesDir("", "", "Player1")
	want := filepath.Join("data", "titles", "halo_infinite", "players", "Player1", "captures")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// deviceFlowStartResponse
// ---------------------------------------------------------------------------

func TestDeviceFlowStartResponse(t *testing.T) {
	a := &auth_platform.Attempt{
		AttemptID:       "att-1",
		UserCode:        "ABC123",
		VerificationURI: "https://example.com/device",
		ExpiresInSec:    900,
	}
	resp := deviceFlowStartResponse(a)
	if resp.AttemptID != "att-1" {
		t.Errorf("AttemptID = %q", resp.AttemptID)
	}
	if resp.UserCode != "ABC123" {
		t.Errorf("UserCode = %q", resp.UserCode)
	}
	if resp.VerificationURI != "https://example.com/device" {
		t.Errorf("VerificationURI = %q", resp.VerificationURI)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("ExpiresIn = %d", resp.ExpiresIn)
	}
}

// ---------------------------------------------------------------------------
// deviceFlowStatusResponse
// ---------------------------------------------------------------------------

func TestDeviceFlowStatusResponse_Pending(t *testing.T) {
	a := &auth_platform.Attempt{
		AttemptID: "att-1",
		Status:    "pending",
	}
	resp := deviceFlowStatusResponse(a)
	if resp.Status != "pending" {
		t.Errorf("Status = %q", resp.Status)
	}
	if resp.Gamertag != nil {
		t.Error("Gamertag should be nil for pending")
	}
}

func TestDeviceFlowStatusResponse_Authorized(t *testing.T) {
	a := &auth_platform.Attempt{
		AttemptID: "att-2",
		Status:    "authorized",
		Gamertag:  "Player1",
		XUID:      "1234567890",
	}
	resp := deviceFlowStatusResponse(a)
	if resp.Status != "authorized" {
		t.Errorf("Status = %q", resp.Status)
	}
	if resp.Gamertag == nil || *resp.Gamertag != "Player1" {
		t.Errorf("Gamertag = %v", resp.Gamertag)
	}
	if resp.XUID == nil || *resp.XUID != "1234567890" {
		t.Errorf("XUID = %v", resp.XUID)
	}
}

func TestDeviceFlowStatusResponse_Failed(t *testing.T) {
	a := &auth_platform.Attempt{
		AttemptID:   "att-3",
		Status:      "failed",
		ErrorCode:   "device_flow_acquire_error",
		ErrorDetail: "timeout",
	}
	resp := deviceFlowStatusResponse(a)
	if resp.Status != "failed" {
		t.Errorf("Status = %q", resp.Status)
	}
	if resp.ErrorCode == nil || *resp.ErrorCode != "device_flow_acquire_error" {
		t.Errorf("ErrorCode = %v", resp.ErrorCode)
	}
	if resp.ErrorDetail == nil || *resp.ErrorDetail != "timeout" {
		t.Errorf("ErrorDetail = %v", resp.ErrorDetail)
	}
}

// ---------------------------------------------------------------------------
// writeJSONCached
// ---------------------------------------------------------------------------

func TestWriteJSONCached_Returns200AndETag(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	writeJSONCached(w, r, 200, map[string]string{"key": "value"})

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("ETag") == "" {
		t.Error("expected non-empty ETag header")
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", w.Header().Get("Content-Type"))
	}
}

func TestWriteJSONCached_Returns304WhenETagMatches(t *testing.T) {
	// Premier appel pour obtenir l'ETag.
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	writeJSONCached(w1, r1, 200, map[string]string{"key": "value"})
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag from first call")
	}

	// Deuxième appel avec If-None-Match → doit retourner 304.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	r2.Header.Set("If-None-Match", etag)
	writeJSONCached(w2, r2, 200, map[string]string{"key": "value"})

	if w2.Code != 304 {
		t.Errorf("status = %d, want 304", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Errorf("body should be empty for 304, got %d bytes", w2.Body.Len())
	}
}

func TestWriteJSONCached_DifferentPayload_Returns200(t *testing.T) {
	// ETag obtenu pour payload A.
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	writeJSONCached(w1, r1, 200, map[string]string{"key": "a"})
	etagA := w1.Header().Get("ETag")

	// Payload B avec ETag de A → doit retourner 200 (données changées).
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	r2.Header.Set("If-None-Match", etagA)
	writeJSONCached(w2, r2, 200, map[string]string{"key": "b"})

	if w2.Code != 200 {
		t.Errorf("status = %d, want 200 (payload changed)", w2.Code)
	}
}
