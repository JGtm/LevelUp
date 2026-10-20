package skillchain

// chains_test.go — L'EXHAUSTIVITÉ DE [Chains], mesurée sur un corpus qui couvre
// chaque branche de [ClassifyLUSRChain] (G.5b, constat R7 de la revue du 2026-09-13).

import (
	"sort"
	"strings"
	"testing"
)

// corpusToutesBranches — un pair_name par branche de ClassifyLUSRChain. Chaque
// entrée nomme la branche qu'elle exerce : un corpus dont on ne sait plus ce qu'il
// couvre ne prouve pas l'exhaustivité.
var corpusToutesBranches = map[string]string{
	"Ranked:Slayer on Aquarius":         "ModeCategoryRanked (exclu)",
	"Firefight:KotH on Argyle":          "ModeCategoryFirefight (exclu)",
	"BTB:Slayer on Fragmentation":       "ModeCategoryBTB",
	"BTB Heavies:CTF on Highpower":      "ModeCategoryBTB, sous-mode objectif",
	"Rocket Hog Race:BTB on Highpower":  "ModeCategoryBTB + rocket hog (chaos)",
	"Fiesta:Slayer on Bazaar":           "ModeCategoryFiesta",
	"Super Fiesta:Slayer on Streets":    "ModeCategorySuperFiesta",
	"Super Husky Raid:CTF on Pharaoh":   "ModeCategoryHuskyRaid",
	"Infection:Slayer on Bazaar":        "ModeCategoryOther, infection (chaos)",
	"Griffball:Slayer on Streets":       "ModeCategoryOther, griffball (chaos)",
	"Action Sack:Slayer on Bazaar":      "ModeCategoryOther, action sack (chaos)",
	"Rumble Pit:Slayer on Bazaar":       "ModeCategoryOther, repli arena_slayer",
	"Rumble Pit:CTF on Recharge":        "ModeCategoryOther, sous-mode objectif",
	"Arena:Slayer on Bazaar":            "ModeCategoryAssassin, slayer",
	"Arena:CTF on Recharge":             "ModeCategoryAssassin, sous-mode objectif",
	"Arena:Strongholds on Streets":      "ModeCategoryAssassin, sous-mode objectif",
	"Tactical Slayer:Slayer on Streets": "ModeCategoryAssassin (tactical)",
	"":                                  "pair_name vide",
}

// TestChainsEstExhaustive — les deux sens de l'exhaustivité. Ce test est ce que le
// commentaire du gate d'intégration promettait sans le tenir.
func TestChainsEstExhaustive(t *testing.T) {
	declarees := map[string]bool{}
	for _, c := range Chains() {
		if c == "" {
			t.Error("Chains() contient la chaîne vide — elle signifie « exclu du LUSR », pas une chaîne")
		}
		if declarees[c] {
			t.Errorf("Chains() contient %q en double", c)
		}
		declarees[c] = true
	}

	rendues := map[string]bool{}
	for pair, branche := range corpusToutesBranches {
		got := ClassifyLUSRChain(pair)
		if got == "" {
			continue // branche d'exclusion : rien à lister
		}
		if !declarees[got] {
			t.Errorf("ClassifyLUSRChain(%q) [%s] rend %q, ABSENTE de Chains() — "+
				"tout consommateur de la liste (invariant I14 du gate d'intégration) "+
				"classerait cette chaîne LÉGITIME comme étrangère",
				pair, branche, got)
		}
		rendues[got] = true
	}

	var jamaisRendues []string
	for c := range declarees {
		if !rendues[c] {
			jamaisRendues = append(jamaisRendues, c)
		}
	}
	sort.Strings(jamaisRendues)
	if len(jamaisRendues) > 0 {
		t.Errorf("chaîne(s) déclarée(s) dans Chains() qu'AUCUN pair_name du corpus ne produit : %v — "+
			"soit la chaîne est morte, soit le corpus ne couvre plus toutes les branches",
			jamaisRendues)
	}
}

// TestChainsEstUneCopieDefensive — la liste partagée ne doit pas pouvoir être
// modifiée par un appelant.
func TestChainsEstUneCopieDefensive(t *testing.T) {
	a := Chains()
	if len(a) == 0 {
		t.Fatal("Chains() est vide")
	}
	avant := strings.Join(Chains(), ",")
	a[0] = "sabote"
	if strings.Join(Chains(), ",") != avant {
		t.Fatal("la liste partagée a été modifiée par l'appelant — copie non défensive")
	}
}
