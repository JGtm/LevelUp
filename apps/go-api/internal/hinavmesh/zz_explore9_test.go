package hinavmesh

import "testing"

func TestZZMiroir(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()
		it, _ := f.tableau(rac, "userEdgePairs")
		pas := f.types.taille(it.Type)
		b, _ := f.octets(it, pas)
		type cle struct{ a, b int32 }
		vus := map[cle]int{}
		for i := 0; i < it.Compte; i++ {
			o := i * pas
			vus[cle{int32(u32(b, o+56)), int32(u32(b, o+60))}]++
		}
		var apparies, orphelins int
		for k := range vus {
			if _, ok := vus[cle{k.b, k.a}]; ok {
				apparies++
			} else {
				orphelins++
			}
		}
		t.Logf("%s : %d couples (faceA,faceB) distincts — %d ont leur miroir (faceB,faceA), %d orphelins",
			asset, len(vus), apparies, orphelins)
	}
}
