//go:build integration

// Package persist — kill_opening_persister_test.go : ce que [KillOpeningPersister] ECRIT, et
// surtout que la lecture passe par la vue `_latest` (ADR 0026) et jamais par la table brute.
//
// Le schema est celui des migrations REELLES (RunForDB sur TargetShared), pas un DDL recopie —
// meme doctrine que kill_position_persister_test.go, dont ce fichier est le jumeau.

package persist

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestKillOpeningPersistPass_EcritEtRelitParLaVue — le chemin nominal, entames a un seul cote
// comprises (une entame dont la victime n est pas localisee reste une entame).
func TestKillOpeningPersistPass_EcritEtRelitParLaVue(t *testing.T) {
	db := openKillPositionTestDB(t) // meme fixture : les deux tables viennent des memes migrations
	ctx := context.Background()

	rows := []KillOpeningInsert{
		{MatchID: "m1", KillerXUID: "111", TimeMS: 1000,
			KillerX: f64(1), KillerY: f64(2), KillerZ: f64(3),
			VictimX: f64(4), VictimY: f64(5), VictimZ: f64(6)},
		{MatchID: "m1", KillerXUID: "222", TimeMS: 2000,
			KillerX: f64(7), KillerY: f64(8), KillerZ: f64(9)},
	}
	if err := NewKillOpeningPersister(db).PersistPass(ctx, "m1", rows); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_openings_latest WHERE match_id = 'm1'`).Scan(&n); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 2 {
		t.Fatalf("kill_openings_latest = %d lignes, attendu 2", n)
	}
	var vx *float64
	if err := db.QueryRow(
		`SELECT victim_x FROM kill_openings_latest WHERE match_id = 'm1' AND time_ms = 2000`).Scan(&vx); err != nil {
		t.Fatalf("select victim_x: %v", err)
	}
	if vx != nil {
		t.Errorf("victim_x = %v, attendu NULL (entame non localisee cote victime)", *vx)
	}
}

// TestKillOpeningPersistPass_ReDecodeSupersede — une seconde passe ne SUPPRIME rien (append-only)
// mais la vue ne sert que la derniere ligne par (match_id, killer_xuid, time_ms). C est ce qui
// rend un re-decodage sur : sans la vue, la table servirait deux entames pour un meme frag.
func TestKillOpeningPersistPass_ReDecodeSupersede(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()
	p := NewKillOpeningPersister(db)

	if err := p.PersistPass(ctx, "m2", []KillOpeningInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
	}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	if err := p.PersistPass(ctx, "m2", []KillOpeningInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(42)},
	}); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_openings WHERE match_id = 'm2'`).Scan(&brut); err != nil {
		t.Fatalf("select table brute: %v", err)
	}
	if brut != 2 {
		t.Errorf("table brute = %d lignes, attendu 2 (append-only : rien n est supprime)", brut)
	}

	var n int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), min(killer_x) FROM kill_openings_latest WHERE match_id = 'm2'`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 || kx != 42 {
		t.Errorf("vue = %d ligne(s) / killer_x %v, attendu 1 / 42 (la derniere passe gagne)", n, kx)
	}
}

// TestKillOpeningPersistPass_PasseVideNEcritRien — une passe sans ligne est journalisee et
// n ecrit rien. Sur cette table le cas est MOINS exceptionnel que pour les positions (un film
// dont aucune mort n a d echantillon 1,5 s plus tot) : il ne doit surtout pas etre une erreur.
func TestKillOpeningPersistPass_PasseVideNEcritRien(t *testing.T) {
	db := openKillPositionTestDB(t)
	if err := NewKillOpeningPersister(db).PersistPass(context.Background(), "m3", nil); err != nil {
		t.Fatalf("passe vide: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_openings WHERE match_id = 'm3'`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites pour une passe vide, attendu 0", n)
	}
}

// TestKillOpeningPersistPass_RefuseLignesIncoherentes — un tueur non identifie (l entame ne se
// joindrait a aucune mort) et un match_id etranger (deux matchs melanges) sont refuses AVANT
// toute transaction.
func TestKillOpeningPersistPass_RefuseLignesIncoherentes(t *testing.T) {
	db := openKillPositionTestDB(t)
	p := NewKillOpeningPersister(db)
	cas := map[string][]KillOpeningInsert{
		"tueur vide":        {{MatchID: "m4", KillerXUID: "", TimeMS: 1000, KillerX: f64(1)}},
		"match_id etranger": {{MatchID: "autre", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)}},
	}
	for nom, rows := range cas {
		if err := p.PersistPass(context.Background(), "m4", rows); err == nil {
			t.Errorf("%s: attendu un refus, obtenu nil", nom)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_openings`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites malgre le refus, attendu 0", n)
	}
}

// TestKillOpeningPersistPass_MatchIDVide — jamais de passe anonyme.
func TestKillOpeningPersistPass_MatchIDVide(t *testing.T) {
	db := openKillPositionTestDB(t)
	if err := NewKillOpeningPersister(db).PersistPass(context.Background(), "",
		[]KillOpeningInsert{{KillerXUID: "111", TimeMS: 1000}}); err == nil {
		t.Fatal("attendu un refus (matchID vide), obtenu nil")
	}
}
