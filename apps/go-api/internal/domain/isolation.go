package domain

// isolation.go — MOURIR SEUL : les types de la lecture d'isolement.
//
// ILS VIVENT ICI ET NON DANS `analysis/coordination` (arch-rules, et le ratchet
// `no_naked_rate_test` de ce paquet-la le verifie) : un type de RESULTAT traverse la
// frontiere algo -> service -> handler, et un algo qui exporte sa forme de sortie fige son
// appelant sur son implementation. C'est aussi ce qui empeche d'emballer un taux dans une
// struct maison pour contourner la liste blanche des types de retour.

// MortAExaminer est une mort dont on veut savoir si elle fut isolee.
//
// `DistancesCoequipiers` ne porte QUE des coequipiers VIVANTS a cet instant : le tri par
// equipe et par vitalite est fait par l'appelant, qui seul connait les camps.
type MortAExaminer struct {
	MatchID string
	// Frame et X, Y situent la mort — ils voyagent pour que l'appelant puisse peindre les
	// morts isolees sans re-joindre quoi que ce soit.
	Frame int
	X, Y  float64
	// DistancesCoequipiers : en metres, les coequipiers vivants. VIDE = aucun coequipier
	// debout, donc mort EXCLUE du denominateur.
	DistancesCoequipiers []float64
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
	// SansCoequipierVivant : les morts ECARTEES parce que toute l'equipe etait deja a
	// terre. Publie plutot qu'avale — c'est une part du jeu, pas un detail.
	SansCoequipierVivant int
	// MatchsSansRayon : les matchs dont la variante n'est pas dans la table du rayon. Leurs
	// morts ne sont ni examinees ni isolees ; le pied de carte doit pouvoir le dire.
	MatchsSansRayon int
}
