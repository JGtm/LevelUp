// Package service — timeseries_service_lives.go : « MES VIES : PRÈS D'UN COÉQUIPIER OU SEUL »
// (Séries temporelles › Usages ; plan `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L3 ;
// calcul : analysis/coordination.ViesPresOuSeul, type publié : domain.TimeseriesLivesNearTeammate).
//
// Orchestration seule : la lecture bornée et le calcul sont `lireViesPresOuSeul`
// (solo_lives_block.go, partagé avec la page Sessions), sur les matchs de la fenêtre.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
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
	resp.LivesNearTeammate = lireViesPresOuSeul(ctx, viesQuery{
		Page: "timeseries", Player: s.gamertag, PlayerXUID: s.playerXUID,
		Repo: s.usages.livesRepo, Radar: s.usages.radarRange, MatchIDs: synthesisMatchIDs(filteredCanon),
	})
}
