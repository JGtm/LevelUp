// Package middleware — match_participation.go : couche B de l'ADR 0029 sur les routes
// d'un match (/players/{player_slug}/matches/{match_id}/...).
//
// Dans son propre profil, un joueur n'ouvre que les matchs auxquels il a participé :
// la vue du match répondait déjà 404 match_not_participant à un autre match, mais le
// rejeu, les positions, les objectifs et les événements le servaient (et le rejeu est
// public en production). Ce middleware pose la même réponse sur toutes les routes du
// match. Il se monte en middleware de ROUTE (r.With), seul endroit où {match_id} est
// résolu par chi ; une route sans {match_id} le traverse.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain/title"
)

// MatchParticipationCheck dit si le joueur du profil `playerSlug` a participé au
// match `matchID`.
type MatchParticipationCheck func(ctx context.Context, playerSlug, matchID string) (bool, error)

// RequireMatchParticipation répond 404 match_not_participant quand le joueur du
// chemin n'a pas participé au match du chemin, 503 participation_unavailable quand la
// vérification échoue (jamais d'ouverture par défaut).
//
// Avant toute vérification, et même sans vérification (check nil : démo, auth
// inactive), un {match_id} hors de la forme d'un identifiant de match
// (title.IsValidMatchID) reçoit le même 404 : il deviendrait la clé d'un chemin
// d'artefact (rejeu, faits de film, raster tactique). Même réponse que l'absence, la
// frontière n'est pas un oracle.
func RequireMatchParticipation(check MatchParticipationCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			matchID := chi.URLParam(r, "match_id")
			slug := chi.URLParam(r, "player_slug")
			if matchID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !title.IsValidMatchID(matchID) {
				slog.InfoContext(r.Context(), "participation: match_id mal formé refusé",
					"match_id", matchID, "player", slug, "path", r.URL.Path)
				writeMatchError(w, http.StatusNotFound, "match_not_participant", false)
				return
			}
			if check == nil {
				next.ServeHTTP(w, r)
				return
			}
			participated, err := check(r.Context(), slug, matchID)
			if err != nil {
				slog.ErrorContext(r.Context(), "participation: vérification impossible",
					"err", err, "match_id", matchID, "player", slug)
				writeMatchError(w, http.StatusServiceUnavailable, "participation_unavailable", true)
				return
			}
			if !participated {
				slog.InfoContext(r.Context(), "participation: match non joué refusé",
					"match_id", matchID, "player", slug, "path", r.URL.Path)
				writeMatchError(w, http.StatusNotFound, "match_not_participant", false)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeMatchError(w http.ResponseWriter, status int, code string, retryable bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		errKeyCode:      code,
		errKeyMessage:   code,
		errKeyRetryable: retryable,
	})
}
