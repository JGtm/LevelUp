package weapons

// weapon_names_completeness_test.go — GARDE-RAIL V72-06 (source unique des noms
// d'armes). Interdit toute divergence entre le registre (weapons/weapon_ids) et le
// fichier de traduction canonique config/titles/{slug}/mappings/weapon_names.toml :
//
//   - toute weapon_key du registre DOIT avoir une entrée nom {en,fr} dans le fichier
//     de son titre (sinon le Match view retomberait sur weapon_labels / vide) ;
//   - aucune entrée orpheline (clé du fichier absente du registre).
//
// Sans ce test, ajouter une arme au registre sans son nom (ou l'inverse) re-ouvrirait
// la porte aux 3 sources qui se marchaient dessus. C'est le garde-rail qui empêche la
// factorisation de re-diverger (règle repo « ≤ 2 copies + garde-rail »).

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

func TestWeaponNamesFileCoversRegistry(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller indisponible")
	}
	// apps/go-api/internal/games/weapons → 5 niveaux au-dessus = racine du dépôt.
	configTitles := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "config", "titles")

	// weapon_keys du registre, groupées par titre.
	regByTitle := map[string]map[string]bool{}
	for _, w := range weaponRegistryWeapons {
		if regByTitle[w.title] == nil {
			regByTitle[w.title] = map[string]bool{}
		}
		regByTitle[w.title][w.key] = true
	}

	for _, slug := range []string{titleHINF, titleH5} {
		path := filepath.Join(configTitles, slug, "mappings", "weapon_names.toml")
		set, err := mappings.LoadWeaponNamesFromFile(path)
		if err != nil {
			t.Fatalf("%s: chargement weapon_names.toml: %v", slug, err)
		}
		fileKeys := set.Names()
		regKeys := regByTitle[slug]
		if len(regKeys) == 0 {
			t.Fatalf("%s: aucune arme registre (attendu > 0)", slug)
		}

		// 1. Complétude : chaque weapon_key du registre a une entrée nom.
		for k := range regKeys {
			if _, ok := fileKeys[k]; !ok {
				t.Errorf("%s: weapon_key %q présent au registre mais ABSENT de weapon_names.toml (source unique incomplète)", slug, k)
			}
		}
		// 2. Zéro orphelin : chaque clé du fichier existe au registre.
		for k := range fileKeys {
			if !regKeys[k] {
				t.Errorf("%s: weapon_key %q dans weapon_names.toml SANS ligne registre (orphelin)", slug, k)
			}
		}
	}
}
