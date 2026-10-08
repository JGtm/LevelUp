package sessionusage

// squad.go — LES MEMBRES D'UN SCOPE DE PÉRIODE : les membres désignés par la page qui ont été mes
// alliés (même camp, hors moi) dans au moins un match du scope, et la machine d'alliés qui les
// trouve.

import (
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// MaxTrackedSquadPlayers borne les lignes d'escouade — même plafond que la
// composition de la page Escouade (jetons squad-player-1..3 côté front).
const MaxTrackedSquadPlayers = 3

// ResolveScopeMembers — les membres désignés (`members`, la sélection de la page) qui ont été MES
// ALLIÉS dans AU MOINS UN match du scope, classés par nombre de matchs partagés décroissant (nom
// croissant à égalité, puis xuid), plafonnés à MaxTrackedSquadPlayers.
//
// L'APPARTENANCE SE DÉCIDE PAR XUID (ADR 0035 : le xuid est la clé d'identité, le gamertag un
// affichage), jamais par égalité de nom : `match_participants.gamertag` peut être vide sur tout un
// scope, et le membre disparaissait alors des blocs. Le nom affiché est celui que la page donne au
// membre ; le gamertag des participants n'est que son repli.
//
// L'UNION, PAS L'INTERSECTION : sur un scope de période, aucun allié n'est présent dans tous les
// matchs — exiger cette présence continue rendrait toujours vide. Les lignes s'attribuent match par
// match, joueur par joueur : l'union garde des parts exclusives et exhaustives.
//
// SANS MEMBRE DÉSIGNÉ, AUCUN MEMBRE, et c'est délibéré : retenir les trois alliés les plus
// fréquents nommerait « l'escouade » des inconnus rencontrés en file d'attente.
func ResolveScopeMembers(
	playerXUID string, participants []ParticipantRow, members []domain.SessionUsageSquadPlayer,
) []domain.SessionUsageSquadPlayer {
	nomDe := map[string]string{} // xuid désigné -> nom donné par la page
	for _, m := range members {
		if _, deja := nomDe[m.XUID]; m.XUID != "" && !deja {
			nomDe[m.XUID] = strings.TrimSpace(m.Gamertag)
		}
	}
	if len(nomDe) == 0 {
		return nil
	}
	shared := map[string]int{}
	repli := map[string]string{} // xuid -> gamertag non vide des participants
	for _, allies := range alliesOf(playerXUID, participants) {
		for xuid, gt := range allies {
			if _, ok := nomDe[xuid]; !ok {
				continue
			}
			shared[xuid]++
			if repli[xuid] == "" {
				repli[xuid] = gt
			}
		}
	}
	out := make([]domain.SessionUsageSquadPlayer, 0, len(shared))
	for xuid := range shared {
		nom := nomDe[xuid]
		if nom == "" {
			nom = repli[xuid]
		}
		out = append(out, domain.SessionUsageSquadPlayer{XUID: xuid, Gamertag: nom})
	}
	trierParPartage(out, shared)
	if len(out) > MaxTrackedSquadPlayers {
		out = out[:MaxTrackedSquadPlayers]
	}
	return out
}

// trierParPartage — matchs partagés décroissants, puis nom (sans casse), puis xuid : un ordre
// total, l'ordre d'itération d'une map n'en étant pas un.
func trierParPartage(out []domain.SessionUsageSquadPlayer, shared map[string]int) {
	sort.Slice(out, func(a, b int) bool {
		if shared[out[a].XUID] != shared[out[b].XUID] {
			return shared[out[a].XUID] > shared[out[b].XUID]
		}
		if !strings.EqualFold(out[a].Gamertag, out[b].Gamertag) {
			return strings.ToLower(out[a].Gamertag) < strings.ToLower(out[b].Gamertag)
		}
		return out[a].XUID < out[b].XUID
	})
}

// alliesOf — par match, les alliés du joueur (même camp, hors lui-même), map
// xuid -> gamertag des participants (vide quand la ligne n'en porte pas). Un
// match où le camp du joueur est inconnu n'a pas d'allié.
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
