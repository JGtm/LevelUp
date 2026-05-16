//go:build integration

// Package migration — steps_player_perf_chain_test.go : vérifie que la
// migration `player_match_enrichment_performance_chain_v1` ajoute bien la
// colonne `performance_chain` au schéma player, sur DB neuve comme sur DB
// pré-existante (legacy).
//
// Tag `integration` car ces tests nécessitent le driver DuckDB (CGO).

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// mustHaveColumn fait échouer le test si la colonne est absente.
func mustHaveColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	exists, err := columnExists(db, table, column)
	if err != nil {
		t.Fatalf("columnExists(%q.%q): %v", table, column, err)
	}
	if !exists {
		t.Errorf("colonne %q absente sur la table %q", column, table)
	}
}

// TestRunForDB_Player_PerformanceChainColumn vérifie que performance_chain est
// présent sur player_match_enrichment après RunForDB(TargetPlayer).
func TestRunForDB_Player_PerformanceChainColumn(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	mustHaveColumn(t, db, "player_match_enrichment", "performance_chain")
}

// TestRunForDB_Player_PerformanceChain_OnLegacyDB simule une DB legacy ne
// disposant pas encore de la colonne performance_chain : on crée la table
// avec uniquement les colonnes d'origine, on exécute la migration ciblée, et
// on vérifie que la colonne est ajoutée.
//
// Reflète le scénario d'upgrade : DB de production existante → upgrade.
func TestRunForDB_Player_PerformanceChain_OnLegacyDB(t *testing.T) {
	db := openMemDB(t)

	// Reproduire l'état legacy : table sans performance_chain.
	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id          VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id        VARCHAR
		);
	`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	if exists, err := columnExists(db, "player_match_enrichment", "performance_chain"); err != nil {
		t.Fatalf("columnExists pré-migration: %v", err)
	} else if exists {
		t.Fatal("seed legacy : performance_chain ne devrait pas exister avant migration")
	}

	// Récupérer la migration ciblée et l'appliquer directement.
	var mig *Migration
	for i := range All() {
		m := &All()[i]
		if m.Name == "player_match_enrichment_performance_chain_v1" {
			mig = m
			break
		}
	}
	if mig == nil {
		t.Fatal("migration player_match_enrichment_performance_chain_v1 introuvable dans le registre")
	}

	if err := mig.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema performance_chain: %v", err)
	}
	mustHaveColumn(t, db, "player_match_enrichment", "performance_chain")

	// 2e passe : ADD COLUMN IF NOT EXISTS doit rester silencieux.
	if err := mig.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema (2e passe, colonne déjà existante) doit être idempotent: %v", err)
	}
}
