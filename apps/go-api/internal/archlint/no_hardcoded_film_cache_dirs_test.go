// Package archlint — no_hardcoded_film_cache_dirs_test.go : garde-rail de la
// centralisation du lot hygiène 5.3 (`.ai/V7.5/REGISTRE_REPORTS.md`, L62).
//
// Les noms des deux sous-dossiers du cache film disque (`film_manifests`,
// `film_chunks`) sont déclarés UNE fois dans
// `internal/games/halo_infinite/film/filmcache/filmcache.go` (constantes
// `manifestsDir`/`chunksDir`), qui expose [filmcache.ManifestsRoot],
// [filmcache.ChunksRoot], [filmcache.ManifestPath] et [filmcache.ChunkDir] pour
// construire ces chemins. Avant ce lot, le littéral était recopié à la main dans
// 3 outils (`cmd/fetch_film_chunks`, `cmd/killsource`,
// `cmd/levelup/cmd_backfill_killsource_selection.go`) et dans le lecteur legacy
// `internal/sync/haloclient/local_film_cache.go` : une disposition de cache qui
// dérive silencieusement dès que le nom change ailleurs (même défaut que celui
// qui a justifié la création du paquet `filmcache`, cf. son en-tête). Ce test
// walk TOUT le module (`internal/` + `cmd/`) et interdit toute NOUVELLE
// occurrence du littéral hors du paquet canonique.
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

// filmCacheDirAllowlist : fichiers (chemin relatif depuis apps/go-api) où le
// littéral est TOLÉRÉ.
//   - filmcache.go : définition canonique des constantes.
//   - pooled_client.go : "film_chunks" y est un NOM D'APPEL pour la métrique
//     `observeHaloCall` (observabilité réseau), pas un chemin disque — rien à
//     centraliser via filmcache pour cet usage.
var filmCacheDirAllowlist = map[string]bool{
	"internal/games/halo_infinite/film/filmcache/filmcache.go": true,
	"internal/sync/pooled_client.go":                           true,
}

var filmCacheDirRE = regexp.MustCompile(`"film_manifests"|"film_chunks"`)

func TestNoHardcodedFilmCacheDirs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	var violations []string
	err := filepath.WalkDir(goAPIRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != goAPIRoot && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(goAPIRoot, path)
		rel = filepath.ToSlash(rel)
		if filmCacheDirAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if filmCacheDirRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", goAPIRoot, err)
	}
	if len(violations) > 0 {
		t.Errorf("littéral \"film_manifests\"/\"film_chunks\" en dur interdit (lot hygiène 5.3, "+
			"L62) — passer par filmcache.ManifestsRoot/ChunksRoot/ManifestPath/ChunkDir :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
