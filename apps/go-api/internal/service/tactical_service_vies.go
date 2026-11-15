package service

// tactical_service_vies.go — LE PARAMETRE `vies` DANS LE DETAIL D'UNE CELLULE.
//
// La lecture agregee (`Raster`) applique le filtre de grappe par `perimetreDuSpawn`, avant son
// dispatch. Le detail d'une cellule d'artefact n'a pas ce passage : il relit ses sidecars
// match par match. Sous « toutes les vies », il doit pourtant ne lister que les routes des vies
// qui partent de la grappe, comme la cellule agregee qu'il detaille — sans quoi un clic sur une
// cellule listerait des vies que la carte ne compte pas.

import (
	"context"

	"levelup/go-api/internal/domain"
)

// restreindreAuxVies resout le filtre de grappe d'un detail de cellule SOUS `toutes` : la liste
// de matchs du scope est reduite a ceux qui ont une vie partie de la grappe, et les sidecars lus
// (routes reduites aux vies de la grappe) sont rendus. Sans filtre de grappe, ou sous `depart`,
// elle ne touche a rien et rend `nil`.
func (s *TacticalService) restreindreAuxVies(ctx context.Context, carte string,
	scope *domain.TacticalScope) (map[string]*domain.TacticalRasterSidecar, error) {
	if scope.Vies != domain.TacticalViesToutes || scope.Spawn == "" {
		return nil, nil
	}
	per, err := s.perimetreDuSpawn(ctx, carte, *scope)
	if err != nil {
		return nil, err
	}
	scope.MatchIDs = per.MatchIDs
	return per.Sidecars, nil
}

// sidecarDeLaCellule rend le sidecar exploitable d'un match pour le detail d'une cellule : celui
// que la resolution du filtre de grappe a deja lu (et reduit) quand il y en a un, sinon celui du
// disque. `nil` : match non mesure, la contribution n'existe pas.
func (s *TacticalService) sidecarDeLaCellule(ctx context.Context, carte, matchID string,
	dejaLus map[string]*domain.TacticalRasterSidecar) *domain.TacticalRasterSidecar {
	if dejaLus != nil {
		return dejaLus[matchID]
	}
	sc, err := s.rasters.Charger(ctx, matchID)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: sidecar de raster illisible (detail de cellule)",
			"player", s.xuid, "map_id", carte, "match_id", matchID, "err", err)
		return nil
	}
	if !s.sidecarExploitable(ctx, sc, matchID) {
		return nil
	}
	return sc
}
