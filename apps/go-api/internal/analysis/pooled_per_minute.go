package analysis

// PooledPerMinute rend la cadence par minute d'un ENSEMBLE de matchs : la SOMME des comptes rapportée à la
// SOMME des minutes jouées, jamais la moyenne des rapports par match (qui donnerait autant de poids à un
// match de trois minutes qu'à un match de quinze).
//
// Un match dont le compte est inconnu (`count` rend faux) ou dont la durée n'est pas strictement positive
// sort du numérateur ET du dénominateur : un compte inconnu n'est pas un zéro, une durée inconnue ne
// définit pas de cadence. Sans aucun match retenu, l'ensemble n'a pas de valeur (faux), jamais zéro. La
// division finale est celle de [PerMinute].
//
// C'est la règle du groupe « Rythme » de Tendances et des dégâts par minute de l'escouade. Le miroir web est
// `pooledPerMinute` (lib/charts/perMinute.ts) ; le jeu de cas testdata/par_minute_cas.json est lu par les
// deux tests, et c'est lui qui les empêche de diverger.
func PooledPerMinute[T any](items []T, count func(T) (float64, bool), seconds func(T) float64) (float64, bool) {
	var sum, secs float64
	for _, it := range items {
		s := seconds(it)
		if s <= 0 {
			continue
		}
		v, ok := count(it)
		if !ok {
			continue
		}
		sum += v
		secs += s
	}
	if secs <= 0 {
		return 0, false
	}
	return PerMinute(sum, secs), true
}
