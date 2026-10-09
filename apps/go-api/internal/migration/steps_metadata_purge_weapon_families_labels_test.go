//go:build integration

// Tests pour la migration purge_weapon_families_labels_columns (plan libellés en dur,
// lot M5 L4).
//
// Couvre les 3 garanties demandées par le pattern (calqué sur
// steps_metadata_purge_weapons_name_fr_test.go, V721-05.1) :
//   - retire bien les colonnes sur une DB legacy qui les porte (schéma pré-lot) ;
//   - ne casse rien sur une DB qui ne les a jamais eues (schéma post-lot) ;
//   - conserve toutes les rows et la PRIMARY KEY (family_key).
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedWeaponFamiliesLegacySchema crée `weapon_families` avec le schéma PRÉ-lot
// (name_en/name_fr présentes) et 3 rows représentatives.
func seedWeaponFamiliesLegacySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE weapon_families (
			family_key VARCHAR PRIMARY KEY,
			name_en    VARCHAR NOT NULL,
			name_fr    VARCHAR NOT NULL
		);
	`); err != nil {
		t.Fatalf("seed legacy schema: %v", err)
	}
	rows := []struct{ key, en, fr string }{
		{"battle_rifle", "Battle Rifle", "Fusil de combat"},
		{"sniper_rifle", "Sniper Rifle", "Fusil de précision"},
		{"equipment", "Equipment", "Équipement"},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO weapon_families (family_key, name_en, name_fr) VALUES (?, ?, ?)`,
			r.key, r.en, r.fr); err != nil {
			t.Fatalf("insert %s: %v", r.key, err)
		}
	}
}

// seedWeaponFamiliesCurrentSchema crée `weapon_families` avec le schéma POST-lot
// (sans name_en/name_fr) — reproduit une DB neuve où add_weapon_registry a déjà tourné
// après ce lot.
func seedWeaponFamiliesCurrentSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE weapon_families (
			family_key VARCHAR PRIMARY KEY
		);
		INSERT INTO weapon_families (family_key) VALUES ('battle_rifle');
	`); err != nil {
		t.Fatalf("seed current schema: %v", err)
	}
}

func TestPurgeWeaponFamiliesLabels_RemovesColumnsWhenPresent(t *testing.T) {
	db := openMemDB(t)
	seedWeaponFamiliesLegacySchema(t, db)

	for _, col := range []string{"name_en", "name_fr"} {
		has, err := columnExists(db, "weapon_families", col)
		if err != nil || !has {
			t.Fatalf("seed attendu avec %s, got exists=%v err=%v", col, has, err)
		}
	}

	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("applyPurgeWeaponFamiliesLabels: %v", err)
	}

	for _, col := range []string{"name_en", "name_fr"} {
		has, err := columnExists(db, "weapon_families", col)
		if err != nil {
			t.Fatalf("columnExists(%s) after: %v", col, err)
		}
		if has {
			t.Fatalf("%s toujours présente après la purge", col)
		}
	}
	has, err := columnExists(db, "weapon_families", "family_key")
	if err != nil {
		t.Fatalf("columnExists(family_key): %v", err)
	}
	if !has {
		t.Error("family_key perdue par la purge (attendue préservée)")
	}
}

func TestPurgeWeaponFamiliesLabels_PreservesRowsAndPrimaryKey(t *testing.T) {
	db := openMemDB(t)
	seedWeaponFamiliesLegacySchema(t, db)

	before := countRows(t, db, "weapon_families")
	if before != 3 {
		t.Fatalf("seed attendu 3 rows, got %d", before)
	}

	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("applyPurgeWeaponFamiliesLabels: %v", err)
	}

	after := countRows(t, db, "weapon_families")
	if after != before {
		t.Fatalf("rows perdues par la purge: before=%d after=%d", before, after)
	}

	var key string
	if err := db.QueryRow(`SELECT family_key FROM weapon_families WHERE family_key='sniper_rifle'`).Scan(&key); err != nil {
		t.Fatalf("select sniper_rifle after purge: %v", err)
	}
	if key != "sniper_rifle" {
		t.Errorf("sniper_rifle altérée par la purge: key=%q", key)
	}

	// PK recréée : un doublon family_key doit être rejeté.
	if _, err := db.Exec(`INSERT INTO weapon_families (family_key) VALUES ('sniper_rifle')`); err == nil {
		t.Fatal("PK absente après purge : INSERT dupliqué a réussi")
	}
}

func TestPurgeWeaponFamiliesLabels_NoOpWhenColumnAbsent(t *testing.T) {
	db := openMemDB(t)
	seedWeaponFamiliesCurrentSchema(t, db)

	before := countRows(t, db, "weapon_families")

	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("applyPurgeWeaponFamiliesLabels (no-op attendu): %v", err)
	}

	after := countRows(t, db, "weapon_families")
	if after != before {
		t.Fatalf("no-op attendu, rows modifiées: before=%d after=%d", before, after)
	}
	has, err := columnExists(db, "weapon_families", "name_en")
	if err != nil {
		t.Fatalf("columnExists: %v", err)
	}
	if has {
		t.Fatal("name_en ne devrait jamais apparaître (schéma courant ne la crée pas)")
	}
}

func TestPurgeWeaponFamiliesLabels_NoOpWhenTableAbsent(t *testing.T) {
	db := openMemDB(t)
	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("applyPurgeWeaponFamiliesLabels (table absente): %v", err)
	}
}

func TestPurgeWeaponFamiliesLabels_IdempotentSecondRun(t *testing.T) {
	db := openMemDB(t)
	seedWeaponFamiliesLegacySchema(t, db)

	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("purge 1: %v", err)
	}
	between := countRows(t, db, "weapon_families")

	if err := applyPurgeWeaponFamiliesLabels(db); err != nil {
		t.Fatalf("purge 2 (doit être no-op): %v", err)
	}
	after := countRows(t, db, "weapon_families")
	if after != between {
		t.Fatalf("2e passe a modifié rows: between=%d after=%d", between, after)
	}
}
