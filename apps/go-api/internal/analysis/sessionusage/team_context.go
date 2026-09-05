package sessionusage

// team_context.go — le contexte de camp et d'effectif d'un scope de matchs,
// dérivé de match_participants (S1 ne duplique pas l'effectif, à dessein — §3
// du handoff). Consommé par l'assemblage des deux blocs (usage et objectifs).

// TeamContext — par match du scope : le camp du joueur suivi, le camp de chaque
// participant, et les effectifs présents à la fin.
type TeamContext struct {
	// PlayerTeam : matchID -> camp du joueur suivi. Clé ABSENTE = camp inconnu
	// (FFA, participant manquant) : les parts d'équipe de ce match sont hors
	// calcul — jamais un 0 inventé.
	PlayerTeam map[string]int
	// TeamOf : matchID -> (xuid -> camp), TOUS les participants à camp connu
	// (présents ou non : l'attribution d'une ligne d'usage ne dépend pas de la
	// présence à la fin).
	TeamOf map[string]map[string]int
	// TeamSize / LobbySize : participants PRÉSENTS à la fin (bots inclus).
	// TeamSize n'est défini que si le camp du joueur est connu.
	TeamSize  map[string]int
	LobbySize map[string]int
}

// BuildTeamContext dérive le contexte de camp d'un scope de participants.
func BuildTeamContext(playerXUID string, participants []ParticipantRow) TeamContext {
	tc := TeamContext{
		PlayerTeam: map[string]int{},
		TeamOf:     map[string]map[string]int{},
		TeamSize:   map[string]int{},
		LobbySize:  map[string]int{},
	}
	for i := range participants {
		p := &participants[i]
		if p.XUID == playerXUID && p.TeamID != nil {
			tc.PlayerTeam[p.MatchID] = *p.TeamID
		}
	}
	for i := range participants {
		p := &participants[i]
		if p.TeamID != nil {
			if tc.TeamOf[p.MatchID] == nil {
				tc.TeamOf[p.MatchID] = map[string]int{}
			}
			tc.TeamOf[p.MatchID][p.XUID] = *p.TeamID
		}
		if !p.PresentAtCompletion {
			continue
		}
		tc.LobbySize[p.MatchID]++
		if team, ok := tc.PlayerTeam[p.MatchID]; ok && p.TeamID != nil && *p.TeamID == team {
			tc.TeamSize[p.MatchID]++
		}
	}
	return tc
}
