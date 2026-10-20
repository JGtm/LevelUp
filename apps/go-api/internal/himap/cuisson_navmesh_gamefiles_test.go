//go:build gamefiles

package himap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCuitIsolationDepuisSonNavmesh — LE FOND D'ISOLATION, RENDU DEPUIS LE SOL JOUABLE.
//
// C'est le premier fond du chantier qui ne cherche pas a RETIRER quelque chose : il change de
// source. Le test verifie ce qui compte — que les ancres retrouvent leur sol — et journalise
// l'emprise, pour que le gate visuel se fasse sur une image dont on connait les chiffres.
func TestCuitIsolationDepuisSonNavmesh(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join("..", "hinavmesh", "testdata",
		"01af558d-53ab-4f05-ba68-92d805fc6260.navmesh.blob"))
	if err != nil {
		t.Skipf("blob de navigation absent : %v", err)
	}
	// Le catalogue keye une carte FORGE par son map_id, pas par un nom de module : on le lit
	// donc directement, comme le fait l'oracle du paquet hinavmesh.
	ancres := ancresDuCatalogue(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	if len(ancres) == 0 {
		t.Skip("ancres d'Isolation introuvables")
	}
	r, b, err := CuitCarteNavmesh(context.Background(), OptionsCuissonNavmesh{
		Blob: blob, Ancres: ancres, Cle: "01af558d-53ab-4f05-ba68-92d805fc6260",
		CibleCadrePx: CibleCadrePx,
	})
	if err != nil {
		t.Fatalf("cuisson depuis le navmesh : %v", err)
	}
	if b.AncresAvecSol == 0 {
		t.Fatal("aucune ancre au sol : le maillage ne couvre pas l'arene")
	}
	t.Logf("Isolation par son navmesh : %d triangles, %d/%d ancres au sol, grille %dx%d",
		b.Dessinees, b.AncresAvecSol, b.AncresDansLeCadre, r.NX, r.NY)
}

// ancresDuCatalogue lit les positions d'objectif d'une carte par sa CLE de catalogue — le
// map_id pour une carte Forge. `ancresDuModule` filtre sur le nom de module, ce qui ne
// convient pas ici : celui d'Isolation est generique.
func ancresDuCatalogue(t *testing.T, cle string) [][3]float64 {
	t.Helper()
	chemin, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // fichier versionne, lecture seule
	if err != nil {
		t.Skipf("catalogue illisible : %v", err)
	}
	var ref struct {
		Maps map[string]struct {
			Objectives []struct {
				Pos struct{ X, Y, Z float64 } `json:"pos"`
			} `json:"objectives"`
		} `json:"maps"`
	}
	if err := json.Unmarshal(brut, &ref); err != nil {
		t.Fatalf("catalogue illisible : %v", err)
	}
	var out [][3]float64
	for _, o := range ref.Maps[cle].Objectives {
		out = append(out, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	return out
}
