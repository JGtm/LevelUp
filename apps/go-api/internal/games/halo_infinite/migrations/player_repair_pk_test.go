//go:build integration

// player_repair_pk_test.go — déplacé depuis internal/migration (Phase 1.5 b21).
// Garde-fou des migrations correctives PK player legacy (player_match_enrichment +
// match_citations). Résout les steps via StepsFor(TargetPlayer) et appelle ApplySchema
// direct (pas de RunForDB → provider non requis).
package migrations

import (
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func migByName(t *testing.T, name string) *migration.Migration {
	t.Helper()
	steps := StepsFor(migration.TargetPlayer)
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	t.Fatalf("migration %s introuvable dans StepsFor(TargetPlayer)", name)
	return nil
}

// TestRepairPME_ConvertsLegacyToAppendOnly : append-only #23046 — sur une table
// legacy SANS colonne id, la migration repair convertit en append-only (id PK + stage
// + written_at + vue _latest) et PRÉSERVE les données. Plus de PK(match_id) : la
// migration ne pose JAMAIS de PK match_id (qui rouvrirait le vecteur ART).
func TestRepairPME_ConvertsLegacyToAppendOnly(t *testing.T) {
	db := openEngMemDB(t)
	// Legacy : pas de schéma append-only (id absent) + une colonne additive.
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
	if hasID, _ := migration.ColumnExists(db, "player_match_enrichment", "id"); hasID {
		t.Fatal("seed invalide : PME legacy ne devrait pas avoir de colonne id")
	}

	if err := migByName(t, "repair_player_match_enrichment_primary_key").ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair PME: %v", err)
	}

	// Append-only : id présent + vue _latest queryable.
	if hasID, _ := migration.ColumnExists(db, "player_match_enrichment", "id"); !hasID {
		t.Fatal("colonne id absente après migration (append-only attendu)")
	}
	// Données préservées (socle stage='legacy', lues via la vue merge).
	var score, eng float64
	if err := db.QueryRow(`SELECT performance_score, engagement_score
		FROM player_match_enrichment_latest WHERE match_id='m1'`).Scan(&score, &eng); err != nil {
		t.Fatalf("donnée perdue après migration: %v", err)
	}
	if score != 42.0 || eng != 7.5 {
		t.Errorf("donnée altérée: score=%v eng=%v (want 42 / 7.5)", score, eng)
	}
	// PLUS de PK(match_id) : un INSERT du même match_id (stage distinct) doit réussir.
	if _, err := db.Exec(
		`INSERT INTO player_match_enrichment (match_id, dominance_flag, stage) VALUES ('m1', 1, 'dominance')`); err != nil {
		t.Errorf("append-only : INSERT match_id dupliqué (stage distinct) devrait réussir: %v", err)
	}
}

func TestRepairCitationsPK_OnLegacyDB(t *testing.T) {
	db := openEngMemDB(t)
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

	if hasPK, err := migration.HasPrimaryKey(db, "match_citations"); err != nil || !hasPK {
		t.Fatalf("PK absente après migration (hasPK=%v err=%v)", hasPK, err)
	}
	var cnt, val int
	_ = db.QueryRow(`SELECT COUNT(*), MAX(value) FROM match_citations
		WHERE match_id='m1' AND citation_name_norm='helping_hand'`).Scan(&cnt, &val)
	if cnt != 1 || val != 5 {
		t.Errorf("dédup KO : count=%d value=%d (want 1 / 5)", cnt, val)
	}
	var total int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_citations`).Scan(&total)
	if total != 2 {
		t.Errorf("lignes à clé NULL non écartées (total=%d, want 2)", total)
	}
	if _, err := db.Exec(`INSERT INTO match_citations (match_id, citation_name_norm, value)
		VALUES ('m1','helping_hand',9) ON CONFLICT (match_id, citation_name_norm) DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT doit fonctionner APRÈS la migration: %v", err)
	}
}

func TestRepairCitationsPK_Idempotent(t *testing.T) {
	db := openEngMemDB(t)
	if _, err := db.Exec(`CREATE TABLE match_citations (
		match_id VARCHAR NOT NULL, citation_name_norm VARCHAR NOT NULL,
		value INTEGER NOT NULL DEFAULT 1, created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
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
