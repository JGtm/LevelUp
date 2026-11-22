package squadimpact

// evenings.go — LES SESSIONS DE L'IMPACT : quelles sessions publier (SelectEvenings), avant leur
// cumul par BuildSessions. Voir le type publié, domain.SquadImpactSessions.

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
	// Outcome : l'issue du match pour le joueur principal (codes d'issue de la page).
	Outcome int
}

// Evening — une soirée : un libellé de session, ses matchs (un par identifiant), son premier
// instant.
type Evening struct {
	Label   string
	First   time.Time
	Matches []EveningMatch
}

// SelectEvenings rend les soirées à publier, de la plus ancienne à la plus récente : les
// soirées de la population affichée (current, la population de la page), chacune
// avec ses seuls matchs affichés, précédées des soirées de l'historique de la composition
// (timeline) jouées avant la première d'entre elles. Seules les
// domain.SquadImpactSessionsMax dernières sont gardées. Un match sans libellé de
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
	if n := len(out); n > domain.SquadImpactSessionsMax {
		out = out[n-domain.SquadImpactSessionsMax:]
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
