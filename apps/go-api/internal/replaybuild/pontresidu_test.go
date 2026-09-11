package replaybuild

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// pontresidu_test.go — LA QUATRIEME VOIE DU PONT DESCEND JUSQU'AU CALQUE.
//
// CE QUI EST EN JEU. Sur `43716616` (Oddball, deux manches), la manche 0 laisse QUATRE slots
// muets pour CINQ xuids libres : l'elimination, qui exige l'unicite, s'abstient. Le plus gros
// porteur du match (2533274978052136, 62,3 s a l'oracle API) tombait dans ce trou et le calque
// le montrait a 0 (rapport 6.2 §6, decouverte 1).
//
// CE TEST FIGE LE CABLAGE, pas la regle : la regle et ses quatre gardes sont testees a leur
// source (`objectiveevents/slotidentity_residue_test.go`).

// deuxManchesDeuxMuets — un film SYNTHETIQUE a DEUX manches ou :
//
//	manche 1  les quatre slots meurent au moins trois fois -> tous nommes par les morts ;
//	manche 0  les slots 14 et 16 meurent moins de trois fois -> DEUX muets, DEUX libres, donc
//	          l'elimination se tait ; leurs segments sont distincts, donc le residu tranche.
func deuxManchesDeuxMuets() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant,
	[]objectiveevents.PlayerLine) {
	tueMort := func(t, slot, round int, kills, deaths int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{2: {A: kills, B: deaths}}}
	}
	assist := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{3: {A: v}}}
	}
	mode := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}}}
	}
	var recs []objectiveevents.StatRecord
	var deaths []objectiveevents.DeathInstant
	slots := []int{10, 12, 14, 16}
	// Segments par manche : (frags, morts, assistances). Manche 0 : les slots 14 et 16 meurent
	// 1 et 2 fois, sous `deathInstantMin` = 3 — hors de portee du pont par morts.
	seg := map[int]map[int][3]int64{
		0: {10: {3, 3, 1}, 12: {4, 3, 2}, 14: {5, 1, 3}, 16: {6, 2, 4}},
		1: {10: {2, 3, 1}, 12: {2, 3, 1}, 14: {3, 3, 1}, 16: {4, 3, 1}},
	}
	// Les slots sont REATTRIBUES d'une manche a l'autre — c'est ce qui rend un pont plat faux.
	porteur := map[int]map[int]string{
		0: {10: "aaa", 12: "bbb", 14: "ccc", 16: "ddd"},
		1: {10: "ddd", 12: "ccc", 14: "bbb", 16: "aaa"},
	}
	// LES QUATRE SLOTS EMETTENT EN ALTERNANCE, comme sur un film reel : tous declarent la manche
	// dans la meme salve, sinon `ResolveRoundBounds` ecarte les emissions du slot le plus precoce.
	for _, round := range []int{0, 1} {
		base := 10000 + round*200000
		for pas := int64(1); pas <= 8; pas++ {
			for i, slot := range slots {
				s := seg[round][slot]
				t := base + int(pas)*4000 + i*500
				k, d := pas, pas
				if k > s[0] {
					k = s[0]
				}
				if d > s[1] {
					d = s[1]
				}
				if pas <= s[1] {
					deaths = append(deaths,
						objectiveevents.DeathInstant{XUID: porteur[round][slot], TimeMS: t})
				}
				recs = append(recs, tueMort(t, slot, round, k, d), assist(t, slot, round, min(pas, s[2])))
			}
		}
		// Une suite coherente du score de mode rend la manche REELLE.
		for n := int64(1); n <= 5; n++ {
			recs = append(recs, mode(base+int(n)*4000+200, 10, round, n))
		}
	}
	lines := []objectiveevents.PlayerLine{
		{XUID: "aaa", Kills: 3 + 4, Deaths: 3 + 3, Assists: 1 + 1},
		{XUID: "bbb", Kills: 4 + 3, Deaths: 3 + 3, Assists: 2 + 1},
		{XUID: "ccc", Kills: 5 + 2, Deaths: 1 + 3, Assists: 3 + 1},
		{XUID: "ddd", Kills: 6 + 2, Deaths: 2 + 3, Assists: 4 + 1},
	}
	return recs, deaths, lines
}

// TestPontParMancheNommeParResiduCeQueLEliminationLaisse — LE CŒUR DE L'ITEM 1.
//
// MUTATION : la meme chaine SANS la quatrieme voie laisse les deux slots anonymes. Le test la
// rejoue explicitement, si bien qu'il ne prouverait rien si les trois premieres voies savaient
// deja nommer ces slots-la.
func TestPontParMancheNommeParResiduCeQueLEliminationLaisse(t *testing.T) {
	recs, deaths, lines := deuxManchesDeuxMuets()
	if n := len(objectiveevents.RealRounds(recs)); n != 2 {
		t.Fatalf("le film synthetique porte %d manche(s) reelle(s), attendu 2", n)
	}

	// MUTATION : la chaine d'AVANT ce lot (trois voies) laisse les deux slots muets.
	avant := objectiveevents.ResolveRoundIdentity(recs, deaths).
		CompletedByLines(recs, lines).
		CompletedByElimination(recs, lines)
	if x := avant.AtRound(0, 14); x != "" {
		t.Fatalf("la chaine d'avant nomme deja le slot 14 (%q) : ce test ne prouve rien", x)
	}
	if x := avant.AtRound(0, 16); x != "" {
		t.Fatalf("la chaine d'avant nomme deja le slot 16 (%q) : ce test ne prouve rien", x)
	}

	pont := &pontParManche{recs: recs, deaths: deaths, lines: lines}
	got := skullInput(recs, true, pont).Identity
	if x := got.AtRound(0, 14); x != "ccc" {
		t.Errorf("manche 0 slot 14 = %q, attendu \"ccc\" (residu de manche)", x)
	}
	if x := got.AtRound(0, 16); x != "ddd" {
		t.Errorf("manche 0 slot 16 = %q, attendu \"ddd\" (residu de manche)", x)
	}
	if o := got.Origin(0, 14); o != objectiveevents.OriginRoundResidue {
		t.Errorf("provenance du slot 14 = %q, attendue %q", o, objectiveevents.OriginRoundResidue)
	}
	if x := got.AtRound(0, 10); x != "aaa" {
		t.Errorf("manche 0 slot 10 = %q, attendu \"aaa\" : le residu COMPLETE, il ne contredit pas", x)
	}

	// Sans lignes de match, la quatrieme voie s'abstient comme les deux autres.
	sansLignes := skullInput(recs, true, &pontParManche{recs: recs, deaths: deaths}).Identity
	if x := sansLignes.AtRound(0, 14); x != "" {
		t.Errorf("sans lignes de match, slot 14 = %q, attendu vide", x)
	}
}
