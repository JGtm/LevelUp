// Package handlers — session_context_test.go : tests unitaires SessionHandler.PostContext.
package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

func newSessionContextRouter(t *testing.T) *chi.Mux {
	t.Helper()
	dir := t.TempDir()
	store := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := handlers.NewSessionHandler(store)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(store, false))
	r.Post("/session/context", h.PostContext)
	return r
}

func TestSessionHandler_PostContext_OK(t *testing.T) {
	r := newSessionContextRouter(t)
	slug := "test-player"
	locale := "fr"
	body, _ := json.Marshal(domain.SessionContextRequest{
		PlayerSlug: &slug,
		Locale:     &locale,
	})

	req := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionHandler_PostContext_InvalidBody(t *testing.T) {
	r := newSessionContextRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSessionHandler_PostContext_TitleSwitch(t *testing.T) {
	r := newSessionContextRouter(t)
	titleSlug := "halo_infinite"
	body, _ := json.Marshal(domain.SessionContextRequest{
		TitleSlug: &titleSlug,
	})

	req := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.SessionContextResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CurrentTitleSlug != "halo_infinite" {
		t.Errorf("expected title slug halo_infinite, got %q", resp.CurrentTitleSlug)
	}
}

func TestSessionHandler_PostContext_AvailableTitles(t *testing.T) {
	r := newSessionContextRouter(t)
	body, _ := json.Marshal(domain.SessionContextRequest{})

	req := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.SessionContextResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.AvailableTitles) == 0 {
		t.Fatal("expected at least one available title")
	}
	found := false
	for _, title := range resp.AvailableTitles {
		if title.Slug == "halo_infinite" {
			found = true
			break
		}
	}
	if !found {
		t.Error("halo_infinite not found in available_titles")
	}
}

func TestSessionHandler_PostContext_TitleSwitchResetsPlayer(t *testing.T) {
	r := newSessionContextRouter(t)

	// Step 1: set a player
	slug := "my-player"
	body1, _ := json.Marshal(domain.SessionContextRequest{PlayerSlug: &slug})
	req1 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("step1: expected 200, got %d", w1.Code)
	}

	// Extract session cookie
	respCk1 := w1.Result()
	cookies := respCk1.Cookies()
	_ = respCk1.Body.Close()

	// Step 2: switch title → player should be reset
	titleSlug := "halo_infinite"
	body2, _ := json.Marshal(domain.SessionContextRequest{TitleSlug: &titleSlug})
	req2 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("step2: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp domain.SessionContextResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CurrentPlayerSlug != nil {
		t.Errorf("expected nil player after title switch, got %v", *resp.CurrentPlayerSlug)
	}
}

func TestSessionHandler_PostContext_UnknownTitleIgnored(t *testing.T) {
	r := newSessionContextRouter(t)

	// Set a player first
	slug := "my-player"
	body1, _ := json.Marshal(domain.SessionContextRequest{PlayerSlug: &slug})
	req1 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	respCk2 := w1.Result()
	cookies := respCk2.Cookies()
	_ = respCk2.Body.Close()

	// Try unknown title → should be silently ignored
	badTitle := "fortnite_infinite"
	body2, _ := json.Marshal(domain.SessionContextRequest{TitleSlug: &badTitle})
	req2 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp domain.SessionContextResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Player should NOT have been reset (title switch was ignored)
	if resp.CurrentPlayerSlug == nil || *resp.CurrentPlayerSlug != "my-player" {
		t.Errorf("player should be preserved when unknown title is ignored, got %v", resp.CurrentPlayerSlug)
	}
}

func TestSessionHandler_PostContext_Propagation(t *testing.T) {
	r := newSessionContextRouter(t)

	// Step 1: set locale
	locale := "en"
	body1, _ := json.Marshal(domain.SessionContextRequest{Locale: &locale})
	req1 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	respCk3 := w1.Result()
	cookies := respCk3.Cookies()
	_ = respCk3.Body.Close()

	// Step 2: verify locale persisted in subsequent call
	body2, _ := json.Marshal(domain.SessionContextRequest{})
	req2 := httptest.NewRequest(http.MethodPost, "/session/context", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp domain.SessionContextResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Locale != "en" {
		t.Errorf("expected locale en to propagate, got %q", resp.Locale)
	}
}
