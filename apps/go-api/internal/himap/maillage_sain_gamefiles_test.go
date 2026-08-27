package himap

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// UN MAILLAGE MAL DECODE SE MESURE — hypothese de l'utilisateur, 2026-08-27 :
// « ce gribouillis vient peut-etre d'un parametre qu'on interprete alors qu'on ne devrait pas ».
//
// Le symptome visuel du gribouillis — de longs rubans courbes qui balaient la carte — est
// exactement celui d'un INDEX DE TRIANGLES mal lu : les faces relient alors des sommets
// eloignes au lieu de sommets voisins. Cela se mesure sans rien afficher : sur un maillage
// sain, l'arete mediane d'un triangle est PETITE devant la diagonale du modele (quelques
// pour cent) ; sur un maillage dont les indices sont mal decodes, elle en approche le tiers.
//
// Ce test ne conclut pas : il classe les modeles poses par Isolation selon ce rapport.
func TestMaillageSainDesObjetsForge(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	var carte CarteForge
	for _, c := range CartesForge {
		if c.Nom == "Isolation" {
			carte = c
		}
	}
	if carte.MapID == "" {
		t.Skip("Isolation n'est pas declaree dans CartesForge")
	}
	depot, err := cheminDepuisDepot(DepotVariantesCarte)
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(filepath.Join(depot, carte.FichierMvar))
	if err != nil {
		t.Skipf("variante absente : %v", err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Skipf("variante illisible : %v", err)
	}
	objets := v.Objects
	opts := OptionsCuissonForge{
		RacineDeploy: racine, Objets: objets,
		CheminModuleCanevas: CheminCanevasForge(carte), Cle: carte.MapID,
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	modeles := modeleParType(t.Context(), objets, idx, forge)

	type ligne struct {
		typeID  int32
		rapport float64
		tris    int
	}
	var l []ligne
	vus := map[uint32]bool{}
	for typeID, m := range modeles {
		if vus[m.id] {
			continue
		}
		vus[m.id] = true
		a := ouvreAsset(t.Context(), idx, m.id, m.groupe)
		if a == nil {
			continue
		}
		r, tris := rapportAreteDiagonale(a)
		if tris == 0 {
			continue
		}
		l = append(l, ligne{typeID, r, tris})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].rapport > l[j].rapport })
	t.Logf("%d modeles mesures — rapport arete mediane / diagonale du modele", len(l))
	for i, x := range l {
		if i >= 12 {
			break
		}
		t.Logf("  type %12d  rapport %.3f  %d triangles", x.typeID, x.rapport, x.tris)
	}
	if len(l) > 0 {
		med := l[len(l)/2]
		t.Logf("  mediane des modeles : rapport %.3f", med.rapport)
	}
}

// rapportAreteDiagonale rend le rapport entre l'arete MEDIANE des triangles et la diagonale de
// la boite du modele, plus le nombre de triangles mesures.
func rapportAreteDiagonale(a *RuntimeGeoAsset) (float64, int) {
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	var aretes []float64
	tris := 0
	for mi := 0; mi < a.MeshCount(); mi++ {
		m := a.Mesh(mi)
		if m == nil {
			continue
		}
		for _, v := range m.Vertices {
			for k := 0; k < 3; k++ {
				lo[k] = math.Min(lo[k], v[k])
				hi[k] = math.Max(hi[k], v[k])
			}
		}
		for _, tr := range m.Triangles {
			tris++
			if len(aretes) < 3000 {
				aretes = append(aretes, distance(m.Vertices[tr[0]], m.Vertices[tr[1]]))
			}
		}
	}
	if tris == 0 || len(aretes) == 0 {
		return 0, 0
	}
	diag := distance(lo, hi)
	if diag <= 0 {
		return 0, 0
	}
	sort.Float64s(aretes)
	return aretes[len(aretes)/2] / diag, tris
}

func distance(a, b [3]float64) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}
