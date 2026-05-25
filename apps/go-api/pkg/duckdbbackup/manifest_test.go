package duckdbbackup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".manifest.json")

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest on absent file: %v", err)
	}
	if len(m.Databases) != 0 {
		t.Fatal("expected empty manifest")
	}

	// Create a dummy DB file to fingerprint.
	dbPath := filepath.Join(dir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := Target{Key: "test", Path: dbPath}

	changed, err := m.HasChanged(target)
	if err != nil {
		t.Fatalf("HasChanged on new target: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true for unseen target")
	}

	if err := m.MarkSaved(target); err != nil {
		t.Fatalf("MarkSaved: %v", err)
	}
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload and verify fingerprint matches.
	m2, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest after save: %v", err)
	}
	changed2, err := m2.HasChanged(target)
	if err != nil {
		t.Fatalf("HasChanged after save: %v", err)
	}
	if changed2 {
		t.Fatal("expected changed=false after saving fingerprint")
	}
}

func TestManifest_DetectsChange(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	if err := os.WriteFile(dbPath, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := Target{Key: "test", Path: dbPath}

	m, _ := LoadManifest(filepath.Join(dir, ".manifest.json"))
	_ = m.MarkSaved(target)
	_ = m.Save()

	// Modify the file and advance mtime by 1 second.
	if err := os.WriteFile(dbPath, []byte("v1v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(dbPath, future, future); err != nil {
		t.Fatal(err)
	}

	changed, err := m.HasChanged(target)
	if err != nil {
		t.Fatalf("HasChanged: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true after file modification")
	}
}

func TestManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	m, _ := LoadManifest(filepath.Join(dir, ".manifest.json"))

	target := Target{Key: "missing", Path: filepath.Join(dir, "nonexistent.duckdb")}
	_, err := m.HasChanged(target)
	if err == nil {
		t.Fatal("expected error for missing DB file")
	}
}
