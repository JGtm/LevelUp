package coordination

// statut_plus_proche.go — LE STATUT D UNE MORT PAR SON COEQUIPIER VISIBLE LE PLUS PROCHE.
//
// QUATRE STATUTS, qui se partagent toutes les morts d'un match au rayon connu — la meme regle que
// `Isolement`, ecrite ici une fois pour la fiche d'une mort :
//
//	accompagnee         un coequipier visible a portee (`APortee`, borne inclusive) ;
//	visible_hors_rayon  un coequipier visible, mais au-dela de la portee ;
//	aucun_visible       aucun visible, au moins un coequipier vivant hors de vue ;
//	equipe_a_terre      personne ne pouvait accompagner (aucun visible, aucun hors de vue).

import "levelup/go-api/internal/domain"

// StatutDeLaMort rend le statut d'une mort. `plusProcheM` est la distance au coequipier visible le
// plus proche (nil = aucun visible), `rayon` la portee du radar du match.
func StatutDeLaMort(visibles, horsDeVue int, plusProcheM *float64, rayon float64) string {
	switch {
	case visibles+horsDeVue == 0:
		return domain.TacticalStatutEquipeATerre
	case plusProcheM == nil:
		return domain.TacticalStatutAucunVisible
	case APortee(plusProcheM, rayon):
		return domain.TacticalStatutAccompagnee
	default:
		return domain.TacticalStatutVisibleHorsRayon
	}
}
