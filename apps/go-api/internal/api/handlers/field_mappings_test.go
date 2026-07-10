package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
)

const handlerTestTOML = `
[meta]
title_slug     = "test_title"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
description   = { en = "Total kills.", fr = "Total des éliminations." }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"
`

type stubRegistry struct {
	set      map[string]*mappings.FieldMappingSet
	assets   map[string]*mappings.AssetMappingSet
	outcomes map[string]*mappings.OutcomeMappingSet
}

func (s *stubRegistry) Get(slug string) (*mappings.FieldMappingSet, bool) {
	v, ok := s.set[slug]
	return v, ok
}

func (s *stubRegistry) GetAssets(slug string) (*mappings.AssetMappingSet, bool) {
	v, ok := s.assets[slug]
	return v, ok
}

func (s *stubRegistry) GetOutcomes(slug string) (*mappings.OutcomeMappingSet, bool) {
	v, ok := s.outcomes[slug]
	return v, ok
}

func newHandler(reg FieldMappingsRegistry) *FieldMappingsHandler {
	return NewFieldMappingsHandler(reg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestFieldMappingsHandler_Success_FR(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	if etag := w.Header().Get("ETag"); etag == "" {
		t.Errorf("ETag absent")
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("Cache-Control = %q", cc)
	}

	var body fieldMappingsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.TitleSlug != "test_title" {
		t.Errorf("title_slug = %q", body.TitleSlug)
	}
	if body.SchemaVersion != 1 {
		t.Errorf("schema_version = %d", body.SchemaVersion)
	}
	kills := body.Fields["kills"]
	if kills.Label != "Éliminations" {
		t.Errorf("kills FR label = %q", kills.Label)
	}
	if kills.Description != "Total des éliminations." {
		t.Errorf("kills FR description = %q", kills.Description)
	}
}

func TestFieldMappingsHandler_FallbackEN_OnUnknownLocale(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings?locale=es", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body fieldMappingsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Fields["kills"].Label != "Kills" {
		t.Errorf("locale inconnue devrait fallback EN, got %q", body.Fields["kills"].Label)
	}
}

func TestFieldMappingsHandler_NotFound(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/unknown_title/field-mappings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestFieldMappingsHandler_ETag304(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("first request: ETag absent")
	}

	req2 := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Errorf("re-request with same ETag: status = %d, want 304", w2.Code)
	}
}

// mustLoad charge le fixture TOML et retourne le map slug→set utilisé par stubRegistry.
func mustLoad(t *testing.T) map[string]*mappings.FieldMappingSet {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("test.toml", []byte(handlerTestTOML))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return map[string]*mappings.FieldMappingSet{"test_title": set}
}

// ─── Saisons : DTO assets expose start_date / end_date / extra ───────────────

const handlerSeasonAssetsTOML = `
[meta]
title_slug     = "test_title"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50

[assets.season.season6]
labels = { en = "Spirit of Fire", fr = "Spirit of Fire" }
display_order = 60
start_date = "2024-03-19T00:00:00Z"
end_date   = "2024-06-18T00:00:00Z"
extra = { csr_season_id = "CsrSeason6", short_label = "S6" }
`

func mustLoadAssets(t *testing.T) map[string]*mappings.AssetMappingSet {
	t.Helper()
	set, err := mappings.LoadAssetsFromBytes("test.toml", []byte(handlerSeasonAssetsTOML))
	if err != nil {
		t.Fatalf("load season assets fixture: %v", err)
	}
	return map[string]*mappings.AssetMappingSet{"test_title": set}
}

func TestFieldMappingsHandler_SeasonAssetIncludesDates(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var body fieldMappingsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	season := body.Assets["season"]["season6"]
	if season.Label != "Spirit of Fire" {
		t.Errorf("season6 label = %q, want Spirit of Fire", season.Label)
	}
	if season.StartDate == nil || season.StartDate.Format("2006-01-02") != "2024-03-19" {
		t.Errorf("season6 start_date = %v, want 2024-03-19", season.StartDate)
	}
	if season.EndDate == nil || season.EndDate.Format("2006-01-02") != "2024-06-18" {
		t.Errorf("season6 end_date = %v, want 2024-06-18", season.EndDate)
	}
	if season.Extra["csr_season_id"] != "CsrSeason6" {
		t.Errorf("season6 extra.csr_season_id = %q, want CsrSeason6", season.Extra["csr_season_id"])
	}
	if season.Extra["short_label"] != "S6" {
		t.Errorf("season6 extra.short_label = %q, want S6", season.Extra["short_label"])
	}
}

func TestFieldMappingsHandler_OtherKindsOmitDateFields(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}
	h := newHandler(stub)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Vérifier dans le JSON brut que les champs date n'apparaissent pas pour mode.ranked.
	raw := w.Body.String()
	rankedMarker := `"ranked":{`
	idx := strings.Index(raw, rankedMarker)
	if idx < 0 {
		t.Fatalf("ranked entry absent du JSON")
	}
	// Extraire l'objet ranked (recherche grossière jusqu'au '}').
	end := strings.Index(raw[idx:], "}")
	if end < 0 {
		t.Fatalf("ranked object malformé")
	}
	rankedObj := raw[idx : idx+end+1]
	if strings.Contains(rankedObj, `"start_date"`) {
		t.Errorf("ranked DTO contient start_date (devrait être omitempty) : %s", rankedObj)
	}
	if strings.Contains(rankedObj, `"end_date"`) {
		t.Errorf("ranked DTO contient end_date : %s", rankedObj)
	}
	if strings.Contains(rankedObj, `"extra"`) {
		t.Errorf("ranked DTO contient extra : %s", rankedObj)
	}
}

// ─── V2 saisons : merge DB+TOML via SeasonsCatalogResolver ─────────────────

// fakeSeasonsCatalog implémente SeasonsCatalogResolver en mémoire.
type fakeSeasonsCatalog struct {
	entries []SeasonCatalogEntry
}

func (f *fakeSeasonsCatalog) Load(_ context.Context, _ string) []SeasonCatalogEntry {
	return f.entries
}

func TestFieldMappingsHandler_SeasonsCatalog_OverridesTOMLBucket(t *testing.T) {
	t.Parallel()
	// TOML : season6 (FR : "Spirit of Fire")
	// Catalog (TOML+DB merged) : season6 (DB-fresh dates) + season14 (DB-only "Skyfall")
	// Le handler doit retourner les 2 dans assets.season, avec les dates et
	// labels du catalog (qui sont la source de vérité).
	stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}

	mergedStart6 := time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC)
	mergedEnd6 := time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC) // valeur DB
	skyStart := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	catalog := &fakeSeasonsCatalog{
		entries: []SeasonCatalogEntry{
			{
				ID:           "season6",
				Label:        "Spirit of Fire",
				Start:        mergedStart6,
				End:          &mergedEnd6,
				DisplayOrder: 60,
				Extra:        map[string]string{"csr_season_id": "CsrSeason6", "short_label": "S6"},
			},
			{
				ID:           "season14",
				Label:        "Skyfall", // DB-only (pas de TOML correspondant)
				Start:        skyStart,
				End:          nil,
				DisplayOrder: 70,
			},
		},
	}

	h := newHandler(stub).WithSeasonsCatalog(catalog)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	var body fieldMappingsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	seasons := body.Assets["season"]
	if len(seasons) != 2 {
		t.Fatalf("len(assets.season) = %d, want 2 (merge TOML+DB)", len(seasons))
	}

	s6 := seasons["season6"]
	if s6.EndDate == nil || s6.EndDate.Day() != 18 {
		t.Errorf("season6 end_date = %v, want 2024-06-18 (DB wins)", s6.EndDate)
	}
	if s6.Extra["csr_season_id"] != "CsrSeason6" {
		t.Errorf("season6 extra perdue : %v", s6.Extra)
	}

	s14, ok := seasons["season14"]
	if !ok {
		t.Fatal("season14 absente — DB-only saison perdue dans le merge")
	}
	if s14.Label != "Skyfall" {
		t.Errorf("season14 label = %q, want Skyfall (fallback Name Waypoint)", s14.Label)
	}
	if s14.EndDate != nil {
		t.Errorf("season14 end_date = %v, want nil (saison ouverte)", s14.EndDate)
	}

	// Les autres kinds (mode.ranked) restent intacts (fallback FR par défaut).
	if body.Assets["mode"]["ranked"].Label != "Classé" {
		t.Errorf("mode.ranked label perdu : %v", body.Assets["mode"]["ranked"])
	}
}

func TestFieldMappingsHandler_SeasonsCatalogEmpty_FallsBackToTOML(t *testing.T) {
	t.Parallel()
	// Catalog vide → le handler doit conserver les saisons du TOML
	// (dégradation gracieuse, pas de crash).
	stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}
	catalog := &fakeSeasonsCatalog{entries: nil}
	h := newHandler(stub).WithSeasonsCatalog(catalog)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body fieldMappingsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)

	// Le bucket season du TOML est conservé tel quel quand le catalog est vide.
	if len(body.Assets["season"]) != 1 || body.Assets["season"]["season6"].Label != "Spirit of Fire" {
		t.Errorf("catalog vide → TOML conservé attendu, got %v", body.Assets["season"])
	}
}

// TestFieldMappingsHandler_SeasonsCatalog_LocaleAware prouve GH3-1 : la liste
// des saisons de la SaisonPill suit la locale de requête. season2 a des libellés
// distincts (FR "Loups solitaires" / EN "Lone Wolves") ; le bucket season doit
// servir le bon selon ?locale=. Une saison DB-only sans traduction (LabelEN vide)
// garde son Name brut dans les deux locales.
func TestFieldMappingsHandler_SeasonsCatalog_LocaleAware(t *testing.T) {
	t.Parallel()
	start := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	dbOnlyStart := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	catalog := &fakeSeasonsCatalog{
		entries: []SeasonCatalogEntry{
			{ID: "season2", Label: "Loups solitaires", LabelEN: "Lone Wolves", Start: start, DisplayOrder: 20},
			// DB-only : pas de traduction → même Name FR et EN.
			{ID: "season14", Label: "Skyfall", LabelEN: "Skyfall", Start: dbOnlyStart, DisplayOrder: 70},
		},
	}

	load := func(locale string) map[string]assetMappingDTO {
		t.Helper()
		stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}
		h := newHandler(stub).WithSeasonsCatalog(catalog)
		r := chi.NewRouter()
		r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })
		req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings?locale="+locale, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("locale=%s status = %d, body=%s", locale, w.Code, w.Body.String())
		}
		var body fieldMappingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("locale=%s unmarshal: %v", locale, err)
		}
		return body.Assets["season"]
	}

	fr := load("fr")
	if fr["season2"].Label != "Loups solitaires" {
		t.Errorf("FR season2 label = %q, want Loups solitaires", fr["season2"].Label)
	}
	en := load("en")
	if en["season2"].Label != "Lone Wolves" {
		t.Errorf("EN season2 label = %q, want Lone Wolves (jamais le FR sous EN)", en["season2"].Label)
	}
	// DB-only : identique dans les deux locales.
	if fr["season14"].Label != "Skyfall" || en["season14"].Label != "Skyfall" {
		t.Errorf("season14 DB-only label FR=%q EN=%q, want Skyfall/Skyfall", fr["season14"].Label, en["season14"].Label)
	}
}
