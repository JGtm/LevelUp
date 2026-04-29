package halo_infinite

import "testing"

func TestInferModeCategoryFromPairName(t *testing.T) {
	cases := []struct {
		pairName string
		want     string
	}{
		// Format standard
		{"Arena:Slayer", ModeCategoryAssassin},
		{"Arena:Slayer on Bazaar", ModeCategoryAssassin},
		{"Arena:CTF on Recharge", ModeCategoryAssassin},
		{"Tactical:Slayer", ModeCategoryAssassin},
		{"Community:Team Slayer on Solution", ModeCategoryAssassin},
		{"Super Fiesta:Slayer on Catalyst - Forge", ModeCategorySuperFiesta},
		{"Fiesta:Slayer", ModeCategoryFiesta},
		{"Husky Raid:Slayer", ModeCategoryHuskyRaid},
		{"BTB:Slayer", ModeCategoryBTB},
		{"BTB Heavies:CTF", ModeCategoryBTB},
		{"Ranked:Slayer on Aquarius", ModeCategoryRanked},
		{"Firefight:KOTH", ModeCategoryFirefight},
		{"Gruntpocalypse:Slayer", ModeCategoryFirefight},
		// Sans sÃ©parateur (mode parent qui est lui-mÃªme une catÃ©gorie)
		{"Husky Raid", ModeCategoryHuskyRaid},
		{"BTB", ModeCategoryBTB},
		{"Castle Wars", ModeCategoryFiesta},
		// Format inversÃ© (prÃ©fixe Ã  droite)
		{"CTF:Arena", ModeCategoryAssassin},
		{"Slayer:Ranked", ModeCategoryRanked},
		// PrÃ©fixe inconnu â†’ Other
		{"Custom:Slayer", ModeCategoryOther},
		{"Slayer", ModeCategoryOther},
		// Casse normalisation
		{"super fiesta:slayer", ModeCategorySuperFiesta},
		{"BTB:slayer", ModeCategoryBTB},
		// Empty
		{"", ModeCategoryOther},
		{"   ", ModeCategoryOther},
	}
	for _, tc := range cases {
		t.Run(tc.pairName, func(t *testing.T) {
			got := InferModeCategoryFromPairName(tc.pairName)
			if got != tc.want {
				t.Errorf("InferModeCategoryFromPairName(%q) = %q, want %q", tc.pairName, got, tc.want)
			}
		})
	}
}

func TestPairNamePrefixesForCategory(t *testing.T) {
	cases := []struct {
		category string
		want     map[string]bool
	}{
		{ModeCategoryFiesta, map[string]bool{"Fiesta": true, "Castle Wars": true}},
		{ModeCategorySuperFiesta, map[string]bool{"Super Fiesta": true}},
		{ModeCategoryHuskyRaid, map[string]bool{"Husky Raid": true, "Super Husky Raid": true}},
		{ModeCategoryBTB, map[string]bool{"BTB": true, "BTB Heavies": true}},
		{ModeCategoryRanked, map[string]bool{"Ranked": true}},
		{ModeCategoryAssassin, map[string]bool{
			"Arena": true, "Tactical": true, "Assault": true, "Community": true,
		}},
		{ModeCategoryFirefight, map[string]bool{"Firefight": true, "Gruntpocalypse": true}},
		{ModeCategoryOther, map[string]bool{}}, // Other = NIL cÃ´tÃ© Go (l'appelant utilise AllKnownPairNamePrefixes pour NOT IN)
	}
	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			got := PairNamePrefixesForCategory(tc.category)
			gotSet := make(map[string]bool, len(got))
			for _, p := range got {
				gotSet[p] = true
			}
			if len(gotSet) != len(tc.want) {
				t.Errorf("category %q: got %v, want %v", tc.category, got, tc.want)
			}
			for k := range tc.want {
				if !gotSet[k] {
					t.Errorf("category %q: missing prefix %q (got %v)", tc.category, k, got)
				}
			}
		})
	}
}
