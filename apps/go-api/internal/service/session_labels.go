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

// SessionLabelInput est l'entrée minimale pour dériver les labels (un match).
type SessionLabelInput struct {
	Label         string
	StartTime     time.Time
	IsWithFriends bool
}

// BuildSessionLabelsList construit la liste solo/squad triée StartedAt DESC.
// Calcule les bornes min/max StartTime par session_label. Les entrées avec
// label vide sont ignorées.
func BuildSessionLabelsList(inputs []SessionLabelInput) domain.SessionLabelsList {
	type bounds struct {
		startedAt time.Time
		endedAt   time.Time
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
			if m.StartTime.Before(b.startedAt) {
				b.startedAt = m.StartTime
			}
			if m.StartTime.After(b.endedAt) {
				b.endedAt = m.StartTime
			}
		} else {
			em[m.Label] = &bounds{startedAt: m.StartTime, endedAt: m.StartTime}
		}
	}

	toSlice := func(m map[string]*bounds) []domain.SessionLabelEntry {
		out := make([]domain.SessionLabelEntry, 0, len(m))
		for label, b := range m {
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
