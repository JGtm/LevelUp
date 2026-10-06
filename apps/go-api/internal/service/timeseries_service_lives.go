// Package service — timeseries_service_lives.go : « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL »
// (Séries temporelles › Usages ; plan `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L3 ;
// calcul : analysis/coordination.ViesPresOuSeul, type publié : domain.TimeseriesLivesNearTeammate).
//
// Orchestration seule : UNE lecture bornée par les matchs de la fenêtre et le joueur consulté
// (port.SoloLivesRepository, ADR 0036), la portée COURANTE du radar de chaque match par
// `mappings.PorteesDuRadarParMatch` (source unique), le calcul pur. Capability absente (repo non
// câblé ou table manquante) : bloc absent, journalisé en Debug ; lecture en échec : bloc absent,
// journalisé en Error — jamais une erreur de page, jamais un zéro inventé.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// WithLivesNearTeammate injecte le lecteur des vies (câblé sous `film.kill_positions`).
func (s *TimeseriesService) WithLivesNearTeammate(repo port.SoloLivesRepository) *TimeseriesService {
	s.usages.livesRepo = repo
	return s
}

// WithRadarRange injecte la table des portées de radar du titre (game_variant_name -> mètres) — la
// MÊME source que l'onglet Tactique et le placement de l'Emprise. Sans table, aucun match n'a de
// portée : toutes les vies sont écartées et comptées, jamais rapportées à un rayon de repli.
func (s *TimeseriesService) WithRadarRange(parVariante map[string]int) *TimeseriesService {
	s.usages.radarRange = parVariante
	return s
}

// attachLives pose `lives_near_teammate` sur la réponse.
func (s *TimeseriesService) attachLives(
	ctx context.Context, resp *domain.TimeseriesPageResponse, filteredCanon []canonical.PlayerMatchRow,
) {
	if len(filteredCanon) == 0 || s.playerXUID == "" {
		return
	}
	if s.usages.livesRepo == nil {
		slog.DebugContext(ctx, "timeseries_vies_capability_absente", "player", s.gamertag,
			"capability", string(games.CapFilmKillPositions), "err", games.ErrCapabilityNotSupported)
		return
	}
	defer timing.FromContext(ctx).Section("lives")()
	ids := synthesisMatchIDs(filteredCanon)
	lues, err := s.usages.livesRepo.LoadLivesNearTeammate(ctx, ids, s.playerXUID)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "timeseries_vies_capability_absente", "player", s.gamertag,
			"capability", string(games.CapFilmKillPositions), "err", err)
		return
	case err != nil:
		slog.ErrorContext(ctx, "timeseries_vies_en_echec", "player", s.gamertag, "matchs", len(ids), "err", err)
		return
	}
	rayons, sansRayon := mappings.PorteesDuRadarParMatch(s.usages.radarRange, lues.Variantes)
	bloc, fragsEcartes := coordination.ViesPresOuSeul(lues, rayons)
	if bloc.MatchesRead == 0 {
		slog.InfoContext(ctx, "timeseries_vies_sans_vie", "player", s.gamertag, "matchs", len(ids))
		return
	}
	slog.InfoContext(ctx, "timeseries_vies",
		"player", s.gamertag, "matchs", len(ids), "matchs_lus", bloc.MatchesRead,
		"matchs_sans_portee", bloc.MatchesWithoutRadar, "variantes_sans_portee", sansRayon,
		"pres", bloc.Near.Lives, "seul", bloc.Alone.Lives, "ecartees_sans_coequipier", bloc.ExcludedUnlocated,
		"ecartees_sans_portee", bloc.ExcludedNoRadar, "frags_ecartes", fragsEcartes)
	resp.LivesNearTeammate = &bloc
}
