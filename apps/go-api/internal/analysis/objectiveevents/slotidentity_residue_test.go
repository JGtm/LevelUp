package objectiveevents

import "testing"

// Ces tests testent [RoundIdentity.CompletedByRoundResidue] SEULE : une defense en profondeur
// (le calque du crane) masquerait la mutation du composant qu'elle protege.

// deuxManchesQuatreSlots : deux manches reelles, quatre slots emetteurs par manche, des
// segments (K,D,A) TOUS DISTINCTS — la matiere d'un appariement par residu.
func deuxManchesQuatreSlots() []StatRecord {
	var out []StatRecord
	t := 1000
	for _, r := range []int{0, 1} {
		for i, slot := range []int{10, 12, 14, 16} {
			for n := 1; n <= 3; n++ {
				out = append(out, recKDA(t, slot, r, n+i, n, n+2*i))
				t += 100
			}
		}
	}
	return out
}

// TestResiduNommeDeuxSlotsQueLEliminationNeTranchePas est LE cas de `43716616` : deux slots
// muets et deux xuids libres dans la meme manche — l'elimination s'abstient (elle exige
// l'unicite), le residu les nomme tous les deux.
//
// MUTATION : retirer l'appel a CompletedByRoundResidue -> les deux slots restent anonymes.
func TestResiduNommeDeuxSlotsQueLEliminationNeTranchePas(t *testing.T) {
	recs := deuxManchesQuatreSlots()
	// Segments par manche : slot 10 = (3,3,3), 12 = (4,3,5), 14 = (5,3,7), 16 = (6,3,9).
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 10},
		{XUID: "C", Kills: 10, Deaths: 6, Assists: 14},
		{XUID: "D", Kills: 12, Deaths: 6, Assists: 18},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B"},
		1: {10: "A", 12: "B", 14: "C", 16: "D"},
	})
	// Controle d'entree : l'elimination ne tranche PAS (deux muets, deux libres).
	if got := base.CompletedByElimination(recs, lines); got.AtRound(0, 14) != "" || got.AtRound(0, 16) != "" {
		t.Fatalf("l'elimination a tranche, le cas de test ne prouve rien : 14=%q 16=%q",
			got.AtRound(0, 14), got.AtRound(0, 16))
	}
	got := base.CompletedByRoundResidue(recs, lines)
	if x := got.AtRound(0, 14); x != "C" {
		t.Fatalf("manche 0 slot 14 = %q, attendu \"C\" (residu)", x)
	}
	if x := got.AtRound(0, 16); x != "D" {
		t.Fatalf("manche 0 slot 16 = %q, attendu \"D\" (residu)", x)
	}
	if o := got.Origin(0, 14); o != OriginRoundResidue {
		t.Fatalf("provenance = %q, attendu %q", o, OriginRoundResidue)
	}
	if x := base.AtRound(0, 14); x != "" {
		t.Fatalf("l'identite d'origine a ete mutee : %q", x)
	}
}

// TestResiduSeTaitSurUnResiduPartage : deux slots muets au MEME segment — rien ne dit lequel
// est lequel, aucun n'est nomme.
func TestResiduSeTaitSurUnResiduPartage(t *testing.T) {
	var recs []StatRecord
	t0 := 1000
	for _, r := range []int{0, 1} {
		for _, slot := range []int{10, 12, 14, 16} {
			for n := 1; n <= 3; n++ {
				recs = append(recs, recKDA(t0, slot, r, n, n, n))
				t0 += 100
			}
		}
	}
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "C", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "D", Kills: 6, Deaths: 6, Assists: 6},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B"},
		1: {10: "A", 12: "B", 14: "C", 16: "D"},
	})
	got := base.CompletedByRoundResidue(recs, lines)
	if x := got.AtRound(0, 14); x != "" {
		t.Fatalf("slot 14 nomme %q alors que deux residus sont identiques", x)
	}
	if x := got.AtRound(0, 16); x != "" {
		t.Fatalf("slot 16 nomme %q alors que deux residus sont identiques", x)
	}
}

// TestResiduRefuseLeSegmentNul : un slot qui n'a RIEN fait dans la manche et un joueur au
// residu nul s'apparieraient sur du vide — la regle refuse.
func TestResiduRefuseLeSegmentNul(t *testing.T) {
	var recs []StatRecord
	t0 := 1000
	for _, r := range []int{0, 1} {
		for _, slot := range []int{10, 12} {
			for n := 1; n <= 3; n++ {
				recs = append(recs, recKDA(t0, slot, r, n, n, n))
				t0 += 100
			}
		}
	}
	// slot 14 emet le compteur de base, mais a zero partout, dans les deux manches.
	recs = append(recs, recKDA(9000, 14, 0, 0, 0, 0), recKDA(9100, 14, 1, 0, 0, 0))
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "Z", Kills: 0, Deaths: 0, Assists: 0},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B"},
		1: {10: "A", 12: "B", 14: "Z"},
	})
	if x := base.CompletedByRoundResidue(recs, lines).AtRound(0, 14); x != "" {
		t.Fatalf("slot 14 nomme %q sur un segment nul", x)
	}
}

// TestResiduNeTouchePasUnFilmMonoManche : sur un film a UNE manche le residu vaut le total, et
// la voie du triplet ([RoundIdentity.CompletedByLines]) est deja celle qui repond. La garde
// mono-manche existe pour que ce lot ne change AUCUN film mono-manche du parc.
func TestResiduNeTouchePasUnFilmMonoManche(t *testing.T) {
	var recs []StatRecord
	t0 := 1000
	for _, slot := range []int{10, 12, 14} {
		for n := 1; n <= 3; n++ {
			recs = append(recs, recKDA(t0, slot, 0, n, n, n))
			t0 += 100
		}
	}
	lines := []PlayerLine{
		{XUID: "A", Kills: 3, Deaths: 3, Assists: 3},
		{XUID: "B", Kills: 3, Deaths: 3, Assists: 3},
		{XUID: "C", Kills: 3, Deaths: 3, Assists: 3},
	}
	base := identiteAvecTrous(map[int]map[int]string{0: {10: "A"}})
	got := base.CompletedByRoundResidue(recs, lines)
	if got.NamedCount() != base.NamedCount() {
		t.Fatalf("le residu a nomme %d slots sur un film mono-manche (attendu %d)",
			got.NamedCount(), base.NamedCount())
	}
}

// TestResiduSansLignesRendLIdentiteInchangee : hors ligne, sans feuille de match, le calque
// reste publiable — meme garde que les deux autres voies de completion.
func TestResiduSansLignesRendLIdentiteInchangee(t *testing.T) {
	recs := deuxManchesQuatreSlots()
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A"},
		1: {10: "A"},
	})
	if got := base.CompletedByRoundResidue(recs, nil); got.NamedCount() != base.NamedCount() {
		t.Fatalf("sans lignes, %d slots nommes (attendu %d)", got.NamedCount(), base.NamedCount())
	}
}

// TestResiduNeContreditJamais : un slot deja nomme garde son nom, meme si un autre residu
// concorde.
func TestResiduNeContreditJamais(t *testing.T) {
	recs := deuxManchesQuatreSlots()
	lines := []PlayerLine{
		{XUID: "A", Kills: 6, Deaths: 6, Assists: 6},
		{XUID: "B", Kills: 8, Deaths: 6, Assists: 10},
		{XUID: "C", Kills: 10, Deaths: 6, Assists: 14},
		{XUID: "D", Kills: 12, Deaths: 6, Assists: 18},
	}
	base := identiteAvecTrous(map[int]map[int]string{
		0: {10: "A", 12: "B", 14: "D"},
		1: {10: "A", 12: "B", 14: "C", 16: "D"},
	})
	got := base.CompletedByRoundResidue(recs, lines)
	if x := got.AtRound(0, 14); x != "D" {
		t.Fatalf("slot 14 de la manche 0 = %q, le nom d'origine a ete contredit", x)
	}
}
