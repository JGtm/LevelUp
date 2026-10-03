//go:build cgo

package migration

// Tests du DDL `match_weapon_hit_distance`. Ils verrouillent trois choses, comme le test
// jumeau de match_weapon_shots :
//
//  1. la vue retient LA DERNIERE PASSE ENTIERE par match (meme unite de generation que
//     match_weapon_shots) — sinon les armes d une passe precedente survivraient a une passe
//     qui ne les decode plus, produisant un histogramme qui n a jamais existe ;
//  2. l append-only n efface rien : la table garde toutes les passes, la vue n en montre qu une ;
//  3. `weapon_id` accepte les identifiants filmshell dont le bit de poids fort est a 1 (plus
//     de la moitie du catalogue) — c est la colonne UBIGINT et l insertion en chaine qui le permettent.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func weaponHitDistanceDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openTmpDB(t)
	if err := applyMatchWeaponHitDistance(db); err != nil {
		t.Fatalf("applyMatchWeaponHitDistance: %v", err)
	}
	// Idempotence : un second passage ne doit pas echouer (CREATE IF NOT EXISTS / OR REPLACE).
	if err := applyMatchWeaponHitDistance(db); err != nil {
		t.Fatalf("applyMatchWeaponHitDistance (2e passage): %v", err)
	}
	return db
}

func insertHitDistanceRow(t *testing.T, db *sql.DB, matchID, pass, xuid, weapon, bucketJSON string, distN int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_weapon_hit_distance (
			match_id, decode_pass, decoder_rev, xuid, weapon_id,
			dist_bucket_json, dist_n
		) VALUES (?, ?, ?, ?, CAST(? AS UBIGINT), ?, ?)`,
		matchID, pass, WeaponHitDistanceDecoderRev, xuid, weapon, bucketJSON, distN)
	if err != nil {
		t.Fatalf("insert (%s/%s): %v", matchID, pass, err)
	}
}

// TestWeaponHitDistanceVueRetientLaDernierePasse — la passe B (2 armes) remplace ENTIEREMENT la
// passe A (3 armes). Une vue par cle laisserait survivre la 3e arme de la passe A.
func TestWeaponHitDistanceVueRetientLaDernierePasse(t *testing.T) {
	db := weaponHitDistanceDB(t)
	insertHitDistanceRow(t, db, "m1", "passeA", "xa", "1000", `[1,2,3]`, 6)
	insertHitDistanceRow(t, db, "m1", "passeA", "xa", "2000", `[0,1,1]`, 2)
	insertHitDistanceRow(t, db, "m1", "passeA", "xb", "3000", `[4,0,0]`, 4)
	insertHitDistanceRow(t, db, "m1", "passeB", "xa", "1000", `[2,2,4]`, 8)
	insertHitDistanceRow(t, db, "m1", "passeB", "xa", "2000", `[1,1,1]`, 3)

	var n, total int
	var pass string
	if err := db.QueryRow(`SELECT COUNT(*), SUM(dist_n), MIN(decode_pass)
		FROM match_weapon_hit_distance_latest WHERE match_id = 'm1'`).Scan(&n, &total, &pass); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 || pass != "passeB" || total != 11 {
		t.Errorf("vue = %d lignes / passe %q / dist_n %d, attendu 2 / passeB / 11", n, pass, total)
	}
	if got := countRows(t, db, "match_weapon_hit_distance"); got != 5 {
		t.Errorf("table = %d lignes, attendu 5 — append-only : rien n est efface", got)
	}
}

// TestWeaponHitDistancePlusieursMatchs — la vue partitionne par match : deux matchs, chacun
// garde SA derniere passe, independamment.
func TestWeaponHitDistancePlusieursMatchs(t *testing.T) {
	db := weaponHitDistanceDB(t)
	insertHitDistanceRow(t, db, "m1", "p1", "xa", "1000", `[1]`, 1)
	insertHitDistanceRow(t, db, "m2", "p1", "xa", "1000", `[2]`, 2)
	insertHitDistanceRow(t, db, "m2", "p2", "xa", "1000", `[9]`, 9)

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_weapon_hit_distance_latest`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	// m1 = 1 ligne (passe p1), m2 = 1 ligne (passe p2, la plus recente) => 2.
	if n != 2 {
		t.Errorf("vue = %d lignes, attendu 2 (m1/p1 + m2/p2)", n)
	}
}

// TestWeaponHitDistanceWeaponIDBitDePoidsFort — Fuel Rod SPNKr = 0x9d6aaed242c9679f. En int64
// signe il serait NEGATIF : c est le piege que la colonne UBIGINT et le passage en chaine
// decimale evitent.
func TestWeaponHitDistanceWeaponIDBitDePoidsFort(t *testing.T) {
	db := weaponHitDistanceDB(t)
	const fuelRod = "11343070829572876191" // 0x9d6aaed242c9679f

	insertHitDistanceRow(t, db, "m1", "p1", "xa", fuelRod, `[3,2,1]`, 6)

	var got string
	if err := db.QueryRow(`SELECT CAST(weapon_id AS VARCHAR) FROM match_weapon_hit_distance_latest`).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != fuelRod {
		t.Errorf("weapon_id relu = %s, attendu %s", got, fuelRod)
	}
}
