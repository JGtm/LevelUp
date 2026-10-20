// Package squadagg — lectures_usage.go : LES TROIS LECTURES DU RÉSUMÉ D'USAGE SUR UN SCOPE (films,
// joueurs, participants), faites UNE fois par requête et partagées par les blocs qui les lisent :
// l'Emprise et « les formes retenues » des Séries temporelles (ADR 0036 I4).
//
// AUCUN SQL ICI : les lectures sont celles de la page Sessions (port.SessionUsageRepository →
// duckdb.SessionUsageRepo), qui prennent un scope FERMÉ de match_id — seule la liste
// d'identifiants change d'une page à l'autre.
package squadagg

import (
	"context"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/port"
)

// LecturesUsage : les trois lectures communes aux blocs qui lisent le résumé d'usage d'un scope —
// les films, les joueurs, les participants —, erreurs comprises.
type LecturesUsage struct {
	Films        map[string]sessionusage.FilmRow
	Players      []sessionusage.PlayerRow
	Participants []sessionusage.ParticipantRow
	erreurs      []error // dans l'ordre films, joueurs, participants
}

// LireUsage fait les trois lectures communes sur le scope, toutes, même après un échec ;
// Erreur rend la première en échec.
func LireUsage(ctx context.Context, repo port.SessionUsageRepository, matchIDs []string) *LecturesUsage {
	films, filmsErr := repo.LoadUsageFilms(ctx, matchIDs)
	players, playersErr := repo.LoadUsagePlayers(ctx, matchIDs)
	participants, partErr := repo.LoadParticipants(ctx, matchIDs)
	return &LecturesUsage{
		Films: films, Players: players, Participants: participants,
		erreurs: []error{filmsErr, playersErr, partErr},
	}
}

// Erreur rend la première lecture en échec, dans l'ordre films, joueurs, participants.
func (l *LecturesUsage) Erreur() error {
	for _, err := range l.erreurs {
		if err != nil {
			return err
		}
	}
	return nil
}
