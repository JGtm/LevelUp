// Package handlers_test — changelog_test.go : tests de la logique de cache du
// ChangelogHandler (Content). La route HTTP GET /changelog est migrée vers Huma
// (Phase 3b) et testée au contrat dans internal/api (huma_routes_test.go).
package handlers_test

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/api/handlers"
)

func writeChangelog(t *testing.T, dir, content string) {
	t.Helper()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestChangelogHandler_Content_OK(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n## v1.0\n- Initial release"
	writeChangelog(t, dir, content)

	got, err := handlers.NewChangelogHandler(dir).Content()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("content mismatch: got %q", got)
	}
}

func TestChangelogHandler_Content_NotFound(t *testing.T) {
	got, err := handlers.NewChangelogHandler(t.TempDir()).Content()
	if err == nil {
		t.Fatalf("expected error for missing CHANGELOG.md, got content %q", got)
	}
}

func TestChangelogHandler_Content_CacheHit(t *testing.T) {
	dir := t.TempDir()
	writeChangelog(t, dir, "v1")
	h := handlers.NewChangelogHandler(dir)

	if got, _ := h.Content(); got != "v1" {
		t.Fatalf("first call: got %q want v1", got)
	}
	// Modifier le fichier → le cache (TTL) ne doit PAS relire immédiatement.
	writeChangelog(t, dir, "v2")
	if got, _ := h.Content(); got != "v1" {
		t.Error("expected cached 'v1' within TTL on second call")
	}
}
