package tactical

import "levelup/go-api/internal/domain"

// ZonesDepuisLePort projette les zones du port (catalogue des callouts) vers le type de ce paquet.
//
// C'est LA seule projection : le service et les commandes passent par elle, et le garde-rail
// `archlint/zones_projection_unique_test.go` refuse toute autre construction d'un ZoneNommee hors de
// ce paquet. Le type du port porte aussi Z et Big (hauteur du centre et grande zone, pour le rendu du
// plan) ; ce paquet ne les lit pas, donc la projection ne les copie pas.
func ZonesDepuisLePort(zones []domain.ZoneNommee) []ZoneNommee {
	out := make([]ZoneNommee, 0, len(zones))
	for _, z := range zones {
		out = append(out, ZoneNommee{
			Nom: z.Nom, X: z.X, Y: z.Y,
			Polygone: z.Polygone, Parties: z.Parties, Trous: z.Trous,
			ZBas: z.ZBas, ZHaut: z.ZHaut, VolumeIndex: z.VolumeIndex,
		})
	}
	return out
}
