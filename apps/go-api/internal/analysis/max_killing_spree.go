// Package analysis — max_killing_spree.go : calcul title-agnostic de la « folie
// meurtrière max » (max killing spree) d'un joueur sur un match, dérivé des events
// kill/death HORODATÉS.
//
// Définition (alignée sur la sémantique jeu) : le plus grand nombre de kills réalisés
// d'affilée par le joueur AVANT de mourir. Le compteur s'incrémente à chaque kill du
// joueur et se remet à zéro à chaque mort du joueur ; on retient le maximum atteint.
//
// Pur, sans dépendance DB / titre. Les events sont passés en canonical.HighlightEvent
// (même substrat que la synthèse kvPairs → kill/death, cf. kv_synthetic_events.go) :
// un titre qui ne porte pas ses kills dans highlight_events (Halo 5) les fournit via
// SynthesizeKillEventsFromKVPairs, et ce calcul s'applique tel quel.
//
// Convention de champ XUID (héritée de shared.highlight_events) : pour un event "kill"
// XUID = tueur, pour un event "death" XUID = victime. On filtre donc sur XUID == xuid
// pour les deux types : un kill du joueur (il est le tueur) incrémente, une mort du
// joueur (il est la victime) reset.
package analysis

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// ComputeMaxKillingSpree calcule la folie meurtrière max du joueur `xuid` à partir de
// ses events kill/death. Les events sont triés par TimeMS croissant (copie locale, ne
// mute pas l'entrée) avant le balayage : kill du joueur → compteur+1 (et MAJ du max),
// mort du joueur → compteur remis à 0. Les events des autres joueurs et les autres
// types (médaille, assist, …) sont ignorés.
//
// Retourne 0 si xuid est vide, si aucun event, ou si le joueur n'a aucun kill.
func ComputeMaxKillingSpree(events []canonical.HighlightEvent, xuid string) int {
	if xuid == "" || len(events) == 0 {
		return 0
	}

	// Copie locale triée par TimeMS (stable) — ne pas muter le slice du caller.
	ordered := make([]canonical.HighlightEvent, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].TimeMS < ordered[j].TimeMS })

	maxSpree, current := 0, 0
	for _, e := range ordered {
		if e.XUID != xuid {
			continue
		}
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventKill:
			current++
			if current > maxSpree {
				maxSpree = current
			}
		case canonical.EventDeath:
			current = 0
		}
	}
	return maxSpree
}
