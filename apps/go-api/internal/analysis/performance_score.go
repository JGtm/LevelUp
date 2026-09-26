// Package analysis — constantes de résultat de match et petits utilitaires numériques
// partagés par plusieurs algorithmes du paquet (skill rating, narrative, breakdown).
//
// Historique : ce fichier portait aussi l'algorithme « Performance Score relatif »
// (port de src/analysis/_performance_relative.py, v5-relative — ComputeRelativePerformanceScore,
// ComputePerformanceSeries) qui alimentait l'onglet « Forme » de /stats. Cet onglet n'avait
// plus aucun consommateur web (confirmé par l'utilisateur, chantier note de perf,
// 2026-08-27) : l'algorithme, buildFormTab (internal/service/stats_service.go) et leurs
// tests ont été retirés le 2026-09-10 (CLAUDE.md règle 0 code mort, lot hygiène 5.3,
// `.ai/V7.5/REGISTRE_REPORTS.md`).
package analysis

// Codes numériques des issues de match Halo Infinite.
//
// DEPRECATED : utiliser domain.OutcomeWin / domain.OutcomeLoss à la place.
// Ces alias sont conservés pour ne pas casser les call-sites existants
// (analysis/comeback.go, ...). Migration progressive en P4 (canonical big-bang).
const (
	OutcomeWin  = 2
	OutcomeLoss = 3
)

// MinMatchesForRelative est le nombre minimum de matchs pour activer un score
// relatif à l'historique (skill rating, synthèse de session, simulateur diag).
const MinMatchesForRelative = 10

// Clés canoniques de composantes narrative héritées du vocabulaire de l'ex
// performance score relatif. Partagées avec profile.narrativeAxesForComponent.
const (
	PerfMetricKillsVsExpected  = "kills_vs_expected"
	PerfMetricDeathsVsExpected = "deaths_vs_expected"
)

// clampF restreint une valeur à [min, max]. Partagé par skill_rating.go.
func clampF(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
