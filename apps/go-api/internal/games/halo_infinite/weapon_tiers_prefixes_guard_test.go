package halo_infinite

// weapon_tiers_prefixes_guard_test.go — LA LISTE DES MODES A DEPARTS ALEATOIRES NE DERIVE PAS.
//
// # POURQUOI CE GARDE-RAIL
//
// La liste vit DANS LE TOML du titre (`regulation.toml`, `[weapon_tiers]
// random_start_mode_prefixes`) : c est la couche de synchronisation qui la lit, et elle ne doit
// pas connaitre la taxonomie de modes d un titre en particulier. Mais la MEME connaissance
// existe deja en Go — `PairNamePrefixesForCategory` — et une liste recopiee qui derive est
// exactement ce que la regle du depot interdit (CLAUDE.md n°6 : centraliser ET poser un
// garde-rail).
//
// CE QU IL EXIGE : les prefixes declares au TOML sont EXACTEMENT ceux que la taxonomie du titre
// associe aux categories a departs aleatoires. Ni plus (un mode regulier prive de son niveau
// « base ») ni moins (un mode Fiesta qui publierait une arme de base inventee).
//
// SI LA TAXONOMIE CHANGE (une categorie de plus, un prefixe renomme), ce test rougit et dit
// quoi ecrire dans le TOML.

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// categoriesADepartsAleatoires : les categories de mode dont l equipement de debut de vie est
// TIRE AU SORT. Mesure du 2026-09-14 (76 artefacts) : en mode regulier les trois armes de
// depart pesent 94,4 % (Assassin) et 92,6 % (BTB) des equipements de premiere emission ; en
// Super Fiesta la distribution est PLATE sur 22 cles, entre 3,2 % et 7,3 %.
var categoriesADepartsAleatoires = []string{
	ModeCategoryFiesta, ModeCategorySuperFiesta, ModeCategoryHuskyRaid,
}

func TestWeaponTiers_PrefixesDuTomlCollentALaTaxonomie(t *testing.T) {
	reg, err := mappings.LoadRegulationFromFile(filepath.Join(
		"..", "..", "..", "..", "..", "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation.toml illisible : %v", err)
	}
	got := reg.RandomStartModePrefixes()

	vus := map[string]bool{}
	var want []string
	for _, cat := range categoriesADepartsAleatoires {
		for _, p := range PairNamePrefixesForCategory(cat) {
			if !vus[p] {
				vus[p] = true
				want = append(want, p)
			}
		}
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("[weapon_tiers].random_start_mode_prefixes = %v, attendu %v "+
			"(la taxonomie du titre fait foi — corriger le TOML)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("prefixe #%d = %q, attendu %q (liste attendue : %v)", i, got[i], want[i], want)
		}
	}
}
