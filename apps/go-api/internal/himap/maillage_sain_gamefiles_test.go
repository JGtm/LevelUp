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

// TestModelesFilairesDesObjetsForge — QUELS TYPES DESSINENT DES BRANCHES.
//
// Le 2026-08-27, isoler un seul type d objet a montre que le plus nombreux d Isolation
// (349 exemplaires) dessine de longues BRANCHES : ce sont elles, multipliees par des
// centaines, qui font le gribouillis. Un premier essai d exclusion avait echoue parce qu il
// classait les types par EMPRISE du modele — ce qui attrape les gros rochers et laisse passer
// les branches, longues mais minuscules en matiere.
//
// Le bon critere est la MINCEUR : l aire du maillage rapportee au carre de son emprise. Une
// branche de 8 m d envergure porte quelques dixiemes de metre carre ; un rocher de meme
// emprise en porte des dizaines. Ce test mesure la distribution pour choisir le seuil sur
// piece, et verifie que le type des branches y tombe bien dans le bas du classement.
func TestModelesFilairesDesObjetsForge(t *testing.T) {
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
		t.Skip("Isolation n est pas declaree dans CartesForge")
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
	opts := OptionsCuissonForge{
		RacineDeploy: racine, Objets: v.Objects,
		CheminModuleCanevas: CheminCanevasForge(carte), Cle: carte.MapID,
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	modeles := modeleParType(t.Context(), v.Objects, idx, forge)
	compte := map[int32]int{}
	for _, o := range v.Objects {
		compte[o.TypeID]++
	}

	type ligne struct {
		typeID  int32
		minceur float64
		n       int
	}
	var l []ligne
	for typeID, m := range modeles {
		a := ouvreAsset(t.Context(), idx, m.id, m.groupe)
		if a == nil {
			continue
		}
		mn, ok := MinceurDuModele(a)
		if !ok {
			continue
		}
		l = append(l, ligne{typeID, mn, compte[typeID]})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].minceur < l[j].minceur })
	t.Logf("%d modeles mesures — minceur = part de l emprise au sol reellement couverte", len(l))
	for i, x := range l {
		if i >= 10 {
			break
		}
		t.Logf("  LES PLUS FILAIRES  type %12d  minceur %.4f  x%d exemplaires", x.typeID, x.minceur, x.n)
	}
	for i := len(l) - 3; i < len(l); i++ {
		if i >= 0 {
			t.Logf("  LES PLUS PLEINS    type %12d  minceur %.4f  x%d exemplaires", l[i].typeID, l[i].minceur, l[i].n)
		}
	}
	for _, x := range l {
		if x.typeID == 2050581668 {
			rang := 0
			for _, y := range l {
				if y.minceur < x.minceur {
					rang++
				}
			}
			t.Logf("  LE TYPE DES BRANCHES (349 exemplaires) : minceur %.4f, rang %d sur %d", x.minceur, rang, len(l))
		}
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
