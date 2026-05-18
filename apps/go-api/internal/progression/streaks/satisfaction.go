package streaks

import "time"

// satisfaction.go — prédicats de satisfaction d'un bucket par type de streak.
//
// Une fonction `IsBucketSatisfied` centralise la logique : selon le type de
// streak, on regarde si l'activité dans la fenêtre [bucketStart, bucketEnd]
// satisfait la condition (au moins 1 match, KDA moyen > seuil, etc.).

// MatchActivity est l'input minimal pour évaluer une streak : timing + stats
// pertinentes. Construit par l'orchestrateur depuis match_participants.
type MatchActivity struct {
	PlayedAt time.Time
	Stats    map[string]float64
}

// WeeklyPlayMinMatches est le seuil minimum de matchs pour satisfaire une
// streak `weekly_play`. Cf. PLAN §4.1.
const WeeklyPlayMinMatches = 5

// IsBucketSatisfied retourne true si l'activité dans le bucket
// [bucketStart, bucketEnd] satisfait la condition du type de streak.
//
// threshold n'est utilisé que pour les types perf-based (daily_perf,
// weekly_kda_threshold). Ignoré pour les types play-based.
func IsBucketSatisfied(matches []MatchActivity, bucketStart, bucketEnd time.Time, st StreakType, threshold float64) bool {
	inBucket := filterInBucket(matches, bucketStart, bucketEnd)
	switch st {
	case StreakTypeDailyPlay:
		return len(inBucket) >= 1
	case StreakTypeDailyPerf:
		return hasMatchAboveThreshold(inBucket, "kda", threshold)
	case StreakTypeWeeklyPlay:
		return len(inBucket) >= WeeklyPlayMinMatches
	case StreakTypeWeeklyKDAThreshold:
		return averageKDA(inBucket) > threshold
	default:
		return false
	}
}

// filterInBucket retourne les matchs joués dans [start, end].
func filterInBucket(matches []MatchActivity, start, end time.Time) []MatchActivity {
	out := make([]MatchActivity, 0, len(matches))
	for _, m := range matches {
		if !m.PlayedAt.Before(start) && !m.PlayedAt.After(end) {
			out = append(out, m)
		}
	}
	return out
}

// hasMatchAboveThreshold retourne true si au moins un match a metric > threshold.
func hasMatchAboveThreshold(matches []MatchActivity, metric string, threshold float64) bool {
	for _, m := range matches {
		if v, ok := m.Stats[metric]; ok && v > threshold {
			return true
		}
	}
	return false
}

// averageKDA retourne la moyenne arithmétique des KDA des matchs fournis.
// Retourne 0 si liste vide.
func averageKDA(matches []MatchActivity) float64 {
	if len(matches) == 0 {
		return 0
	}
	var sum float64
	var n int
	for _, m := range matches {
		if v, ok := m.Stats["kda"]; ok {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
