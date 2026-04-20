// Package handlers_test — match_favorite_test.go : tests unitaires MatchFavoriteHandler.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
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
