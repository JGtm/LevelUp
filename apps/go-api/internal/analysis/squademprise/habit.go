package squademprise

// habit.go — « PAR RAPPORT À D'HABITUDE » (décision D5 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// Les dix soirées précédentes de la composition, lues sur les SEULES familles de mode jouées ce
// soir-là, chaque soirée à part : notre part des prises de bonus et d'armes spéciales. Une soirée
// précédente est jouée avant le premier match du périmètre et n'en contient aucun ; elle n'est
// retenue que si au moins un de ses matchs comparables est filmé. La famille d'un match est
// résolue par le service (libellé de mode de l'historique de l'escouade) : aucune liste ici.
//
// Le compte de matchs d'une soirée vient de composition_sessions (ADR 0033), jamais d'un
// recomptage.

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// evening — une soirée de l'historique, réduite à ses matchs comparables.
type evening struct {
	label   string
	first   time.Time
	matches []Match
}

// HabitCandidates rend les identifiants des matchs comparables des soirées précédentes
// candidates : ce que le service doit lire EN PLUS du périmètre pour l'habitude.
func HabitCandidates(current, timeline []Match) []string {
	var out []string
	for _, ev := range previousEvenings(current, timeline) {
		for _, m := range ev.matches {
			out = append(out, m.MatchID)
		}
	}
	return out
}

// familiesOf — les familles de mode connues d'un ensemble de matchs.
func familiesOf(matches []Match) map[string]bool {
	out := map[string]bool{}
	for _, m := range matches {
		if m.Family != "" {
			out[m.Family] = true
		}
	}
	return out
}

// previousEvenings — les soirées précédentes candidates, dans l'ordre chronologique, réduites à
// leurs matchs comparables (aucune n'est vide).
func previousEvenings(current, timeline []Match) []evening {
	families := familiesOf(current)
	inCurrent := map[string]bool{}
	var firstCurrent time.Time
	for _, m := range current {
		inCurrent[m.MatchID] = true
		if firstCurrent.IsZero() || m.StartTime.Before(firstCurrent) {
			firstCurrent = m.StartTime
		}
	}
	var out []evening
	for _, ev := range groupEvenings(timeline) {
		if touches(ev.matches, inCurrent) || (!firstCurrent.IsZero() && !ev.first.Before(firstCurrent)) {
			continue
		}
		comparables := make([]Match, 0, len(ev.matches))
		for _, m := range ev.matches {
			if families[m.Family] {
				comparables = append(comparables, m)
			}
		}
		if len(comparables) > 0 {
			ev.matches = comparables
			out = append(out, ev)
		}
	}
	return out
}

// groupEvenings groupe par libellé de session, dans l'ordre du premier match ; un match sans
// session n'appartient à aucune soirée.
func groupEvenings(matches []Match) []evening {
	byLabel := map[string]*evening{}
	var order []*evening
	for _, m := range matches {
		if m.SessionLabel == "" {
			continue
		}
		ev := byLabel[m.SessionLabel]
		if ev == nil {
			ev = &evening{label: m.SessionLabel, first: m.StartTime}
			byLabel[m.SessionLabel] = ev
			order = append(order, ev)
		}
		ev.matches = append(ev.matches, m)
		if m.StartTime.Before(ev.first) {
			ev.first = m.StartTime
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].first.Before(order[j].first) })
	out := make([]evening, 0, len(order))
	for _, ev := range order {
		out = append(out, *ev)
	}
	return out
}

func touches(matches []Match, current map[string]bool) bool {
	for _, m := range matches {
		if current[m.MatchID] {
			return true
		}
	}
	return false
}

// buildHabit rend l'habitude : la soirée affichée, puis les soirées précédentes retenues.
func buildHabit(in *Input, ix *index, current []Match) *domain.SquadEmpriseHabit {
	set := familiesOf(current)
	families := make([]string, 0, len(set))
	for f := range set {
		families = append(families, f)
	}
	sort.Strings(families)
	h := &domain.SquadEmpriseHabit{Families: families, Previous: []domain.SquadEmpriseEvening{}}
	h.Current = eveningOf(commonLabel(current), current, ix, in.SessionMatchCounts)
	for _, ev := range previousEvenings(current, in.Timeline) {
		e := eveningOf(ev.label, ev.matches, ix, in.SessionMatchCounts)
		if e.MeasuredMatches > 0 {
			h.Previous = append(h.Previous, e)
		}
	}
	if n := len(h.Previous); n > domain.SquadEmpriseHabitMaxPrevious {
		h.Previous = h.Previous[n-domain.SquadEmpriseHabitMaxPrevious:]
	}
	return h
}

// eveningOf — notre part des prises d'une soirée, par ressource.
func eveningOf(label string, matches []Match, ix *index, counts map[string]int) domain.SquadEmpriseEvening {
	s := newSoiree()
	var first time.Time
	for _, m := range matches {
		s.add(tallyMatch(m.MatchID, ix))
		if first.IsZero() || m.StartTime.Before(first) {
			first = m.StartTime
		}
	}
	ev := domain.SquadEmpriseEvening{
		SessionLabel: label, MeasuredMatches: s.bonusMatches, Shares: []domain.SquadEmpriseShare{},
	}
	if label != "" {
		ev.MatchCount = counts[label]
	}
	if !first.IsZero() {
		ev.StartTime = first.UTC().Format(time.RFC3339)
	}
	for _, res := range []string{domain.EmpriseResourcePowerup, domain.EmpriseResourcePowerWeapon, domain.EmpriseResourceVehicle} {
		c := s.obj.total(res)
		if c.Us+c.Them > 0 {
			ev.Shares = append(ev.Shares, domain.SquadEmpriseShare{
				Resource: res, Taken: c, Share: float64(c.Us) / float64(c.Us+c.Them),
			})
		}
	}
	return ev
}

// commonLabel — le libellé de session partagé par tous les matchs, vide sinon.
func commonLabel(matches []Match) string {
	label := ""
	for _, m := range matches {
		if m.SessionLabel == "" || (label != "" && m.SessionLabel != label) {
			return ""
		}
		label = m.SessionLabel
	}
	return label
}
