package patterns

import "time"

// engine.go — orchestrateur du Pattern Engine.
//
// Analyze() est la seule fonction publique du package : elle délègue
// à analyzeContext, analyzeBehavior et selectLevers.
// Stateless : aucun accès DB.

// Analyze exécute toutes les analyses de patterns sur les données en entrée.
// Retourne un PatternReport vide si les données sont insuffisantes.
// Analyse limitée aux N premières lignes si N > 0.
func Analyze(input AnalyzeInput) PatternReport {
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	rows := input.Rows
	if input.N > 0 && len(rows) > input.N {
		rows = rows[:input.N]
	}
	report := PatternReport{
		WindowSize:          len(rows),
		ComputedAt:          input.Now,
		MinMatchesForSignal: MinMatchesForSignal,
	}
	if len(rows) == 0 {
		return report
	}
	report.ContextPatterns = analyzeContext(rows, input.Config)
	report.BehaviorPatterns = analyzeBehavior(rows, input.Config)
	report.Levers = selectLevers(report.ContextPatterns, report.BehaviorPatterns, rows, input.Config)
	return report
}
