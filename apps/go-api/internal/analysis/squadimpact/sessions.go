package squadimpact

// sessions.go — L'IMPACT DES COÉQUIPIERS PAR SESSION : le cumul des rôles de chaque joueur sur
// chacune des sessions publiées (SelectEvenings) et sur leur ensemble. Voir le type publié,
// domain.SquadImpactSessions. Les rôles viennent de RolesOfMatch et les points du barème de
// Weight : une seule définition.

import (
	"slices"
	"time"

	"levelup/go-api/internal/domain"
)

// SessionsInput — tout ce que le calcul des sessions demande.
type SessionsInput struct {
	// Evenings : les sessions publiées, de la plus ancienne à la plus récente (SelectEvenings).
	Evenings []Evening
	// Players : les joueurs de l'escouade, dans l'ordre des colonnes du tableau.
	Players []string
	// Roles : par identifiant de match, les rôles tombés sur l'escouade (RolesOfMatch). Un match
	// absent n'a attribué aucun rôle.
	Roles map[string][]Attribution
	// Allied : par identifiant de match, les joueurs de l'escouade présents dans l'équipe alliée du
	// joueur principal. Un joueur a joué un match quand il y figure ; un rôle attribué dans un match
	// vaut présence (un rôle ne tombe que sur un joueur du match).
	Allied map[string]map[string]bool
}

// BuildSessions rend le bloc publié : les sessions de la plus récente à la plus ancienne, leurs
// matchs du plus ancien au plus récent, et les totaux sommés sur les seules sessions publiées. Nil
// quand il n'y a ni session ni joueur, ou quand aucune session n'attribue un seul rôle à
// l'escouade (titre sans événements horodatés ni équipe alliée : rien à montrer).
func BuildSessions(in SessionsInput) *domain.SquadImpactSessions {
	if len(in.Evenings) == 0 || len(in.Players) == 0 {
		return nil
	}
	out := &domain.SquadImpactSessions{
		Players:   slices.Clone(in.Players),
		Scale:     sessionsScale(),
		RoleOrder: RoleOrder(),
		Sessions:  make([]domain.SquadImpactSessionsSession, 0, len(in.Evenings)),
	}
	totals := make([]*playerTally, len(in.Players))
	for i := range totals {
		totals[i] = newPlayerTally()
	}
	anyRole := false
	for i := len(in.Evenings) - 1; i >= 0; i-- {
		session, tallies := buildSession(in.Evenings[i], in)
		for j, t := range tallies {
			anyRole = anyRole || len(t.perMatch) > 0
			totals[j].add(t)
		}
		out.Sessions = append(out.Sessions, session)
	}
	if !anyRole {
		return nil
	}
	out.Totals = make([]domain.SquadImpactSessionsTotal, 0, len(in.Players))
	for j, p := range in.Players {
		out.Totals = append(out.Totals, domain.SquadImpactSessionsTotal{
			Player: p, MatchesPlayed: totals[j].played, Points: totals[j].points, Counts: totals[j].countList(),
		})
	}
	return out
}

// playerTally — le cumul d'un joueur sur une session ou sur l'ensemble.
type playerTally struct {
	played   int
	points   float64
	counts   map[string]int
	perMatch []domain.SquadImpactSessionsPerMatch
}

func newPlayerTally() *playerTally { return &playerTally{counts: map[string]int{}} }

// add somme les comptes, les points et les matchs joués de t ; les rôles match par match ne se
// somment pas (ils restent propres à leur session).
func (t *playerTally) add(o *playerTally) {
	t.played += o.played
	t.points += o.points
	for role, n := range o.counts {
		t.counts[role] += n
	}
}

// countList rend le compte de chaque rôle dans l'ordre de RoleOrder, comptes nuls compris.
func (t *playerTally) countList() []domain.SquadImpactSessionsRoleCount {
	out := make([]domain.SquadImpactSessionsRoleCount, 0, len(roleOrder))
	for _, role := range roleOrder {
		out = append(out, domain.SquadImpactSessionsRoleCount{Role: role, Count: t.counts[role]})
	}
	return out
}

// buildSession cumule une session : ses matchs du plus ancien au plus récent, puis une entrée par
// joueur. Rend aussi les cumuls bruts, que BuildSessions somme en totaux.
func buildSession(e Evening, in SessionsInput) (domain.SquadImpactSessionsSession, []*playerTally) {
	matches := slices.Clone(e.Matches)
	slices.SortStableFunc(matches, func(a, b EveningMatch) int { return a.StartTime.Compare(b.StartTime) })
	session := domain.SquadImpactSessionsSession{
		SessionLabel: e.Label,
		StartTime:    e.First.UTC().Format(time.RFC3339),
		Matches:      make([]domain.SquadImpactSessionsMatch, 0, len(matches)),
		Players:      make([]domain.SquadImpactSessionsPlayer, 0, len(in.Players)),
	}
	tallies := make([]*playerTally, len(in.Players))
	for i := range tallies {
		tallies[i] = newPlayerTally()
	}
	for _, m := range matches {
		session.Matches = append(session.Matches, domain.SquadImpactSessionsMatch{MatchID: m.MatchID, Outcome: m.Outcome})
		byPlayer := rolesByPlayer(in.Roles[m.MatchID])
		for i, p := range in.Players {
			tallies[i].addMatch(m.MatchID, byPlayer[p], in.Allied[m.MatchID][p])
		}
	}
	for i, p := range in.Players {
		session.Players = append(session.Players, domain.SquadImpactSessionsPlayer{
			Player: p, MatchesPlayed: tallies[i].played, Points: tallies[i].points,
			Counts: tallies[i].countList(), PerMatch: nonNilPerMatch(tallies[i].perMatch),
		})
	}
	return session, tallies
}

// addMatch cumule un match : sa présence dans l'équipe alliée (ou un rôle y valant présence), puis
// ses rôles dans l'ordre de RoleOrder.
func (t *playerTally) addMatch(matchID string, roles []string, allied bool) {
	if allied || len(roles) > 0 {
		t.played++
	}
	if len(roles) == 0 {
		return
	}
	roles = slices.Clone(roles)
	slices.SortStableFunc(roles, func(a, b string) int { return slices.Index(roleOrder, a) - slices.Index(roleOrder, b) })
	for _, role := range roles {
		t.counts[role]++
		t.points += weights[role]
	}
	t.perMatch = append(t.perMatch, domain.SquadImpactSessionsPerMatch{MatchID: matchID, Roles: roles})
}

// rolesByPlayer regroupe les rôles d'un match par joueur.
func rolesByPlayer(attributions []Attribution) map[string][]string {
	out := make(map[string][]string, len(attributions))
	for _, a := range attributions {
		out[a.Player] = append(out[a.Player], a.Role)
	}
	return out
}

// nonNilPerMatch rend une liste vide plutôt que nil : le contrat sert `[]`, jamais `null`.
func nonNilPerMatch(l []domain.SquadImpactSessionsPerMatch) []domain.SquadImpactSessionsPerMatch {
	if l == nil {
		return []domain.SquadImpactSessionsPerMatch{}
	}
	return l
}

// sessionsScale rend le barème publié, dans l'ordre d'empilement.
func sessionsScale() []domain.SquadImpactSessionsRoleWeight {
	out := make([]domain.SquadImpactSessionsRoleWeight, 0, len(stackOrder))
	for _, role := range stackOrder {
		out = append(out, domain.SquadImpactSessionsRoleWeight{Role: role, Points: weights[role]})
	}
	return out
}
