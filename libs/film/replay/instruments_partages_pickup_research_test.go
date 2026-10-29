//go:build research

package replay

// instruments_partages_pickup_research_test.go — graine de temoin et aide de pads utilisees par des instruments research
// (J12.7 lint). Deplacement pur.

import "fmt"

// gwPickupWitnessSeed fige le tirage des temoins. Un temoin qui change d'une execution a
// l'autre n'est pas un temoin : c'est un bruit qu'on relance jusqu'a ce qu'il arrange.
const gwPickupWitnessSeed = 20260817

func gwPadsPart(k, n int) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d=%.1f%%", k, n, 100*float64(k)/float64(n))
}
