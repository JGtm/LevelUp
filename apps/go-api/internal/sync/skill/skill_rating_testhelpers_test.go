package skill

// skill_rating_testhelpers_test.go — helper de test : le score composite seul, sans breakdown.

// computeCompositeScore calcule le score composite [0,1] d'un match. Wrapper
// rétro-compatible autour de computeCompositeScoreWithBreakdown qui ne retourne
// que le composite (utilisé par les tests existants).
//
// Les composantes manquantes (valeur 0 ou avg nil) sont ignorées et les poids
// renormalisés. 4 params réservés pour futures composantes du score composite
// (enemyAvgKE = carry adjustment vs adversaires ; avgMedalExploit = bonus exploit ;
// avgOffConv = offensive conversion ; avgDefRes = defensive resistance).
//
//nolint:unparam // les tests passent nil pour les moyennes non exercées.
func computeCompositeScore(
	row *compositeMatchRow,
	avgAccuracy *float64,
	enemyAvgKE *float64,
	avgDamageEff *float64,
	avgMedalExploit *float64,
	avgOffConv *float64,
	avgDefRes *float64,
) float64 {
	composite, _ := computeCompositeScoreWithBreakdown(row, avgAccuracy, enemyAvgKE, avgDamageEff, avgMedalExploit, avgOffConv, avgDefRes)
	return composite
}
