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

// ObjectiveTeamRepository charge, sur un scope fermé de matchs, les lignes d'objectif de TOUS les
// participants (les deux camps) attendues par narrative.ComputeObjectiveTeamShare — l'axe « Objectif »
// des profils de participation (Session, Escouade, Ascension) et des défis de campagne : la note
// d'objectif du match (narrative.ObjectiveScoreOf), qui repose sur la part du joueur dans l'objectif de
// son équipe.
//
// Une erreur de lecture est rendue au caller, qui journalise et RETIRE l'axe (jamais un zéro) ; une vue
// absente (DB non migrée, titre sans la capability) rend une map vide sans erreur. Implémenté par
// internal/platform/duckdb.ObjectiveStatsRepo (vue match_objective_stats_latest). Câblé UNIQUEMENT pour
// les titres portant la capability match.objective.stats.
type ObjectiveTeamRepository interface {
	// LoadObjectiveTeamRows : par match, les lignes de tous les participants. Une requête par lot de
	// matchs, jamais une par match.
	LoadObjectiveTeamRows(ctx context.Context, matchIDs []string) (map[string][]narrative.ObjectiveTeamRow, error)

	// XUIDsByGamertag : variante gamertag (coéquipiers non suivis, résolution shared.xuid_aliases) ;
	// un gamertag peut porter plusieurs xuids, un gamertag inconnu est absent.
	XUIDsByGamertag(ctx context.Context, gamertags []string) (map[string][]string, error)
}
