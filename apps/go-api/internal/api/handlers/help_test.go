// Package handlers_test — help_test.go : tests HelpHandler.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/service"
)

// noopGit implémente port.GitProvider en retournant des résultats vides.
// Utilisé par les tests qui veulent s'appuyer uniquement sur le README disque.
type noopGit struct{}

func (noopGit) LogSHAs(_ context.Context, _, _ string) ([]string, error) { return nil, nil }
func (noopGit) ShowFile(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

// makeHelpHandler instancie un HelpHandler avec le service release notes
// configuré pour ne pas appeler git (P8.10) — fallback disque uniquement.
func makeHelpHandler(t *testing.T, dir string) *handlers.HelpHandler {
	t.Helper()
	builder := service.NewReleaseNotesService(dir, noopGit{})
	return handlers.NewHelpHandler(builder, filepath.Join(dir, "data", "cache"))
}

// setupHelpRepo crée un repo git minimal dans un répertoire temporaire
// avec un README.md contenant les sections What's new spécifiées.
func setupHelpRepo(t *testing.T, readmeEN, readmeFR string) string {
	t.Helper()
	dir := t.TempDir()

	// Initialiser un repo git bare (pas de vrai git init nécessaire pour les tests
	// sans git — on teste le fallback sur disque).
	docsDir := filepath.Join(dir, "docs", "FR")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeEN), 0o600); err != nil {
		t.Fatal(err)
	}
	if readmeFR != "" {
		if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte(readmeFR), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const sampleReadmeEN = `# LevelUp

## What's new

**v7.0 — Challenges**
- Feature A
- Feature B

**v6.5 — Heatmap**
- Feature C

## Features

Some content.
`

const sampleReadmeFR = `# LevelUp

## Dernières nouveautés

**v7.0 — Défis**
- Fonctionnalité A

**v6.5 — Heatmap**
- Fonctionnalité C

## Fonctionnalités
`

func TestHelpHandler_EN_ReturnsWhatsNew(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN, sampleReadmeFR)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	content := resp["content"]
	if !contains(content, "v7.0") {
		t.Errorf("expected v7.0 block in content, got: %q", content)
	}
	if !contains(content, "v6.5") {
		t.Errorf("expected v6.5 block in content, got: %q", content)
	}
	if contains(content, "Some content") {
		t.Error("should not include content after ## Features heading")
	}
}

func TestHelpHandler_FR_ReturnsWhatsNew(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN, sampleReadmeFR)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=fr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	content := resp["content"]
	if !contains(content, "Défis") {
		t.Errorf("expected FR content (Défis), got: %q", content)
	}
}

func TestHelpHandler_DefaultLangFR(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN, sampleReadmeFR)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	// Pas de paramètre lang → défaut FR
	req := httptest.NewRequest(http.MethodGet, "/help/release-notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !contains(resp["content"], "Défis") {
		t.Errorf("expected FR default content, got: %q", resp["content"])
	}
}

func TestHelpHandler_MissingReadme_Returns500(t *testing.T) {
	dir := t.TempDir() // Pas de README.md
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHelpHandler_CacheHit(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN, "")
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", w1.Code)
	}

	// Modifier le fichier → le cache ne relit pas immédiatement
	newContent := "# Modified\n## What's new\n**v99.0 — New**\n- Changed\n## Other\n"
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte(newContent), 0o600)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil))

	var resp map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if contains(resp["content"], "v99.0") {
		t.Error("cache should not have been invalidated within TTL")
	}
}

func TestHelpHandler_VersionOrder(t *testing.T) {
	readme := `# LevelUp
## What's new
**v6.0 — Old**
- old feature
**v7.0 — New**
- new feature
**v6.5 — Mid**
- mid feature
## Features
`
	dir := setupHelpRepo(t, readme, "")
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	r.Get("/help/release-notes", h.GetReleaseNotes)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	content := resp["content"]

	pos70 := indexOf(content, "v7.0")
	pos65 := indexOf(content, "v6.5")
	pos60 := indexOf(content, "v6.0")
	if pos70 < 0 || pos65 < 0 || pos60 < 0 {
		t.Fatalf("missing version blocks: %q", content)
	}
	if pos70 >= pos65 || pos65 >= pos60 {
		t.Errorf("expected v7.0 > v6.5 > v6.0, positions: %d %d %d in:\n%s", pos70, pos65, pos60, content)
	}
}

func TestHelpHandler_DiskCacheSurvivesRestart(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN, "")
	// Premier handler — construit le cache et l'écrit sur disque.
	h1 := makeHelpHandler(t, dir)
	r1 := chi.NewRouter()
	r1.Get("/help/release-notes", h1.GetReleaseNotes)
	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first handler: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var resp1 map[string]string
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	original := resp1["content"]

	// Deuxième handler (simule un redémarrage) — mémoire vide, doit utiliser le disque.
	h2 := makeHelpHandler(t, dir)
	r2 := chi.NewRouter()
	r2.Get("/help/release-notes", h2.GetReleaseNotes)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/help/release-notes?lang=en", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("second handler: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var resp2 map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp2["content"] != original {
		t.Errorf("disk cache should return same content after restart\ngot: %q\nwant: %q", resp2["content"], original)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
