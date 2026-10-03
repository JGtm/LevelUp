package revision

// export_test.go — LE BANC DU MECANISME, POUR LES TESTS SEULS.

// Calculer hache des ARBRES ENTIERS, sans fermeture des imports — le banc des tests du cadre, des
// jetons et des fichiers embarques. La production passe par [Module.CalculerCouche] depuis le
// lot J3.2 : garder cette fonction en production sans appelant serait du code mort (CLAUDE.md
// regle 7).
func Calculer(racines []string, exclure func(rel string) bool, valeursAmont ...string) (Resultat, error) {
	e := nouvelEmpreinteur(valeursAmont)
	for _, racine := range racines {
		if err := e.arbre(racine, exclure); err != nil {
			return Resultat{}, err
		}
	}
	return e.resultat(), nil
}
