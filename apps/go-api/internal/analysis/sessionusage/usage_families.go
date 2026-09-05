package sessionusage

// usage_families.go — les deux ventilations du bloc usage :
//   - prises de socle d'ARME nommées, par clé de famille NORMALISÉE
//     (replay.PadWeaponFamilyKey — les clés arrivent déjà normalisées de S1,
//     pad_pickups_json) ;
//   - occupations de socle de BONUS, ANONYMES au grain du match
//     (powerup_pickups_json) : total et cadence seulement, jamais de part —
//     aucun ramassage natif ne peut les nommer (§4 du handoff).

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// computePadFamilies agrège la ventilation des prises nommées par famille
// d'arme, triée par volume de lobby décroissant (clé croissante à volume égal).
//
// MÊME RÈGLE DE SCOPE que computeMetric (usage.go) : les grandeurs d'équipe se
// calculent sur le sous-ensemble des matchs mesurés à camp connu — numérateurs
// ET dénominateurs (sinon player_share_of_team dépasse 100 % sur une session
// mêlant équipe et FFA) ; sous-ensemble vide = nil, jamais un 0 inventé.
func computePadFamilies(playerXUID string, measured []MatchInput) []domain.SessionUsagePadFamily {
	type sums struct{ player, team, lobby, playerTeamScope, lobbyTeamScope float64 }
	byFam := map[string]*sums{}
	at := func(fam string) *sums {
		s := byFam[fam]
		if s == nil {
			s = &sums{}
			byFam[fam] = s
		}
		return s
	}
	teamKnown := false
	for i := range measured {
		m := &measured[i]
		if m.PlayerTeam != nil {
			teamKnown = true
		}
		for j := range m.Players {
			p := &m.Players[j]
			for fam, v := range p.PadPickupsByFamily {
				s := at(fam)
				s.lobby += float64(v)
				if p.XUID == playerXUID {
					s.player += float64(v)
				}
				if m.PlayerTeam != nil {
					s.lobbyTeamScope += float64(v)
					if p.XUID == playerXUID {
						s.playerTeamScope += float64(v)
					}
					if teamID, ok := m.TeamOf[p.XUID]; ok && teamID == *m.PlayerTeam {
						s.team += float64(v)
					}
				}
			}
		}
	}
	out := make([]domain.SessionUsagePadFamily, 0, len(byFam))
	for fam, s := range byFam {
		shares := domain.SessionUsageShares{
			PlayerTotal:           s.player,
			LobbyTotal:            s.lobby,
			PlayerShareOfLobbyPct: sharePct(s.player, s.lobby),
		}
		if teamKnown {
			team := s.team
			shares.TeamTotal = &team
			shares.TeamShareOfLobbyPct = sharePct(s.team, s.lobbyTeamScope)
			shares.PlayerShareOfTeamPct = sharePct(s.playerTeamScope, s.team)
		}
		out = append(out, domain.SessionUsagePadFamily{FamilyKey: fam, SessionUsageShares: shares})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].LobbyTotal != out[b].LobbyTotal {
			return out[a].LobbyTotal > out[b].LobbyTotal
		}
		return out[a].FamilyKey < out[b].FamilyKey
	})
	return out
}

// computePowerups agrège les occupations de socle de bonus par famille (clé
// canonique verbatim, "powerup_camo"...), triées par clé.
func computePowerups(measured []MatchInput, durationSeconds float64) []domain.SessionUsagePowerup {
	totals := map[string]int{}
	for i := range measured {
		for fam, v := range measured[i].PowerupPickups {
			totals[fam] += v
		}
	}
	fams := make([]string, 0, len(totals))
	for fam := range totals {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	out := make([]domain.SessionUsagePowerup, 0, len(fams))
	for _, fam := range fams {
		out = append(out, domain.SessionUsagePowerup{
			FamilyKey:   fam,
			Occupations: totals[fam],
			Per10Min:    per10Min(float64(totals[fam]), durationSeconds),
		})
	}
	return out
}
