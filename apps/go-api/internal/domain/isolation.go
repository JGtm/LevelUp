package domain

// isolation.go — MOURIR SEUL : les types de la lecture d'isolement.
//
// ILS VIVENT ICI ET NON DANS `analysis/coordination` (arch-rules, et le ratchet
// `no_naked_rate_test` de ce paquet-la le verifie) : un type de RESULTAT traverse la
// frontiere algo -> service -> handler, et un algo qui exporte sa forme de sortie fige son
// appelant sur son implementation.
//
// # LE MODELE
//
// Les morts viennent de la BASE (journal des morts) ; les positions viennent du sidecar,
// qui ne juge rien. Le service assemble les deux et rend, pour chaque mort de mon camp,
// l'etat de chaque coequipier : present ou non, et a quelle distance.
//
// CE QUE « PRESENT » VEUT DIRE EST UNE DECISION PRODUIT EN COURS (2026-09-07). La lecture
// tient en attendant sur « une position connue a cet instant ». Le modele vise — mort au
// journal, reapparition observee ou delai MESURE sur le match, depart lu dans la base — est
// ecrit et consigne au registre des reports avec sa condition de reprise ; le point
// d'attente est commente sur pieces dans service/tactical_service_lectures.go.

// MortAExaminer est une mort du journal, avec l'etat de chaque coequipier a cet instant.
type MortAExaminer struct {
	MatchID string
	// X, Y situent la mort — elles voyagent pour que l'appelant puisse peindre les morts
	// isolees sans re-joindre quoi que ce soit.
	X, Y float64

	// Coequipiers porte, pour CHAQUE coequipier du mort, ce que la lecture a etabli de lui
	// a cet instant. Le tri par equipe et la resolution de vitalite sont faits par
	// l'appelant, qui seul a le journal et les departs.
	//
	// UNE LISTE VIDE VEUT DIRE « aucun coequipier du tout » (joueur solo de son camp) — ce
	// qui, pour le placement, revient au meme que « toute l'equipe a terre » : on ne peut
	// pas etre mal accompagne.
	Coequipiers []EtatCoequipier
}

// EtatCoequipier est ce que la lecture sait d'un coequipier a l'instant d'une mort.
type EtatCoequipier struct {
	// Vivant : il n'avait pas quitte, et il n'etait pas en attente de reapparition.
	Vivant bool
	// DistanceM n'a de sens que si Vivant. C'est la distance a la DERNIERE position connue
	// du coequipier — decision utilisateur : un joueur vivant est quelque part, et sa
	// derniere position vaut mieux qu'un « inconnu » qui ne se mesure pas.
	DistanceM float64
	// PositionConnue : faux quand le film n'a jamais donne de position de ce joueur avant
	// cet instant. Il est alors vivant mais nulle part : il ne peut ni accompagner ni
	// prouver l'isolement.
	PositionConnue bool
}

// BilanIsolement est ce que la lecture rend.
type BilanIsolement struct {
	// Isolees : les morts sans aucun coequipier vivant dans le rayon. Ce sont elles que la
	// carte peint.
	Isolees []MortAExaminer
	// Couverture porte le taux (isolees / examinees), son compte brut, la quantite par
	// match, la taille de l'echantillon et le drapeau d'echantillon faible. C'est la SEULE
	// forme sous laquelle un taux sort du paquet de coordination (cf. measure.go).
	Couverture Couverture
	// Examinees est le denominateur : les morts qui avaient au moins un coequipier vivant,
	// dans un match dont le rayon est connu.
	Examinees int
	// EquipeATerre : les morts ECARTEES parce que tous les coequipiers etaient morts ou
	// partis. Elles ne disent rien du placement — on ne peut pas etre mal accompagne quand
	// personne ne peut accompagner — et les compter isolees ferait grimper le taux de
	// l'equipe qui perd un combat entier, c'est-a-dire mesurer la defaite.
	EquipeATerre int
	// MatchsSansRayon : les matchs dont la variante n'a pas de portee mesuree. Leurs morts
	// ne sont ni examinees ni isolees ; le pied de carte doit pouvoir le dire.
	MatchsSansRayon int
}
