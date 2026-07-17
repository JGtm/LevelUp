// Package analysis — longest_run.go : helper title-agnostic « plus longue série
// consécutive » (streaks victoires/défaites, tilt, etc.).
//
// Centralise le balayage best/cur qui avait été recopié dans quatre call-sites
// (service.longestOutcomeRun, analysis.sliceBestWinStreakCanonical,
// service.buildSynthesisOverviewCanonical winStreak/maxStreak, patterns.detectTilt)
// — CLAUDE.md n°6. Garde-rail anti-divergence :
// internal/archlint/no_local_longest_run_test.go.
package analysis

// LongestRun retourne la longueur de la plus longue séquence d'éléments
// CONSÉCUTIFS de items satisfaisant pred, ainsi que l'index de DÉPART de cette
// séquence dans items. Une séquence est rompue par tout élément ne satisfaisant
// pas pred. En cas d'égalité de longueur, la PREMIÈRE séquence l'emporte (start
// n'est mis à jour que sur amélioration STRICTE de la longueur). Si aucun élément
// ne satisfait pred, retourne (0, 0).
func LongestRun[T any](items []T, pred func(T) bool) (length, start int) {
	best, bestStart := 0, 0
	cur, curStart := 0, 0
	for i := range items {
		if pred(items[i]) {
			if cur == 0 {
				curStart = i
			}
			cur++
			if cur > best {
				best = cur
				bestStart = curStart
			}
		} else {
			cur = 0
		}
	}
	return best, bestStart
}
