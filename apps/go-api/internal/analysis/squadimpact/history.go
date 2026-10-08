package squadimpact

// history.go — LES POINTS D'IMPACT PAR SOIRÉE : quelles soirées publier (SelectEvenings), puis
// le cumul des rôles de chaque joueur sur chacune (BuildHistory). Voir le type publié,
// domain.SquadImpactHistory.

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// EveningMatch — un match de la composition, tel que la page le connaît.
type EveningMatch struct {
	MatchID      string
	SessionLabel string
	StartTime    time.Time
	// Won : le joueur principal a gagné le match.
	Won bool
}

// Evening — une soirée : un libellé de session, ses matchs (un par identifiant), son premier
// instant.
type Evening struct {
	Label   string
	First   time.Time
	Matches []EveningMatch
}

// SelectEvenings rend les soirées à publier, de la plus ancienne à la plus récente : les
// soirées de la population affichée (current, la population de la matrice d'impact), chacune
// avec ses seuls matchs affichés, précédées des soirées de l'historique de la composition
// (timeline) jouées avant la première d'entre elles. Seules les
// domain.SquadImpactHistoryMaxEvenings dernières sont gardées. Un match sans libellé de
// session n'appartient à aucune soirée. Deux sessions du même jour restent deux soirées.
func SelectEvenings(current, timeline []EveningMatch) []Evening {
	shown := groupEvenings(current)
	if len(shown) == 0 {
		return nil
	}
	shownLabels := make(map[string]bool, len(shown))
	for _, e := range shown {
		shownLabels[e.Label] = true
	}
	firstShown := shown[0].First
	out := make([]Evening, 0, len(shown))
	for _, e := range groupEvenings(timeline) {
		// UNE SOIRÉE PRÉCÉDENTE EST JOUÉE AVANT LA SOIRÉE AFFICHÉE : une population filtrée sur
		// une soirée ancienne ne se compare pas à celles qui l'ont suivie.
		if shownLabels[e.Label] || !e.First.Before(firstShown) {
			continue
		}
		out = append(out, e)
	}
	out = append(out, shown...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].First.Before(out[j].First) })
	if n := len(out); n > domain.SquadImpactHistoryMaxEvenings {
		out = out[n-domain.SquadImpactHistoryMaxEvenings:]
	}
	return out
}

// MatchIDs rend les identifiants des matchs des soirées.
func MatchIDs(evenings []Evening) []string {
	var out []string
	for _, e := range evenings {
		for _, m := range e.Matches {
			out = append(out, m.MatchID)
		}
	}
	return out
}

// groupEvenings groupe les matchs par libellé de session, dans l'ordre chronologique du premier
// match ; un identifiant de match répété ne compte qu'une fois.
func groupEvenings(matches []EveningMatch) []Evening {
	byLabel := map[string]*Evening{}
	seen := map[string]bool{}
	var order []*Evening
	for _, m := range matches {
		if m.SessionLabel == "" || seen[m.MatchID] {
			continue
		}
		seen[m.MatchID] = true
		e := byLabel[m.SessionLabel]
		if e == nil {
			e = &Evening{Label: m.SessionLabel, First: m.StartTime}
			byLabel[m.SessionLabel] = e
			order = append(order, e)
		}
		e.Matches = append(e.Matches, m)
		if m.StartTime.Before(e.First) {
			e.First = m.StartTime
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].First.Before(order[j].First) })
	out := make([]Evening, 0, len(order))
	for _, e := range order {
		out = append(out, *e)
	}
	return out
}

// HistoryInput — tout ce que le cumul demande.
type HistoryInput struct {
	Evenings []Evening
	// Players : les joueurs de l'escouade, dans l'ordre des barres d'une soirée.
	Players []string
	// Roles : par identifiant de match, les rôles du match tombés sur l'escouade (RolesOfMatch).
	// Un match absent n'a attribué aucun rôle.
	Roles map[string][]Attribution
}

// BuildHistory rend le bloc publié. Nil quand il n'y a ni soirée ni joueur, ou quand aucune
// soirée n'attribue un seul rôle à l'escouade (titre sans événements horodatés ni équipe
// alliée : rien à montrer).
func BuildHistory(in HistoryInput) *domain.SquadImpactHistory {
	if len(in.Evenings) == 0 || len(in.Players) == 0 {
		return nil
	}
	out := &domain.SquadImpactHistory{
		Players:  append([]string(nil), in.Players...),
		Scale:    scale(),
		Evenings: make([]domain.SquadImpactEvening, 0, len(in.Evenings)),
	}
	anyRole := false
	for _, e := range in.Evenings {
		ev, roles := buildEvening(e, in.Players, in.Roles)
		anyRole = anyRole || roles
		out.Evenings = append(out.Evenings, ev)
	}
	if !anyRole {
		return nil
	}
	return out
}

// buildEvening cumule les rôles de chaque joueur sur une soirée ; le booléen dit si un rôle au
// moins y tombe sur un joueur publié.
func buildEvening(e Evening, players []string, roles map[string][]Attribution) (domain.SquadImpactEvening, bool) {
	counts := make(map[string]map[string]int, len(players))
	for _, p := range players {
		counts[p] = map[string]int{}
	}
	ev := domain.SquadImpactEvening{
		SessionLabel: e.Label,
		StartTime:    e.First.UTC().Format(time.RFC3339),
		Matches:      len(e.Matches),
		Players:      make([]domain.SquadImpactEveningPlayer, 0, len(players)),
	}
	anyRole := false
	for _, m := range e.Matches {
		if m.Won {
			ev.Wins++
		}
		for _, a := range roles[m.MatchID] {
			if c, ok := counts[a.Player]; ok {
				c[a.Role]++
				anyRole = true
			}
		}
	}
	for _, p := range players {
		ev.Players = append(ev.Players, playerEvening(p, counts[p]))
	}
	return ev, anyRole
}

// playerEvening — les rôles d'un joueur dans l'ordre du barème, et leur net.
func playerEvening(player string, counts map[string]int) domain.SquadImpactEveningPlayer {
	out := domain.SquadImpactEveningPlayer{Player: player, Roles: []domain.SquadImpactRoleCount{}}
	for _, role := range stackOrder {
		n := counts[role]
		if n == 0 {
			continue
		}
		pts := float64(n) * weights[role]
		out.Roles = append(out.Roles, domain.SquadImpactRoleCount{Role: role, Count: n, Points: pts})
		out.Points += pts
	}
	return out
}

// scale — le barème publié, dans l'ordre d'empilement.
func scale() []domain.SquadImpactRoleWeight {
	out := make([]domain.SquadImpactRoleWeight, 0, len(stackOrder))
	for _, role := range stackOrder {
		out = append(out, domain.SquadImpactRoleWeight{Role: role, Points: weights[role]})
	}
	return out
}
