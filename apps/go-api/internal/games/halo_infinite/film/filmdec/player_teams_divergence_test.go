package filmdec

// player_teams_divergence_test.go — UN INDEX DIVERGENT SE COMPTE ET NE SE PUBLIE PAS
// (revue de jalon M1, lentille L6 : ce que les tests ne couvrent pas).
//
// # LE TROU QUE CE FICHIER FERME, ET IL A ETE MESURE
//
// [publierEquipes] porte la garde qui donne son sens a tout le calque : « entre deux equipes pour
// un meme joueur, il n y a rien a choisir ». Mutation jouee le 2026-09-15 sur la base `34fa53da5`
// — le `continue` qui suit `rep.IndexDivergences++` SUPPRIME — et le seul test rouge du paquet
// etait `TestGrammarRevSuitLaGrammaire`, c est-a-dire l empreinte des sources : AUCUN test de
// comportement ne voyait qu un index divergent venait d etre publie.
//
// La raison est simple et elle ne se corrige pas en durcissant `player_teams_test.go` : les sept
// bobines par build rendent ZERO divergence (`verifierRapportEquipes` l asserte), donc la branche
// n y est jamais prise. Un invariant qu aucune donnee versionnee n exerce se tient par une
// fixture, pas par un corpus.
//
// # POURQUOI CES TESTS ASSERTENT UNE ABSENCE, ET JAMAIS UNE VALEUR
//
// Sans la garde, `out[idx]` prend la DERNIERE valeur d un parcours de map : l ordre d iteration
// de Go est volontairement non deterministe, et un test qui attendrait « 1 plutot que 0 » serait
// intermittent. Ce qui est deterministe, et qui est exactement l invariant, c est que l index
// divergent N EST PAS DANS LA TABLE — et que `Indices` ne le compte pas.

import "testing"

// indexDivergentTemoin : un index vu avec DEUX designateurs, et un index d accord pour temoin de
// controle. Les deux valeurs divergentes sont des camps reels (`0` et `1`) et non [TeamNone] :
// une divergence n a rien a voir avec « aucune equipe », et melanger les deux masquerait le
// compteur `NoTeam`.
func indexDivergentTemoin() map[int]map[int]int {
	return map[int]map[int]int{
		3: {0: 7, 1: 2}, // deux lectures, deux camps : le film se contredit sur ce joueur
		5: {1: 4},       // un seul camp, plusieurs lectures d accord : celui-la se publie
	}
}

// TestIndexDivergentNEstPasPublie — LE TEMOIN.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer le `continue` qui suit `rep.IndexDivergences++`
// dans [publierEquipes]. L index 3 entre alors dans la table avec l un de ses deux camps, et
// `Indices` passe de 1 a 2. Jouee et restauree par nom le 2026-09-15 (sorties collees au §5 du
// plan).
func TestIndexDivergentNEstPasPublie(t *testing.T) {
	var rep TeamScanReport
	out := publierEquipes(indexDivergentTemoin(), &rep)

	if _, publie := out[3]; publie {
		t.Errorf("l index 3 est publie (equipe %d) alors que deux lectures lui donnent deux camps "+
			"differents : entre deux equipes pour un meme joueur, il n y a rien a choisir", out[3])
	}
	if rep.IndexDivergences != 1 {
		t.Errorf("IndexDivergences = %d, attendu 1 : la divergence se COMPTE, elle ne se tait pas",
			rep.IndexDivergences)
	}
	if rep.Indices != 1 {
		t.Errorf("Indices = %d, attendu 1 : seul l index d accord est publie", rep.Indices)
	}
	if got, ok := out[5]; !ok || got != 1 {
		t.Errorf("index 5 -> (%d, %v), attendu (1, true) : un index dont toutes les lectures "+
			"s accordent se publie", got, ok)
	}
	if rep.NoTeam != 0 {
		t.Errorf("NoTeam = %d, attendu 0 : aucune lecture du temoin ne vaut TeamNone (%d)",
			rep.NoTeam, TeamNone)
	}
}

// TestTableEntierementDivergenteNeRendRien : le cas limite. Quand AUCUN index ne s accorde, la
// table publiee est nil et non une table vide — c est ce que `nil` veut dire pour l appelant
// (`replay/player_teams.go` teste `len(byIndex) == 0` pour ne pas projeter), et une table vide
// non nil passerait le meme test tout en pretendant qu une lecture a eu lieu.
func TestTableEntierementDivergenteNeRendRien(t *testing.T) {
	var rep TeamScanReport
	out := publierEquipes(map[int]map[int]int{3: {0: 1, 1: 1}}, &rep)

	if out != nil {
		t.Errorf("table = %v, attendue nil : le seul index lu diverge, il n y a rien a publier", out)
	}
	if rep.IndexDivergences != 1 || rep.Indices != 0 {
		t.Errorf("rapport = %d divergence(s) / %d index, attendu 1 / 0",
			rep.IndexDivergences, rep.Indices)
	}
}
