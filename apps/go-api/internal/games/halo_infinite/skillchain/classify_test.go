package skillchain

import "testing"

// TestClassifyLUSRChain_Golden est le corpus de référence (48 cas) déplacé depuis
// internal/sync (TestGetLUSRChain) : prouve que la classification relocalisée reste
// byte-identique. Couvre les 8 sorties d'InferModeCategoryFromPairName, les deux
// cas spéciaux rocket-hog (branche BTB ET branche Other), les sous-modes objectif
// Assassin, et le fallback pair_name vide → arena_slayer.
func TestClassifyLUSRChain_Golden(t *testing.T) {
	cases := []struct {
		pairName string
		want     string
	}{
		// Exclus
		{"Ranked:Slayer on Aquarius", ""},
		{"Ranked:CTF on Recharge", ""},
		{"Firefight:King of the Hill on Argyle", ""},
		{"Gruntpocalypse:Slayer on Deadlock", ""},
		// BTB
		{"BTB:Slayer on Fragmentation", chainBTB},
		{"BTB Heavies:CTF on Highpower", chainBTB},
		// Chaos — catégorie Fiesta/SuperFiesta/HuskyRaid
		{"Fiesta:Slayer on Bazaar", chainChaos},
		{"Super Fiesta:Slayer on Catalyst", chainChaos},
		{"Husky Raid:CTF on Pharaoh", chainChaos},
		{"Super Husky Raid:CTF on Pharaoh", chainChaos},
		{"Castle Wars", chainChaos},
		// Chaos — Other avec keywords
		{"Infection:Slayer on Bazaar", chainChaos},
		{"Griffball", chainChaos},
		{"Rocket Hog Race:BTB on Highpower", chainChaos},
		{"Action Sack:Slayer on Recharge", chainChaos},
		{"Event:Last Spartan Standing on Fragmentation", chainChaos},
		// arena_slayer — Other fallback
		{"Rumble Pit:Slayer on Bazaar", chainArenaSlayer},
		{"Custom:Unknown on MapX", chainArenaSlayer},
		// arena_slayer — Assassin
		{"Arena:Slayer on Bazaar", chainArenaSlayer},
		{"Arena:Team Slayer on Bazaar", chainArenaSlayer},
		{"Arena:Attrition on Live Fire", chainArenaSlayer},
		{"Arena:Elimination on Bazaar", chainArenaSlayer},
		{"Tactical:Slayer on Recharge", chainArenaSlayer},
		{"Community:Team Slayer on Solution", chainArenaSlayer},
		// arena_objectif — Assassin
		{"Arena:CTF on Recharge", chainArenaObjectif},
		{"Arena:Neutral Flag CTF on Live Fire", chainArenaObjectif},
		{"Arena:One Flag CTF on Highpower", chainArenaObjectif},
		{"Arena:Strongholds on Streets", chainArenaObjectif},
		{"Arena:Oddball on Aquarius", chainArenaObjectif},
		{"Arena:King of the Hill on Catalyst", chainArenaObjectif},
		{"Arena:Total Control on Fragmentation", chainArenaObjectif},
		{"Arena:Land Grab on Recharge", chainArenaObjectif},
		{"Arena:Extraction on Live Fire", chainArenaObjectif},
		{"Arena:Stockpile on Deadlock", chainArenaObjectif},
		{"BTB:CTF on Highpower", chainBTB}, // CTF dans BTB → btb (pas arena_objectif)
		// pair_name vide → arena_slayer (fallback safe)
		{"", chainArenaSlayer},
	}
	for _, tc := range cases {
		t.Run(tc.pairName, func(t *testing.T) {
			got := ClassifyLUSRChain(tc.pairName)
			if got != tc.want {
				t.Errorf("ClassifyLUSRChain(%q) = %q, want %q", tc.pairName, got, tc.want)
			}
		})
	}
}

// TestContainsI — déplacé depuis internal/sync (MT-15).
func TestContainsI(t *testing.T) {
	cases := []struct {
		s, sub string
		want   bool
	}{
		{"Ranked Arena", "ranked", true},
		{"RANKED Arena", "ranked", true},
		{"Quick Play", "ranked", false},
		{"anything", "", true},
		{"", "x", false},
	}
	for _, c := range cases {
		got := containsI(c.s, c.sub)
		if got != c.want {
			t.Errorf("containsI(%q, %q) = %v, want %v", c.s, c.sub, got, c.want)
		}
	}
}
