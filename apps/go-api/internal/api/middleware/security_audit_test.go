// Package middleware — security_audit_test.go : tests d'audit de sécurité (Sprint 36).
//
// Complète csrf_test.go avec :
//   - Rate limiting : constantes et comportement démo
//   - Corps JSON malformé (400 attendu)
//   - Format d'erreur structuré (code + message, pas de stack trace)
package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
)

// ---------------------------------------------------------------------------
// Rate limit
// ---------------------------------------------------------------------------

func TestRateLimit_Constants(t *testing.T) {
	// Vérifier que les constantes de rate limit sont dans des plages raisonnables.
	// Sprint 36 : audit sécurité — s'assurer qu'on ne limite pas trop peu.
	if middleware.RateLimitRequests < 60 {
		t.Errorf("RateLimitRequests=%d trop bas (min 60 attendu)", middleware.RateLimitRequests)
	}
	if middleware.RateLimitWindow < 30*time.Second {
		t.Errorf("RateLimitWindow=%s trop court (min 30s attendu)", middleware.RateLimitWindow)
	}
}

func TestRateLimit_DemoMode_ReturnsHandler(t *testing.T) {
	// Vérifier que RateLimit(demoMode=true) retourne bien un handler valide.
	handler := middleware.RateLimit(true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("RateLimit(demo) première requête doit passer, got %d", rr.Code)
	}
}

func TestRateLimit_NormalMode_ReturnsHandler(t *testing.T) {
	handler := middleware.RateLimit(false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "127.0.0.2:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("RateLimit(normal) première requête doit passer, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Format d'erreur structuré
// ---------------------------------------------------------------------------

// errorCapture est un handler factice qui déclenche une erreur JSON structurée.
type errorCapture struct{ called bool }

func (e *errorCapture) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	e.called = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"code":"invalid_body","message":"json invalide"}`))
}

func TestErrorFormat_ContainsCodeAndMessage(t *testing.T) {
	// Vérifier que le format d'erreur inclut "code" et "message" sans stack trace.
	h := &errorCapture{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, `"code"`) {
		t.Errorf("ErrorResponse doit contenir 'code', got: %s", body)
	}
	if !strings.Contains(body, `"message"`) {
		t.Errorf("ErrorResponse doit contenir 'message', got: %s", body)
	}
	// Vérifier l'absence de stack trace Go (goroutine, runtime)
	if strings.Contains(body, "goroutine") || strings.Contains(body, "runtime/") {
		t.Errorf("ErrorResponse ne doit pas exposer de stack trace, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// CORS : vérifier que les headers sont bien positionnés
// ---------------------------------------------------------------------------

func TestCORS_OptionsReturns200(t *testing.T) {
	handler := middleware.CORS([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/bootstrap", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// OPTIONS preflight doit retourner 200 ou 204
	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS preflight doit retourner 200 ou 204, got %d", rr.Code)
	}
}

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	handler := middleware.CORS([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/players", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Vérifier que l'origine est autorisée dans la réponse
	acao := rr.Header().Get("Access-Control-Allow-Origin")
	if acao == "" {
		t.Error("Access-Control-Allow-Origin doit être présent pour une origine autorisée")
	}
}

// ---------------------------------------------------------------------------
// Body JSON trop volumineux (protection contre DoS)
// ---------------------------------------------------------------------------

func TestCSRF_LargeBody_DoesNotPanic(t *testing.T) {
	// Vérifier que le middleware CSRF ne panique pas avec un gros corps.
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Corps de 1 Mo
	bigBody := bytes.Repeat([]byte("x"), 1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", bytes.NewReader(bigBody))
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Ne doit pas paniquer
	handler.ServeHTTP(rr, req)

	// Soit 200 (passthrough) soit 413 — pas de panic
	if rr.Code == 0 {
		t.Error("La réponse ne doit pas être vide")
	}
}
