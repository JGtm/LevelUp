package squadagg

import (
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// IntersectByMatchID retourne les matchs présents chez TOUS les joueurs de
// perPlayer. Trié par StartedAt DESC (match le plus récent en premier).
//
// Si perPlayer est vide ou contient des slices vides, retourne nil.
func IntersectByMatchID(perPlayer map[string][]canonical.PlayerMatchRow) []domain.SquadSharedMatch {
	if len(perPlayer) == 0 {
		return nil
	}

	indexed := make(map[string]map[string]canonical.PlayerMatchRow, len(perPlayer))
	for gt, rows := range perPlayer {
		idx := make(map[string]canonical.PlayerMatchRow, len(rows))
		for _, r := range rows {
			idx[r.Summary.MatchID] = r
		}
		indexed[gt] = idx
	}

	// Trouver l'intersection : un match_id présent chez tous.
	sharedIDs := matchIDsPresentInAll(indexed)
	if len(sharedIDs) == 0 {
		return nil
	}

	out := make([]domain.SquadSharedMatch, 0, len(sharedIDs))
	for _, id := range sharedIDs {
		sm := buildSharedMatch(id, indexed)
		out = append(out, sm)
	}

	// Tri par StartedAt DESC, fallback alphabétique sur MatchID si égalité.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].StartedAt.After(out[j].StartedAt)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// matchIDsPresentInAll retourne la liste des match_id présents dans toutes les
// maps (intersection ensembliste sur les clés).
func matchIDsPresentInAll(indexed map[string]map[string]canonical.PlayerMatchRow) []string {
	if len(indexed) == 0 {
		return nil
	}
	// Choisir la map la plus petite comme base pour minimiser le travail.
	var smallestGT string
	smallestSize := -1
	for gt, m := range indexed {
		if smallestSize == -1 || len(m) < smallestSize {
			smallestSize = len(m)
			smallestGT = gt
		}
	}

	var out []string
	for id := range indexed[smallestGT] {
		present := true
		for gt, m := range indexed {
			if gt == smallestGT {
				continue
			}
			if _, ok := m[id]; !ok {
				present = false
				break
			}
		}
		if present {
			out = append(out, id)
		}
	}
	return out
}

// buildSharedMatch hydrate un SquadSharedMatch depuis les rows de chaque joueur.
// Les champs niveau-match (Map, Mode, Outcome, StartedAt) sont pris du joueur
// principal sortable (premier dans l'ordre alphabétique des gamertags pour
// reproductibilité).
func buildSharedMatch(matchID string, indexed map[string]map[string]canonical.PlayerMatchRow) domain.SquadSharedMatch {
	sm := domain.SquadSharedMatch{
		MatchID: matchID,
		Players: make(map[string]canonical.PlayerMatchRow, len(indexed)),
	}
	gts := make([]string, 0, len(indexed))
	for gt := range indexed {
		gts = append(gts, gt)
	}
	sort.Strings(gts)
	for _, gt := range gts {
		row := indexed[gt][matchID]
		sm.Players[gt] = row
		if sm.StartedAt.IsZero() {
			sm.StartedAt = row.Summary.StartedAtUTC
			sm.Map = row.Summary.Map
			sm.Mode = row.Summary.PairMode
			if sm.Mode == nil {
				sm.Mode = row.Summary.GameVariant
			}
			sm.Playlist = row.Summary.Playlist
			sm.Outcome = row.Summary.Outcome
		}
	}
	return sm
}
