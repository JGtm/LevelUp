package himap

import (
	"path/filepath"
	"sort"
	"testing"
)

// LE VERROU DU CHOIX DE BSP — sur les fichiers du jeu installe.
//
// Deux garanties, et elles vont ensemble :
//
//  1. Live Fire doit designer l'ARENE de `common-rtx-new.module` et non son horizon. C'est le
//     seul cas ou l'egalite d'ancres se joue entre une boite de 63 x 64 m et une de
//     7 890 x 7 776 m, et le seul ou un departage par l'altitude se serait trompe.
//  2. AUCUNE carte native ne change de bsp. Le departage par l'emprise remplace un departage
//     par la taille du tag qui, lui, n'etait ecrit nulle part : la seule preuve acceptable est
//     de rejouer l'ANCIENNE regle a cote de la nouvelle sur toutes les cartes installees et de
//     constater l'egalite tag par tag.

// moduleCommun est le module global qui porte la geometrie des cartes dont le module propre est
// un talon (Live Fire / `sgh_interlock` : six fichiers, aucun sbsp).
const moduleCommun = "common-rtx-new.module"

// ancresLiveFireInstallees rejoue l'assemblage EXACT de la production : mapfond-build regroupe
// les entrees du catalogue par dossier installe et DEDUPLIQUE les ancres par position, donc
// Live Fire est cuite avec l'union de sa variante libre et de sa variante classee.
func ancresLiveFireInstallees(t *testing.T) [][3]float64 {
	t.Helper()
	vues := map[[3]float64]bool{}
	var union [][3]float64
	for _, m := range []string{"live_fire_sgh_interlock", "live_fire_-_ranked_sgh_interlock"} {
		for _, e := range ancresDuModule(t, m) {
			for _, o := range e.Objectives {
				a := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
				if !vues[a] {
					vues[a] = true
					union = append(union, a)
				}
			}
		}
	}
	return union
}

// TestChoixBSPLiveFireDesigneLArene verrouille le cas qui a motive le departage.
func TestChoixBSPLiveFireDesigneLArene(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ancres := ancresLiveFireInstallees(t)
	if len(ancres) == 0 {
		t.Skip("catalogue sans ancre pour Live Fire")
	}
	bsps, err := ReadModuleInstances(filepath.Join(racine, "pc", "globals", moduleCommun))
	if err != nil {
		t.Skip(err)
	}
	// Le test ne vaut que si la CONFIGURATION mesuree tient encore : plusieurs bsp, et au
	// moins deux qui contiennent toutes les ancres. Si une mise a jour du jeu change cela, il
	// faut le VOIR, pas passer au vert sur un cas devenu trivial.
	complets := 0
	for _, b := range bsps {
		if CompteAncresDansBoite(b.Bounds, ancres) == len(ancres) {
			complets++
		}
	}
	if complets < 2 {
		t.Fatalf("%s : %d bsp contiennent les %d ancres, le departage n'est plus exerce "+
			"(mesure du 2026-08-27 : 2 sur 4)", moduleCommun, complets, len(ancres))
	}

	retenu := ChoisitBSP(bsps, ancres)
	t.Logf("Live Fire : %d ancres, %d bsp candidats, retenu tag#%d — %.1f x %.1f m, %d instances",
		len(ancres), len(bsps), retenu.FileIndex,
		retenu.Bounds.Extent(0), retenu.Bounds.Extent(1), len(retenu.Instances))

	if n := CompteAncresDansBoite(retenu.Bounds, ancres); n != len(ancres) {
		t.Fatalf("le bsp retenu ne contient que %d ancres sur %d", n, len(ancres))
	}
	// L'ARENE, PAS L'HORIZON. Mesure : arene 63,2 x 63,8 m, horizon 7 890 x 7 776 m. Le seuil
	// est place a 200 m, tres au-dessus de la plus grande arene mesuree (Catalyst, 408 m, mais
	// dans son propre module) et tres en dessous du plus petit horizon (501 m sur Streets).
	const coteAreneMax = 200.0
	if retenu.Bounds.Extent(0) > coteAreneMax || retenu.Bounds.Extent(1) > coteAreneMax {
		t.Fatalf("bsp retenu de %.0f x %.0f m : c'est un decor lointain, pas l'arene de Live Fire",
			retenu.Bounds.Extent(0), retenu.Bounds.Extent(1))
	}
	// ET IL PORTE LA MATIERE DE L'AIRE DE JEU. Mesure : 6 561 des 12 556 instances du bsp
	// retenu tombent dans la boite des ancres, contre 0 pour l'horizon.
	if n := instancesDansLaBoite(retenu, ancres); n < 1000 {
		t.Fatalf("le bsp retenu ne pose que %d instances dans la boite des ancres "+
			"(mesure du 2026-08-27 : 6 561)", n)
	}
	if retenu.FileIndex != 1429 {
		t.Logf("ATTENTION : le tag retenu n'est plus #1429 mais #%d — l'installation a change "+
			"depuis la mesure du 2026-08-27. Le verrou geometrique tient, la valeur est notee.",
			retenu.FileIndex)
	}
}

// instancesDansLaBoite compte les instances dont le centre tombe dans l'emprise XY des ancres.
func instancesDansLaBoite(b BSPInstances, ancres [][3]float64) int {
	lo, hi := boiteDesPoints(ancres)
	n := 0
	for _, in := range b.Instances {
		if in.Position[0] >= lo[0] && in.Position[0] <= hi[0] &&
			in.Position[1] >= lo[1] && in.Position[1] <= hi[1] {
			n++
		}
	}
	return n
}

// choixBSPAvantDepartage rejoue la regle EN VIGUEUR JUSQU'AU 2026-08-27 : le plus d'ancres, et
// l'egalite tranchee par l'ordre de lecture — donc, `ReadModuleInstances` triant les tags sbsp
// par taille decompressee decroissante, par le POIDS du tag.
//
// Elle ne vit QUE dans ce test, et pour une seule raison : prouver que le departage par
// l'emprise ne change aucune carte native. La supprimer supprimerait la preuve.
func choixBSPAvantDepartage(bsps []BSPInstances, ancres [][3]float64) BSPInstances {
	var meilleur BSPInstances
	if len(ancres) > 0 {
		mieux := 0
		for _, b := range bsps {
			if n := CompteAncresDansBoite(b.Bounds, ancres); n > mieux {
				mieux, meilleur = n, b
			}
		}
		if mieux > 0 {
			return meilleur
		}
	}
	for _, b := range bsps {
		if len(b.Instances) > len(meilleur.Instances) {
			meilleur = b
		}
	}
	return meilleur
}

// TestChoixBSPCartesNativesInchange : sur toutes les cartes du catalogue installees localement,
// l'ancienne regle et la nouvelle doivent designer le MEME tag. Les trois temoins nommes au
// chantier — Forbidden, Chasm, Catalyst — sont verifies explicitement et leur absence est une
// ERREUR, pas un saut : un balayage qui ne balaie rien passe au vert pour rien.
func TestChoixBSPCartesNativesInchange(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	chemin, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := chargeCatalogue(chemin)
	if err != nil {
		t.Fatal(err)
	}
	var modules []string
	vus := map[string]bool{}
	for _, e := range cat {
		if e.Module != "" && !vus[e.Module] {
			vus[e.Module] = true
			modules = append(modules, e.Module)
		}
	}
	sort.Strings(modules)

	temoins := map[string]bool{"ctf_forbidden": false, "chasm_map": false, "catalyst_map": false}
	mesurees, multi := 0, 0
	for _, mod := range modules {
		p, ok := ChercheModuleInstalle(mod)
		if !ok {
			continue
		}
		ancres := ancresXYZ(t, mod)
		if len(ancres) == 0 {
			continue
		}
		bsps, err := ReadModuleInstances(p)
		if err != nil {
			t.Logf("%-34s lecture KO : %v", mod, err)
			continue
		}
		mesurees++
		if len(bsps) > 1 {
			multi++
		}
		avant, apres := choixBSPAvantDepartage(bsps, ancres), ChoisitBSP(bsps, ancres)
		if _, estTemoin := temoins[mod]; estTemoin {
			temoins[mod] = true
			t.Logf("temoin %-16s %d bsp — tag#%d (%.0f x %.0f m, %d instances)",
				mod, len(bsps), apres.FileIndex, apres.Bounds.Extent(0), apres.Bounds.Extent(1),
				len(apres.Instances))
		}
		if avant.FileIndex != apres.FileIndex {
			t.Errorf("%s : le departage par l'emprise change le bsp — tag#%d (%.0f x %.0f m) "+
				"au lieu de tag#%d (%.0f x %.0f m)", mod,
				apres.FileIndex, apres.Bounds.Extent(0), apres.Bounds.Extent(1),
				avant.FileIndex, avant.Bounds.Extent(0), avant.Bounds.Extent(1))
		}
	}
	for mod, vu := range temoins {
		if !vu {
			t.Errorf("temoin %s non mesure : le balayage ne prouve rien sans lui", mod)
		}
	}
	if mesurees == 0 {
		t.Skip("aucune carte du catalogue installee")
	}
	t.Logf("BILAN : %d cartes mesurees, dont %d a plusieurs bsp — aucun changement de bsp",
		mesurees, multi)
}
