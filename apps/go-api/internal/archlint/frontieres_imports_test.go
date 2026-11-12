// Package archlint — frontieres_imports_test.go : garde-rails d'imports des frontières de couche,
// et l'outil commun qui les exprime.
//
// Les imports se lisent dans l'AST (un chemin cité dans un commentaire ou dans une chaîne n'est pas
// un import) et se comparent au chemin EXACT du paquet : `.../internal/sync` n'interdit pas
// `.../internal/sync/replayartifacts`.
package archlint

import (
	"sort"
	"strconv"
	"strings"
	"testing"
)

// cheminModuleAPI : préfixe des chemins d'import des paquets de `apps/go-api`.
const cheminModuleAPI = "levelup/go-api/internal/"

// importsInterdits rend, triés, les chemins parmi `interdits` que le fichier importe (chemins
// exacts, sous n'importe quel alias).
func importsInterdits(src goSource, interdits []string) []string {
	voulu := make(map[string]bool, len(interdits))
	for _, p := range interdits {
		voulu[p] = true
	}
	var trouves []string
	for _, imp := range src.File.Imports {
		chemin, err := strconv.Unquote(imp.Path.Value)
		if err == nil && voulu[chemin] {
			trouves = append(trouves, chemin)
		}
	}
	sort.Strings(trouves)
	return trouves
}

// TestOutilImportsInterdits : auto-preuve de l'outil, sur une source de test.
func TestOutilImportsInterdits(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	"fmt"
	gosync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
)

// "levelup/go-api/internal/service" cité en commentaire
var _ = fmt.Sprint("levelup/go-api/internal/service")
`)
	interdits := []string{cheminModuleAPI + "sync", cheminModuleAPI + "service"}
	got := importsInterdits(src, interdits)
	if len(got) != 1 || got[0] != cheminModuleAPI+"sync" {
		t.Fatalf("attendu le seul import exact de internal/sync, obtenu %v", got)
	}
}

// TestWorldenrichSansService : le câblage du classement mondial ne dépend pas de `service`, de sorte
// que `service` puisse importer les paquets qui s'appuient sur `worldenrich` (le suivi en direct de
// Halo 5) sans cycle. L'assemblage qui a besoin des deux vit à l'appelant (`cmd/server`).
func TestWorldenrichSansService(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, "internal/worldenrich") {
		for _, p := range importsInterdits(src, []string{cheminModuleAPI + "service"}) {
			violations = append(violations, src.Rel+" importe "+p)
		}
	}
	if len(violations) > 0 {
		t.Errorf("internal/worldenrich ne doit pas importer internal/service (cycle avec le suivi en direct) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
