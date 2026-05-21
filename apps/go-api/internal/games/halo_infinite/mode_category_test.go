package halo_infinite

import "testing"

// ─── stripMapSuffix ─────────────────────────────────────────────────────

func TestStripMapSuffix_RemovesOnMap(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Arena:Slayer on Bazaar":          "Arena:Slayer",
		"BTB:CTF on Highpower":            "BTB:CTF",
		"Slayer on Aquarius":              "Slayer",
		"Super Fiesta:Slayer on Behemoth": "Super Fiesta:Slayer",
	}
	for in, want := range cases {
		if got := stripMapSuffix(in); got != want {
			t.Errorf("stripMapSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripMapSuffix_NoOnMap(t *testing.T) {
	t.Parallel()
	cases := []string{
		"Arena:Slayer",
		"Husky Raid",
		"",
		"BTB",
	}
	for _, in := range cases {
		if got := stripMapSuffix(in); got != in {
			t.Errorf("stripMapSuffix(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestStripMapSuffix_RemovesTechnicalIDSuffix(t *testing.T) {
	t.Parallel()
	// Le suffixe " - <8+ alphanum>" doit être stripé.
	got := stripMapSuffix("Slayer - ABCDEFGH123")
	if got != "Slayer" {
		t.Errorf("stripMapSuffix(Slayer - ABCDEFGH123) = %q, want Slayer", got)
	}
}

func TestStripMapSuffix_PreservesShortHyphen(t *testing.T) {
	t.Parallel()
	// Si le suffixe est <8 caractères → pas stripé.
	got := stripMapSuffix("Slayer - X")
	if got != "Slayer - X" {
		t.Errorf("stripMapSuffix(Slayer - X) = %q, want unchanged (suffixe < 8 chars)", got)
	}
}

// ─── normalizeModeCase ──────────────────────────────────────────────────

func TestNormalizeModeCase_KnownAcronyms(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"btb":              "BTB",
		"BTB":              "BTB",
		"btb heavies":      ModePrefixBTBHeavies,
		"super fiesta":     ModeCategorySuperFiesta,
		"super husky raid": ModePrefixSuperHuskyRaid,
		"husky raid":       ModeCategoryHuskyRaid,
		"castle wars":      ModePrefixCastleWars,
	}
	for in, want := range cases {
		if got := normalizeModeCase(in); got != want {
			t.Errorf("normalizeModeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeModeCase_AllUpperPreserved(t *testing.T) {
	t.Parallel()
	// Tout-majuscules → préservé tel quel (acronymes inconnus).
	cases := []string{"ARENA", "CTF", "BTB"}
	for _, in := range cases {
		if got := normalizeModeCase(in); got != in {
			t.Errorf("normalizeModeCase(%q) = %q, want %q (all-upper preserved)", in, got, in)
		}
	}
}

func TestNormalizeModeCase_TitleCase(t *testing.T) {
	t.Parallel()
	// Mots inconnus, casse mixte → title case sur chaque mot.
	cases := map[string]string{
		"team slayer":  "Team Slayer",
		"oddball mode": "Oddball Mode",
		"a b c":        "A B C",
		"single":       "Single",
	}
	for in, want := range cases {
		if got := normalizeModeCase(in); got != want {
			t.Errorf("normalizeModeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeModeCase_EmptyAndWhitespace(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "  ", "\t"} {
		if got := normalizeModeCase(in); got != "" {
			t.Errorf("normalizeModeCase(%q) = %q, want empty", in, got)
		}
	}
}

// ─── AllKnownPairNamePrefixes ───────────────────────────────────────────

func TestAllKnownPairNamePrefixes_ExcludesOther(t *testing.T) {
	t.Parallel()
	prefixes := AllKnownPairNamePrefixes()
	if len(prefixes) == 0 {
		t.Fatal("AllKnownPairNamePrefixes() returned empty")
	}
	// "Event" est mappé sur Other → ne doit PAS être dans la liste.
	for _, p := range prefixes {
		if p == "Event" {
			t.Errorf("AllKnownPairNamePrefixes contains Event (mapped to Other)")
		}
	}
}

func TestAllKnownPairNamePrefixes_ContainsCorePrefixes(t *testing.T) {
	t.Parallel()
	prefixes := AllKnownPairNamePrefixes()
	set := make(map[string]bool, len(prefixes))
	for _, p := range prefixes {
		set[p] = true
	}
	required := []string{
		"Arena", "Tactical", "Assault", "Community",
		"Fiesta", ModeCategorySuperFiesta, ModeCategoryHuskyRaid,
		"BTB", ModePrefixBTBHeavies,
		"Ranked", "Firefight", "Gruntpocalypse",
	}
	for _, p := range required {
		if !set[p] {
			t.Errorf("AllKnownPairNamePrefixes missing required prefix %q", p)
		}
	}
}

func TestPairNamePrefixesForCategory_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := PairNamePrefixesForCategory(""); got != nil {
		t.Errorf("PairNamePrefixesForCategory(empty) = %v, want nil", got)
	}
}

func TestPairNamePrefixesForCategory_OtherReturnsNil(t *testing.T) {
	t.Parallel()
	if got := PairNamePrefixesForCategory(ModeCategoryOther); got != nil {
		t.Errorf("PairNamePrefixesForCategory(Other) = %v, want nil (caller doit utiliser AllKnownPairNamePrefixes)", got)
	}
}

func TestPairNamePrefixesForCategory_UnknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := PairNamePrefixesForCategory("NotARealCategory")
	if len(got) != 0 {
		t.Errorf("PairNamePrefixesForCategory(unknown) = %v, want empty", got)
	}
}

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
