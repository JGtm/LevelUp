// Package handlers_test — multititle_test.go : F4 — propagation X-LevelUp-Title.
//
// Sprint 54 F4 : vérifie que le titre courant extrait du contexte (via le middleware
// TitleExtractor) est bien passé aux services Compare et Leaderboard, et qu'aucun
// mélange de données ne se produit entre halo_infinite et halo_mcc.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// ─── Compare multi-titre ──────────────────────────────────────────────────────

// captureCompareService enregistre le context reçu lors de GetPage.
type captureCompareService struct {
	capturedCtx context.Context
	resp        domain.CompareResponse
}

func (s *captureCompareService) GetPage(ctx context.Context, _ domain.CompareRequest) (domain.CompareResponse, error) {
	s.capturedCtx = ctx
	return s.resp, nil
}

func newCompareMultiTitleRouter(svc *captureCompareService) *chi.Mux {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug:     "halo_mcc",
		Name:     "Halo MCC",
		Provider: "halo_mcc",
		Status:   titlePkg.StatusComingSoon,
	})

	factory := func(_ context.Context, slug string) (port.CompareService, string, string, error) {
		return svc, "xuid-1", "TestPlayer", nil
	}

	r := chi.NewRouter()
	r.Use(middleware.TitleExtractor(registry))
	h := handlers.NewCompareHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Post("/pages/compare", h.PostComparePage)
	})
	return r
}

// TestCompare_TitleSlug_HaloInfinite vérifie que halo_infinite est propagé par défaut.
func TestCompare_TitleSlug_HaloInfinite(t *testing.T) {
	svc := &captureCompareService{resp: domain.CompareResponse{TitleSlug: "halo_infinite"}}
	r := newCompareMultiTitleRouter(svc)

	body, _ := json.Marshal(domain.CompareRequest{TargetGamertag: "OtherPlayer"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Pas de X-LevelUp-Title → fallback halo_infinite
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := ctxkeys.TitleSlug(svc.capturedCtx)
	if got != "halo_infinite" {
		t.Errorf("expected title_slug halo_infinite in ctx, got %q", got)
	}
}

// TestCompare_TitleSlug_HaloMCC vérifie que halo_mcc est propagé via X-LevelUp-Title.
func TestCompare_TitleSlug_HaloMCC(t *testing.T) {
	svc := &captureCompareService{resp: domain.CompareResponse{TitleSlug: "halo_mcc"}}
	r := newCompareMultiTitleRouter(svc)

	body, _ := json.Marshal(domain.CompareRequest{TargetGamertag: "OtherPlayer"})
	req := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LevelUp-Title", "halo_mcc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := ctxkeys.TitleSlug(svc.capturedCtx)
	if got != "halo_mcc" {
		t.Errorf("expected title_slug halo_mcc in ctx, got %q", got)
	}
}

// TestCompare_NoTitleMixing vérifie que deux requêtes avec des titres différents
// ne partagent pas le même contexte (absence de mélange de données).
func TestCompare_NoTitleMixing(t *testing.T) {
	svcHI := &captureCompareService{resp: domain.CompareResponse{TitleSlug: "halo_infinite"}}
	svcMCC := &captureCompareService{resp: domain.CompareResponse{TitleSlug: "halo_mcc"}}

	rHI := newCompareMultiTitleRouter(svcHI)
	rMCC := newCompareMultiTitleRouter(svcMCC)

	body, _ := json.Marshal(domain.CompareRequest{TargetGamertag: "OtherPlayer"})

	// Requête 1 : halo_infinite (sans header)
	req1 := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/compare", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	rHI.ServeHTTP(httptest.NewRecorder(), req1)

	// Requête 2 : halo_mcc
	req2 := httptest.NewRequest(http.MethodPost, "/players/test-player/pages/compare", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-LevelUp-Title", "halo_mcc")
	rMCC.ServeHTTP(httptest.NewRecorder(), req2)

	slugHI := ctxkeys.TitleSlug(svcHI.capturedCtx)
	slugMCC := ctxkeys.TitleSlug(svcMCC.capturedCtx)

	if slugHI == slugMCC {
		t.Errorf("title slugs should differ between requests: both %q", slugHI)
	}
	if slugHI != "halo_infinite" {
		t.Errorf("req1: want halo_infinite, got %q", slugHI)
	}
	if slugMCC != "halo_mcc" {
		t.Errorf("req2: want halo_mcc, got %q", slugMCC)
	}
}

// ─── Leaderboard multi-titre ──────────────────────────────────────────────────

// captureLeaderboardService enregistre le context reçu lors de GetPage.
type captureLeaderboardService struct {
	capturedCtx context.Context
	resp        domain.LeaderboardResponse
}

func (s *captureLeaderboardService) GetPage(ctx context.Context, _ domain.LeaderboardRequest) (domain.LeaderboardResponse, error) {
	s.capturedCtx = ctx
	return s.resp, nil
}

func newLeaderboardMultiTitleRouter(svc *captureLeaderboardService) *chi.Mux {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug:     "halo_mcc",
		Name:     "Halo MCC",
		Provider: "halo_mcc",
		Status:   titlePkg.StatusComingSoon,
	})

	factory := func(_ context.Context, slug string) (port.LeaderboardService, string, string, error) {
		return svc, "xuid-1", "TestPlayer", nil
	}

	r := chi.NewRouter()
	r.Use(middleware.TitleExtractor(registry))
	h := handlers.NewLeaderboardHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Get("/pages/leaderboard", h.GetLeaderboardPage)
	})
	return r
}

// TestLeaderboard_TitleSlug_HaloInfinite vérifie le fallback halo_infinite.
func TestLeaderboard_TitleSlug_HaloInfinite(t *testing.T) {
	svc := &captureLeaderboardService{}
	r := newLeaderboardMultiTitleRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/leaderboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := ctxkeys.TitleSlug(svc.capturedCtx)
	if got != "halo_infinite" {
		t.Errorf("expected halo_infinite, got %q", got)
	}
}

// TestLeaderboard_TitleSlug_HaloMCC vérifie la propagation de halo_mcc.
func TestLeaderboard_TitleSlug_HaloMCC(t *testing.T) {
	svc := &captureLeaderboardService{}
	r := newLeaderboardMultiTitleRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/leaderboard", nil)
	req.Header.Set("X-LevelUp-Title", "halo_mcc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := ctxkeys.TitleSlug(svc.capturedCtx)
	if got != "halo_mcc" {
		t.Errorf("expected halo_mcc, got %q", got)
	}
}

// TestLeaderboard_InvalidTitle_Fallback vérifie qu'un titre inconnu retombe sur halo_infinite.
func TestLeaderboard_InvalidTitle_Fallback(t *testing.T) {
	svc := &captureLeaderboardService{}
	r := newLeaderboardMultiTitleRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/leaderboard", nil)
	req.Header.Set("X-LevelUp-Title", "unknown_game_2077")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := ctxkeys.TitleSlug(svc.capturedCtx)
	if got != titlePkg.DefaultSlug {
		t.Errorf("expected fallback %q for unknown title, got %q", titlePkg.DefaultSlug, got)
	}
}
