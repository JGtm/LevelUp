//go:build cgo

package migration

// Tests du DDL `match_weapon_shots`. Ils verrouillent trois choses :
//
//  1. la vue retient LA DERNIERE PASSE ENTIERE par match (meme unite de generation que
//     match_kill_events) — sinon les armes d une passe precedente survivraient a une passe
//     qui ne les decode plus, produisant une ventilation qui n a jamais existe ;
//  2. la vue NE FILTRE PAS sur `publishable` — filtrer ici rendrait le refus invisible et un
//     lecteur ne saurait pas distinguer « aucun tir decode » de « decodage refuse » ;
//  3. `weapon_id` accepte les identifiants filmshell dont le bit de poids fort est a 1 (plus
//     de la moitie du catalogue) — c est la colonne UBIGINT qui le permet.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func weaponShotsDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := applyMatchWeaponShots(db); err != nil {
		t.Fatalf("applyMatchWeaponShots: %v", err)
	}
	if err := applyMatchWeaponShots(db); err != nil {
		t.Fatalf("applyMatchWeaponShots (2e passage): %v", err)
	}
	return db
}

func insertShotRow(t *testing.T, db *sql.DB, matchID, pass string, idx int, weapon string, shots int, publishable bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_weapon_shots (
			match_id, decode_pass, decoder_rev, player_index, weapon_id,
			shots_decoded, publishable, gate_reason
		) VALUES (?, ?, 'test-rev', CAST(? AS SMALLINT), CAST(? AS UBIGINT), ?, ?, 'motif')`,
		matchID, pass, idx, weapon, shots, publishable)
	if err != nil {
		t.Fatalf("insert (%s/%s): %v", matchID, pass, err)
	}
}

// TestVueTirsRetientLaDernierePasse — meme invariant que pour les morts : la passe B (2 armes)
// remplace ENTIEREMENT la passe A (3 armes). Une vue par cle laisserait survivre la 3e arme.
func TestVueTirsRetientLaDernierePasse(t *testing.T) {
	db := weaponShotsDB(t)
	insertShotRow(t, db, "m1", "passeA", 0, "1000", 10, true)
	insertShotRow(t, db, "m1", "passeA", 0, "2000", 20, true)
	insertShotRow(t, db, "m1", "passeA", 1, "3000", 30, true)
	insertShotRow(t, db, "m1", "passeB", 0, "1000", 11, true)
	insertShotRow(t, db, "m1", "passeB", 0, "2000", 21, true)

	var n, total int
	var pass string
	if err := db.QueryRow(`SELECT COUNT(*), SUM(shots_decoded), MIN(decode_pass)
		FROM match_weapon_shots_latest WHERE match_id = 'm1'`).Scan(&n, &total, &pass); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 || pass != "passeB" || total != 32 {
		t.Errorf("vue = %d lignes / passe %q / %d tirs, attendu 2 / passeB / 32", n, pass, total)
	}
	if got := countRows(t, db, "match_weapon_shots"); got != 5 {
		t.Errorf("table = %d lignes, attendu 5 — append-only : rien n est efface", got)
	}
}

// TestVueTirsNeFiltrePasPublishable — le refus doit rester VISIBLE dans la vue. Sinon un
// lecteur confondrait « decodage refuse » et « aucun tir decode ».
func TestVueTirsNeFiltrePasPublishable(t *testing.T) {
	db := weaponShotsDB(t)
	insertShotRow(t, db, "m1", "p1", 0, "1000", 10, true)
	insertShotRow(t, db, "m1", "p1", 1, "2000", 20, false)

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_weapon_shots_latest`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 2 {
		t.Errorf("vue = %d lignes, attendu 2 — le filtre publishable appartient au LECTEUR, "+
			"pas a la vue", n)
	}
}

// TestWeaponIDBitDePoidsFort — Fuel Rod SPNKr = 0x9d6aaed242c9679f. En int64 signe il serait
// NEGATIF : c est le piege que la colonne UBIGINT et le passage en chaine decimale evitent.
func TestWeaponIDBitDePoidsFort(t *testing.T) {
	db := weaponShotsDB(t)
	const fuelRod = "11343070829572876191" // 0x9d6aaed242c9679f

	insertShotRow(t, db, "m1", "p1", 0, fuelRod, 7, true)

	var got string
	if err := db.QueryRow(`SELECT CAST(weapon_id AS VARCHAR) FROM match_weapon_shots_latest`).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != fuelRod {
		t.Errorf("weapon_id relu = %s, attendu %s", got, fuelRod)
	}
}
