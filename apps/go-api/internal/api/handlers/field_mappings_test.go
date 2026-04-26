package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
	r.Get("/api/v1/titles/{slug}/field-mappings", h.ServeHTTP)

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
	r.Get("/api/v1/titles/{slug}/field-mappings", h.ServeHTTP)

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
	r.Get("/api/v1/titles/{slug}/field-mappings", h.ServeHTTP)

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
	r.Get("/api/v1/titles/{slug}/field-mappings", h.ServeHTTP)

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

func TestMultiTitleAPIEnabled(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
	}
	for _, tc := range cases {
		t.Setenv("MULTI_TITLE_API_ENABLED", tc.val)
		if got := MultiTitleAPIEnabled(); got != tc.want {
			t.Errorf("env=%q → %v, want %v", tc.val, got, tc.want)
		}
	}
	_ = os.Unsetenv("MULTI_TITLE_API_ENABLED")
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
