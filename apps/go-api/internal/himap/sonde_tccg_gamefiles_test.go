// Sonde d'investigation (2026-08-27, voie de secours des fonds Forge) : que porte le tag
// `tccg` (terrain collision cell) du canevas Forge, et peut-il servir de plan de sol ?
//
// Non commitee comme garde-rail : c'est un instrument de mesure. Elle ne modifie rien de la
// chaine de cuisson. Accumulation STREAMING : le corpus fait 151 Mo, un nuage materialise
// serait une bombe RAM (lecon `reference_statrecords_corpus_sweep_ram_bomb`).
package himap

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// ancresIsolation : les 25 ancres d'objectif d'Isolation (map_objectives.json, asset
// 01af558d-53ab-4f05-ba68-92d805fc6260), dedoublonnees. Terrain JOUE par definition.
var ancresIsolation = [][3]float64{
	{-24.342, -53.705, 112.608}, {-25.379, -16.837, 115.131}, {-3.680, -22.297, 117.916},
	{-40.296, -23.659, 118.219}, {-25.028, -31.159, 114.392}, {-25.717, -10.789, 121.540},
	{-21.150, -22.947, 115.280}, {-29.600, -23.253, 115.280}, {-25.214, -26.607, 119.987},
	{-24.922, -35.563, 118.518}, {-25.481, -19.225, 115.178}, {-30.066, -23.254, 115.033},
	{-20.695, -22.948, 115.040}, {-25.589, -17.098, 121.550}, {-35.586, -23.517, 115.584},
	{-15.059, -22.703, 115.584}, {-25.213, -27.209, 118.517},
}

func moduleCanevas(t *testing.T, variante, canevas string) string {
	t.Helper()
	root, err := DeployRoot()
	if err != nil {
		t.Skipf("installation absente : %v", err)
	}
	dir := filepath.Join(root, variante, "levels", "multi", canevas)
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("dossier %s absent : %v", dir, err)
	}
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".module" {
			return filepath.Join(dir, e.Name())
		}
	}
	t.Skipf("aucun .module dans %s", dir)
	return ""
}

// TestSondeInventaireTccg : combien de tccg/scgt, dans quelle variante, de quelle taille.
func TestSondeInventaireTccg(t *testing.T) {
	for _, v := range []string{"any", "ds", "pc"} {
		chemin := moduleCanevas(t, v, "fo08_wetland")
		m, err := himodule.Open(chemin)
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		vus := 0
		for _, g := range []string{"tccg", "scgt", "sddt", "pfnd"} {
			fs := m.Files(g)
			if len(fs) == 0 {
				continue
			}
			vus++
			total, mx := 0, 0
			for _, f := range fs {
				total += f.UncompSize
				if f.UncompSize > mx {
					mx = f.UncompSize
				}
			}
			t.Logf("variante %-3s  %s : %5d tags, %10d o au total, plus gros %d o",
				v, g, len(fs), total, mx)
		}
		if vus == 0 {
			t.Logf("variante %-3s : AUCUN des quatre tags", v)
		}
	}
}

// nuage accumule une emprise, un histogramme d'altitude et la couverture des ancres SANS
// materialiser les points.
type nuage struct {
	n         int
	mn, mx    [3]float64
	hz        map[int]int
	couvertes []bool
}

func nouveauNuage() *nuage {
	return &nuage{
		mn:        [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)},
		mx:        [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)},
		hz:        map[int]int{},
		couvertes: make([]bool, len(ancresIsolation)),
	}
}

func (u *nuage) ajoute(p [3]float64) {
	u.n++
	for a := 0; a < 3; a++ {
		u.mn[a] = math.Min(u.mn[a], p[a])
		u.mx[a] = math.Max(u.mx[a], p[a])
	}
	u.hz[int(math.Floor(p[2]/10))*10]++
	for i, a := range ancresIsolation {
		if u.couvertes[i] {
			continue
		}
		dx, dy := p[0]-a[0], p[1]-a[1]
		if dx*dx+dy*dy <= 9 {
			u.couvertes[i] = true
		}
	}
}

func (u *nuage) rapporte(t *testing.T, nom string) {
	t.Helper()
	if u.n == 0 {
		t.Logf("%s : AUCUN vec3 plausible", nom)
		return
	}
	t.Logf("%s : %d vec3 ; emprise x[%.1f;%.1f] y[%.1f;%.1f] z[%.1f;%.1f]",
		nom, u.n, u.mn[0], u.mx[0], u.mn[1], u.mx[1], u.mn[2], u.mx[2])
	var cles []int
	for k := range u.hz {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	s := ""
	for _, k := range cles {
		s += fmt.Sprintf(" [%d;%d):%d", k, k+10, u.hz[k])
	}
	t.Logf("%s : altitudes par tranche de 10 m :%s", nom, s)
	c := 0
	for _, b := range u.couvertes {
		if b {
			c++
		}
	}
	t.Logf("%s : ancres d'Isolation couvertes en XY (rayon 3 m) : %d / %d",
		nom, c, len(ancresIsolation))
}

// balaieVec3 balaye le tag en f32 ALIGNES (pas 4) et pousse dans le nuage les triplets
// f32 dont chaque composante tient dans la boite monde d'un canevas Forge.
func balaieVec3(tag []byte, u *nuage) {
	// Pas de balayage a l octet : sur 151 Mo il fabrique 65 M de faux triplets et la
	// couverture des ancres devient vraie par saturation, donc sans valeur.
	for i := 0; i+12 <= len(tag); i += 4 {
		x, y, z := f32(tag, i), f32(tag, i+4), f32(tag, i+8)
		if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) {
			continue
		}
		if math.Abs(x) > 400 || math.Abs(y) > 400 || z < -400 || z > 400 {
			continue
		}
		if x == 0 && y == 0 && z == 0 {
			continue
		}
		u.ajoute([3]float64{x, y, z})
	}
}

// TestSondeGeometrieTccg : les tccg portent-ils des coordonnees MONDE, et couvrent-ils les
// ancres d'Isolation ? Un plan de sol exploitable doit repondre oui aux deux.
func TestSondeGeometrieTccg(t *testing.T) {
	chemin := moduleCanevas(t, "any", "fo08_wetland")
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fs := m.Files("tccg")
	if len(fs) == 0 {
		t.Skip("aucun tccg")
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].UncompSize > fs[j].UncompSize })

	u := nouveauNuage()
	decodes, echecs := 0, 0
	for _, f := range fs {
		tag, err := m.Extract(f)
		if err != nil {
			echecs++
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			echecs++
		} else {
			decodes++
			if decodes == 1 {
				dumpArbreTag(t, ti, f.Index, len(tag))
			}
		}
		balaieVec3(tag, u)
	}
	t.Logf("tccg : %d tags, %d decodes par la struct-table generique, %d echecs",
		len(fs), decodes, echecs)
	u.rapporte(t, "tccg fo08_wetland")
}

// dumpArbreTag journalise la table des blocs d'un tag : tailles et strides candidats.
func dumpArbreTag(t *testing.T, ti tagInfo, idx, taille int) {
	t.Helper()
	t.Logf("--- tccg#%d (%d o) : %d data-blocks, %d structs ---",
		idx, taille, ti.dataBlocks, ti.structs)
	for i := 0; i < ti.dataBlocks && i < 16; i++ {
		abs, size := ti.blockAbs(i)
		t.Logf("    bloc %2d : offset %7d, %8d o", i, abs, size)
	}
	for _, l := range liensBlocs(ti) {
		if l.owner < 0 {
			continue
		}
		_, size := ti.blockAbs(l.target)
		c := compteChamp(ti, l)
		stride := 0
		if c > 0 {
			stride = size / c
		}
		t.Logf("    lien bloc %d @0x%02x -> bloc %d : %d records, stride %d",
			l.owner, l.fieldOff, l.target, c, stride)
	}
}

// TestSondeCadreTccg : les deux petits blocs d'un tccg (40 o et 48 o) portent-ils le cadre
// monde de la cellule ? Si oui, les 1 024 cellules pavent le canevas et leur AABB dit a
// quelle ALTITUDE vit le terrain — donc s'il peut servir de plan de sol a Isolation.
func TestSondeCadreTccg(t *testing.T) {
	chemin := moduleCanevas(t, "any", "fo08_wetland")
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fs := m.Files("tccg")
	sort.Slice(fs, func(i, j int) bool { return fs[i].UncompSize > fs[j].UncompSize })

	// 1. dump brut des petits blocs sur les 3 plus gros tags
	for k := 0; k < 3 && k < len(fs); k++ {
		tag, err := m.Extract(fs[k])
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		t.Logf("--- tccg#%d (%d o) ---", fs[k].Index, len(tag))
		for b := 0; b < ti.dataBlocks; b++ {
			abs, size := ti.blockAbs(b)
			if size > 256 || size < 8 {
				continue
			}
			t.Logf("  bloc %d (%d o) en f32 : %s", b, size, motsF32(tag, abs, size))
			t.Logf("  bloc %d (%d o) en u32 : %s", b, size, motsU32(tag, abs, size))
		}
	}

	// 2. emprise agregee des cellules, lue au bloc de 48 o
	u := nouveauNuage()
	lus := 0
	for _, f := range fs {
		tag, err := m.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		for b := 0; b < ti.dataBlocks; b++ {
			abs, size := ti.blockAbs(b)
			if size != 48 {
				continue
			}
			// hypothese : 2 vec3 min/max en tete du record
			mn := [3]float64{f32(tag, abs), f32(tag, abs+4), f32(tag, abs+8)}
			mx := [3]float64{f32(tag, abs+12), f32(tag, abs+16), f32(tag, abs+20)}
			if plausibleMonde(mn) && plausibleMonde(mx) {
				u.ajoute(mn)
				u.ajoute(mx)
				lus++
			}
		}
	}
	t.Logf("cellules dont le bloc de 48 o se lit comme un AABB monde : %d / %d", lus, len(fs))
	u.rapporte(t, "AABB des cellules tccg")
}

func plausibleMonde(p [3]float64) bool {
	for _, v := range p {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1000 {
			return false
		}
	}
	return true
}

func motsF32(b []byte, abs, size int) string {
	s := ""
	for i := 0; i+4 <= size && i < 64; i += 4 {
		s += fmt.Sprintf("%.4g ", f32(b, abs+i))
	}
	return s
}

func motsU32(b []byte, abs, size int) string {
	s := ""
	for i := 0; i+4 <= size && i < 64; i += 4 {
		s += fmt.Sprintf("%d ", u32(b, abs+i))
	}
	return s
}

// TestSondeCharge Tccg : de quelle nature est le gros bloc opaque d'un tccg ?
func TestSondeChargeTccg(t *testing.T) {
	chemin := moduleCanevas(t, "any", "fo08_wetland")
	m, _ := himodule.Open(chemin)
	fs := m.Files("tccg")
	sort.Slice(fs, func(i, j int) bool { return fs[i].UncompSize > fs[j].UncompSize })
	tag, err := m.Extract(fs[0])
	if err != nil {
		t.Fatalf("%v", err)
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		t.Fatalf("%v", err)
	}
	abs, size := ti.blockAbs(0)
	t.Logf("bloc 0 : offset %d, %d o", abs, size)
	t.Logf("  128 premiers octets : % x", tag[abs:abs+128])
	// magies Havok / conteneurs connus
	for _, mag := range []string{"TAG0", "hkcd", "hknp", "hkpc", "SDKV", "\x57\xe0\xe0\x57"} {
		n := 0
		for i := abs; i+len(mag) <= abs+size; i++ {
			if string(tag[i:i+len(mag)]) == mag {
				n++
			}
		}
		if n > 0 {
			t.Logf("  magie %q x%d", mag, n)
		}
	}
	// f32 ALIGNES sur 4 (pas de balayage a l'octet : il fabrique du bruit)
	u := nouveauNuage()
	for i := abs; i+12 <= abs+size; i += 4 {
		x, y, z := f32(tag, i), f32(tag, i+4), f32(tag, i+8)
		p := [3]float64{x, y, z}
		if !plausibleMonde(p) || (x == 0 && y == 0 && z == 0) {
			continue
		}
		u.ajoute(p)
	}
	u.rapporte(t, "bloc 0 tccg, vec3 f32 ALIGNES")
	// profil d'octets
	var hist [256]int
	for _, c := range tag[abs : abs+size] {
		hist[c]++
	}
	t.Logf("  octets nuls %.1f %%, 0xBC %.1f %%",
		100*float64(hist[0])/float64(size), 100*float64(hist[0xBC])/float64(size))
}

// TestSondeConteneurHavok : quels tags de carte portent un tagfile Havok `TAG0` ? C'est le
// COUT PARTAGE : un seul lecteur TAG0 les ouvre tous (navmesh.blob compris).
func TestSondeConteneurHavok(t *testing.T) {
	cas := []struct{ variante, canevas string }{
		{"any", "fo08_wetland"},
		{"any", "ctf_aquarius"},
		{"ds", "ctf_aquarius"},
		{"pc", "ctf_aquarius"},
	}
	for _, c := range cas {
		chemin := moduleCanevas(t, c.variante, c.canevas)
		m, err := himodule.Open(chemin)
		if err != nil {
			continue
		}
		for _, g := range []string{"tccg", "scgt", "sddt", "pfnd", "sbsp"} {
			fs := m.Files(g)
			if len(fs) == 0 {
				continue
			}
			sort.Slice(fs, func(i, j int) bool { return fs[i].UncompSize > fs[j].UncompSize })
			tag, err := m.Extract(fs[0])
			if err != nil {
				continue
			}
			tag0, hknp := 0, 0
			for i := 0; i+4 <= len(tag); i++ {
				switch string(tag[i : i+4]) {
				case "TAG0":
					tag0++
				case "hknp":
					hknp++
				}
			}
			t.Logf("%s/%-13s %s : %4d tags ; plus gros %7d o -> TAG0 x%d, hknp x%d",
				c.variante, c.canevas, g, len(fs), fs[0].UncompSize, tag0, hknp)
		}
	}
}
