package service

// tactical_service_zones.go — LES ZONES NOMMÉES DU PLAN : les callouts de la carte, publiés avec la
// lecture dans la forme même du rejeu 2D (replaydoc.CalloutZone), que le plan dessine avec le même
// peintre (lib/replay/calloutsPaint.ts côté web).

import (
	"context"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
)

// zonesDuPlan rend les zones nommées de la carte, sous la porte des zones des grappes
// (`film.replay_artifact`) : un titre sans artefact de rejeu n'a pas de catalogue de callouts, et le
// magasin n'est pas interrogé. Carte hors catalogue : aucune zone.
//
// Le nom de CONCEPTION d'une zone n'est pas publié : le plan n'écrit que les libellés FR et EN.
func (s *TacticalService) zonesDuPlan(ctx context.Context, carte string) []replaydoc.CalloutZone {
	if !s.caps.Has(games.CapFilmReplayArtifact) {
		return nil
	}
	defer timing.FromContext(ctx).Section("tactical_zones")()
	zones := s.zonesDeLaCarte(ctx, carte)
	if len(zones) == 0 {
		return nil
	}
	out := make([]replaydoc.CalloutZone, 0, len(zones))
	for _, z := range zones {
		out = append(out, replaydoc.CalloutZone{
			VolumeIndex: z.VolumeIndex, EN: z.NomEN, FR: z.NomFR,
			X: z.X, Y: z.Y, Z: z.Z, ZBottom: z.ZBas, ZTop: z.ZHaut, Big: z.Big,
			Polygon: z.Polygone, Parts: z.Parties, Holes: z.Trous,
		})
	}
	return out
}
