// Package sync — tests de GetPerformanceChain.
//
// Verrouille la table de correspondance (pair_name, is_ranked, is_firefight) →
// chaîne du score de performance. Couvre les 6 chaînes possibles + le fallback.
package sync

import "testing"

func TestGetPerformanceChain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		pairName    string
		isRanked    bool
		isFirefight bool
		want        string
	}{
		// Priorité 1 — Ranked court-circuite tout (peu importe pair_name).
		{name: "ranked flag prend la priorité sur pair_name BTB", pairName: "BTB:Slayer", isRanked: true, want: PerfChainRanked},
		{name: "ranked flag avec pair_name vide", pairName: "", isRanked: true, want: PerfChainRanked},
		{name: "ranked Arena Slayer", pairName: "Ranked:Slayer", isRanked: true, want: PerfChainRanked},
		{name: "ranked + firefight (incohérent) → ranked gagne", pairName: "", isRanked: true, isFirefight: true, want: PerfChainRanked},

		// Priorité 2 — Firefight.
		{name: "firefight flag", pairName: "Firefight:King of the Hill", isFirefight: true, want: PerfChainFirefight},
		{name: "firefight gruntpocalypse", pairName: "Gruntpocalypse", isFirefight: true, want: PerfChainFirefight},

		// Priorité 3 — Délégation à GetLUSRChain (PvP non classé).
		{name: "BTB social → btb", pairName: "BTB:Slayer", want: LUSRChainBTB},
		{name: "BTB CTF → btb (objectif resté dans BTB)", pairName: "BTB:CTF", want: LUSRChainBTB},
		{name: "BTB Rocket Hog → chaos", pairName: "BTB:Rocket Hog Race", want: LUSRChainChaos},
		{name: "Fiesta Slayer → chaos", pairName: "Fiesta:Slayer", want: LUSRChainChaos},
		{name: "Super Fiesta → chaos", pairName: "Super Fiesta:Slayer", want: LUSRChainChaos},
		{name: "Husky Raid → chaos", pairName: "Husky Raid", want: LUSRChainChaos},
		{name: "Infection → chaos (Other → contient 'infection')", pairName: "Infection:Last Spartan Standing", want: LUSRChainChaos},
		{name: "Griffball → chaos", pairName: "Griffball", want: LUSRChainChaos},
		{name: "Action Sack → chaos", pairName: "Action Sack:Trick Shot", want: LUSRChainChaos},
		{name: "Event → chaos", pairName: "Event:Slayer", want: LUSRChainChaos},
		{name: "Arena CTF → arena_objectif", pairName: "Arena:CTF", want: LUSRChainArenaObjectif},
		{name: "Arena Strongholds → arena_objectif", pairName: "Arena:Strongholds", want: LUSRChainArenaObjectif},
		{name: "Arena Oddball → arena_objectif", pairName: "Arena:Oddball", want: LUSRChainArenaObjectif},
		{name: "Arena KotH → arena_objectif", pairName: "Arena:King of the Hill", want: LUSRChainArenaObjectif},
		{name: "Arena Slayer → arena_slayer", pairName: "Arena:Slayer", want: LUSRChainArenaSlayer},
		{name: "Tactical Slayer → arena_slayer", pairName: "Tactical:Slayer", want: LUSRChainArenaSlayer},
		{name: "Community Slayer → arena_slayer", pairName: "Community:Slayer", want: LUSRChainArenaSlayer},
		{name: "Rumble Pit → arena_slayer (fallback Other)", pairName: "Rumble Pit", want: LUSRChainArenaSlayer},

		// Priorité 4 — fallback ultime sur pair_name vide / inconnu.
		{name: "pair_name vide (aucun flag) → arena_slayer", pairName: "", want: LUSRChainArenaSlayer},
		{name: "pair_name inconnu → fallback arena_slayer", pairName: "Truc Inconnu XYZ", want: LUSRChainArenaSlayer},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetPerformanceChain("", tc.pairName, tc.isRanked, tc.isFirefight)
			if got != tc.want {
				t.Errorf("GetPerformanceChain(%q, ranked=%v, ff=%v) = %q, want %q",
					tc.pairName, tc.isRanked, tc.isFirefight, got, tc.want)
			}
		})
	}
}

// TestGetPerformanceChain_NeverEmpty vérifie l'invariant : aucun match n'est
// orphelin. Tout combo de (pairName, isRanked, isFirefight) doit produire une
// chaîne non vide — c'est la garantie principale par rapport à GetLUSRChain.
func TestGetPerformanceChain_NeverEmpty(t *testing.T) {
	t.Parallel()
	pairs := []string{
		"",
		"Big Team Battle - Slayer",
		"Fiesta - Slayer",
		"Arena - Slayer",
		"Arena - CTF",
		"Rumble Pit",
		"pair_name complètement inventé",
		"Firefight - King of the Hill",
	}
	flags := []struct {
		ranked, firefight bool
	}{
		{false, false},
		{true, false},
		{false, true},
		{true, true},
	}
	for _, p := range pairs {
		for _, f := range flags {
			got := GetPerformanceChain("", p, f.ranked, f.firefight)
			if got == "" {
				t.Errorf("GetPerformanceChain(%q, ranked=%v, ff=%v) returned empty string (invariant violated)",
					p, f.ranked, f.firefight)
			}
		}
	}
}
