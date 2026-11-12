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

// racinesInterditesAuxHandlers : le moteur de synchronisation, ses écritures (`persist`) et le code
// propre à un titre (`games/halo_infinite`, `games/halo_5` et leurs sous-paquets, suivi en direct
// compris). Un handler lance une sync par un port (`port.SyncJobService`), lit et écrit par les ports de
// `internal/port` et passe par les adaptateurs sémantiques et les capacités pour tout ce qui dépend du
// titre ; il n'importe aucun de ces paquets.
var racinesInterditesAuxHandlers = []string{
	cheminModuleAPI + "sync",
	cheminModuleAPI + "persist",
	cheminModuleAPI + "games/halo_infinite",
	cheminModuleAPI + "games/halo_5",
}

// handlersAvecMoteurDeSyncAutorises : handlers qui importent encore l'une de ces racines, avec la date et
// la raison. Vide : plus aucun ne le fait ; une entrée ne s'ajoute pas, elle se supprime avec le code qui
// la justifiait. (Le nom garde « MoteurDeSync » : c'est le garde de ce périmètre, étendu le 2026-11-12
// à `persist` et aux paquets des titres.)
var handlersAvecMoteurDeSyncAutorises = map[string]string{}

func TestOutilImportsSousArbre(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	gosync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
	"levelup/go-api/internal/synchro"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/persistance"
)
`)
	got := importsSousArbre(src, racinesInterditesAuxHandlers)
	want := []string{
		cheminModuleAPI + "games/halo_5",
		cheminModuleAPI + "games/halo_5/livesync",
		cheminModuleAPI + "games/halo_infinite/haloadapter",
		cheminModuleAPI + "persist",
		cheminModuleAPI + "sync",
		cheminModuleAPI + "sync/replayartifacts",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("imports détectés %v, attendu %v (ni internal/synchro, ni internal/games/canonical, ni internal/persistance)", got, want)
	}
}

// TestHandlersSansMoteurDeSync : aucun handler HTTP n'importe le moteur de synchronisation, `persist`
// ni les paquets propres à un titre. Les jobs de sync (sync initiale ou delta, rattrapage, recalcul des
// sessions) se lancent par `port.SyncJobService`, que `service.SyncJobService` implémente : changer de
// moteur, ou ajouter un titre, ne touche pas la couche HTTP.
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
		t.Errorf("un handler importe le moteur de sync, persist ou un paquet de titre — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// racinesStockageInterditesAuxHandlers : le stockage DuckDB et ses sous-paquets. Un handler lit et écrit
// par un port de `internal/port` ; il ne construit ni dépôt ni connexion.
var racinesStockageInterditesAuxHandlers = []string{
	cheminModuleAPI + "platform/duckdb",
}

// handlersAvecStockageAutorises : handlers qui importent encore le stockage DuckDB, avec la date et la
// raison. Vide depuis le 2026-11-12 : le classement d'une panne de lecture (base occupée, poignée
// invalidée) passe par `platform/dbfault`. Une entrée ne s'ajoute pas : elle se supprime avec le code
// qui la justifiait.
var handlersAvecStockageAutorises = map[string]string{}

func TestOutilImportsSousArbreStockage(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/platform/duckdbfake"
	"levelup/go-api/internal/platform/settings"
)
`)
	got := importsSousArbre(src, racinesStockageInterditesAuxHandlers)
	want := []string{
		cheminModuleAPI + "platform/duckdb",
		cheminModuleAPI + "platform/duckdb/sharedprovider",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("imports détectés %v, attendu %v (ni platform/duckdbfake, ni platform/settings)", got, want)
	}
}

// TestHandlersSansStockageDuckDB : aucun handler HTTP n'importe `platform/duckdb` ni ses sous-paquets,
// hors `handlersAvecStockageAutorises`. Les lectures de progression, de campagne et de profil passent par
// `port.ProgressionService`, `port.CampaignService` et `profile.PlayerProfileService` ; le plafond mémoire
// des bases par `port.DatabaseMemory`.
func TestHandlersSansStockageDuckDB(t *testing.T) {
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/api/handlers") {
		vus[src.Rel] = true
		trouves := importsSousArbre(src, racinesStockageInterditesAuxHandlers)
		if _, ok := handlersAvecStockageAutorises[src.Rel]; ok || len(trouves) == 0 {
			continue
		}
		violations = append(violations, src.Rel+" importe "+strings.Join(trouves, ", "))
	}
	for rel := range handlersAvecStockageAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de handlersAvecStockageAutorises")
		}
	}
	if len(violations) > 0 {
		t.Errorf("un handler importe le stockage DuckDB — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
