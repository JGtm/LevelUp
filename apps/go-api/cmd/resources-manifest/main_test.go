package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/resourcepack"
)

func TestRunEcritLeManifesteEtImprimeLEmpreinte(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "static", "a.png"), []byte("A"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	zipPath := filepath.Join(t.TempDir(), "pack.zip")
	if code := run([]string{"-dir", dir, "-version", "1.2.3", "-zip", zipPath}, out, os.Stderr); code != 0 {
		t.Fatalf("code %d", code)
	}
	raw, _ := os.ReadFile(out.Name())
	printed := strings.TrimSpace(string(raw))
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(printed) {
		t.Fatalf("sortie inattendue : %q", printed)
	}
	m, err := resourcepack.ReadManifest(dir)
	if err != nil || m.Fingerprint != printed || m.Version != "1.2.3" {
		t.Fatalf("manifeste : %v %+v", err, m)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("archive absente : %v", err)
	}
	if code := run([]string{}, out, os.Stderr); code != 2 {
		t.Fatalf("sans -dir : code %d", code)
	}
	if code := run([]string{"-dir", filepath.Join(dir, "absent")}, out, os.Stderr); code != 1 {
		t.Fatalf("dossier absent : code %d", code)
	}
}
