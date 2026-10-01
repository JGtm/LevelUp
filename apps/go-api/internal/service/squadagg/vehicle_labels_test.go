package squadagg

import (
	"context"
	"path/filepath"
	"testing"
)

// TestVehicleFamilyLabels — seules les familles que le manifeste du titre qualifie portent un
// libellé, dans la langue de la requête (français par défaut, anglais sur demande) ; les noms
// propres du jeu (Warthog) n'en ont pas, le client les affiche depuis leur clé. Un dépôt ou un
// titre inconnus rendent une map vide, jamais un panic.
func TestVehicleFamilyLabels(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	fr := VehicleFamilyLabels(ctx, root, "halo_infinite", "fr")
	if fr["tourelle_fixe"] != "Tourelle fixe" {
		t.Errorf("libellé FR = %q, attendu « Tourelle fixe »", fr["tourelle_fixe"])
	}
	if _, nomme := fr["warthog"]; nomme {
		t.Errorf("Warthog est un nom propre, sans libellé : %v", fr)
	}
	if en := VehicleFamilyLabels(ctx, root, "halo_infinite", "en"); en["tourelle_fixe"] != "Fixed turret" {
		t.Errorf("libellé EN = %q, attendu « Fixed turret »", en["tourelle_fixe"])
	}
	for _, c := range [][2]string{{"", "halo_infinite"}, {root, ""}, {root, "titre_inconnu"}} {
		if got := VehicleFamilyLabels(ctx, c[0], c[1], "fr"); got == nil || len(got) != 0 {
			t.Errorf("(%q, %q) : %v, attendu une map vide non nil", c[0], c[1], got)
		}
	}
}
