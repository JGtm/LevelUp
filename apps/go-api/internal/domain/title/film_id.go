package title

import "strings"

// film_id.go — LA CLÉ DE TOUT CE QUI DÉRIVE D'UN FILM.
//
// Un fichier à part plutôt qu'une ligne de plus dans `registry.go` : ce n'est pas une
// méthode de `PathResolver`, et `registry.go` dépasse déjà largement le seuil du dépôt.

// FilmShortMatchID rend la forme COURTE d'un identifiant de match : les 8 caractères
// hexadécimaux qui précèdent le premier tiret.
//
// C'EST LA CLÉ DES CHUNKS, DES MANIFESTS ET DE L'ARTEFACT DE REJEU. Elle vient du cache
// historique et elle est TENUE par les répertoires déjà sur disque
// (`data/cache/film_chunks/000d5950/`, `film_manifests/000d5950.json`). Le reste de
// l'application, lui, manipule le match_id COMPLET.
//
// POURQUOI CETTE FONCTION EXISTE, ET UNE SEULE FOIS. Sans elle, les deux formes se
// croisaient sans jamais se rencontrer : l'outil hors ligne écrivait `000d5950.json`
// (forme courte, celle du film qu'il venait de décoder) et l'API cherchait
// `000d5950-….json` (forme complète, celle de la route). L'artefact existait, et le
// service répondait « aucun rejeu » — un 404 sur un fichier présent, donc un lien qui
// n'aurait JAMAIS pu apparaître. Mesuré le 2026-08-02 : les 3 artefacts du dépôt sont
// tous en forme courte.
//
// Un identifiant sans tiret et plus court que 8 caractères est rendu tel quel : on
// tronque une forme connue, on n'en fabrique pas une.
func FilmShortMatchID(matchID string) string {
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		return matchID[:i]
	}
	if len(matchID) > 8 {
		return matchID[:8]
	}
	return matchID
}
