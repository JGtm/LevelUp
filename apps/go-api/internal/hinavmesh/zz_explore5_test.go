package hinavmesh

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// TestZZRacineBrute hexdumpe les 128 octets de la racine pour lever le doute sur `up`.
func TestZZRacineBrute(t *testing.T) {
	f := region3(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	rac, _ := f.racine()
	b := f.data[rac.Offset : rac.Offset+128]
	for o := 0; o < 128; o += 16 {
		t.Logf("  +%3d: % x   | f32 %.4f %.4f %.4f %.4f | u32 %d %d %d %d", o, b[o:o+16],
			flottant(b, o), flottant(b, o+4), flottant(b, o+8), flottant(b, o+12),
			u32(b, o), u32(b, o+4), u32(b, o+8), u32(b, o+12))
	}
	// PTCH : les offsets rapieces dans DATA (les pointeurs). Sert a savoir si +32 en est un.
	sections := map[string][2]int{}
	region := chargeRegionsTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")[2]
	_ = parcoursSections(region, 0, len(region), sections, 0)
	if s, ok := sections["PTCH"]; ok {
		t.Logf("  section PTCH: %d octets", s[1])
		p := s[0]
		for p+8 <= s[0]+s[1] {
			typ := binary.LittleEndian.Uint32(region[p:])
			n := int(binary.LittleEndian.Uint32(region[p+4:]))
			offs := []uint32{}
			for k := 0; k < n && p+8+4*k+4 <= s[0]+s[1]; k++ {
				offs = append(offs, binary.LittleEndian.Uint32(region[p+8+4*k:]))
			}
			t.Logf("    type %d (%s) : %d offsets %v", typ, f.nomType(int(typ)), n, offs)
			p += 8 + 4*n
		}
	}
}

// TestZZGeometrieRegion3 relit les UserEdgePair en tenant compte de la disposition SoA
// (x = les 4 abscisses, y = les 4 ordonnees, z = les 4 cotes) et compare l'emprise a
// celle du maillage de navigation de la meme carte.
func TestZZGeometrieRegion3(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		rac, _ := f.racine()
		it, _ := f.tableau(rac, "userEdgePairs")
		pas := f.types.taille(it.Type)
		b, _ := f.octets(it, pas)

		inf := math.Inf(1)
		mn := [3]float64{inf, inf, inf}
		mx := [3]float64{-inf, -inf, -inf}
		var hors int
		m := decodeTemoin(t, asset)
		for i := 0; i < it.Compte; i++ {
			o := i * pas
			for lane := 0; lane < 4; lane++ {
				p := [3]float64{flottant(b, o+4*lane), flottant(b, o+16+4*lane), flottant(b, o+32+4*lane)}
				for k := 0; k < 3; k++ {
					mn[k] = math.Min(mn[k], p[k])
					mx[k] = math.Max(mx[k], p[k])
				}
				if p[0] < m.Min.X-0.5 || p[0] > m.Max.X+0.5 || p[1] < m.Min.Y-0.5 || p[1] > m.Max.Y+0.5 ||
					p[2] < m.Min.Z-0.5 || p[2] > m.Max.Z+0.5 {
					hors++
				}
			}
			if i < 3 {
				var pts []string
				for lane := 0; lane < 4; lane++ {
					pts = append(pts, fmt.Sprintf("(%.2f, %.2f, %.2f)",
						flottant(b, o+4*lane), flottant(b, o+16+4*lane), flottant(b, o+32+4*lane)))
				}
				t.Logf("  pair %d, 4 points SoA : %v", i, pts)
			}
		}
		t.Logf("=== %s ===", asset)
		t.Logf("  UserEdgePair (%d x 4 points) : X[%.2f;%.2f] Y[%.2f;%.2f] Z[%.2f;%.2f]",
			it.Compte, mn[0], mx[0], mn[1], mx[1], mn[2], mx[2])
		t.Logf("  Maillage nav                 : X[%.2f;%.2f] Y[%.2f;%.2f] Z[%.2f;%.2f]",
			m.Min.X, m.Max.X, m.Min.Y, m.Max.Y, m.Min.Z, m.Max.Z)
		t.Logf("  points hors de l'emprise du maillage (tolerance 0,5 m) : %d / %d", hors, it.Compte*4)

		// faceA / faceB doivent etre des indices VALIDES dans les faces du maillage.
		var horsFaces int
		for i := 0; i < it.Compte; i++ {
			o := i * pas
			for _, fi := range []int32{int32(u32(b, o+56)), int32(u32(b, o+60))} {
				if fi < 0 || int(fi) >= len(m.Faces) {
					horsFaces++
				}
			}
		}
		t.Logf("  faceA/faceB hors des %d faces du maillage : %d / %d", len(m.Faces), horsFaces, it.Compte*2)
	}
}
