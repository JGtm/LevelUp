package trends

import (
	"time"

	"levelup/go-api/internal/domain"
)

// SquadSample porte ce que l'équipe alliée du joueur a fait dans un match :
// TeamKills est la somme des frags de l'équipe (le joueur principal compris),
// Members le bilan de chaque allié, par xuid.
type SquadSample struct {
	TeamKills int
	Members   map[string]MemberStat
}

// MemberStat est le bilan d'un joueur dans un match.
type MemberStat struct {
	Kills, Deaths, Assists int
}

// SquadSamples construit l'échantillon d'escouade de chaque match (clé =
// identifiant de match) à partir des participants de l'équipe alliée. Les lignes
// sans identifiant de match sont ignorées ; sans xuid, elles comptent dans les
// frags de l'équipe mais n'ont pas d'entrée dans Members.
func SquadSamples(allies []domain.AllyParticipant) map[string]SquadSample {
	out := map[string]SquadSample{}
	for i := range allies {
		a := &allies[i]
		if a.MatchID == "" {
			continue
		}
		s, ok := out[a.MatchID]
		if !ok {
			s = SquadSample{Members: map[string]MemberStat{}}
		}
		s.TeamKills += a.Kills
		if a.XUID != "" {
			st := s.Members[a.XUID]
			st.Kills += a.Kills
			st.Deaths += a.Deaths
			st.Assists += a.Assists
			s.Members[a.XUID] = st
		}
		out[a.MatchID] = s
	}
	return out
}

// AttachSquad pose les échantillons d'escouade sur les matchs, par identifiant.
func AttachSquad(matches []Match, samples map[string]SquadSample) {
	for i := range matches {
		if s, ok := samples[matches[i].ID]; ok {
			s := s
			matches[i].Squad = &s
		}
	}
}

// SquadMember est un membre de la composition : le joueur principal d'abord,
// puis les coéquipiers dans l'ordre demandé. XUID est la clé de
// SquadSample.Members ; Gamertag nomme les lignes de la matrice.
type SquadMember struct {
	XUID, Gamertag string
}

// SquadOptions paramètre BuildSquad.
type SquadOptions struct {
	Options
	// Members : le joueur principal, puis les coéquipiers.
	Members []SquadMember
	// Alone : les matchs du joueur principal sans amis (taux de victoire « sans
	// l'escouade »), triés par heure croissante.
	Alone []Match
}

// BuildSquad calcule la réponse de la vue Escouade à partir des matchs de la
// composition, triés par heure croissante (les matchs postérieurs à Now sont
// ignorés). Il remplit AsOf, GameType, Timezone, Months, GameTypes (types joués
// par la composition, sans filtre) et Indicators ; Calendar, WinLoss, Medals et
// Mix restent vides. Le filtre GameType s'applique aux matchs de la composition
// et à Alone ; View et Capabilities restent au service.
func BuildSquad(matches []Match, opts SquadOptions) domain.TrendsPageResponse {
	loc := opts.Loc
	if loc == nil {
		loc = time.UTC
	}
	all := upTo(matches, opts.Now)
	f := newFrame(filterChain(all, opts.GameType), opts.Now, loc)
	fAlone := newFrame(filterChain(upTo(opts.Alone, opts.Now), opts.GameType), opts.Now, loc)

	inds := buildIndicators([]indicator{squadWinRate(domain.TrendsKeyWinRate)}, f)
	inds = append(inds, buildIndicators([]indicator{squadWinRate(domain.TrendsKeyWinRateAlone)}, fAlone)...)
	inds = append(inds, buildIndicators(squadRows(opts.Members), f)...)
	members := make([]domain.TrendsMember, 0, len(opts.Members))
	for _, m := range opts.Members {
		members = append(members, domain.TrendsMember{XUID: m.XUID, Gamertag: m.Gamertag})
	}
	return domain.TrendsPageResponse{
		AsOf:       opts.Now,
		GameType:   opts.GameType,
		Timezone:   loc.String(),
		GameTypes:  buildGameTypes(all, opts.Now),
		Months:     f.monthKeys(),
		Indicators: inds,
		Members:    members,
		Calendar:   []domain.TrendsCalendarDay{},
		WinLoss:    []domain.TrendsWinLossBlock{},
		Medals:     []domain.TrendsMedalsBlock{},
		Mix: domain.TrendsMix{
			Day: []domain.TrendsMixBucket{}, Week: []domain.TrendsMixBucket{}, Month: []domain.TrendsMixBucket{},
		},
	}
}
