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

// TestKillPositionPersistPass_ReDecodeSupersedeLaPasseEntiere — LE cas que le lot 1.7 ferme
// (2026-09-09). `kill_positions_latest` arbitrait par CLE (match_id, killer_xuid, time_ms) :
// une passe B qui ne retrouvait PLUS la position d un frag n ecrivait aucune ligne pour lui,
// et la ligne de la passe A restait servie A JAMAIS, melangee aux nouvelles. La vue arbitre
// desormais par DERNIERE PASSE ENTIERE par match, comme sa soeur kill_openings_latest : la
// position rétractee disparait de la vue, la table brute la garde (append-only).
func TestKillPositionPersistPass_ReDecodeSupersedeLaPasseEntiere(t *testing.T) {
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

	// La passe B ne retrouve QUE le premier kill (le second n a plus de position localisable) —
	// elle ne le publie donc pas du tout (BuildKillPositions n ecrit jamais de ligne vide).
	passeB := []KillPositionInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(9), KillerY: f64(9), KillerZ: f64(9)},
	}
	if err := p.PersistPass(ctx, "m2", passeB); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions WHERE match_id = 'm2'`).Scan(&brut); err != nil {
		t.Fatalf("select table brute: %v", err)
	}
	if brut != 3 {
		t.Errorf("table brute = %d lignes, attendu 3 (append-only : rien n est supprime)", brut)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = 'm2'`).Scan(&n); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 {
		t.Fatalf("kill_positions_latest = %d lignes, attendu 1 (la passe B fait foi ENTIERE)", n)
	}

	var kx111 float64
	if err := db.QueryRow(`SELECT killer_x FROM kill_positions_latest
		WHERE match_id = 'm2' AND killer_xuid = '111' AND time_ms = 1000`).Scan(&kx111); err != nil {
		t.Fatalf("select 111: %v", err)
	}
	if kx111 != 9 {
		t.Errorf("killer_x du kill 111 = %v, attendu 9 (la passe la PLUS RECENTE)", kx111)
	}

	// Explicite, parce que c est LE fait qui compte : plus aucune ligne pour le frag rétracté.
	var survivante int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest
		WHERE match_id = 'm2' AND time_ms = 2000`).Scan(&survivante); err != nil {
		t.Fatalf("select survivante: %v", err)
	}
	if survivante != 0 {
		t.Errorf("%d ligne(s) a t=2000 dans la vue, attendu 0 (la vue arbitre par PASSE, pas par cle)",
			survivante)
	}
}

// TestKillPositionPersistPass_UnSeulDecodePassParPasse — toutes les lignes d une passe portent
// la MEME generation. Deux generations dans une seule passe feraient rendre a la vue une
// FRACTION de passe, ce qui est pire qu une passe entiere perimee.
func TestKillPositionPersistPass_UnSeulDecodePassParPasse(t *testing.T) {
	db := openKillPositionTestDB(t)
	if err := NewKillPositionPersister(db).PersistPass(context.Background(), "m7", []KillPositionInsert{
		{MatchID: "m7", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
		{MatchID: "m7", KillerXUID: "222", TimeMS: 2000, KillerX: f64(2)},
		{MatchID: "m7", KillerXUID: "333", TimeMS: 3000, KillerX: f64(3)},
	}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var distincts int
	if err := db.QueryRow(
		`SELECT COUNT(DISTINCT decode_pass) FROM kill_positions WHERE match_id = 'm7'`).Scan(&distincts); err != nil {
		t.Fatalf("select: %v", err)
	}
	if distincts != 1 {
		t.Errorf("%d decode_pass distincts sur une seule passe, attendu 1", distincts)
	}
}

// TestKillPositionPersistPass_PasseVideNeRetractePas — LA BORNE de la retractation, la meme
// que pour kill_openings : une passe qui ne resout AUCUNE position du match n ecrit RIEN (ni
// ligne, ni decode_pass neuf), donc la vue continue de servir la passe precedente ENTIERE.
// C est assume — la seule alternative serait une ligne sentinelle qu aucun lecteur n exploite —
// et c est epingle ici pour qu un changement de doctrine le dise.
func TestKillPositionPersistPass_PasseVideNeRetractePas(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()
	p := NewKillPositionPersister(db)

	if err := p.PersistPass(ctx, "m6", []KillPositionInsert{
		{MatchID: "m6", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
	}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	if err := p.PersistPass(ctx, "m6", nil); err != nil {
		t.Fatalf("passe B vide: %v", err)
	}

	var n int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), min(killer_x) FROM kill_positions_latest WHERE match_id = 'm6'`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 || kx != 1 {
		t.Errorf("vue = %d ligne(s) / killer_x %v, attendu 1 / 1 — une passe VIDE n ecrit aucune "+
			"generation, la passe A reste donc servie ENTIERE (comportement assume)", n, kx)
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
