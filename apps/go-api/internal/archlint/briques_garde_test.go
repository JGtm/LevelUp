// briques_garde_test.go — auto-preuves de la table des briques (parcours récursif) et test
// d'exhaustivité des dossiers de `libs/`.
//
// # EXHAUSTIVITÉ (2026-10-29, plan D2 lot 1, découverte 38 de D1)
//
// `TestBriquesImports` ne garde que ce que la table `briques` déclare : un dossier Go ajouté sous
// `libs/` et absent de la table n'est contrôlé par personne. `TestChaqueDossierGoDeLibsEstDansLaTable`
// exige que tout dossier qui porte au moins un `.go` sous `libs/` figure dans la table (paquet exact,
// ou arbre d'une brique récursive).
//
// Mutation qui doit le faire rougir : créer `libs/himap/nouveau/x.go` (`package nouveau`).
package archlint

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestBriquesImports_ParcoursRecursifEtFrontiere : un import vers l'arbre d'une brique récursive
// est interne, mais un dossier voisin qui partage seulement le début du nom (`filmprofile` pour
// `film`) ne l'est pas, ni le même nom sous un autre module.
func TestBriquesImports_ParcoursRecursifEtFrontiere(t *testing.T) {
	b := brique{
		nom:      "x",
		paquets:  []string{"libs/film"},
		recursif: true,
		permis:   []string{"libs/y/q"},
	}
	for imp, attendu := range map[string]bool{
		prefixeBriques + "film":                                       true,
		prefixeBriques + "film/internal/source":                       true,
		prefixeBriques + "filmprofile":                                false,
		prefixeModuleAPI + "internal/games/halo_infinite/film":        false,
		prefixeModuleAPI + "internal/games/halo_infinite/filmprofile": false,
		"github.com/JGtm/LevelUp/libs/y/q":                            true,
		"github.com/JGtm/LevelUp/libs/y/q/sous":                       false,
	} {
		if got := b.interne(imp); got != attendu {
			t.Errorf("interne(%q) = %v, attendu %v", imp, got, attendu)
		}
	}
	exact := brique{nom: "e", paquets: []string{"libs/x/p"}}
	if exact.interne("github.com/JGtm/LevelUp/libs/x/p/sous") {
		t.Error("une brique non récursive ne couvre pas ses sous-répertoires")
	}
}

// dossiersHorsTable rend, triés, les dossiers (relatifs à la racine du dépôt) qui ne sont couverts
// par aucune brique : ni paquet exact, ni sous-arbre d'une brique récursive. Fonction pure.
func dossiersHorsTable(dossiers []string, table []brique) []string {
	var hors []string
	for _, d := range dossiers {
		couvert := false
		for _, b := range table {
			for _, p := range b.paquets {
				if d == p || (b.recursif && strings.HasPrefix(d, p+"/")) {
					couvert = true
				}
			}
		}
		if !couvert {
			hors = append(hors, d)
		}
	}
	sort.Strings(hors)
	return hors
}

// dossiersGoDeLibs liste les dossiers de `libs/` qui portent au moins un `.go` (tests compris),
// hors ceux que l'outil Go n'ouvre jamais (`testdata`, `vendor`, `.x`, `_x`).
func dossiersGoDeLibs(t *testing.T) []string {
	t.Helper()
	depot := workspaceRoot(t)
	base := filepath.Join(depot, "libs")
	vus := map[string]bool{}
	err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		nom := d.Name()
		if d.IsDir() {
			if chemin != base && (dossiersInvisiblesAuGo[nom] || strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(nom, ".go") {
			rel, _ := filepath.Rel(depot, filepath.Dir(chemin))
			vus[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de libs/ : %v", err)
	}
	out := make([]string, 0, len(vus))
	for d := range vus {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func TestChaqueDossierGoDeLibsEstDansLaTable(t *testing.T) {
	dossiers := dossiersGoDeLibs(t)
	if len(dossiers) < 5 {
		t.Fatalf("%d dossiers Go sous libs/, au moins 5 attendus (gamefiles x4, himap x1) : le parcours ne voit plus l'arborescence", len(dossiers))
	}
	if hors := dossiersHorsTable(dossiers, briques); len(hors) > 0 {
		t.Errorf("dossiers Go sous libs/ absents de la table `briques` (briques_imports_test.go) — leurs imports "+
			"ne sont contrôlés par personne : les déclarer dans la brique qui les porte :\n  %s", strings.Join(hors, "\n  "))
	}
}

// TestChaqueDossierGoDeLibsEstDansLaTable_AutoPreuve : un dossier fictif hors table rougit, un
// paquet exact et un sous-arbre d'une brique récursive passent.
func TestChaqueDossierGoDeLibsEstDansLaTable_AutoPreuve(t *testing.T) {
	table := []brique{
		{nom: "exacte", paquets: []string{"libs/a/p"}},
		{nom: "arbre", paquets: []string{"libs/b"}, recursif: true},
	}
	got := dossiersHorsTable([]string{"libs/a/p", "libs/a/p/sous", "libs/a/q", "libs/b", "libs/b/x/y", "libs/bb", "libs/x/y"}, table)
	attendu := []string{"libs/a/p/sous", "libs/a/q", "libs/bb", "libs/x/y"}
	if strings.Join(got, ",") != strings.Join(attendu, ",") {
		t.Fatalf("hors table = %v, attendu %v", got, attendu)
	}
}
