package service

// tactical_service_vies.go — LE FILTRE DE GRAPPE DANS LE DETAIL D'UNE CELLULE.
//
// La lecture agregee (`Raster`) applique le filtre de grappe par `perimetreDuSpawn`, avant son
// dispatch. Le detail d'une cellule d'artefact n'a pas ce passage de lui-meme : il relit ses
// sidecars match par match. Il doit pourtant ne lister que ce que la cellule agregee compte —
// sous `depart`, les matchs dont la premiere vie part de la grappe ; sous `toutes`, ceux qui ont
// une vie partie de la grappe, et pour « routes » les seules routes de ces vies. Sans quoi un clic
// sur une cellule listerait des vies que la carte ne compte pas.

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// viesAcceptees valide le parametre `vies` d'une demande et rend sa valeur canonique : vide vaut
// `depart`, une valeur hors vocabulaire est refusee (`ErrTacticalViesInconnue`).
//
// `toutes` EXIGE `film.spawn_forecast` : c'est la capacite qui dit que les sidecars du titre portent
// les reapparitions et les debuts de manche. Un titre qui n'a que `film.replay_artifact` n'y ecrit
// que la premiere vie ; le servir en `toutes` rendrait un 200 qui ne contient que les departs sous
// un libelle « toutes les vies ». Il est refuse (`ErrCapabilityNotSupported`, 503 cote HTTP). La
// capacite se lit sur la carte du titre, jamais sur son slug.
func (s *TacticalService) viesAcceptees(ctx context.Context, carte, brut string) (string, error) {
	vies, err := domain.NormaliserVies(brut)
	if err != nil {
		s.logger.WarnContext(ctx, "tactique: vies refusees", "player", s.xuid, "map_id", carte, "err", err)
		return "", err
	}
	if vies == domain.TacticalViesToutes && !s.caps.Has(games.CapFilmSpawnForecast) {
		s.logger.WarnContext(ctx, "tactique: toutes les vies demandees sur un titre sans reapparitions dans ses sidecars",
			"player", s.xuid, "map_id", carte, "capability", string(games.CapFilmSpawnForecast))
		return "", games.ErrCapabilityNotSupported
	}
	return vies, nil
}

// appliquerLeFiltreDeGrappe resout le filtre de grappe d'un detail de cellule, sous les deux
// valeurs de `vies` : la liste de matchs du scope est reduite a ceux que la grappe retient, et les
// sidecars lus (routes reduites aux vies de la grappe sous `toutes`) sont rendus. Sans filtre de
// grappe, elle ne touche a rien et rend `nil`.
func (s *TacticalService) appliquerLeFiltreDeGrappe(ctx context.Context, carte string,
	scope *domain.TacticalScope) (map[string]*domain.TacticalRasterSidecar, error) {
	if scope.Spawn == "" {
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
