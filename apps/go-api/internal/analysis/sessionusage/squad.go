package sessionusage

// squad.go — LES AMIS D'UN SCOPE DE PÉRIODE : les amis configurés qui ont été mes alliés (même camp,
// hors moi) dans au moins un match du scope, et la machine d'alliés qui les trouve.

import (
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// MaxTrackedSquadPlayers borne les lignes d'escouade — même plafond que la
// composition de la page Escouade (jetons squad-player-1..3 côté front).
const MaxTrackedSquadPlayers = 3

// ResolveScopeFriends — les amis configurés qui ont été MES ALLIÉS dans AU MOINS
// UN match du scope, classés par nombre de matchs partagés décroissant (gamertag
// croissant à égalité), plafonnés à MaxTrackedSquadPlayers.
//
// L'UNION, PAS L'INTERSECTION : sur un scope de période, aucun allié n'est présent dans tous les
// matchs — exiger cette présence continue rendrait toujours vide. Les lignes s'attribuent match par
// match, joueur par joueur : l'union garde des parts exclusives et exhaustives.
//
// SANS AMI CONFIGURÉ, AUCUN AMI, et c'est délibéré : retenir les trois alliés les plus fréquents
// nommerait « mes amis » des inconnus rencontrés en file d'attente.
func ResolveScopeFriends(
	playerXUID string, participants []ParticipantRow, friendGamertags []string,
) []domain.SessionUsageSquadPlayer {
	friendSet := lowerSet(friendGamertags)
	if friendSet == nil {
		return nil
	}
	shared := map[string]int{}
	byXUID := map[string]string{}
	for _, allies := range alliesOf(playerXUID, participants) {
		for xuid, gt := range allies {
			if _, ok := friendSet[strings.ToLower(gt)]; !ok {
				continue
			}
			shared[xuid]++
			byXUID[xuid] = gt
		}
	}
	out := make([]domain.SessionUsageSquadPlayer, 0, len(shared))
	for xuid := range shared {
		out = append(out, domain.SessionUsageSquadPlayer{XUID: xuid, Gamertag: byXUID[xuid]})
	}
	sort.Slice(out, func(a, b int) bool {
		if shared[out[a].XUID] != shared[out[b].XUID] {
			return shared[out[a].XUID] > shared[out[b].XUID]
		}
		if !strings.EqualFold(out[a].Gamertag, out[b].Gamertag) {
			return strings.ToLower(out[a].Gamertag) < strings.ToLower(out[b].Gamertag)
		}
		return out[a].XUID < out[b].XUID
	})
	if len(out) > MaxTrackedSquadPlayers {
		out = out[:MaxTrackedSquadPlayers]
	}
	return out
}

// alliesOf — par match, les alliés du joueur (même camp, hors lui-même), map
// xuid -> gamertag. Un match où le camp du joueur est inconnu n'a pas d'allié.
func alliesOf(playerXUID string, participants []ParticipantRow) map[string]map[string]string {
	teamByMatch := map[string]int{}
	for i := range participants {
		p := &participants[i]
		if p.XUID == playerXUID && p.TeamID != nil {
			teamByMatch[p.MatchID] = *p.TeamID
		}
	}
	out := map[string]map[string]string{}
	for i := range participants {
		p := &participants[i]
		team, ok := teamByMatch[p.MatchID]
		if !ok || p.XUID == playerXUID || p.TeamID == nil || *p.TeamID != team {
			continue
		}
		if out[p.MatchID] == nil {
			out[p.MatchID] = map[string]string{}
		}
		out[p.MatchID][p.XUID] = p.Gamertag
	}
	return out
}

// lowerSet — ensemble insensible à la casse, nil si aucune entrée exploitable
// (ResolveScopeFriends rend alors aucun ami).
func lowerSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			set[strings.ToLower(v)] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}
