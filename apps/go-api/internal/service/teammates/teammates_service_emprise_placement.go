// Package teammates — teammates_service_emprise_placement.go : LE BLOC « GROUPÉS OU ISOLÉS » de
// l'onglet Emprise (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3.3 ; type publié :
// domain.SquadEmprisePlacement, calcul : analysis/squademprise.Placement).
//
// Orchestration seule, sur le périmètre D2 de l'Emprise et sa composition (les fiches du bloc) :
//
//	lecture   UNE, bornée par les matchs du périmètre ET les xuids de la composition
//	          (port.SquadLifePlacementRepository, ADR 0036), sous sa propre section de durée ;
//	portée    la portée COURANTE de la variante de chaque match, résolue par
//	          mappings.PorteeDuRadar (source unique, la même que l'écriture au sync) ;
//	calcul    pur ; les vies à portée périmée sont écartées par le calcul et journalisées ici.
//
// Capability `film.kill_positions` absente (le lecteur n'est pas câblé, ou la table manque) :
// bloc absent, games.ErrCapabilityNotSupported journalisé en Debug — la porte des vies au sync.
package teammates

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// WithLifePlacement injecte le lecteur du placement des vies. Câblé sous `film.kill_positions`
// (la porte des vies au sync) ; sans lui, le bloc « Groupés ou isolés » est absent.
func (s *TeammatesService) WithLifePlacement(repo port.SquadLifePlacementRepository) *TeammatesService {
	s.placementRepo = repo
	return s
}

// WithRadarRange injecte la table des portées de radar du titre (game_variant_name -> mètres,
// `regulation.toml [radar_range_m]`) — MÊME source que `ServiceRegistry.radarRangeFor` pour
// l'onglet Tactique. Sans injection (nil ou vide), aucun match n'a de portée courante : toutes les
// vies sortent de l'univers du placement, comptées, jamais rapportées à un rayon de repli.
func (s *TeammatesService) WithRadarRange(parVariante map[string]int) *TeammatesService {
	s.radarRange = parVariante
	return s
}

// attacherPlacement publie le placement des vies dans le bloc de l'Emprise (bloc nil : rien).
func (s *TeammatesService) attacherPlacement(
	ctx context.Context, bloc *domain.SquadEmpriseBlock, scope []legacymatch.SynthesisMatchRow,
) {
	if bloc == nil {
		return
	}
	defer timing.FromContext(ctx).Section("emprise_placement")()
	if s.placementRepo == nil {
		slog.DebugContext(ctx, "teammates_emprise_placement_capability_absente",
			"player", s.gamertag, "capability", string(games.CapFilmKillPositions),
			"err", games.ErrCapabilityNotSupported)
		return
	}
	ids := teammatesMatchIDs(scope)
	read, err := s.placementRepo.LoadLifePlacement(ctx, ids, xuidsDesFiches(bloc.Players))
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "teammates_emprise_placement_capability_absente",
			"player", s.gamertag, "capability", string(games.CapFilmKillPositions), "err", err)
		return
	case err != nil:
		slog.ErrorContext(ctx, "teammates_emprise_placement_en_echec",
			"player", s.gamertag, "matchs", len(ids), "err", err)
		return
	}
	rayon, sansRayon := rayonParMatchDuScope(read.Variants, s.radarRange)
	placement, bilan := squademprise.Placement(squademprise.PlacementInput{
		Players: bloc.Players, Scope: ids, Read: read, CurrentRadar: rayon,
	})
	s.journaliserPlacement(ctx, placement, bilan, len(ids), sansRayon)
	bloc.Placement = placement
}

// journaliserPlacement : Warn sur les vies à portée périmée (un rattrapage est dû) et sur des
// lignes hors du périmètre demandé (le dépôt n'en rend pas), Info du bloc sinon.
func (s *TeammatesService) journaliserPlacement(
	ctx context.Context, p *domain.SquadEmprisePlacement, bilan squademprise.PlacementBilan, matchs, sansRayon int,
) {
	if bilan.IgnoredRows > 0 {
		slog.WarnContext(ctx, "teammates_emprise_placement_lignes_hors_perimetre",
			"player", s.gamertag, "lignes", bilan.IgnoredRows)
	}
	if p == nil {
		slog.InfoContext(ctx, "teammates_emprise_placement_sans_vie",
			"player", s.gamertag, "matchs", matchs, "cause", "aucune vie ecrite pour la composition")
		return
	}
	if p.Coverage.StaleLives > 0 {
		slog.WarnContext(ctx, "teammates_emprise_placement_portee_perimee",
			"player", s.gamertag, "vies", p.Coverage.StaleLives, "matchs", bilan.StaleMatches,
			"cause", "portee ecrite au sync differente de la portee courante : rattrapage du")
	}
	slog.InfoContext(ctx, "teammates_emprise_placement",
		"player", s.gamertag, "matchs", matchs, "matchs_avec_placement", p.Coverage.MatchesWithPlacement,
		"matchs_sans_rayon", sansRayon, "vies", p.Coverage.LivesTotal, "vies_mesurees", p.Coverage.LivesMeasured,
		"vies_perimees", p.Coverage.StaleLives)
}

// xuidsDesFiches — les xuids de la composition, dans l'ordre des fiches.
func xuidsDesFiches(players []domain.SessionUsageSquadPlayer) []string {
	out := make([]string, 0, len(players))
	for _, p := range players {
		if p.XUID != "" {
			out = append(out, p.XUID)
		}
	}
	return out
}

// rayonParMatchDuScope résout la portée COURANTE du radar de chaque match, par sa variante —
// même univers que `TacticalService.rayonsParMatch` (service Tactique), la source
// (`s.radarRange`) vivant sur un service différent, avec sa propre injection (WithRadarRange).
// La résolution d'une variante est `mappings.PorteeDuRadar`, la seule du dépôt (plan Emprise
// vies, lot V2b). Un match dont la variante n'a pas de portée SORT de l'univers de la lecture et
// se compte (correction G2, doctrine reprise telle quelle). Lecteur : le placement des vies
// depuis le lot V3 (il servait le nuage « Frags non ripostés », retiré par la décision V7).
func rayonParMatchDuScope(variantes map[string]string, radar map[string]int) (map[string]float64, int) {
	out := make(map[string]float64, len(variantes))
	sans := 0
	for matchID, variante := range variantes {
		metres, ok := mappings.PorteeDuRadar(radar, variante)
		if !ok {
			sans++
			continue
		}
		out[matchID] = metres
	}
	return out, sans
}
