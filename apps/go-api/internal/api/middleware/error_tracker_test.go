// Package middleware_test — error_tracker_test.go : tests ErrorTracker.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
)

func TestErrorTracker_NoAlert_OnSuccess(t *testing.T) {
	et := middleware.NewErrorTracker(middleware.ErrorTrackerConfig{
		AlertThreshold: 5.0,
		AlertCooldown:  time.Minute,
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := et.Middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestErrorTracker_Tracks500(t *testing.T) {
	et := middleware.NewErrorTracker(middleware.ErrorTrackerConfig{
		AlertThreshold: 5.0,
		AlertCooldown:  time.Minute,
		// No webhook → no Discord call
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	handler := et.Middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fail", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestErrorTracker_DefaultThresholds(t *testing.T) {
	et := middleware.NewErrorTracker(middleware.ErrorTrackerConfig{})
	if et == nil {
		t.Fatal("NewErrorTracker returned nil with zero config")
	}
	// Should not panic when serving requests
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := et.Middleware(inner)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	}
}
