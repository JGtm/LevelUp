// Package sync — transforms_personal_scores.go : extraction des PersonalScores
// (awards) depuis le JSON match Halo.
//
// Portage de src/data/sync/transformers/_personal_scores.py.
// Chemin API : Players[].PlayerTeamStats[].Stats.CoreStats.PersonalScores[]
package sync

// PersonalScoreAwardRow représente une ligne à insérer dans
// player.personal_score_awards. Champs alignés sur le schéma DDL
// (cf. internal/sync/schema.go).
type PersonalScoreAwardRow struct {
	MatchID       string
	XUID          string
	AwardName     string
	AwardCategory string
	AwardCount    int
	AwardScore    int
}

// findPersonalScores parcourt récursivement player_obj["PlayerTeamStats"]
// pour trouver la première liste PersonalScores non vide. Mirror de
// `_find_personal_scores` Python.
func findPersonalScores(playerObj map[string]any) []any {
	stats, ok := playerObj["PlayerTeamStats"]
	if !ok {
		return nil
	}
	return findPSRecursive(stats)
}

func findPSRecursive(x any) []any {
	switch v := x.(type) {
	case map[string]any:
		if ps, ok := v["PersonalScores"].([]any); ok && len(ps) > 0 {
			return ps
		}
		for _, child := range v {
			if r := findPSRecursive(child); r != nil {
				return r
			}
		}
	case []any:
		for _, child := range v {
			if r := findPSRecursive(child); r != nil {
				return r
			}
		}
	}
	return nil
}

// findPlayerByXUID retourne le dict joueur dont PlayerId contient le xuid donné.
// Mirror de `_find_player` Python — tolère les structures PlayerId variables
// (le JSON Halo enveloppe parfois le xuid dans `{"value": "xuid(...)"}`).
func findPlayerByXUID(players []any, xuid string) map[string]any {
	if xuid == "" {
		return nil
	}
	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		// Le PlayerId peut être string brut ou {value: "xuid(...)"} ; on cherche
		// le xuid dans la représentation string.
		pid := asString(player["PlayerId"])
		if pid == "" {
			// Tente JSON-stringify : pid est peut-être une map.
			pid = asString(player["PlayerId"])
		}
		if extractXUID(pid) == xuid {
			return player
		}
	}
	return nil
}

// safeIntFromAny convertit any en int (depuis float64/int/json.Number).
// Retourne 0 et false si non numérique.
func safeIntFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

// safeUint64FromAny pour les NameId (peuvent être > MaxInt32).
func safeUint64FromAny(v any) (uint64, bool) {
	switch n := v.(type) {
	case float64:
		return uint64(n), true
	case int:
		return uint64(n), true
	case int64:
		return uint64(n), true
	}
	return 0, false
}

// ExtractPersonalScoreAwards extrait les PersonalScores d'un joueur depuis le
// JSON match. Retourne les lignes prêtes à insérer dans personal_score_awards.
//
// Comportement aligné sur le pipeline Python :
//   - NameId inconnu → ligne ignorée (pas de panic).
//   - TotalPersonalScoreAwarded présent → utilisé pour award_score.
//   - TotalPersonalScoreAwarded absent → fallback count * psaPoints[NameId].
//   - Count absent → 0.
func ExtractPersonalScoreAwards(matchJSON map[string]any, matchID, xuid string) []PersonalScoreAwardRow {
	if matchID == "" || xuid == "" {
		return nil
	}
	players, _ := matchJSON["Players"].([]any)
	if len(players) == 0 {
		return nil
	}
	player := findPlayerByXUID(players, xuid)
	if player == nil {
		return nil
	}
	personalScores := findPersonalScores(player)
	if len(personalScores) == 0 {
		return nil
	}

	rows := make([]PersonalScoreAwardRow, 0, len(personalScores))
	for _, ps := range personalScores {
		entry, ok := ps.(map[string]any)
		if !ok {
			continue
		}
		nameID, ok := safeUint64FromAny(entry["NameId"])
		if !ok || nameID == 0 {
			continue
		}
		technical := technicalIDForPSA(nameID)
		if technical == "" {
			continue // NameId inconnu — on skip plutôt que stocker un id "0".
		}
		count, _ := safeIntFromAny(entry["Count"])
		var totalScore int
		if v, ok := safeIntFromAny(entry["TotalPersonalScoreAwarded"]); ok {
			totalScore = v
		} else {
			totalScore = count * psaPoints[nameID]
		}
		rows = append(rows, PersonalScoreAwardRow{
			MatchID:       matchID,
			XUID:          xuid,
			AwardName:     technical,
			AwardCategory: categorizePSA(nameID),
			AwardCount:    count,
			AwardScore:    totalScore,
		})
	}
	return rows
}
