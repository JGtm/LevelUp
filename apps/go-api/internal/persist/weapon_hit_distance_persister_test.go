//go:build integration

// Package persist — weapon_hit_distance_persister_test.go : le numerateur film (accuracy +
// distance), la porte d effectif, la garde d idempotence weapon_accuracy et l append-only distance.

package persist

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openHitDistanceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

func distBatch(rows ...WeaponHitDistanceRow) WeaponHitDistanceBatch {
	return WeaponHitDistanceBatch{MatchID: "m-1", DecoderRev: migration.WeaponHitDistanceDecoderRev, Rows: rows}
}

func countTableRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestHitsGatePredicate — la porte d effectif, seule copie de la regle Nmin.
func TestWeaponHitDistanceGatePredicate(t *testing.T) {
	if EvaluateHitsGate(migration.WeaponHitsMinShots - 1) {
		t.Errorf("effectif sous Nmin (%d) devrait etre refuse", migration.WeaponHitsMinShots-1)
	}
	if !EvaluateHitsGate(migration.WeaponHitsMinShots) {
		t.Errorf("effectif = Nmin (%d) devrait passer", migration.WeaponHitsMinShots)
	}
}

// TestWeaponHitDistancePorteEffectifAccuracy — weapon_accuracy n ecrit QUE les armes >= Nmin.
func TestWeaponHitDistancePorteEffectifAccuracy(t *testing.T) {
	db := openHitDistanceTestDB(t)
	acc := []WeaponAccuracyInsert{
		{MatchID: "m-1", XUID: "xuid(1)", WeaponID: 0x9d6aaed242c9679f, ShotsFired: 10, ShotsLanded: 6}, // >= 8, bit fort a 1
		{MatchID: "m-1", XUID: "xuid(1)", WeaponID: 2000, ShotsFired: 4, ShotsLanded: 3},                // < 8 : ecarte
	}
	if err := NewWeaponHitDistancePersister(db).PersistPass(context.Background(), acc, distBatch()); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	if n := countTableRows(t, db, "weapon_accuracy"); n != 1 {
		t.Fatalf("attendu 1 ligne weapon_accuracy (>= Nmin), obtenu %d", n)
	}
	var wid uint64
	var sf, sl int
	if err := db.QueryRow(`SELECT weapon_id, shots_fired, shots_landed FROM weapon_accuracy`).Scan(&wid, &sf, &sl); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if wid != 0x9d6aaed242c9679f || sf != 10 || sl != 6 {
		t.Errorf("ligne inattendue : wid=%x sf=%d sl=%d (bit fort a 1 doit survivre au UBIGINT)", wid, sf, sl)
	}
}

// TestWeaponHitDistanceIdempotenceAccuracy — un 2e run ne DOUBLE pas weapon_accuracy (garde SELECT),
// mais REGENERE la distance (nouveau decode_pass, _latest retient la derniere passe).
func TestWeaponHitDistanceIdempotenceAccuracy(t *testing.T) {
	db := openHitDistanceTestDB(t)
	acc := []WeaponAccuracyInsert{{MatchID: "m-1", XUID: "xuid(1)", WeaponID: 1000, ShotsFired: 12, ShotsLanded: 8}}
	dist := distBatch(WeaponHitDistanceRow{XUID: "xuid(1)", WeaponID: 1000, DistBucketJSON: "[8,0,0]", DistN: 8})
	p := NewWeaponHitDistancePersister(db)
	for i := 0; i < 2; i++ {
		if err := p.PersistPass(context.Background(), acc, dist); err != nil {
			t.Fatalf("PersistPass run %d: %v", i, err)
		}
	}
	if n := countTableRows(t, db, "weapon_accuracy"); n != 1 {
		t.Errorf("weapon_accuracy double par le re-run : %d (attendu 1, garde d idempotence)", n)
	}
	if n := countTableRows(t, db, "match_weapon_hit_distance"); n != 2 {
		t.Errorf("distance : attendu 2 lignes brutes (2 passes append-only), obtenu %d", n)
	}
	if n := countTableRows(t, db, "match_weapon_hit_distance_latest"); n != 1 {
		t.Errorf("vue _latest : attendu 1 ligne (derniere passe), obtenu %d", n)
	}
}

// TestWeaponHitDistanceValidationRefus — les refus de la validation de passe.
func TestWeaponHitDistanceValidationRefus(t *testing.T) {
	db := openHitDistanceTestDB(t)
	p := NewWeaponHitDistancePersister(db)
	cas := []struct {
		nom   string
		batch WeaponHitDistanceBatch
		frag  string
	}{
		{"match_id vide", WeaponHitDistanceBatch{DecoderRev: "whd-v1"}, "MatchID vide"},
		{"decoder_rev vide", WeaponHitDistanceBatch{MatchID: "m-1"}, "DecoderRev vide"},
		{"dist_n nul", distBatch(WeaponHitDistanceRow{XUID: "x", WeaponID: 1, DistN: 0}), "dist_n <= 0"},
		{"arme nulle", distBatch(WeaponHitDistanceRow{XUID: "x", WeaponID: 0, DistN: 3}), "arme nulle"},
		{"xuid vide", distBatch(WeaponHitDistanceRow{XUID: "", WeaponID: 1, DistN: 3}), "sans xuid"},
		{"doublon", distBatch(
			WeaponHitDistanceRow{XUID: "x", WeaponID: 1, DistN: 3},
			WeaponHitDistanceRow{XUID: "x", WeaponID: 1, DistN: 4},
		), "doublon"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := p.PersistPass(context.Background(), nil, c.batch)
			if err == nil || !strings.Contains(err.Error(), c.frag) {
				t.Errorf("attendu erreur contenant %q, obtenu %v", c.frag, err)
			}
		})
	}
}

// TestWeaponHitDistancePasseVide — une passe entierement vide n ecrit rien et ne casse pas.
func TestWeaponHitDistancePasseVide(t *testing.T) {
	db := openHitDistanceTestDB(t)
	if err := NewWeaponHitDistancePersister(db).PersistPass(context.Background(), nil, distBatch()); err != nil {
		t.Fatalf("passe vide: %v", err)
	}
	if n := countTableRows(t, db, "match_weapon_hit_distance"); n != 0 {
		t.Errorf("passe vide a ecrit %d lignes distance", n)
	}
}
