//go:build research

package replay

// depot_research_test.go — le seul helper de depot que les instruments de recherche utilisent seuls
// (plan D2, item 2.11) : hors du build par defaut, il serait un symbole mort.

import "strings"

// idCourtDeMatch rend la forme courte d'un match_id : les 8 caracteres hexadecimaux qui
// precedent le premier tiret, la cle des films et des artefacts du parc. Un identifiant sans
// tiret et de 8 caracteres ou moins est rendu tel quel. Les tests ne passent que des
// identifiants du parc, bien formes.
func idCourtDeMatch(matchID string) string {
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		return matchID[:i]
	}
	if len(matchID) > 8 {
		return matchID[:8]
	}
	return matchID
}
