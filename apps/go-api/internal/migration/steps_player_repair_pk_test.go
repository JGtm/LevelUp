//go:build integration

// Package migration — steps_player_repair_pk_test.go : garde-fou anti-régression
// pour les migrations correctives PK player legacy (player_match_enrichment +
// match_citations). Reproduit l'état buggé (table SANS PK), vérifie que la
// migration pose la PK, préserve la donnée (colonnes dynamiques incluses),
// dédup défensivement et débloque les writes PK-dépendants.
//
// Tag `integration` car le driver DuckDB (CGO) est requis.

package migration

import (
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func migByName(t *testing.T, name string) *Migration {
	t.Helper()
	for i := range All() {
		if All()[i].Name == name {
			return &All()[i]
		}
	}
	t.Fatalf("migration %s introuvable", name)
	return nil
}

// --- player_match_enrichment ------------------------------------------------

// TestRepairPMEPrimaryKey_OnLegacyDB : table sans PK + colonne additive → la
// migration pose PK(match_id), préserve toutes les colonnes/données, débloque
// INSERT OR IGNORE.
func TestRepairPMEPrimaryKey_OnLegacyDB(t *testing.T) {
	db := openMemDB(t)
	// Legacy : pas de PK + une colonne additive (simule un ADD COLUMN historique).
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR, performance_score DOUBLE, session_id VARCHAR,
		engagement_score DOUBLE)`); err != nil {
		t.Fatalf("seed legacy PME: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment
		(match_id, performance_score, engagement_score)
		VALUES ('m1', 42.0, 7.5), ('m2', 13.0, 1.0)`); err != nil {
		t.Fatalf("insert seed: %v", err)
	}

	if hasPK, _ := hasPrimaryKey(db, "player_match_enrichment"); hasPK {
		t.Fatal("seed invalide : PME ne devrait pas avoir de PK")
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO player_match_enrichment (match_id) VALUES ('m1')`); err == nil {
		t.Fatal("INSERT OR IGNORE devrait échouer AVANT la migration (PK manquante)")
	}

	if err := migByName(t, "repair_player_match_enrichment_primary_key").ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair PME: %v", err)
	}

	if hasPK, err := hasPrimaryKey(db, "player_match_enrichment"); err != nil || !hasPK {
		t.Fatalf("PK absente après migration (hasPK=%v err=%v)", hasPK, err)
	}
	// Données + colonne additive préservées.
	var score, eng float64
	if err := db.QueryRow(`SELECT performance_score, engagement_score
		FROM player_match_enrichment WHERE match_id='m1'`).Scan(&score, &eng); err != nil {
		t.Fatalf("donnée perdue après migration: %v", err)
	}
	if score != 42.0 || eng != 7.5 {
		t.Errorf("donnée altérée: score=%v eng=%v (want 42 / 7.5)", score, eng)
	}
	// INSERT OR IGNORE fonctionne désormais (no-op sur PK existante).
	if _, err := db.Exec(`INSERT OR IGNORE INTO player_match_enrichment (match_id) VALUES ('m1')`); err != nil {
		t.Fatalf("INSERT OR IGNORE doit fonctionner APRÈS la migration: %v", err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n)
	if n != 2 {
		t.Errorf("INSERT OR IGNORE a dupliqué m1 (count=%d, want 2)", n)
	}
}

// --- match_citations --------------------------------------------------------

// TestRepairCitationsPK_OnLegacyDB : table sans PK avec doublon + clé NULL → la
// migration pose la PK composite, dédup (garde la plus récente), écarte les
// lignes à clé NULL, débloque ON CONFLICT.
func TestRepairCitationsPK_OnLegacyDB(t *testing.T) {
	db := openMemDB(t)
	if _, err := db.Exec(`CREATE TABLE match_citations (
		match_id VARCHAR, citation_name_norm VARCHAR, value INTEGER DEFAULT 1,
		created_at TIMESTAMP)`); err != nil {
		t.Fatalf("seed legacy citations: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_citations VALUES
		('m1','helping_hand',1, TIMESTAMP '2026-01-01 00:00:00'),
		('m1','helping_hand',5, TIMESTAMP '2026-06-01 00:00:00'),
		('m2','eagle_eye',3, TIMESTAMP '2026-01-01 00:00:00'),
		(NULL,'orphan',1, TIMESTAMP '2026-01-01 00:00:00'),
		('m3',NULL,1, TIMESTAMP '2026-01-01 00:00:00')`); err != nil {
		t.Fatalf("insert seed citations: %v", err)
	}

	if err := migByName(t, "repair_match_citations_primary_key").ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair citations: %v", err)
	}

	if hasPK, err := hasPrimaryKey(db, "match_citations"); err != nil || !hasPK {
		t.Fatalf("PK absente après migration (hasPK=%v err=%v)", hasPK, err)
	}
	// Dédup : m1/helping_hand → 1 ligne, valeur la plus récente (5).
	var cnt, val int
	_ = db.QueryRow(`SELECT COUNT(*), MAX(value) FROM match_citations
		WHERE match_id='m1' AND citation_name_norm='helping_hand'`).Scan(&cnt, &val)
	if cnt != 1 || val != 5 {
		t.Errorf("dédup KO : count=%d value=%d (want 1 / 5)", cnt, val)
	}
	// Lignes à clé NULL écartées → total = 2 clés distinctes (m1/hh, m2/ee).
	var total int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_citations`).Scan(&total)
	if total != 2 {
		t.Errorf("lignes à clé NULL non écartées (total=%d, want 2)", total)
	}
	// ON CONFLICT fonctionne désormais.
	if _, err := db.Exec(`INSERT INTO match_citations (match_id, citation_name_norm, value)
		VALUES ('m1','helping_hand',9) ON CONFLICT (match_id, citation_name_norm) DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT doit fonctionner APRÈS la migration: %v", err)
	}
}

// TestRepairCitationsPK_Idempotent : table déjà PK → no-op, donnée préservée.
func TestRepairCitationsPK_Idempotent(t *testing.T) {
	db := openMemDB(t)
	if _, err := db.Exec(`CREATE TABLE match_citations (
		match_id VARCHAR NOT NULL, citation_name_norm VARCHAR NOT NULL,
		value INTEGER NOT NULL DEFAULT 1, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (match_id, citation_name_norm))`); err != nil {
		t.Fatalf("seed sain: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_citations (match_id, citation_name_norm, value)
		VALUES ('m9','perfect',7)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	mig := migByName(t, "repair_match_citations_primary_key")
	for pass := 1; pass <= 2; pass++ {
		if err := mig.ApplySchema(db); err != nil {
			t.Fatalf("ApplySchema passe %d (doit être no-op): %v", pass, err)
		}
	}
	var val int
	if err := db.QueryRow(`SELECT value FROM match_citations
		WHERE match_id='m9' AND citation_name_norm='perfect'`).Scan(&val); err != nil || val != 7 {
		t.Fatalf("donnée altérée par un no-op (val=%d err=%v)", val, err)
	}
}
