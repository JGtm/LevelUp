package replay

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// player_index.go — CE QUE LA PUBLICATION FAIT DE LA TABLE D INDEX DE JOUEUR.
//
// LA LECTURE EST DESCENDUE EN `grammar` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM,
// DU-3 = S1) : `grammar.ScanPlayerIndices` lit l index de chaque xuid dans les chunks de
// replication, et son en-tete porte le raisonnement (le lien est ECRIT dans le film ; les
// lectures doivent CONCORDER). Restent ici les deux decisions de publication : le roster qu on
// lui donne et l injectivite exigee de la table.

// rosterFromDeaths rend les xuids distincts du fil des morts, en ordre stable. C'est le seul
// roster dont le rejeu dispose sans base de données — et il est déjà, lui aussi, une lecture.
func rosterFromDeaths(deaths []types.Death) []uint64 {
	return rosterOf(deaths, nil)
}

// rosterOf rend les xuids du fil des morts COMPLÉTÉS par ceux que l'appelant fournit, en
// ordre stable.
//
// POURQUOI LE COMPLÉMENT EXISTE : un joueur qui ne meurt jamais n'apparaît dans aucun
// enregistrement du fil, donc dans aucun roster qui en dérive — et il est alors invisible
// jusqu'au bout de la chaîne (pas d'index de joueur, pas de pont, pas d'entrée au roster
// publié). Mesure : `3372e7eb`, 6 joueurs publiés pour 8, les deux manquants à 0 mort.
//
// `extra` vide rend EXACTEMENT ce que le fil des morts donne : c'est ce qui garde le rejeu
// publiable hors ligne (cf. [Options.RosterXUIDs]).
func rosterOf(deaths []types.Death, extra []uint64) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(deaths)+len(extra))
	ajouter := func(x uint64) {
		if x == 0 || seen[x] {
			return
		}
		seen[x] = true
		out = append(out, x)
	}
	for _, d := range deaths {
		ajouter(d.XUID)
	}
	for _, x := range extra {
		ajouter(x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// injectiveOrEmpty rend la table si elle est injective (deux joueurs ne peuvent pas partager
// un index), et une table VIDE sinon.
//
// L'injectivité n'est pas une préférence esthétique : un index partagé placerait les tirs de
// deux joueurs sur la même trace, sans que rien ne le signale.
func injectiveOrEmpty(t types.PlayerIndexTable) (types.PlayerIndexTable, int) {
	byIdx := map[int]int{}
	for _, pi := range t.ByXUID {
		byIdx[pi]++
	}
	collisions := 0
	for _, n := range byIdx {
		if n > 1 {
			collisions++
		}
	}
	if collisions > 0 {
		return types.PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: t.Readings,
			Disagreements: t.Disagreements}, collisions
	}
	return t, 0
}
