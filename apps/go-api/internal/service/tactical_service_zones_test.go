package service

// tactical_service_zones_test.go — les zones nommées de la carte servies avec la lecture du plan
// (plan Tactique v2, lot L13 F7) : même forme que les zones du rejeu 2D, sous la porte des zones
// des grappes (`film.replay_artifact`).

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

func zonesDeDeuxEtages() *mockCallouts {
	return &mockCallouts{zones: []domain.ZoneNommee{
		{NomFR: "Rez", NomEN: "Ground", X: 2.5, Y: 2.5, Z: 1, Big: true, Polygone: carre(0, 5),
			Parties: [][][2]float64{carre(6, 8)}, Trous: [][][2]float64{carre(1, 2)}, ZBas: 0, ZHaut: 2, VolumeIndex: 1},
		{NomFR: "Étage", NomEN: "Upper", X: 2.5, Y: 2.5, Z: 4, Polygone: carre(0, 5), ZBas: 3, ZHaut: 6, VolumeIndex: 2},
	}}
}

// TestRaster_ZonesDuPlan_Servies : la lecture du plan porte les zones de la carte, champ par champ.
func TestRaster_ZonesDuPlan_Servies(t *testing.T) {
	repo := repoDeuxFaces()
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithCalloutsStore(zonesDeDeuxEtages())
	got, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster : %v", err)
	}
	if len(got.Zones) != 2 {
		t.Fatalf("zones = %+v, want les deux zones de la carte", got.Zones)
	}
	z := got.Zones[0]
	if z.FR != "Rez" || z.EN != "Ground" || z.X != 2.5 || z.Y != 2.5 || z.Z != 1 || !z.Big ||
		len(z.Polygon) != 4 || len(z.Parts) != 1 || len(z.Holes) != 1 || z.ZBottom != 0 || z.ZTop != 2 || z.VolumeIndex != 1 {
		t.Errorf("zone = %+v, want la zone « Rez » telle que le catalogue la porte", z)
	}
	if got.Zones[1].FR != "Étage" || got.Zones[1].Big {
		t.Errorf("seconde zone = %+v, want « Étage », fine", got.Zones[1])
	}
}

// TestRaster_ZonesDuPlan_SansArtefactDeRejeu : un titre sans artefact de rejeu (Halo 5) ne sert
// aucune zone, et le magasin n'est pas interrogé.
func TestRaster_ZonesDuPlan_SansArtefactDeRejeu(t *testing.T) {
	repo := repoDeuxFaces()
	magasin := zonesDeDeuxEtages()
	halo5 := games.CapabilityMap{games.CapMatchEventsSpatial: games.CapSupported}
	got, err := NewTacticalService(repo, halo5, tsMoi).WithCalloutsStore(magasin).
		Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster : %v", err)
	}
	if got.Zones != nil || magasin.appels != 0 {
		t.Errorf("zones = %+v (magasin appelé %d fois), want aucune et aucun appel", got.Zones, magasin.appels)
	}
}
