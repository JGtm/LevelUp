//go:build cgo

// Package duckdb — objective_family_sql_test.go : le SELECT des lectures d'objectif est généré depuis
// les tables narrative ; ce garde-rail vérifie que chaque colonne générée existe. Harnais partagé avec
// objective_stats_repo_test.go (vraies migrations shared sur :memory:).
package duckdb

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

// TestObjectiveFamilySQL_WeightKeysSubsetOfSchema : les clés des tables de poids
// et de durées narrative sont TOUTES des colonnes réelles de
// match_objective_stats — le SELECT étant généré depuis ces tables, une clé
// fantôme casserait la requête en dégradation silencieuse (axe muet sans erreur).
func TestObjectiveFamilySQL_WeightKeysSubsetOfSchema(t *testing.T) {
	_, db := newObjectiveStatsRepoOnMem(t, true)

	rows, err := db.Query(`
		SELECT column_name FROM information_schema.columns
		WHERE table_name = 'match_objective_stats'`)
	if err != nil {
		t.Fatalf("information_schema: %v", err)
	}
	defer rows.Close()
	schema := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan: %v", err)
		}
		schema[c] = true
	}

	for fam, weights := range narrative.ObjectiveFamilyActionWeights {
		for col := range weights {
			if !schema[col] {
				t.Errorf("famille %s : colonne d'action %q absente du schéma match_objective_stats", fam, col)
			}
		}
	}
	for fam, cols := range narrative.ObjectiveFamilyHoldColumns {
		for _, col := range cols {
			if !schema[col] {
				t.Errorf("famille %s : colonne de durée %q absente du schéma match_objective_stats", fam, col)
			}
		}
	}
	if !schema["times_selected_as_vip"] {
		t.Error("colonne times_selected_as_vip absente du schéma (dénominateur VIP)")
	}
}
