package main

// Diagnostic helper séparé : dump complet BIPED #35 i12..i46 + statut ported.
// Appelé via -diag (voir main si besoin). Ici une fonction exportée appelée au boot.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// dumpBipedTail liste i12..i46 et marque les composants probablement non-portés
// (ceux qui ne sont pas dans le switch consumeByName). Comme consumeByName est privé,
// on infère le statut via une traversée contrôlée : on force un mask à un seul bit i
// et on voit si TraverseEntity rapporte ported pour ce composant.
func dumpBipedTail(reg *filmdec.Registry) {
	biped, _ := reg.Archetype(35)
	fmt.Println("\n=== BIPED #35 : i12..i46 (entre dead-state et held-weapon) ===")
	ported := portedSet()
	for i := 12; i <= 46 && i < len(biped.Components); i++ {
		name := biped.Components[i]
		mark := "  NON-PORTÉ (desync ici si présent)"
		if ported[name] {
			mark = "  porté"
		}
		fmt.Printf("  i%-2d %-50s%s\n", i, name, mark)
	}
}

// portedSet réplique la liste des noms portés dans consumeByName (source de vérité :
// traverse.go). Maintenu à la main pour le diagnostic.
func portedSet() map[string]bool {
	names := []string{
		"object-position-dynamic-precision-component",
		"object-translational-velocity-dynamic-precision-component",
		"object-angular-velocity-dynamic-precision-component",
		"object-region-state-component",
		"object-damage-sections-component",
		"object-constraint-component",
		"object-parent-state-component",
		"object-scale-component",
		"object-maximum-vitalities-component",
		"object-dissolver-component",
		"object-low-frequency-component",
		"object-physics-flags-component",
		"object-frame-configuration-component",
		"unit-actor-control-component",
		"unit-actor-state-component",
		"unit-malleable-property-component",
		"object-multiplayer-properties-component",
		"weapon-state-type-info",
		"object-forward-and-up-component",
		"object-body-vitality-component",
		"object-shield-vitality-component",
		"object-dead-state-component",
		"weapon-state-ammo",
		"weapon-state-rounds-inventory",
		"weapon-state-overheated",
		"biped-desired-weapon-set",
		"unit-control-component",
		"unit-grenade-counts-component",
		"unit-equipment-component",
		"unit-crouch-component",
		"unit-active-camo-state-component",
		"unit-command-tick-component",
		"unit-low-frequency-component",
		"unit-stun-component",
		"unit-desired-aiming-vector-component",
	}
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}
