package service

// synthesis_service_usage.go — LES BLOCS ANNEXES DE LA SYNTHÈSE chargés par un
// repo OPTIONNEL, gaté par capability et best-effort : les KPI d'objectifs.
//
// Le bloc « servi ou gâché » de l'équipement a quitté la Synthèse le 2026-09-13 pour
// l'onglet Progression des Séries temporelles (cf. timeseries_service_sections.go) : même
// producteur (squadagg.BuildEquipmentUsageBlock), même scope, une seule page l'affiche.
//
// Fichier thématique au sens de l'en-tête de synthesis_service.go : ce dernier
// tient le seuil des 500 lignes du dépôt, les responsabilités qui s'y ajoutent
// vivent à côté. Même patron que synthesis_weapon_range.go.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

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
