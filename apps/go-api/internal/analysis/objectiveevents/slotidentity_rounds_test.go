package objectiveevents

import (
	"reflect"
	"sort"
	"testing"
)

// slotidentity_rounds_test.go — L'IDENTITE PAR MANCHE, et les DEUX proprietes qui comptent :
//
//	film MONO-MANCHE  -> le resultat est EXACTEMENT celui du pont plat (neutralite garantie
//	                     pour les calques deja livres, couronne VIP et drapeau CTF) ;
//	film MULTI-MANCHE -> quand le SLOT est reattribue d'une manche a l'autre, le pont par manche
//	                     nomme le BON joueur de chaque manche, la ou le pont plat se trompe apres
//	                     la bascule (il ne voit que la premiere manche, le compteur de morts
//	                     repartant de zero).
//
// Les enregistrements sont SYNTHETIQUES : ils portent le compteur de morts (`comp 2 B`) que les
// deux ponts lisent et le score de mode (`comp 0 A`) que [RealRounds] exige pour reconnaitre une
// manche. Le test tourne en CI, sans film.

// roundBridgeRec est une emission synthetique (raccourci pour construire un [StatRecord]).
func modeRec(t, slot, round int, score int64) StatRecord {
	return StatRecord{TimeMS: t, Slot: slot, Round: round, Comps: map[int]StatValue{modeScoreComp: {A: score}}}
}

func deathRec(t, slot, round int, deaths int64) StatRecord {
	return StatRecord{TimeMS: t, Slot: slot, Round: round, Comps: map[int]StatValue{coreKillsComp: {B: deaths}}}
}

// twoRoundReassignedFixture fabrique un film a DEUX manches ou le slot 22 est REATTRIBUE :
// joueur "A" en manche 0, joueur "B" en manche 1. Le slot 20 reste le joueur "C" aux deux
// manches. Chaque manche porte un train de score de mode (>= 3 emissions) pour etre reconnue.
func twoRoundReassignedFixture() ([]StatRecord, []DeathInstant) {
	recs := []StatRecord{
		// Score de mode : trois emissions croissantes par manche => manches 0 et 1 reelles.
		modeRec(900, 22, 0, 10), modeRec(1900, 22, 0, 20), modeRec(2900, 22, 0, 30),
		modeRec(10900, 22, 1, 10), modeRec(11900, 22, 1, 20), modeRec(12900, 22, 1, 30),
		// Compteur de morts, RESET par manche : slot 22.
		deathRec(1000, 22, 0, 1), deathRec(2000, 22, 0, 2), deathRec(3000, 22, 0, 3),
		deathRec(11000, 22, 1, 1), deathRec(12000, 22, 1, 2), deathRec(13000, 22, 1, 3),
		// Slot 20, stable = C.
		deathRec(1500, 20, 0, 1), deathRec(2500, 20, 0, 2), deathRec(3500, 20, 0, 3),
		deathRec(11500, 20, 1, 1), deathRec(12500, 20, 1, 2), deathRec(13500, 20, 1, 3),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []DeathInstant{
		{XUID: "A", TimeMS: 1000}, {XUID: "A", TimeMS: 2000}, {XUID: "A", TimeMS: 3000},
		{XUID: "B", TimeMS: 11000}, {XUID: "B", TimeMS: 12000}, {XUID: "B", TimeMS: 13000},
		{XUID: "C", TimeMS: 1500}, {XUID: "C", TimeMS: 2500}, {XUID: "C", TimeMS: 3500},
		{XUID: "C", TimeMS: 11500}, {XUID: "C", TimeMS: 12500}, {XUID: "C", TimeMS: 13500},
	}
	return recs, deaths
}

// TestSlotIdentityByRoundReassignedSlot — LE CAS QUI FONDE LE LOT : le pont plat se trompe apres
// la bascule de manche, le pont par manche corrige.
func TestSlotIdentityByRoundReassignedSlot(t *testing.T) {
	recs, deaths := twoRoundReassignedFixture()

	flat := SlotIdentityByDeaths(recs, deaths)
	// Le pont plat ne voit que la manche 0 (le compteur repart de zero en manche 1).
	if flat[22] != "A" {
		t.Fatalf("pont plat slot 22 = %q, attendu \"A\" (il ne voit que la manche 0)", flat[22])
	}
	if flat[20] != "C" {
		t.Fatalf("pont plat slot 20 = %q, attendu \"C\"", flat[20])
	}

	byRound := SlotIdentityByRound(recs, deaths)
	if len(byRound) != 2 {
		t.Fatalf("identite par manche : %d manche(s), attendu 2 (%v)", len(byRound), byRound)
	}
	if byRound[0][22] != "A" || byRound[0][20] != "C" {
		t.Errorf("manche 0 = %v, attendu {22:A, 20:C}", byRound[0])
	}
	// LA DIFFERENCE PROUVEE : en manche 1 le slot 22 est B, pas A.
	if byRound[1][22] != "B" {
		t.Errorf("manche 1 slot 22 = %q, attendu \"B\" (slot reattribue)", byRound[1][22])
	}
	if byRound[1][20] != "C" {
		t.Errorf("manche 1 slot 20 = %q, attendu \"C\" (slot stable)", byRound[1][20])
	}
	if byRound[1][22] == flat[22] {
		t.Errorf("le pont par manche ne differe PAS du pont plat en manche 1 — la correction est nulle")
	}
}

// TestRoundIdentityResolveByTime — le resolveur place un evenement dans sa manche par son instant.
func TestRoundIdentityResolveByTime(t *testing.T) {
	recs, deaths := twoRoundReassignedFixture()
	ri := ResolveRoundIdentity(recs, deaths)

	if got := ri.At(22, 2000); got != "A" {
		t.Errorf("At(22, 2000 ms, manche 0) = %q, attendu \"A\"", got)
	}
	if got := ri.At(22, 12000); got != "B" {
		t.Errorf("At(22, 12000 ms, manche 1) = %q, attendu \"B\"", got)
	}
	if got := ri.AtRound(0, 22); got != "A" {
		t.Errorf("AtRound(0, 22) = %q, attendu \"A\"", got)
	}
	if got := ri.AtRound(1, 22); got != "B" {
		t.Errorf("AtRound(1, 22) = %q, attendu \"B\"", got)
	}
	if rounds := ri.Rounds(); !reflect.DeepEqual(rounds, []int{0, 1}) {
		t.Errorf("Rounds() = %v, attendu [0 1]", rounds)
	}
}

// TestSlotIdentityByRoundMonoRoundNeutral — NEUTRALITE : un film mono-manche rend EXACTEMENT le
// pont plat, sous la seule manche. C'est ce qui garantit que la couronne VIP et le drapeau CTF
// mono-manche ne bougent pas.
func TestSlotIdentityByRoundMonoRoundNeutral(t *testing.T) {
	recs := []StatRecord{
		modeRec(900, 22, 0, 10), modeRec(1900, 22, 0, 20), modeRec(2900, 22, 0, 30),
		deathRec(1000, 22, 0, 1), deathRec(2000, 22, 0, 2), deathRec(3000, 22, 0, 3),
		deathRec(1500, 20, 0, 1), deathRec(2500, 20, 0, 2), deathRec(3500, 20, 0, 3),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []DeathInstant{
		{XUID: "A", TimeMS: 1000}, {XUID: "A", TimeMS: 2000}, {XUID: "A", TimeMS: 3000},
		{XUID: "C", TimeMS: 1500}, {XUID: "C", TimeMS: 2500}, {XUID: "C", TimeMS: 3500},
	}

	flat := SlotIdentityByDeaths(recs, deaths)
	byRound := SlotIdentityByRound(recs, deaths)
	if len(byRound) != 1 {
		t.Fatalf("mono-manche : %d manche(s), attendu 1", len(byRound))
	}
	if !reflect.DeepEqual(byRound[0], flat) {
		t.Errorf("mono-manche : identite par manche %v != pont plat %v", byRound[0], flat)
	}

	// Le resolveur d'une manche ignore le temps : At rend l'unique table quel que soit l'instant.
	ri := ResolveRoundIdentity(recs, deaths)
	if ri.At(22, 999999) != flat[22] {
		t.Errorf("mono-manche : At ne rend pas le pont plat")
	}
}
