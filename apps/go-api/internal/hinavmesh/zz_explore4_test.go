package hinavmesh

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func u64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }
func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func vec(b []byte, o int) [4]float64 {
	return [4]float64{flottant(b, o), flottant(b, o+4), flottant(b, o+8), flottant(b, o+12)}
}

// pointeurTableau lit le u64 d'un membre hkArray et rend l'indice d'item brut (sans
// verification de type) — pour voir les tableaux VIDES qui ne pointent nulle part.
func pointeurBrut(f *fichierTag, obj itemHavok, nom string) (int, int, int, string) {
	m, ok := f.types.membre(obj.Type, nom)
	if !ok {
		return -1, 0, 0, "membre absent"
	}
	idx := int(u64(f.data, obj.Offset+m.Offset))
	if idx <= 0 || idx >= len(f.items) {
		return idx, 0, 0, "pointeur nul / hors items"
	}
	it := f.items[idx]
	return idx, it.Offset, it.Compte, f.nomType(it.Type)
}

func TestZZRacineRegion3(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()
		t.Logf("=== RACINE region 3, %s ===", asset)
		mUp, _ := f.types.membre(rac.Type, "up")
		mIn, _ := f.types.membre(mUp.Type, "up")
		t.Logf("  up = %v", vec(f.data, rac.Offset+mUp.Offset+mIn.Offset))
		for _, nom := range []string{"userEdgePairs", "annotations", "vectorLibrary"} {
			i, off, n, ty := pointeurBrut(f, rac, nom)
			t.Logf("  %-24s -> item %d : off=%d compte=%d type=%s", nom, i, off, n, ty)
		}
		// intervalPartitionLibrary est une struct inline a +48 : ses deux hkArray sont dedans.
		mIPL, _ := f.types.membre(rac.Type, "intervalPartitionLibrary")
		base := rac.Offset + mIPL.Offset
		for _, nom := range []string{"data", "partitionRecords"} {
			m, _ := f.types.membre(mIPL.Type, nom)
			idx := int(u64(f.data, base+m.Offset))
			desc := "pointeur NUL (tableau vide)"
			if idx > 0 && idx < len(f.items) {
				it := f.items[idx]
				desc = fmt.Sprintf("item %d off=%d compte=%d type=%s", idx, it.Offset, it.Compte, f.nomType(it.Type))
			}
			t.Logf("  intervalPartitionLibrary.%-18s -> %s", nom, desc)
		}
	}
}

func TestZZContenuRegion3(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()

		itPairs, err := f.tableau(rac, "userEdgePairs")
		if err != nil {
			t.Fatalf("userEdgePairs: %v", err)
		}
		pasP := f.types.taille(itPairs.Type)
		bp, _ := f.octets(itPairs, pasP)

		itAnn, err := f.tableau(rac, "annotations")
		if err != nil {
			t.Fatalf("annotations: %v", err)
		}
		pasA := f.types.taille(itAnn.Type)
		ba, _ := f.octets(itAnn, pasA)

		t.Logf("=== CONTENU region 3, %s : %d UserEdgePair (pas %d), %d Annotation (pas %d) ===",
			asset, itPairs.Compte, pasP, itAnn.Compte, pasA)

		inf := math.Inf(1)
		mn := [3]float64{inf, inf, inf}
		mx := [3]float64{-inf, -inf, -inf}
		typesTrav := map[uint8]int{}
		dirs := map[uint8]int{}
		userdatas := map[uint64]int{}
		for i := 0; i < itPairs.Compte; i++ {
			o := i * pasP
			for _, v := range [][4]float64{vec(bp, o), vec(bp, o+16), vec(bp, o+32)} {
				for k := 0; k < 3; k++ {
					mn[k] = math.Min(mn[k], v[k])
					mx[k] = math.Max(mx[k], v[k])
				}
			}
			dirs[bp[o+132]]++
			if i < 6 {
				t.Logf("  pair %2d: x=%.2f y=%.2f z=%.2f | uidA=%d uidB=%d faceA=%d faceB=%d dir=%d cost=%04x/%04x",
					i, vec(bp, o), vec(bp, o+16), vec(bp, o+32),
					u32(bp, o+48), u32(bp, o+52), int32(u32(bp, o+56)), int32(u32(bp, o+60)),
					bp[o+132], binary.LittleEndian.Uint16(bp[o+128:]), binary.LittleEndian.Uint16(bp[o+130:]))
			}
		}
		for i := 0; i < itAnn.Compte; i++ {
			o := i * pasA
			typesTrav[ba[o+34]]++
			userdatas[u64(ba, o+16)]++
			if i < 6 {
				t.Logf("  ann  %2d: tEq=%.3f userdata=0x%016x firstPart=%d firstVec=%d nPart=%d nVec=%d trav=%d",
					i, vec(ba, o), u64(ba, o+16), u32(ba, o+24), u32(ba, o+28), ba[o+32], ba[o+33], ba[o+34])
			}
		}
		t.Logf("  EMPRISE des UserEdgePair (x,y,z des 3 vecteurs): X[%.2f;%.2f] Y[%.2f;%.2f] Z[%.2f;%.2f]",
			mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
		t.Logf("  traversalType observes: %v ; direction observees: %v", typesTrav, dirs)
		t.Logf("  userdata distincts: %d -> %v", len(userdatas), userdatas)
	}
}
