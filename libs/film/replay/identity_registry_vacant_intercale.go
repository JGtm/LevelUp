package replay

// identity_registry_vacant_intercale.go — LE RANG ABSOLU D UNE TABLE A VACANT INTERCALE, CONFIRME PAR
// LES RECORDS DE CREATION.
//
// # LA QUESTION
//
// Quand un siege vacant tombe entre deux occupes de la table du film, le RANG ABSOLU d un siege (son
// index dans la table de 32, vacants compris) et son index PARMI LES OCCUPES divergent, et la table
// seule ne dit pas lequel des deux est l index de participant du film. Sans autre lecture, le
// registre s abstient (`vacant_intercale`) : la table entiere passe en repli.
//
// # LA LECTURE QUI TRANCHE
//
// Le record de creation de chaque bipede porte l index de participant de son proprietaire
// (identity_registry_creation.go). Les deux hypotheses y predisent des index DIFFERENTS : sous le
// rang absolu, aucun corps ne porte l index d un vacant intercale, et le dernier siege a un corps a
// son rang ; sous l index parmi les occupes, ce sont les index 0..n-1 qui portent des corps. Le rang
// absolu est donc CONFIRME, et la table employee, a deux conditions lues dans le film :
//
//  1. le rang absolu de CHAQUE siege occupe est l index d au moins un record de creation ;
//  2. aucun record de creation ne porte l index d un siege VACANT anterieur au dernier occupe —
//     les index en plus (arrivants, bots) tombent au-dela du dernier siege.
//
// Une seule des deux qui manque, et rien ne departage : le refus `vacant_intercale` reste, publie
// dans la couverture comme avant. Aucune table par film, aucun ordre d arrivee : la regle ne lit que
// la table du film et les records de creation du meme film.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// rangAbsoluConfirmeParLesCreations dit si les records de creation confirment que le rang absolu des
// sieges d une table LUE a vacant intercale est l index de participant (cf. l en-tete). Faux pour
// toute table sans vacant intercale, refusee ou vide : la question ne s y pose pas.
func rangAbsoluConfirmeParLesCreations(t grammar.FilmPlayerTable, creations []grammar.BipedCreation) bool {
	if t.Refusal != grammar.FilmTableRead || !t.InterleavedVacant || len(t.Seats) == 0 {
		return false
	}
	lus := map[int]bool{}
	for _, c := range creations {
		if c.HasIndex {
			lus[int(c.ParticipantIndex)] = true
		}
	}
	sieges, dernier := map[int]bool{}, -1
	for _, s := range t.Seats {
		if !lus[s.FilmIndex] {
			return false
		}
		sieges[s.FilmIndex], dernier = true, max(dernier, s.FilmIndex)
	}
	for i := range lus {
		if i < dernier && !sieges[i] {
			return false
		}
	}
	return true
}

// tableDuFilmEmployee dit si le registre emploie la table du film : lue sans vacant intercale, ou
// lue avec un vacant intercale dont les records de creation confirment le rang absolu. C est LE
// predicat de la composition ; les lecteurs du registre qui demandent « la table est-elle employee »
// lisent `couverture.Read`, qu il pose.
func tableDuFilmEmployee(in IdentityInput) bool {
	return in.FilmTable.Lue() || rangAbsoluConfirmeParLesCreations(in.FilmTable, in.BipedCreations)
}
