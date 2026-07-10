// Package service - teammates_service_intersect.go : intersection des matchs
// d'escouade par composition exacte. Découpe de teammates_service.go (god-file
// split). Corrige le bug "coéquipier ajouté à une session qu'il n'a pas jouée" :
// la page Escouade unionnait les matchs par coéquipier au lieu de les intersecter.
package teammates

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// intersectSquadRowsByMatchID retourne les matchs présents chez TOUS les
// coéquipiers (composition exacte). Chaque set vient de
// LoadSquadMatches(main, teammate) = "main ∩ teammate" ; l'intersection des sets
// = matchs joués par le joueur principal ET tous les coéquipiers sélectionnés.
//
//   - 0 set  -> nil
//   - 1 set  -> set dédupliqué par match_id (== comportement historique mono-coéquipier)
//   - N sets -> match_id présents dans les N sets
//
// Émet une seule row par match_id (les rows partagent la perspective du joueur
// principal, donc sont équivalentes). Trie par StartTime ASC puis MatchID pour
// un résultat déterministe (les builders en aval re-trient selon leur besoin).
func intersectSquadRowsByMatchID(setsByTeammate [][]domain.SquadMatchRow) []domain.SquadMatchRow {
	if len(setsByTeammate) == 0 {
		return nil
	}

	// Index par match_id pour chaque set (dédup intra-set). On retient l'index du
	// plus petit set pour minimiser le balayage de l'intersection.
	indexed := make([]map[string]domain.SquadMatchRow, len(setsByTeammate))
	smallest := 0
	for i, set := range setsByTeammate {
		idx := make(map[string]domain.SquadMatchRow, len(set))
		for _, m := range set {
			if _, dup := idx[m.MatchID]; !dup {
				idx[m.MatchID] = m
			}
		}
		indexed[i] = idx
		if len(idx) < len(indexed[smallest]) {
			smallest = i
		}
	}

	out := make([]domain.SquadMatchRow, 0, len(indexed[smallest]))
	for id, row := range indexed[smallest] {
		present := true
		for i, idx := range indexed {
			if i == smallest {
				continue
			}
			if _, ok := idx[id]; !ok {
				present = false
				break
			}
		}
		if present {
			out = append(out, row)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartTime.Equal(out[j].StartTime) {
			return out[i].StartTime.Before(out[j].StartTime)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}
