package main

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testutil"
)

// dumpRidgeline charge le dump versionné de référence (échec dur si l'arbre ne le porte pas :
// fichier versionné, son absence sur un checkout propre est une anomalie).
func dumpRidgeline(t *testing.T) map[int]decoupeZone {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	_, zones, err := chargeDecoupe(filepath.Join(root, ".ai", "V7.5", "dumps",
		"callout_zones_ridgeline_clipped.json"))
	if err != nil {
		t.Fatal(err)
	}
	return zones
}

// TestClassifyRidgelineReproduitLePOC — L'ÉTALONNAGE DU SEUIL, rejoué en continu.
//
// Le POC (rendu de référence, artefact eb7b8af2) classe les 16 zones dessinées de
// Ridgeline en 11 GRANDES (pavage) et 5 FINES (Horseshoe, Hex Roof, Hex Basement,
// Red Hallway, Lower Horseshoe — étages imbriqués). Le classement par recouvrement doit
// le reproduire EXACTEMENT : si ce test tombe, c'est le seuil ou la mesure qui a bougé,
// et le rendu de toutes les cartes avec lui.
func TestClassifyRidgelineReproduitLePOC(t *testing.T) {
	zones := dumpRidgeline(t)
	// Les 16 zones à forme propre du tag : volumes 10..25 (dump : a_forme_propre).
	var shaped []shapedPoly
	for vi := 10; vi <= 25; vi++ {
		z, ok := zones[vi]
		if !ok || len(z.Brut.Polygone) < 3 {
			t.Fatalf("volume %d absent ou sans polygone brut dans le dump", vi)
		}
		shaped = append(shaped, shapedPoly{vi: vi, poly: z.Brut.Polygone})
	}
	big := classifyBig(shaped)

	raster := classementRaster{zones: shaped, boxes: bboxes(shaped), cell: classifyCell}
	attenduFine := map[int]bool{10: true, 14: true, 23: true, 24: true, 25: true}
	for _, z := range shaped {
		frac := raster.couverture(indexOf(shaped, z.vi))
		t.Logf("vi=%2d %-16s recouvert=%.2f -> big=%v", z.vi, zones[z.vi].LibelleEN, frac, big[z.vi])
		if attenduFine[z.vi] == big[z.vi] {
			t.Errorf("vi=%d (%s) : big=%v, le POC dit l'inverse", z.vi, zones[z.vi].LibelleEN, big[z.vi])
		}
	}
}

func bboxes(zs []shapedPoly) [][4]float64 {
	out := make([][4]float64, len(zs))
	for i, z := range zs {
		out[i] = bbox(z.poly)
	}
	return out
}

func indexOf(zs []shapedPoly, vi int) int {
	for i, z := range zs {
		if z.vi == vi {
			return i
		}
	}
	return -1
}

// TestPointInPoly — la règle pair-impair sur un carré troué implicite (concave en U).
func TestPointInPoly(t *testing.T) {
	u := [][2]float64{{0, 0}, {3, 0}, {3, 3}, {2, 3}, {2, 1}, {1, 1}, {1, 3}, {0, 3}}
	cas := []struct {
		x, y float64
		in   bool
	}{
		{0.5, 0.5, true},   // pied gauche
		{1.5, 0.5, true},   // base du U
		{1.5, 2.0, false},  // creux du U
		{2.5, 2.0, true},   // pied droit
		{-0.5, 0.5, false}, // dehors
	}
	for _, c := range cas {
		if got := pointInPoly(u, c.x, c.y); got != c.in {
			t.Errorf("(%.1f, %.1f) : in=%v, attendu %v", c.x, c.y, got, c.in)
		}
	}
}
