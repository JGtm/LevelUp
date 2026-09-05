package port

import (
	"context"

	"levelup/go-api/internal/analysis/sessionusage"
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
	// LoadParticipants : les participants (match_participants) du scope —
	// appartenance de camp (attribution) + présence à la fin (effectifs).
	LoadParticipants(ctx context.Context, matchIDs []string) ([]sessionusage.ParticipantRow, error)
}
