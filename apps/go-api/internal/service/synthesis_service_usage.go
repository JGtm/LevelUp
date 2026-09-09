package service

// synthesis_service_usage.go — LES BLOCS ANNEXES DE LA SYNTHÈSE chargés par un
// repo OPTIONNEL, gaté par capability et best-effort : les KPI d'objectifs, et le
// bloc « servi ou gâché » de l'équipement (étape E5.5 du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// Fichier thématique au sens de l'en-tête de synthesis_service.go : ce dernier
// tient le seuil des 500 lignes du dépôt, les responsabilités qui s'y ajoutent
// vivent à côté. Même patron que synthesis_weapon_range.go.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// WithEquipmentUsage injecte la source du résumé d'usage (vues _latest) et le
// résolveur d'amis configurés — la même paire que le bloc de la page Sessions,
// et la même source que l'accueil pour les amis.
//
// Câblé UNIQUEMENT pour les titres portant film.usage_summary (registry, jamais
// slug==) : repo nil ⇒ bloc servi avec Available=false et raison machine, réponse
// partielle propre (ADR 0011). Le xuid du joueur vient de
// WithPersonalScoreAwardsRepo, déjà câblé sur toutes les Synthèses.
func (s *SynthesisService) WithEquipmentUsage(
	repo port.SessionUsageRepository, friends teammates.FriendGamertagsResolver,
) *SynthesisService {
	s.sessionUsageRepo = repo
	s.usageFriends = friends
	return s
}

// loadEquipmentUsage publie le bloc « servi ou gâché » sur le scope FILTRÉ de la
// Synthèse (période + cascade déjà appliquées) : c'est le même scope que les KPI
// de la page, donc la même question posée aux mêmes matchs.
func (s *SynthesisService) loadEquipmentUsage(
	ctx context.Context, filteredCanon []canonical.PlayerMatchRow,
) *domain.EquipmentUsageBlock {
	return buildEquipmentUsageBlock(ctx, equipmentUsageQuery{
		Repo:            s.sessionUsageRepo,
		PlayerXUID:      s.playerXUID,
		MatchIDs:        synthesisMatchIDs(filteredCanon),
		FriendGamertags: s.friendGamertags(ctx),
	})
}

// friendGamertags résout les amis configurés (nil = aucun ami déclaré ; sur le
// grain période cela vaut « aucune part mes amis » — cf. ResolveScopeFriends).
func (s *SynthesisService) friendGamertags(ctx context.Context) []string {
	if s.usageFriends == nil {
		return nil
	}
	return s.usageFriends(ctx)
}

// synthesisMatchIDs — les identifiants d'un scope canonical, dans l'ordre. Helper
// unique du fichier Synthèse : la boucle existait en trois exemplaires, plafond
// de la règle CLAUDE.md n°6 — un quatrième l'aurait franchi.
func synthesisMatchIDs(rows []canonical.PlayerMatchRow) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Summary.MatchID)
	}
	return out
}

// loadObjectiveStats agrège (SUM) les stats objectifs du joueur sur le scope filtré.
// Best-effort : nil si repo non câblé (capability absente), joueur inconnu, scope vide,
// erreur SQL, ou aucun match à objectif dans le scope (bloc omis de la réponse).
func (s *SynthesisService) loadObjectiveStats(
	ctx context.Context, filteredCanon []canonical.PlayerMatchRow,
) *domain.ObjectiveAggregate {
	if s.objectiveStatsRepo == nil || s.playerXUID == "" || len(filteredCanon) == 0 {
		return nil
	}
	matchIDs := synthesisMatchIDs(filteredCanon)
	byXUID, err := s.objectiveStatsRepo.LoadAggregatedByXUID(ctx, matchIDs, []string{s.playerXUID})
	if err != nil {
		slog.WarnContext(ctx, "synthesis: objective stats query failed (best-effort)",
			"player_xuid", s.playerXUID, "match_count", len(matchIDs), "err", err)
		return nil
	}
	return byXUID[s.playerXUID]
}
