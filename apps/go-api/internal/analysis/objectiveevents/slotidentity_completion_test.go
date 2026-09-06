package objectiveevents

// slotidentity_completion_test.go — LE PONT PAR MORTS NE VOIT PAS LES JOUEURS QUI MEURENT PEU,
// ET LE TRIPLET LES RATTRAPE.
//
// # LE DEFAUT QUE CES TESTS FERMENT
//
// `deathInstantMin` = 3 : un slot n'est nomme par les instants de mort que s'il en aligne au
// moins trois. Un joueur qui meurt une ou deux fois est donc HORS DE PORTEE par construction —
// et ce sont les MEILLEURS joueurs, ceux qui portent le drapeau. Le 2026-08-28, `d173b1a8c` a
// bascule le calque des actions d'objectif du pont par TRIPLET vers le pont par MORTS en
// annoncant une « neutralite mono-manche prouvee par construction » : elle valait contre le pont
// PLAT PAR MORTS, mais ce calque-la n'etait pas sur ce pont — il etait sur le triplet. Mesure sur
// le film `c0a82e88` (Husky Raid:CTF, une manche) : 17 actions avant, 12 apres, et les DEUX
// SEULES actions de famille `flag` du match perdues avec le slot de leur auteur (7 frags,
// 2 morts). [RoundIdentity.CompletedByLines] rend la parite.

import (
	"sort"
	"testing"
)

// killRec est une emission synthetique du triplet complet (frags A, morts B) plus assistances.
func tripletRec(t, slot, round int, kills, deaths, assists int64) StatRecord {
	return StatRecord{TimeMS: t, Slot: slot, Round: round, Comps: map[int]StatValue{
		coreKillsComp:   {A: kills, B: deaths},
		coreAssistsComp: {A: assists},
	}}
}

// filmUneMancheDeuxMortsFixture : un film MONO-MANCHE ou
//
//	le slot 20 aligne 3 morts  -> le pont par morts le nomme (« D ») ;
//	le slot 22 n'en aligne que 2 -> il lui echappe, alors que son triplet (7,2,1) est UNIQUE.
//
// C'est la forme exacte du cas mesure sur `c0a82e88`.
func filmUneMancheDeuxMortsFixture() ([]StatRecord, []DeathInstant, []PlayerLine) {
	recs := []StatRecord{
		// Une seule manche reelle (train de score de mode croissant).
		modeRec(900, 22, 0, 10), modeRec(1900, 22, 0, 20), modeRec(2900, 22, 0, 30),
		// Slot 20 : trois morts -> a la portee du pont par morts. Triplet (1,3,0).
		tripletRec(1500, 20, 0, 1, 1, 0), tripletRec(2500, 20, 0, 1, 2, 0), tripletRec(3500, 20, 0, 1, 3, 0),
		// Slot 22 : DEUX morts seulement. Triplet (7,2,1), unique dans les lignes.
		tripletRec(1000, 22, 0, 7, 1, 1), tripletRec(2000, 22, 0, 7, 2, 1),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []DeathInstant{
		{XUID: "D", TimeMS: 1500}, {XUID: "D", TimeMS: 2500}, {XUID: "D", TimeMS: 3500},
		{XUID: "S", TimeMS: 1000}, {XUID: "S", TimeMS: 2000},
	}
	lines := []PlayerLine{
		{XUID: "D", Kills: 1, Deaths: 3, Assists: 0},
		{XUID: "S", Kills: 7, Deaths: 2, Assists: 1},
	}
	return recs, deaths, lines
}

// TestCompletedByLinesRattrapeLeJoueurQuiMeurtPeu — LE CAS QUI FONDE LE CORRECTIF.
func TestCompletedByLinesRattrapeLeJoueurQuiMeurtPeu(t *testing.T) {
	recs, deaths, lines := filmUneMancheDeuxMortsFixture()

	nu := ResolveRoundIdentity(recs, deaths)
	if got := nu.At(20, 1500); got != "D" {
		t.Fatalf("pont par morts, slot 20 : %q, attendu \"D\" (trois morts, a sa portee)", got)
	}
	if got := nu.At(22, 1000); got != "" {
		t.Fatalf("pont par morts, slot 22 : %q, attendu \"\" — deux morts, sous le seuil de %d",
			got, deathInstantMin)
	}

	complete := nu.CompletedByLines(recs, lines)
	if got := complete.At(22, 1000); got != "S" {
		t.Errorf("apres completion, slot 22 : %q, attendu \"S\" — son triplet (7,2,1) ne designe "+
			"qu'une ligne, et c'est le joueur dont les actions de drapeau etaient perdues", got)
	}
	if got := complete.At(20, 1500); got != "D" {
		t.Errorf("la completion a CHANGE un slot deja nomme par les morts : slot 20 = %q, attendu \"D\"", got)
	}
	if n := complete.NamedCount(); n != 2 {
		t.Errorf("slots nommes apres completion : %d, attendu 2", n)
	}
}

// TestCompletedByLinesSansLignesNeChangeRien : le calque reste publiable hors ligne.
func TestCompletedByLinesSansLignesNeChangeRien(t *testing.T) {
	recs, deaths, _ := filmUneMancheDeuxMortsFixture()
	nu := ResolveRoundIdentity(recs, deaths)
	for _, lignes := range [][]PlayerLine{nil, {}} {
		complete := nu.CompletedByLines(recs, lignes)
		if complete.NamedCount() != nu.NamedCount() || complete.At(22, 1000) != "" {
			t.Errorf("sans lignes de match, l'identite doit rester INCHANGEE (nommes %d -> %d, slot 22 = %q)",
				nu.NamedCount(), complete.NamedCount(), complete.At(22, 1000))
		}
	}
}

// TestCompletedByLinesRefuseLeMultiManche — LA GARDE QUI PROTEGE LA CORRECTION DE `d173b1a8c`.
// Le triplet apparie des TOTAUX DE MATCH ; en multi-manche le slot est reattribue et le compteur
// repart de zero. La completion doit se taire, sans quoi elle reintroduirait le defaut corrige.
func TestCompletedByLinesRefuseLeMultiManche(t *testing.T) {
	recs, deaths := twoRoundReassignedFixture()
	lines := []PlayerLine{
		{XUID: "A", Kills: 0, Deaths: 3, Assists: 0},
		{XUID: "B", Kills: 0, Deaths: 3, Assists: 0},
		{XUID: "C", Kills: 0, Deaths: 6, Assists: 0},
	}
	nu := ResolveRoundIdentity(recs, deaths)
	complete := nu.CompletedByLines(recs, lines)
	if complete.NamedCount() != nu.NamedCount() {
		t.Errorf("film MULTI-MANCHE : la completion par totaux doit se taire (nommes %d -> %d)",
			nu.NamedCount(), complete.NamedCount())
	}
	for _, r := range nu.Rounds() {
		for _, slot := range []int{20, 22} {
			if complete.AtRound(r, slot) != nu.AtRound(r, slot) {
				t.Errorf("manche %d, slot %d : la completion a modifie un film multi-manche (%q -> %q)",
					r, slot, nu.AtRound(r, slot), complete.AtRound(r, slot))
			}
		}
	}
}

// TestCompletedByLinesNAttribueJamaisUnXUIDDejaPris : la troisieme garde. Si le triplet designe
// pour un slot non nomme un xuid que le pont par morts a DEJA donne a un autre slot, la
// completion s'abstient — se taire vaut mieux que porter le meme joueur a deux endroits.
func TestCompletedByLinesNAttribueJamaisUnXUIDDejaPris(t *testing.T) {
	recs, deaths, _ := filmUneMancheDeuxMortsFixture()
	// Le triplet du slot 22 (7,2,1) est ici celui de "D" — deja nomme par les morts sur le slot 20.
	lines := []PlayerLine{{XUID: "D", Kills: 7, Deaths: 2, Assists: 1}}
	nu := ResolveRoundIdentity(recs, deaths)
	complete := nu.CompletedByLines(recs, lines)
	if got := complete.At(22, 1000); got != "" {
		t.Errorf("slot 22 = %q : « D » est deja porte par le slot 20, la completion devait se taire", got)
	}
}
