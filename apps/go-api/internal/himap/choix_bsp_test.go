package himap

import (
	"math"
	"testing"
)

// LES TEMOINS DU DEPARTAGE — la table de `ChoisitBSP`, rejouee sans le jeu installe.
//
// Les boites sont celles MESUREES le 2026-08-27 (`TestSondeBSPCommunLiveFire`,
// `TestSondeBSPCartesNatives`), pas des valeurs inventees : c'est la configuration reelle qui
// doit rester tranchee du bon cote, et un temoin bati sur des chiffres arbitraires ne dirait
// rien du jour ou la regle changera.

// bspTemoin fabrique un bsp de bornes donnees et de n instances, toutes au centre de la boite.
// Seuls les bornes et le NOMBRE d'instances entrent dans la decision.
func bspTemoin(index int, min, max [3]float64, n int) BSPInstances {
	b := BSPInstances{FileIndex: index, Bounds: Bounds{Min: min, Max: max}}
	centre := [3]float64{(min[0] + max[0]) / 2, (min[1] + max[1]) / 2, (min[2] + max[2]) / 2}
	for i := 0; i < n; i++ {
		b.Instances = append(b.Instances, Instance{Index: i, Position: centre})
	}
	return b
}

// areneLiveFire / horizonCommun : les deux bsp de `common-rtx-new.module` qui contiennent, l'un
// comme l'autre, les 28 ancres d'objectif de Live Fire.
func areneLiveFire() BSPInstances {
	return bspTemoin(1429, [3]float64{-16.7, -10.1, -3.4}, [3]float64{46.5, 53.7, 19.6}, 12556)
}

func horizonCommun() BSPInstances {
	return bspTemoin(1285, [3]float64{-3922, -3854, -444}, [3]float64{3968, 3921, 445}, 4863)
}

// ancresLiveFire : quelques ancres reelles de la carte, dans les bornes mesurees
// X [-9,14 ; +25,20] Y [+21,28 ; +46,32] Z [-0,99 ; +3,70].
func ancresLiveFire() [][3]float64 {
	return [][3]float64{
		{-9.14, 21.28, -0.99},
		{0.00, 33.80, 1.40},
		{25.20, 46.32, 3.70},
	}
}

// TestChoisitBSPDepartageParEmpriseALEgaliteDAncres — LE CAS LIVE FIRE.
//
// Les deux bsp contiennent TOUTES les ancres. L'horizon est place EN TETE de la liste et
// l'arene en queue : si la regle dependait encore de l'ordre de lecture, ce test la prendrait
// en flagrant delit.
func TestChoisitBSPDepartageParEmpriseALEgaliteDAncres(t *testing.T) {
	ancres := ancresLiveFire()
	arene, horizon := areneLiveFire(), horizonCommun()
	if got := CompteAncresDansBoite(horizon.Bounds, ancres); got != len(ancres) {
		t.Fatalf("le temoin n'est pas le cas mesure : l'horizon contient %d ancres sur %d", got, len(ancres))
	}
	if got := CompteAncresDansBoite(arene.Bounds, ancres); got != len(ancres) {
		t.Fatalf("le temoin n'est pas le cas mesure : l'arene contient %d ancres sur %d", got, len(ancres))
	}
	for _, ordre := range [][]BSPInstances{{horizon, arene}, {arene, horizon}} {
		if got := ChoisitBSP(ordre, ancres); got.FileIndex != arene.FileIndex {
			t.Fatalf("bsp retenu tag#%d (emprise %.0f m2), attendu l'arene tag#%d (emprise %.0f m2)",
				got.FileIndex, EmpriseAuSol(got.Bounds), arene.FileIndex, EmpriseAuSol(arene.Bounds))
		}
	}
}

// TestChoisitBSPNeSuitPasLAltitude verrouille la REFUTATION mesuree : sur Live Fire, la
// geometrie de l'horizon est PLUS PROCHE du niveau de jeu que celle de l'arene (0,5 m contre
// 1,7 m de la mediane des ancres). Un departage par l'altitude designerait donc l'horizon.
// Ce temoin existe pour qu'on ne le reintroduise pas en croyant bien faire.
func TestChoisitBSPNeSuitPasLAltitude(t *testing.T) {
	ancres := ancresLiveFire()
	zJeu := MedianeZ(ancres)
	// L'horizon a sa geometrie AU NIVEAU DES ANCRES, l'arene 1,7 m plus haut.
	horizon := horizonCommun()
	for i := range horizon.Instances {
		horizon.Instances[i].Position[2] = zJeu
	}
	arene := areneLiveFire()
	for i := range arene.Instances {
		arene.Instances[i].Position[2] = zJeu + 1.7
	}
	if got := ChoisitBSP([]BSPInstances{horizon, arene}, ancres); got.FileIndex != arene.FileIndex {
		t.Fatalf("bsp retenu tag#%d : la regle suit l'altitude, elle ne doit suivre que l'emprise", got.FileIndex)
	}
}

// TestChoisitBSPIgnoreUneBoiteVideALEgalite : `ReadModuleInstances` conserve les tags sbsp dont
// le bloc d'instances est vide. Une boite vide plus serree que l'arene la battrait sur
// l'emprise seule, et rendrait un fond BLANC.
func TestChoisitBSPIgnoreUneBoiteVideALEgalite(t *testing.T) {
	ancres := ancresLiveFire()
	vide := bspTemoin(9001, [3]float64{-12, 18, -2}, [3]float64{30, 50, 5}, 0)
	if got := CompteAncresDansBoite(vide.Bounds, ancres); got != len(ancres) {
		t.Fatalf("le temoin vide doit contenir toutes les ancres, il en contient %d", got)
	}
	if EmpriseAuSol(vide.Bounds) >= EmpriseAuSol(areneLiveFire().Bounds) {
		t.Fatal("le temoin vide doit etre PLUS SERRE que l'arene, sinon il ne teste rien")
	}
	got := ChoisitBSP([]BSPInstances{vide, areneLiveFire()}, ancres)
	if got.FileIndex != 1429 {
		t.Fatalf("bsp retenu tag#%d : une boite sans instance ne doit jamais gagner un departage", got.FileIndex)
	}
}

// TestChoisitBSPPrivilegieLeNombreDAncres : l'emprise ne departage QU'A nombre d'ancres egal.
// Une boite minuscule qui n'attrape qu'une ancre ne prend pas le pas sur l'arene.
func TestChoisitBSPPrivilegieLeNombreDAncres(t *testing.T) {
	ancres := ancresLiveFire()
	minuscule := bspTemoin(9002, [3]float64{-10, 20, -2}, [3]float64{-8, 22, 0}, 500)
	if n := CompteAncresDansBoite(minuscule.Bounds, ancres); n != 1 {
		t.Fatalf("le temoin doit attraper exactement 1 ancre, il en attrape %d", n)
	}
	got := ChoisitBSP([]BSPInstances{minuscule, areneLiveFire()}, ancres)
	if got.FileIndex != 1429 {
		t.Fatalf("bsp retenu tag#%d : le nombre d'ancres prime sur l'emprise", got.FileIndex)
	}
}

// TestChoisitBSPSansAncreRetombeSurLePlusPeuple : le repli inchange, quand aucune ancre ne
// tombe dans aucune boite (ou qu'il n'y a pas d'ancre du tout).
func TestChoisitBSPSansAncreRetombeSurLePlusPeuple(t *testing.T) {
	petit := bspTemoin(1, [3]float64{0, 0, 0}, [3]float64{10, 10, 10}, 7)
	gros := bspTemoin(2, [3]float64{0, 0, 0}, [3]float64{1000, 1000, 100}, 70)
	if got := ChoisitBSP([]BSPInstances{petit, gros}, nil); got.FileIndex != gros.FileIndex {
		t.Fatalf("sans ancre, bsp retenu tag#%d, attendu le plus peuple tag#%d", got.FileIndex, gros.FileIndex)
	}
	hors := [][3]float64{{-500, -500, -500}}
	if got := ChoisitBSP([]BSPInstances{petit, gros}, hors); got.FileIndex != gros.FileIndex {
		t.Fatalf("ancres hors de toute boite : bsp retenu tag#%d, attendu le plus peuple tag#%d",
			got.FileIndex, gros.FileIndex)
	}
}

// TestEmpriseAuSolNonMesurableVautInfini : une boite degeneree ne doit jamais gagner un
// departage — sinon un bsp dont les bornes sont illisibles raflerait toutes les cartes.
func TestEmpriseAuSolNonMesurableVautInfini(t *testing.T) {
	cas := map[string]Bounds{
		"plate":   {Min: [3]float64{0, 0, 0}, Max: [3]float64{0, 10, 10}},
		"inverse": {Min: [3]float64{10, 10, 0}, Max: [3]float64{0, 0, 10}},
		"nan":     {Min: [3]float64{0, 0, 0}, Max: [3]float64{math.NaN(), 10, 10}},
		"infinie": {Min: [3]float64{0, 0, 0}, Max: [3]float64{math.Inf(1), 10, 10}},
	}
	for nom, b := range cas {
		if e := EmpriseAuSol(b); !math.IsInf(e, 1) {
			t.Errorf("%s : emprise %v, attendu +Inf", nom, e)
		}
	}
}
