//go:build cgo

package api

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

func writeTmpChangelog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestRegisterChangelogHuma_ContractPreserved (Phase 3b, 1re route migrée) : la
// route GET /changelog migrée vers Huma préserve STRICTEMENT le contrat —
// 200 {content} SANS champ $schema, et 404 au format writeError
// {code:CHANGELOG_NOT_FOUND, message, retryable:false}. C'est le golden qui
// valide le pattern de migration (error-model + $schema off) bout-en-bout.
func TestRegisterChangelogHuma_ContractPreserved(t *testing.T) {
	// 200 — contenu présent.
	r := chi.NewRouter()
	registerChangelogHuma(newHumaAPI(r), handlers.NewChangelogHandler(writeTmpChangelog(t, "# CL\n- x")))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/changelog", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("200 attendu, got %d: %s", rec.Code, rec.Body.String())
	}
	var ok map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if ok["content"] != "# CL\n- x" {
		t.Errorf("content = %v", ok["content"])
	}
	if _, has := ok["$schema"]; has {
		t.Error("$schema ne doit PAS être injecté (contrat front identique à writeJSON)")
	}

	// 404 — CHANGELOG.md absent → format d'erreur writeError.
	r2 := chi.NewRouter()
	registerChangelogHuma(newHumaAPI(r2), handlers.NewChangelogHandler(t.TempDir()))
	rec2 := httptest.NewRecorder()
	r2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/changelog", nil))
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("404 attendu, got %d", rec2.Code)
	}
	var er map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &er); err != nil {
		t.Fatalf("JSON erreur invalide: %v", err)
	}
	if er["code"] != "CHANGELOG_NOT_FOUND" {
		t.Errorf("code = %v, want CHANGELOG_NOT_FOUND", er["code"])
	}
	if er["retryable"] != false {
		t.Errorf("retryable = %v, want false", er["retryable"])
	}
}
