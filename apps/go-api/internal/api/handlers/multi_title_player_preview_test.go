package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

const playerPreviewToml = `
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[fields.current_xp]
labels        = { en = "Current XP", fr = "XP actuelle" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "career"
`

func mustLoadPlayerPreview(t *testing.T) *mappings.FieldMappingSet {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("preview.toml", []byte(playerPreviewToml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return set
}

func newPlayerPreviewHandler(
	t *testing.T,
	dataAdapter games.TitleDataAdapter,
	dataErr error,
	semantic games.TitleSemanticAdapter,
	semanticErr error,
) *MultiTitlePlayerPreviewHandler {
	t.Helper()
	dataFactory := func(_ context.Context, _ string) (games.TitleDataAdapter, error) {
		return dataAdapter, dataErr
	}
	semanticFactory := func(_ context.Context, _ string) (games.TitleSemanticAdapter, error) {
		return semantic, semanticErr
	}
	return NewMultiTitlePlayerPreviewHandler(
		dataFactory, semanticFactory, "halo_infinite",
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
	)
}

func TestMultiTitlePlayerPreview_HappyPath(t *testing.T) {
	t.Parallel()
	xp := 1500
	snap := &canonical.CareerSnapshot{
		Player:    canonical.PlayerIdentity{XUID: "0xPLAYER"},
		CurrentXP: &xp,
		CurrentRank: &canonical.AssetReference{
			Kind: "career_rank", ID: "diamond_3", DefaultLabel: "Diamant 3",
		},
	}
	data := &fakeData{
		caps: games.CapabilityMap{games.CapCareerProgression: games.CapSupported},
		snap: snap,
	}
	semantic := &fakeSemantic{set: mustLoadPlayerPreview(t)}
	h := newPlayerPreviewHandler(t, data, nil, semantic, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/players/test_player/preview/career-multi-title?locale=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body careerPreviewDTO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.TitleSlug != "halo_infinite" {
		t.Errorf("title_slug = %q", body.TitleSlug)
	}
	if body.XUID != "0xPLAYER" {
		t.Errorf("XUID = %q (devrait venir de snap.Player.XUID)", body.XUID)
	}
	if body.CurrentXP == nil || body.CurrentXP.Label != "XP actuelle" {
		t.Errorf("CurrentXP = %+v", body.CurrentXP)
	}
	if body.CurrentRank == nil || body.CurrentRank.DefaultLabel != "Diamant 3" {
		t.Errorf("CurrentRank = %+v", body.CurrentRank)
	}
}

func TestMultiTitlePlayerPreview_DefaultLocaleFR(t *testing.T) {
	t.Parallel()
	data := &fakeData{
		caps: games.CapabilityMap{games.CapCareerProgression: games.CapSupported},
		snap: &canonical.CareerSnapshot{Player: canonical.PlayerIdentity{XUID: "0x1"}},
	}
	semantic := &fakeSemantic{set: mustLoadPlayerPreview(t)}
	h := newPlayerPreviewHandler(t, data, nil, semantic, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	// Sans paramètre locale
	req := httptest.NewRequest("GET", "/api/v1/players/test/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body careerPreviewDTO
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Locale != "fr" {
		t.Errorf("locale par défaut = %q, want fr", body.Locale)
	}
}

func TestMultiTitlePlayerPreview_PlayerNotFound(t *testing.T) {
	t.Parallel()
	h := newPlayerPreviewHandler(t, nil, errors.New("player not found"), nil, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/players/unknown/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestMultiTitlePlayerPreview_SemanticNotFound(t *testing.T) {
	t.Parallel()
	data := &fakeData{
		caps: games.CapabilityMap{games.CapCareerProgression: games.CapSupported},
		snap: &canonical.CareerSnapshot{Player: canonical.PlayerIdentity{XUID: "0x1"}},
	}
	h := newPlayerPreviewHandler(t, data, nil, nil, errors.New("semantic missing"))

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/players/test/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestMultiTitlePlayerPreview_CapabilityNotSupported(t *testing.T) {
	t.Parallel()
	data := &fakeData{
		caps: games.CapabilityMap{games.CapCareerProgression: games.CapNotExposed},
		err:  games.ErrCapabilityNotSupported,
	}
	semantic := &fakeSemantic{set: mustLoadPlayerPreview(t)}
	h := newPlayerPreviewHandler(t, data, nil, semantic, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/players/test/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (degradation gracieuse)", w.Code)
	}
	var body careerPreviewDTO
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.NotSupportedReason == "" {
		t.Errorf("NotSupportedReason vide pour capability not_exposed")
	}
}

func TestMultiTitlePlayerPreview_EmptySlug(t *testing.T) {
	t.Parallel()
	h := newPlayerPreviewHandler(t, nil, nil, nil, nil)

	// Pas de chi.URLParam → empty slug → 400.
	req := httptest.NewRequest("GET", "/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	h.GetCareerPreview(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 pour slug vide", w.Code)
	}
}

func TestMultiTitlePlayerPreview_InternalError(t *testing.T) {
	t.Parallel()
	data := &fakeData{
		caps: games.CapabilityMap{games.CapCareerProgression: games.CapSupported},
		err:  errors.New("db connection lost"),
	}
	semantic := &fakeSemantic{set: mustLoadPlayerPreview(t)}
	h := newPlayerPreviewHandler(t, data, nil, semantic, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/players/{player_slug}/preview/career-multi-title", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/players/test/preview/career-multi-title", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 pour erreur non-capability", w.Code)
	}
}

func TestNewMultiTitlePlayerPreviewHandler_Defaults(t *testing.T) {
	t.Parallel()
	// nil logger doit retomber sur slog.Default
	h := NewMultiTitlePlayerPreviewHandler(nil, nil, "", nil)
	if h == nil {
		t.Fatal("handler nil")
	}
	if h.defaultSlug != "halo_infinite" {
		t.Errorf("defaultSlug fallback = %q", h.defaultSlug)
	}
	if h.logger == nil {
		t.Errorf("logger devrait avoir été remplacé par slog.Default()")
	}
}
