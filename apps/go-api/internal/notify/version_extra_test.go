package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLastNotifiedVersion_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(p, []byte("not json"), 0o644)
	v := readLastNotifiedVersion(p)
	if v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
}

func TestReadLastNotifiedVersion_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(p, []byte(`{"last_notified_version":"1.2.3"}`), 0o644)
	v := readLastNotifiedVersion(p)
	if v != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %q", v)
	}
}

func TestWriteLastNotifiedVersion_NewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	if err := writeLastNotifiedVersion(p, "2.0.0"); err != nil {
		t.Fatal(err)
	}
	v := readLastNotifiedVersion(p)
	if v != "2.0.0" {
		t.Fatalf("expected 2.0.0, got %q", v)
	}
}

func TestWriteLastNotifiedVersion_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(p, []byte(`{"theme":"dark","last_notified_version":"1.0.0"}`), 0o644)
	if err := writeLastNotifiedVersion(p, "1.1.0"); err != nil {
		t.Fatal(err)
	}
	v := readLastNotifiedVersion(p)
	if v != "1.1.0" {
		t.Fatalf("expected 1.1.0, got %q", v)
	}
}

func TestExtractWhatsNew_MissingReadme(t *testing.T) {
	result := extractWhatsNew("v1.0", "/nonexistent/settings.json")
	if result != "" {
		t.Fatalf("expected empty, got %q", result)
	}
}

func TestExtractWhatsNew_WithReadme(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	content := "# LevelUp\n\n**v2.1** — New features\n- Feature A\n- Feature B\n\n**v2.0** — Old\n- Old feature\n"
	_ = os.WriteFile(readme, []byte(content), 0o644)

	settings := filepath.Join(dir, "settings.json")
	result := extractWhatsNew("v2.1.0", settings)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestExtractWhatsNew_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	_ = os.WriteFile(readme, []byte("# test"), 0o644)

	settings := filepath.Join(dir, "settings.json")
	result := extractWhatsNew("x", settings)
	if result != "" {
		t.Fatalf("expected empty for invalid version, got %q", result)
	}
}
