package trends

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Décimales et sens des indicateurs.
const (
	decimalsRatio   = 3
	decimalsSeconds = 1
	decimalsHours   = 1
	decimalsInt     = 0
	decimalsOne     = 1
	decimalsTwo     = 2
	betterHigh      = 1
	betterNeutral   = 0
	betterLow       = -1
	otherChain      = domain.TrendsGameTypeOther
	secondsPerHour  = 3600.0
	dayLayout       = "2006-01-02"
	monthKeyLayout  = "2006-01"
	// registryCapacity : taille initiale du registre (lignes fixes + variantes de classement usuelles).
	registryCapacity = 40
)

// evalFunc calcule la valeur d'un indicateur sur un ensemble de matchs ; false
// quand l'ensemble ne permet aucune valeur.
type evalFunc func(ms []Match) (float64, bool)

// indicator est la définition d'une ligne de la matrice.
type indicator struct {
	key, variant, group, unit string
	decimals, better          int
	inMatrix                  bool
	// noMatchSeries : pas de courbe par match (l'indicateur n'a de sens que sur un ensemble).
	noMatchSeries bool
	// eval : évaluateur des séries par match, jour et semaine.
	eval evalFunc
	// cellEval : évaluateur des cellules de la matrice (mois et horizons) et de la
	// série par mois ; nil = eval.
	cellEval evalFunc
}

// cells retourne l'évaluateur des cellules de la matrice et de la série par mois.
func (d indicator) cells() evalFunc {
	if d.cellEval != nil {
		return d.cellEval
	}
	return d.eval
}

// registry construit les indicateurs de la vue Solo : les définitions fixes,
// puis une ligne CSR et une ligne LUSR par groupe de classement rencontré dans
// ms. hpToKill est le seuil de dégâts par frag (rendement, résistance).
func registry(ms []Match, hpToKill float64) []indicator {
	out := append(make([]indicator, 0, registryCapacity), []indicator{
		{key: domain.TrendsKeyEnemyMMR, group: domain.TrendsGroupLevel, unit: domain.TrendsUnitNumber, decimals: decimalsInt, better: betterHigh, inMatrix: true, eval: meanOpt(func(m Match) *float64 { return m.EnemyMMR })},
		{key: domain.TrendsKeyTeamMMR, group: domain.TrendsGroupLevel, unit: domain.TrendsUnitNumber, decimals: decimalsInt, better: betterHigh, inMatrix: true, eval: meanOpt(func(m Match) *float64 { return m.TeamMMR })},
	}...)
	out = append(out, ratingIndicators(ms)...)
	out = append(out, resultIndicators()...)
	out = append(out, combatIndicators(hpToKill)...)
	out = append(out, styleIndicators()...)
	out = append(out, objectiveIndicators()...)
	out = append(out, activityIndicators()...)
	return out
}

// ratingIndicators : une ligne par (type de classement, groupe) ayant au moins
// une valeur dans ms ; CSR avant LUSR, groupes triés.
func ratingIndicators(ms []Match) []indicator {
	type pair struct {
		typ   canonical.RatingType
		group string
	}
	seen := map[pair]bool{}
	for _, m := range ms {
		if m.RatingValue != nil && (m.RatingType == canonical.RatingTypeCSR || m.RatingType == canonical.RatingTypeLUSR) {
			seen[pair{m.RatingType, m.RatingGroup}] = true
		}
	}
	pairs := make([]pair, 0, len(seen))
	for p := range seen {
		pairs = append(pairs, p)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].typ != pairs[j].typ {
			return pairs[i].typ == canonical.RatingTypeCSR
		}
		return pairs[i].group < pairs[j].group
	})
	out := make([]indicator, 0, len(pairs))
	for _, p := range pairs {
		key := domain.TrendsKeyCSRValue
		if p.typ == canonical.RatingTypeLUSR {
			key = domain.TrendsKeyLUSRValue
		}
		out = append(out, indicator{
			key: key, variant: p.group, group: domain.TrendsGroupLevel, unit: domain.TrendsUnitNumber,
			decimals: decimalsInt, better: betterHigh, inMatrix: true, eval: lastRating(p.typ, p.group),
		})
	}
	return out
}

func resultIndicators() []indicator {
	return []indicator{
		{key: domain.TrendsKeyWinRate, group: domain.TrendsGroupResults, unit: domain.TrendsUnitRatio, decimals: decimalsRatio, better: betterHigh, inMatrix: true, noMatchSeries: true, eval: winRate},
		{key: domain.TrendsKeyPerformanceScore, group: domain.TrendsGroupResults, unit: domain.TrendsUnitNumber, decimals: decimalsOne, better: betterHigh, inMatrix: true, eval: meanOpt(func(m Match) *float64 { return m.PerformanceScore })},
	}
}

func combatIndicators(hp float64) []indicator {
	num := domain.TrendsUnitNumber
	cb := domain.TrendsGroupCombat
	return []indicator{
		{key: domain.TrendsKeyKDA, group: cb, unit: num, decimals: decimalsTwo, better: betterHigh, inMatrix: true, eval: aggregateKDA},
		{key: domain.TrendsKeyDamageBalance, group: cb, unit: num, decimals: decimalsInt, better: betterHigh, inMatrix: true, eval: meanOpt(damageBalance)},
		{key: domain.TrendsKeyAccuracy, group: cb, unit: domain.TrendsUnitRatio, decimals: decimalsRatio, better: betterHigh, inMatrix: true, eval: accuracy},
		{key: domain.TrendsKeyAssistsPerMatch, group: cb, unit: num, decimals: decimalsOne, better: betterHigh, inMatrix: true, eval: meanVal(func(m Match) float64 { return float64(m.Assists) })},
		{key: domain.TrendsKeyAvgMaxKillingSpree, group: cb, unit: num, decimals: decimalsOne, better: betterHigh, inMatrix: true, eval: meanOpt(func(m Match) *float64 { return m.MaxSpree })},
		{key: domain.TrendsKeyDefensiveResistance, group: cb, unit: num, decimals: decimalsTwo, better: betterHigh, inMatrix: true, eval: yieldEval(hp, false)},
		{key: domain.TrendsKeyOffensiveConversion, group: cb, unit: num, decimals: decimalsTwo, better: betterHigh, inMatrix: true, eval: yieldEval(hp, true)},
		{key: domain.TrendsKeyKillsPerMatch, group: cb, unit: num, decimals: decimalsOne, better: betterNeutral, eval: meanVal(func(m Match) float64 { return float64(m.Kills) })},
		{key: domain.TrendsKeyDeathsPerMatch, group: cb, unit: num, decimals: decimalsOne, better: betterNeutral, eval: meanVal(func(m Match) float64 { return float64(m.Deaths) })},
		{key: domain.TrendsKeyAvgDamageDealt, group: cb, unit: num, decimals: decimalsInt, better: betterNeutral, eval: meanOpt(func(m Match) *float64 { return m.DamageDealt })},
		{key: domain.TrendsKeyAvgDamageTaken, group: cb, unit: num, decimals: decimalsInt, better: betterNeutral, eval: meanOpt(func(m Match) *float64 { return m.DamageTaken })},
	}
}

func styleIndicators() []indicator {
	st := domain.TrendsGroupStyle
	return []indicator{
		{key: domain.TrendsKeyAvgLifeSeconds, group: st, unit: domain.TrendsUnitSeconds, decimals: decimalsSeconds, better: betterHigh, inMatrix: true, eval: meanOpt(func(m Match) *float64 { return m.AvgLife })},
		{key: domain.TrendsKeyHeadshotShare, group: st, unit: domain.TrendsUnitRatio, decimals: decimalsRatio, better: betterHigh, inMatrix: true, eval: killShare(func(m Match) int { return m.HeadshotKills })},
		{key: domain.TrendsKeyPowerWeaponShare, group: st, unit: domain.TrendsUnitRatio, decimals: decimalsRatio, better: betterNeutral, inMatrix: true, eval: killShare(func(m Match) int { return m.PowerKills })},
		equipmentUsedShare(),
	}
}

func activityIndicators() []indicator {
	ac := domain.TrendsGroupActivity
	num := domain.TrendsUnitNumber
	return []indicator{
		{key: domain.TrendsKeyMatchCount, group: ac, unit: num, decimals: decimalsInt, better: betterNeutral, inMatrix: true, noMatchSeries: true, eval: matchCount},
		{key: domain.TrendsKeyHoursPlayed, group: ac, unit: domain.TrendsUnitHours, decimals: decimalsHours, better: betterNeutral, inMatrix: true, noMatchSeries: true, eval: hoursPlayed},
		{key: domain.TrendsKeyDaysPlayed, group: ac, unit: num, decimals: decimalsInt, better: betterNeutral, inMatrix: true, noMatchSeries: true, eval: daysPlayed},
		{key: domain.TrendsKeyDNFRate, group: ac, unit: domain.TrendsUnitRatio, decimals: decimalsRatio, better: betterLow, inMatrix: true, noMatchSeries: true, eval: dnfRate},
	}
}

// meanOpt : moyenne d'une valeur optionnelle, sur les matchs qui la portent.
func meanOpt(get func(Match) *float64) evalFunc {
	return func(ms []Match) (float64, bool) {
		var sum float64
		n := 0
		for _, m := range ms {
			if v := get(m); v != nil {
				sum += *v
				n++
			}
		}
		if n == 0 {
			return 0, false
		}
		return sum / float64(n), true
	}
}

// meanVal : moyenne d'une valeur toujours définie, sur tous les matchs.
func meanVal(get func(Match) float64) evalFunc {
	return func(ms []Match) (float64, bool) {
		if len(ms) == 0 {
			return 0, false
		}
		var sum float64
		for _, m := range ms {
			sum += get(m)
		}
		return sum / float64(len(ms)), true
	}
}

// damageBalance : dégâts infligés moins subis, absent si l'un des deux manque.
func damageBalance(m Match) *float64 {
	if m.DamageDealt == nil || m.DamageTaken == nil {
		return nil
	}
	v := *m.DamageDealt - *m.DamageTaken
	return &v
}

// winRate : victoires / tous les matchs.
func winRate(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	wins := 0
	for _, m := range ms {
		if m.Outcome == canonical.OutcomeWin {
			wins++
		}
	}
	return analysis.WinRate(wins, len(ms)), true
}

// dnfRate : abandons / tous les matchs.
func dnfRate(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	n := 0
	for _, m := range ms {
		if m.Outcome == canonical.OutcomeDNF {
			n++
		}
	}
	return float64(n) / float64(len(ms)), true
}

// aggregateKDA : FDA agrégé canonique (net moyen par match, jamais un quotient).
func aggregateKDA(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	var k, a, d int
	for _, m := range ms {
		k += m.Kills
		a += m.Assists
		d += m.Deaths
	}
	return analysis.AggregateKDA(k, a, d, len(ms)), true
}

// accuracy : touchés / tirés sur les matchs ayant tiré.
func accuracy(ms []Match) (float64, bool) {
	var fired, hit int
	for _, m := range ms {
		if m.ShotsFired > 0 {
			fired += m.ShotsFired
			hit += m.ShotsHit
		}
	}
	if fired == 0 {
		return 0, false
	}
	return analysis.Accuracy(hit, fired), true
}

// yieldEval : rendement offensif (offensive) ou résistance défensive sur les
// totaux des matchs aux dégâts connus. Absent si les dégâts infligés (offensif)
// ou les morts (défensif) sont nuls.
func yieldEval(hp float64, offensive bool) evalFunc {
	return func(ms []Match) (float64, bool) {
		var k, a, d, dealt, taken float64
		for _, m := range ms {
			if m.DamageDealt == nil || m.DamageTaken == nil {
				continue
			}
			k += float64(m.Kills)
			a += float64(m.Assists)
			d += float64(m.Deaths)
			dealt += *m.DamageDealt
			taken += *m.DamageTaken
		}
		if offensive && dealt <= 0 || !offensive && d <= 0 {
			return 0, false
		}
		cy := analysis.ComputeCombatYieldFloat(k, a, dealt, taken, d, hp)
		if offensive {
			return cy.OffensiveConversion, true
		}
		return cy.DefensiveResistance, true
	}
}

// killShare : part des frags obtenue d'une certaine manière ; absente sans frag.
func killShare(part func(Match) int) evalFunc {
	return func(ms []Match) (float64, bool) {
		var p, k int
		for _, m := range ms {
			p += part(m)
			k += m.Kills
		}
		if k == 0 {
			return 0, false
		}
		return float64(p) / float64(k), true
	}
}

func matchCount(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	return float64(len(ms)), true
}

// hoursPlayed : somme des secondes jouées, en heures.
func hoursPlayed(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	var s float64
	for _, m := range ms {
		s += m.Seconds
	}
	return s / secondsPerHour, true
}

// daysPlayed : nombre de jours locaux distincts.
func daysPlayed(ms []Match) (float64, bool) {
	if len(ms) == 0 {
		return 0, false
	}
	days := map[string]bool{}
	for _, m := range ms {
		days[m.Start.Format(dayLayout)] = true
	}
	return float64(len(days)), true
}

// lastRating : dernière valeur de classement connue du type et du groupe, dans
// l'ordre chronologique de ms.
func lastRating(typ canonical.RatingType, group string) evalFunc {
	return func(ms []Match) (float64, bool) {
		var best *Match
		for i := range ms {
			m := &ms[i]
			if m.RatingValue == nil || m.RatingType != typ || m.RatingGroup != group {
				continue
			}
			if best == nil || !m.Start.Before(best.Start) {
				best = m
			}
		}
		if best == nil {
			return 0, false
		}
		return *best.RatingValue, true
	}
}
