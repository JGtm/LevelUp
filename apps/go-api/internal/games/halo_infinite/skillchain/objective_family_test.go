package skillchain

import "testing"

// TestIsObjectiveSubMode verrouille la SOURCE UNIQUE de la famille objectif,
// partagée par la chaîne LUSR sociale (arena_objectif) et la chaîne de performance
// classée (ranked_objectif). Un ajout/retrait dans la liste déplace des matchs dans
// les DEUX classifications : ce corpus est le contrat.
func TestIsObjectiveSubMode(t *testing.T) {
	cases := []struct {
		pairName string
		want     bool
		why      string
	}{
		// Les 12 entrées de la liste, sous leur forme normalisée directe.
		{"ctf", true, "liste"},
		{"capture the flag", true, "liste"},
		{"neutral flag ctf", true, "liste"},
		{"one flag ctf", true, "liste"},
		{"covert one flag", true, "liste"},
		{"strongholds", true, "liste"},
		{"oddball", true, "liste"},
		{"king of the hill", true, "liste"},
		{"total control", true, "liste"},
		{"land grab", true, "liste"},
		{"extraction", true, "liste"},
		{"stockpile", true, "liste"},

		// Formes pair_name réelles : préfixe de playlist + suffixe de carte,
		// normalisés par NormalizeModeLabel avant comparaison.
		{"Arena:CTF on Recharge", true, "pair_name social complet"},
		{"Ranked:Oddball on Streets", true, "pair_name classé complet"},
		{"Ranked:Strongholds on Live Fire", true, "pair_name classé complet"},
		{"Ranked:King of the Hill on Solitude", true, "sous-mode à espaces"},
		{"Tactical:Total Control on Argyle", true, "pair_name social complet"},

		// Casse : la comparaison se fait en ASCII minuscule.
		{"ARENA:ODDBALL", true, "casse haute"},
		{"Arena:OddBall on Bazaar", true, "casse mixte"},

		// Famille slayer / combat.
		{"Arena:Slayer on Bazaar", false, "slayer"},
		{"Ranked:Slayer on Aquarius", false, "slayer classé"},
		{"Arena:Attrition on Recharge", false, "hors liste"},
		{"Arena:Elimination on Aquarius", false, "hors liste"},
		{"Rumble Pit:Slayer on Bazaar", false, "hors liste"},

		// Entrée vide / inconnue → famille slayer (fallback assumé des 2 consommateurs).
		{"", false, "pair_name vide"},
		{"Truc Inconnu XYZ", false, "sous-mode inconnu"},

		// LACUNES CONNUES D-I (.ai/PLAN_PERF_NOTE_OBJECTIFS.md, 2026-08-27) : ces
		// matchs SONT des matchs d'objectif mais tombent en famille slayer. Corriger
		// la liste déplacerait aussi des chaînes LUSR persistées (recompute LUSR hors
		// périmètre) — le comportement actuel est verrouillé ICI pour que la
		// correction future soit un changement DÉLIBÉRÉ, pas un effet de bord.
		{"Ranked:CTF 3 Captures on Argyle", false, "D-I : variante CTF non listée"},
		{"Arena:VIP on Streets", false, "D-I : VIP non listé"},
		{"Arena:One Bomb on Behemoth", false, "D-I : Assaut non listé"},
		{"Arena:Neutral Bomb on Fragmentation", false, "D-I : Assaut non listé"},
		{"Strongholds:Arena on Behemoth", false, "D-I : pair_name inversé"},
	}

	for _, tc := range cases {
		if got := IsObjectiveSubMode(tc.pairName); got != tc.want {
			t.Errorf("IsObjectiveSubMode(%q) = %v, want %v (%s)", tc.pairName, got, tc.want, tc.why)
		}
	}
}

// TestLusrChainForAssassinUsesSharedList prouve que la chaîne LUSR sociale est
// bien BRANCHÉE sur le helper (et non sur une copie locale de la liste) : pour tout
// pair_name de catégorie Assassin, arena_objectif ⇔ IsObjectiveSubMode.
func TestLusrChainForAssassinUsesSharedList(t *testing.T) {
	corpus := []string{
		"Arena:CTF on Recharge", "Arena:Strongholds on Streets", "Arena:Oddball on Bazaar",
		"Arena:King of the Hill on Live Fire", "Arena:Total Control on Argyle",
		"Arena:Land Grab on Highpower", "Arena:Extraction on Aquarius",
		"Arena:Stockpile on Fragmentation", "Arena:Slayer on Bazaar",
		"Tactical:Slayer on Recharge", "Community:Attrition on Streets",
		"Arena:CTF 3 Captures on Argyle", "",
	}
	for _, p := range corpus {
		wantObjectif := IsObjectiveSubMode(p)
		got := lusrChainForAssassin(p)
		if wantObjectif && got != chainArenaObjectif {
			t.Errorf("lusrChainForAssassin(%q) = %q, want %q (helper dit famille objectif)",
				p, got, chainArenaObjectif)
		}
		if !wantObjectif && got != chainArenaSlayer {
			t.Errorf("lusrChainForAssassin(%q) = %q, want %q (helper dit famille slayer)",
				p, got, chainArenaSlayer)
		}
	}
}
