package trends

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// paceIndicators : le groupe « Rythme », cinq stats par minute jouée. Chacune est le
// rapport de la SOMME des comptes à la SOMME des minutes jouées de l'ensemble évalué
// (jamais la moyenne des rapports par match, qui donnerait autant de poids à un match
// de trois minutes qu'à un match de quinze).
func paceIndicators() []indicator {
	pace := func(key string, better int, decimals int, count func(Match) (float64, bool)) indicator {
		return indicator{
			key: key, group: domain.TrendsGroupPace, unit: domain.TrendsUnitPerMin, decimals: decimals,
			better: better, inMatrix: true, eval: perMinute(count),
		}
	}
	return []indicator{
		pace(domain.TrendsKeyKillsPerMinute, betterHigh, decimalsTwo, func(m Match) (float64, bool) { return float64(m.Kills), true }),
		pace(domain.TrendsKeyDeathsPerMinute, betterLow, decimalsTwo, func(m Match) (float64, bool) { return float64(m.Deaths), true }),
		pace(domain.TrendsKeyAssistsPerMinute, betterHigh, decimalsTwo, func(m Match) (float64, bool) { return float64(m.Assists), true }),
		pace(domain.TrendsKeyDamageDealtPerMinute, betterHigh, decimalsInt, optCount(func(m Match) *float64 { return m.DamageDealt })),
		pace(domain.TrendsKeyDamageTakenPerMinute, betterLow, decimalsInt, optCount(func(m Match) *float64 { return m.DamageTaken })),
	}
}

// optCount adapte un compte optionnel (nil = inconnu) à la forme de perMinute.
func optCount(get func(Match) *float64) func(Match) (float64, bool) {
	return func(m Match) (float64, bool) {
		if v := get(m); v != nil {
			return *v, true
		}
		return 0, false
	}
}

// perMinute : somme des comptes / somme des minutes jouées ([analysis.PooledPerMinute]) de l'ensemble
// évalué.
func perMinute(count func(Match) (float64, bool)) evalFunc {
	return func(ms []Match) (float64, bool) {
		return analysis.PooledPerMinute(ms, count, func(m Match) float64 { return m.Seconds })
	}
}
