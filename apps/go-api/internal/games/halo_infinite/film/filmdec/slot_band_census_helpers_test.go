package filmdec

import (
	"sort"
	"testing"
)

// slot_band_census_helpers_test.go — LA BANDE D'UN ARCHÉTYPE QUAND LE RELEVÉ EST DÉJÀ FAIT,
// POUR PLUSIEURS ARCHÉTYPES À LA FOIS.
//
// POURQUOI CE FICHIER EXISTE. La règle COMBLÉE (`worldObjectSlotBand`) et la règle OBSERVÉE
// (`observedSlotBand`) relèvent elles-mêmes les images-clés, pour UN archétype : chaque appel
// relit tout le film. Un instrument de recherche qui a besoin des bandes de DIX archétypes ne
// peut pas les appeler dix fois — il marche les images-clés une seule fois et en sort un relevé
// `ti -> slots vus`. Il lui reste alors une seule chose à faire : dire quels slots sont ceux des
// AUTRES archétypes. C'est cette convention d'exclusion, et elle seule, que ce fichier tient.
//
// CE QUE ÇA ÉVITE. Sans ce point unique, chaque instrument réécrit la boucle « tous les slots
// des ti != le mien » — la copie que `archlint.TestNoRewrittenSlotBand` refuse, parce qu'une
// convention d'exclusion recopiée diverge au premier correctif de la règle.
//
// LA RÈGLE ELLE-MÊME N'EST PAS ICI : elle reste dans `slotBandExcluding` (projectiles.go),
// à qui ce helper délègue. Ce fichier ne marche PAS les images-clés — il part d'un relevé.

// slotBandFromCensus rend la bande COMBLÉE de l'archétype `ti` à partir d'un relevé
// multi-archétypes des images-clés (`ti -> slots vus`) : les slots vus porter un AUTRE
// archétype forment l'exclusion, puis la règle canonique s'applique.
//
// Même résultat que `worldObjectSlotBand(dir, n, ti)` sur le même film, sans relire le film.
func slotBandFromCensus(seenByTI map[int]map[uint32]bool, ti int) map[uint32]bool {
	others := map[uint32]bool{}
	for o, slots := range seenByTI {
		if o == ti {
			continue
		}
		for s := range slots {
			others[s] = true
		}
	}
	return slotBandExcluding(seenByTI[ti], others)
}

// TestSlotBandFromCensus fixe les deux moitiés de la règle sur un relevé écrit à la main : le
// COMBLEMENT de la plage (le trou 11 entre 10 et 14 entre dans la bande) et l'EXCLUSION (12,
// vu porter ti=42, en sort — 14 aussi, bien qu'il soit vu pour ti=37, parce qu'il est partagé).
func TestSlotBandFromCensus(t *testing.T) {
	census := map[int]map[uint32]bool{
		37: {10: true, 14: true},
		42: {12: true, 14: true},
		36: {20: true},
	}
	cases := []struct {
		ti   int
		want []uint32
	}{
		{ti: 37, want: []uint32{10, 11, 13}},
		{ti: 42, want: []uint32{12, 13}},
		{ti: 36, want: []uint32{20}},
		{ti: 41, want: nil}, // archétype absent du relevé : bande vide, pas de panique
	}
	for _, c := range cases {
		got := slotBandKeys(slotBandFromCensus(census, c.ti))
		if len(got) != len(c.want) {
			t.Fatalf("ti=%d : bande %v attendue, %v obtenue", c.ti, c.want, got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("ti=%d : bande %v attendue, %v obtenue", c.ti, c.want, got)
			}
		}
	}
}

// slotBandKeys rend les slots d'une bande triés, pour une comparaison stable.
func slotBandKeys(band map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(band))
	for s := range band {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
