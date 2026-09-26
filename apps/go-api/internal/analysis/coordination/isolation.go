package coordination

// isolation.go — OU JE MEURS ISOLE.
//
// # LA QUESTION, ET CE QU'IL RESTE A DECIDER ICI
//
// « Un coequipier etait-il assez pres pour peser sur l'echange ? » Le FAIT est deja etabli au
// sync (`match_death_context` : combien de coequipiers dans chacun des quatre etats, et a
// quelle distance etait le plus proche de ceux qu'on VOYAIT). Ce fichier ne fait plus qu'une
// chose : comparer cette distance au RAYON DU MATCH, et compter. Il ne connait ni base, ni
// fichier, ni carte.
//
// # TROIS SORTIES, ET AUCUNE NE SE CONFOND AVEC UNE AUTRE
//
//	ACCOMPAGNEE      au moins un coequipier VISIBLE a portee. Au denominateur, pas au
//	                 numerateur.
//	ISOLEE           au moins un coequipier en mesure d'accompagner, aucun visible a portee.
//	                 Au denominateur ET au numerateur ; elle se peint sur la carte.
//	EQUIPE A TERRE   personne ne pouvait accompagner (`visibles + hors_de_vue == 0`). ECARTEE
//	                 du denominateur ET COMPTEE : elle ne dit rien du placement, mais son
//	                 nombre doit sortir — un denominateur ampute ressemble sinon a un
//	                 denominateur complet.
//
// # UN COEQUIPIER VIVANT MAIS HORS DE VUE NE SAUVE PAS LA MORT, ET NE L'ECARTE PAS NON PLUS
//
// C'est le cas du joueur en vehicule, dont le biped cesse d'etre replique. Il PEUT accompagner
// (la mort reste examinable), mais on ne sait pas ou il est (il ne peut pas etre « a portee »).
// Le compter mort ferait sortir « equipe a terre » une mort survenue a trois metres d'un
// coequipier en Warthog ; le compter present a portee inventerait une distance. Les deux
// erreurs ont ete commises, dans cet ordre.

import "levelup/go-api/internal/domain"

// Isolement compare chaque mort au rayon de SON match et rend le bilan.
//
// LE RAYON EST PAR MATCH, et c'est ce qui rend la lecture juste sur un filtre mixte : 18 m en
// Arene, 24 m en BTB. Un rayon unique melangerait deux regles de jeu sous une seule mesure — et
// un filtre qui contient les deux formats est le cas normal.
//
// UN MATCH SANS RAYON N'ENTRE PAS : ses morts ne sont ni examinees ni comptees isolees, et le
// compte des matchs ecartes est pose par l'appelant, qui seul connait l'univers.
func Isolement(morts []domain.MortAExaminer, rayonParMatch map[string]float64,
	matchsMesures int,
) domain.BilanIsolement {
	b := domain.BilanIsolement{Isolees: []domain.MortAExaminer{}}
	for _, m := range morts {
		rayon, connu := rayonParMatch[m.MatchID]
		if !connu || rayon <= 0 {
			continue
		}
		if m.EquipeATerre() {
			b.EquipeATerre++
			continue
		}
		b.Examinees++
		if !accompagnee(m, rayon) {
			b.Isolees = append(b.Isolees, m)
		}
	}
	// LE TAUX SORT SOUS SA FORME CANONIQUE, jamais nu : `Mesurer` pose le brut, le
	// denominateur, la quantite par match et la reserve d'echantillon faible (30 morts).
	b.Couverture = Mesurer(len(b.Isolees), b.Examinees, matchsMesures)
	return b
}

// accompagnee : au moins un coequipier VISIBLE a portee.
//
// LA BORNE EST INCLUSIVE : le rayon est la portee du radar, et se voir juste a la limite, c'est
// se voir. Une distance ABSENTE (`nil`) veut dire « aucun coequipier visible » — pas « a
// distance nulle » ni « a distance infinie » : la mort n'est donc pas accompagnee, et elle est
// bien examinee puisqu'un coequipier hors de vue POUVAIT accompagner.
func accompagnee(m domain.MortAExaminer, rayon float64) bool {
	return m.PlusProcheM != nil && *m.PlusProcheM <= rayon
}
