//go:build integration

// player_engagement_pkfix_test.go — déplacé depuis internal/migration (Phase 1.5 b17).
// Garde-fou anti-régression de repair_engagement_coefficients_primary_key (title-owned).
// Résout le step via StepsFor(TargetPlayer) (au lieu du registre global All()) et appelle
// ApplySchema directement (pas de RunForDB → pas de provider requis).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func openEngMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func engCoefsRepairMig(t *testing.T) *migration.Migration {
	t.Helper()
	steps := StepsFor(migration.TargetPlayer)
	for i := range steps {
		if steps[i].Name == "repair_engagement_coefficients_primary_key" {
			return &steps[i]
		}
	}
	t.Fatal("migration repair_engagement_coefficients_primary_key introuvable dans StepsFor(TargetPlayer)")
	return nil
}

func seedLegacyEngCoefs(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE engagement_coefficients (
			xuid             VARCHAR NOT NULL,
			mode_category    VARCHAR NOT NULL,
			coef_team_share  DOUBLE NOT NULL,
			coef_lobby_share DOUBLE NOT NULL,
			n_matches        INTEGER NOT NULL,
			last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
}

func upsertCoef(db *sql.DB, xuid, mode string, n int) error {
	_, err := db.Exec(`
		INSERT INTO engagement_coefficients
			(xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated)
		VALUES (?, ?, 0.5, 0.3, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (xuid, mode_category) DO UPDATE SET n_matches = EXCLUDED.n_matches`,
		xuid, mode, n)
	return err
}

// TestRepairEngCoefsPK_OnLegacyDB : table sans PK → la migration ajoute la PK,
// préserve la donnée, débloque le UPSERT ON CONFLICT.
func TestRepairEngCoefsPK_OnLegacyDB(t *testing.T) {
	db := openEngMemDB(t)
	seedLegacyEngCoefs(t, db)
	if _, err := db.Exec(`INSERT INTO engagement_coefficients VALUES
		('xuidA', 'PvP_unranked', 0.4, 0.2, 10, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert seed row: %v", err)
	}

	if hasPK, _ := migration.HasPrimaryKey(db, "engagement_coefficients"); hasPK {
		t.Fatal("seed invalide : la table ne devrait pas avoir de PK")
	}
	if err := upsertCoef(db, "xuidA", "PvP_unranked", 11); err == nil {
		t.Fatal("le UPSERT devrait échouer AVANT la migration (PK manquante)")
	}

	if err := engCoefsRepairMig(t).ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair: %v", err)
	}

	if hasPK, err := migration.HasPrimaryKey(db, "engagement_coefficients"); err != nil || !hasPK {
		t.Fatalf("PK absente après migration (hasPK=%v err=%v)", hasPK, err)
	}
	var n int
	if err := db.QueryRow(`SELECT n_matches FROM engagement_coefficients
		WHERE xuid='xuidA' AND mode_category='PvP_unranked'`).Scan(&n); err != nil {
		t.Fatalf("donnée perdue après migration: %v", err)
	}
	if n != 10 {
		t.Errorf("n_matches=%d, want 10 (donnée d'origine préservée)", n)
	}
	if err := upsertCoef(db, "xuidA", "PvP_unranked", 12); err != nil {
		t.Fatalf("le UPSERT doit fonctionner APRÈS la migration: %v", err)
	}
	_ = db.QueryRow(`SELECT n_matches FROM engagement_coefficients
		WHERE xuid='xuidA' AND mode_category='PvP_unranked'`).Scan(&n)
	if n != 12 {
		t.Errorf("UPSERT non appliqué : n_matches=%d, want 12", n)
	}
}

// TestRepairEngCoefsPK_Idempotent : table avec PK (DB saine) → no-op silencieux.
func TestRepairEngCoefsPK_Idempotent(t *testing.T) {
	db := openEngMemDB(t)
	if _, err := db.Exec(`
		CREATE TABLE engagement_coefficients (
			xuid VARCHAR NOT NULL, mode_category VARCHAR NOT NULL,
			coef_team_share DOUBLE NOT NULL, coef_lobby_share DOUBLE NOT NULL,
			n_matches INTEGER NOT NULL, last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, mode_category))`); err != nil {
		t.Fatalf("seed sain: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO engagement_coefficients VALUES
		('xuidB', 'FFA', 0.1, 0.1, 5, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	mig := engCoefsRepairMig(t)
	for pass := 1; pass <= 2; pass++ {
		if err := mig.ApplySchema(db); err != nil {
			t.Fatalf("ApplySchema passe %d (doit être no-op): %v", pass, err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT n_matches FROM engagement_coefficients
		WHERE xuid='xuidB' AND mode_category='FFA'`).Scan(&n); err != nil || n != 5 {
		t.Fatalf("donnée altérée par un no-op (n=%d err=%v)", n, err)
	}
}

// TestRepairEngCoefsPK_DedupsBeforePK : doublons (xuid, mode_category) → garde la
// ligne la plus récente.
func TestRepairEngCoefsPK_DedupsBeforePK(t *testing.T) {
	db := openEngMemDB(t)
	seedLegacyEngCoefs(t, db)
	if _, err := db.Exec(`INSERT INTO engagement_coefficients VALUES
		('xuidC', 'PvP_ranked', 0.4, 0.2, 1, TIMESTAMP '2026-01-01 00:00:00'),
		('xuidC', 'PvP_ranked', 0.9, 0.8, 99, TIMESTAMP '2026-06-01 00:00:00')`); err != nil {
		t.Fatalf("insert doublons: %v", err)
	}

	if err := engCoefsRepairMig(t).ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair (dedup): %v", err)
	}

	var cnt, n int
	_ = db.QueryRow(`SELECT COUNT(*), MAX(n_matches) FROM engagement_coefficients
		WHERE xuid='xuidC' AND mode_category='PvP_ranked'`).Scan(&cnt, &n)
	if cnt != 1 {
		t.Errorf("doublon non dédupliqué : %d lignes, want 1", cnt)
	}
	if n != 99 {
		t.Errorf("mauvaise ligne conservée : n_matches=%d, want 99 (la plus récente)", n)
	}
}
