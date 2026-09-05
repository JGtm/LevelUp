package sessionusage

// squad.go — le contexte ESCOUADE du bloc usage : résolution des coéquipiers
// suivis d'une session, et lignes par joueur suivi sur chaque grandeur (la
// « piste du lobby découpée par joueur » des seize formes en dépend).
//
// LA RÈGLE DE RÉSOLUTION EST CELLE DE L'ACCUEIL (sessionCoreTeammates,
// service/home_squad_session_teammates.go) : un coéquipier suivi est un ALLIÉ
// (même camp que le joueur de la route) présent dans TOUS les matchs de la
// session, restreint aux amis configurés quand il y en a, cappé et trié
// alphabétiquement. L'intersection est volontaire : la composition affichée
// reste EXACTE sur toute la session — un ami qui n'a joué qu'une partie des
// matchs fausserait toutes les parts de session de sa ligne.

import (
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// MaxTrackedSquadPlayers borne les lignes d'escouade — même plafond que la
// composition de la page Escouade (jetons squad-player-1..3 côté front).
const MaxTrackedSquadPlayers = 3

// ResolveTrackedSquad dérive les coéquipiers suivis d'une session : alliés du
// joueur de la route présents dans TOUS les matchs du scope, restreints à
// friendGamertags (clé insensible à la casse ; vide = aucune restriction),
// plafonnés à MaxTrackedSquadPlayers, ordre alphabétique de gamertag. Vide dès
// qu'un match n'a pas d'allié identifiable (camp du joueur inconnu — FFA).
func ResolveTrackedSquad(
	playerXUID string,
	matchIDs []string,
	participants []ParticipantRow,
	friendGamertags []string,
) []domain.SessionUsageSquadPlayer {
	if len(matchIDs) == 0 {
		return nil
	}
	friendSet := lowerSet(friendGamertags)
	alliesByMatch := alliesOf(playerXUID, participants)
	// Intersection : présent (comme allié) dans chacun des matchs du scope.
	present := map[string]int{}
	byXUID := map[string]string{}
	for _, mid := range matchIDs {
		for xuid, gt := range alliesByMatch[mid] {
			if friendSet != nil {
				if _, ok := friendSet[strings.ToLower(gt)]; !ok {
					continue
				}
			}
			present[xuid]++
			byXUID[xuid] = gt
		}
	}
	out := make([]domain.SessionUsageSquadPlayer, 0, MaxTrackedSquadPlayers)
	for xuid, n := range present {
		if n == len(matchIDs) {
			out = append(out, domain.SessionUsageSquadPlayer{XUID: xuid, Gamertag: byXUID[xuid]})
		}
	}
	sort.Slice(out, func(a, b int) bool {
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
// (= aucune restriction, même convention que configuredFriendSet côté Home).
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

// appendSquadLines pose sur la métrique la ligne de chaque coéquipier suivi :
// total (dénominateur d'honnêteté), parts contre les mêmes dénominateurs ET les
// mêmes scopes que le joueur de la route (part d'équipe sur les matchs à camp
// connu, cadence sur les matchs à durée connue — règle de scope de
// computeMetric), cadence sur la même durée mesurée (durAll).
func appendSquadLines(
	m *domain.SessionUsageMetric, measured []MatchInput, squadXUIDs []string, durAll float64,
) {
	if len(squadXUIDs) == 0 {
		return
	}
	var teamTotal float64
	if m.TeamTotal != nil {
		teamTotal = *m.TeamTotal
	}
	tracked := map[string]bool{}
	for _, x := range squadXUIDs {
		tracked[x] = true
	}
	totals := map[string]float64{}    // tout le scope mesuré (Total, part lobby)
	teamScope := map[string]float64{} // matchs à camp connu (part d'équipe)
	durKnown := map[string]float64{}  // matchs à durée connue (cadence)
	for i := range measured {
		mi := &measured[i]
		for j := range mi.Players {
			p := &mi.Players[j]
			if !tracked[p.XUID] {
				continue
			}
			v := float64(metricValue(m.Key, p))
			totals[p.XUID] += v
			if mi.PlayerTeam != nil {
				teamScope[p.XUID] += v
			}
			if mi.DurationSeconds > 0 {
				durKnown[p.XUID] += v
			}
		}
	}
	for _, x := range squadXUIDs {
		m.Squad = append(m.Squad, domain.SessionUsageSquadShare{
			XUID:            x,
			Total:           totals[x],
			ShareOfTeamPct:  sharePct(teamScope[x], teamTotal),
			ShareOfLobbyPct: sharePct(totals[x], m.LobbyTotal),
			Per10Min:        per10Min(durKnown[x], durAll),
		})
	}
}
