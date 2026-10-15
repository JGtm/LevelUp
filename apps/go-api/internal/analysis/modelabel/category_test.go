package modelabel

import "testing"

// ─── stripForCategory ─────────────────────────────────────────────────────

func TestStripMapSuffix_RemovesOnMap(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Arena:Slayer on Bazaar":          "Arena:Slayer",
		"BTB:CTF on Highpower":            "BTB:CTF",
		"Slayer on Aquarius":              "Slayer",
		"Super Fiesta:Slayer on Behemoth": "Super Fiesta:Slayer",
	}
	for in, want := range cases {
		if got := stripForCategory(in); got != want {
			t.Errorf("stripForCategory(%q) = %q, want %q", in, got, want)
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
		if got := stripForCategory(in); got != in {
			t.Errorf("stripForCategory(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestStripMapSuffix_RemovesTechnicalIDSuffix(t *testing.T) {
	t.Parallel()
	// Le suffixe " - <8+ alphanum>" doit être stripé.
	got := stripForCategory("Slayer - ABCDEFGH123")
	if got != "Slayer" {
		t.Errorf("stripForCategory(Slayer - ABCDEFGH123) = %q, want Slayer", got)
	}
}

func TestStripMapSuffix_PreservesShortHyphen(t *testing.T) {
	t.Parallel()
	// Si le suffixe est <8 caractères → pas stripé.
	got := stripForCategory("Slayer - X")
	if got != "Slayer - X" {
		t.Errorf("stripForCategory(Slayer - X) = %q, want unchanged (suffixe < 8 chars)", got)
	}
}

// ─── normalizePrefixCase ──────────────────────────────────────────────────

func TestNormalizeModeCase_KnownAcronyms(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"btb":              "BTB",
		"BTB":              "BTB",
		"btb heavies":      PrefixBTBHeavies,
		"super fiesta":     CategorySuperFiesta,
		"super husky raid": PrefixSuperHuskyRaid,
		"husky raid":       CategoryHuskyRaid,
		"castle wars":      PrefixCastleWars,
	}
	for in, want := range cases {
		if got := normalizePrefixCase(in); got != want {
			t.Errorf("normalizePrefixCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeModeCase_AllUpperPreserved(t *testing.T) {
	t.Parallel()
	// Tout-majuscules → préservé tel quel (acronymes inconnus).
	cases := []string{"ARENA", "CTF", "BTB"}
	for _, in := range cases {
		if got := normalizePrefixCase(in); got != in {
			t.Errorf("normalizePrefixCase(%q) = %q, want %q (all-upper preserved)", in, got, in)
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
		if got := normalizePrefixCase(in); got != want {
			t.Errorf("normalizePrefixCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeModeCase_EmptyAndWhitespace(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "  ", "\t"} {
		if got := normalizePrefixCase(in); got != "" {
			t.Errorf("normalizePrefixCase(%q) = %q, want empty", in, got)
		}
	}
}

// ─── KnownPrefixes ───────────────────────────────────────────

func TestKnownPrefixes_ExcludesOther(t *testing.T) {
	t.Parallel()
	prefixes := KnownPrefixes()
	if len(prefixes) == 0 {
		t.Fatal("KnownPrefixes() returned empty")
	}
	// "Event" est mappé sur Other → ne doit PAS être dans la liste.
	for _, p := range prefixes {
		if p == "Event" {
			t.Errorf("KnownPrefixes contains Event (mapped to Other)")
		}
	}
}

func TestKnownPrefixes_ContainsCorePrefixes(t *testing.T) {
	t.Parallel()
	prefixes := KnownPrefixes()
	set := make(map[string]bool, len(prefixes))
	for _, p := range prefixes {
		set[p] = true
	}
	required := []string{
		"Arena", "Tactical", "Assault", "Community",
		"Fiesta", CategorySuperFiesta, CategoryHuskyRaid,
		"BTB", PrefixBTBHeavies,
		"Ranked", "Firefight", "Gruntpocalypse",
	}
	for _, p := range required {
		if !set[p] {
			t.Errorf("KnownPrefixes missing required prefix %q", p)
		}
	}
}

func TestPrefixesForCategory_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	if got := PrefixesForCategory(""); got != nil {
		t.Errorf("PrefixesForCategory(empty) = %v, want nil", got)
	}
}

func TestPrefixesForCategory_OtherReturnsNil(t *testing.T) {
	t.Parallel()
	if got := PrefixesForCategory(CategoryOther); got != nil {
		t.Errorf("PrefixesForCategory(Other) = %v, want nil (caller doit utiliser KnownPrefixes)", got)
	}
}

func TestPrefixesForCategory_UnknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := PrefixesForCategory("NotARealCategory")
	if len(got) != 0 {
		t.Errorf("PrefixesForCategory(unknown) = %v, want empty", got)
	}
}

func TestInferCategory(t *testing.T) {
	cases := []struct {
		pairName string
		want     string
	}{
		// Format standard
		{"Arena:Slayer", CategoryAssassin},
		{"Arena:Slayer on Bazaar", CategoryAssassin},
		{"Arena:CTF on Recharge", CategoryAssassin},
		{"Tactical:Slayer", CategoryAssassin},
		{"Community:Team Slayer on Solution", CategoryAssassin},
		{"Super Fiesta:Slayer on Catalyst - Forge", CategorySuperFiesta},
		{"Fiesta:Slayer", CategoryFiesta},
		{"Husky Raid:Slayer", CategoryHuskyRaid},
		{"BTB:Slayer", CategoryBTB},
		{"BTB Heavies:CTF", CategoryBTB},
		{"Ranked:Slayer on Aquarius", CategoryRanked},
		{"Firefight:KOTH", CategoryFirefight},
		{"Gruntpocalypse:Slayer", CategoryFirefight},
		// Paires mesurées fausses dans match_registry le 2026-10-09 (colonne écrite par un autre
		// classifieur ou avant que le nom soit connu).
		{"Ranked:Strongholds on Live Fire", CategoryRanked},
		{"Community:Team Slayer on Dynasty", CategoryAssassin},
		{"Arena:Oddball on Live Fire", CategoryAssassin},
		{"BTB:Slayer on Deadlock", CategoryBTB},
		{"Gruntpocalypse:Fiesta on Fathom Firefight", CategoryFirefight},
		{"BTB:Fiesta Slayer on Highpower", CategoryBTB},
		{"Super Husky Raid:CTF on Chasm", CategoryHuskyRaid},
		// Identifiant d asset non résolu : Other, recalculé quand le nom se résout.
		{"2d1a4b3c-5e6f-4a7b-8c9d-0e1f2a3b4c5d", CategoryOther},
		// Sans séparateur (mode parent qui est lui-même une catégorie)
		{"Husky Raid", CategoryHuskyRaid},
		{"BTB", CategoryBTB},
		{"Castle Wars", CategoryFiesta},
		// Format inversé (préfixe à droite)
		{"CTF:Arena", CategoryAssassin},
		{"Slayer:Ranked", CategoryRanked},
		// Préfixe inconnu → Other
		{"Custom:Slayer", CategoryOther},
		{"Slayer", CategoryOther},
		// Casse normalisation
		{"super fiesta:slayer", CategorySuperFiesta},
		{"BTB:slayer", CategoryBTB},
		// Empty
		{"", CategoryOther},
		{"   ", CategoryOther},
	}
	for _, tc := range cases {
		t.Run(tc.pairName, func(t *testing.T) {
			got := InferCategory(tc.pairName)
			if got != tc.want {
				t.Errorf("InferCategory(%q) = %q, want %q", tc.pairName, got, tc.want)
			}
		})
	}
}

func TestPrefixesForCategory(t *testing.T) {
	cases := []struct {
		category string
		want     map[string]bool
	}{
		{CategoryFiesta, map[string]bool{"Fiesta": true, "Castle Wars": true}},
		{CategorySuperFiesta, map[string]bool{"Super Fiesta": true}},
		{CategoryHuskyRaid, map[string]bool{"Husky Raid": true, "Super Husky Raid": true}},
		{CategoryBTB, map[string]bool{"BTB": true, "BTB Heavies": true}},
		{CategoryRanked, map[string]bool{"Ranked": true}},
		{CategoryAssassin, map[string]bool{
			"Arena": true, "Tactical": true, "Assault": true, "Community": true,
		}},
		{CategoryFirefight, map[string]bool{"Firefight": true, "Gruntpocalypse": true}},
		{CategoryOther, map[string]bool{}}, // Other = NIL côté Go (l'appelant utilise KnownPrefixes pour NOT IN)
	}
	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			got := PrefixesForCategory(tc.category)
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
