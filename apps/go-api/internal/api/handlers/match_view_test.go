// Package handlers — match_view_test.go : tests unitaires MatchViewHandler avec mock service.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

type mockMatchViewService struct {
	resp        domain.MatchViewResponse
	err         error
	objEvents   []domain.ObjectiveEvent
	objEventErr error
	positions   []positions.PlayerPosition
	posErr      error
}

func (m *mockMatchViewService) GetMatchView(_ context.Context, _ string) (domain.MatchViewResponse, error) {
	return m.resp, m.err
}

func (m *mockMatchViewService) GetMatchNeighbors(_ context.Context, _ string) (domain.MatchNeighbors, error) {
	return domain.MatchNeighbors{}, nil
}

func (m *mockMatchViewService) GetMatchNeighborsFiltered(_ context.Context, _ string, _ *domain.MatchFilterSpec) (domain.MatchNeighbors, error) {
	return domain.MatchNeighbors{}, nil
}

func (m *mockMatchViewService) GetObjectiveEvents(_ context.Context, _ string) ([]domain.ObjectiveEvent, error) {
	return m.objEvents, m.objEventErr
}

func (m *mockMatchViewService) GetMatchPositions(_ context.Context, _ string) ([]positions.PlayerPosition, error) {
	return m.positions, m.posErr
}

func newMatchViewRouter(factory handlers.ServiceFactory[port.MatchViewService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func TestMatchViewHandler_OK(t *testing.T) {
	expected := domain.MatchViewResponse{Header: domain.MatchViewHeader{MatchID: "abc123"}}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{resp: expected}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.MatchViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Header.MatchID != expected.Header.MatchID {
		t.Errorf("MatchID: got %q, want %q", resp.Header.MatchID, expected.Header.MatchID)
	}
}

func TestMatchViewHandler_PlayerNotFound(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return nil, errors.New("player_not_found")
	}

	req := httptest.NewRequest(http.MethodGet, "/players/unknown/matches/abc123", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestMatchViewHandler_LocalisationFR_ChampsJSON vérifie que map_ui, mode_ui et
// playlist_label sont correctement sérialisés dans la réponse JSON. Régression :
// un changement de JSON tag ou un oubli d'assignation dans buildMatchHeader
// rendrait ces champs vides — le frontend afficherait "Forbidden" seul.
func TestMatchViewHandler_LocalisationFR_ChampsJSON(t *testing.T) {
	expected := domain.MatchViewResponse{
		Header: domain.MatchViewHeader{
			MatchID:       "fr-match",
			MapUI:         "Forbidden",
			ModeUI:        "Capture du drapeau",
			PlaylistLabel: "Partie rapide",
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{resp: expected}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/fr-match", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MatchViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Header.MapUI != "Forbidden" {
		t.Errorf("map_ui = %q, want 'Forbidden'", resp.Header.MapUI)
	}
	if resp.Header.ModeUI != "Capture du drapeau" {
		t.Errorf("mode_ui = %q, want 'Capture du drapeau'", resp.Header.ModeUI)
	}
	if resp.Header.PlaylistLabel != "Partie rapide" {
		t.Errorf("playlist_label = %q, want 'Partie rapide'", resp.Header.PlaylistLabel)
	}
}

// ADR 0029 Couche B : un match non-participé (APIError match_not_participant)
// est mappé en 404 avec le code distinct, pas en 500.
func TestMatchViewHandler_NotParticipant_404(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{
			err: &domain.APIError{Code: "match_not_participant", Message: "non participant"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/xyz", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "match_not_participant" {
		t.Errorf("code = %v, want match_not_participant", body["code"])
	}
}

// TestMatchViewHandler_NotFound_404 : un match absent du substrat local (jamais
// synchronisé, ou pas encore) — APIError{Code:"not_found"} émise par
// MatchViewService.GetMatchView (domain.ErrNotFound) — est mappé en 404 avec le
// code "match_not_found", PAS en 500. C'est le contrat qui remplace le fallback
// LIVE retiré le 2026-07-25 (BACKLOG "Retirer le fallback LIVE du Match view") :
// aucun appel API live n'est tenté, le service renvoie directement ce typed error.
func TestMatchViewHandler_NotFound_404(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{
			err: domain.ErrNotFound("match", "unknown-match-id"),
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/unknown-match-id", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "match_not_found" {
		t.Errorf("code = %v, want match_not_found", body["code"])
	}
}

func TestMatchViewHandler_ServiceError(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{err: errors.New("db error")}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/p/matches/xyz", nil)
	w := httptest.NewRecorder()
	newMatchViewRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// TestMatchViewHandler_MediaURLsTransformed vérifie que l'onglet médias réécrit
// les chemins bruts (file_path + thumbnail_url) en URLs servables /api/v1/...,
// et que le kind est bien transmis. Régression : sans transformation, le front
// reçoit un chemin relatif → vignette webp et média dans le lecteur en 404.
// L'owner du chemin (JGtm) reste préfixe ; le slug d'URL ne sert qu'au scope.
func TestMatchViewHandler_MediaURLsTransformed(t *testing.T) {
	thumb := "JGtm/thumbs/clip.webp"
	expected := domain.MatchViewResponse{
		Header: domain.MatchViewHeader{MatchID: "m-media"},
		MediaTab: domain.MatchMediaTab{
			MediaItems: []domain.MatchAssociatedMedia{{
				FileID:       "1",
				FileName:     "clip.mp4",
				FilePath:     "JGtm/clip.mp4",
				Kind:         "video",
				ThumbnailURL: &thumb,
			}},
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{resp: expected}, nil
	}

	r := chi.NewRouter()
	// WithMediaURLs(nil, "") : pas de settings store en test ; les chemins relatifs
	// (post-migration) se transforment sans capturesBase/repoRoot.
	h := handlers.NewMatchViewHandler(factory).WithMediaURLs(nil, "")
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/m-media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp domain.MatchViewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.MediaTab.MediaItems) != 1 {
		t.Fatalf("media_items = %d, want 1", len(resp.MediaTab.MediaItems))
	}
	got := resp.MediaTab.MediaItems[0]
	const wantFile = "/api/v1/players/test-player/media/files/JGtm/clip.mp4"
	if got.FilePath != wantFile {
		t.Errorf("file_path = %q, want %q", got.FilePath, wantFile)
	}
	const wantThumb = "/api/v1/players/test-player/media/files/JGtm/thumbs/clip.webp"
	if got.ThumbnailURL == nil || *got.ThumbnailURL != wantThumb {
		t.Errorf("thumbnail_url = %v, want %q", got.ThumbnailURL, wantThumb)
	}
	if got.Kind != "video" {
		t.Errorf("kind = %q, want 'video'", got.Kind)
	}
}

// newObjectiveEventsRouter monte la route /objective-events sur un MatchViewHandler.
func newObjectiveEventsRouter(factory handlers.ServiceFactory[port.MatchViewService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

// TestMatchViewHandler_ObjectiveEvents_OK : le service renvoie 2 events →
// 200 + JSON décodable avec des clés camelCase (matchId, timeMs, players...).
func TestMatchViewHandler_ObjectiveEvents_OK(t *testing.T) {
	tms := 12000
	team := 0
	events := []domain.ObjectiveEvent{
		{
			MatchID: "abc123", Seq: 0, TimeMS: &tms,
			ObjectiveType: "flag", EventType: "capture", TeamID: &team,
			Source: "ctf", Confidence: "high",
			Players: []domain.ObjectiveEventPlayer{{XUID: "2535", Role: "capturer"}},
		},
		{
			MatchID: "abc123", Seq: 1,
			ObjectiveType: "hill", EventType: "score",
			Source: "koth", Confidence: "low",
		},
	}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{objEvents: events}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123/objective-events", nil)
	w := httptest.NewRecorder()
	newObjectiveEventsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("events = %d, want 2", len(resp))
	}
	if resp[0]["matchId"] != "abc123" {
		t.Errorf("matchId = %v, want abc123", resp[0]["matchId"])
	}
	if resp[0]["objectiveType"] != "flag" {
		t.Errorf("objectiveType = %v, want flag", resp[0]["objectiveType"])
	}
	if resp[0]["timeMs"] != float64(12000) {
		t.Errorf("timeMs = %v, want 12000", resp[0]["timeMs"])
	}
	players, ok := resp[0]["players"].([]any)
	if !ok || len(players) != 1 {
		t.Fatalf("players = %v, want 1 entry", resp[0]["players"])
	}
	if p0, _ := players[0].(map[string]any); p0["xuid"] != "2535" || p0["role"] != "capturer" {
		t.Errorf("player[0] = %v, want xuid=2535 role=capturer", players[0])
	}
	// timeMs omitempty : le second event (TimeMS nil) ne doit pas porter la clé.
	if _, present := resp[1]["timeMs"]; present {
		t.Errorf("event[1] timeMs should be omitted (nil), got %v", resp[1]["timeMs"])
	}
}

// TestMatchViewHandler_ObjectiveEvents_CapabilityUnsupported : le service remonte
// games.ErrCapabilityNotSupported → 503 + code 'capability_not_supported'.
func TestMatchViewHandler_ObjectiveEvents_CapabilityUnsupported(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{objEventErr: games.ErrCapabilityNotSupported}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123/objective-events", nil)
	w := httptest.NewRecorder()
	newObjectiveEventsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "capability_not_supported" {
		t.Errorf("code = %v, want capability_not_supported", body["code"])
	}
}

// newPositionsRouter monte la route /positions sur un MatchViewHandler.
func newPositionsRouter(factory handlers.ServiceFactory[port.MatchViewService]) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

// TestMatchViewHandler_Positions_OK : le service renvoie 2 positions → 200 +
// JSON décodable avec des clés camelCase (timeMs, x, y, z, team).
func TestMatchViewHandler_Positions_OK(t *testing.T) {
	pos := []positions.PlayerPosition{
		{TimeMS: 0, X: 25.6, Y: 10.4, Z: 1.2, Team: 0},
		{TimeMS: 20000, X: 34.8, Y: 13.5, Z: 0.5, Team: positions.TeamUnknown},
	}
	factory := func(_ context.Context, slug string) (port.MatchViewService, error) {
		if slug != testPlayerSlug {
			return nil, errors.New("player_not_found")
		}
		return &mockMatchViewService{positions: pos}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123/positions", nil)
	w := httptest.NewRecorder()
	newPositionsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("positions = %d, want 2", len(resp))
	}
	if resp[0]["timeMs"] != float64(0) {
		t.Errorf("timeMs = %v, want 0", resp[0]["timeMs"])
	}
	// float32 25.6 sérialisé en JSON → "25.6" (repr décimale la plus courte),
	// donc 25.6 (float64) après unmarshal — pas la valeur élargie float32.
	if resp[0]["x"] != 25.6 {
		t.Errorf("x = %v, want 25.6", resp[0]["x"])
	}
	if resp[0]["team"] != float64(0) {
		t.Errorf("team = %v, want 0", resp[0]["team"])
	}
	// team -1 (TeamUnknown) doit être sérialisé tel quel (pas omitempty).
	if resp[1]["team"] != float64(-1) {
		t.Errorf("team = %v, want -1 (TeamUnknown)", resp[1]["team"])
	}
}

// TestMatchViewHandler_Positions_CapabilityUnsupported : le service remonte
// games.ErrCapabilityNotSupported → 503 + code 'capability_not_supported'.
func TestMatchViewHandler_Positions_CapabilityUnsupported(t *testing.T) {
	factory := func(_ context.Context, _ string) (port.MatchViewService, error) {
		return &mockMatchViewService{posErr: games.ErrCapabilityNotSupported}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/matches/abc123/positions", nil)
	w := httptest.NewRecorder()
	newPositionsRouter(factory).ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "capability_not_supported" {
		t.Errorf("code = %v, want capability_not_supported", body["code"])
	}
}
