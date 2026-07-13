// Package analysis — expected_win.go : comparaison attendu vs réel du taux de
// victoire d'un scope de matchs (module classé du briefing Explorer).
package analysis

// ExpectedVsActual calcule le taux de victoire ATTENDU (moyenne des probabilités
// de victoire pré-match disponibles, ratio 0..1) et RÉEL (wins/total, ratio
// 0..1) d'un scope de matchs.
//
//   - expected est nil quand aucune probabilité pré-match n'est disponible
//     (expectedProbs vide) — le module ne peut pas afficher d'attendu.
//   - actual vaut 0 si total <= 0.
//
// Helper pur (aucune dépendance canonical/DB) : l'appelant collecte les
// probabilités non nil et le couple wins/total du scope.
func ExpectedVsActual(expectedProbs []float64, wins, total int) (expected *float64, actual float64) {
	if total > 0 {
		actual = float64(wins) / float64(total)
	}
	if len(expectedProbs) > 0 {
		var sum float64
		for _, p := range expectedProbs {
			sum += p
		}
		avg := sum / float64(len(expectedProbs))
		expected = &avg
	}
	return expected, actual
}
