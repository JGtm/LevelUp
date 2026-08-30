//go:build integration

// Package persist — kill_position_persister_test.go : ce que [KillPositionPersister] ECRIT, et
// surtout que la lecture passe par la vue `_latest` (ADR 0026) et jamais par la table brute.
//
// Le schema est celui des migrations REELLES (RunForDB sur TargetShared), pas un DDL recopie —
// meme doctrine que kill_events_persister_test.go.

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openKillPositionTestDB(t *testing.T) *sql.DB {
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

func f64(v float64) *float64 { return &v }

// TestKillPositionPersistPass_EcritEtRelitParLaVue — le chemin nominal.
func TestKillPositionPersistPass_EcritEtRelitParLaVue(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()

	rows := []KillPositionInsert{
		{MatchID: "m1", KillerXUID: "111", TimeMS: 1000,
			KillerX: f64(1), KillerY: f64(2), KillerZ: f64(3),
			VictimX: f64(4), VictimY: f64(5), VictimZ: f64(6)},
		{MatchID: "m1", KillerXUID: "222", TimeMS: 2000,
			KillerX: f64(7), KillerY: f64(8), KillerZ: f64(9)}, // victime non localisee : nil autorise
	}
	if err := NewKillPositionPersister(db).PersistPass(ctx, "m1", rows); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'm1'`).Scan(&n); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 {
		t.Fatalf("kill_positions_latest = %d lignes, attendu 2", n)
	}

	var kx, ky, kz float64
	var vx, vy, vz sql.NullFloat64
	err := db.QueryRow(`SELECT killer_x, killer_y, killer_z, victim_x, victim_y, victim_z
		FROM kill_positions_latest WHERE match_id = 'm1' AND killer_xuid = '111' AND time_ms = 1000`).
		Scan(&kx, &ky, &kz, &vx, &vy, &vz)
	if err != nil {
		t.Fatalf("select ligne: %v", err)
	}
	if kx != 1 || ky != 2 || kz != 3 || !vx.Valid || vx.Float64 != 4 {
		t.Errorf("position relue inattendue : killer=(%v,%v,%v) victim_x=%v", kx, ky, kz, vx)
	}

	var victimX2 sql.NullFloat64
	if err := db.QueryRow(`SELECT victim_x FROM kill_positions_latest
		WHERE match_id = 'm1' AND killer_xuid = '222' AND time_ms = 2000`).Scan(&victimX2); err != nil {
		t.Fatalf("select ligne 2: %v", err)
	}
	if victimX2.Valid {
		t.Errorf("victim_x devait rester NULL (victime non localisee), lu %v", victimX2.Float64)
	}
}

// TestKillPositionPersistPass_ReDecodeNeSupprimeQueLaClePartagee — le dedoublonnage de
// `kill_positions_latest` est PAR LIGNE (match_id, killer_xuid, time_ms), pas par passe entiere
// (contrairement a match_kill_events/decode_pass) : un re-decodage qui ne retrouve QU UNE partie
// des kills ne doit PAS effacer les positions des kills qu il n a pas retrouves.
func TestKillPositionPersistPass_ReDecodeNeSupprimeQueLaClePartagee(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()
	p := NewKillPositionPersister(db)

	passeA := []KillPositionInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1), KillerY: f64(1), KillerZ: f64(1)},
		{MatchID: "m2", KillerXUID: "222", TimeMS: 2000, KillerX: f64(2), KillerY: f64(2), KillerZ: f64(2)},
	}
	if err := p.PersistPass(ctx, "m2", passeA); err != nil {
		t.Fatalf("passe A: %v", err)
	}

	// La passe B ne retrouve QUE le premier kill (le second n a pas de position cette fois) —
	// elle ne le publie donc pas du tout (BuildKillPositions n ecrit jamais de ligne vide).
	passeB := []KillPositionInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(9), KillerY: f64(9), KillerZ: f64(9)},
	}
	if err := p.PersistPass(ctx, "m2", passeB); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'm2'`).Scan(&n); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 {
		t.Fatalf("kill_positions_latest = %d lignes, attendu 2 (le kill 222 de la passe A survit)", n)
	}

	var kx111 float64
	if err := db.QueryRow(`SELECT killer_x FROM kill_positions_latest
		WHERE match_id = 'm2' AND killer_xuid = '111' AND time_ms = 1000`).Scan(&kx111); err != nil {
		t.Fatalf("select 111: %v", err)
	}
	if kx111 != 9 {
		t.Errorf("killer_x du kill 111 = %v, attendu 9 (la ligne la PLUS RECENTE)", kx111)
	}
}

// TestKillPositionPersistPass_PasseVideNEcritRien — une passe sans ligne est journalisee et
// ignoree, jamais une erreur (un match dont aucun kill n a de position localisable est un cas
// normal, pas une panne).
func TestKillPositionPersistPass_PasseVideNEcritRien(t *testing.T) {
	db := openKillPositionTestDB(t)
	if err := NewKillPositionPersister(db).PersistPass(context.Background(), "m3", nil); err != nil {
		t.Fatalf("PersistPass(vide): %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("select table: %v", err)
	}
	if n != 0 {
		t.Errorf("kill_positions = %d lignes, attendu 0", n)
	}
}

// TestKillPositionPersistPass_RefuseKillerXUIDVide — une ligne sans tueur identifie serait
// injointable (la cle fonctionnelle EST killer_xuid) : le persister la refuse plutot que
// d ecrire une ligne qui ne se relira jamais.
func TestKillPositionPersistPass_RefuseKillerXUIDVide(t *testing.T) {
	db := openKillPositionTestDB(t)
	err := NewKillPositionPersister(db).PersistPass(context.Background(), "m4",
		[]KillPositionInsert{{MatchID: "m4", KillerXUID: "", TimeMS: 1000, KillerX: f64(1)}})
	if err == nil {
		t.Fatal("attendu un refus (killer_xuid vide), obtenu nil")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions`).Scan(&n); err != nil {
		t.Fatalf("select table: %v", err)
	}
	if n != 0 {
		t.Errorf("le refus doit laisser la table intacte (0 ligne), lu %d", n)
	}
}

// TestKillPositionPersistPass_RefuseMatchIDIncoherent — une ligne portant un match_id different
// du parametre serait une passe qui se trompe de match ; refuser vaut mieux qu ecrire silencieux.
func TestKillPositionPersistPass_RefuseMatchIDIncoherent(t *testing.T) {
	db := openKillPositionTestDB(t)
	err := NewKillPositionPersister(db).PersistPass(context.Background(), "m5",
		[]KillPositionInsert{{MatchID: "AUTRE", KillerXUID: "111", TimeMS: 1000}})
	if err == nil {
		t.Fatal("attendu un refus (match_id incoherent), obtenu nil")
	}
}
