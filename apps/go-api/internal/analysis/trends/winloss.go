package trends

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// winLossRequired : valeurs requises (matchs gagnés ou perdus) pour afficher le bloc d'un horizon.
const winLossRequired = 30

// winLossStat est une statistique par match du bloc « Victoires et défaites ».
type winLossStat struct {
	key, group string
	value      func(m Match) (float64, bool)
}

// always adapte une valeur toujours définie.
func always(get func(Match) float64) func(Match) (float64, bool) {
	return func(m Match) (float64, bool) { return get(m), true }
}

// winLossStats retourne les statistiques comparées, avant tri. hp est le seuil
// de dégâts par frag.
func winLossStats(hp float64) []winLossStat {
	st, co, cx := domain.TrendsWinLossStats, domain.TrendsWinLossComposite, domain.TrendsWinLossContext
	return []winLossStat{
		{domain.TrendsKeyAvgLifeSeconds, st, optVal(func(m Match) *float64 { return m.AvgLife })},
		{domain.TrendsKeyDamageBalance, st, optVal(damageBalance)},
		{domain.TrendsKeyDeaths, st, always(func(m Match) float64 { return float64(m.Deaths) })},
		{domain.TrendsKeyKills, st, always(func(m Match) float64 { return float64(m.Kills) })},
		{domain.TrendsKeyAssists, st, always(func(m Match) float64 { return float64(m.Assists) })},
		{domain.TrendsKeyAccuracy, st, matchAccuracy},
		{domain.TrendsKeyHeadshotKills, st, always(func(m Match) float64 { return float64(m.HeadshotKills) })},
		{domain.TrendsKeyMaxKillingSpree, st, optVal(func(m Match) *float64 { return m.MaxSpree })},
		{domain.TrendsKeyPerformanceScore, co, optVal(func(m Match) *float64 { return m.PerformanceScore })},
		{domain.TrendsKeyKDA, co, optVal(func(m Match) *float64 { return m.KDA })},
		{domain.TrendsKeyOffensiveConversion, co, yieldOne(hp, true)},
		{domain.TrendsKeyDefensiveResistance, co, yieldOne(hp, false)},
		{domain.TrendsKeyMMRGap, cx, mmrGap},
	}
}

func optVal(get func(Match) *float64) func(Match) (float64, bool) {
	return func(m Match) (float64, bool) {
		if v := get(m); v != nil {
			return *v, true
		}
		return 0, false
	}
}

// matchAccuracy : précision d'un match, absente sans tir.
func matchAccuracy(m Match) (float64, bool) {
	if m.ShotsFired <= 0 {
		return 0, false
	}
	return analysis.Accuracy(m.ShotsHit, m.ShotsFired), true
}

// mmrGap : MMR de l'équipe moins MMR adverse, absent si l'un des deux manque.
func mmrGap(m Match) (float64, bool) {
	if m.TeamMMR == nil || m.EnemyMMR == nil {
		return 0, false
	}
	return *m.TeamMMR - *m.EnemyMMR, true
}

// yieldOne : rendement offensif ou résistance défensive d'un seul match.
func yieldOne(hp float64, offensive bool) func(Match) (float64, bool) {
	ev := yieldEval(hp, offensive)
	return func(m Match) (float64, bool) { return ev([]Match{m}) }
}

// buildWinLoss retourne un bloc par horizon (365, 90, 30, 7). La population est
// l'ensemble des matchs gagnés ou perdus de l'horizon ; sous 30, Rows est vide.
func buildWinLoss(f *frame, hp float64) []domain.TrendsWinLossBlock {
	stats := winLossStats(hp)
	out := make([]domain.TrendsWinLossBlock, 0, len(f.hor))
	for _, w := range f.hor {
		pop := decided(w.cur)
		blk := domain.TrendsWinLossBlock{Days: w.days, Matches: len(pop), Required: winLossRequired, Rows: []domain.TrendsWinLossRow{}}
		if len(pop) >= winLossRequired {
			for _, s := range stats {
				if row, ok := winLossRow(s, pop); ok {
					blk.Rows = append(blk.Rows, row)
				}
			}
			sortWinLoss(blk.Rows)
		}
		out = append(out, blk)
	}
	return out
}

// decided ne garde que les matchs gagnés ou perdus.
func decided(ms []Match) []Match {
	out := make([]Match, 0, len(ms))
	for _, m := range ms {
		if m.Outcome == canonical.OutcomeWin || m.Outcome == canonical.OutcomeLoss {
			out = append(out, m)
		}
	}
	return out
}

// winLossRow calcule une ligne : au moins 30 valeurs et une corrélation définie,
// sinon false. z_win et z_loss sont les moyennes en victoire et en défaite,
// centrées sur la moyenne de l'horizon et réduites par son écart-type de population.
func winLossRow(s winLossStat, pop []Match) (domain.TrendsWinLossRow, bool) {
	var vals, outcomes, winVals, lossVals []float64
	for _, m := range pop {
		v, ok := s.value(m)
		if !ok {
			continue
		}
		vals = append(vals, v)
		if m.Outcome == canonical.OutcomeWin {
			outcomes = append(outcomes, 1)
			winVals = append(winVals, v)
		} else {
			outcomes = append(outcomes, 0)
			lossVals = append(lossVals, v)
		}
	}
	if len(vals) < winLossRequired {
		return domain.TrendsWinLossRow{}, false
	}
	r, ok := pearson(vals, outcomes)
	if !ok {
		return domain.TrendsWinLossRow{}, false
	}
	mean := meanOf(vals)
	sd := sdOrOne(stdPop(vals, mean))
	wm, lm := meanOf(winVals), meanOf(lossVals)
	return domain.TrendsWinLossRow{
		Key: s.key, Group: s.group, Matches: len(vals),
		WinMean: wm, LossMean: lm, ZWin: (wm - mean) / sd, ZLoss: (lm - mean) / sd, R: r,
	}, true
}

// groupRank ordonne les groupes de statistiques.
func groupRank(g string) int {
	switch g {
	case domain.TrendsWinLossStats:
		return 0
	case domain.TrendsWinLossComposite:
		return 1
	}
	return 2
}

// sortWinLoss trie par groupe puis |r| décroissant (clé en dernier recours).
func sortWinLoss(rows []domain.TrendsWinLossRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if ga, gb := groupRank(a.Group), groupRank(b.Group); ga != gb {
			return ga < gb
		}
		if aa, ab := abs(a.R), abs(b.R); aa != ab {
			return aa > ab
		}
		return a.Key < b.Key
	})
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
