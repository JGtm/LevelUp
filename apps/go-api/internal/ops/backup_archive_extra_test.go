// Package ops — backup_archive_extra_test.go : tests purs filesystem pour
// writeBackupMetadata et writeArchiveIndex (pas de DuckDB requis).
package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// writeBackupMetadata
// ─────────────────────────────────────────────────────────────────────────────

func TestWriteBackupMetadata_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup_meta.json")

	result := BackupResult{
		Timestamp: "2025-01-01T12:00:00Z",
		Tables:    map[string]TableBackupInfo{"player_match_enrichment": {Rows: 100, ParquetPath: "player_match_enrichment.parquet"}},
	}

	err := writeBackupMetadata(path, "TestPlayer", 9, result)
	if err != nil {
		t.Fatalf("writeBackupMetadata inattendu: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var meta backupMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Gamertag != "TestPlayer" {
		t.Errorf("Gamertag = %q, want TestPlayer", meta.Gamertag)
	}
	if meta.Compression != "zstd" {
		t.Errorf("Compression = %q, want zstd", meta.Compression)
	}
	if meta.Level != 9 {
		t.Errorf("Level = %d, want 9", meta.Level)
	}
	if len(meta.Tables) != 1 {
		t.Errorf("Tables len = %d, want 1", len(meta.Tables))
	}
}

func TestWriteBackupMetadata_InvalidPath(t *testing.T) {
	err := writeBackupMetadata("/nonexistent/dir/meta.json", "Player", 9, BackupResult{})
	if err == nil {
		t.Error("expected error pour path invalide")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// writeArchiveIndex
// ─────────────────────────────────────────────────────────────────────────────

func TestWriteArchiveIndex_Valid(t *testing.T) {
	dir := t.TempDir()
	cutoff := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	files := []string{"2024_q1.parquet", "2024_q2.parquet"}

	err := writeArchiveIndex(dir, "TestPlayer", cutoff, 250, files)
	if err != nil {
		t.Fatalf("writeArchiveIndex inattendu: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "archive_index.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var idx archiveIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if idx.Gamertag != "TestPlayer" {
		t.Errorf("Gamertag = %q, want TestPlayer", idx.Gamertag)
	}
	if idx.MatchCount != 250 {
		t.Errorf("MatchCount = %d, want 250", idx.MatchCount)
	}
	if len(idx.Files) != 2 {
		t.Errorf("Files len = %d, want 2", len(idx.Files))
	}
}

func TestWriteArchiveIndex_EmptyFiles(t *testing.T) {
	dir := t.TempDir()
	cutoff := time.Now()
	err := writeArchiveIndex(dir, "Player2", cutoff, 0, nil)
	if err != nil {
		t.Fatalf("writeArchiveIndex inattendu: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "archive_index.json")); statErr != nil {
		t.Error("archive_index.json doit exister")
	}
}

func TestWriteArchiveIndex_InvalidPath(t *testing.T) {
	err := writeArchiveIndex("/nonexistent/dir", "Player", time.Now(), 0, nil)
	if err == nil {
		t.Error("expected error pour dir invalide")
	}
}
