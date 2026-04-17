// Package middleware — middleware_internal_test.go : tests internes des fonctions privées.
//
// Couvre : contractResponseWriter, validateErrorShape, errorTrackWriter,
// discordSimplePayload, checkWindow, notifyError, shadowCall, shadowResponseWriter.
package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// ---------------------------------------------------------------------------
// errorTrackWriter
// ---------------------------------------------------------------------------

func TestErrorTrackWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	ew := &errorTrackWriter{ResponseWriter: w, status: http.StatusOK}
	ew.WriteHeader(http.StatusInternalServerError)
	if ew.status != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", ew.status)
	}
	// Second call should not overwrite.
	ew.WriteHeader(http.StatusOK)
	if ew.status != http.StatusInternalServerError {
		t.Fatalf("expected 500 (unchanged), got %d", ew.status)
	}
}

func TestErrorTrackWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	ew := &errorTrackWriter{ResponseWriter: w, status: http.StatusOK}
	body := []byte("hello")
	n, err := ew.Write(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(body) {
		t.Fatalf("expected %d, got %d", len(body), n)
	}
	if ew.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", ew.status)
	}
}

// ---------------------------------------------------------------------------
// discordSimplePayload
// ---------------------------------------------------------------------------

func TestDiscordSimplePayload(t *testing.T) {
	p := discordSimplePayload("hello world")
	content, ok := p["content"]
	if !ok {
		t.Fatal("expected 'content' key")
	}
	if content != "hello world" {
		t.Fatalf("expected 'hello world', got %q", content)
	}
}

// ---------------------------------------------------------------------------
// ErrorTracker.checkWindow — window expired + threshold breach
// ---------------------------------------------------------------------------

func TestCheckWindow_ExpiredWindow(t *testing.T) {
	et := NewErrorTracker(ErrorTrackerConfig{
		WebhookURL:     "", // no webhook → no alert sent
		AlertThreshold: 5.0,
		AlertCooldown:  time.Minute,
	})
	// Simulate window started >1 min ago.
	et.mu.Lock()
	et.windowStart = time.Now().Add(-2 * time.Minute)
	et.mu.Unlock()

	et.atomicTotal.Store(100)
	et.atomicErrors.Store(10)

	// checkWindow should reset counters.
	et.checkWindow()

	if et.atomicTotal.Load() != 0 {
		t.Fatalf("expected total reset to 0, got %d", et.atomicTotal.Load())
	}
	if et.atomicErrors.Load() != 0 {
		t.Fatalf("expected errors reset to 0, got %d", et.atomicErrors.Load())
	}
}

func TestCheckWindow_WithWebhook_ThresholdBreach(t *testing.T) {
	// Fake Discord server to receive the alert.
	received := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		buf.ReadFrom(r.Body) //nolint:errcheck
		received <- buf.Bytes()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	et := NewErrorTracker(ErrorTrackerConfig{
		WebhookURL:     srv.URL,
		AlertThreshold: 5.0,
		AlertCooldown:  time.Millisecond, // immediate
	})
	et.mu.Lock()
	et.windowStart = time.Now().Add(-2 * time.Minute)
	et.mu.Unlock()

	et.atomicTotal.Store(100)
	et.atomicErrors.Store(20) // 20% > 5% threshold

	et.checkWindow()

	// Wait for the goroutine to post.
	select {
	case body := <-received:
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := payload["content"]; !ok {
			t.Fatal("expected 'content' field in Discord payload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Discord alert")
	}
}

// ---------------------------------------------------------------------------
// ErrorTracker.notifyError — with a fake webhook
// ---------------------------------------------------------------------------

func TestNotifyError_SendsWebhook(t *testing.T) {
	received := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		buf.ReadFrom(r.Body) //nolint:errcheck
		received <- buf.Bytes()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	et := NewErrorTracker(ErrorTrackerConfig{WebhookURL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/test", nil)
	et.notifyError(req, 500)

	select {
	case body := <-received:
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if _, ok := payload["content"]; !ok {
			t.Fatal("expected 'content' key in payload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook")
	}
}

// ---------------------------------------------------------------------------
// ErrorTracker.postDiscord — non-200 response handling
// ---------------------------------------------------------------------------

func TestPostDiscord_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	et := NewErrorTracker(ErrorTrackerConfig{WebhookURL: srv.URL})
	// Should not panic on non-200 response.
	et.postDiscord(discordSimplePayload("test"))
}

func TestPostDiscord_InvalidURL(t *testing.T) {
	et := NewErrorTracker(ErrorTrackerConfig{WebhookURL: "http://127.0.0.1:1"})
	// Should not panic on connection refused.
	et.postDiscord(discordSimplePayload("test"))
}

// ---------------------------------------------------------------------------
// shadowCall — with a fake Python server
// ---------------------------------------------------------------------------

func TestShadowCall_Divergence(t *testing.T) {
	// Fake Python server returning different response.
	pythonSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"python":true}`))
	}))
	defer pythonSrv.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	goResp := []byte(`{"go":true}`)
	// Should not panic — logs divergence.
	shadowCall(req, nil, goResp, http.StatusOK, pythonSrv.URL)
}

func TestShadowCall_Match(t *testing.T) {
	body := []byte(`{"same":true}`)
	pythonSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer pythonSrv.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Same body → match.
	shadowCall(req, nil, body, http.StatusOK, pythonSrv.URL)
}

func TestShadowCall_PythonDown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Unreachable Python → should not panic.
	shadowCall(req, nil, []byte(`{}`), 200, "http://127.0.0.1:1")
}

// ---------------------------------------------------------------------------
// shadowResponseWriter
// ---------------------------------------------------------------------------

func TestShadowResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &shadowResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
	}
	sw.WriteHeader(http.StatusAccepted)
	if sw.status != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", sw.status)
	}
}

func TestShadowResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &shadowResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
	}
	body := []byte("data")
	n, err := sw.Write(body)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(body) {
		t.Fatalf("expected %d, got %d", len(body), n)
	}
	if sw.buf.String() != "data" {
		t.Fatalf("buffer mismatch: %q", sw.buf.String())
	}
}

// ---------------------------------------------------------------------------
// resolveTitleSlug — header branch with valid slug
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
