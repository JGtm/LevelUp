package grammar

import (
	"reflect"
	"testing"
)

// keyframe_datums_lis_test.go — LA TABLE DE DATUMS TRANCHE UNE COINCIDENCE COMME LA GRAMMAIRE, ET
// `ambigus` COMPTE DES SLOTS AMBIGUS (J10.5, GA1-4 de l audit du 2026-09-24).
//
// LE DEFAUT : la plus longue suite croissante remplacait la queue d un slot par le DERNIER
// candidat du meme slot — la coincidence la plus TARDIVE gagnait, la ou l election de la marche
// d image-cle ([kfCand.betterThan]) retient, a slot egal, la generation la plus basse puis le bit
// le plus bas. Et `ambigus` rendait le nombre de CANDIDATS ecartes, la ou ses lecteurs (le monde,
// `types.MovementStateStats.DatumAmbiguous`, les instruments) annoncent des SLOTS ambigus.

// TestPlusLongueSuiteCroissanteCoincidenceSelonLaGrammaire : a slot egal, le premier candidat en
// bit gagne ; a slot egal et generation differente, la generation basse gagne.
func TestPlusLongueSuiteCroissanteCoincidenceSelonLaGrammaire(t *testing.T) {
	cas := []struct {
		nom   string
		cands []candidatDeDatum
		veut  []int
	}{
		{"meme generation : le plus tot", []candidatDeDatum{
			{bit: 10, slot: 3, ti: 1, gen: 1}, {bit: 20, slot: 5, ti: 2, gen: 1},
			{bit: 30, slot: 5, ti: 9, gen: 1}, {bit: 40, slot: 7, ti: 4, gen: 1},
		}, []int{0, 1, 3}},
		{"generation basse d abord", []candidatDeDatum{
			{bit: 10, slot: 3, ti: 1, gen: 1}, {bit: 20, slot: 5, ti: 2, gen: 2},
			{bit: 30, slot: 5, ti: 9, gen: 1}, {bit: 40, slot: 7, ti: 4, gen: 1},
		}, []int{0, 2, 3}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := plusLongueSuiteCroissante(c.cands); !reflect.DeepEqual(got, c.veut) {
				t.Fatalf("suite retenue = %v, attendue %v", got, c.veut)
			}
		})
	}
}

// TestTableDesCandidatsAmbigusCompteLesSlots : `ambigus` compte les SLOTS dont une lecture a ete
// ecartee en disant AUTRE CHOSE que la table — un slot vu avec deux archetypes, ou un slot que la
// croissance a ecarte en entier. Deux lectures identiques d un meme slot ne sont pas une ambiguite.
func TestTableDesCandidatsAmbigusCompteLesSlots(t *testing.T) {
	cands := []candidatDeDatum{
		{bit: 10, slot: 5, ti: 1, gen: 1},
		{bit: 20, slot: 5, ti: 2, gen: 1}, // deux archetypes pour le slot 5 : ambigu
		{bit: 30, slot: 6, ti: 3, gen: 1},
		{bit: 40, slot: 6, ti: 3, gen: 1}, // meme lecture du slot 6 : pas une ambiguite
		{bit: 50, slot: 2, ti: 4, gen: 1}, // ecarte par la croissance, seul de son slot
	}
	table, ambigus := tableDesCandidats(cands)
	if veut := map[uint32]uint32{5: 1, 6: 3}; !reflect.DeepEqual(table, veut) {
		t.Errorf("table = %v, attendue %v (le slot 5 garde sa PREMIERE lecture)", table, veut)
	}
	if ambigus != 2 {
		t.Errorf("ambigus = %d, attendu 2 slots (5 : deux archetypes ; 2 : ecarte)", ambigus)
	}
}
