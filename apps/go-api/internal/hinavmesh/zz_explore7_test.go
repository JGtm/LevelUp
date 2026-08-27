package hinavmesh

import (
	"fmt"
	"sort"
	"testing"
)

// TestZZUserData decode les hkInplaceArray<hkInt32> userDataA / userDataB, seuls
// porteurs possibles d'un identifiant cote UserEdgePair.
func TestZZUserData(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()
		it, _ := f.tableau(rac, "userEdgePairs")
		pas := f.types.taille(it.Type)
		b, _ := f.octets(it, pas)
		valeurs := map[int32]int{}
		tailles := map[int]int{}
		for i := 0; i < it.Compte; i++ {
			o := i * pas
			for _, off := range []int{64, 96} {
				idx := int(u64(b, o+off))
				if idx <= 0 || idx >= len(f.items) {
					tailles[-1]++
					continue
				}
				e := f.items[idx]
				tailles[e.Compte]++
				br, err := f.octets(e, 4)
				if err != nil {
					continue
				}
				for k := 0; k < e.Compte; k++ {
					valeurs[int32(u32(br, 4*k))]++
				}
			}
		}
		cles := []int32{}
		for v := range valeurs {
			cles = append(cles, v)
		}
		sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })
		apercu := []string{}
		for _, c := range cles {
			if len(apercu) < 20 {
				apercu = append(apercu, fmt.Sprintf("%d(x%d)", c, valeurs[c]))
			}
		}
		t.Logf("%s : userDataA/B — tailles de tableau %v ; %d valeurs distinctes : %v",
			asset, tailles, len(cles), apercu)
	}
}

// TestZZCoherenceUp confronte les 4 mots du champ `up` de la region 3 aux comptes des
// autres regions, pour trancher entre "champ non initialise" et "champ detourne".
func TestZZCoherenceUp(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f3 := region3(t, asset)
		rac3, _ := f3.racine()
		m3, _ := f3.types.membre(rac3.Type, "up")
		b := f3.data[rac3.Offset+m3.Offset : rac3.Offset+m3.Offset+16]
		m := decodeTemoin(t, asset)
		t.Logf("%s : region3.up = u32{%d, %d, %d, %d} | navmesh: %d sommets, %d faces | maillage.Haut=%v",
			asset, u32(b, 0), u32(b, 4), u32(b, 8), u32(b, 12), len(m.Sommets), len(m.Faces), m.Haut)
		// Comptes des autres regions, pour comparaison.
		for _, r := range chargeRegionsTemoin(t, asset) {
			if len(r) < 8 || string(r[4:8]) != "TAG0" {
				continue
			}
			ff, err := lireFichierTag(r)
			if err != nil {
				continue
			}
			rr, _ := ff.racine()
			if ff.nomType(rr.Type) != "hkaiClusterGraph" {
				continue
			}
			for _, it := range ff.items {
				t.Logf("    clusterGraph item type=%-30s compte=%d", ff.nomType(it.Type), it.Compte)
			}
		}
	}
}
