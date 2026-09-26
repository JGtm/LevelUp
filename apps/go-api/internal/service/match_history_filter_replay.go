// Package service — match_history_filter_replay.go : le filtre « Rejeu » des lignes de
// match (Explorer / historique) et ses trois états. Fichier à part de
// match_history_service_filters.go, qui est au seuil des 500 lignes.
package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Les trois états du filtre rejeu, miroir exact du scope escouade : vide = tous.
const (
	scopeWithReplay    = "with"
	scopeWithoutReplay = "without"
)

// filterByExplorerReplayScope filtre selon la présence d'un artefact de rejeu 2D :
// "with" = le match a un rejeu, "without" = il n'en a pas, sinon noop (tous).
//
// `replays` est l'ensemble résolu UNE FOIS par requête (un listing de dossier) : vide
// (titre sans rejeu cuit) → "with" ne garde rien et "without" garde tout, ce qui est la
// vérité mesurée et pas une panne.
func filterByExplorerReplayScope(
	rows []domain.MatchHistoryRawRow, scope string, replays port.ReplayAvailability,
) []domain.MatchHistoryRawRow {
	switch scope {
	case scopeWithReplay:
		out := rows[:0:0]
		for _, r := range rows {
			if replays.Has(r.MatchID) {
				out = append(out, r)
			}
		}
		return out
	case scopeWithoutReplay:
		out := rows[:0:0]
		for _, r := range rows {
			if !replays.Has(r.MatchID) {
				out = append(out, r)
			}
		}
		return out
	}
	return rows
}
