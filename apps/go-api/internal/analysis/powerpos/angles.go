package powerpos

// angles.go — LES SIGNAUX ANGULAIRES : de quelles directions on tue, de quelles directions
// on meurt (D10b du plan des positions de force, 2026-09-20).
//
// # POURQUOI DE L'ANGLE PLUTOT QU'UN RAPPORT DE DUEL
//
// Le verdict v1 (VERDICT_ORACLE_2026-09-20.md) a montre que le rapport de duel, meme lisse
// sur un disque de 2 m, retrouve les BASES et les points de defense, pas les lieux que les
// equipes tiennent. La raison tient a ce qu'il mesure : QUI gagne, et non COMMENT le lieu
// est fait. Deux cellules au meme rapport de duel peuvent etre un couloir ouvert aux quatre
// vents et un surplomb qui ne se prend que par un cote.
//
// Une position de force, telle que les guides la decrivent, se reconnait a deux proprietes
// GEOMETRIQUES que nos donnees savent mesurer sans lire la carte :
//
//   - ELLE COUVRE PLUSIEURS VOIES. Les kills qui en partent s'eparpillent en direction :
//     on y tient plusieurs lignes de vue a la fois.
//   - ELLE N'EST PRISE QUE PAR QUELQUES ANGLES. Les morts qu'on y subit viennent toutes du
//     meme cote : le reste est couvert par de la geometrie.
//
// Ces deux mesures sont des DISPERSIONS ANGULAIRES de directions unitaires, et non des
// comptages : elles ne dependent pas du volume de duels, seulement de leur repartition.
//
// # LE PIEGE DU PETIT ECHANTILLON, ET SA CORRECTION
//
// La longueur resultante moyenne d'un paquet de n directions TIREES AU HASARD ne vaut pas
// zero : son carre vaut 1/n en esperance (statistique de Rayleigh). A n = 10, une cellule
// parfaitement isotrope rend donc R ~ 0,32, soit une dispersion de 0,68 au lieu de 1,00 —
// et une cellule a peu de morts passerait pour un abri. C'est exactement la faute que la v1
// a commise avec le rapport de duel par cellule (bruit binomial lu comme du terrain).
//
// La correction est classique et elle est APPLIQUEE ICI, pas signalee en note de bas de
// page : l'estimateur sans biais du carre de la resultante est (n*R^2 - 1) / (n - 1), et
// c'est lui que `ResultanteCorrigee` rend. Sous l'hypothese isotrope il vaut zero en
// esperance, quel que soit n.

import "math"

// SommeAngulaire accumule des directions unitaires : la somme de leurs cosinus, celle de
// leurs sinus, et leur nombre.
//
// POURQUOI DES SOMMES ET NON DES ANGLES. Une somme vectorielle est ADDITIVE : la somme de
// deux cellules voisines est la somme de leurs sommes. C'est ce qui permet au score de
// lisser sur un disque de 2 m en agregeant des cellules, sans jamais avoir a re-parcourir
// les eliminations. Une moyenne d'angles, elle, n'a pas de sens (0 deg et 359 deg ont pour
// moyenne 179,5 deg).
type SommeAngulaire struct {
	Cos float64 `json:"cos"`
	Sin float64 `json:"sin"`
	N   int     `json:"n"`
}

// Ajoute compte une direction donnee par un vecteur NON normalise (dx, dy). Le vecteur est
// normalise ici : seule compte la direction, la distance est deja mesuree par la portee.
//
// Un vecteur NUL est ignore et non compte : deux positions confondues ne portent aucune
// direction, et lui en inventer une (zero radian) biaiserait la resultante vers l'est.
func (s *SommeAngulaire) Ajoute(dx, dy float64) {
	norme := math.Hypot(dx, dy)
	if norme <= 0 || math.IsNaN(norme) || math.IsInf(norme, 0) {
		return
	}
	s.Cos += dx / norme
	s.Sin += dy / norme
	s.N++
}

// Plus rend la somme de deux accumulations (agregation d'un voisinage).
func (s SommeAngulaire) Plus(autre SommeAngulaire) SommeAngulaire {
	return SommeAngulaire{Cos: s.Cos + autre.Cos, Sin: s.Sin + autre.Sin, N: s.N + autre.N}
}

// Resultante rend la longueur resultante moyenne BRUTE, dans [0, 1] : 1 quand toutes les
// directions coincident, 0 quand elles s'annulent. Zero pour une somme vide.
//
// Elle n'est PAS le signal publie : a petit n elle est biaisee vers le haut (cf. l'en-tete).
// Elle sert a la correction et aux tests.
func (s SommeAngulaire) Resultante() float64 {
	if s.N <= 0 {
		return 0
	}
	return borne01(math.Hypot(s.Cos, s.Sin) / float64(s.N))
}

// ResultanteCorrigee rend l'estimateur SANS BIAIS de la longueur resultante :
// sqrt(max(0, (n*R^2 - 1) / (n - 1))). Zero quand n < 2 (une direction seule ne dit rien
// d'une repartition) et zero quand l'estimateur est negatif (echantillon plus disperse que
// l'isotropie, ce qui arrive et n'a pas de sens physique).
func (s SommeAngulaire) ResultanteCorrigee() float64 {
	if s.N < 2 {
		return 0
	}
	n := float64(s.N)
	r := math.Hypot(s.Cos, s.Sin) / n
	carre := (n*r*r - 1) / (n - 1)
	if carre <= 0 {
		return 0
	}
	return borne01(math.Sqrt(carre))
}

// Dispersion rend 1 - ResultanteCorrigee : 0 quand toutes les directions coincident, 1
// quand elles sont indiscernables de l'isotropie.
//
// ATTENTION AU CAS VIDE : une somme de moins de deux directions rend 1,0, c'est-a-dire
// « aussi disperse qu'on ne sait pas ». L'appelant DOIT verifier `N` avant de s'en servir
// (le score exige un minimum de directions et retombe sinon sur une valeur neutre) : une
// cellule sans mort n'est pas une cellule ou l'on meurt de partout.
func (s SommeAngulaire) Dispersion() float64 {
	return 1 - s.ResultanteCorrigee()
}
