//go:build cgo

package migration

// Tests cardinalité de l'append-only weapon_kills + vue v_weapon_kills.
// Verrouille la sémantique « dernière génération par (match_id, xuid) » (DENSE_RANK)
// et la colonne dérivée effective_weapon_id = COALESCE(reconciled_as, weapon_id).
//
// Note : on construit ici le schéma weapon_kills PER-KILL (celui que cible la
// migration append-only en prod : add_weapon_kills + reconciled_as, sans PK) et on
// appelle applyAppendOnlyWeaponKills directement. RunForDB(TargetShared) n'est PAS
// utilisé car le bootstrap shared porte une définition agrégée concurrente (PK
// match_id,xuid,weapon_id, colonne `kills`) incompatible avec le modèle générationnel.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupPerKillWeaponKills crée le schéma per-kill (équivalent add_weapon_kills +
// add_weapon_kills_reconciled_as) puis applique l'append-only.
func setupPerKillWeaponKills(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t) // défini dans append_only_rebuild_test.go (même tag cgo)
	if _, err := db.Exec(`
		CREATE TABLE weapon_kills (
			match_id       VARCHAR NOT NULL,
			xuid           VARCHAR NOT NULL,
			time_ms        INTEGER NOT NULL,
			weapon_id      UBIGINT,
			delta_ms       INTEGER,
			confidence     VARCHAR NOT NULL DEFAULT 'none',
			swap_detected  BOOLEAN NOT NULL DEFAULT FALSE,
			delayed_damage BOOLEAN NOT NULL DEFAULT FALSE,
			reconciled_as  UBIGINT,
			attribution_path VARCHAR DEFAULT 'none',
			player_index   INTEGER
		)`); err != nil {
		t.Fatalf("create per-kill weapon_kills: %v", err)
	}
	if err := applyAppendOnlyWeaponKills(db); err != nil {
		t.Fatalf("applyAppendOnlyWeaponKills: %v", err)
	}
	return db
}

func wkInsert(t *testing.T, db *sql.DB, match, xuid string, timeMs int, weaponID uint64, gen int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, generation_id)
		 VALUES (?, ?, ?, ?, ?)`,
		match, xuid, timeMs, weaponID, gen,
	); err != nil {
		t.Fatalf("insert weapon_kills (%s,%s,gen%d): %v", match, xuid, gen, err)
	}
}

func wkCount(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return n
}

// TestWeaponKillsAppendOnly_GenerationSupersedes : une nouvelle génération (id
// supérieur) remplace l'ENSEMBLE des kills d'un (match,xuid) dans v_weapon_kills,
// par INSERT pur — la table physique croît, la vue ne montre que la dernière génération.
func TestWeaponKillsAppendOnly_GenerationSupersedes(t *testing.T) {
	db := setupPerKillWeaponKills(t)

	// Append-only appliqué : colonnes generation_id + written_at présentes.
	for _, col := range []string{"generation_id", "written_at"} {
		if has, _ := columnExists(db, "weapon_kills", col); !has {
			t.Fatalf("colonne %s absente — append-only weapon_kills non appliqué", col)
		}
	}

	// Génération 0 (legacy) : 2 kills arme 100 pour (m1,x1).
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	wkInsert(t, db, "m1", "x1", 2000, 100, 0)
	// Indépendance par clé : (m1,x2) garde sa génération 0.
	wkInsert(t, db, "m1", "x2", 1500, 555, 0)

	// Génération 1 (réécriture) : 3 kills arme 200 pour (m1,x1).
	wkInsert(t, db, "m1", "x1", 1100, 200, 1)
	wkInsert(t, db, "m1", "x1", 2100, 200, 1)
	wkInsert(t, db, "m1", "x1", 3100, 200, 1)

	// Table physique : append pur → 6 rows (2+1+3), rien supprimé.
	if got := wkCount(t, db, `SELECT COUNT(*) FROM weapon_kills`); got != 6 {
		t.Fatalf("weapon_kills physique = %d, attendu 6 (append pur)", got)
	}

	// Vue : (m1,x1) ne montre que la génération 1 = 3 kills, tous arme 200.
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1'`); got != 3 {
		t.Fatalf("v_weapon_kills (m1,x1) = %d, attendu 3 (dernière génération)", got)
	}
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1' AND weapon_id=200`); got != 3 {
		t.Fatalf("v_weapon_kills (m1,x1) arme 200 = %d, attendu 3 (gen 0 supersédée)", got)
	}
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1' AND weapon_id=100`); got != 0 {
		t.Fatalf("v_weapon_kills (m1,x1) arme 100 = %d, attendu 0 (génération périmée)", got)
	}

	// Indépendance : (m1,x2) intacte (sa génération 0 reste la MAX pour cette clé).
	if got := wkCount(t, db,
		`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='x2'`); got != 1 {
		t.Fatalf("v_weapon_kills (m1,x2) = %d, attendu 1 (clé indépendante)", got)
	}
}

// TestWeaponKillsAppendOnly_EffectiveWeaponID : effective_weapon_id = reconciled_as
// quand présent, sinon weapon_id brut.
func TestWeaponKillsAppendOnly_EffectiveWeaponID(t *testing.T) {
	db := setupPerKillWeaponKills(t)

	// Kill arme brute 100 sans reconciled_as → effective = 100.
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	// Kill arme brute 200 reconciliée vers 999 → effective = 999.
	if _, err := db.Exec(
		`INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, reconciled_as, generation_id)
		 VALUES ('m2','x1',1000,200,999,0)`); err != nil {
		t.Fatalf("insert reconciled: %v", err)
	}

	var eff uint64
	if err := db.QueryRow(
		`SELECT effective_weapon_id FROM v_weapon_kills WHERE match_id='m1' AND xuid='x1'`).Scan(&eff); err != nil {
		t.Fatalf("select effective m1: %v", err)
	}
	if eff != 100 {
		t.Fatalf("effective_weapon_id (sans reconcile) = %d, attendu 100", eff)
	}
	if err := db.QueryRow(
		`SELECT effective_weapon_id FROM v_weapon_kills WHERE match_id='m2' AND xuid='x1'`).Scan(&eff); err != nil {
		t.Fatalf("select effective m2: %v", err)
	}
	if eff != 999 {
		t.Fatalf("effective_weapon_id (reconciled_as=999) = %d, attendu 999", eff)
	}
}

// TestWeaponKillsAppendOnly_Idempotent : 2e application = no-op, vue + colonnes stables.
func TestWeaponKillsAppendOnly_Idempotent(t *testing.T) {
	db := setupPerKillWeaponKills(t)
	wkInsert(t, db, "m1", "x1", 1000, 100, 0)
	if err := applyAppendOnlyWeaponKills(db); err != nil {
		t.Fatalf("applyAppendOnlyWeaponKills pass2 (idempotence): %v", err)
	}
	if got := wkCount(t, db, `SELECT COUNT(*) FROM weapon_kills`); got != 1 {
		t.Fatalf("weapon_kills après pass2 = %d, attendu 1 (données préservées)", got)
	}
	if got := wkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills`); got != 1 {
		t.Fatalf("v_weapon_kills après pass2 = %d, attendu 1", got)
	}
}
