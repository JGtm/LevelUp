package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// ObjectiveStatsRepository charge, sur un scope fermé de matchs, le cumul (SUM) des
// stats objectifs (CTF/Zones/Oddball) par joueur — pour les KPI « Objectifs » sobres
// des pages Synthèse et Escouade (PLAN_V72_OBJECTIVE_STATS.md, P4).
//
// Contrat de dégradation : best-effort. Une vue absente (DB non migrée) ou une erreur
// SQL dégrade en map vide SANS échec de page (le caller log en best-effort). Seuls les
// xuids portant au moins une stat objectif figurent dans le résultat.
//
// Implémenté par internal/platform/duckdb.ObjectiveStatsRepo (lecture de la vue
// match_objective_stats_latest sur le SharedReader). Câblé UNIQUEMENT pour les titres
// portant la capability match.objective.stats (jamais de gating par slug).
type ObjectiveStatsRepository interface {
	LoadAggregatedByXUID(
		ctx context.Context,
		matchIDs []string,
		xuids []string,
	) (map[string]*domain.ObjectiveAggregate, error)
}
