//go:build cgo

// Package ops — seed_cgo_test.go : tests de SeedCareerRanks, SeedCitationMappings,
// SeedMedalDefinitions avec une DB DuckDB de fichier temporaire.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
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

// openTempDBWithAssetTranslations crée une base dotée de asset_translations (créée par les migrations
// en production), où les seeds de citations écrivent leurs textes par langue.
func openTempDBWithAssetTranslations(t *testing.T) string {
	t.Helper()
	dbPath := openTempDB(t)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR NOT NULL,
		description VARCHAR, fetched_at TIMESTAMP, PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("asset_translations: %v", err)
	}
	return dbPath
}

func TestSeedCitationMappings_SansAssetTranslations(t *testing.T) {
	if _, err := SeedCitationMappings(context.Background(), SeedOptions{MetaDBPath: openTempDB(t)}); err == nil {
		t.Fatal("attendu : refus explicite quand asset_translations est absente")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SeedCareerRanks
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedCareerRanks_JSONAbsent(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	_, err := SeedCareerRanks(context.Background(), SeedOptions{MetaDBPath: dbPath, DataDir: dir})
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

	result, err := SeedCareerRanks(context.Background(), SeedOptions{MetaDBPath: dbPath, DataDir: dir})
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
	r1, err := SeedCareerRanks(context.Background(), opts)
	if err != nil {
		t.Fatalf("première exécution: %v", err)
	}
	r2, err := SeedCareerRanks(context.Background(), opts)
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

	_, err := SeedCareerRanks(context.Background(), SeedOptions{MetaDBPath: dbPath, DataDir: dir})
	if err == nil {
		t.Error("expected error (JSON invalide)")
	}
}

// TestSeedCareerRanks_SchemaCompatibleWithEnrichment garantit que le schéma
// produit par SeedCareerRanks honore le contrat lu par EnrichFromMetadata
// (title_en, tier_type, grade, xp_required, adornment_icon_path) et
// LoadCareerRankImageURLs (large_icon_path). Régression : un schéma amputé de
// ces colonnes plante l'enrichissement carrière (Binder Error) → jauges
// "progression rang / Héros" + "XP prochain rang" à 0. Cf. fix carrière 2026-05-29.
func TestSeedCareerRanks_SchemaCompatibleWithEnrichment(t *testing.T) {
	dir := t.TempDir()
	dbPath := openTempDB(t)

	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	// rank 1 : JSON legacy minimal (sans tier_type/large/adornment) → colonnes vides.
	// rank 2 : JSON complet → colonnes peuplées.
	ranks := []map[string]any{
		{"rank_id": 1, "title_en": "Recruit", "tier": "Bronze", "grade": 1, "xp_required": 1000, "icon_path": "i/1.png"},
		{"rank_id": 2, "title_en": "Corporal", "subtitle": "Platinum", "tier": "Platinum",
			"tier_type": "Platinum", "grade": 2, "xp_required": 15000,
			"icon_path": "i/2.png", "large_icon_path": "l/2.png", "adornment_icon_path": "a/2.png"},
	}
	data, _ := json.Marshal(ranks)
	if err := os.WriteFile(filepath.Join(cacheDir, "career_ranks_metadata.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := SeedCareerRanks(context.Background(), SeedOptions{MetaDBPath: dbPath, DataDir: dir}); err != nil {
		t.Fatalf("SeedCareerRanks: %v", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Requête EXACTE d'EnrichFromMetadata (career_live_repo.go) — doit binder.
	var (
		titleEN    string
		tierType   sql.NullString
		grade      sql.NullInt64
		xpRequired int
		adornment  sql.NullString
	)
	if err := db.QueryRow(`SELECT title_en, tier_type, grade, xp_required, adornment_icon_path
		FROM career_ranks WHERE rank_id = ?`, 2).
		Scan(&titleEN, &tierType, &grade, &xpRequired, &adornment); err != nil {
		t.Fatalf("requête EnrichFromMetadata incompatible avec le schéma seedé: %v", err)
	}
	if titleEN != "Corporal" || xpRequired != 15000 {
		t.Errorf("title_en=%q xp_required=%d, want Corporal/15000", titleEN, xpRequired)
	}
	if !tierType.Valid || tierType.String != "Platinum" {
		t.Errorf("tier_type=%v, want Platinum (peuplé depuis le JSON)", tierType)
	}
	if !adornment.Valid || adornment.String != "a/2.png" {
		t.Errorf("adornment_icon_path=%v, want a/2.png", adornment)
	}

	// xp_total = SUM(xp_required WHERE rank_id < ?) + current_xp (cf. EnrichFromMetadata).
	var completed int
	if err := db.QueryRow(`SELECT COALESCE(SUM(xp_required),0) FROM career_ranks WHERE rank_id < ?`, 2).
		Scan(&completed); err != nil {
		t.Fatalf("requête SUM xp_required: %v", err)
	}
	if completed != 1000 {
		t.Errorf("SUM(xp_required < 2) = %d, want 1000", completed)
	}

	// Requête LoadCareerRankImageURLs (large_icon_path) — doit binder + retourner l/2.png.
	var img sql.NullString
	if err := db.QueryRow(`SELECT COALESCE(NULLIF(TRIM(large_icon_path),''), NULLIF(TRIM(icon_path),''))
		FROM career_ranks WHERE rank_id = ?`, 2).Scan(&img); err != nil {
		t.Fatalf("requête LoadCareerRankImageURLs incompatible: %v", err)
	}
	if !img.Valid || img.String != "l/2.png" {
		t.Errorf("image_path=%v, want l/2.png", img)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SeedCitationMappings
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedCitationMappings_Valid(t *testing.T) {
	dbPath := openTempDBWithAssetTranslations(t)

	result, err := SeedCitationMappings(context.Background(), SeedOptions{MetaDBPath: dbPath})
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

// TestSeedCitationMappings_UneLignePartLangue : le seed écrit, pour chaque citation, une ligne de
// asset_translations par langue (première étiquette Halo), à partir de citation_texts.toml.
func TestSeedCitationMappings_UneLignePartLangue(t *testing.T) {
	dbPath := openTempDBWithAssetTranslations(t)
	if _, err := SeedCitationMappings(context.Background(), SeedOptions{MetaDBPath: dbPath}); err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	total := len(defaultCitationMappings())
	for _, l := range i18n.Locales() {
		tag := i18n.HaloTags(l)[0]
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_type = 'citation' AND lang = ? AND name <> '' AND description <> ''`, tag).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != total {
			t.Errorf("langue %s (%s) : %d citations avec nom et description, want %d", l, tag, n, total)
		}
	}
	names, _, err := citationTextsFor(citationNormCharge)
	if err != nil {
		t.Fatal(err)
	}
	var fr string
	if err := db.QueryRow(`SELECT name FROM asset_translations WHERE asset_type = 'citation' AND asset_id = ? AND lang = ?`,
		citationNormCharge, i18n.HaloTags(i18n.Default)[0]).Scan(&fr); err != nil || fr != names[i18n.Default] {
		t.Errorf("nom de %s en %s = %q (%v), want %q", citationNormCharge, i18n.Default, fr, err, names[i18n.Default])
	}
}

func TestSeedCitationMappings_Idempotent(t *testing.T) {
	dbPath := openTempDBWithAssetTranslations(t)

	opts := SeedOptions{MetaDBPath: dbPath}
	r1, err := SeedCitationMappings(context.Background(), opts)
	if err != nil {
		t.Fatalf("première exécution: %v", err)
	}
	r2, err := SeedCitationMappings(context.Background(), opts)
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

	result, err := SeedMedalDefinitions(context.Background(), SeedOptions{MetaDBPath: dbPath})
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

	result, err := SeedMedalDefinitions(context.Background(), SeedOptions{MetaDBPath: dbPath})
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if result.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", result.Skipped)
	}
}
