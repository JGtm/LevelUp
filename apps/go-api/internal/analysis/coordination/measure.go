package coordination

import "levelup/go-api/internal/domain"

// SeuilEchantillonFaible est le nombre minimal d'evenements au denominateur pour qu'un taux
// se lise sans reserve : 30 morts (plan tactique, 2026-09-05).
//
// En dessous, la mesure existe mais ne classe personne : « 100 % de morts vengees » sur huit
// morts est une coincidence de tirage, et l'afficher a cote d'un joueur mesure sur deux
// cents morts inverserait le classement au profit du moins observe. Le drapeau ne cache pas
// la valeur, il interdit de la comparer.
const SeuilEchantillonFaible = 30

// Mesurer construit la seule forme sous laquelle un taux sort de ce paquet.
//
//	brut   : le numerateur — les evenements comptes (morts vengees, ...) ;
//	n      : le denominateur — la taille de l'echantillon (morts examinees, ...) ;
//	matchs : le nombre de matchs retenus, pour la quantite par match.
//
// Un denominateur nul ou negatif rend un taux nul ET un echantillon faible : « aucune
// donnee » n'est pas « zero pour cent », et rien ne doit se lire comme une performance.
func Mesurer(brut, n, matchs int) domain.Couverture {
	c := domain.Couverture{
		Brut:              brut,
		N:                 n,
		EchantillonFaible: n < SeuilEchantillonFaible,
	}
	if n > 0 {
		c.Taux = float64(brut) / float64(n)
	}
	if matchs > 0 {
		c.ParMatch = float64(brut) / float64(matchs)
	}
	return c
}
