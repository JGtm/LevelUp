package analysis

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// DebriefDelay met bout à bout les instants mesurés d'UN match (domain.MatchStepEvent) et rend
// son délai du débrief.
//
// Règles :
//   - chaque étape est retenue à sa PREMIÈRE occurrence (un même instant peut être capté deux
//     fois : deux joueurs suivis du même match, une passe rejouée) ;
//   - les citations sont écrites par joueur : est retenue la dernière écrite APRÈS les
//     dérivations (les citations calculées avant ne lisent pas le film) ; sans dérivations, la
//     dernière tout court ;
//   - le délai total va de la fin du match à la dernière de (dérivations, citations postérieures
//     aux dérivations) et n'existe que si les trois sont mesurées.
//
// Les événements d'autres matchs sont ignorés ; le premier événement fixe le match.
func DebriefDelay(events []domain.MatchStepEvent) domain.MatchDelay {
	var out domain.MatchDelay
	out.Steps = []domain.MatchDelayStep{}
	if len(events) == 0 {
		return out
	}
	out.TitleSlug, out.MatchID = events[0].TitleSlug, events[0].MatchID
	premieres, citations := regrouperEtapes(events, out.MatchID)

	end, hasEnd := premieres[domain.MatchStepEnd]
	if hasEnd {
		out.EndedAt = end.At.UTC().Format(time.RFC3339)
	}
	derivees, hasDerivees := premieres[domain.MatchStepDerivationsWritten]
	retenue, joueurs, hasCitations := citationsRetenues(citations, derivees.At, hasDerivees)
	out.CitationsPlayers = joueurs
	if hasCitations {
		premieres[domain.MatchStepCitationsWritten] = retenue
	}

	for _, step := range domain.MatchSteps {
		ev, ok := premieres[step]
		if !ok {
			continue
		}
		s := domain.MatchDelayStep{Step: step, At: ev.At.UTC().Format(time.RFC3339), Detail: ev.Detail}
		if hasEnd {
			s.OffsetSeconds = secondesDepuis(end.At, ev.At)
		}
		out.Steps = append(out.Steps, s)
	}

	if hasEnd && hasDerivees && joueurs > 0 {
		fin := derivees.At
		if retenue.At.After(fin) {
			fin = retenue.At
		}
		out.TotalSeconds = secondesDepuis(end.At, fin)
		tenu := fin.Sub(end.At) <= domain.DebriefObjective
		out.ObjectiveMet = &tenu
	}
	return out
}

// regrouperEtapes rend la première occurrence de chaque étape (hors citations) et toutes les
// citations, du match donné.
func regrouperEtapes(events []domain.MatchStepEvent, matchID string) (map[domain.MatchStep]domain.MatchStepEvent, []domain.MatchStepEvent) {
	premieres := make(map[domain.MatchStep]domain.MatchStepEvent, len(domain.MatchSteps))
	var citations []domain.MatchStepEvent
	for _, ev := range events {
		if ev.MatchID != matchID {
			continue
		}
		if ev.Step == domain.MatchStepCitationsWritten {
			citations = append(citations, ev)
			continue
		}
		if cur, ok := premieres[ev.Step]; !ok || ev.At.Before(cur.At) {
			premieres[ev.Step] = ev
		}
	}
	return premieres, citations
}

// citationsRetenues rend la citation à afficher, le nombre de joueurs distincts dont les
// citations suivent les dérivations, et si une citation est à afficher.
func citationsRetenues(citations []domain.MatchStepEvent, derivees time.Time, hasDerivees bool) (domain.MatchStepEvent, int, bool) {
	if len(citations) == 0 {
		return domain.MatchStepEvent{}, 0, false
	}
	sort.SliceStable(citations, func(i, j int) bool { return citations[i].At.Before(citations[j].At) })
	derniere := citations[len(citations)-1]
	if !hasDerivees {
		return derniere, 0, true
	}
	joueurs := map[string]bool{}
	var apres *domain.MatchStepEvent
	for i := range citations {
		if citations[i].At.Before(derivees) {
			continue
		}
		joueurs[citations[i].Detail] = true
		apres = &citations[i]
	}
	if apres == nil {
		return derniere, 0, true
	}
	return *apres, len(joueurs), true
}

// secondesDepuis rend l'écart en secondes (au dixième) entre deux instants.
func secondesDepuis(debut, fin time.Time) *float64 {
	s := float64(fin.Sub(debut).Milliseconds()/100) / 10
	return &s
}
