package himap

import (
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himodule"
)

// SONDE — MESURER LES QUATRE BSP DE `common-rtx-new` AVANT DE CHOISIR UNE REGLE.
//
// Le probleme, etabli le 2026-08-27 : `ChoisitBSP` retient le bsp qui contient le PLUS
// d'ancres, et sur `common-rtx-new.module` — le module qui porte la geometrie de Live Fire —
// DEUX des quatre bsp contiennent les 24 ancres. Le depart se fait alors au hasard de l'ordre
// de lecture, et l'ordre de lecture est la taille decompressee du tag : c'est le plus GROS qui
// gagne, donc le decor lointain.
//
// Cette sonde ne conclut pas : elle publie, pour chaque bsp, les trois grandeurs candidates a
// departager deux bsp a nombre d'ancres egal — l'emprise au sol, le nombre d'instances, et
// l'ecart vertical entre la mediane des ancres et la geometrie du bsp. C'est le tableau qu'elle
// imprime qui doit justifier la regle, pas l'inverse.
func TestSondeBSPCommunLiveFire(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ancres := ancresXYZ(t, "live_fire_sgh_interlock")
	if len(ancres) == 0 {
		t.Skip("aucune ancre pour live_fire_sgh_interlock")
	}
	zMed := MedianeZ(ancres)
	t.Logf("Live Fire — %d ancres, mediane z = %+.2f", len(ancres), zMed)

	for _, nom := range []string{"common-rtx-new.module", "multiplayer-rtx-new.module"} {
		chemin := filepath.Join(racine, "pc", "globals", nom)
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%s : lecture KO : %v", nom, err)
			continue
		}
		t.Logf("--- %s : %d bsp", nom, len(bsps))
		for i, b := range bsps {
			imprimeBSP(t, i, b, ancres, zMed)
		}
	}
}

// TestSondeBSPCartesNatives publie le meme tableau pour les cartes natives temoins : c'est lui
// qui dit si une regle nouvelle changerait leur choix.
func TestSondeBSPCartesNatives(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, module := range []string{"ctf_forbidden", "chasm_map", "catalyst_map", "cliffhanger_ridgeline", "streets_sgh_streets"} {
		chemin, ok := ChercheModuleInstalle(module)
		if !ok {
			t.Logf("%-24s module introuvable", module)
			continue
		}
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%-24s lecture KO : %v", module, err)
			continue
		}
		ancres := ancresXYZ(t, module)
		zMed := MedianeZ(ancres)
		t.Logf("--- %s : %d bsp, %d ancres, mediane z = %+.2f", module, len(bsps), len(ancres), zMed)
		for i, b := range bsps {
			imprimeBSP(t, i, b, ancres, zMed)
		}
	}
}

func imprimeBSP(t *testing.T, i int, b BSPInstances, ancres [][3]float64, zMed float64) {
	t.Helper()
	n := 0
	for _, a := range ancres {
		if contientPoint(b.Bounds, a) {
			n++
		}
	}
	empriseX := b.Bounds.Extent(0)
	empriseY := b.Bounds.Extent(1)
	zs := make([]float64, 0, len(b.Instances))
	for _, in := range b.Instances {
		zs = append(zs, in.Position[2])
	}
	sort.Float64s(zs)
	medZ, ecartVertical := math.NaN(), math.NaN()
	if len(zs) > 0 {
		medZ = zs[len(zs)/2]
		ecartVertical = math.Abs(medZ - zMed)
	}
	t.Logf("  bsp %d  ancres %2d/%2d  emprise %8.1f x %8.1f m = %11.0f m2  %6d inst  medianeZ %+9.1f  |dZ| %s  tag#%d %d o",
		i, n, len(ancres), empriseX, empriseY, empriseX*empriseY, len(b.Instances),
		medZ, fmtNaN(ecartVertical), b.FileIndex, b.UncompSize)
}

func fmtNaN(v float64) string {
	if math.IsNaN(v) {
		return "   n/a"
	}
	return fmt.Sprintf("%6.1f", v)
}

func contientPoint(b Bounds, p [3]float64) bool {
	return p[0] >= b.Min[0] && p[0] <= b.Max[0] &&
		p[1] >= b.Min[1] && p[1] <= b.Max[1] &&
		p[2] >= b.Min[2] && p[2] <= b.Max[2]
}

func ancresXYZ(t *testing.T, module string) [][3]float64 {
	t.Helper()
	var pts [][3]float64
	for _, e := range ancresDuModule(t, module) {
		for _, o := range e.Objectives {
			pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	return pts
}

// TestSondeBSPQuiEstChoisi imprime, pour chaque module a quatre bsp, l'index que la regle
// COURANTE retient — et l'emprise de l'ancrage, pour dire si le bsp retenu est l'arene ou
// l'horizon.
func TestSondeBSPQuiEstChoisi(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ancres := ancresXYZ(t, "live_fire_sgh_interlock")
	lo, hi := boiteDesPoints(ancres)
	t.Logf("ancres Live Fire : X [%+8.2f ; %+8.2f]  Y [%+8.2f ; %+8.2f]  Z [%+8.2f ; %+8.2f]  emprise %.1f x %.1f m",
		lo[0], hi[0], lo[1], hi[1], lo[2], hi[2], hi[0]-lo[0], hi[1]-lo[1])

	chemin := filepath.Join(racine, "pc", "globals", "common-rtx-new.module")
	bsps, err := ReadModuleInstances(chemin)
	if err != nil {
		t.Fatal(err)
	}
	choisi := ChoisitBSP(bsps, ancres)
	for i, b := range bsps {
		dedans := 0
		for _, in := range b.Instances {
			if in.Position[0] >= lo[0] && in.Position[0] <= hi[0] &&
				in.Position[1] >= lo[1] && in.Position[1] <= hi[1] {
				dedans++
			}
		}
		cx := (b.Bounds.Min[0] + b.Bounds.Max[0]) / 2
		cy := (b.Bounds.Min[1] + b.Bounds.Max[1]) / 2
		ax := (lo[0] + hi[0]) / 2
		ay := (lo[1] + hi[1]) / 2
		marque := " "
		if b.FileIndex == choisi.FileIndex {
			marque = "*"
		}
		t.Logf(" %s bsp %d tag#%-5d  instances dans la boite des ancres %5d / %5d (%5.1f%%)  centre a %6.1f m des ancres",
			marque, i, b.FileIndex, dedans, len(b.Instances),
			100*float64(dedans)/math.Max(1, float64(len(b.Instances))),
			math.Hypot(cx-ax, cy-ay))
	}
	t.Logf("ChoisitBSP retient tag#%d (%d instances, emprise %.0f x %.0f m)",
		choisi.FileIndex, len(choisi.Instances), choisi.Bounds.Extent(0), choisi.Bounds.Extent(1))
}

func boiteDesPoints(p [][3]float64) ([3]float64, [3]float64) {
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, a := range p {
		for k := 0; k < 3; k++ {
			lo[k] = math.Min(lo[k], a[k])
			hi[k] = math.Max(hi[k], a[k])
		}
	}
	return lo, hi
}

// TestSondeBSPAncresUnionLiveFire rejoue le jeu d'ancres EXACT de la production : mapfond-build
// regroupe les entrees du catalogue par dossier installe et DEDUPLIQUE par position, donc Live
// Fire est cuite avec l'union de sa variante libre et de sa variante classee.
func TestSondeBSPAncresUnionLiveFire(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	vues := map[[3]float64]bool{}
	var union [][3]float64
	for _, m := range []string{"live_fire_sgh_interlock", "live_fire_-_ranked_sgh_interlock"} {
		n := 0
		for _, a := range ancresXYZ(t, m) {
			n++
			if !vues[a] {
				vues[a] = true
				union = append(union, a)
			}
		}
		t.Logf("%-34s %2d ancres (union courante : %d)", m, n, len(union))
	}
	lo, hi := boiteDesPoints(union)
	t.Logf("union : %d ancres  X [%+8.2f ; %+8.2f]  Y [%+8.2f ; %+8.2f]  Z [%+8.2f ; %+8.2f]",
		len(union), lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])
	bsps, err := ReadModuleInstances(filepath.Join(racine, "pc", "globals", "common-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	zMed := MedianeZ(union)
	for i, b := range bsps {
		imprimeBSP(t, i, b, union, zMed)
	}
	choisi := ChoisitBSP(bsps, union)
	t.Logf("ChoisitBSP retient tag#%d (%d instances, emprise %.0f x %.0f m)",
		choisi.FileIndex, len(choisi.Instances), choisi.Bounds.Extent(0), choisi.Bounds.Extent(1))
}

// TestSondeLevlInterlockReferenceQuelBSP — LE CRITERE DU MOTEUR, APPLIQUE AU CAS LIVE FIRE.
//
// `sbsp_region.go` a deja etabli le principe : le tag de niveau (`levl`) reference ses tags
// sbsp par GlobalID, et c'est le moteur lui-meme qui l'ecrit. Le module de Live Fire n'a AUCUN
// sbsp mais porte un `levl` de 2,3 Mo : s'il reference l'un des quatre sbsp de
// `common-rtx-new`, la carte designe elle-meme sa geometrie et il n'y a plus rien a deviner.
func TestSondeLevlInterlockReferenceQuelBSP(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	commun := filepath.Join(racine, "pc", "globals", "common-rtx-new.module")
	bsps, err := ReadModuleBSPBounds(commun)
	if err != nil {
		t.Fatal(err)
	}
	gids := map[uint32]int{}
	for i, b := range bsps {
		gids[b.GlobalID] = i
		t.Logf("common bsp %d : GlobalID %08x  tag#%d  %d o  X %.0f Y %.0f Z %.0f",
			i, b.GlobalID, b.FileIndex, b.UncompSize,
			b.Bounds.Extent(0), b.Bounds.Extent(1), b.Bounds.Extent(2))
	}
	for _, module := range []string{"live_fire_sgh_interlock", "academy_tutorial", "recharge_sgh_blueprint"} {
		chemin, ok := ChercheModuleInstalle(module)
		if !ok {
			t.Logf("%-24s introuvable", module)
			continue
		}
		m, err := himodule.Open(chemin)
		if err != nil {
			t.Logf("%-24s ouverture KO : %v", module, err)
			continue
		}
		for _, f := range m.Files("levl") {
			tag, err := m.Extract(f)
			if err != nil {
				t.Logf("%-24s extraction levl KO : %v", module, err)
				continue
			}
			compte := map[uint32]int{}
			for p := 0; p+4 <= len(tag); p++ {
				g := binary.LittleEndian.Uint32(tag[p:])
				if _, ok := gids[g]; ok {
					compte[g]++
				}
			}
			var detail []string
			for g, i := range gids {
				detail = append(detail, fmt.Sprintf("bsp%d(%08x)=%d", i, g, compte[g]))
			}
			sort.Strings(detail)
			t.Logf("%-24s levl#%d %d o — occurrences : %v", module, f.Index, len(tag), detail)
		}
	}
}
