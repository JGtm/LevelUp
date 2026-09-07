package coordination

// isolation.go — MOURIR SEUL : la part des morts sans coequipier a portee.
//
// # LA QUESTION, ET OU CHAQUE MORCEAU DE REPONSE VIT
//
// « Ou je meurs isole » demande si, a l'instant de ma mort, un coequipier etait assez pres
// pour peser sur l'echange. La reponse tient a quatre choses que personne ne detient
// ensemble : les MORTS (la base : journal des morts), les POSITIONS (le sidecar de rejeu,
// qui ne juge rien), les EQUIPES (la base) et le RAYON (une table mesuree, par variante).
//
// Le service assemble les quatre et rend, par mort, l'etat de chaque coequipier. CE FICHIER
// NE FAIT QUE TRANCHER : il ne connait ni carte, ni base, ni fichier.
//
// # TROIS SORTIES, ET AUCUNE NE SE CONFOND AVEC UNE AUTRE
//
//	ACCOMPAGNEE          au moins un coequipier VIVANT a portee. Elle compte au
//	                     denominateur et pas au numerateur.
//	ISOLEE               au moins un coequipier vivant, aucun a portee.
//	EQUIPE A TERRE       tous morts ou partis (ou aucun coequipier du tout) : la mort sort
//	                     du DENOMINATEUR et se compte a part. Elle ne dit rien du
//	                     placement — on ne peut pas etre mal accompagne quand personne ne
//	                     peut accompagner —, et l'y compter ferait mesurer la defaite.
//
//	VARIANTE SANS RAYON  le match entier sort de la lecture, et il se COMPTE au niveau du
//	                     MATCH — pas au fil des morts, sinon un match ou l'on ne meurt pas
//	                     ne serait jamais signale. Un rayon devine rendrait une mesure
//	                     d'apparence normale sur une regle qu'on n'a pas mesuree.
//
// UN ADVERSAIRE PROCHE N'ANNULE RIEN : la question porte sur le SOUTIEN, pas sur la
// solitude. Mourir a deux metres d'un ennemi et a quarante de son equipe, c'est mourir
// isole — c'est meme le cas typique.

import "levelup/go-api/internal/domain"

// Isolement tranche l'isolement de chaque mort, rayon PAR MATCH.
//
// `rayonParMatch` : matchID -> rayon en metres. IL EST AUSSI L'UNIVERS DE LA LECTURE : un
// match qui n'y figure pas n'a pas de portee mesuree et ses morts sont ecartees. Le COMPTE
// des matchs ainsi ecartes est pose par l'appelant (`MatchsSansRayon`), qui seul connait
// les matchs sans mort.
//
// `matchsMesures` : les matchs mesures AYANT un rayon. C'est le denominateur « par match ».
func Isolement(morts []domain.MortAExaminer, rayonParMatch map[string]float64,
	matchsMesures int) domain.BilanIsolement {
	b := domain.BilanIsolement{Isolees: []domain.MortAExaminer{}}
	for _, m := range morts {
		rayon, ok := rayonParMatch[m.MatchID]
		if !ok || rayon <= 0 {
			continue
		}
		accompagne, unVivant := false, false
		for _, c := range m.Coequipiers {
			if !c.Vivant {
				continue
			}
			unVivant = true
			// LA BORNE EST INCLUSIVE : le rayon est la portee du radar, et se voir juste a
			// la limite, c'est se voir. Un coequipier vivant dont on n'a AUCUNE position
			// ne prouve rien : il compte comme presence, jamais comme soutien.
			if c.PositionConnue && c.DistanceM <= rayon {
				accompagne = true
				break
			}
		}
		switch {
		case accompagne:
			b.Examinees++
		case unVivant:
			b.Examinees++
			b.Isolees = append(b.Isolees, m)
		default:
			b.EquipeATerre++
		}
	}
	b.Couverture = Mesurer(len(b.Isolees), b.Examinees, matchsMesures)
	return b
}
