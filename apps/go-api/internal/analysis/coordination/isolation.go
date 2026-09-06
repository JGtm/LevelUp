package coordination

// isolation.go — MOURIR SEUL : la part des morts sans coequipier a portee.
//
// # LA QUESTION, ET POURQUOI ELLE SE MESURE EN DEUX TEMPS
//
// « Ou je meurs isole » demande si, a l'instant de ma mort, un coequipier etait assez pres
// pour peser sur l'echange. La reponse tient a trois choses que personne ne detient
// ensemble : la POSITION de chacun (le film), les EQUIPES (la base — le film ne les porte
// pas, `Track.Team` vaut -1 pour tout le monde), et le RAYON de la portee (une table
// mesuree, par variante de jeu).
//
// La cuisson mesure donc, hors ligne, la distance a CHAQUE autre joueur nomme vivant
// (`analysis/tactical/vies.go`) ; ce fichier-ci tranche, une fois les equipes jointes et le
// rayon connu. Il ne connait ni carte, ni base, ni fichier : il recoit des distances.
//
// # LES DEUX EXCLUSIONS, ET ELLES NE SE CONFONDENT PAS
//
//	TOUS COEQUIPIERS MORTS   la mort sort du DENOMINATEUR (decision produit du plan). Elle
//	                         ne dit rien du placement : on ne peut pas etre mal accompagne
//	                         quand personne ne peut accompagner. La compter en « isole »
//	                         ferait grimper le taux de l'equipe qui perd un combat entier,
//	                         c'est-a-dire mesurer la defaite au lieu du placement.
//	VARIANTE SANS RAYON      le match entier sort de la lecture, et il se COMPTE
//	                         (MatchsSansRayon). Un rayon devine — « 18 m, c'est l'usage » —
//	                         rendrait une mesure d'apparence normale sur une regle de jeu
//	                         qu'on n'a pas mesuree.
//
// UN ADVERSAIRE PROCHE N'ANNULE RIEN : la question porte sur le SOUTIEN, pas sur la
// solitude. Mourir a deux metres d'un ennemi et a quarante de son equipe, c'est mourir
// isole — c'est meme le cas typique.

import "levelup/go-api/internal/domain"

// Isolement tranche l'isolement de chaque mort, rayon PAR MATCH.
//
// `rayonParMatch` : matchID -> rayon en metres. Un match absent de la table est un match
// dont la variante n'a pas de portee mesuree — ses morts sont ECARTEES et le match compte
// dans MatchsSansRayon.
//
// `matchsMesures` est le denominateur « par match » de la couverture : il vient de
// l'appelant (les matchs retenus de la lecture), jamais du contenu des morts — un match
// sans aucune mort examinee reste un match joue.
func Isolement(morts []domain.MortAExaminer, rayonParMatch map[string]float64,
	matchsMesures int) domain.BilanIsolement {
	b := domain.BilanIsolement{Isolees: []domain.MortAExaminer{}}
	sansRayon := make(map[string]bool)
	for _, m := range morts {
		rayon, ok := rayonParMatch[m.MatchID]
		if !ok || rayon <= 0 {
			sansRayon[m.MatchID] = true
			continue
		}
		if len(m.DistancesCoequipiers) == 0 {
			b.SansCoequipierVivant++
			continue
		}
		b.Examinees++
		if !unCoequipierDansLeRayon(m.DistancesCoequipiers, rayon) {
			b.Isolees = append(b.Isolees, m)
		}
	}
	b.MatchsSansRayon = len(sansRayon)
	b.Couverture = Mesurer(len(b.Isolees), b.Examinees, matchsMesures)
	return b
}

// unCoequipierDansLeRayon dit si au moins un coequipier vivant etait a portee.
//
// LA BORNE EST INCLUSIVE : un coequipier EXACTEMENT au rayon est a portee. Le rayon est la
// portee du radar — la distance a laquelle on se voit —, et se voir juste a la limite,
// c'est se voir.
func unCoequipierDansLeRayon(distances []float64, rayon float64) bool {
	for _, d := range distances {
		if d <= rayon {
			return true
		}
	}
	return false
}
