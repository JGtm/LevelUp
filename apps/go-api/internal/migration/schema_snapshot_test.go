//go:build integration

package migration

// schema_snapshot_test.go — M1b (déterminisme) + M1c (sensibilité, morsure dans les
// deux sens) de l'outil SchemaSnapshot. Sonde jetable des débuts remplacée par ces
// tests permanents (le tool est du code livré, réutilisé par M2 et un futur cmd).

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// buildRefSchema pose un schéma de référence non trivial (table PK + colonnes NOT
// NULL/default, index unique, vue, séquence) — assez riche pour exercer chaque
// section du snapshot.
func buildRefSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE SEQUENCE ref_seq START 1;
		CREATE TABLE ref_t (
			id    BIGINT PRIMARY KEY DEFAULT nextval('ref_seq'),
			name  VARCHAR NOT NULL DEFAULT '',
			score DOUBLE,
			flag  BOOLEAN NOT NULL DEFAULT FALSE
		);
		CREATE UNIQUE INDEX idx_ref_name ON ref_t(name);
		CREATE VIEW ref_v AS SELECT id, name FROM ref_t WHERE flag = TRUE;
	`); err != nil {
		t.Fatalf("buildRefSchema: %v", err)
	}
}

func snap(t *testing.T, db *sql.DB) string {
	t.Helper()
	s, err := SchemaSnapshot(db)
	if err != nil {
		t.Fatalf("SchemaSnapshot: %v", err)
	}
	return s
}

// M1b — déterminisme : deux DB au schéma identique produisent le MÊME snapshot,
// octet pour octet.
func TestSchemaSnapshot_DeterministicIdenticalSchema(t *testing.T) {
	dbA := openMemDB(t)
	dbB := openMemDB(t)
	buildRefSchema(t, dbA)
	buildRefSchema(t, dbB)
	if a, b := snap(t, dbA), snap(t, dbB); a != b {
		t.Errorf("snapshots divergents pour un schéma identique:\n--- A ---\n%s\n--- B ---\n%s", a, b)
	}
}

// M1b — déterminisme sur un provisioning RÉEL (registre global, deux fois).
func TestSchemaSnapshot_DeterministicRunForDB(t *testing.T) {
	target := TargetDB("snap_det_target")
	Register(Migration{
		Name:        "snap_det_step",
		TargetDB:    target,
		Description: "step de test déterminisme snapshot",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`CREATE TABLE snap_det (a INTEGER PRIMARY KEY, b VARCHAR NOT NULL DEFAULT 'z');
				CREATE INDEX idx_snap_det_b ON snap_det(b);`)
			return err
		},
	})
	dbA := openMemDB(t)
	dbB := openMemDB(t)
	if err := RunForDB(dbA, target); err != nil {
		t.Fatalf("RunForDB A: %v", err)
	}
	if err := RunForDB(dbB, target); err != nil {
		t.Fatalf("RunForDB B: %v", err)
	}
	// Le snapshot est SCHÉMA-seul : les lignes de schema_migrations/title_schema_version
	// (données) diffèrent d'un run à l'autre par timestamp mais ne sont PAS capturées.
	if a, b := snap(t, dbA), snap(t, dbB); a != b {
		t.Errorf("snapshots RunForDB divergents (attendu identique):\n--- A ---\n%s\n--- B ---\n%s", a, b)
	}
}

// M1c — sensibilité : chaque type d'altération de schéma produit un snapshot
// DIFFÉRENT (morsure prouvée). Table-driven sur les 5 dimensions.
func TestSchemaSnapshot_SensitiveToMutations(t *testing.T) {
	cases := []struct {
		name   string
		mutate string
	}{
		{"colonne ajoutée", `ALTER TABLE ref_t ADD COLUMN extra INTEGER;`},
		{"default changé", `ALTER TABLE ref_t ALTER COLUMN name SET DEFAULT 'changed';`},
		{"table ajoutée", `CREATE TABLE ref_extra (x INTEGER);`},
		{"vue modifiée", `CREATE OR REPLACE VIEW ref_v AS SELECT id FROM ref_t;`},
		{"index ajouté", `CREATE INDEX idx_ref_score ON ref_t(score);`},
		{"séquence ajoutée", `CREATE SEQUENCE ref_seq2 START 5;`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := openMemDB(t)
			mut := openMemDB(t)
			buildRefSchema(t, base)
			buildRefSchema(t, mut)
			before := snap(t, mut)
			if _, err := mut.Exec(c.mutate); err != nil {
				t.Fatalf("mutation %q: %v", c.mutate, err)
			}
			after := snap(t, mut)
			if before == after {
				t.Errorf("mutation %q non détectée (snapshot inchangé)", c.name)
			}
			if base := snap(t, base); base == after {
				t.Errorf("mutation %q : snapshot muté == snapshot base (fausse équivalence)", c.name)
			}
		})
	}
}

// M1c bis — l'ORDRE des colonnes est observable : deux tables aux mêmes colonnes
// dans un ordre différent DOIVENT produire des snapshots différents (garde-fou
// contre une fausse équivalence par tri lexical des colonnes).
func TestSchemaSnapshot_ColumnOrderIsObservable(t *testing.T) {
	dbA := openMemDB(t)
	dbB := openMemDB(t)
	if _, err := dbA.Exec(`CREATE TABLE ord_t (a INTEGER, b INTEGER);`); err != nil {
		t.Fatal(err)
	}
	if _, err := dbB.Exec(`CREATE TABLE ord_t (b INTEGER, a INTEGER);`); err != nil {
		t.Fatal(err)
	}
	a, b := snap(t, dbA), snap(t, dbB)
	if a == b {
		t.Errorf("ordre de colonnes différent non détecté (fausse équivalence)")
	}
	if !strings.Contains(a, "ord_t") {
		t.Errorf("snapshot ne contient pas ord_t")
	}
}
