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

// TestKillOpeningPersistPass_ReDecodeSupersede — une seconde passe ne SUPPRIME rien
// (append-only) mais la vue ne sert que la DERNIERE PASSE ENTIERE du match (`decode_pass`).
//
// LE CAS QUI JUSTIFIE `decode_pass`, ET QU UN ARBITRAGE PAR CLE RATAIT (revue adversariale du
// 2026-09-06, constat D) : la passe B ne resout PLUS l entame du frag t=2000 — elle n ecrit
// aucune ligne pour lui, ce qui est le cas NOMINAL quand le filtre « meme vie » de
// `replay.BuildKillOpenings` ecarte les deux cotes. Avec une vue qui retenait la derniere ligne
// par (match_id, killer_xuid, time_ms), l entame de la passe A pour ce frag serait restee
// servie A JAMAIS, melangee aux lignes de B. Ici elle DISPARAIT.
func TestKillOpeningPersistPass_ReDecodeSupersede(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()
	p := NewKillOpeningPersister(db)

	// Passe A : deux frags resolus.
	if err := p.PersistPass(ctx, "m2", []KillOpeningInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
		{MatchID: "m2", KillerXUID: "111", TimeMS: 2000, KillerX: f64(7)},
	}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	// Passe B : le second frag n a plus d entame lisible.
	if err := p.PersistPass(ctx, "m2", []KillOpeningInsert{
		{MatchID: "m2", KillerXUID: "111", TimeMS: 1000, KillerX: f64(42)},
	}); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_openings WHERE match_id = 'm2'`).Scan(&brut); err != nil {
		t.Fatalf("select table brute: %v", err)
	}
	if brut != 3 {
		t.Errorf("table brute = %d lignes, attendu 3 (append-only : rien n est supprime)", brut)
	}

	var n int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), min(killer_x) FROM kill_openings_latest WHERE match_id = 'm2'`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 || kx != 42 {
		t.Errorf("vue = %d ligne(s) / killer_x %v, attendu 1 / 42 — la passe B fait foi ENTIERE, "+
			"l entame que B ne resout plus ne doit PAS survivre depuis A", n, kx)
	}

	// Explicite, parce que c est LE fait qui compte : plus aucune ligne a t=2000.
	var survivante int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM kill_openings_latest WHERE match_id = 'm2' AND time_ms = 2000`).
		Scan(&survivante); err != nil {
		t.Fatalf("select survivante: %v", err)
	}
	if survivante != 0 {
		t.Errorf("%d ligne(s) a t=2000 dans la vue, attendu 0 (la vue arbitre par PASSE, pas par cle)",
			survivante)
	}
}

// TestKillOpeningPersistPass_PasseVideNeRetractePas — LA BORNE DE LA RÉTRACTATION (résidu 4.0b
// du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md), assertée TELLE QU ELLE EST et non telle qu on
// l aimerait.
//
// Une passe B qui ne resout AUCUNE entame de tout le match n ecrit RIEN : ni ligne, ni
// `decode_pass` neuf. La vue continue donc de servir la passe A ENTIERE. C est la limite du
// mecanisme de `decode_pass`, il est assume (cf. l en-tete de la migration
// steps_shared_kill_openings.go, section « la retractation exige que la passe suivante ecrive
// au moins une ligne ») et il est epingle ici pour qu un changement de doctrine le dise.
func TestKillOpeningPersistPass_PasseVideNeRetractePas(t *testing.T) {
	db := openKillPositionTestDB(t)
	ctx := context.Background()
	p := NewKillOpeningPersister(db)

	if err := p.PersistPass(ctx, "m6", []KillOpeningInsert{
		{MatchID: "m6", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
	}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	// Passe B : plus AUCUNE entame lisible sur ce match (filtre « meme vie » de
	// replay.BuildKillOpenings, film re-telecharge plus court...). Aucune erreur, aucune ligne.
	if err := p.PersistPass(ctx, "m6", nil); err != nil {
		t.Fatalf("passe B vide: %v", err)
	}

	var n int
	var kx float64
	if err := db.QueryRow(
		`SELECT COUNT(*), min(killer_x) FROM kill_openings_latest WHERE match_id = 'm6'`).Scan(&n, &kx); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if n != 1 || kx != 1 {
		t.Errorf("vue = %d ligne(s) / killer_x %v, attendu 1 / 1 — une passe VIDE n ecrit aucune "+
			"generation, la passe A reste donc servie ENTIERE (comportement assume)", n, kx)
	}
}

// TestKillOpeningPersistPass_UnSeulDecodePassParPasse — toutes les lignes d une passe portent
// la MEME generation. Deux generations dans une seule passe feraient rendre a la vue une
// FRACTION de passe, ce qui est pire qu une passe entiere perimee.
func TestKillOpeningPersistPass_UnSeulDecodePassParPasse(t *testing.T) {
	db := openKillPositionTestDB(t)
	if err := NewKillOpeningPersister(db).PersistPass(context.Background(), "m5", []KillOpeningInsert{
		{MatchID: "m5", KillerXUID: "111", TimeMS: 1000, KillerX: f64(1)},
		{MatchID: "m5", KillerXUID: "222", TimeMS: 2000, KillerX: f64(2)},
		{MatchID: "m5", KillerXUID: "333", TimeMS: 3000, KillerX: f64(3)},
	}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var distincts int
	if err := db.QueryRow(
		`SELECT COUNT(DISTINCT decode_pass) FROM kill_openings WHERE match_id = 'm5'`).Scan(&distincts); err != nil {
		t.Fatalf("select: %v", err)
	}
	if distincts != 1 {
		t.Errorf("%d decode_pass distincts sur une seule passe, attendu 1", distincts)
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
