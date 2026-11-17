package tactical

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestZonesDepuisLePort_CopieLaFormeEtLaTranche : chaque champ que le paquet lit est copié, et la
// projection d'une liste vide reste une liste non nulle (même contrat que les anciennes copies).
func TestZonesDepuisLePort_CopieLaFormeEtLaTranche(t *testing.T) {
	poly := [][2]float64{{0, 0}, {1, 0}, {1, 1}}
	part := [][][2]float64{{{5, 5}, {6, 5}, {6, 6}}}
	trou := [][][2]float64{{{2, 2}, {3, 2}, {3, 3}}}
	in := []domain.ZoneNommee{{
		VolumeIndex: 7, X: 1, Y: 2, ZBas: -1.5, ZHaut: 3.25, Z: 9, Big: true,
		Polygone: poly, Parties: part, Trous: trou,
	}}
	out := ZonesDepuisLePort(in)
	if len(out) != 1 {
		t.Fatalf("projection : %d zones, attendu 1", len(out))
	}
	z := out[0]
	if z.VolumeIndex != 7 || z.X != 1 || z.Y != 2 || z.ZBas != -1.5 || z.ZHaut != 3.25 {
		t.Fatalf("champs scalaires non projetés : %+v", z)
	}
	if len(z.Polygone) != 3 || len(z.Parties) != 1 || len(z.Trous) != 1 {
		t.Fatalf("forme non projetée : %+v", z)
	}
	if vide := ZonesDepuisLePort(nil); vide == nil || len(vide) != 0 {
		t.Fatalf("liste vide : %#v, attendu une liste non nulle de longueur 0", vide)
	}
}
