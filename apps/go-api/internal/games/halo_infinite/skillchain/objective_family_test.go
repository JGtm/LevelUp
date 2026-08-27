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
		// Les 17 entrées de la liste, sous leur forme normalisée directe.
		{"ctf", true, "liste"},
		{"capture the flag", true, "liste"},
		{"neutral flag ctf", true, "liste"},
		{"one flag ctf", true, "liste"},
		{"covert one flag", true, "liste"},
		{"ctf 3 captures", true, "liste (ajout lot 1bis)"},
		{"strongholds", true, "liste"},
		{"oddball", true, "liste"},
		{"king of the hill", true, "liste"},
		{"total control", true, "liste"},
		{"land grab", true, "liste"},
		{"extraction", true, "liste"},
		{"stockpile", true, "liste"},
		{"vip", true, "liste (ajout lot 1bis)"},
		{"neutral bomb", true, "liste (ajout lot 1bis)"},
		{"one bomb", true, "liste (ajout lot 1bis)"},
		{"neutral bomb squad", true, "liste (ajout lot 1bis)"},

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
		{"STRONGHOLDS:ARENA on Behemoth", true, "casse haute sur le préfixe"},

		// Famille slayer / combat.
		{"Arena:Slayer on Bazaar", false, "slayer"},
		{"Ranked:Slayer on Aquarius", false, "slayer classé"},
		{"Arena:Attrition on Recharge", false, "hors liste"},
		{"Arena:Elimination on Aquarius", false, "hors liste"},
		{"Rumble Pit:Slayer on Bazaar", false, "hors liste"},

		// Entrée vide / inconnue → famille slayer (fallback assumé des consommateurs).
		{"", false, "pair_name vide"},
		{"Truc Inconnu XYZ", false, "sous-mode inconnu"},
		{"Arena", false, "conteneur de playlist seul, jamais un mode"},

		// LACUNES D-I CORRIGÉES (lot 1bis, .ai/PLAN_PERF_NOTE_OBJECTIFS.md,
		// 2026-08-27) : les 26 matchs du corpus qui tombaient en famille slayer.
		{"Ranked:CTF 3 Captures on Argyle", true, "D-I corrigé : variante CTF listée"},
		{"Arena:VIP on Streets", true, "D-I corrigé : VIP listé"},
		{"Arena:VIP on Catalyst", true, "D-I corrigé : fixture corpus"},
		{"Arena:One Bomb on Behemoth", true, "D-I corrigé : Assaut listé"},
		{"Assault:One Bomb on Curfew", true, "D-I corrigé : fixture corpus"},
		{"Arena:Neutral Bomb on Fragmentation", true, "D-I corrigé : Assaut listé"},
		{"Assault:Neutral Bomb on Origin", true, "D-I corrigé : fixture corpus"},
		{"Assault:Neutral Bomb Squad on Rat's Nest", true, "D-I corrigé : fixture corpus"},

		// RÈGLE DU PRÉFIXE (lot 1bis) : pair_name inversés, mode à GAUCHE.
		{"Strongholds:Arena on Behemoth", true, "préfixe objectif, pair_name inversé"},
		{"CTF:Arena on Aquarius", true, "préfixe objectif, pair_name inversé"},
		{"Oddball:Arena", true, "préfixe objectif sans suffixe de carte"},
		{"King of the Hill:Arena on Streets", true, "préfixe objectif à espaces"},

		// La règle du préfixe ne blanchit QUE les préfixes objectif : un préfixe
		// hors liste (conteneur ou mode de combat) reste famille slayer.
		{"Slayer:Arena on Behemoth", false, "préfixe non objectif, inversé"},
		{"Team Slayer:Arena", false, "préfixe non objectif, inversé"},
		{"Arena:Arena", false, "aucune des deux moitiés n'est un mode objectif"},
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
		"Arena:CTF 3 Captures on Argyle", "Arena:VIP on Catalyst",
		"Assault:Neutral Bomb on Origin", "Assault:One Bomb on Curfew",
		"Assault:Neutral Bomb Squad on Rat's Nest", "Strongholds:Arena on Behemoth",
		"Slayer:Arena on Behemoth", "",
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

// TestLusrChainForOtherUsesSharedList — même preuve pour la catégorie Other
// (lot 1bis, B1b.2), avec la PRIORITÉ des règles chaos : un pair_name chaos reste
// chaos même quand il porte un mode objectif.
func TestLusrChainForOtherUsesSharedList(t *testing.T) {
	chaos := []string{
		"Infection:CTF on Bazaar", "Griffball:Oddball", "Rocket Hog Race:Strongholds",
		"Action Sack:CTF on Recharge", "Event:King of the Hill on Streets",
	}
	for _, p := range chaos {
		if got := lusrChainForOther(p); got != chainChaos {
			t.Errorf("lusrChainForOther(%q) = %q, want %q (règle chaos prioritaire)", p, got, chainChaos)
		}
	}

	objectif := []string{
		"Rumble Pit:Oddball on Bazaar", "Strongholds:Rumble Pit", "Sniper:CTF on Streets",
	}
	for _, p := range objectif {
		if got := lusrChainForOther(p); got != chainArenaObjectif {
			t.Errorf("lusrChainForOther(%q) = %q, want %q (mode objectif hors chaos)", p, got, chainArenaObjectif)
		}
	}

	slayer := []string{
		"Rumble Pit:Slayer on Bazaar", "Rumble Pit", "Custom:Unknown on MapX", "",
	}
	for _, p := range slayer {
		if got := lusrChainForOther(p); got != chainArenaSlayer {
			t.Errorf("lusrChainForOther(%q) = %q, want %q (fallback)", p, got, chainArenaSlayer)
		}
	}
}
