package grammar

// recuperations.go — LA COUCHE DE RECUPERATION DU FILM, RELEVEE UNE FOIS PAR CONTEXTE (ADR 0037
// IR-6 ; lots 2.4 et 2.5 du plan de l etape 2).
//
// Trois releves heuristiques que plusieurs lecteurs de la cuisson demandaient chacun pour son
// compte, avec les memes parametres : l ancrage des records bipedes (`ancres_bipedes.go`), les
// pistes des objets du monde (`pistes_du_monde.go`) et leurs creations (`creations_du_monde.go`).
// Chacun se fait une fois, au premier lecteur, et se range ici, dans le contexte du film — comme les
// lectures bipedes de la marche, que l ancrage complete derriere elle (`lectures_bipedes.go`).

// memoDesRecuperations porte ce que la couche de recuperation a releve pour un film.
type memoDesRecuperations struct {
	ancres    *ancresBipedes
	pistes    []*pistesRelevees
	creations []*creationsRelevees
	// objets : les records d objets du monde que la marche a lus ([objetsDuMondeLus]).
	objets *objetsDuMondeLus
	// lectures : les lectures bipedes de la marche et de l ancrage derriere elle ([lecturesBipedes]).
	lectures *lecturesBipedes
}
