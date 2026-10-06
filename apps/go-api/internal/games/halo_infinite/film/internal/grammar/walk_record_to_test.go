package grammar

// walk_record_to_test.go — LA MARCHE VERS UNE CIBLE, AIDE DES TESTS ET DES INSTRUMENTS.
//
// Elle etait le moteur des lecteurs ancres jusqu au lot 2.7.b de la representation intermediaire ;
// les lecteurs rejouent desormais les lectures bipedes ([recordBipedeLu.parcourirJusqua]). Elle
// reste ici pour les tests et les instruments qui marchent un record ancre jusqu a un composant.

// walkRecordTo marche les composants du masque avec les désers de PRODUCTION jusqu'à
// consommer celui d'index target — c'est cette consommation qui déclenche le hook. Rend
// false dès qu'un composant intermédiaire n'est pas porté ou que la marche déborde du
// payload : au-delà, la position du curseur ne serait plus digne de confiance, et lire du
// bruit vaut moins que ne rien lire.
//
// walkRecordTo s'exprime en UNE ligne de walkRecordComponents : la marche elle-même n'existe
// qu'à un seul exemplaire (règle des <= 2 copies, CLAUDE.md n°6).
func walkRecordTo(pay []byte, i0, total int, idx []int, g grammaireRecord, target int) bool {
	found := false
	walkRecordComponents(pay, i0, total, idx, g, func(id int) bool {
		if id == target {
			found = true
			return false
		}
		return true
	})
	return found
}
