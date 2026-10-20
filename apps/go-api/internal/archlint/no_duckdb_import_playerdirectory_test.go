// Package archlint — no_duckdb_import_playerdirectory_test.go : ratchet « aucune
// base ouverte par l'annuaire des joueurs » (ADR 0035 D6, 2026-09-16).
//
// POURQUOI. L'annuaire lit quatre registres FICHIERS et le disque, et il PURGE
// une identité. Deux invariants tiennent à ce qu'il n'ouvre jamais une base :
//
//  1. La purge ne touche JAMAIS l'entrepôt partagé (ADR 0035 D6). Les matchs déjà
//     persistés portent aussi les données des adversaires et des coéquipiers du
//     joueur purgé, et l'entrepôt est append-only par construction (ADR 0026).
//     `purge_test.go` compare le sha256 du fichier avant/après ; ce ratchet-ci
//     rend l'écart IMPOSSIBLE plutôt que seulement mesuré.
//  2. Un seul process writer par base (ADR 0013/0016). Le témoin disque de
//     l'annuaire CONSTATE l'existence d'une player DB et, à la purge, la supprime
//     avec son dossier — il ne l'ouvre pas. Ouvrir une base ici, c'est risquer de
//     la prendre pendant que le serveur la tient.
//
// C'est la même discipline que `service.ProfileService`, qui reçoit sa fonction
// d'éviction des handles DuckDB par injection plutôt que d'importer le paquet.
//
// PORTÉE. `internal/service/playerdirectory/` (récursif). Sont interdits l'import
// du paquet d'infrastructure `internal/platform/duckdb` ET celui du driver
// `github.com/duckdb/duckdb-go`. Les `_test.go` sont exclus : un test pourrait
// avoir à fabriquer une base factice — celui de la purge, lui, n'écrit que des
// octets quelconques, ce qui suffit à prouver qu'ils ne bougent pas.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// duckdbImportRE — import d'un paquet DuckDB, sous n'importe quel alias.
var duckdbImportRE = regexp.MustCompile(
	`"levelup/go-api/internal/platform/duckdb"|"github\.com/duckdb/duckdb-go`)

// playerDirectoryPkg — paquet sous ratchet, relatif à internal/.
const playerDirectoryPkg = "service/playerdirectory"

func TestNoDuckDBImportInPlayerDirectory(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	root := filepath.Join(internalRoot, filepath.FromSlash(playerDirectoryPkg))
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("paquet %s introuvable (a-t-il déménagé ?) : %v", playerDirectoryPkg, err)
	}

	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue // un commentaire cite le paquet, il ne l'importe pas
			}
			if duckdbImportRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if len(violations) > 0 {
		t.Errorf("l'annuaire des joueurs importe un paquet DuckDB — il LIT des registres "+
			"fichiers et SUPPRIME des dossiers, il n'ouvre aucune base : la purge ne doit "+
			"pas pouvoir toucher l'entrepôt partagé (ADR 0035 D6) et un seul process peut "+
			"tenir une base (ADR 0013). Injecter ce dont on a besoin, comme "+
			"ProfileService.WithDBEvictor :\n  %s", strings.Join(violations, "\n  "))
	}
}
