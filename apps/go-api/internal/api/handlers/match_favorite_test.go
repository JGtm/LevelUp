// Package handlers_test — match_favorite_test.go : tests unitaires MatchFavoriteHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// mockSocialService implémente port.SocialService.
type mockSocialService struct {
	toggleErr error
}

func (m *mockSocialService) ToggleMatchFavorite(_ context.Context, _ domain.MatchFavoriteRequest) error {
	return m.toggleErr
}

func newMatchFavoriteRouter(factory handlers.ServiceFactory[port.SocialService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchFavoriteHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Patch("/matches/{match_id}/favorite", h.PatchMatchFavorite)
	})
	return r
}

func TestMatchFavoriteHandler_Add_OK(t *testing.T) {
	mock := &mockSocialService{}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MatchFavoriteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MatchID != "match-abc" {
		t.Errorf("expected match_id=match-abc, got %s", resp.MatchID)
	}
	if !resp.Favorited {
		t.Error("expected favorited=true")
	}
}

func TestMatchFavoriteHandler_Remove_OK(t *testing.T) {
	mock := &mockSocialService{}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: false})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MatchFavoriteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Favorited {
		t.Error("expected favorited=false")
	}
}

func TestMatchFavoriteHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.SocialService, error) {
		return nil, errors.New("player_not_found")
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/unknown/matches/match-abc/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchFavoriteHandler_InvalidBody(t *testing.T) {
	mock := &mockSocialService{}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchFavoriteRouter(factory)

	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/favorite",
		bytes.NewBufferString("{not json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchFavoriteHandler_DBError(t *testing.T) {
	mock := &mockSocialService{toggleErr: errors.New("db failure")}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMatchFavoriteHandler_SlugAndMatchIDPropagated(t *testing.T) {
	var capturedReq domain.MatchFavoriteRequest
	mock := &mockSocialService{}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &captureService{inner: mock, capture: &capturedReq}, nil
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-xyz/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capturedReq.PlayerSlug != testPlayerSlug {
		t.Errorf("expected player_slug=%s, got %s", testPlayerSlug, capturedReq.PlayerSlug)
	}
	if capturedReq.MatchID != "match-xyz" {
		t.Errorf("expected match_id=match-xyz, got %s", capturedReq.MatchID)
	}
}

// captureService wraps mockSocialService et capture la requête pour inspection.
type captureService struct {
	inner   port.SocialService
	capture *domain.MatchFavoriteRequest
}

func (c *captureService) ToggleMatchFavorite(ctx context.Context, req domain.MatchFavoriteRequest) error {
	*c.capture = req
	return c.inner.ToggleMatchFavorite(ctx, req)
}

// ─── ErrDBLocked → 503 (commit 5 db-concurrency) ───

func TestMatchFavoriteHandler_DBLocked_Returns503(t *testing.T) {
	mock := &mockSocialService{
		toggleErr: fmt.Errorf("simulated lease busy: %w", dblease.ErrDBLocked),
	}
	factory := func(_ context.Context, slug string) (port.SocialService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return mock, nil
	}
	r := newMatchFavoriteRouter(factory)

	body, _ := json.Marshal(domain.MatchFavoriteRequest{Favorited: true})
	req := httptest.NewRequest(http.MethodPatch, "/players/test-player/matches/match-abc/favorite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "5" {
		t.Errorf("Retry-After header = %q, want %q", got, "5")
	}
	var body503 map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body503); err != nil {
		t.Fatalf("response body not JSON: %v", err)
	}
	if code, _ := body503["code"].(string); code != "db_busy" {
		t.Errorf("error code = %v, want db_busy", body503["code"])
	}
}
