// Package archlint — film_lane_duckdb_free_test.go : garde-rail de la voie « briques » du gate local
// (chantier b1016, étape E1.7).
//
// `scripts/check_test_baseline.sh` (mode autonome, celui de `make gate-push`) joue deux voies en
// parallèle : la voie des briques (les quatre briques publiées, jouées depuis apps/go-api à la version
// que fixe go.mod, ADR 0043) et la voie DuckDB (tout go-api, en `-p 1`). Le parallélisme n'est sûr que
// parce que la voie des briques n'ouvre AUCUNE base : le driver DuckDB est mono-process (ADR 0013),
// deux voies qui l'embarqueraient se disputeraient les mêmes fichiers et produiraient les durées
// FANTÔMES et faux-verts du 2026-07-03.
//
// INVARIANT : aucun paquet de la voie, tests compris (`go list -test -deps`), ne dépend du driver
// `github.com/duckdb/duckdb-go`. Une version de brique qui le casse fait échouer ce test ; le paquet
// doit alors repasser dans la voie DuckDB (retirer son motif de la voie, pas le test).
package archlint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const duckDBDriverPrefix = "github.com/duckdb/duckdb-go"

// filmLanePatterns — motifs de la voie des briques, tels qu'écrits dans scripts/check_test_baseline.sh
// (tableau FILM_LANE_PATTERNS), joués depuis apps/go-api.
var filmLanePatterns = []string{
	importBriqueFilm + "/...",
	"github.com/JGtm/halo-api/...",
	"github.com/JGtm/halo-infinite-files-reader/...",
	"github.com/JGtm/halo-infinite-map-images/...",
}

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

// TestFilmLanePackagesDoNotDependOnDuckDB — la voie des briques du gate local ne contient pas DuckDB.
func TestFilmLanePackagesDoNotDependOnDuckDB(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("`go` absent du PATH : le critère transitif exige `go list` (garde-rail non exécuté)")
	}
	// -test : les dépendances des fichiers de test comptent (un test qui ouvre une base suffit).
	// -tags=integration : le tag du gate (certains tests ne se compilent qu'avec lui).
	args := append([]string{"list", "-test", "-deps", "-tags=integration", "-f", "{{.ImportPath}}"}, filmLanePatterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = "../.."
	sortie, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -test -deps %v : %v", filmLanePatterns, err)
	}
	listing := string(sortie)
	if !strings.Contains(listing, importBriqueFilm) {
		t.Fatalf("go list n'a rendu aucun paquet film (%d octets) : le garde-rail a perdu sa cible", len(listing))
	}
	if found := duckDBDriverImports(listing); len(found) > 0 {
		t.Errorf(`la voie des briques (%v) dépend du driver DuckDB : %v

  La voie des briques du gate local tourne EN PARALLÈLE de la voie DuckDB (scripts/check_test_baseline.sh) ;
  ce parallélisme n'est sûr que sans DuckDB dans cette voie (driver mono-process, ADR 0013).
  CORRECTION : sortir le paquet fautif de la voie.`, filmLanePatterns, found)
	}
}

// TestFilmLaneAnalysisDetectsDuckDB — preuve que l'analyse sait échouer sur un listing synthétique.
func TestFilmLaneAnalysisDetectsDuckDB(t *testing.T) {
	clean := "fmt\ngithub.com/JGtm/halo-infinite-film-decoder/types\ngithub.com/pelletier/go-toml/v2\n"
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

// TestFilmLaneScriptDeclaresTheSamePattern — la voie du script est celle que ce garde-rail juge.
func TestFilmLaneScriptDeclaresTheSamePattern(t *testing.T) {
	root := repoRootForWorkflows(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "check_test_baseline.sh"))
	if err != nil {
		t.Fatalf("lecture de scripts/check_test_baseline.sh : %v", err)
	}
	var attendu strings.Builder
	attendu.WriteString("FILM_LANE_PATTERNS=(\n")
	for _, p := range filmLanePatterns {
		attendu.WriteString("  '" + p + "'\n")
	}
	attendu.WriteString(")")
	if !strings.Contains(strings.ReplaceAll(string(data), "\r\n", "\n"), attendu.String()) {
		t.Errorf("scripts/check_test_baseline.sh ne déclare plus :\n%s\nla voie jugée par TestFilmLanePackagesDoNotDependOnDuckDB n'est plus celle du gate local — aligner les deux", attendu.String())
	}
}
