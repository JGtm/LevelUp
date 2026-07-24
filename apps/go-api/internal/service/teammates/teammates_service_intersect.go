// Package service - teammates_service_intersect.go : intersection des matchs
// d'escouade par composition exacte. Découpe de teammates_service.go (god-file
// split). Corrige le bug "coéquipier ajouté à une session qu'il n'a pas jouée" :
// la page Escouade unionnait les matchs par coéquipier au lieu de les intersecter.
//
// L'intersection (intersectSquadRowsByMatchID) garantit seulement que le joueur
// principal a joué avec TOUS les coéquipiers sélectionnés — mais PAS qu'aucun
// autre coéquipier connu n'était présent. filterExactComposition ajoute cette
// exclusivité : à partir de l'équipe alliée du main par match, il écarte les
// matchs où un coéquipier connu HORS sélection (extraPool = amis ∪ top \ sélection)
// figure sur l'équipe du main. Les fills de lobby / bots / adversaires (hors
// pool) n'entrent jamais dans la comparaison → ils sont conservés.
package teammates

import (
	"sort"
	"strings"

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

// collectSelectedXUIDs extrait les xuids (non vides) des coéquipiers sélectionnés
// construits (leur *XUID est résolu par buildTeammateRowWithMatches). Sert de set
// "composition sélectionnée" pour le filtre exclusif ET de squadXUIDs pour
// LoadMapStatsForSquad.
func collectSelectedXUIDs(teammates []domain.TeammateRow) []string {
	out := make([]string, 0, len(teammates))
	for _, t := range teammates {
		if t.XUID != nil && *t.XUID != "" {
			out = append(out, *t.XUID)
		}
	}
	return out
}

// resolveFriendXUIDs traduit les gamertags amis (settings.friend_gamertags) en
// xuids via la table gamertag→xuid des top coéquipiers déjà chargés (Q29). Les
// amis hors top-50 ne sont pas résolus ici (co-jouer avec eux serait de toute
// façon marginal ; le pool top les couvre pour l'essentiel). Case-insensitive.
func resolveFriendXUIDs(friendGamertags []string, topRows []domain.TopTeammateRow) []string {
	if len(friendGamertags) == 0 {
		return nil
	}
	gtToXUID := make(map[string]string, len(topRows))
	for _, r := range topRows {
		if r.XUID != "" {
			gtToXUID[strings.ToLower(strings.TrimSpace(r.Gamertag))] = r.XUID
		}
	}
	out := make([]string, 0, len(friendGamertags))
	for _, gt := range friendGamertags {
		if x, ok := gtToXUID[strings.ToLower(strings.TrimSpace(gt))]; ok {
			out = append(out, x)
		}
	}
	return out
}

// buildExtraPoolXUIDs calcule le set des "autres coéquipiers connus" à écarter
// pour une composition exacte : tous les top coéquipiers (Q29) ∪ les amis résolus,
// MOINS la composition sélectionnée et le joueur principal. Les fills de lobby,
// bots et adversaires n'appartenant pas à ce pool ne cassent jamais la composition.
func buildExtraPoolXUIDs(
	topRows []domain.TopTeammateRow,
	friendXUIDs []string,
	selectedXUIDs []string,
	mainXUID string,
) map[string]struct{} {
	exclude := make(map[string]struct{}, len(selectedXUIDs)+1)
	for _, x := range selectedXUIDs {
		if x != "" {
			exclude[x] = struct{}{}
		}
	}
	if mainXUID != "" {
		exclude[mainXUID] = struct{}{}
	}
	pool := make(map[string]struct{}, len(topRows)+len(friendXUIDs))
	add := func(x string) {
		if x == "" {
			return
		}
		if _, skip := exclude[x]; skip {
			return
		}
		pool[x] = struct{}{}
	}
	for _, r := range topRows {
		add(r.XUID)
	}
	for _, x := range friendXUIDs {
		add(x)
	}
	return pool
}

// sortedXUIDSlice matérialise un set de xuids en slice triée (déterminisme des
// placeholders SQL de l'anti-join LoadMapStatsForSquad).
func sortedXUIDSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// collectMatchIDs déduplique les match_id de plusieurs jeux de SquadMatchRow
// (union). Sert à charger l'équipe alliée du main en un seul appel batch.
func collectMatchIDs(sets ...[]domain.SquadMatchRow) []string {
	total := 0
	for _, set := range sets {
		total += len(set)
	}
	seen := make(map[string]struct{}, total)
	out := make([]string, 0, total)
	for _, set := range sets {
		for _, r := range set {
			if r.MatchID == "" {
				continue
			}
			if _, ok := seen[r.MatchID]; ok {
				continue
			}
			seen[r.MatchID] = struct{}{}
			out = append(out, r.MatchID)
		}
	}
	return out
}

// buildMainTeamXUIDSet indexe la sortie de LoadMainTeamParticipants en
// match_id -> set(xuid) de l'équipe alliée du main (main inclus). nil si vide.
func buildMainTeamXUIDSet(allies []domain.AllyParticipant) map[string]map[string]struct{} {
	if len(allies) == 0 {
		return nil
	}
	out := make(map[string]map[string]struct{})
	for _, a := range allies {
		if a.MatchID == "" || a.XUID == "" {
			continue
		}
		set, ok := out[a.MatchID]
		if !ok {
			set = make(map[string]struct{})
			out[a.MatchID] = set
		}
		set[a.XUID] = struct{}{}
	}
	return out
}

// matchHasExactComposition indique si l'équipe alliée du main pour un match
// correspond EXACTEMENT à la composition sélectionnée parmi les joueurs connus :
// chaque xuid sélectionné est présent ET aucun xuid de l'extraPool (autres
// coéquipiers connus) n'est présent. Équipe inconnue (aucune info chargée) -> false
// (on écarte : on ne peut pas confirmer l'exactitude).
func matchHasExactComposition(
	team map[string]struct{},
	extraPool map[string]struct{},
	selectedXUIDs []string,
) bool {
	if team == nil {
		return false
	}
	for _, x := range selectedXUIDs {
		if _, ok := team[x]; !ok {
			return false
		}
	}
	// Itère l'équipe (petite : ~4-12 joueurs) plutôt que l'extraPool (~50).
	for x := range team {
		if _, ok := extraPool[x]; ok {
			return false
		}
	}
	return true
}

// filterExactComposition ne garde que les matchs dont l'équipe du main correspond
// EXACTEMENT à la composition sélectionnée (cf. matchHasExactComposition).
// mainTeamByMatch nil (chargement échoué / non tenté) => rows inchangés
// (dégradation gracieuse, page non blanchie).
func filterExactComposition(
	rows []domain.SquadMatchRow,
	mainTeamByMatch map[string]map[string]struct{},
	extraPool map[string]struct{},
	selectedXUIDs []string,
) []domain.SquadMatchRow {
	if mainTeamByMatch == nil || len(rows) == 0 {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if matchHasExactComposition(mainTeamByMatch[r.MatchID], extraPool, selectedXUIDs) {
			out = append(out, r)
		}
	}
	return out
}

// exactCompositionFilter porte les données du filtre "composition exacte" pour
// les surfaces qui ne dérivent PAS de allSquadRows (le briefing header, calculé à
// partir des rows canoniques). Un pointeur nil ou un teamByMatch nil désactive le
// filtre (dégradation gracieuse). Bundle -> respecte la limite d'arguments.
type exactCompositionFilter struct {
	teamByMatch   map[string]map[string]struct{}
	extraPool     map[string]struct{}
	selectedXUIDs []string
}

func (f *exactCompositionFilter) enabled() bool {
	return f != nil && f.teamByMatch != nil
}

// applyShared filtre les matchs partagés du briefing par composition exacte.
func (f *exactCompositionFilter) applyShared(matches []domain.SquadSharedMatch) []domain.SquadSharedMatch {
	if !f.enabled() || len(matches) == 0 {
		return matches
	}
	out := matches[:0:0]
	for _, m := range matches {
		if matchHasExactComposition(f.teamByMatch[m.MatchID], f.extraPool, f.selectedXUIDs) {
			out = append(out, m)
		}
	}
	return out
}
