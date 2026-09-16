package port

import (
	"context"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
)

// SessionUsageRepository charge, sur un scope fermé de matchs (la session
// affichée), les lignes du résumé d'usage S1 et l'appartenance de camp — les
// entrées de sessionusage.ComputeUsage (bloc « usages » de la page Sessions).
//
// LECTURE PAR LES VUES `_latest` UNIQUEMENT (ADR 0026) : une lecture des tables
// brutes servirait les lignes d'une passe précédente. L'absence d'une ligne film
// pour un match EST l'information « match non mesuré » — jamais une erreur.
//
// Implémenté par internal/platform/duckdb.SessionUsageRepo. Câblé UNIQUEMENT
// pour les titres portant la capability film.usage_summary (jamais de gating par
// slug) — repo absent ⇒ bloc Available=false avec raison, réponse partielle
// propre.
type SessionUsageRepository interface {
	// LoadUsageFilms : par match_id, la ligne match_usage_films_latest. Un match
	// absent de la map n'est pas mesuré.
	LoadUsageFilms(ctx context.Context, matchIDs []string) (map[string]sessionusage.FilmRow, error)
	// LoadUsagePlayers : les lignes match_usage_players_latest du scope.
	LoadUsagePlayers(ctx context.Context, matchIDs []string) ([]sessionusage.PlayerRow, error)
	// LoadPadTiers : les PRISES DE SOCLE PAR NIVEAU D'ARME, une ligne par
	// (match, joueur, niveau, arme), lues de `match_pad_pickups_by_tier_latest`.
	// Un match ABSENT n'est pas un match a zero : sa passe n'a pas eu lieu, et
	// les niveaux se lisent « non mesures ».
	LoadPadTiers(ctx context.Context, matchIDs []string) ([]sessionusage.PadTierRow, error)
	// LoadParticipants : les participants (match_participants) du scope —
	// appartenance de camp (attribution) + présence à la fin (effectifs).
	LoadParticipants(ctx context.Context, matchIDs []string) ([]sessionusage.ParticipantRow, error)
}

// SquadFormesUsageRepository — le résumé d'usage, PLUS le grain match des socles
// (`pad_named`, `weapon_pads_json`) que seul le bloc « formes retenues » lit.
//
// Interface SÉPARÉE plutôt que trois méthodes de plus sur SessionUsageRepository :
// les lecteurs existants (page Sessions, Synthèse, Escouade/équipement) n'ont que
// faire des socles au grain match, et élargir leur port aurait obligé chaque
// double de test à implémenter une méthode qu'il n'appelle jamais.
type SquadFormesUsageRepository interface {
	SessionUsageRepository
	// LoadUsageFilmPads : par match_id, les occupations de socle du match et les
	// prises qui portent un nom. Un match absent n'est pas mesuré.
	LoadUsageFilmPads(ctx context.Context, matchIDs []string) (map[string]squadformes.FilmPads, error)
}

// SquadFormesObjectiveRepository — les colonnes d'objectif par joueur (les deux
// camps) du scope. Implémenté par duckdb.ObjectiveStatsRepo, câblé gated par la
// capability des stats d'objectif ; nil ⇒ le bloc se sert sans ses cartes
// d'objectif, jamais un échec.
type SquadFormesObjectiveRepository interface {
	LoadObjectiveColumnRows(ctx context.Context, matchIDs []string) ([]squadformes.ObjectiveColumnRow, error)
	// LoadFlagGrabsNet : les PRISES NETTES de drapeau, une ligne par (match, joueur),
	// lues du film. Un (match, joueur) ABSENT n'est pas un zéro : son
	// artefact n'a pas été lu, et la grandeur s'affiche « non mesurée ».
	//
	// Lecture SÉPARÉE de la précédente parce que la table l'est : les colonnes
	// d'objectif viennent du sync API, les prises nettes du film. Deux lectures
	// indépendantes dégradent indépendamment.
	LoadFlagGrabsNet(ctx context.Context, matchIDs []string) ([]sessionusage.FlagGrabsNetRow, error)
}
