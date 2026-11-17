package service

// tactical_service_reapparitions_adverses.go — LA LECTURE « REAPPARITIONS ADVERSES » : ou les
// adversaires reapparaissent quand l'equipe du joueur de la page tient la zone choisie, a cote de la
// repartition sans condition. Le calcul est pur (analysis/tactical.LireReapparitionsAdverses) ; ici,
// l'orchestration : porte, univers, sidecars, verdict.
//
// # LES PORTES, DANS L'ORDRE
//
//	capability        film.replay_artifact ET film.spawn_forecast (les sidecars du titre portent les
//	                  reapparitions et l'instantane), plus un lecteur de sidecars cable : sinon
//	                  ErrCapabilityNotSupported (503 propre) ;
//
// AUCUN FILTRE PAR MODE : tout match mesure de l'univers entre, quel que soit son mode ; ce sont les
// planchers de donnees (matchs distincts, reapparitions) qui decident si la repartition est servie.
//
// Ni les poids du modele de reapparition ni leur fichier ne servent ici : la lecture COMPTE des
// reapparitions observees, elle n'en prevoit aucune (la prevision est celle du rejeu).

import (
	"context"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/i18n"
)

// ReapparitionsAdverses rend la lecture pour la zone choisie de la carte.
func (s *TacticalService) ReapparitionsAdverses(ctx context.Context,
	req domain.TacticalReapparitionsAdversesRequest) (domain.TacticalReapparitionsAdverses, error) {
	out := domain.TacticalReapparitionsAdverses{
		Etat: domain.TacticalEtatVide,
		Plancher: domain.TacticalPlancherReapparitions{
			Matchs:           domain.ReapparitionsAdversesPlancherMatchs,
			Reapparitions:    domain.ReapparitionsAdversesPlancherReapparitions,
			JoueursPourTenir: domain.ReapparitionsAdversesJoueursPourTenir,
		},
		Reference: domain.TacticalRepartitionZones{Zones: []domain.TacticalPartDeZone{}},
		Tenue:     domain.TacticalRepartitionZones{Zones: []domain.TacticalPartDeZone{}},
	}
	scope := domain.TacticalScope{MatchIDs: req.Scope.MatchIDs, Coequipiers: compositionNettoyee(req.Scope.Coequipiers)}
	if err := domain.ValiderComposition(scope.Coequipiers); err != nil {
		s.logger.WarnContext(ctx, "tactique: composition refusee (reapparitions adverses)",
			"player", s.xuid, "coequipiers", len(scope.Coequipiers), "err", err)
		return out, err
	}
	if err := s.porteReapparitionsAdverses(ctx, req.MapID); err != nil {
		return out, err
	}
	debut := time.Now()
	univers, err := s.repo.Univers(ctx, s.requeteAvecRetention(s.cartesDe(ctx, req.MapID), scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers des reapparitions adverses en echec",
			"player", s.xuid, "map_id", req.MapID, "err", err)
		return out, err
	}
	if len(univers.Matchs) == 0 {
		s.logger.InfoContext(ctx, "tactique: carte sans match retenu (reapparitions adverses)",
			"player", s.xuid, "map_id", req.MapID)
		return out, domain.ErrTacticalCarteInconnue
	}
	zones := s.zonesDeLaCarte(ctx, req.MapID)
	choisie, ok := zoneDuVolume(tactical.ZonesDepuisLePort(zones), req.VolumeIndex)
	if !ok {
		s.logger.InfoContext(ctx, "tactique: zone inconnue (reapparitions adverses)",
			"player", s.xuid, "map_id", req.MapID, "volume_index", req.VolumeIndex, "zones", len(zones))
		return out, domain.ErrTacticalZoneInconnue
	}
	loc := i18n.FromContext(ctx)
	out.Zone = domain.TacticalZoneChoisie{Nom: choisie.Nom.PickOptional(loc)}

	sidecars, _ := s.chargerSidecars(ctx, univers, req.MapID)
	obs, matchs := s.observationsAdverses(univers, sidecars)
	out.Reference, out.Tenue = tactical.LireReapparitionsAdverses(loc, obs, choisie, tactical.ZonesDepuisLePort(zones))
	s.verdictReapparitionsAdverses(&out, len(sidecars), matchs.retenus, tactical.ZoneAUneForme(choisie))
	s.logger.InfoContext(ctx, "tactique: reapparitions adverses",
		"player", s.xuid, "map_id", req.MapID, "volume_index", req.VolumeIndex, "etat", out.Etat, "raison", out.Raison,
		"matchs_filtres", len(univers.Matchs), "matchs_retenus", matchs.retenus,
		"matchs_sans_equipe", matchs.sansEquipe,
		"reference", out.Reference.Reapparitions, "tenue", out.Tenue.Reapparitions,
		"sans_zone_reference", out.Reference.HorsZone(), "sans_zone_tenue", out.Tenue.HorsZone(),
		"duration", time.Since(debut))
	return out, nil
}

// porteReapparitionsAdverses verifie ce que la lecture exige du titre et du cablage.
func (s *TacticalService) porteReapparitionsAdverses(ctx context.Context, carte string) error {
	if s.repo == nil {
		return games.ErrCapabilityNotSupported
	}
	if !s.caps.Has(games.CapFilmReplayArtifact) || !s.caps.Has(games.CapFilmSpawnForecast) {
		s.logger.WarnContext(ctx, "tactique: reapparitions adverses indisponibles — le titre ne les porte pas dans ses sidecars",
			"player", s.xuid, "map_id", carte,
			"capability", string(games.CapFilmSpawnForecast), "capability_artefact", string(games.CapFilmReplayArtifact))
		return games.ErrCapabilityNotSupported
	}
	if s.rasters == nil {
		s.logger.ErrorContext(ctx, "tactique: reapparitions adverses demandees sans lecteur de sidecars cable",
			"player", s.xuid, "map_id", carte)
		return games.ErrCapabilityNotSupported
	}
	return nil
}

// zoneDuVolume retrouve la zone d'un catalogue par son indice de volume.
func zoneDuVolume(zones []tactical.ZoneNommee, volume int) (tactical.ZoneNommee, bool) {
	for _, z := range zones {
		if z.VolumeIndex == volume {
			return z, true
		}
	}
	return tactical.ZoneNommee{}, false
}

// comptesAdverses : les matchs de la lecture. retenus nourrissent la lecture (sidecar exploitable, equipe
// du joueur nommee) ; sansEquipe sont mesures mais sans equipe nommee pour le joueur. Un match sans
// sidecar n'est pas mesure : il n'est dans aucun des deux.
type comptesAdverses struct{ retenus, sansEquipe int }

// observationsAdverses rassemble les reapparitions adverses des matchs qui peuvent servir la lecture et
// compte les matchs retenus et ceux sans equipe.
func (s *TacticalService) observationsAdverses(univers domain.TacticalUnivers,
	sidecars map[string]*domain.TacticalRasterSidecar) ([]tactical.ObservationAdverse, comptesAdverses) {
	var obs []tactical.ObservationAdverse
	var matchs comptesAdverses
	for _, m := range univers.Matchs {
		sc := sidecars[m.MatchID]
		if sc == nil {
			continue
		}
		joueurs := make([]tactical.JoueurLu, 0, len(sc.Joueurs))
		for _, j := range sc.Joueurs {
			joueurs = append(joueurs, joueurLu(j, true))
		}
		du, equipeConnue := tactical.ObservationsAdversesDuMatch(m.MatchID, joueurs, s.xuid)
		if !equipeConnue {
			matchs.sansEquipe++
			continue
		}
		matchs.retenus++
		obs = append(obs, du...)
	}
	return obs, matchs
}

// verdictReapparitionsAdverses pose l'etat de la reponse : servie quand la repartition « zone tenue »
// est lisible, sinon vide avec la raison la plus precise. `mesures` est le nombre de matchs au sidecar
// exploitable, `retenus` celui des matchs qui nourrissent la lecture.
func (s *TacticalService) verdictReapparitionsAdverses(out *domain.TacticalReapparitionsAdverses,
	mesures, retenus int, zoneAForme bool) {
	switch {
	case mesures == 0:
		out.Raison = domain.TacticalRaisonAucunMatchMesure
	case retenus == 0:
		out.Raison = domain.TacticalRaisonEquipeInconnue
	case !zoneAForme:
		out.Raison = domain.TacticalRaisonZoneSansForme
	case !out.Tenue.Suffisante:
		out.Raison = out.Tenue.RaisonSousPlancher()
	default:
		out.Etat = domain.TacticalEtatServie
	}
}
