package himap

// LE CHOIX DU BSP DE DEQUANTIFICATION, MATERIALISE EN GARDE-RAIL ET EN PIECE JUSTIFICATIVE.
//
// Ce fichier fait deux choses a la fois, et c'est voulu : il PUBLIE l'inventaire des tags sbsp
// de toutes les cartes installees (c'est la mesure sur laquelle le critere a ete etabli le
// 2026-08-16) et il VERROUILLE le critere retenu. Le tableau sans le verrou se perime ; le
// verrou sans le tableau ne dit pas sur quoi il repose.
//
// CE QUE LA MESURE A MONTRE (29 modules porteurs de sbsp sur l'installation) :
//
//   - le critere « le plus gros tag » (`ReadModuleBSPBounds`[0]) et le critere « plus petit
//     AABB » DIVERGENT sur exactement SIX modules, tous des canevas Forge : `fo03_space`,
//     `fo05_desert`, `fo06_deepsea`, `fo10_deadland`, `fo11_blank`, `fo13_frost` ;
//   - sur ces six, le plus gros tag donne W [18 18 18] (etendue 3867 x 3662 x 2664) quand le
//     film lit [15 15 17] (etendue 463 x 453 x 1189, l'arene) ;
//   - `fo08_wetland` et `fo09_academy` portent LES MEMES DEUX AABB, a l'octet pres — leur
//     chaine tombait juste parce que leur arene pese plus d'octets que leur decor, pas parce
//     que le critere de taille disait quoi que ce soit de vrai ;
//   - l'ordre des regions lu dans le tag de niveau (`BSPQuantification`) est LISIBLE SUR LES
//     29, et il coincide avec le plus petit AABB sur les 29. C'est le critere retenu : il est
//     celui du moteur, le plus petit AABB n'en est que le controle croise.

import (
	"os"
	"path/filepath"
	"testing"
)

// canevasForgeCorriges : les modules dont le BSP de dequantification N'EST PAS le plus gros
// tag. Les lister nommement est ce qui fait de ce test un ratchet : si l'un d'eux redevenait
// « le plus gros tag », le defaut du 2026-08-16 serait revenu sans que rien ne le dise.
var canevasForgeCorriges = map[string]bool{
	"fo03_space": true, "fo05_desert": true, "fo06_deepsea": true,
	"fo10_deadland": true, "fo11_blank": true, "fo13_frost": true,
}

// largeursAreneForge : le decoupage que les films de TOUS les canevas Forge donnent, et que
// l'arene de ces canevas reproduit. Les six canevas partagent le meme gabarit d'arene.
var largeursAreneForge = [3]int{15, 15, 17}

// TestBSPQuantificationTousModules balaie l'installation entiere : le BSP de dequantification
// doit se lire sans ambiguite sur chaque module, et coincider avec son controle croise.
func TestBSPQuantificationTousModules(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	var avecSbsp, corriges int
	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		mods, _ := filepath.Glob(filepath.Join(dir, e.Name(), "*.module"))
		if len(mods) == 0 {
			continue
		}
		bsps, err := ReadModuleBSPBounds(mods[0])
		if err != nil {
			// Un module sans tag sbsp est une exception INSTRUITE (sgh_interlock,
			// academy_tutorial) : elle se declare, elle ne fait pas rougir.
			t.Logf("%-22s : %v", e.Name(), err)
			continue
		}
		avecSbsp++
		if verifieModule(t, e.Name(), mods[0], bsps) {
			corriges++
		}
	}
	t.Logf("%d modules porteurs de sbsp · %d dont le BSP de dequantification n'est PAS le plus gros tag",
		avecSbsp, corriges)
	if avecSbsp == 0 {
		t.Fatal("aucun module balaye — le garde-rail ne garderait rien")
	}
	if corriges != len(canevasForgeCorriges) {
		t.Errorf("%d modules corriges, %d attendus (%v)", corriges, len(canevasForgeCorriges),
			canevasForgeCorriges)
	}
}

// verifieModule joue les invariants sur un module et rend vrai si son BSP de dequantification
// differe du plus gros tag.
func verifieModule(t *testing.T, nom, chemin string, bsps []BSP) bool {
	t.Helper()
	q, _, err := BSPQuantification(chemin)
	if err != nil {
		t.Errorf("%s : BSP de dequantification illisible : %v", nom, err)
		return false
	}
	w := q.Bounds.AxisWidths()
	corrige := q.GlobalID != bsps[0].GlobalID
	t.Logf("%-22s : %d sbsp · plus gros tag -> W %2d/%2d/%2d · region 0 -> W %2d/%2d/%2d (gid %08x, etendue %.2f/%.2f/%.2f)",
		nom, len(bsps), bsps[0].Bounds.AxisWidths()[0], bsps[0].Bounds.AxisWidths()[1],
		bsps[0].Bounds.AxisWidths()[2], w[0], w[1], w[2], q.GlobalID,
		q.Bounds.Extent(0), q.Bounds.Extent(1), q.Bounds.Extent(2))

	// Controle croise INDEPENDANT de l'ordre des regions : le BSP de dequantification est
	// l'arene, donc le plus petit AABB. Les deux lectures doivent se confirmer.
	if petit := plusPetitAABB(bsps); petit.GlobalID != q.GlobalID {
		t.Errorf("%s : region 0 = gid %08x mais le plus petit AABB est gid %08x — les deux "+
			"lectures se contredisent", nom, q.GlobalID, petit.GlobalID)
	}
	if !q.Bounds.Valid() {
		t.Errorf("%s : AABB de la region 0 degeneree (%v)", nom, q.Bounds)
	}
	if corrige != canevasForgeCorriges[nom] {
		t.Errorf("%s : corrige=%v, attendu %v — la liste des canevas Forge corriges a bouge",
			nom, corrige, canevasForgeCorriges[nom])
	}
	if canevasForgeCorriges[nom] && [3]int{w[0], w[1], w[2]} != largeursAreneForge {
		t.Errorf("%s : region 0 -> W %v, attendu %v (gabarit d'arene Forge)", nom, w, largeursAreneForge)
	}
	return corrige
}

// plusPetitAABB rend le bsp de plus petit volume d'AABB — le controle croise de l'ordre des
// regions, lu sur une propriete tout autre (les bornes, pas le tag de niveau).
func plusPetitAABB(bsps []BSP) BSP {
	best := bsps[0]
	for _, b := range bsps[1:] {
		if volumeAABB(b.Bounds) < volumeAABB(best.Bounds) {
			best = b
		}
	}
	return best
}

func volumeAABB(b Bounds) float64 { return b.Extent(0) * b.Extent(1) * b.Extent(2) }

// TestBSPQuantificationCanevasForge publie l'inventaire DETAILLE des cinq canevas du
// perimetre : c'est la piece justificative de la phase 0 du plan, celle qui montre qu'il n'y
// a qu'UN candidat compatible par canevas et pourquoi le critere de taille se trompait.
func TestBSPQuantificationCanevasForge(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	canevas := []struct{ module, statut string }{
		{"fo08_wetland", "TEMOIN, controle film ACCORD 4/4"},
		{"fo09_academy", "TEMOIN, controle film ACCORD 3/3"},
		{"fo05_desert", "REFUTE 3/3"},
		{"fo11_blank", "REFUTE 7/7"},
		{"fo13_frost", "REFUTE 2/2"},
		{"fo03_space", "jamais controle (Starboard)"},
		{"fo06_deepsea", "jamais controle (Dredge)"},
	}
	for _, c := range canevas {
		chemin := filepath.Join(dir, c.module, c.module+"-rtx-new.module")
		if _, err := os.Stat(chemin); err != nil {
			t.Errorf("module absent (%s)", chemin)
			continue
		}
		bsps, err := ReadModuleBSPBounds(chemin)
		if err != nil {
			t.Errorf("%s : %v", c.module, err)
			continue
		}
		q, _, err := BSPQuantification(chemin)
		if err != nil {
			t.Errorf("%s : %v", c.module, err)
			continue
		}
		t.Logf("== %s (%s) ==", c.module, c.statut)
		compatibles := 0
		for i, b := range bsps {
			w := b.Bounds.AxisWidths()
			marque := " "
			if [3]int{w[0], w[1], w[2]} == largeursAreneForge {
				marque = "*"
				compatibles++
			}
			role := ""
			if b.GlobalID == q.GlobalID {
				role = "  <- REGION 0"
			}
			t.Logf("  %s [%d] file#%-3d gid=%08x taille=%-8d etendue %8.2f/%8.2f/%8.2f -> W %2d/%2d/%2d%s",
				marque, i, b.FileIndex, b.GlobalID, b.UncompSize,
				b.Bounds.Extent(0), b.Bounds.Extent(1), b.Bounds.Extent(2), w[0], w[1], w[2], role)
			t.Logf("      X[%10.3f %10.3f] Y[%10.3f %10.3f] Z[%10.3f %10.3f]",
				b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2])
		}
		if compatibles != 1 {
			t.Errorf("%s : %d candidat(s) compatible(s) avec le decoupage film %v, attendu 1",
				c.module, compatibles, largeursAreneForge)
		}
		if w := q.Bounds.AxisWidths(); [3]int{w[0], w[1], w[2]} != largeursAreneForge {
			t.Errorf("%s : la region 0 donne W %v, le film lit %v", c.module, w, largeursAreneForge)
		}
	}
	t.Logf("(* = largeurs deduites egales au decoupage film %v)", largeursAreneForge)

}
