package replay

import (
	"log/slog"
	"sort"
)

// inventory_dead_readings.go — POURQUOI CETTE LECTURE D'INVENTAIRE EST VIDE.
//
// CE QUE LE CROISEMENT AJOUTE. `buildInventory` marque toute lecture sans grenade ni munition
// (`Empty = unknown`) : c'est tout ce que le DÉCODEUR peut dire. Ici, à l'assemblage, le FIL DES
// MORTS est disponible — et lui sait si le porteur du slot était mort à cet instant. Les
// lectures corroborées passent à `dead` ; les autres gardent `unknown`.
//
// POURQUOI ICI ET PAS DANS LE DÉCODEUR. Le fil des morts n'est pas une entrée de la projection
// des inventaires ; il l'est de l'ASSEMBLAGE, qui le possède déjà (`Options.Deaths`) et qui a
// résolu son décalage d'horloge (`OwnerReport.DeathOffsetMS`). Faire descendre les morts dans le
// projecteur mêlerait deux sources que l'artefact tient séparées partout ailleurs.
//
// LA PREUVE, ET SON TÉMOIN. Sur 8 films (1 419 records d'image-clé, 247 lectures vides) :
// 88,3 % des lectures VIDES tombent dans les 8 s qui suivent une mort de leur porteur, contre
// 1,1 % des lectures PLEINES soumises à la même fenêtre — un rapport de 82x. Sur le seul film
// de vérité terrain : 93,8 % contre 0,7 %, soit 137x. Détail, balayage de fenêtres et
// reproduction : inventory_mort_recouvrement_test.go.

// markInventoryDeadReadings requalifie en `dead` les lectures vides que le fil des morts
// corrobore. Rend le nombre de lectures requalifiées.
//
// SANS FIL DES MORTS, RIEN NE BOUGE : les lectures gardent `unknown`, ce qui est exactement ce
// qu'on sait d'elles. C'est une dégradation, et elle est journalisée par l'appelant avec les
// autres couvertures — jamais une requalification par défaut.
func markInventoryDeadReadings(inv []Inventory, deaths []Death, own OwnerReport, clk replayClock) int {
	if len(inv) == 0 || len(deaths) == 0 || len(own.SlotXUID) == 0 {
		return 0
	}
	byXUID := deathTimesByVictimMS(deaths, own.DeathOffsetMS)
	marked := 0
	for i := range inv {
		if inv[i].Empty != InventoryEmptyUnknown {
			continue
		}
		xuid, ok := own.SlotXUID[inv[i].Slot]
		if !ok {
			continue
		}
		// L'INSTANT SE RECONSTRUIT DEPUIS LA GRILLE, et l'arrondi n'a aucune portée ici : un pas
		// de grille vaut ~100 ms, la fenêtre 8 000. Relire l'horodatage brut du record obligerait
		// à faire voyager les entrées du décodeur jusqu'ici pour gagner un centième de fenêtre.
		tMS := int64((clk.origin + uint64(inv[i].T)*clk.step) / 1000)
		if since, has := invSinceLastDeathMS(byXUID[xuid], tMS); has && since <= invDeadWindowMS {
			inv[i].Empty = InventoryEmptyDead
			marked++
		}
	}
	return marked
}

// deathTimesByVictimMS regroupe les morts par VICTIME et les recale sur l'horloge du FILM.
//
// L'IDENTITÉ EST LE XUID, jamais un index (règle du chantier : un ordre n'est pas une identité).
// Le décalage vient de `bestDeathOffset` et s'applique dans le même sens que `nameLivesByDeaths` :
// instant_film = instant_match + offset.
func deathTimesByVictimMS(deaths []Death, offsetMS int64) map[uint64][]int64 {
	out := make(map[uint64][]int64, len(deaths))
	for _, d := range deaths {
		out[d.XUID] = append(out[d.XUID], d.TimeMS+offsetMS)
	}
	for x := range out {
		sort.Slice(out[x], func(i, j int) bool { return out[x][i] < out[x][j] })
	}
	return out
}

// invSinceLastDeathMS rend l'écart entre un instant et la mort du même joueur qui le PRÉCÈDE
// immédiatement.
//
// LE SECOND RETOUR EST FAUX QUAND AUCUNE MORT NE PRÉCÈDE, et il le faut : la première vie du
// match n'a pas de fenêtre de réapparition derrière elle. Rendre 0 ferait passer chaque lecture
// vide d'avant la première mort pour une mort fraîche.
func invSinceLastDeathMS(sorted []int64, tMS int64) (int64, bool) {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] > tMS })
	if i == 0 {
		return 0, false
	}
	return tMS - sorted[i-1], true
}

// logInventoryEmptyCoverage rend visible ce que le croisement a tranché. Une lecture vide non
// requalifiée n'est PAS une anomalie — c'est un trou du décodeur que le fil des morts n'explique
// pas —, mais son volume doit se lire, sinon une régression du pont slot->joueur passerait pour
// une amélioration du décodage.
func logInventoryEmptyCoverage(inv []Inventory, marked int) {
	empty := 0
	for _, i := range inv {
		if i.Empty != "" {
			empty++
		}
	}
	if empty == 0 {
		return
	}
	slog.Info("rejeu : lectures d'inventaire vides",
		"lectures", len(inv), "vides", empty,
		"morts", marked, "inexpliquees", empty-marked)
}
