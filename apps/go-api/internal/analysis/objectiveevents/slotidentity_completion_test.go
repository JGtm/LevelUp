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

// deuxManchesTripletResoluFixture — LA FIXTURE QUE LE PREMIER TEST DE GARDE MULTI-MANCHE
// N'AVAIT PAS.
//
// # CE QU'ELLE REPARE (revue CTF-R2, constat 1)
//
// `twoRoundReassignedFixture` donne aux slots 20 et 22 le MEME triplet (0,6,0) : les deux
// revendiquent la meme ligne, la seconde passe de [SlotIdentityFrom] les ecarte, et le triplet
// rend une table VIDE. [RoundIdentity.CompletedByLines] sortait donc au garde PRECEDENT
// (`len(triplet) == 0`) et la garde mono-manche n'etait jamais atteinte : la retirer laissait
// toutes les suites vertes, alors que sur le film reel `9f57c612` (Assaut, 4 manches) elle
// DEPLACE une detonation vers le mauvais joueur.
//
// Celle-ci ajoute un slot 24 que le triplet SAIT resoudre — 9 frags, 2 morts, 4 assistances,
// totaux uniques dans les lignes — et que le pont par morts NE PEUT PAS nommer (2 morts, sous
// `deathInstantMin`). C'est exactement la configuration ou la garde mono-manche est le seul
// rempart.
func deuxManchesTripletResoluFixture() ([]StatRecord, []DeathInstant, []PlayerLine) {
	recs := []StatRecord{
		// Deux manches REELLES : une suite de score de mode croissante par manche.
		modeRec(900, 22, 0, 10), modeRec(1900, 22, 0, 20), modeRec(2900, 22, 0, 30),
		modeRec(10900, 22, 1, 10), modeRec(11900, 22, 1, 20), modeRec(12900, 22, 1, 30),
		// Slot 22 REATTRIBUE : "A" en manche 0, "B" en manche 1 (le compteur repart de zero).
		deathRec(1000, 22, 0, 1), deathRec(2000, 22, 0, 2), deathRec(3000, 22, 0, 3),
		deathRec(11000, 22, 1, 1), deathRec(12000, 22, 1, 2), deathRec(13000, 22, 1, 3),
		// Slot 20 stable = "C".
		deathRec(1500, 20, 0, 1), deathRec(2500, 20, 0, 2), deathRec(3500, 20, 0, 3),
		deathRec(11500, 20, 1, 1), deathRec(12500, 20, 1, 2), deathRec(13500, 20, 1, 3),
		// Slot 24 : DEUX morts en tout — hors de portee du pont par morts — mais un triplet
		// (9,2,4) que SlotIdentityFrom resout SANS AMBIGUITE. Les compteurs REPARTENT DE ZERO a
		// chaque manche, comme dans le film : cumulateRounds les additionne (5+4, 1+1, 2+2). Les compteurs REPARTENT DE ZERO
		// a chaque manche, comme dans le film : `cumulateRounds` les additionne (5+4, 1+1, 2+2).
		tripletRec(1100, 24, 0, 1, 0, 0), tripletRec(1300, 24, 0, 2, 1, 1),
		tripletRec(1500, 24, 0, 3, 1, 1), tripletRec(1700, 24, 0, 4, 1, 2),
		tripletRec(1900, 24, 0, 5, 1, 2), tripletRec(11100, 24, 1, 1, 0, 0),
		tripletRec(11300, 24, 1, 2, 1, 1), tripletRec(11500, 24, 1, 3, 1, 1),
		tripletRec(11700, 24, 1, 4, 1, 2),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []DeathInstant{
		{XUID: "A", TimeMS: 1000}, {XUID: "A", TimeMS: 2000}, {XUID: "A", TimeMS: 3000},
		{XUID: "B", TimeMS: 11000}, {XUID: "B", TimeMS: 12000}, {XUID: "B", TimeMS: 13000},
		{XUID: "C", TimeMS: 1500}, {XUID: "C", TimeMS: 2500}, {XUID: "C", TimeMS: 3500},
		{XUID: "C", TimeMS: 11500}, {XUID: "C", TimeMS: 12500}, {XUID: "C", TimeMS: 13500},
		{XUID: "E", TimeMS: 1300}, {XUID: "E", TimeMS: 11300},
	}
	lines := []PlayerLine{
		{XUID: "A", Kills: 0, Deaths: 6, Assists: 0},
		{XUID: "B", Kills: 0, Deaths: 6, Assists: 0},
		{XUID: "C", Kills: 0, Deaths: 6, Assists: 0},
		{XUID: "E", Kills: 9, Deaths: 2, Assists: 4},
	}
	return recs, deaths, lines
}

// TestCompletedByLinesRefuseLeMultiMancheQuandLeTripletAUneReponse — LA GARDE 1, TESTEE POUR DE BON.
//
// Le triplet a ici une reponse (slot 24 = "E") et le film a DEUX manches. La completion doit se
// taire — sinon elle reintroduirait le defaut que `d173b1a8c` a corrige, et elle EFFONDRERAIT au
// passage l'identite par manche sur une seule table (la boucle de fusion ne garde qu'une manche).
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer `|| len(ri.byRound) != 1` de [CompletedByLines].
func TestCompletedByLinesRefuseLeMultiMancheQuandLeTripletAUneReponse(t *testing.T) {
	recs, deaths, lines := deuxManchesTripletResoluFixture()

	// PRE-REQUIS DU TEST : sans lui, le test sortirait au garde precedent et ne prouverait rien.
	triplet := SlotIdentityFrom(recs, lines)
	if triplet[24] != "E" {
		t.Fatalf("la fixture ne prouve rien : le triplet rend %v, il doit nommer le slot 24 \"E\"", triplet)
	}
	nu := ResolveRoundIdentity(recs, deaths)
	if len(nu.Rounds()) < 2 {
		t.Fatalf("la fixture ne porte pas plusieurs manches : %v", nu.Rounds())
	}

	complete := nu.CompletedByLines(recs, lines)

	// 1. Le slot que le triplet nomme ne doit PAS entrer.
	for _, r := range nu.Rounds() {
		if got := complete.AtRound(r, 24); got != "" {
			t.Errorf("manche %d, slot 24 = %q : le triplet apparie des TOTAUX DE MATCH, il n'a "+
				"aucun sens sur un film multi-manche", r, got)
		}
	}
	// 2. L'identite PAR MANCHE doit survivre intacte — le slot 22 est reattribue.
	if len(complete.Rounds()) != len(nu.Rounds()) {
		t.Errorf("manches apres completion : %v, attendu %v — la completion a effondre "+
			"l'identite par manche sur une seule table", complete.Rounds(), nu.Rounds())
	}
	for _, r := range nu.Rounds() {
		for _, slot := range []int{20, 22} {
			if complete.AtRound(r, slot) != nu.AtRound(r, slot) {
				t.Errorf("manche %d, slot %d : %q -> %q, la completion a modifie un film multi-manche",
					r, slot, nu.AtRound(r, slot), complete.AtRound(r, slot))
			}
		}
	}
}

// contradictionFixture — un film MONO-MANCHE ou les deux ponts DESIGNENT LE MEME SLOT avec des
// joueurs DIFFERENTS, et ou le joueur du triplet n'est revendique par aucun autre slot.
//
// C'est le SEUL cas ou la garde 2 (« completer, jamais contredire ») tire toute seule : la
// garde 3 (« aucun xuid deux fois ») ne la double pas, puisque "Q" est libre.
func contradictionFixture() ([]StatRecord, []DeathInstant, []PlayerLine) {
	recs := []StatRecord{
		// Une seule manche reelle.
		modeRec(900, 20, 0, 10), modeRec(1900, 20, 0, 20), modeRec(2900, 20, 0, 30),
		// Slot 20 : trois morts (le pont par morts le nomme "P"), et un triplet (5,3,2) qui
		// designe "Q" — un AUTRE joueur, que personne d'autre ne revendique.
		tripletRec(1500, 20, 0, 1, 1, 1), tripletRec(2000, 20, 0, 3, 1, 1),
		tripletRec(2500, 20, 0, 4, 2, 2), tripletRec(3000, 20, 0, 5, 2, 2),
		tripletRec(3500, 20, 0, 5, 3, 2),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []DeathInstant{
		{XUID: "P", TimeMS: 1500}, {XUID: "P", TimeMS: 2500}, {XUID: "P", TimeMS: 3500},
	}
	lines := []PlayerLine{
		{XUID: "P", Kills: 9, Deaths: 9, Assists: 9}, // aucun slot ne porte ce triplet
		{XUID: "Q", Kills: 5, Deaths: 3, Assists: 2}, // celui du slot 20
	}
	return recs, deaths, lines
}

// TestCompletedByLinesNeContreditJamaisLePontParMorts — LA GARDE 2, ET ELLE N'AVAIT AUCUN TEST.
//
// Le pont par morts nomme le slot 20 "P" ; le triplet, sur le MEME slot, dit "Q", et "Q" est
// libre. Seule la clause `deja` empeche le triplet d'ECRASER le nom du pont par morts — une
// regression y deplacerait les actions d'un joueur vers un autre, sans aucun signal.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : remplacer `if _, deja := fusion[slot]; deja || pris[xuid]`
// par `if pris[xuid]`.
func TestCompletedByLinesNeContreditJamaisLePontParMorts(t *testing.T) {
	recs, deaths, lines := contradictionFixture()

	// PRE-REQUIS : les deux ponts doivent bien se contredire sur le slot 20, et "Q" etre libre.
	triplet := SlotIdentityFrom(recs, lines)
	if triplet[20] != "Q" {
		t.Fatalf("la fixture ne prouve rien : le triplet rend %v, il doit nommer le slot 20 \"Q\"", triplet)
	}
	nu := ResolveRoundIdentity(recs, deaths)
	if got := nu.At(20, 1500); got != "P" {
		t.Fatalf("la fixture ne prouve rien : le pont par morts rend %q pour le slot 20, attendu \"P\"", got)
	}

	complete := nu.CompletedByLines(recs, lines)
	if got := complete.At(20, 1500); got != "P" {
		t.Errorf("slot 20 = %q apres completion, attendu \"P\" — le pont par morts a la PRIORITE : "+
			"la completion ajoute, elle ne remplace jamais", got)
	}
	if n := complete.NamedCount(); n != 1 {
		t.Errorf("slots nommes = %d, attendu 1 — la completion ne doit rien ajouter ici", n)
	}
}
