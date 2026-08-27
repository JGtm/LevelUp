package hinavmesh

import (
	"math"
	"sort"
	"testing"
)

func TestZZItemsRegion1et4(t *testing.T) {
	for _, r := range chargeRegionsTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260") {
		if len(r) < 8 || string(r[4:8]) != "TAG0" {
			continue
		}
		f, err := lireFichierTag(r)
		if err != nil {
			continue
		}
		rac, _ := f.racine()
		t.Logf("=== %s ===", f.nomType(rac.Type))
		for i, it := range f.items {
			t.Logf("   item %2d type=%-42s off=%-8d compte=%d", i, f.nomType(it.Type), it.Offset, it.Compte)
			if i > 8 {
				t.Logf("   ... (%d items au total)", len(f.items))
				break
			}
		}
	}
}

// TestZZSemantiqueUserEdge caracterise ce que decrit un UserEdgePair : deux segments
// (un par face) et la distance qui les separe — la signature d'un lien de saut.
func TestZZSemantiqueUserEdge(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()
		it, _ := f.tableau(rac, "userEdgePairs")
		pas := f.types.taille(it.Type)
		b, _ := f.octets(it, pas)
		itA, _ := f.tableau(rac, "annotations")
		pasA := f.types.taille(itA.Type)
		ba, _ := f.octets(itA, pasA)

		pt := func(o, lane int) [3]float64 {
			return [3]float64{flottant(b, o+4*lane), flottant(b, o+16+4*lane), flottant(b, o+32+4*lane)}
		}
		dist := func(p, q [3]float64) float64 {
			return math.Sqrt((p[0]-q[0])*(p[0]-q[0]) + (p[1]-q[1])*(p[1]-q[1]) + (p[2]-q[2])*(p[2]-q[2]))
		}
		var ecarts, longA, longB, dz []float64
		parType := map[uint8][]float64{}
		for i := 0; i < it.Compte; i++ {
			o := i * pas
			a0, a1, b0, b1 := pt(o, 0), pt(o, 1), pt(o, 2), pt(o, 3)
			mA := [3]float64{(a0[0] + a1[0]) / 2, (a0[1] + a1[1]) / 2, (a0[2] + a1[2]) / 2}
			mB := [3]float64{(b0[0] + b1[0]) / 2, (b0[1] + b1[1]) / 2, (b0[2] + b1[2]) / 2}
			ecarts = append(ecarts, dist(mA, mB))
			longA = append(longA, dist(a0, a1))
			longB = append(longB, dist(b0, b1))
			dz = append(dz, mB[2]-mA[2])
			tt := ba[i*pasA+34]
			parType[tt] = append(parType[tt], mB[2]-mA[2])
		}
		med := func(v []float64) (float64, float64, float64) {
			s := append([]float64(nil), v...)
			sort.Float64s(s)
			return s[0], s[len(s)/2], s[len(s)-1]
		}
		n0, m0, x0 := med(ecarts)
		n1, m1, x1 := med(longA)
		n2, m2, x2 := med(longB)
		t.Logf("=== %s (%d liens) ===", asset, it.Compte)
		t.Logf("  distance entre les milieux des deux segments : min %.2f med %.2f max %.2f m", n0, m0, x0)
		t.Logf("  longueur du segment A : min %.2f med %.2f max %.2f m", n1, m1, x1)
		t.Logf("  longueur du segment B : min %.2f med %.2f max %.2f m", n2, m2, x2)
		for tt, v := range parType {
			a, b2, c := med(v)
			t.Logf("  traversalType=%d : %3d liens, denivele B-A min %.2f med %.2f max %.2f m", tt, len(v), a, b2, c)
		}
	}
}
