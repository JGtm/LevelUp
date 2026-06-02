package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
)

type stubCapabilitiesRegistry struct {
	set *mappings.CapabilityMappingSet
}

func (s *stubCapabilitiesRegistry) GetCapabilities(slug string) (*mappings.CapabilityMappingSet, bool) {
	if s.set != nil && slug == s.set.TitleSlug() {
		return s.set, true
	}
	return nil, false
}

func newCapsHandler() *CapabilitiesHandler {
	set := mappings.NewCapabilityMappingSet("test_title", 1, map[string]string{
		"match.history":        mappings.CapStatusSupported,
		"analytics.timeseries": mappings.CapStatusNotExposed,
	})
	return NewCapabilitiesHandler(&stubCapabilitiesRegistry{set: set}, nil)
}

func TestCapabilitiesHandler_Success(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/capabilities", newCapsHandler().ServeHTTP)

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/capabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body capabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.TitleSlug != "test_title" || body.SchemaVersion != 1 {
		t.Errorf("meta: got (%q, %d)", body.TitleSlug, body.SchemaVersion)
	}
	if body.Capabilities["match.history"] != mappings.CapStatusSupported {
		t.Errorf("match.history = %q, want supported", body.Capabilities["match.history"])
	}
	if w.Header().Get("ETag") == "" {
		t.Errorf("ETag absent")
	}
}

func TestCapabilitiesHandler_NotFound(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/capabilities", newCapsHandler().ServeHTTP)

	req := httptest.NewRequest("GET", "/api/v1/titles/unknown/capabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestCapabilitiesHandler_ETag304(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Get("/api/v1/titles/{slug}/capabilities", newCapsHandler().ServeHTTP)

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/capabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("ETag absent au 1er appel")
	}

	req2 := httptest.NewRequest("GET", "/api/v1/titles/test_title/capabilities", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", w2.Code)
	}
}
