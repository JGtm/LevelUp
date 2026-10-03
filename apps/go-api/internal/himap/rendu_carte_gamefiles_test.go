//go:build gamefiles

package himap

import (
	"context"
	"fmt"
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
// CE TEST NE PORTE PLUS LA CHAINE : depuis le 2026-08-10 elle vit dans `cuisson.go` et produit
// de vrais assets (`cmd/mapfond-build`). Il l'APPELLE, et reste un outil de revue a la demande.
//
// Variables : RENDU_CARTE (nom de module au catalogue), RENDU_PNG_CARTE, RENDU_STYLE,
// RENDU_SANS_COQUILLE.
func TestRenduCarte(t *testing.T) {
	carte := os.Getenv("RENDU_CARTE")
	if carte == "" {
		t.Skip("RENDU_CARTE non defini (nom de module au catalogue, ex. catalyst_map)")
	}
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	chemin, ok := ChercheModuleInstalle(carte)
	if !ok {
		t.Skipf("aucun module installe pour %q", carte)
	}
	t.Logf("carte %q -> %s", carte, filepath.Base(chemin))

	ancres := ancresPlatesDuModule(t, carte)
	if len(ancres) == 0 {
		t.Skipf("aucune ancre au catalogue pour %q — pas d'oracle, pas de cadre", carte)
	}

	rendu, bilan, err := CuitCarteNative(context.Background(), OptionsCuisson{
		RacineDeploy:  racine,
		CheminModule:  chemin,
		Ancres:        ancres,
		SansFrontiere: os.Getenv("RENDU_SANS_COQUILLE") != "",
	})
	if err != nil {
		t.Fatal(err)
	}
	journaliseBilan(t, rendu, bilan)

	if sortie := os.Getenv("RENDU_PNG_CARTE"); sortie != "" {
		ecritPNG(t, sortie, FondPNG(rendu, rendu.NX, rendu.NY, nil, styleDemande()))
		fmt.Println("carte ecrite:", sortie)
	}
}

// styleDemande rend l'habillage demande par l'environnement, ou celui de production.
func styleDemande() StyleFond {
	if s := StyleFond(os.Getenv("RENDU_STYLE")); s != "" {
		return s
	}
	return StyleFondParDefaut
}

// journaliseBilan rend au journal du test ce que la cuisson a mesure.
func journaliseBilan(t *testing.T, r *Rendu, b BilanCuisson) {
	t.Helper()
	t.Logf("cadre %d x %d px a %.4f m/px · %d instances dessinees · %d ecartees comme decor",
		r.NX, r.NY, r.Cell, b.Dessinees, b.EcarteesDecor)
	t.Logf("ORACLE DES ANCRES : %d/%d ont du sol dessine sous elles · ecart median %+.2f m",
		b.AncresAvecSol, b.AncresDansLeCadre, b.EcartMedianAncre)
	if b.FrontiereAppliquee {
		t.Logf("frontiere : %d triangles · %d cellules hors frontiere effacees",
			b.PlansFrontiere, b.CellulesEffacees)
	}
	if b.VolumesEau > 0 {
		t.Logf("eau : %d volumes sddt -> %d cellules d'eau posees", b.VolumesEau, b.CellulesEau)
	}
	for _, d := range b.Degradations {
		t.Logf("DEGRADATION : %s", d)
	}
}

// ancresPlatesDuModule rend les positions monde des objectifs d'un module du catalogue.
func ancresPlatesDuModule(t *testing.T, module string) [][3]float64 {
	t.Helper()
	var ancres [][3]float64
	for _, e := range ancresDuModule(t, module) {
		for _, o := range e.Objectives {
			ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	return ancres
}

// LES ADAPTATEURS DE TEST — ils ne portent AUCUNE logique.
//
// Les sondes du package (cloture, balayage, physique) ont besoin des etapes de la chaine une
// par une, et d'un journal branche sur `*testing.T`. Ces trois fonctions n'existent que pour
// ca : elles appellent la production et journalisent. Toute logique qui apparaitrait ici serait
// une seconde chaine, donc une divergence en puissance.

func cadreSurAncres(t *testing.T, ancres [][3]float64) *Rendu {
	t.Helper()
	r := CadreSurAncres(ancres)
	t.Logf("cadre [%.1f ; %.1f] x [%.1f ; %.1f] -> %d x %d px a %.4f m/px",
		r.Min[0], r.Min[0]+float64(r.NX)*r.Cell, r.Min[1], r.Min[1]+float64(r.NY)*r.Cell,
		r.NX, r.NY, r.Cell)
	return r
}

func peupleRendu(t *testing.T, rendu *Rendu, racine, chemin string, ancres [][3]float64) (int, int) {
	t.Helper()
	var b BilanCuisson
	err := peupleDepuisModule(context.Background(), rendu, &b, OptionsCuisson{
		RacineDeploy: racine, CheminModule: chemin, Ancres: ancres,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d bsp dans le module · %d instances dans celui des ancres", b.BSPs, b.BSPInstances)
	return b.Dessinees, b.EcarteesDecor
}

func jugeParLesAncres(t *testing.T, r *Rendu, ancres [][3]float64) {
	t.Helper()
	var b BilanCuisson
	JugeParLesAncres(r, &b, ancres)
	if b.AncresDansLeCadre == 0 {
		t.Fatal("aucune ancre dans le cadre — le cadrage est faux")
	}
	t.Logf("ORACLE DES ANCRES : %d/%d ont du sol dessine sous elles (%.0f %%) · ecart median %+.2f m",
		b.AncresAvecSol, b.AncresDansLeCadre,
		100*float64(b.AncresAvecSol)/float64(b.AncresDansLeCadre), b.EcartMedianAncre)
}

func poseEauDepuisSddt(t *testing.T, r *Rendu, cheminModule string) {
	t.Helper()
	var b BilanCuisson
	PoseEauDepuisModule(context.Background(), r, &b, cheminModule)
	for _, d := range b.Degradations {
		t.Logf("eau : %s", d)
	}
	if b.VolumesEau > 0 {
		t.Logf("eau : %d volumes sddt -> %d cellules d'eau posees", b.VolumesEau, b.CellulesEau)
	}
}
