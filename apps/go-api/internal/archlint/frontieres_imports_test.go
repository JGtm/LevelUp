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

// importsSousArbre rend, triés, les imports du fichier qui sont l'un des chemins donnés ou un paquet
// situé dessous (`racine` et `racine/...`), sous n'importe quel alias.
func importsSousArbre(src goSource, racines []string) []string {
	var trouves []string
	for _, imp := range src.File.Imports {
		chemin, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		for _, r := range racines {
			if chemin == r || strings.HasPrefix(chemin, r+"/") {
				trouves = append(trouves, chemin)
				break
			}
		}
	}
	sort.Strings(trouves)
	return trouves
}

// racinesInterditesAuxHandlers : le moteur de synchronisation et le suivi en direct de Halo 5. Un handler
// lance une sync par un port (`port.SyncJobService`) ; il n'importe ni le moteur, ni ses sous-paquets,
// ni le suivi en direct.
var racinesInterditesAuxHandlers = []string{
	cheminModuleAPI + "sync",
	cheminModuleAPI + "games/halo_5/livesync",
}

// handlersAvecMoteurDeSyncAutorises : handlers qui importent encore l'une de ces racines, avec la date et
// la raison. Vide : plus aucun ne le fait ; une entrée ne s'ajoute pas, elle se supprime avec le code qui
// la justifiait.
var handlersAvecMoteurDeSyncAutorises = map[string]string{}

func TestOutilImportsSousArbre(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	gosync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
	"levelup/go-api/internal/synchro"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/games/halo_5"
)
`)
	got := importsSousArbre(src, racinesInterditesAuxHandlers)
	want := []string{
		cheminModuleAPI + "games/halo_5/livesync",
		cheminModuleAPI + "sync",
		cheminModuleAPI + "sync/replayartifacts",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("imports détectés %v, attendu %v (ni internal/synchro, ni internal/games/halo_5)", got, want)
	}
}

// TestHandlersSansMoteurDeSync : aucun handler HTTP n'importe le moteur de synchronisation ni le suivi
// en direct de Halo 5. Les jobs de sync (sync initiale ou delta, rattrapage, recalcul des sessions) se
// lancent par `port.SyncJobService`, que `service.SyncJobService` implémente : changer de moteur ne
// touche pas la couche HTTP.
func TestHandlersSansMoteurDeSync(t *testing.T) {
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/api/handlers") {
		vus[src.Rel] = true
		trouves := importsSousArbre(src, racinesInterditesAuxHandlers)
		if _, ok := handlersAvecMoteurDeSyncAutorises[src.Rel]; ok || len(trouves) == 0 {
			continue
		}
		violations = append(violations, src.Rel+" importe "+strings.Join(trouves, ", "))
	}
	for rel := range handlersAvecMoteurDeSyncAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de handlersAvecMoteurDeSyncAutorises")
		}
	}
	if len(violations) > 0 {
		t.Errorf("un handler importe le moteur de sync ou le suivi en direct — passer par port.SyncJobService :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
