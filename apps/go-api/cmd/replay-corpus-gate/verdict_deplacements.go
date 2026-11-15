package main

// verdict_deplacements.go — UN COMPTEUR QUI BAISSE PARCE QUE SA MATIERE EST LUE AILLEURS.
//
// Certaines couvertures publient la MEME matiere par deux voies de lecture. Quand la voie de
// preference lit ce que la voie de rattrapage lisait seule, le compteur du rattrapage baisse et
// celui de la voie de preference monte : la matiere a change de compteur, elle n'a pas disparu.
// `replaydiff` lit chaque feuille isolement et nomme la baisse une perte ; cette regle, locale au
// verdict du gate, la requalifie quand la hausse de la voie cible la compense.
//
// # LA REGLE
//
// Une baisse de la feuille source (sens perte ou disparu) est un CHANGEMENT, et non une perte, si
// et seulement si, sur le MEME temoin :
//
//   - la feuille cible figure aux ecarts, avec une valeur numerique de chaque cote ;
//   - la hausse de la cible est AU MOINS egale a la baisse de la source.
//
// Autrement dit, la somme source + cible ne baisse pas. Dans tous les autres cas la baisse reste
// ce que `replaydiff` a mesure et bloque comme avant :
//
//   - cible absente des ecarts (inchangee) : rien n'a ete repris par l'autre voie ;
//   - cible en hausse de moins que la baisse : une part de la matiere a disparu ;
//   - cible en baisse : les deux voies perdent ;
//   - valeur non numerique : on ne devine pas.
//
// La comparaison se fait feuille a feuille (`matched` avec `matched`, `published` avec
// `published`) : la hausse d'une feuille ne compense jamais la baisse d'une autre. Une feuille
// source absente de la table n'est jamais requalifiee par cette regle.

import "levelup/go-api/internal/replaydiff"

// deplacementDeCompteur declare une baisse source qu'une hausse cible peut compenser.
type deplacementDeCompteur struct {
	// Source : la feuille dont la baisse est requalifiee, chemin complet de la metrique.
	Source string
	// Cible : la feuille dont la hausse doit couvrir la baisse de la source.
	Cible string
	// Preuve : ou la relation entre les deux voies est declaree.
	Preuve string
}

// deplacementsDeCompteurs — LA TABLE. Une entree par feuille source : la compensation se juge
// feuille a feuille.
func deplacementsDeCompteurs() []deplacementDeCompteur {
	const (
		preuve = "domain/replaydoc/coverage.go, DeathsPathsCoverage : la marche et le balayage direct " +
			"lisent le meme champ de mort ; la mort retrouvee par les deux n'est publiee qu'une fois"
		balayage = "coverage.deathsPaths.directScan."
		marche   = "coverage.deathsPaths.walk."
	)
	return []deplacementDeCompteur{
		{Source: balayage + "matched", Cible: marche + "matched", Preuve: preuve},
		{Source: balayage + "published", Cible: marche + "published", Preuve: preuve},
	}
}

// estDeplacementCompense dit si l'ecart `d` est une baisse de feuille source que la hausse de sa
// cible compense sur ce temoin (`parMetrique` : les ecarts du MEME temoin, par metrique).
func estDeplacementCompense(d replaydiff.Difference,
	parMetrique map[string]replaydiff.Difference) bool {
	for _, r := range deplacementsDeCompteurs() {
		if r.Source != d.Metrique {
			continue
		}
		cible, connue := parMetrique[r.Cible]
		if !connue {
			return false
		}
		baisse, ok := baisseDEcart(d)
		if !ok {
			return false
		}
		hausse, ok := hausseDEcart(cible)
		return ok && baisse > 0 && hausse >= baisse
	}
	return false
}

// baisseDEcart rend de combien la valeur d'un ecart a baisse (ancien - nouveau).
func baisseDEcart(d replaydiff.Difference) (float64, bool) {
	ancien, okA := nombreDEcart(d.Ancien)
	nouveau, okN := nombreDEcart(d.Nouveau)
	return ancien - nouveau, okA && okN
}

// hausseDEcart rend de combien la valeur d'un ecart a monte (nouveau - ancien).
func hausseDEcart(d replaydiff.Difference) (float64, bool) {
	baisse, ok := baisseDEcart(d)
	return -baisse, ok
}
