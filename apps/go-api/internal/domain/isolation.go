package domain

// isolation.go — MOURIR SEUL : les types de la lecture d'isolement.
//
// ILS VIVENT ICI ET NON DANS `analysis/coordination` (arch-rules, et le ratchet
// `no_naked_rate_test` de ce paquet-la le verifie) : un type de RESULTAT traverse la
// frontiere algo -> service -> handler, et un algo qui exporte sa forme de sortie fige son
// appelant sur son implementation. C'est aussi ce qui empeche d'emballer un taux dans une
// struct maison pour contourner la liste blanche des types de retour.

// MortAExaminer est une mort dont on veut savoir si elle fut isolee.
type MortAExaminer struct {
	MatchID string
	// X, Y situent la mort — elles voyagent pour que l'appelant puisse peindre les morts
	// isolees sans re-joindre quoi que ce soit. `Frame`, lui, N'EST PAS ICI : il est ecrit
	// dans le SIDECAR (consommateur nomme : le drilldown de la phase 5), mais la lecture
	// d'isolement ne le lit jamais — un champ porte sans lecteur est du vocabulaire mort.
	X, Y float64

	// PositionInconnue : le film ne dit pas OU cette mort a eu lieu (embarquement sans
	// point de vehicule). Elle n'est ni peinte ni examinee, et se compte a part.
	PositionInconnue bool

	// Coequipiers porte, pour CHAQUE coequipier du mort, ce qu'on sait de lui a cet
	// instant. Le tri par equipe est fait par l'appelant, qui seul connait les camps.
	//
	// UNE LISTE VIDE VEUT DIRE « aucun coequipier du tout » — ce qui, pour le placement,
	// revient au meme que « toute l'equipe a terre » : on ne peut pas etre mal accompagne.
	Coequipiers []StatutCoequipier
}

// StatutCoequipier est ce que le film sait d'un coequipier a l'instant d'une mort.
type StatutCoequipier struct {
	// Statut vaut StatutVoisinVivant / StatutVoisinMort / StatutVoisinInconnu.
	Statut string
	// DistanceM n'a de sens que sous StatutVoisinVivant.
	DistanceM float64
}

// BilanIsolement est ce que la lecture rend.
type BilanIsolement struct {
	// Isolees : les morts sans aucun coequipier dans le rayon. Ce sont elles que la carte
	// peint.
	Isolees []MortAExaminer
	// Couverture porte le taux (isolees / examinees), son compte brut, la quantite par
	// match, la taille de l'echantillon et le drapeau d'echantillon faible. C'est la SEULE
	// forme sous laquelle un taux sort de ce paquet (cf. measure.go).
	Couverture Couverture
	// Examinees est le denominateur : les morts qui avaient au moins un coequipier vivant,
	// dans un match dont le rayon est connu.
	Examinees int
	// SansCoequipierVivant : les morts ECARTEES parce que toute l'equipe etait SUE a terre
	// — tous les coequipiers au statut `mort`. Publie plutot qu'avale : c'est une part du
	// jeu, pas un detail.
	SansCoequipierVivant int
	// Indeterminees : les morts ECARTEES parce qu'au moins un coequipier etait INVISIBLE
	// (en vehicule non attribue, ou survivant anonyme) et qu'aucun coequipier vu n'etait a
	// portee. On ne peut ni dire « isolee » ni dire « accompagnee » : les compter isolees
	// etait le defaut P0-1 de la revue.
	Indeterminees int
	// PositionInconnue : les morts ECARTEES parce que le film ne dit pas OU elles ont eu
	// lieu (embarquement sans point de vehicule). Elles ne sont ni peintes ni examinees.
	PositionInconnue int
	// MatchsSansRayon : les matchs dont la variante n'est pas dans la table du rayon. Leurs
	// morts ne sont ni examinees ni isolees ; le pied de carte doit pouvoir le dire.
	MatchsSansRayon int
}
