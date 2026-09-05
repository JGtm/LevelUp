//go:build gamefiles

// cmd/mapstruct-build — equivalence_gamefiles_test.go : le test qui manquait.
//
// POURQUOI. `internal/himodule` n'a AUCUN test et le découpage de `loadHd1` a été vérifié
// par lecture, pas par exécution. Le gate des artefacts ne le couvre pas : le rejeu lit les
// fichiers de structure FIGÉS, il ne repasse jamais par le lecteur de module. Une dérive du
// lecteur — un offset, une base de données mal calculée, un bloc mal chaîné — ne casserait
// donc rien de visible, et se découvrirait à la prochaine carte produite, des mois plus tard.
//
// Ce test rejoue la production des deux cartes déjà figées dans le dépôt et exige
// l'ÉGALITÉ EXACTE, emprise par emprise. C'est la condition d'entrée du portage de la
// chaîne des triangles (plan `.ai/V7.5/cartes/PLAN_PORT_TRIANGLES_GO.md`, étape E) : on ne
// porte rien au-dessus d'un lecteur dont personne ne vérifie la sortie.
//
// Il se déclare ABSENT quand l'installation du jeu n'est pas là (CI). C'est assumé : sur un
// poste de développement il tourne, et c'est là que la régression se produit.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/himap"
)

// referenceDir : les structures figées, telles que le dépôt les publie.
func referenceDir() string {
	return filepath.Join("..", "..", "..", "..",
		"data", "titles", "halo_infinite", "reference", "map_structure")
}

// TestStructuresFigeesSeReproduisent — régénère chaque carte figée et compare.
//
// Mutation qui doit le faire rougir : décaler d'un octet un offset de lecture d'instance
// dans `internal/himap`, ou changer la base de données de `internal/himodule`.
func TestStructuresFigeesSeReproduisent(t *testing.T) {
	levels, err := himap.LevelsDir(deployVariant)
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	entrees, err := os.ReadDir(referenceDir())
	if err != nil {
		t.Fatalf("dossier des structures figées illisible: %v", err)
	}
	vues := 0
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || filepath.Ext(nom) != ".json" {
			continue
		}
		module := nom[:len(nom)-len(".json")]
		t.Run(module, func(t *testing.T) {
			fige := lireStructure(t, filepath.Join(referenceDir(), nom))
			modPath, err := findModule(levels, module)
			if err != nil {
				t.Skipf("module absent (%s) : %v", module, err)
			}
			produit, err := extractStructure(modPath, module, fige.MapNames)
			if err != nil {
				t.Fatalf("extraction: %v", err)
			}
			comparer(t, fige, produit)
			vues++
		})
	}
	if vues == 0 {
		t.Skip("aucune carte figée n'a pu être régénérée")
	}
}

func lireStructure(t *testing.T, path string) *replay.MapStructure {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s: %v", path, err)
	}
	var ms replay.MapStructure
	if err := json.Unmarshal(buf, &ms); err != nil {
		t.Fatalf("structure figée illisible %s: %v", path, err)
	}
	return &ms
}

// comparer exige l'égalité de tout ce qui vient du LECTEUR. `GeneratedAt` est exclu : il
// change à chaque exécution par construction, ce n'est pas une donnée mesurée.
func comparer(t *testing.T, fige, produit *replay.MapStructure) {
	t.Helper()
	if fige.SchemaVersion != produit.SchemaVersion {
		t.Errorf("schemaVersion : figé %d, produit %d", fige.SchemaVersion, produit.SchemaVersion)
	}
	if fige.BSPIndex != produit.BSPIndex {
		t.Errorf("bspIndex : figé %d, produit %d — le BSP retenu a changé",
			fige.BSPIndex, produit.BSPIndex)
	}
	for a := 0; a < 3; a++ {
		if fige.WorldMin[a] != produit.WorldMin[a] || fige.WorldMax[a] != produit.WorldMax[a] {
			t.Errorf("bornes monde axe %d : figé [%v %v], produit [%v %v]",
				a, fige.WorldMin[a], fige.WorldMax[a], produit.WorldMin[a], produit.WorldMax[a])
		}
	}
	if len(fige.Surfaces) != len(produit.Surfaces) {
		t.Fatalf("emprises : figé %d, produit %d", len(fige.Surfaces), len(produit.Surfaces))
	}
	ecarts := 0
	for i := range fige.Surfaces {
		// DeepEqual : Surface porte `Poly`, une tranche de sommets — la comparaison
		// directe ne compile pas, et comparer les seuls X0..ZB laisserait passer une
		// dérive de l'emprise ORIENTÉE, qui est justement ce que le portage va toucher.
		if !reflect.DeepEqual(fige.Surfaces[i], produit.Surfaces[i]) {
			if ecarts < 3 {
				t.Errorf("emprise %d : figé %+v, produit %+v", i, fige.Surfaces[i], produit.Surfaces[i])
			}
			ecarts++
		}
	}
	if ecarts > 0 {
		t.Errorf("%d emprises diffèrent sur %d", ecarts, len(fige.Surfaces))
	}
	for motif, n := range fige.Excluded {
		if produit.Excluded[motif] != n {
			t.Errorf("exclusions %q : figé %d, produit %d", motif, n, produit.Excluded[motif])
		}
	}
	if !reflect.DeepEqual(fige.Stats, produit.Stats) {
		t.Errorf("statistiques : figé %+v, produit %+v", fige.Stats, produit.Stats)
	}
}
