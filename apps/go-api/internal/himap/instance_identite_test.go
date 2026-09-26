package himap

// instance_identite_test.go — le fixture d'instance partage par les tests de rasterisation
// du paquet.
//
// POURQUOI IL VIT ICI, HORS DU TAG `gamefiles`. Il ne touche ni le jeu ni l'installation : une
// simple Instance identite pour tester une transformation locale->monde sans rien deplacer.
// Il vivait dans heightfield_test.go, supprime au lot v2 G.2 (2026-09-05, code mort) — le
// deplacer ici evite de casser rendu_test.go, rendu_couleur_test.go, rendu_reference_test.go,
// sddt_test.go et volume_test.go, qui n'ont aucun rapport avec heightfield mais s'en
// servaient tous.

// instanceIdentite : une instance qui ne deplace rien, pour tester la rasterisation seule.
//
// Le `Scale` unitaire n'est PAS decoratif : depuis que LocalToWorld applique le champ
// `scale` du sbsp, une instance zero-value ecraserait tout le maillage sur sa position.
func instanceIdentite() Instance {
	return Instance{
		Scale:   [3]float64{1, 1, 1},
		Forward: [3]float64{1, 0, 0},
		Left:    [3]float64{0, 1, 0},
		Up:      [3]float64{0, 0, 1},
	}
}
