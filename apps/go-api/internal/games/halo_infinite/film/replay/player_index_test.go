package replay

// player_index_test.go — LE SECOND MAILLON DU PONT.
//
// LE PONT EST FAIT DE DEUX LECTURES COMPOSEES, et rien d autre : le fil des morts nomme chaque
// vie par le xuid de sa victime, et les cinq bits qui precedent chaque xuid dans le film donnent
// son index de joueur. C est le SECOND maillon qui est teste ici — celui qui decide si un tir
// est publie, et qui n avait aucune couverture.
//
// CE QUE CES TESTS PROTEGENT, ET C EST UNE DOCTRINE AVANT D ETRE UN CODE :
//
//	on EXIGE la concordance, on ne prend pas une majorite  une majorite serait un vote, et ce
//	                                                       chantier a justement retire les votes ;
//	                                                       26 lectures sont 26 occurrences du MEME
//	                                                       fait ecrit dans le film, donc deux
//	                                                       lectures divergentes ne s arbitrent pas
//	on refuse une table NON INJECTIVE                      un index partage placerait les tirs de
//	                                                       deux joueurs sur la meme trace, sans que
//	                                                       rien ne le signale
//	on ne lit QUE les chunks de replication                le registre (chunk 0) et le chunk des
//	                                                       highlights rendent 0 pour tous les
//	                                                       xuids ; les inclure ecraserait la bonne
//	                                                       table par une table nulle

import "testing"

// TestRosterFromDeathsIsStableAndDeduplicated : le roster est une LECTURE, lui aussi.
//
// L ORDRE EST TRIE, ET C EST NECESSAIRE : c est le seul roster dont le rejeu dispose sans base
// de donnees, et il alimente une resolution qui doit etre reproductible d une execution a
// l autre. Un ordre d apparition dependrait du decodage du chunk highlight.
func TestRosterFromDeathsIsStableAndDeduplicated(t *testing.T) {
	deaths := []Death{
		{XUID: 300, TimeMS: 10}, {XUID: 100, TimeMS: 20}, {XUID: 300, TimeMS: 30},
		{XUID: 200, TimeMS: 40}, {XUID: 100, TimeMS: 50},
	}
	got := rosterFromDeaths(deaths)
	want := []uint64{100, 200, 300}
	if len(got) != len(want) {
		t.Fatalf("%d joueur(s), attendu %d : %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("roster[%d] = %d, attendu %d — l ordre doit etre trie, donc reproductible",
				i, got[i], want[i])
		}
	}
	if len(rosterFromDeaths(nil)) != 0 {
		t.Error("un fil des morts vide doit rendre un roster vide, pas une entree fantome")
	}
}

// TestInjectiveOrEmptyRefusesASharedIndex : DEUX JOUEURS NE PARTAGENT PAS UN INDEX.
//
// L injectivite n est pas une preference esthetique. Un index partage ferait poser les tirs de
// deux joueurs sur la meme trace, et rien a l ecran ne le dirait — exactement le mode d echec
// que ce chantier refuse (« je prefere rien afficher que quelque chose de completement faux »).
func TestInjectiveOrEmptyRefusesASharedIndex(t *testing.T) {
	bonne := PlayerIndexTable{ByXUID: map[uint64]int{10: 0, 20: 1, 30: 2}, Readings: 26}
	got, col := injectiveOrEmpty(bonne)
	if col != 0 || len(got.ByXUID) != 3 {
		t.Errorf("une table injective a ete alteree : %d collision(s), %d entree(s)",
			col, len(got.ByXUID))
	}

	partagee := PlayerIndexTable{ByXUID: map[uint64]int{10: 1, 20: 1, 30: 2}, Readings: 26,
		Disagreements: 0}
	vide, col := injectiveOrEmpty(partagee)
	if col != 1 {
		t.Errorf("%d collision(s) rapportee(s), attendu 1", col)
	}
	if len(vide.ByXUID) != 0 {
		t.Errorf("%d entree(s) conservee(s) sur une table non injective : la table doit etre "+
			"ECARTEE EN ENTIER, pas nettoyee — on ne sait pas laquelle des deux est fausse",
			len(vide.ByXUID))
	}
	if vide.Readings != partagee.Readings {
		t.Error("le compte de lectures est perdu avec la table : il documente POURQUOI on a " +
			"ecarte, et il doit survivre au rejet")
	}
}

// TestScanFilmPlayerIndicesRefusesWithoutARoster : sans roster, il n y a rien a resoudre.
func TestScanFilmPlayerIndicesRefusesWithoutARoster(t *testing.T) {
	if _, err := ScanFilmPlayerIndices(MiniFilmDir, nil); err == nil {
		t.Error("aucune erreur sur un roster vide : le resolveur balayerait le film pour rien")
	}
	if _, err := ScanFilmPlayerIndices("testdata/film-qui-n-existe-pas", []uint64{1}); err == nil {
		t.Error("aucune erreur sur un repertoire sans chunk")
	}
}

// TestScanFilmPlayerIndicesReadsTheFilm : LA TABLE SE LIT, SUR DU BINAIRE REEL.
//
// Elle etait autrefois CALCULEE par affectation de cout minimal sur les 8! permutations : la
// bonne table sortait, mais par un choix a marge etroite (32 contradictions contre 39 pour la
// deuxieme). Le film l ecrit ; ce test verifie qu on la lit toujours, et qu elle est injective.
func TestScanFilmPlayerIndicesReadsTheFilm(t *testing.T) {
	deaths, err := ScanFilmDeaths(MiniFilmDir)
	if err != nil {
		t.Fatalf("ScanFilmDeaths : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(MiniFilmDir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("ScanFilmPlayerIndices : %v", err)
	}
	if idx.Readings < bridgeMinReadings {
		t.Errorf("%d chunk(s) de replication ont livre une table, attendu au moins %d — avec un "+
			"seul, l EXIGENCE DE CONCORDANCE n est jamais exercee et le test ne verifie plus la "+
			"propriete qui remplace le vote", idx.Readings, bridgeMinReadings)
	}
	if idx.Disagreements != 0 {
		t.Errorf("%d identite(s) lue(s) de deux facons : la lecture est fausse, et une majorite "+
			"ne l arbitrerait pas", idx.Disagreements)
	}
	table, collisions := injectiveOrEmpty(idx)
	if collisions != 0 {
		t.Fatalf("%d collision(s) d index sur le film de reference", collisions)
	}
	if len(table.ByXUID) != 8 {
		t.Fatalf("%d identite(s) indexee(s), attendu 8 (les huit joueurs de l arene)",
			len(table.ByXUID))
	}
	vus := map[int]bool{}
	for x, pi := range table.ByXUID {
		if pi < 0 || pi > 7 {
			t.Errorf("xuid %d : index %d hors 0..7", x, pi)
		}
		vus[pi] = true
	}
	if len(vus) != 8 {
		t.Errorf("%d index distincts pour 8 joueurs : la table n est pas une bijection", len(vus))
	}
}
