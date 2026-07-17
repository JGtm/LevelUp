//go:build integration

package migrations

// player_baseline_dm5_heal_test.go — A4 (revue 2026-07, M3/M4). Prouve que le chemin
// DM-5 (baseline squashée réputée satisfaite car la SENTINELLE est présente) HEAL les
// colonnes/tables additives absentes d'une DB au schéma PARTIEL — sans quoi persist LUSR
// (expected_win_prob) et challenges (colonnes render) resteraient cassés durablement,
// aucun step restant ne pouvant réparer (finding n°3/4 de la revue).

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// oldPartialPlayerSentinel simule un step historique NOMMÉ comme la sentinelle DM-5
// réelle, qui crée un schéma player PARTIEL antérieur aux ALTER additifs
// (expected_win_prob, colonnes render challenge_snapshots, engagement_response_bins).
// Sa présence dans schema_migrations déclenche le chemin DM-5 au cycle suivant.
func oldPartialPlayerSentinel() migration.Migration {
	return migration.Migration{
		Name:        "player_append_only_csr_snapshots_v1", // = sentinelle DM-5 réelle
		TargetDB:    migration.TargetPlayer,
		Description: "sentinelle DM-5 (schéma partiel pré-additif, fixture M4)",
		ApplySchema: func(db *sql.DB) error {
			return migration.ExecScript(db, `
				CREATE TABLE match_skill_rank (
					match_id VARCHAR PRIMARY KEY,
					rating_type VARCHAR NOT NULL,
					rating_value FLOAT
				);
				CREATE TABLE player_match_enrichment (
					match_id VARCHAR PRIMARY KEY,
					performance_score DOUBLE
				);
				CREATE TABLE career_progression (
					xuid VARCHAR,
					rank INTEGER
				);
				CREATE TABLE challenge_snapshots (
					snapshot_at TIMESTAMP,
					xuid VARCHAR NOT NULL,
					challenge_path VARCHAR NOT NULL,
					status VARCHAR NOT NULL,
					state_hash VARCHAR NOT NULL
				);
			`)
		},
	}
}

func TestDM5Heal_SentinelPresentButAdditiveColumnsMissing(t *testing.T) {
	db := openEngMemDB(t)

	// Cycle 1 : DB « ancienne » — sentinelle appliquée, schéma partiel.
	if err := migration.RunSteps(db, migration.TargetPlayer, []migration.Migration{oldPartialPlayerSentinel()}); err != nil {
		t.Fatalf("cycle 1 (sentinelle + schéma partiel): %v", err)
	}
	// Préconditions : les colonnes/tables additives sont ABSENTES.
	if healColExists(t, db, "match_skill_rank", "expected_win_prob") {
		t.Fatal("précondition invalide : expected_win_prob déjà présente")
	}
	if healTblExists(t, db, "engagement_response_bins") {
		t.Fatal("précondition invalide : engagement_response_bins déjà présente")
	}
	if healColExists(t, db, "challenge_snapshots", "image_url") {
		t.Fatal("précondition invalide : colonne render déjà présente")
	}

	// Cycle 2 : boot avec la baseline squashée → DM-5 (sentinelle présente, DDL non
	// rejoué) → heal additif idempotent.
	if err := migration.RunSteps(db, migration.TargetPlayer, playerBaselineSteps()); err != nil {
		t.Fatalf("cycle 2 (baseline DM-5 heal): %v", err)
	}

	// Postconditions : le trou est refermé.
	if !healColExists(t, db, "match_skill_rank", "expected_win_prob") {
		t.Error("expected_win_prob toujours absente après le heal DM-5 (persist LUSR resterait cassé)")
	}
	for _, c := range []string{"title", "description", "image_url", "display_path"} {
		if !healColExists(t, db, "challenge_snapshots", c) {
			t.Errorf("colonne render challenge_snapshots.%s toujours absente après heal", c)
		}
	}
	if !healTblExists(t, db, "engagement_response_bins") {
		t.Error("engagement_response_bins toujours absente après le heal DM-5")
	}

	// Persist LUSR OK : un INSERT ciblant expected_win_prob passe désormais.
	if _, err := db.Exec(
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, expected_win_prob) VALUES ('m1','lusr',1200.0,0.55)`,
	); err != nil {
		t.Errorf("INSERT match_skill_rank(expected_win_prob) échoue après heal : %v", err)
	}
}

func healColExists(t *testing.T, db *sql.DB, table, col string) bool {
	t.Helper()
	ok, err := migration.ColumnExists(db, table, col)
	if err != nil {
		t.Fatalf("ColumnExists(%s.%s): %v", table, col, err)
	}
	return ok
}

func healTblExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	ok, err := migration.TableExists(db, table)
	if err != nil {
		t.Fatalf("TableExists(%s): %v", table, err)
	}
	return ok
}
