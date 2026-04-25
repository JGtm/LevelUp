package handlers

import (
	"context"
	"encoding/json"
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

const previewToml = `
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

[fields.xp_for_next_rank]
labels        = { en = "XP for Next Rank", fr = "XP rang suivant" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 20
group         = "career"
`

// fakeData implémente games.TitleDataAdapter pour le preview.
type fakeData struct {
	caps games.CapabilityMap
	snap *canonical.CareerSnapshot
	err  error
}

func (f *fakeData) TitleSlug() string                 { return "halo_infinite" }
func (f *fakeData) Capabilities() games.CapabilityMap { return f.caps }
func (f *fakeData) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, nil
}
func (f *fakeData) LoadMatchDetail(_ context.Context, _ string) (*canonical.MatchDetail, error) {
	return nil, nil
}
func (f *fakeData) LoadPlayerStats(_ context.Context, _ string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, nil
}
func (f *fakeData) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return f.snap, f.err
}
func (f *fakeData) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, nil
}
func (f *fakeData) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, nil
}

// fakeSemantic implémente games.TitleSemanticAdapter via un FieldMappingSet réel.
type fakeSemantic struct {
	set *mappings.FieldMappingSet
}

func (f *fakeSemantic) TitleSlug() string                 { return "halo_infinite" }
func (f *fakeSemantic) SchemaVersion() int                { return f.set.SchemaVersion() }
func (f *fakeSemantic) Fields() *mappings.FieldMappingSet { return f.set }
func (f *fakeSemantic) Ranks() *mappings.RankCatalog {
	return mappings.NewRankCatalog("halo_infinite", nil)
}

// fakeResolver utilisé dans les tests preview.
type fakeResolver struct {
	data     games.TitleDataAdapter
	semantic games.TitleSemanticAdapter
	err      error
}

func (f *fakeResolver) Data(slug string) (games.TitleDataAdapter, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}
func (f *fakeResolver) Semantic(slug string) (games.TitleSemanticAdapter, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.semantic, nil
}
func (f *fakeResolver) DefaultSlug() string { return "halo_infinite" }

func newPreviewHandler(t *testing.T, snap *canonical.CareerSnapshot) *MultiTitlePreviewHandler {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("preview.toml", []byte(previewToml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	resolver := &fakeResolver{
		data: &fakeData{
			caps: games.CapabilityMap{games.CapCareerProgression: games.CapSupported},
			snap: snap,
		},
		semantic: &fakeSemantic{set: set},
	}
	return NewMultiTitlePreviewHandler(resolver, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestPreviewCareer_HappyPath(t *testing.T) {
	t.Parallel()
	xp := 500
	xpNext := 2000
	snap := &canonical.CareerSnapshot{
		Player:        canonical.PlayerIdentity{XUID: "0xABC"},
		CurrentXP:     &xp,
		XPForNextRank: &xpNext,
		CurrentRank: &canonical.AssetReference{
			Kind: "career_rank", ID: "diamond_3", DefaultLabel: "Diamant 3",
		},
	}
	h := newPreviewHandler(t, snap)

	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/preview/career", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/preview/career?xuid=0xABC&locale=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var body careerPreviewDTO
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.TitleSlug != "halo_infinite" || body.Locale != "fr" || body.XUID != "0xABC" {
		t.Errorf("DTO meta = %+v", body)
	}
	if body.CurrentRank == nil || body.CurrentRank.DefaultLabel != "Diamant 3" {
		t.Errorf("CurrentRank = %+v", body.CurrentRank)
	}
	if body.CurrentXP == nil || body.CurrentXP.Value != 500 || body.CurrentXP.Label != "XP actuelle" {
		t.Errorf("CurrentXP = %+v", body.CurrentXP)
	}
	if body.XPForNextRank == nil || body.XPForNextRank.Label != "XP rang suivant" {
		t.Errorf("XPForNextRank = %+v", body.XPForNextRank)
	}
}

func TestPreviewCareer_FallbackEN(t *testing.T) {
	t.Parallel()
	xp := 100
	snap := &canonical.CareerSnapshot{
		Player:    canonical.PlayerIdentity{XUID: "0xABC"},
		CurrentXP: &xp,
	}
	h := newPreviewHandler(t, snap)

	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/preview/career", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/preview/career?xuid=0xABC&locale=de", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body careerPreviewDTO
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.CurrentXP == nil {
		t.Fatal("CurrentXP nil")
	}
	if body.CurrentXP.Label != "Current XP" || !body.CurrentXP.Fallback {
		t.Errorf("locale 'de' devrait fallback EN, got label=%q fallback=%v", body.CurrentXP.Label, body.CurrentXP.Fallback)
	}
}

func TestPreviewCareer_CapabilityNotSupported(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{
		data: &fakeData{
			caps: games.CapabilityMap{games.CapCareerProgression: games.CapNotExposed},
			err:  games.ErrCapabilityNotSupported,
		},
		semantic: &fakeSemantic{set: mustLoadPreview(t)},
	}
	h := NewMultiTitlePreviewHandler(resolver, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/preview/career", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/preview/career?xuid=0xABC", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body careerPreviewDTO
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.NotSupportedReason == "" {
		t.Errorf("NotSupportedReason vide, want 'career_progression_not_exposed_for_title'")
	}
	if body.CurrentXP != nil {
		t.Errorf("CurrentXP non nil malgré capability not_exposed")
	}
}

func TestPreviewCareer_TitleNotFound(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{err: games.ErrTitleNotResolved}
	h := NewMultiTitlePreviewHandler(resolver, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/preview/career", h.GetCareerPreview)

	req := httptest.NewRequest("GET", "/api/v1/titles/unknown/preview/career?xuid=0xABC", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func mustLoadPreview(t *testing.T) *mappings.FieldMappingSet {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("preview.toml", []byte(previewToml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return set
}
