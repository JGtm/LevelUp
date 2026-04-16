package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestCSRF_GET_Passthrough(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET should pass through, got %d", rr.Code)
	}
}

func TestCSRF_POST_NoOrigin_Rejected(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST without Origin should be 403, got %d", rr.Code)
	}
}

func TestCSRF_POST_ValidOrigin_Allowed(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST with valid Origin should pass, got %d", rr.Code)
	}
}

func TestCSRF_POST_InvalidOrigin_Rejected(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST with invalid Origin should be 403, got %d", rr.Code)
	}
}

func TestCSRF_POST_RefererFallback(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Referer", "http://localhost:5173/some/page")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST with valid Referer should pass, got %d", rr.Code)
	}
}
