package replay

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// balayerEquipementPorte lit l equipement porte : les objets ti=37 crees attaches, que la marche des
// trames a deja recueillis (grammar/equipement_porte.go), et la position a la creation des objets
// qu ils engendrent, aux bornes de la carte.
func (s *filmScan) balayerEquipementPorte() {
	reads, st, err := grammar.ScanEquipementPorte(s.fc, &s.world)
	if err != nil {
		slog.WarnContext(s.ctx, "equipement porte illisible — rejeu sans energie d equipement ni pose de faille",
			"err", err, "match_id", s.matchID)
		reads, st = nil, types.CarriedEquipmentStats{}
	}
	s.in.CarriedEquipment, s.in.CarriedEquipmentStats = reads, st
	s.opt.observe(s.ctx, "carriedEquipment", s.in.CarriedEquipment)
}
