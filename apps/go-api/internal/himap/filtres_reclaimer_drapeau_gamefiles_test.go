package himap

import (
	"os"
	"path/filepath"
	"testing"
)

// modulesPortantLeDrapeau : les modules ou le balayage a trouve des instances
// `exclude from intel map`, du plus fourni au moins fourni (mesure du 2026-08-27).
// Ecrits ici pour que le detail par bsp ne re-balaie pas les 41 modules.
var modulesPortantLeDrapeau = []struct{ variante, dossier string }{
	{"globals", "common-rtx-new"},
	{"multi", "va_launchsite"},
	{"multi", "sgh_blueprint"},
	{"multi", "ctf_illusion"},
	{"multi", "academy_weapon_drills"},
	{"multi", "btb_exiled"},
	{"multi", "pve_house"},
	{"multi", "btb_highpower"},
	{"multi", "ridgeline"},
}

// TestDrapeauCarteParBSP repartit le drapeau `exclude from intel map` BSP PAR BSP.
//
// LA QUESTION QUE CELA TRANCHE. Le balayage dit qu'un module porte des instances marquees ;
// il ne dit pas OU. Deux repartitions donnent des conclusions opposees :
//   - le drapeau est concentre sur les bsp de DECOR LOINTAIN → il ne sert a rien de plus que
//     le choix de bsp, qui les ecarte deja en bloc ;
//   - le drapeau est pose DANS le bsp de l'arene → il designe, a l'interieur de la geometrie
//     qu'on dessine, ce que les auteurs ne veulent pas voir sur une vue de dessus. C'est le
//     seul cas ou le brancher change une image.
//
// `common-rtx-new` est le module qui compte le plus ici : c'est lui qui porte la geometrie de
// Live Fire, dont le module de carte n'a aucun tag sbsp.
//
// Aucun index de modules n'est construit : ce test lit les tags sbsp et rien d'autre.
func TestDrapeauCarteParBSP(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	for _, m := range modulesPortantLeDrapeau {
		var chemin string
		if m.variante == "globals" {
			chemin = filepath.Join(racine, "pc", "globals", m.dossier+".module")
		} else {
			chemin = filepath.Join(racine, "pc", "levels", "multi", m.dossier, m.dossier+"-rtx-new.module")
		}
		if _, err := os.Stat(chemin); err != nil {
			t.Logf("%-24s : absent", m.dossier)
			continue
		}
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%-24s : %v", m.dossier, err)
			continue
		}
		t.Logf("=== %s : %d bsp ===", m.dossier, len(bsps))
		for _, b := range bsps {
			marquees, retenues, marqueesRetenues := 0, 0, 0
			for _, in := range b.Instances {
				vivante := !in.QuickDeleted() && !in.ProjecteurOmbre()
				if vivante {
					retenues++
				}
				if in.ExclueDeCarteIntel() {
					marquees++
					if vivante {
						marqueesRetenues++
					}
				}
			}
			part := 0.0
			if retenues > 0 {
				part = 100 * float64(marqueesRetenues) / float64(retenues)
			}
			t.Logf("   bsp #%-4d %6d instances (%6d retenues) · %5d marquees (%5d retenues, %5.2f%%) · emprise %.0f x %.0f m",
				b.FileIndex, len(b.Instances), retenues, marquees, marqueesRetenues, part,
				b.Bounds.Extent(0), b.Bounds.Extent(1))
		}
	}
}
