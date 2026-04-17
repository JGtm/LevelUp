package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractWhatsNew_NoFile(t *testing.T) {
	got := extractWhatsNew("1.0.0", "/nonexistent/dir/settings.json")
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestExtractWhatsNew_VersionTooShort(t *testing.T) {
	got := extractWhatsNew("1", "/some/settings.json")
	if got != "" {
		t.Errorf("expected empty for single-part version, got %s", got)
	}
}

func TestExtractWhatsNew_FoundSection(t *testing.T) {
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")
	content := `# LevelUp

## Changelog

**v2.1** — Nouvelle version
- Ajout feature A
- Fix bug B

**v2.0** — Ancienne version
- Feature C
`
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(tmpDir, "app_settings.json")
	got := extractWhatsNew("v2.1.0", settingsPath)
	if got == "" {
		t.Error("expected non-empty result")
		return
	}
	if len(got) < 10 {
		t.Errorf("expected longer result, got: %s", got)
	}
}

func TestExtractWhatsNew_SectionNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")
	content := `# LevelUp
**v1.0** — old version
- something
`
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(tmpDir, "app_settings.json")
	got := extractWhatsNew("v3.0.0", settingsPath)
	if got != "" {
		t.Errorf("expected empty for missing version, got %s", got)
	}
}
