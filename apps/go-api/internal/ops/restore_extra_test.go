// Package ops — restore_extra_test.go : tests des fonctions filesystem pures
// FindAvailableBackups et ReadBackupMetadata (sans DuckDB requis).
package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// FindAvailableBackups
// ─────────────────────────────────────────────────────────────────────────────

func TestFindAvailableBackups_DirAbsent(t *testing.T) {
	results, err := FindAvailableBackups("/nonexistent/backup/dir")
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 backups, got %d", len(results))
	}
}

func TestFindAvailableBackups_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := FindAvailableBackups(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 backups in empty dir, got %d", len(results))
	}
}

func TestFindAvailableBackups_WithBackups(t *testing.T) {
	dir := t.TempDir()

	// Créer des fichiers backup_metadata_*.json
	for _, ts := range []string{"20250101_120000", "20250215_090000", "20250310_180000"} {
		name := "backup_metadata_" + ts + ".json"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Ajouter un fichier non-metadata (doit être ignoré)
	if err := os.WriteFile(filepath.Join(dir, "other_file.parquet"), []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	results, err := FindAvailableBackups(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 backups, got %d", len(results))
	}
	// Doit être trié en ordre décroissant (plus récent en premier)
	if results[0] <= results[1] {
		t.Errorf("attendu ordre décroissant: %v", results)
	}
}

func TestFindAvailableBackups_Deduplication(t *testing.T) {
	dir := t.TempDir()
	// Même timestamp deux fois (ne devrait pas arriver mais robustesse)
	ts := "20250101_120000"
	if err := os.WriteFile(filepath.Join(dir, "backup_metadata_"+ts+".json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	results, err := FindAvailableBackups(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 unique backup, got %d", len(results))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ReadBackupMetadata
// ─────────────────────────────────────────────────────────────────────────────

func TestReadBackupMetadata_Absent(t *testing.T) {
	_, err := ReadBackupMetadata("/nonexistent", "20250101_120000")
	if err == nil {
		t.Error("expected error (fichier absent)")
	}
}

func TestReadBackupMetadata_Valid(t *testing.T) {
	dir := t.TempDir()
	ts := "20250215_090000"
	meta := map[string]any{
		"gamertag":   "TestPlayer",
		"created_at": "2025-02-15T09:00:00Z",
		"tables":     []string{"player_match_enrichment", "career_progression"},
	}
	data, _ := json.Marshal(meta)
	filename := "backup_metadata_" + ts + ".json"
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ReadBackupMetadata(dir, ts)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result["gamertag"] != "TestPlayer" {
		t.Errorf("gamertag = %v, want TestPlayer", result["gamertag"])
	}
}

func TestReadBackupMetadata_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	ts := "20250301_120000"
	filename := "backup_metadata_" + ts + ".json"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadBackupMetadata(dir, ts)
	if err == nil {
		t.Error("expected error (JSON invalide)")
	}
}
