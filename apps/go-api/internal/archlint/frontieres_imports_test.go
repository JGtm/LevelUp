// Package archlint — frontieres_imports_test.go : garde-rails d'imports des frontières de couche,
// et l'outil commun qui les exprime.
//
// Les imports se lisent dans l'AST (un chemin cité dans un commentaire ou dans une chaîne n'est pas
// un import) et se comparent au chemin EXACT du paquet : `.../internal/sync` n'interdit pas
// `.../internal/sync/replayartifacts`.
package archlint

import (
	"os"
	"path/filepath"
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

// racinesFixesInterditesAuxHandlers : le moteur de synchronisation et ses écritures (`persist`).
var racinesFixesInterditesAuxHandlers = []string{
	cheminModuleAPI + "sync",
	cheminModuleAPI + "persist",
}

// racinesInterditesAuxHandlers : le moteur de synchronisation, ses écritures (`persist`) et le code
// propre à un titre (`games/<titre>` et ses sous-paquets, suivi en direct compris), pour chaque titre
// donné. Un handler lance une sync par un port (`port.SyncJobService`), lit et écrit par les ports de
// `internal/port` et passe par les adaptateurs sémantiques et les capacités pour tout ce qui dépend du
// titre ; il n'importe aucun de ces paquets.
func racinesInterditesAuxHandlers(titres []string) []string {
	racines := append([]string(nil), racinesFixesInterditesAuxHandlers...)
	for _, titre := range titres {
		racines = append(racines, cheminModuleAPI+"games/"+titre)
	}
	return racines
}

// titresDeInternalGames rend, triés, les répertoires d'`internal/games/` qui ne sont pas des paquets
// inter-titres (`paquetsInterTitres`) : un répertoire neuf y est un titre par défaut, comme pour
// `TestServiceImporteAucunPaquetDeTitre`, de sorte qu'un titre ajouté est refusé aux handlers sans
// qu'on édite ce test.
func titresDeInternalGames(t *testing.T) []string {
	t.Helper()
	entrees, err := os.ReadDir(filepath.Join(apiRootDepuisIci(t), "internal", "games"))
	if err != nil {
		t.Fatalf("lecture d'internal/games : %v", err)
	}
	var titres []string
	for _, e := range entrees {
		if e.IsDir() && !paquetsInterTitres[e.Name()] {
			titres = append(titres, e.Name())
		}
	}
	sort.Strings(titres)
	return titres
}

// TestTitresDeInternalGames : le parcours voit les titres connus et aucun paquet inter-titres ; sans
// quoi `TestHandlersSansMoteurDeSync` ne garderait plus rien.
func TestTitresDeInternalGames(t *testing.T) {
	titres := strings.Join(titresDeInternalGames(t), ",")
	for _, attendu := range []string{"halo_5", "halo_infinite"} {
		if !strings.Contains(","+titres+",", ","+attendu+",") {
			t.Errorf("titre %q absent de la liste des titres d'internal/games : %s", attendu, titres)
		}
	}
	for interdit := range paquetsInterTitres {
		if strings.Contains(","+titres+",", ","+interdit+",") {
			t.Errorf("paquet inter-titres %q pris pour un titre : %s", interdit, titres)
		}
	}
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
	"levelup/go-api/internal/games/titre_a_venir/adapter"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/persistance"
)
`)
	got := importsSousArbre(src, racinesInterditesAuxHandlers([]string{"halo_5", "halo_infinite", "titre_a_venir"}))
	want := []string{
		cheminModuleAPI + "games/halo_5",
		cheminModuleAPI + "games/halo_5/livesync",
		cheminModuleAPI + "games/halo_infinite/haloadapter",
		cheminModuleAPI + "games/titre_a_venir/adapter",
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
	violations := violationsImportsHandlers(t, racinesInterditesAuxHandlers(titresDeInternalGames(t)), handlersAvecMoteurDeSyncAutorises, "handlersAvecMoteurDeSyncAutorises")
	if len(violations) > 0 {
		t.Errorf("un handler importe le moteur de sync, persist ou un paquet de titre — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// violationsImportsHandlers rend, une par ligne, les fichiers non-test de `internal/api/handlers` qui
// importent l'une des racines interdites (sous-paquets compris), hors `autorises` ; une entrée de
// `autorises` sans fichier ou sans violation est périmée et compte aussi (elle se retire avec le code qui
// la justifiait : une allowlist ne survit pas à sa raison d'être).
func violationsImportsHandlers(t *testing.T, racines []string, autorises map[string]string, nomAllowlist string) []string {
	t.Helper()
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/api/handlers") {
		vus[src.Rel] = true
		trouves := importsSousArbre(src, racines)
		_, autorise := autorises[src.Rel]
		switch {
		case len(trouves) == 0 && autorise:
			violations = append(violations, src.Rel+" : entrée autorisée sans violation — la retirer de "+nomAllowlist)
		case len(trouves) > 0 && !autorise:
			violations = append(violations, src.Rel+" importe "+strings.Join(trouves, ", "))
		}
	}
	for rel := range autorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de "+nomAllowlist)
		}
	}
	sort.Strings(violations)
	return violations
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
	violations := violationsImportsHandlers(t, racinesStockageInterditesAuxHandlers, handlersAvecStockageAutorises, "handlersAvecStockageAutorises")
	if len(violations) > 0 {
		t.Errorf("un handler importe le stockage DuckDB — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// racinesServiceInterditesAuxHandlers : le paquet `service` et ses sous-paquets. Un handler tient une
// interface de `internal/port` (ou un type de `domain`) ; il ne nomme aucun type concret de `service` :
// ni service, ni résultat, ni erreur sentinelle, ni constructeur. Les erreurs que la couche HTTP classe
// et les types qu'elle échange sont déclarés avec le port qui les rend.
var racinesServiceInterditesAuxHandlers = []string{
	cheminModuleAPI + "service",
}

// handlersAvecServiceAutorises : handlers qui importent encore `service`, avec la date et la raison. Vide
// depuis le 2026-11-12 ; une entrée ne s'ajoute pas, elle se supprime avec le code qui la justifiait.
var handlersAvecServiceAutorises = map[string]string{}

func TestOutilImportsSousArbreService(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/service/replayview"
	"levelup/go-api/internal/servicemesh"
	"levelup/go-api/internal/port"
)
`)
	got := importsSousArbre(src, racinesServiceInterditesAuxHandlers)
	want := []string{cheminModuleAPI + "service", cheminModuleAPI + "service/replayview"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("imports détectés %v, attendu %v (ni internal/servicemesh, ni internal/port)", got, want)
	}
}

// TestHandlersSansTypeConcretDeService : aucun handler HTTP n'importe `service` ni ses sous-paquets,
// hors `handlersAvecServiceAutorises`. Changer l'implémentation d'un service, ou la remplacer par un
// faux, ne touche pas la couche HTTP.
func TestHandlersSansTypeConcretDeService(t *testing.T) {
	violations := violationsImportsHandlers(t, racinesServiceInterditesAuxHandlers, handlersAvecServiceAutorises, "handlersAvecServiceAutorises")
	if len(violations) > 0 {
		t.Errorf("un handler importe le paquet service — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// schedulerAvecStockageAutorises : fichiers du scheduler qui importent encore `platform/duckdb`, avec la
// date et la raison. Vide depuis le 2026-11-12 : les sondes du contrôle de santé et la purge des rejeux
// lisent par `port.DataHealthStore` et `port.MatchDatesReader` (paquet `cronstore`), le cron du classement
// mondial par `port.WorldLeaderboardStore` (`platform/duckdb/worldrepo`), la contention fichier se classe
// par `platform/dbfault`. Une entrée ne s'ajoute pas : elle se supprime avec le code qui la justifiait.
var schedulerAvecStockageAutorises = map[string]string{}

// TestSchedulerSansStockageDuckDB : aucun fichier du scheduler n'importe le paquet `platform/duckdb`
// (chemin exact : `platform/duckdb/sharedprovider`, la politique de reprise du writer du cron mondial,
// reste permis), hors `schedulerAvecStockageAutorises`. Changer de moteur de base ne touche pas les
// tâches de fond.
func TestSchedulerSansStockageDuckDB(t *testing.T) {
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/scheduler") {
		vus[src.Rel] = true
		trouves := importsInterdits(src, []string{cheminModuleAPI + "platform/duckdb"})
		_, autorise := schedulerAvecStockageAutorises[src.Rel]
		switch {
		case len(trouves) == 0 && autorise:
			violations = append(violations, src.Rel+" : entrée autorisée sans violation — la retirer de schedulerAvecStockageAutorises")
		case len(trouves) > 0 && !autorise:
			violations = append(violations, src.Rel+" importe "+strings.Join(trouves, ", "))
		}
	}
	for rel := range schedulerAvecStockageAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de schedulerAvecStockageAutorises")
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("le scheduler importe platform/duckdb — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// analysisAvecStockageAutorises : fichiers de `analysis/` qui importent encore `database/sql` ou le stockage
// DuckDB, avec la date et la raison. Vide depuis le 2026-11-12 : la sonde de la table de preuve des armes et
// la collecte de télémétrie du réglage Prestige vivent dans `platform/duckdb` ; `analysis/` calcule. Une
// entrée ne s'ajoute pas : elle se supprime avec le code qui la justifiait.
var analysisAvecStockageAutorises = map[string]string{}

// TestAnalysisSansStockage : aucun fichier de `analysis/` (sous-paquets compris) n'importe `database/sql` ni
// `platform/duckdb` (sous-paquets compris). Les algorithmes se testent sans base ; seul `analysis/tactical`
// et `analysis/coordination` étaient gardés avant (tactical_pure_test.go).
func TestAnalysisSansStockage(t *testing.T) {
	racines := []string{cheminModuleAPI + "platform/duckdb", "database/sql"}
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/analysis") {
		vus[src.Rel] = true
		trouves := importsSousArbre(src, racines)
		_, autorise := analysisAvecStockageAutorises[src.Rel]
		switch {
		case len(trouves) == 0 && autorise:
			violations = append(violations, src.Rel+" : entrée autorisée sans violation — la retirer de analysisAvecStockageAutorises")
		case len(trouves) > 0 && !autorise:
			violations = append(violations, src.Rel+" importe "+strings.Join(trouves, ", "))
		}
	}
	for rel := range analysisAvecStockageAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de analysisAvecStockageAutorises")
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("analysis importe le stockage — la requête va dans platform/duckdb, le calcul reste :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
