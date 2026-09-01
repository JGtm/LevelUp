package himap

import (
	"math"
	"path/filepath"
	"testing"
)

// LE CANEVAS D'UNE CARTE FORGE PORTE-T-IL LA GEOMETRIE DU TERRAIN ?
//
// Question de l'utilisateur, 2026-08-27 : « et si on avait la mauvaise approche ? Les cartes
// Forge ont une base sans doute, sur laquelle dessiner, et nous on ne doit avoir que cette
// base. Ou inversement. »
//
// L'etat de l'art disait qu'un canevas ne porte AUCUNE instance de geometrie — c'est ecrit
// dans cartes_forge.go, et c'est ce qui a fait identifier Corpo. L'inventaire des tags le
// contredit en partie : `fo08_wetland` porte DEUX sbsp et 9 866 fichiers, `fo11_blank` en
// porte deux aussi. La cuisson Forge, elle, ne pose que les objets de la variante et n'a
// jamais dessine le canevas.
//
// Ce test mesure ce que ces sbsp contiennent et s'ils enveloppent les ancres d'une carte batie
// dessus. Il ne conclut pas : il donne les nombres qui tranchent.
func TestCanevasForgePorteTIlDuTerrain(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	// Isolation et Vagabond sont toutes deux baties sur fo08_wetland.
	for _, carte := range []string{"isolation_map", "vagabond_map"} {
		lo, hi, n := empriseAncresVariante(t, carte)
		t.Logf("%-16s %d ancres  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]",
			carte, n, lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])
	}
	for _, canevas := range []string{CanevasWetland, CanevasBlank} {
		chemin := filepath.Join(racine, "pc", "levels", "multi", canevas, canevas+"-rtx-new.module")
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%s : lecture KO : %v", canevas, err)
			continue
		}
		for i, b := range bsps {
			t.Logf("%-14s bsp %d  X [%+8.1f ; %+8.1f]  Y [%+8.1f ; %+8.1f]  Z [%+8.1f ; %+8.1f]  %5d instances",
				canevas, i, b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2], len(b.Instances))
		}
	}
}

// empriseAncresVariante rend l'emprise des ancres d'objectif d'une variante du depot.
func empriseAncresVariante(t *testing.T, carte string) (lo, hi [3]float64, n int) {
	t.Helper()
	lo = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, e := range ancresDuModule(t, carte) {
		for _, o := range e.Objectives {
			n++
			for k, v := range [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z} {
				lo[k] = math.Min(lo[k], v)
				hi[k] = math.Max(hi[k], v)
			}
		}
	}
	return lo, hi, n
}

// TestCanevasSousLaTrancheDeJeu — LA CARTE EST-ELLE BATIE SUR LE TERRAIN, OU AU-DESSUS ?
//
// Suite de la question de l'utilisateur. Une fois etabli que le canevas porte de la geometrie,
// reste a savoir si la carte s'y POSE. Le depart est numerique : la tranche de rendu est
// [zJeu-12 ; zJeu+28] autour du niveau de jeu ; si le sommet de l'ile du canevas tombe SOUS
// ce plancher, la carte flotte et dessiner le canevas ne changera pas un pixel.
//
// Mesure du 2026-08-27 sur Isolation : ancres Z entre +112,6 et +121,5, ile du canevas jusqu'a
// +242,3 mais son terrain sous les ancres — la cuisson avec canevas a rendu un fichier
// IDENTIQUE A L'OCTET. La carte est donc batie AU-DESSUS de son terrain.
func TestCanevasSousLaTrancheDeJeu(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	lo, hi, n := empriseAncresVariante(t, "isolation_map")
	if n == 0 {
		t.Skip("ancres d'Isolation introuvables dans le depot")
	}
	zJeu := (lo[2] + hi[2]) / 2
	min, max := TrancheDeJeu(zJeu)
	chemin := filepath.Join(racine, "pc", "levels", "multi", CanevasWetland, CanevasWetland+"-rtx-new.module")
	bsps, err := ReadModuleInstances(chemin)
	if err != nil {
		t.Skipf("canevas illisible : %v", err)
	}
	for i, b := range bsps {
		dansLaTranche := b.Bounds.Max[2] >= min && b.Bounds.Min[2] <= max
		t.Logf("bsp %d  Z [%+8.1f ; %+8.1f]  tranche de jeu [%+7.1f ; %+7.1f]  intersecte : %v",
			i, b.Bounds.Min[2], b.Bounds.Max[2], min, max, dansLaTranche)
	}
}
