//go:build cgo

// Package ops — seed_cgo_test.go : tests de SeedCareerRanks, SeedCitationMappings,
// SeedMedalDefinitions avec une DB DuckDB de fichier temporaire.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openTempDB crée une DuckDB temporaire sur disque et retourne son chemin.
func openTempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	// Créer la DB en l'ouvrant et la fermant immédiatement
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("openTempDB: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("openTempDB ping: %v", err)
	}
	db.Close()
	return dbPath
}

// ─────────────────────────────────────────────────────────────────────────────
// SeedCareerRanks
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedCareerRanks_JSONAbsent(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	_, err := SeedCareerRanks(SeedOptions{MetaDBPath: dbPath, DataDir: dir})
	if err == nil {
		t.Error("expected error (JSON absent)")
	}
}

func TestSeedCareerRanks_ValidRanks(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	// Créer le JSON source minimal
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	ranks := []map[string]any{
		{"rank_id": 1, "title_en": "Recruit", "title_fr": "Recrue", "subtitle": "", "tier": "Bronze", "grade": 1, "xp_required": 0, "icon_path": ""},
		{"rank_id": 2, "title_en": "Private", "title_fr": "Soldat", "subtitle": "", "tier": "Bronze", "grade": 2, "xp_required": 1000, "icon_path": ""},
	}
	data, _ := json.Marshal(ranks)
	jsonPath := filepath.Join(cacheDir, "career_ranks_metadata.json")
	if err := os.WriteFile(jsonPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	result, err := SeedCareerRanks(SeedOptions{MetaDBPath: dbPath, DataDir: dir})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result.Inserted != 2 {
		t.Errorf("inserted = %d, want 2", result.Inserted)
	}
	if result.Component != "career_ranks" {
		t.Errorf("component = %q, want career_ranks", result.Component)
	}
}

func TestSeedCareerRanks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	ranks := []map[string]any{
		{"rank_id": 1, "title_en": "R1", "title_fr": "R1", "subtitle": "", "tier": "T", "grade": 1, "xp_required": 0, "icon_path": ""},
	}
	data, _ := json.Marshal(ranks)
	jsonPath := filepath.Join(cacheDir, "career_ranks_metadata.json")
	if err := os.WriteFile(jsonPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	opts := SeedOptions{MetaDBPath: dbPath, DataDir: dir}
	r1, err := SeedCareerRanks(opts)
	if err != nil {
		t.Fatalf("première exécution: %v", err)
	}
	r2, err := SeedCareerRanks(opts)
	if err != nil {
		t.Fatalf("deuxième exécution: %v", err)
	}
	// La deuxième fois : 0 insérés, 1 skipped (INSERT OR IGNORE)
	if r1.Inserted != 1 {
		t.Errorf("r1.Inserted = %d, want 1", r1.Inserted)
	}
	if r2.Inserted != 0 || r2.Skipped != 1 {
		t.Errorf("r2: inserted=%d skipped=%d, want inserted=0 skipped=1", r2.Inserted, r2.Skipped)
	}
}

func TestSeedCareerRanks_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	// JSON invalide
	if err := os.WriteFile(filepath.Join(cacheDir, "career_ranks_metadata.json"), []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := SeedCareerRanks(SeedOptions{MetaDBPath: dbPath, DataDir: dir})
	if err == nil {
		t.Error("expected error (JSON invalide)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SeedCitationMappings
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedCitationMappings_Valid(t *testing.T) {
	dbPath := openTempDB(t)

	result, err := SeedCitationMappings(SeedOptions{MetaDBPath: dbPath})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result.Inserted == 0 {
		t.Error("expected at least 1 inserted")
	}
	if result.Component != "citation_mappings" {
		t.Errorf("component = %q, want citation_mappings", result.Component)
	}
}

func TestSeedCitationMappings_Idempotent(t *testing.T) {
	dbPath := openTempDB(t)

	opts := SeedOptions{MetaDBPath: dbPath}
	r1, err := SeedCitationMappings(opts)
	if err != nil {
		t.Fatalf("première exécution: %v", err)
	}
	r2, err := SeedCitationMappings(opts)
	if err != nil {
		t.Fatalf("deuxième exécution: %v", err)
	}
	if r1.Inserted == 0 {
		t.Error("expected insertions on first run")
	}
	if r2.Inserted != 0 {
		t.Errorf("r2.Inserted = %d, want 0 (idempotent)", r2.Inserted)
	}
	if r2.Skipped != r1.Inserted {
		t.Errorf("r2.Skipped = %d, want %d", r2.Skipped, r1.Inserted)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SeedMedalDefinitions
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedMedalDefinitions_TableAbsent(t *testing.T) {
	dbPath := openTempDB(t)

	result, err := SeedMedalDefinitions(SeedOptions{MetaDBPath: dbPath})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result.Component != "medal_definitions" {
		t.Errorf("component = %q, want medal_definitions", result.Component)
	}
	// Table absente → message spécifique
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestSeedMedalDefinitions_TablePresent(t *testing.T) {
	dbPath := openTempDB(t)

	// Créer la table medal_definitions avec quelques lignes
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE medal_definitions (id BIGINT PRIMARY KEY, name VARCHAR)`); err != nil {
		db.Close()
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO medal_definitions VALUES (1, 'Double Kill'), (2, 'Triple Kill')`); err != nil {
		db.Close()
		t.Fatalf("INSERT: %v", err)
	}
	db.Close()

	result, err := SeedMedalDefinitions(SeedOptions{MetaDBPath: dbPath})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", result.Skipped)
	}
}
