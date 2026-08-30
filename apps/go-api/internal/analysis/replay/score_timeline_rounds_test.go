package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// score_timeline_rounds_test.go — LE CALQUE DU SCORE PAR JOUEUR MIGRE VERS L'IDENTITE PAR MANCHE.
//
// Le slot d'entite du statborg est REATTRIBUE d'une manche a l'autre, et le compteur de morts
// repart de zero par manche : le pont plat (par les totaux, ou par les morts vues sur tout le
// match) ne voit que la manche 0 et n'apparie qu'un joueur. Le nouveau chemin decoupe la courbe de
// chaque slot aux bornes de manche, rattache chaque segment au joueur de SA manche (par les
// instants de mort), et fusionne par xuid les segments d'un meme joueur. Les enregistrements sont
// SYNTHETIQUES — le test tourne en CI, sans film, comme score_timeline_test.go.

// multiRoundClock : origine 0, pas de 100 ms, 300 frames (max 30 000 ms) — assez large pour que la
// manche 1 (11 000-13 500 ms) tombe DANS la fenetre, la ou testClock plafonne a 11 000 ms.
func multiRoundClock() scoreClock {
	return scoreClock{intervalMS: 100, frames: 300, originMS: 0}
}

// reassignFixture fabrique un film a DEUX manches ou DEUX slots sont croises :
//
//	manche 0 : slot 22 = A (1001), slot 20 = B (1002)
//	manche 1 : slot 22 = B (1002), slot 20 = A (1001)
//
// Chaque joueur occupe donc des SLOTS DIFFERENTS selon la manche : sa courbe DOIT fusionner deux
// slots. Le compteur de morts (comp 2 B) est l'ancre d'identite ; ses instants correspondent au
// fil des morts. Chaque (slot, manche) porte aussi un train de score de mode (comp 0 A, >= 3
// emissions) pour que RealRounds tienne les deux manches pour reelles.
func reassignFixture() ([]objectiveevents.StatRecord, []Death) {
	var recs []objectiveevents.StatRecord
	// Score de mode : trois emissions croissantes par manche sur le slot 22 -> manches 0 et 1.
	recs = append(recs, modeRamp(22, 0, 1_000, 1_000, 10, 20, 30)...)
	recs = append(recs, modeRamp(22, 1, 11_000, 1_000, 10, 20, 30)...)
	// Compteurs par (slot, manche) : perso (comp1 B), frags/morts (comp2 A/B), assist (comp3 A).
	// Les valeurs sont PROPRES A LA MANCHE (non cumulees) ; les morts (comp2 B) 1/2/3 datent les
	// instants d'identite.
	add := func(slot, round, t0 int, personal [3]int64) {
		for i := 0; i < 3; i++ {
			k := int64(i + 1)
			recs = append(recs, coreLine(slot, round, t0+i*1_000, k, k, k, personal[i])...)
		}
	}
	add(22, 0, 1_000, [3]int64{100, 200, 300}) // A, manche 0
	add(20, 0, 1_500, [3]int64{40, 80, 120})   // B, manche 0
	add(22, 1, 11_000, [3]int64{50, 150, 250}) // B, manche 1
	add(20, 1, 11_500, [3]int64{60, 160, 260}) // A, manche 1
	deaths := []Death{
		{XUID: 1001, TimeMS: 1_000}, {XUID: 1001, TimeMS: 2_000}, {XUID: 1001, TimeMS: 3_000},
		{XUID: 1002, TimeMS: 1_500}, {XUID: 1002, TimeMS: 2_500}, {XUID: 1002, TimeMS: 3_500},
		{XUID: 1002, TimeMS: 11_000}, {XUID: 1002, TimeMS: 12_000}, {XUID: 1002, TimeMS: 13_000},
		{XUID: 1001, TimeMS: 11_500}, {XUID: 1001, TimeMS: 12_500}, {XUID: 1001, TimeMS: 13_500},
	}
	return recs, deaths
}

// playerByXUID rend le PlayerScore d'un xuid, ou nil.
func playerByXUID(players []PlayerScore, xuid string) *PlayerScore {
	for i := range players {
		if players[i].XUID == xuid {
			return &players[i]
		}
	}
	return nil
}

// lastRoundValue rend la derniere valeur de la manche `round` d'une serie, ou -1 si absente.
func lastRoundValue(s ScoreSeries, round int) int {
	for _, r := range s.Rounds {
		if r.Round == round && len(r.Points) > 0 {
			return r.Points[len(r.Points)-1].V
		}
	}
	return -1
}

// TestScoreTimelineReassignedSlotAttributesToRoundOwner — LE CAS QUI FONDE LE LOT.
//
// En manche 1, le slot 22 est B (pas A) : le score personnel de B en manche 1 doit LUI etre
// attribue. Le total de chaque joueur fusionne ses deux slots dans l'ordre du temps :
//
//	A = slot 22 manche 0 (100,200,300) puis slot 20 manche 1 (60,160,260 decale de 300) -> 560
//	B = slot 20 manche 0 (40,80,120)   puis slot 22 manche 1 (50,150,250 decale de 120) -> 370
func TestScoreTimelineReassignedSlotAttributesToRoundOwner(t *testing.T) {
	recs, deaths := reassignFixture()
	tl, cov := buildScoreTimeline(&ScoreInput{Records: recs}, deaths, multiRoundClock())
	if tl == nil {
		t.Fatal("aucun calque publie")
	}
	if cov.Rounds != 2 {
		t.Fatalf("couverture : %d manche(s), attendu 2 (le chemin multi-manche n'est pas pris)", cov.Rounds)
	}
	if len(tl.Players) != 2 {
		t.Fatalf("%d joueur(s) publie(s), attendu 2 (A et B) : %+v", len(tl.Players), tl.Players)
	}
	a := playerByXUID(tl.Players, "1001")
	b := playerByXUID(tl.Players, "1002")
	if a == nil || b == nil {
		t.Fatalf("A (1001) ou B (1002) absent : %+v", tl.Players)
	}
	// Le score personnel : total fusionne dans l'ordre du temps.
	if got := lastValue(a.Score); got != 560 {
		t.Errorf("score personnel total de A = %d, attendu 560 (slot22 m0 + slot20 m1)", got)
	}
	if got := lastValue(b.Score); got != 370 {
		t.Errorf("score personnel total de B = %d, attendu 370 (slot20 m0 + slot22 m1)", got)
	}
	// La manche 1 de B (slot 22) lui est bien attribuee : sa valeur PROPRE a la manche est 250.
	if got := lastRoundValue(b.Score, 1); got != 250 {
		t.Errorf("score perso de B en manche 1 = %d, attendu 250 (slot 22, valeur propre a la manche)", got)
	}
	// La manche 1 de A vient du slot 20 (valeur propre 260), pas du slot 22.
	if got := lastRoundValue(a.Score, 1); got != 260 {
		t.Errorf("score perso de A en manche 1 = %d, attendu 260 (slot 20, valeur propre a la manche)", got)
	}
	// Frags : A = 3 (m0 slot22) + 3 (m1 slot20) = 6 ; B = 3 + 3 = 6.
	if got := lastValue(a.Kills); got != 6 {
		t.Errorf("frags total de A = %d, attendu 6", got)
	}
}

// TestScoreTimelineReassignedContreEpreuve — LE PONT PLAT SE TROMPE, ET ON LE PROUVE.
//
// Sur le MEME film, l'identite plate (par les morts vues sur tout le match) ne voit que la manche
// 0 : slot 22 = A, slot 20 = B pour TOUT le match. Elle donne donc la courbe ENTIERE du slot 22 a
// A (560 attendu -> 550 fautif : la manche 1 du slot 22 est celle de B) et celle du slot 20 a B
// (370 -> 380). Les totaux DIFFERENT du chemin par manche : c'est exactement le defaut corrige.
func TestScoreTimelineReassignedContreEpreuve(t *testing.T) {
	recs, deaths := reassignFixture()
	c := multiRoundClock()

	flatIdent := objectiveevents.SlotIdentityByDeaths(recs, deathInstantsOf(deaths))
	if flatIdent[22] != "1001" || flatIdent[20] != "1002" {
		t.Fatalf("pont plat = %v, attendu {22:1001(A), 20:1002(B)} (il ne voit que la manche 0)", flatIdent)
	}
	flat := buildPlayerScoresFlat(recs, flatIdent, c)
	fa := playerByXUID(flat, "1001")
	fb := playerByXUID(flat, "1002")
	if fa == nil || fb == nil {
		t.Fatalf("pont plat : A ou B absent : %+v", flat)
	}
	// Le pont plat mele la manche 1 du slot 22 (celle de B) dans le total de A -> 550, pas 560.
	if got := lastValue(fa.Score); got != 550 {
		t.Fatalf("contre-epreuve : total plat de A = %d, attendu 550 (le pont plat prend TOUT le slot 22)", got)
	}
	if got := lastValue(fb.Score); got != 380 {
		t.Fatalf("contre-epreuve : total plat de B = %d, attendu 380 (le pont plat prend TOUT le slot 20)", got)
	}

	// Le chemin par manche corrige : A = 560, B = 370. Il DIFFERE du pont plat.
	tl, _ := buildScoreTimeline(&ScoreInput{Records: recs}, deaths, c)
	pa := playerByXUID(tl.Players, "1001")
	pb := playerByXUID(tl.Players, "1002")
	if pa == nil || pb == nil {
		t.Fatalf("chemin par manche : A ou B absent : %+v", tl.Players)
	}
	if lastValue(pa.Score) == lastValue(fa.Score) || lastValue(pb.Score) == lastValue(fb.Score) {
		t.Errorf("le chemin par manche ne differe PAS du pont plat : la correction est nulle "+
			"(parmanche A=%d B=%d, plat A=%d B=%d)",
			lastValue(pa.Score), lastValue(pb.Score), lastValue(fa.Score), lastValue(fb.Score))
	}
}

// TestScoreTimelineXUIDFusionSingleEntry — la FUSION rend UNE entree par xuid, pas une par slot.
//
// A occupe le slot 22 (manche 0) et le slot 20 (manche 1) ; B l'inverse. Deux slots, deux joueurs,
// mais deux entrees seulement — chacune portant ses DEUX manches, courbe recomposee.
func TestScoreTimelineXUIDFusionSingleEntry(t *testing.T) {
	recs, deaths := reassignFixture()
	tl, _ := buildScoreTimeline(&ScoreInput{Records: recs}, deaths, multiRoundClock())

	seen := map[string]int{}
	for _, p := range tl.Players {
		seen[p.XUID]++
	}
	if len(seen) != 2 {
		t.Fatalf("%d xuid(s) distinct(s), attendu 2 : %v", len(seen), seen)
	}
	for xuid, n := range seen {
		if n != 1 {
			t.Errorf("le xuid %s apparait %d fois, attendu 1 (fusion des slots manquee)", xuid, n)
		}
	}
	// Chaque joueur porte bien SES DEUX manches (la courbe fusionne les deux slots).
	for _, p := range tl.Players {
		if len(p.Score.Rounds) != 2 {
			t.Errorf("%s : %d manche(s) de score perso, attendu 2 (fusion incomplete) : %+v",
				p.XUID, len(p.Score.Rounds), p.Score.Rounds)
		}
	}
}
