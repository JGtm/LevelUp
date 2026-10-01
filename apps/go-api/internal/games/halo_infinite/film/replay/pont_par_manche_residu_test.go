package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// pont_par_manche_residu_test.go — DEPLACE de `replaybuild/pontresidu_test.go` le 2026-09-28 (lot
// V1.4 du plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`) avec le code qu'il teste.
//
// LA QUATRIEME VOIE DU PONT DESCEND JUSQU'AU CALQUE.
//
// CE QUI EST EN JEU. Sur `43716616` (Oddball, deux manches), la manche 0 laisse QUATRE slots
// muets pour CINQ xuids libres : l'elimination, qui exige l'unicite, s'abstient. Le plus gros
// porteur du match (2533274978052136, 62,3 s a l'oracle API) tombait dans ce trou et le calque
// le montrait a 0 (rapport 6.2 §6, decouverte 1).
//
// CE TEST FIGE LE CABLAGE, pas la regle : la regle et ses quatre gardes sont testees a leur
// source (`objectives/slotidentity_residue_test.go`).

// deuxManchesDeuxMuets — un film SYNTHETIQUE a DEUX manches ou :
//
//	manche 1  les quatre slots meurent au moins trois fois -> tous nommes par les morts ;
//	manche 0  les slots 14 et 16 meurent moins de trois fois -> DEUX muets, DEUX libres, donc
//	          l'elimination se tait ; leurs segments sont distincts, donc le residu tranche.
func deuxManchesDeuxMuets() ([]types.StatRecord, []types.DeathInstant,
	[]types.PlayerLine) {
	tueMort := func(t, slot, round int, kills, deaths int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]types.StatValue{2: {A: kills, B: deaths}}}
	}
	assist := func(t, slot, round int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]types.StatValue{3: {A: v}}}
	}
	mode := func(t, slot, round int, v int64) types.StatRecord {
		return types.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]types.StatValue{0: {A: v}}}
	}
	var recs []types.StatRecord
	var deaths []types.DeathInstant
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
	// ET UNE SALVE A ZERO OUVRE CHAQUE MANCHE (lot J8.5, 2026-09-27, constat FO-3) : le pont par
	// instants de mort deroule desormais la serie PUBLIEE, que la borne de manche confronte au temps ;
	// le premier enregistrement du slot le plus precoce (le 10) precede le debut par consensus et en
	// est ecarte. Sans cette salve, c etait sa PREMIERE MORT de la manche 1 — le pont ne nommait plus
	// le slot 10, et le residu de `ddd` ne se calculait plus. Une emission a zero ne gagne aucune unite.
	for _, round := range []int{0, 1} {
		base := 10000 + round*200000
		for pas := int64(0); pas <= 8; pas++ {
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
				if pas >= 1 && pas <= s[1] {
					deaths = append(deaths,
						types.DeathInstant{XUID: porteur[round][slot], TimeMS: t})
				}
				recs = append(recs, tueMort(t, slot, round, k, d), assist(t, slot, round, min(pas, s[2])))
			}
		}
		// Une suite coherente du score de mode rend la manche REELLE.
		for n := int64(1); n <= 5; n++ {
			recs = append(recs, mode(base+int(n)*4000+200, 10, round, n))
		}
	}
	lines := []types.PlayerLine{
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
	if n := len(objectives.RealRounds(recs)); n != 2 {
		t.Fatalf("le film synthetique porte %d manche(s) reelle(s), attendu 2", n)
	}

	// MUTATION : la chaine d'AVANT ce lot (trois voies) laisse les deux slots muets.
	avant := objectives.ResolveRoundIdentity(recs, deaths, nil).
		CompletedByLines(recs, lines).
		CompletedByElimination(recs, lines)
	if x := avant.AtRound(0, 14); x != "" {
		t.Fatalf("la chaine d'avant nomme deja le slot 14 (%q) : ce test ne prouve rien", x)
	}
	if x := avant.AtRound(0, 16); x != "" {
		t.Fatalf("la chaine d'avant nomme deja le slot 16 (%q) : ce test ne prouve rien", x)
	}

	pont := NouveauPontParManche(recs, deaths, lines, nil)
	got := EntreeDuCrane(recs, true, pont).Identity
	if x := got.AtRound(0, 14); x != "ccc" {
		t.Errorf("manche 0 slot 14 = %q, attendu \"ccc\" (residu de manche)", x)
	}
	if x := got.AtRound(0, 16); x != "ddd" {
		t.Errorf("manche 0 slot 16 = %q, attendu \"ddd\" (residu de manche)", x)
	}
	if o := got.Origin(0, 14); o != objectives.OriginRoundResidue {
		t.Errorf("provenance du slot 14 = %q, attendue %q", o, objectives.OriginRoundResidue)
	}
	if x := got.AtRound(0, 10); x != "aaa" {
		t.Errorf("manche 0 slot 10 = %q, attendu \"aaa\" : le residu COMPLETE, il ne contredit pas", x)
	}

	// Sans lignes de match, la quatrieme voie s'abstient comme les deux autres.
	sansLignes := EntreeDuCrane(recs, true, NouveauPontParManche(recs, deaths, nil, nil)).Identity
	if x := sansLignes.AtRound(0, 14); x != "" {
		t.Errorf("sans lignes de match, slot 14 = %q, attendu vide", x)
	}
}
