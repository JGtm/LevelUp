//go:build integration

// declared_keys_repair_test.go — la réparation des clés perdues (internal/migration,
// declared_keys_repair.go) sur les schémas RÉELS du titre : une base joueur et une metadata
// dont des tables ont été recopiées par CREATE TABLE AS (signature d'une restauration Parquet
// ancienne) retrouvent leurs clés au cycle de migrations suivant.
package migrations

import (
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/migration"
)

// dropKeysLikeCTAS remplace chaque table par sa copie CREATE TABLE AS (lignes gardées,
// contraintes et défauts perdus).
func dropKeysLikeCTAS(t *testing.T, db *sql.DB, tables ...string) {
	t.Helper()
	for _, tb := range tables {
		script := fmt.Sprintf(`CREATE TABLE %[1]s__ctas AS SELECT * FROM %[1]s;
			DROP TABLE %[1]s;
			ALTER TABLE %[1]s__ctas RENAME TO %[1]s;`, tb)
		if err := migration.ExecScript(db, script); err != nil {
			t.Fatalf("copie CTAS de %s : %v", tb, err)
		}
	}
}

func primaryKeyCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_constraints()
		WHERE table_name = ? AND constraint_type = 'PRIMARY KEY'`, table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestRepairDeclaredKeys_BaseJoueur : les tables d'une base joueur restaurée (registre des
// migrations compris) retrouvent leur clé ; la table append-only `streak` n'est pas touchée.
func TestRepairDeclaredKeys_BaseJoueur(t *testing.T) {
	db := freshProvision(t, migration.TargetPlayer)
	if _, err := db.Exec(`INSERT INTO arc (id, user_id, title_slug, title)
		VALUES ('a1', 'u', 'halo_infinite', 't')`); err != nil {
		t.Fatalf("seed arc: %v", err)
	}
	repaired := []string{"schema_migrations", "arc", "challenge", "sessions", "milestone_earned", "baseline_state"}
	dropKeysLikeCTAS(t, db, append(repaired, "streak")...)

	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	for _, tb := range repaired {
		if primaryKeyCount(t, db, tb) != 1 {
			t.Errorf("%s : clé primaire non rétablie", tb)
		}
	}
	if primaryKeyCount(t, db, "streak") != 0 {
		t.Error("streak (append-only, vue streak_latest) ne doit pas être réparée au démarrage")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM arc WHERE id = 'a1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("ligne de arc perdue (n=%d, err=%v)", n, err)
	}
	// L'écriture de l'outil qui échouait (ON CONFLICT sur arc) passe.
	if _, err := db.Exec(`INSERT INTO arc (id, user_id, title_slug, title)
		VALUES ('a1', 'u', 'halo_infinite', 't') ON CONFLICT (id) DO NOTHING`); err != nil {
		t.Fatalf("ON CONFLICT sur arc réparée : %v", err)
	}
}

// TestRepairDeclaredKeys_MetadataChallengeTemplate : C2 — `challenge_template` sans clé sur une
// metadata retrouve sa clé, et le seed Prestige (`ON CONFLICT (id) DO UPDATE`) passe ensuite.
func TestRepairDeclaredKeys_MetadataChallengeTemplate(t *testing.T) {
	db := freshProvision(t, migration.TargetMetadata)
	dropKeysLikeCTAS(t, db, "challenge_template")
	if primaryKeyCount(t, db, "challenge_template") != 0 {
		t.Fatal("préparation : la copie devrait être sans clé")
	}
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if primaryKeyCount(t, db, "challenge_template") != 1 {
		t.Fatal("challenge_template : clé primaire non rétablie")
	}
	tomlDir := t.TempDir()
	writePrestigeTOML(t, tomlDir, templatesTOML, presetsTOML)
	if err := seedTemplates(db, tomlDir+"/challenges/templates.toml"); err != nil {
		t.Fatalf("seed Prestige sur la table réparée : %v", err)
	}
	if err := seedTemplates(db, tomlDir+"/challenges/templates.toml"); err != nil {
		t.Fatalf("seed Prestige rejoué (ON CONFLICT DO UPDATE) : %v", err)
	}
}
