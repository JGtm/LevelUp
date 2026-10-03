package domain

// isolation.go — MOURIR SEUL : les types de la lecture d'isolement (lot 7C, 2026-09-07).
//
// ILS VIVENT ICI ET NON DANS `analysis/coordination` (arch-rules, et le ratchet
// `no_naked_rate_test` de ce paquet-la le verifie) : un type de RESULTAT traverse la
// frontiere algo -> service -> handler, et un algo qui exporte sa forme de sortie fige son
// appelant sur son implementation.
//
// # LE FAIT EST DEJA CALCULE, LA LECTURE NE FAIT QUE TRANCHER
//
// Une premiere version deduisait la vitalite des coequipiers a la LECTURE, sur la chronologie
// d'un artefact de rejeu. Elle demandait au film ce qu'il ne sait pas dire (`replay/owners.go`
// nomme aussi une vie par FERMETURE DE SLOT) et faisait dependre un fait de base du calendrier
// de cuisson. Le fait se produit desormais AU SYNC, dans `match_death_context` : chaque mort du
// journal y porte le nombre de coequipiers dans chacun des quatre etats, et la distance au plus
// proche de ceux qu'on VOYAIT. Ne reste ici qu'une comparaison a un seuil.

// MortAExaminer est une mort de mon camp, telle que la base la rend : son lieu, et ce que le
// contexte dit de son voisinage.
type MortAExaminer struct {
	MatchID string
	// X, Y situent la mort (`kill_positions_latest`, position de la VICTIME) — elles
	// voyagent pour que la carte puisse peindre les morts isolees sans re-joindre.
	X, Y float64

	// PlusProcheM : distance au coequipier VISIBLE le plus proche, en metres. nil = aucun
	// coequipier visible. UNE ABSENCE DE MESURE, JAMAIS UNE DISTANCE INFINIE — la
	// distinction porte tout le verdict.
	PlusProcheM *float64

	// Visibles et HorsDeVue comptent les coequipiers VIVANTS a cet instant : ceux que le
	// film montrait, et ceux qu'il ne montrait pas (vehicule non replique) mais qui
	// n'attendaient pas leur reapparition et n'avaient pas quitte.
	//
	// LEUR SOMME DECIDE SI LA MORT EST EXAMINABLE. A zero, personne ne pouvait accompagner :
	// la mort ne dit rien du placement et sort du denominateur.
	Visibles, HorsDeVue int
}

// EquipeATerre dit qu'aucun coequipier n'etait en mesure d'accompagner cette mort.
//
// ON NE PEUT PAS ETRE MAL ACCOMPAGNE QUAND PERSONNE NE PEUT ACCOMPAGNER : compter ces morts au
// denominateur ferait monter le taux d'isolement avec les hecatombes de son equipe, c'est-a-dire
// avec quelque chose que le placement du joueur ne commande pas.
func (m MortAExaminer) EquipeATerre() bool { return m.Visibles+m.HorsDeVue == 0 }

// BilanIsolement est ce que la lecture rend : le taux canonique, les morts a peindre, et les
// deux comptes de ce qu'elle a ECARTE.
type BilanIsolement struct {
	// Isolees : les morts sans coequipier visible a portee. Elles se peignent sur la carte.
	Isolees []MortAExaminer

	// Couverture porte le taux SOUS SA FORME CANONIQUE (taux + brut + par match + N +
	// echantillon faible) — jamais un nombre seul.
	Couverture Couverture

	// Examinees : le denominateur, les morts ayant au moins un coequipier en mesure
	// d'accompagner.
	Examinees int

	// EquipeATerre : les morts ECARTEES parce que personne ne pouvait accompagner. PUBLIEE,
	// et c'est le point : elle etait comptee sans jamais sortir, si bien qu'un denominateur
	// ampute ressemblait a un denominateur complet.
	EquipeATerre int

	// MatchsSansRayon : les matchs dont la variante n'a pas de portee de radar mesuree. Ils
	// sortent de l'UNIVERS de la lecture, pas seulement de son numerateur — les laisser au
	// denominateur diviserait la mesure par des matchs qu'on a refuse de lire (defaut deja
	// corrige trois fois sous « correction G2 »).
	MatchsSansRayon int
}
