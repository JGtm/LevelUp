// Package handlers_test — changelog_test.go : tests ChangelogHandler.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
)

func TestChangelogHandler_OK(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Changelog\n## v1.0\n- Initial release"
	if err := os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	h := handlers.NewChangelogHandler(dir)
	r := chi.NewRouter()
	r.Get("/changelog", h.GetChangelog)

	req := httptest.NewRequest(http.MethodGet, "/changelog", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["content"] != content {
		t.Errorf("content mismatch: got %q", resp["content"])
	}
}

func TestChangelogHandler_NotFound(t *testing.T) {
	dir := t.TempDir() // pas de docs/CHANGELOG.md
	h := handlers.NewChangelogHandler(dir)
	r := chi.NewRouter()
	r.Get("/changelog", h.GetChangelog)

	req := httptest.NewRequest(http.MethodGet, "/changelog", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestChangelogHandler_CacheHit(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	_ = os.MkdirAll(docsDir, 0o755)
	_ = os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte("v1"), 0o600)

	h := handlers.NewChangelogHandler(dir)
	r := chi.NewRouter()
	r.Get("/changelog", h.GetChangelog)

	// Premier appel → charge du disque
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/changelog", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", w1.Code)
	}

	// Modifier le fichier → le cache ne doit PAS relire immédiatement
	_ = os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte("v2"), 0o600)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/changelog", nil))

	var resp map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["content"] != "v1" {
		t.Error("expected cached value 'v1' on second call within TTL")
	}
}
