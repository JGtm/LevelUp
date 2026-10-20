// Package port — replay_availability.go : l'ensemble des matchs qui ont un artefact de
// rejeu 2D. Type à part de services.go : il est consommé par TOUS les producteurs de
// lignes de match (historique, Explorer, escouade), pas seulement par le service de rejeu.
package port

import "levelup/go-api/internal/domain/title"

// ReplayAvailability est l'ensemble des matchs d'un titre dont l'artefact de rejeu 2D
// existe, indexé par la forme COURTE du match_id (title.FilmShortMatchID) — la clé sous
// laquelle l'artefact est rangé sur disque. Un ensemble nil ou vide = aucun rejeu, ce qui
// est un état servi (titre sans film cuit, dossier absent), jamais une erreur.
type ReplayAvailability map[string]struct{}

// Has dit si un match a un artefact. Accepte indifféremment le match_id complet ou sa
// forme courte : la normalisation est faite ici, une seule fois, pour que les appelants
// n'aient jamais à connaître la forme de rangement des artefacts.
func (a ReplayAvailability) Has(matchID string) bool {
	if len(a) == 0 || matchID == "" {
		return false
	}
	_, ok := a[title.FilmShortMatchID(matchID)]
	return ok
}
