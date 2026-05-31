// session_labels.go : helper partagé pour dériver SessionLabelsList depuis
// des matchs annotés (label + start_time + is_with_friends).
//
// Utilisé par TeammatesService (synthesis matches) et MatchHistoryService
// (raw rows). Évite la duplication de la logique min/max + tri DESC.
package service

import (
	"cmp"
	"slices"
	"time"

	"levelup/go-api/internal/domain"
)

// minListedSessionMatches : une session de moins de 2 matchs n'est pas listée dans les
// pickers/navigations (pages Sessions, Solo, Escouade). Filtrage d'AFFICHAGE uniquement —
// n'affecte ni le découpage en sessions (analysis.ComputeSessions) ni l'agrégation des
// stats. Constante partagée par les trois chokepoints : BuildSessionLabelsList (ici),
// buildSessionOptions (filters_options.go) et keepMultiMatchSessionLabels
// (session_compare_service.go).
const minListedSessionMatches = 2

// SessionLabelInput est l'entrée minimale pour dériver les labels (un match).
type SessionLabelInput struct {
	Label         string
	StartTime     time.Time
	IsWithFriends bool
}

// BuildSessionLabelsList construit la liste solo/squad triée StartedAt DESC.
// Calcule les bornes min/max StartTime par session_label. Les entrées avec
// label vide sont ignorées. Les sessions de moins de minListedSessionMatches
// matchs sont exclues de la liste (cf. constante).
func BuildSessionLabelsList(inputs []SessionLabelInput) domain.SessionLabelsList {
	type bounds struct {
		startedAt time.Time
		endedAt   time.Time
		count     int
	}
	soloMap := map[string]*bounds{}
	squadMap := map[string]*bounds{}

	for _, m := range inputs {
		if m.Label == "" {
			continue
		}
		em := soloMap
		if m.IsWithFriends {
			em = squadMap
		}
		if b, ok := em[m.Label]; ok {
			b.count++
			if m.StartTime.Before(b.startedAt) {
				b.startedAt = m.StartTime
			}
			if m.StartTime.After(b.endedAt) {
				b.endedAt = m.StartTime
			}
		} else {
			em[m.Label] = &bounds{startedAt: m.StartTime, endedAt: m.StartTime, count: 1}
		}
	}

	toSlice := func(m map[string]*bounds) []domain.SessionLabelEntry {
		out := make([]domain.SessionLabelEntry, 0, len(m))
		for label, b := range m {
			if b.count < minListedSessionMatches {
				continue
			}
			out = append(out, domain.SessionLabelEntry{
				Label:     label,
				StartedAt: b.startedAt,
				EndedAt:   b.endedAt,
			})
		}
		slices.SortFunc(out, func(a, b domain.SessionLabelEntry) int {
			return cmp.Compare(b.StartedAt.Unix(), a.StartedAt.Unix())
		})
		return out
	}

	return domain.SessionLabelsList{
		Solo:  toSlice(soloMap),
		Squad: toSlice(squadMap),
	}
}
