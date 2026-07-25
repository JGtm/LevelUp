//go:build integration

// Tests pour la migration purge_weapons_name_fr_column (V721-05.1).
//
// Couvre les 3 garanties demandées par le plan V721-05 :
//   - retire bien la colonne sur une DB legacy qui la porte (schéma pré-V72-06) ;
//   - ne casse rien sur une DB qui ne l'a jamais eue (schéma post-V72-06) ;
//   - conserve toutes les rows et la PRIMARY KEY composite (title_slug, weapon_key).
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedWeaponsLegacySchema crée `weapons` avec le schéma PRÉ-V72-06 (name_fr juste
// après `name`, cf. commit cebe2fed9) et 3 rows représentatives (2 titres, un extra
// JSON non trivial pour vérifier que les colonnes non-name_fr survivent intactes).
func seedWeaponsLegacySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE weapons (
			weapon_key   VARCHAR NOT NULL,
			title_slug   VARCHAR NOT NULL,
			name         VARCHAR NOT NULL,
			name_fr      VARCHAR,
			class        VARCHAR,
			role         VARCHAR,
			family_key   VARCHAR,
			faction      VARCHAR,
			damage_type  VARCHAR,
			manufacturer VARCHAR,
			extra        JSON,
			PRIMARY KEY (title_slug, weapon_key)
		);
	`); err != nil {
		t.Fatalf("seed legacy schema: %v", err)
	}
	rows := []struct{ key, title, name, nameFR, class, role string }{
		{"hinf_br75", "halo_infinite", "BR75", "BR75", "shoulder", "precision"},
		{"hinf_sidekick", "halo_infinite", "Mk50 Sidekick", "Mk50 Sidekick", "sidearm", "sidearm"},
		{"h5_magnum", "halo_5", "Magnum (M6H2)", "Magnum (M6H2)", "sidearm", "sidearm"},
	}
	for _, r := range rows {
		if _, err := db.Exec(`
			INSERT INTO weapons (weapon_key, title_slug, name, name_fr, class, role, family_key, faction, damage_type, manufacturer)
			VALUES (?, ?, ?, ?, ?, ?, 'magnum', 'human', 'ballistic', 'Misriah Armory')
		`, r.key, r.title, r.name, r.nameFR, r.class, r.role); err != nil {
			t.Fatalf("insert %s: %v", r.key, err)
		}
	}
}

// seedWeaponsCurrentSchema crée `weapons` avec le schéma POST-V72-06 (sans
// name_fr) — reproduit une DB neuve où add_weapon_registry a déjà tourné.
func seedWeaponsCurrentSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE weapons (
			weapon_key   VARCHAR NOT NULL,
			title_slug   VARCHAR NOT NULL,
			name         VARCHAR NOT NULL,
			class        VARCHAR,
			role         VARCHAR,
			family_key   VARCHAR,
			faction      VARCHAR,
			damage_type  VARCHAR,
			manufacturer VARCHAR,
			extra        JSON,
			PRIMARY KEY (title_slug, weapon_key)
		);
		INSERT INTO weapons (weapon_key, title_slug, name, class, role, family_key, faction, damage_type, manufacturer)
		VALUES ('hinf_br75', 'halo_infinite', 'BR75', 'shoulder', 'precision', 'battle_rifle', 'human', 'ballistic', 'Misriah Armory');
	`); err != nil {
		t.Fatalf("seed current schema: %v", err)
	}
}

// TestPurgeWeaponsNameFR_RemovesColumnWhenPresent : DB legacy → la colonne
// name_fr disparaît, toutes les autres colonnes survivent.
func TestPurgeWeaponsNameFR_RemovesColumnWhenPresent(t *testing.T) {
	db := openMemDB(t)
	seedWeaponsLegacySchema(t, db)

	hasBefore, err := columnExists(db, "weapons", "name_fr")
	if err != nil || !hasBefore {
		t.Fatalf("seed attendu avec name_fr, got exists=%v err=%v", hasBefore, err)
	}

	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("applyPurgeWeaponsNameFR: %v", err)
	}

	hasAfter, err := columnExists(db, "weapons", "name_fr")
	if err != nil {
		t.Fatalf("columnExists after: %v", err)
	}
	if hasAfter {
		t.Fatal("name_fr toujours présente après la purge")
	}

	for _, col := range []string{"weapon_key", "title_slug", "name", "class", "role", "family_key", "faction", "damage_type", "manufacturer", "extra"} {
		has, err := columnExists(db, "weapons", col)
		if err != nil {
			t.Fatalf("columnExists(%s): %v", col, err)
		}
		if !has {
			t.Errorf("colonne %q perdue par la purge (attendue préservée)", col)
		}
	}
}

// TestPurgeWeaponsNameFR_PreservesRowsAndPrimaryKey : les 3 rows survivent avec
// leurs valeurs intactes, et la PK composite (title_slug, weapon_key) est recréée
// (un doublon doit échouer après la purge).
func TestPurgeWeaponsNameFR_PreservesRowsAndPrimaryKey(t *testing.T) {
	db := openMemDB(t)
	seedWeaponsLegacySchema(t, db)

	before := countRows(t, db, "weapons")
	if before != 3 {
		t.Fatalf("seed attendu 3 rows, got %d", before)
	}

	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("applyPurgeWeaponsNameFR: %v", err)
	}

	after := countRows(t, db, "weapons")
	if after != before {
		t.Fatalf("rows perdues par la purge: before=%d after=%d", before, after)
	}

	// Valeur préservée sur une row précise.
	var name, class string
	if err := db.QueryRow(`SELECT name, class FROM weapons WHERE title_slug='halo_5' AND weapon_key='h5_magnum'`).
		Scan(&name, &class); err != nil {
		t.Fatalf("select h5_magnum after purge: %v", err)
	}
	if name != "Magnum (M6H2)" || class != "sidearm" {
		t.Errorf("h5_magnum altérée par la purge: name=%q class=%q", name, class)
	}

	// PK recréée : un doublon (title_slug, weapon_key) doit être rejeté.
	if _, err := db.Exec(`
		INSERT INTO weapons (weapon_key, title_slug, name, class, role, family_key, faction, damage_type, manufacturer)
		VALUES ('hinf_br75', 'halo_infinite', 'dup', 'shoulder', 'precision', 'battle_rifle', 'human', 'ballistic', 'Misriah Armory')
	`); err == nil {
		t.Fatal("PK composite absente après purge : INSERT dupliqué a réussi")
	}
}

// TestPurgeWeaponsNameFR_NoOpWhenColumnAbsent : DB déjà sur le schéma courant
// (post-V72-06) → aucune erreur, rows et schéma inchangés (no-op idempotent).
func TestPurgeWeaponsNameFR_NoOpWhenColumnAbsent(t *testing.T) {
	db := openMemDB(t)
	seedWeaponsCurrentSchema(t, db)

	before := countRows(t, db, "weapons")

	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("applyPurgeWeaponsNameFR (no-op attendu): %v", err)
	}

	after := countRows(t, db, "weapons")
	if after != before {
		t.Fatalf("no-op attendu, rows modifiées: before=%d after=%d", before, after)
	}
	has, err := columnExists(db, "weapons", "name_fr")
	if err != nil {
		t.Fatalf("columnExists: %v", err)
	}
	if has {
		t.Fatal("name_fr ne devrait jamais apparaître (schéma courant ne la crée pas)")
	}
}

// TestPurgeWeaponsNameFR_NoOpWhenTableAbsent : pas de crash si `weapons` n'existe
// pas encore (add_weapon_registry pas encore joué, ou provider title-owned nil
// dans le binaire de test du package migration).
func TestPurgeWeaponsNameFR_NoOpWhenTableAbsent(t *testing.T) {
	db := openMemDB(t)
	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("applyPurgeWeaponsNameFR (table absente): %v", err)
	}
}

// TestPurgeWeaponsNameFR_IdempotentSecondRun : un 2e appel sur une DB déjà
// purgée est un no-op (la colonne n'existe plus → early-return avant tout DDL).
func TestPurgeWeaponsNameFR_IdempotentSecondRun(t *testing.T) {
	db := openMemDB(t)
	seedWeaponsLegacySchema(t, db)

	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("purge 1: %v", err)
	}
	between := countRows(t, db, "weapons")

	if err := applyPurgeWeaponsNameFR(db); err != nil {
		t.Fatalf("purge 2 (doit être no-op): %v", err)
	}
	after := countRows(t, db, "weapons")
	if after != between {
		t.Fatalf("2e passe a modifié rows: between=%d after=%d", between, after)
	}
}
