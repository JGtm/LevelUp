package port

import (
	"context"

	"levelup/go-api/internal/analysis/narrative"
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

// ObjectiveIndexRepository charge, sur un scope fermé de matchs, les agrégats par
// (joueur × famille de mode) attendus par narrative.ComputeObjectiveIndex — l'axe
// « Objectifs » par opportunité des profils de participation (Session, Escouade,
// Ascension ; plan PLAN_AXE_OBJECTIFS_INDEX).
//
// Même contrat de dégradation que ObjectiveStatsRepository : best-effort, résultat
// vide sans échec de page (l'axe est alors retiré, pas affiché à 0). Implémenté par
// internal/platform/duckdb.ObjectiveStatsRepo (vue match_objective_stats_latest).
// Câblé UNIQUEMENT pour les titres portant la capability match.objective.stats.
type ObjectiveIndexRepository interface {
	// LoadObjectiveIndexInputs : par xuid, agrégats par famille sur le scope.
	LoadObjectiveIndexInputs(
		ctx context.Context,
		matchIDs []string,
		xuids []string,
	) (map[string]narrative.ObjectiveIndexInput, error)

	// LoadObjectiveIndexInputsByGamertag : variante gamertag (coéquipiers non
	// suivis, résolution shared.xuid_aliases).
	LoadObjectiveIndexInputsByGamertag(
		ctx context.Context,
		matchIDs []string,
		gamertag string,
	) (narrative.ObjectiveIndexInput, error)
}
