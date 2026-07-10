// Package analysis â€” home_sessions.go : rÃ©sumÃ©s de sessions (BuildSessionSummaries
// et BuildSessionSummary) + helpers de tri par label/temps.
package analysis

import (
	"sort"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------

// distinctSessionLabels retourne les labels distincts triÃ©s par start_time DESC.
func distinctSessionLabels(sessions []legacymatch.HomeSessionRow) []string {
	// Calculer le start_time max par label.
	labelTimes := make(map[string]time.Time)
	for _, s := range sessions {
		if s.SessionLabel == nil || *s.SessionLabel == "" {
			continue
		}
		lbl := *s.SessionLabel
		if s.StartTime != nil {
			if t, ok := labelTimes[lbl]; !ok || s.StartTime.After(t) {
				labelTimes[lbl] = *s.StartTime
			}
		} else {
			if _, ok := labelTimes[lbl]; !ok {
				labelTimes[lbl] = time.Time{}
			}
		}
	}
	labels := make([]string, 0, len(labelTimes))
	for lbl := range labelTimes {
		labels = append(labels, lbl)
	}
	sort.Slice(labels, func(i, j int) bool {
		return labelTimes[labels[i]].After(labelTimes[labels[j]])
	})
	return labels
}

func latestSessionLabel(sessions []legacymatch.HomeSessionRow) string {
	// Trier par start_time DESC, prendre le premier session_label.
	sorted := make([]legacymatch.HomeSessionRow, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartTime == nil {
			return false
		}
		if sorted[j].StartTime == nil {
			return true
		}
		return sorted[i].StartTime.After(*sorted[j].StartTime)
	})
	for _, s := range sorted {
		if s.SessionLabel != nil && *s.SessionLabel != "" {
			return *s.SessionLabel
		}
	}
	return ""
}

func earliestStartTime(matches []legacymatch.HomeMatchRow) *time.Time {
	var earliest *time.Time
	for i := range matches {
		t := matches[i].StartTime
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}

// latestEndTime retourne l'heure de fin estimÃ©e du dernier match de la session.
func latestEndTime(matches []legacymatch.HomeMatchRow) *time.Time {
	var latest *legacymatch.HomeMatchRow
	for i := range matches {
		if latest == nil || matches[i].StartTime.After(latest.StartTime) {
			latest = &matches[i]
		}
	}
	if latest == nil {
		return nil
	}
	if latest.TimePlayedSecs != nil && *latest.TimePlayedSecs > 0 {
		t := latest.StartTime.Add(time.Duration(*latest.TimePlayedSecs) * time.Second)
		return &t
	}
	t := latest.StartTime
	return &t
}
