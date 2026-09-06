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
// # QUATRE SORTIES, ET AUCUNE NE SE CONFOND AVEC UNE AUTRE
//
//	TOUS COEQUIPIERS MORTS   la mort sort du DENOMINATEUR (decision produit du plan). Elle
//	                         ne dit rien du placement : on ne peut pas etre mal accompagne
//	                         quand personne ne peut accompagner. La compter en « isole »
//	                         ferait grimper le taux de l'equipe qui perd un combat entier,
//	                         c'est-a-dire mesurer la defaite au lieu du placement. Elle
//	                         exige que TOUS les coequipiers soient SUS morts.
//	UN COEQUIPIER INVISIBLE  la mort est INDETERMINEE : elle sort du denominateur et se
//	                         compte a part. Un occupant de vehicule non attribue ou un
//	                         survivant anonyme est VIVANT et invisible — le compter mort
//	                         rendait « isolee » une mort survenue a trois metres d'un
//	                         coequipier (defaut P0-1). MAIS un coequipier VU A PORTEE
//	                         tranche : la mort est accompagnee, quel que soit le reste.
//	POSITION INCONNUE        la mort elle-meme n'a pas de lieu (embarquement sans point de
//	                         vehicule) : ni peinte ni examinee, comptee a part.
//	VARIANTE SANS RAYON      le match entier sort de la lecture, et il se COMPTE au niveau
//	                         du MATCH — pas au fil des morts, sinon un match ou l'on ne
//	                         meurt pas ne serait jamais signale. Un rayon devine rendrait
//	                         une mesure d'apparence normale sur une regle qu'on n'a pas
//	                         mesuree.
//
// UN ADVERSAIRE PROCHE N'ANNULE RIEN : la question porte sur le SOUTIEN, pas sur la
// solitude. Mourir a deux metres d'un ennemi et a quarante de son equipe, c'est mourir
// isole — c'est meme le cas typique.

import "levelup/go-api/internal/domain"

// Isolement tranche l'isolement de chaque mort, rayon PAR MATCH.
//
// `rayonParMatch` : matchID -> rayon en metres. IL EST AUSSI L'UNIVERS DE LA LECTURE : un
// match qui n'y figure pas n'a pas de portee mesuree, ses morts sont ecartees, et il compte
// dans MatchsSansRayon. Le compte se fait sur CETTE TABLE et sur `matchsMesures`, jamais sur
// les morts parcourues — un match sans mort du joueur serait sinon invisible.
//
// `matchsMesures` : les matchs mesures AYANT un rayon. C'est le denominateur « par match »,
// et il vient de l'appelant : un match sans aucune mort examinee reste un match joue.
func Isolement(morts []domain.MortAExaminer, rayonParMatch map[string]float64,
	matchsMesures int) domain.BilanIsolement {
	b := domain.BilanIsolement{Isolees: []domain.MortAExaminer{}}
	for _, m := range morts {
		rayon, ok := rayonParMatch[m.MatchID]
		if !ok || rayon <= 0 {
			// Le match est deja compte au niveau du match par l'appelant : ici on se
			// contente d'ecarter ses morts.
			continue
		}
		switch verdict(m, rayon) {
		case verdictAccompagne:
			b.Examinees++
		case verdictIsolee:
			b.Examinees++
			b.Isolees = append(b.Isolees, m)
		case verdictEquipeATerre:
			b.SansCoequipierVivant++
		case verdictIndetermine:
			b.Indeterminees++
		case verdictSansPosition:
			b.PositionInconnue++
		}
	}
	b.Couverture = Mesurer(len(b.Isolees), b.Examinees, matchsMesures)
	return b
}

// Les cinq verdicts possibles d'une mort.
type verdictMort int

const (
	verdictAccompagne verdictMort = iota
	verdictIsolee
	verdictEquipeATerre
	verdictIndetermine
	verdictSansPosition
)

// verdict tranche UNE mort.
//
// L'ORDRE DES TESTS EST LA REGLE, et il n'est pas interchangeable : un coequipier VU A
// PORTEE tranche AVANT tout le reste. Sans cela, une mort survenue a deux metres d'un
// coequipier serait rangee « indeterminee » au motif qu'un TROISIEME joueur etait en
// vehicule — on perdrait une mesure certaine a cause d'une incertitude sans effet.
func verdict(m domain.MortAExaminer, rayon float64) verdictMort {
	if m.PositionInconnue {
		return verdictSansPosition
	}
	vus, inconnus, morts := 0, 0, 0
	for _, c := range m.Coequipiers {
		switch c.Statut {
		case domain.StatutVoisinVivant:
			if c.DistanceM <= rayon {
				// LA BORNE EST INCLUSIVE : le rayon est la portee du radar, et se voir
				// juste a la limite, c'est se voir.
				return verdictAccompagne
			}
			vus++
		case domain.StatutVoisinInconnu:
			inconnus++
		default:
			morts++
		}
	}
	if inconnus > 0 {
		// Personne de vu a portee, et au moins un invisible : on ne peut ni affirmer ni
		// infirmer. On ne tranche pas.
		return verdictIndetermine
	}
	if vus == 0 {
		// Aucun coequipier vu, aucun inconnu : tous sont SUS morts (ou il n'y en a aucun,
		// ce qui revient au meme pour le placement).
		return verdictEquipeATerre
	}
	return verdictIsolee
}
