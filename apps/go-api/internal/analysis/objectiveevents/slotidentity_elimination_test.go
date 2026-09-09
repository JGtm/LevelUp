package objectiveevents

import "testing"

// Les quatre tests ci-dessous testent [RoundIdentity.CompletedByElimination] SEULE : une
// defense en profondeur (le score du rejeu) masquerait la mutation du composant qu'elle
// protege (regle §1 du plan v2).

// recKDA fabrique un enregistrement de statborg portant les trois compteurs de base.
func recKDA(timeMS, slot, round, k, d, a int) StatRecord {
	return StatRecord{
		TimeMS: timeMS, Slot: slot, Round: round,
		Comps: map[int]StatValue{
			coreKillsComp:   {A: int64(k), B: int64(d)},
			coreAssistsComp: {A: int64(a)},
			modeScoreComp:   {A: int64(k + d + a + 1)},
		},
	}
}

// deuxManchesTroisSlots : deux manches reelles, trois slots emetteurs par manche.
func deuxManchesTroisSlots() []StatRecord {
	var out []StatRecord
	t := 1000
	for _, r := range []int{0, 1} {
		for i, slot := range []int{10, 12, 14} {
			for n := 1; n <= 3; n++ {
				out = append(out, recKDA(t, slot, r, n+i, n, n))
				t += 100
			}
		}
	}
	return out
}

// identiteAvecTrous fabrique un resolveur multi-manche dont certains couples sont nommes.
func identiteAvecTrous(byRound map[int]map[int]string) RoundIdentity {
	origins := make(map[int]map[int]string, len(byRound))
	for r, m := range byRound {
		o := make(map[int]string, len(m))
		for slot := range m {
			o[slot] = OriginDeathInstants
		}
		origins[r] = o
	}
	return RoundIdentity{byRound: byRound, origins: origins}
}

// TestCompletionParEliminationNommeLeSlotUnique : trois slots emetteurs, deux nommes par les
// morts, le troisieme sans aucune mort — le troisieme xuid de la feuille lui est attribue.
//
// MUTATION : retirer la completion (ne pas appeler CompletedByElimination) -> le slot reste
// anonyme, le test rougit.
func TestCompletionParEliminationNommeLeSlotUnique(t *testing.T) {
	recs := deuxManchesTroisSlots()
	// Segments de la manche 0 : slot 10 = (3,3,3), slot 12 = (4,3,3), slot 14 = (5,3,3).
	// Segments de la manche 1 : identiques.
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 6},
		{XUID: "C", Kills: 10, Deaths: 6, Assists: 6},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B"},
		1: {10: "A", 12: "B", 14: "C"},
	})
	got := base.CompletedByElimination(recs, lines)
	if x := got.AtRound(0, 14); x != "C" {
		t.Fatalf("manche 0 slot 14 = %q, attendu \"C\" (elimination)", x)
	}
	if o := got.Origin(0, 14); o != OriginElimination {
		t.Fatalf("provenance = %q, attendu %q", o, OriginElimination)
	}
	if x := base.AtRound(0, 14); x != "" {
		t.Fatalf("l'identite d'origine a ete mutee : %q", x)
	}
}

// TestCompletionParEliminationSeTaitADeuxCandidats : deux slots non nommes et deux xuids
// libres — AUCUN n'est nomme.
//
// MUTATION : remplacer « exactement un » par « le premier » -> une identite est inventee, rouge.
func TestCompletionParEliminationSeTaitADeuxCandidats(t *testing.T) {
	recs := deuxManchesTroisSlots()
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 6},
		{XUID: "C", Kills: 10, Deaths: 6, Assists: 6},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A"},
		1: {10: "A", 12: "B", 14: "C"},
	})
	got := base.CompletedByElimination(recs, lines)
	if got.AtRound(0, 12) != "" || got.AtRound(0, 14) != "" {
		t.Fatalf("deux candidats ont ete nommes : 12=%q 14=%q",
			got.AtRound(0, 12), got.AtRound(0, 14))
	}
}

// TestCompletionParEliminationNeContreditJamaisLePontParMorts : un slot deja nomme garde son
// nom, meme si l'elimination en proposerait un autre.
//
// MUTATION : inverser l'ordre de fusion (ecrire avant de lire les deja-nommes) -> rouge.
func TestCompletionParEliminationNeContreditJamaisLePontParMorts(t *testing.T) {
	recs := deuxManchesTroisSlots()
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 6},
		{XUID: "C", Kills: 10, Deaths: 6, Assists: 6},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B", 14: "C"},
		1: {10: "A", 12: "B", 14: "C"},
	})
	got := base.CompletedByElimination(recs, lines)
	for slot, want := range map[int]string{10: "A", 12: "B", 14: "C"} {
		if x := got.AtRound(0, slot); x != want {
			t.Fatalf("slot %d = %q, attendu %q — l'elimination a contredit la lecture",
				slot, x, want)
		}
		if o := got.Origin(0, slot); o != OriginDeathInstants {
			t.Fatalf("slot %d : provenance = %q, la lecture a ete ecrasee", slot, o)
		}
	}
}

// TestCompletionParEliminationRefuseSurResiduDiscordant : le segment du slot candidat ne vaut
// pas le residu de la feuille — rien n'est nomme.
//
// MUTATION : retirer le controle du residu -> le slot est nomme a tort, rouge.
func TestCompletionParEliminationRefuseSurResiduDiscordant(t *testing.T) {
	recs := deuxManchesTroisSlots()
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 6},
		// Le total de C ne correspond a AUCUNE decomposition : son residu de manche 0 vaudrait
		// (42, 3, 3) quand le slot candidat porte (5, 3, 3).
		{XUID: "C", Kills: 47, Deaths: 6, Assists: 6},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B"},
		1: {10: "A", 12: "B", 14: "C"},
	})
	got := base.CompletedByElimination(recs, lines)
	if x := got.AtRound(0, 14); x != "" {
		t.Fatalf("slot nomme malgre un residu discordant : %q", x)
	}
}

// TestCompletionParEliminationSansFeuilleNeChangeRien : `lines` vide rend l'identite
// INCHANGEE — l'artefact reste publiable hors ligne.
func TestCompletionParEliminationSansFeuilleNeChangeRien(t *testing.T) {
	recs := deuxManchesTroisSlots()
	base := identiteAvecTrous(map[int]map[int]string{0: {10: "A"}, 1: {10: "A"}})
	got := base.CompletedByElimination(recs, nil)
	if got.NamedCount() != base.NamedCount() {
		t.Fatalf("NamedCount = %d, attendu %d", got.NamedCount(), base.NamedCount())
	}
}
