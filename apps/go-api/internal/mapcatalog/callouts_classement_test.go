package mapcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testutil"
)

// zoneDuDumpRidgeline : la partie du dump découpé de Ridgeline que l'étalonnage lit.
type zoneDuDumpRidgeline struct {
	VolumeIndex int    `json:"volumeIndex"`
	LibelleEN   string `json:"libelle_en"`
	Brut        struct {
		Polygone [][2]float64 `json:"polygone"`
	} `json:"brut"`
}

// dumpRidgeline charge le dump versionné de référence (échec dur si l'arbre ne le porte pas :
// fichier versionné, son absence sur un checkout propre est une anomalie).
func dumpRidgeline(t *testing.T) map[int]zoneDuDumpRidgeline {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(root, ".ai", "V7.5", "dumps", "callout_zones_ridgeline_clipped.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Zones []zoneDuDumpRidgeline `json:"zones"`
	}
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatal(err)
	}
	out := make(map[int]zoneDuDumpRidgeline, len(d.Zones))
	for _, z := range d.Zones {
		out[z.VolumeIndex] = z
	}
	return out
}

// TestClassementRidgelineReproduitLePOC — L'ÉTALONNAGE DU SEUIL, rejoué en continu.
//
// Le POC (rendu de référence) classe les 16 zones dessinées de Ridgeline en 11 GRANDES
// (pavage) et 5 FINES (Horseshoe, Hex Roof, Hex Basement, Red Hallway, Lower Horseshoe —
// étages imbriqués). Le classement par recouvrement doit le reproduire EXACTEMENT : si ce test
// tombe, c'est le seuil ou la mesure qui a bougé, et le rendu de toutes les cartes avec lui.
func TestClassementRidgelineReproduitLePOC(t *testing.T) {
	zones := dumpRidgeline(t)
	// Les 16 zones à forme propre du tag : volumes 10..25 (dump : a_forme_propre).
	var formes []FormeDeZone
	for vi := 10; vi <= 25; vi++ {
		z, ok := zones[vi]
		if !ok || len(z.Brut.Polygone) < 3 {
			t.Fatalf("volume %d absent ou sans polygone brut dans le dump", vi)
		}
		formes = append(formes, FormeDeZone{Index: vi, Contour: z.Brut.Polygone})
	}
	grandes := ClasserGrandes(formes)

	r := classementRaster{formes: formes, boites: make([][4]float64, len(formes)), cell: PasDeClassementNatif}
	for i, f := range formes {
		r.boites[i] = boiteDe(f.Contour)
	}
	attenduFine := map[int]bool{10: true, 14: true, 23: true, 24: true, 25: true}
	for i, f := range formes {
		t.Logf("vi=%2d %-16s recouvert=%.2f -> grande=%v", f.Index, zones[f.Index].LibelleEN, r.couverture(i), grandes[f.Index])
		if attenduFine[f.Index] == grandes[f.Index] {
			t.Errorf("vi=%d (%s) : grande=%v, le POC dit l'inverse", f.Index, zones[f.Index].LibelleEN, grandes[f.Index])
		}
	}
}

// TestPointDansPolygone — la règle pair-impair sur un polygone concave en U.
func TestPointDansPolygone(t *testing.T) {
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
		if got := pointDansPolygone(u, c.x, c.y); got != c.in {
			t.Errorf("(%.1f, %.1f) : in=%v, attendu %v", c.x, c.y, got, c.in)
		}
	}
}
