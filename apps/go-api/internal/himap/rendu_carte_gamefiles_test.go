package himap

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// LE RENDU SUR UNE CARTE QUELCONQUE — la regle transfere-t-elle ?
//
// `TestRenduCliffhanger` est adosse a `carte_validee_v1.png` : il cadre sur elle, et son juge
// est la comparaison pixel a pixel. UNE SEULE CARTE en possede une. Pour les 36 autres, il
// faut cadrer sans reference et juger sans reference — c'est le renversement du §1 ter :
// calibrer la regle une fois sur la carte qui a l'oracle fort, la VALIDER ailleurs avec
// l'oracle faible des ancres d'objectifs.
//
// Ici l'oracle faible demande : chaque ancre d'objectif a-t-elle du sol dessine sous elle ?
// Une ancre sans sol est un trou de reconstruction, pas une question de gout.
//
// Variables : RENDU_CARTE (nom de module au catalogue), RENDU_PNG_CARTE, RENDU_STYLE.

// margeCadre : marge autour des ancres, en metres. Meme portee que `PorteeAncre`, doublee
// pour que le cadre montre les abords de l'aire de jeu et pas seulement elle.
const margeCadre = 2 * PorteeAncre

func TestRenduCarte(t *testing.T) {
	carte := os.Getenv("RENDU_CARTE")
	if carte == "" {
		t.Skip("RENDU_CARTE non defini (nom de module au catalogue, ex. catalyst_map)")
	}
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	chemin, ok := chercheModule(racine, carte)
	if !ok {
		t.Skipf("aucun module installe pour %q", carte)
	}
	t.Logf("carte %q -> %s", carte, filepath.Base(chemin))

	var ancres [][3]float64
	for _, e := range ancresDuModule(t, carte) {
		for _, o := range e.Objectives {
			ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	if len(ancres) == 0 {
		t.Skipf("aucune ancre au catalogue pour %q — pas d'oracle, pas de cadre", carte)
	}

	rendu := cadreSurAncres(t, ancres)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
	rendu.NiveauDeJeu(medianeZ(ancres) - AncrageDecalageSol)
	n, ecartees := peupleRendu(t, rendu, racine, chemin, ancres)
	t.Logf("%d instances dessinees · %d ecartees comme decor", n, ecartees)

	jugeParLesAncres(t, rendu, ancres)
	poseEauDepuisSddt(t, rendu, chemin)

	if sortie := os.Getenv("RENDU_PNG_CARTE"); sortie != "" {
		style := styleRendu(os.Getenv("RENDU_STYLE"))
		if style == "" {
			style = StyleCarteParDefaut
		}
		ecritPNG(t, sortie, carteSeulePNG(rendu, rendu.NX, rendu.NY, nil, style))
		fmt.Println("carte ecrite:", sortie)
	}
}

// cadreSurAncres borne l'emprise au voisinage des ancres.
//
// Il ne s'agit PAS du tri refute au §10.8 du handoff — celui-la ecartait des instances, ce
// qui ne changeait rien. Ici c'est le CADRE de l'image : sans lui, l'emprise du sbsp reclame
// des grilles de plusieurs Go sur les grandes cartes.
func cadreSurAncres(t *testing.T, ancres [][3]float64) *Rendu {
	t.Helper()
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, a := range ancres {
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], a[k]-margeCadre)
			hi[k] = math.Max(hi[k], a[k]+margeCadre)
		}
	}
	r := NewRendu(lo, hi, gateEchelle)
	t.Logf("cadre [%.1f ; %.1f] x [%.1f ; %.1f] = %.0f x %.0f m -> %d x %d px a %.4f m/px",
		lo[0], hi[0], lo[1], hi[1], hi[0]-lo[0], hi[1]-lo[1], r.NX, r.NY, r.Cell)
	return r
}

// peupleRendu applique la chaine complete, avec les regles validees et aucun reglage par carte.
func peupleRendu(t *testing.T, rendu *Rendu, racine, chemin string, ancres [][3]float64) (int, int) {
	t.Helper()
	chemins, err := GeometrySearchPath(racine, chemin)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, err := ReadModuleInstances(chemin)
	if err != nil {
		t.Fatal(err)
	}
	bsp := choisitBSP(bsps, ancres)
	t.Logf("%d bsp dans le module · %d instances dans celui des ancres", len(bsps), len(bsp.Instances))

	assets := map[uint32]*RuntimeGeoAsset{}
	n, decor := 0, 0
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != "rtgo" {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			continue
		}
		if EstDecorGrossier(m, in, AireMaxTriangleJouable) {
			decor++
			continue
		}
		n++
		rendu.AddMeshBorne(m, in, 0.5)
	}
	return n, decor
}

// poseEauDepuisSddt pose les volumes d'eau du tag sddt (variante any/) sur un rendu.
//
// Une carte sans module any/, sans tag sddt ou sans volume d'eau est SIGNALEE au journal du
// test, jamais tue — piege paye par ce chantier : un oracle absent qui passe au vert.
func poseEauDepuisSddt(t *testing.T, r *Rendu, cheminModule string) {
	t.Helper()
	chemin, err := ModuleVariante(cheminModule, "any")
	if err != nil {
		t.Logf("eau : %v", err)
		return
	}
	s, err := LitSddt(chemin)
	if err != nil {
		t.Logf("eau : lecture sddt : %v", err)
		return
	}
	if len(s.VolumesEau) == 0 {
		t.Logf("eau : aucun volume d'eau dans le sddt de %s", filepath.Base(chemin))
		return
	}
	r.PoseEau(s.VolumesEau)
	n := 0
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			if r.Eau(i, j) {
				n++
			}
		}
	}
	t.Logf("eau : %d volumes sddt -> %d cellules d'eau posees", len(s.VolumesEau), n)
}

// jugeParLesAncres : chaque ancre, ramenee au sol, doit trouver de la matiere sous elle.
//
// C'est l'oracle FAIBLE — il ne dit pas que la carte est belle, il dit qu'elle n'est pas
// trouee la ou on joue. Le seul juge du rendu reste l'utilisateur.
func jugeParLesAncres(t *testing.T, r *Rendu, ancres [][3]float64) {
	t.Helper()
	avecSol, dansLeCadre := 0, 0
	var ecarts []float64
	for _, a := range ancres {
		i := int((a[0] - r.Min[0]) / r.Cell)
		j := int((a[1] - r.Min[1]) / r.Cell)
		if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
			continue
		}
		dansLeCadre++
		if z, ok := r.Altitude(i, j); ok {
			avecSol++
			ecarts = append(ecarts, a[2]-AncrageDecalageSol-z)
		}
	}
	if dansLeCadre == 0 {
		t.Fatal("aucune ancre dans le cadre — le cadrage est faux")
	}
	med := math.NaN()
	if len(ecarts) > 0 {
		med = centile(ecarts, 0.5)
	}
	t.Logf("ORACLE DES ANCRES : %d/%d ont du sol dessine sous elles (%.0f %%) · ecart median %+.2f m",
		avecSol, dansLeCadre, 100*float64(avecSol)/float64(dansLeCadre), med)
}
