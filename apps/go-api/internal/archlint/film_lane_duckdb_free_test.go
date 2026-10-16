// Package archlint — film_lane_duckdb_free_test.go : garde-rail de la voie « film » du gate local
// (chantier b1016, étape E1.7).
//
// `scripts/check_test_baseline.sh` (mode autonome, celui de `make gate-push`) joue deux voies en
// parallèle : la voie film (`./internal/games/halo_infinite/film/...`) et la voie DuckDB (tout le
// reste, en `-p 1`). Le parallélisme n'est sûr que parce que la voie film n'ouvre AUCUNE base : le
// driver DuckDB est mono-process (ADR 0013), deux voies qui l'embarqueraient se disputeraient les
// mêmes fichiers et produiraient les durées FANTÔMES et faux-verts du 2026-07-03.
//
// INVARIANT : aucun paquet de `film/...`, tests compris (`go list -test -deps`), ne dépend du
// driver `github.com/duckdb/duckdb-go`. Un import ajouté qui le casse fait échouer ce test ; la voie
// film doit alors repasser dans la voie DuckDB (retirer le paquet de la voie, pas le test).
package archlint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	duckDBDriverPrefix = "github.com/duckdb/duckdb-go"
	// filmLanePattern — motif de la voie film, tel qu'écrit dans scripts/check_test_baseline.sh.
	filmLanePattern = "./internal/games/halo_infinite/film/..."
)

// duckDBDriverImports — dépendances du listing `go list -deps` qui sont le driver DuckDB.
func duckDBDriverImports(listing string) []string {
	var found []string
	for _, line := range strings.Split(listing, "\n") {
		path := strings.TrimSpace(line)
		if path == duckDBDriverPrefix || strings.HasPrefix(path, duckDBDriverPrefix+"/") {
			found = append(found, path)
		}
	}
	return found
}

// TestFilmLanePackagesDoNotDependOnDuckDB — la voie film du gate local ne contient pas DuckDB.
func TestFilmLanePackagesDoNotDependOnDuckDB(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("`go` absent du PATH : le critère transitif exige `go list` (garde-rail non exécuté)")
	}
	racine := racineGoAPI(t)

	// -test : les dépendances des fichiers de test comptent (un test qui ouvre une base suffit).
	// -tags=integration : le tag du gate (certains tests ne se compilent qu'avec lui).
	cmd := exec.Command("go", "list", "-test", "-deps", "-tags=integration", "-f", "{{.ImportPath}}", filmLanePattern)
	cmd.Dir = racine
	sortie, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -test -deps %s : %v", filmLanePattern, err)
	}
	listing := string(sortie)
	if !strings.Contains(listing, "levelup/go-api/internal/games/halo_infinite/film") {
		t.Fatalf("go list n'a rendu aucun paquet film (%d octets) : le garde-rail a perdu sa cible", len(listing))
	}
	if found := duckDBDriverImports(listing); len(found) > 0 {
		t.Errorf(`la voie film (%s) dépend du driver DuckDB : %v

  La voie film du gate local tourne EN PARALLÈLE de la voie DuckDB (scripts/check_test_baseline.sh) ;
  ce parallélisme n'est sûr que sans DuckDB dans la voie film (driver mono-process, ADR 0013).
  CORRECTION : retirer l'import fautif du paquet film, ou sortir ce paquet de la voie film.`, filmLanePattern, found)
	}
}

// TestFilmLaneAnalysisDetectsDuckDB — preuve que l'analyse sait échouer sur un listing synthétique.
func TestFilmLaneAnalysisDetectsDuckDB(t *testing.T) {
	clean := "fmt\nlevelup/go-api/internal/games/halo_infinite/film/types\ngithub.com/pelletier/go-toml/v2\n"
	if found := duckDBDriverImports(clean); len(found) != 0 {
		t.Fatalf("listing sain refusé : %v", found)
	}
	for name, listing := range map[string]string{
		"driver direct": "fmt\ngithub.com/duckdb/duckdb-go/v2\n",
		"sous-paquet":   "fmt\ngithub.com/duckdb/duckdb-go/v2/internal/x\n",
	} {
		if found := duckDBDriverImports(listing); len(found) == 0 {
			t.Errorf("%s : driver DuckDB non détecté", name)
		}
	}
}

// TestFilmLaneScriptDeclaresTheSamePattern — la voie film du script est celle que ce garde-rail juge.
func TestFilmLaneScriptDeclaresTheSamePattern(t *testing.T) {
	root := repoRootForWorkflows(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "check_test_baseline.sh"))
	if err != nil {
		t.Fatalf("lecture de scripts/check_test_baseline.sh : %v", err)
	}
	if !strings.Contains(string(data), "FILM_LANE_PATTERN='"+filmLanePattern+"'") {
		t.Errorf("scripts/check_test_baseline.sh ne déclare plus FILM_LANE_PATTERN='%s' : la voie jugée par TestFilmLanePackagesDoNotDependOnDuckDB n'est plus celle du gate local — aligner les deux", filmLanePattern)
	}
}
