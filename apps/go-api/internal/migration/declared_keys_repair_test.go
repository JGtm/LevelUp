//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// keyRepairSteps : jeu d'étapes de test. kr_t porte une PRIMARY KEY et une UNIQUE, kr_nokey n'en
// porte aucune par conception, kr_ao est append-only (clé technique + vue _latest).
var keyRepairSteps = []Migration{{
	Name:     "kr_schema_v1",
	TargetDB: TargetDB("kr_target"),
	ApplySchema: func(db *sql.DB) error {
		return ExecScript(db, `
			CREATE TABLE IF NOT EXISTS kr_t (id VARCHAR PRIMARY KEY, code VARCHAR UNIQUE,
				v INTEGER DEFAULT 5, updated_at TIMESTAMP);
			CREATE INDEX IF NOT EXISTS idx_kr_t_v ON kr_t(v);
			CREATE TABLE IF NOT EXISTS kr_nokey (v INTEGER);
			CREATE TABLE IF NOT EXISTS kr_ao (id BIGINT PRIMARY KEY, v INTEGER);
			CREATE OR REPLACE VIEW kr_ao_latest AS SELECT * FROM kr_ao;
			CREATE OR REPLACE VIEW kr_t_view AS SELECT id, v FROM kr_t;`)
	},
}}

// withTitleProvider pose un provider title-owned (vide) le temps du test : sans lui, la référence
// n'est pas réputée complète et la passe ne répare rien.
func withTitleProvider(t *testing.T) {
	t.Helper()
	prev := titleStepsProvider
	SetTitleStepsProvider(func(TargetDB) []Migration { return nil })
	t.Cleanup(func() { SetTitleStepsProvider(prev) })
}

// seedKeylessCopies pose les tables du jeu de test SANS leurs contraintes (signature d'une copie
// CREATE TABLE AS), avec doublons, puis inscrit l'étape comme déjà appliquée.
func seedKeylessCopies(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := ExecScript(db, `
		CREATE TABLE kr_t (id VARCHAR, code VARCHAR, v INTEGER, updated_at TIMESTAMP);
		INSERT INTO kr_t VALUES
			('a', 'A', 1, '2026-01-01'), ('a', 'A', 2, '2026-03-01'),
			('b', 'B', 7, NULL);
		CREATE INDEX idx_kr_t_v ON kr_t(v);
		CREATE VIEW kr_t_view AS SELECT id, v FROM kr_t;
		CREATE TABLE kr_nokey (v INTEGER);
		INSERT INTO kr_nokey VALUES (1), (1);
		CREATE TABLE kr_ao (id BIGINT, v INTEGER);
		INSERT INTO kr_ao VALUES (1, 1), (1, 1);
		CREATE VIEW kr_ao_latest AS SELECT * FROM kr_ao;`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := ensureMigrationTable(bootCtx(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (name, schema_done, backfill_done)
		VALUES ('kr_schema_v1', TRUE, TRUE)`); err != nil {
		t.Fatal(err)
	}
}

func constraintCount(t *testing.T, db *sql.DB, table, kind string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_constraints()
		WHERE table_name = ? AND constraint_type = ?`, table, kind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestRepairDeclaredKeys_RetablitLesClesDuSchemaCanonique : la table qui a perdu ses clés les
// retrouve (doublon départagé par l'horodatage, défaut perdu repris,
// index et vue intacts) ; la table sans clé par conception et la table append-only ne sont pas
// touchées ; une écriture ON CONFLICT réussit ensuite.
func TestRepairDeclaredKeys_RetablitLesClesDuSchemaCanonique(t *testing.T) {
	withTitleProvider(t)
	db := openMemDB(t)
	seedKeylessCopies(t, db)

	if err := RunSteps(db, TargetDB("kr_target"), keyRepairSteps); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}

	if constraintCount(t, db, "kr_t", "PRIMARY KEY") != 1 || constraintCount(t, db, "kr_t", "UNIQUE") != 1 {
		t.Fatal("kr_t : clés non rétablies")
	}
	var n, v int
	if err := db.QueryRow(`SELECT count(*) FROM kr_t`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("kr_t : %d lignes (err=%v), want 2 (doublon écarté)", n, err)
	}
	if err := db.QueryRow(`SELECT v FROM kr_t WHERE id = 'a'`).Scan(&v); err != nil || v != 2 {
		t.Fatalf("kr_t : la ligne la plus récente de 'a' doit être gardée (v=%d, err=%v)", v, err)
	}
	if _, err := db.Exec(`INSERT INTO kr_t (id, code) VALUES ('c', 'C')
		ON CONFLICT (id) DO UPDATE SET code = excluded.code`); err != nil {
		t.Fatalf("ON CONFLICT après réparation : %v", err)
	}
	if err := db.QueryRow(`SELECT v FROM kr_t WHERE id = 'c'`).Scan(&v); err != nil || v != 5 {
		t.Fatalf("défaut canonique non repris (v=%d, err=%v)", v, err)
	}
	var idx int
	if err := db.QueryRow(`SELECT count(*) FROM duckdb_indexes() WHERE index_name = 'idx_kr_t_v'`).Scan(&idx); err != nil || idx != 1 {
		t.Fatalf("index secondaire non recréé (n=%d, err=%v)", idx, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM kr_t_view`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("vue dépendante cassée (n=%d, err=%v)", n, err)
	}
	if constraintCount(t, db, "kr_nokey", "PRIMARY KEY") != 0 {
		t.Fatal("kr_nokey : table sans clé par conception modifiée")
	}
	if constraintCount(t, db, "kr_ao", "PRIMARY KEY") != 0 {
		t.Fatal("kr_ao : une table append-only ne doit pas être réparée au démarrage")
	}
	if err := db.QueryRow(`SELECT count(*) FROM kr_ao`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("kr_ao : lignes modifiées (n=%d, err=%v)", n, err)
	}

	// Idempotence : un second cycle ne touche plus rien.
	if err := RunSteps(db, TargetDB("kr_target"), keyRepairSteps); err != nil {
		t.Fatalf("RunSteps (2e cycle): %v", err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM kr_t`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("2e cycle : %d lignes (err=%v), want 3", n, err)
	}
}

// TestRepairDeclaredKeys_SansJeuCompletNeTouchePas : sans les étapes title-owned câblées, la
// référence n'est pas complète : aucune table n'est reconstruite.
func TestRepairDeclaredKeys_SansJeuCompletNeTouchePas(t *testing.T) {
	prev := titleStepsProvider
	SetTitleStepsProvider(nil)
	t.Cleanup(func() { SetTitleStepsProvider(prev) })
	db := openMemDB(t)
	seedKeylessCopies(t, db)
	if err := RunSteps(db, TargetDB("kr_target"), keyRepairSteps); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if constraintCount(t, db, "kr_t", "PRIMARY KEY") != 0 {
		t.Fatal("kr_t réparée alors que le jeu d'étapes n'est pas complet")
	}
}

// TestRepairDeclaredKeys_SequencePerdueRefusee : une colonne de clé alimentée par une séquence
// que la table réelle a perdue n'est pas réparée (la clé refuserait chaque nouvelle ligne).
func TestRepairDeclaredKeys_SequencePerdueRefusee(t *testing.T) {
	withTitleProvider(t)
	steps := []Migration{{
		Name:     "kr_seq_v1",
		TargetDB: TargetDB("kr_seq_target"),
		ApplySchema: func(db *sql.DB) error {
			return ExecScript(db, `CREATE SEQUENCE IF NOT EXISTS kr_seq;
				CREATE TABLE IF NOT EXISTS kr_s (id BIGINT PRIMARY KEY DEFAULT nextval('kr_seq'), v INTEGER);`)
		},
	}}
	db := openMemDB(t)
	if err := ExecScript(db, `CREATE TABLE kr_s (id BIGINT, v INTEGER); INSERT INTO kr_s VALUES (1, 1);`); err != nil {
		t.Fatal(err)
	}
	if err := ensureMigrationTable(bootCtx(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (name, schema_done, backfill_done) VALUES ('kr_seq_v1', TRUE, TRUE)`); err != nil {
		t.Fatal(err)
	}
	if err := RunSteps(db, TargetDB("kr_seq_target"), steps); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if constraintCount(t, db, "kr_s", "PRIMARY KEY") != 0 {
		t.Fatal("kr_s réparée alors que sa clé dépend d'une séquence perdue")
	}
}

// TestRepairDeclaredKeys_CleePrimaireNullRefusee : une ligne dont une colonne de la future clé
// primaire est NULL ne peut pas entrer dans la table reconstruite ; la passe ne supprime jamais
// une ligne au démarrage : elle refuse la table, qui reste inchangée (lignes et absence de clé).
func TestRepairDeclaredKeys_CleePrimaireNullRefusee(t *testing.T) {
	withTitleProvider(t)
	db := openMemDB(t)
	seedKeylessCopies(t, db)
	if _, err := db.Exec(`INSERT INTO kr_t VALUES (NULL, 'Y', 8, '2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if err := RunSteps(db, TargetDB("kr_target"), keyRepairSteps); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if constraintCount(t, db, "kr_t", "PRIMARY KEY") != 0 {
		t.Fatal("kr_t réparée malgré des lignes à clé primaire NULL")
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM kr_t`).Scan(&n); err != nil || n != 4 {
		t.Fatalf("kr_t : %d lignes (err=%v), want 4 (aucune ligne supprimée)", n, err)
	}
}
