// Package archlint — no_hardcoded_film_cache_dirs_test.go : garde-rail de la
// centralisation du lot hygiène 5.3 (`.ai/V7.5/REGISTRE_REPORTS.md`, L62), étendu au NOM DE
// FICHIER des chunks par J2.3 du plan de suite de l'audit du décodeur de film (2026-09-26).
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
//
// LE NOM DU FICHIER D'UN CHUNK (`chunk_NN.bin`) suit la même règle depuis J2.3 : il est déclaré
// par `chunkName` dans `filmcache.go` et se construit par [filmcache.CheminDuChunk] ; un chunk du
// cache se LIT par [filmcache.LireChunk], qui le valide contre la taille de son manifeste. Le
// client Halo (`haloclient.LocalFilmCache.LoadChunk`) et quatre outils recomposaient le nom
// chez eux — et, avec lui, une lecture sans validation.
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
//   - filmcache.go : définition canonique de `film_manifests` / `film_chunks`.
//   - registry_film_facts.go : définition canonique de `film_facts` (constante
//     `title.SousDossierFilmFacts`, lot 4.1.1-c du 2026-09-17). Ce troisième
//     sous-dossier du cache film n'est PAS défini dans `filmcache` et il ne peut
//     pas l'être : son chemin se construit sur `PathResolver.CacheRootDir()`
//     (source unique de `data/cache`) et il est rangé PAR TITRE, alors que
//     `filmcache` range les chunks à plat et ne connaît pas de titre. Sans cette
//     entrée, le ratchet mordrait sa propre source.
//   - pooled_client.go : "film_chunks" y est un NOM D'APPEL pour la métrique
//     `observeHaloCall` (observabilité réseau), pas un chemin disque — rien à
//     centraliser via filmcache pour cet usage.
var filmCacheDirAllowlist = map[string]bool{
	"internal/games/halo_infinite/film/filmcache/filmcache.go": true,
	"internal/domain/title/registry_film_facts.go":             true,
	"internal/sync/pooled_client.go":                           true,
}

// filmCacheDirRE — les TROIS sous-dossiers du cache film. `film_facts` y est
// entré au lot 4.1.1-c (2026-09-17), DANS LE COMMIT qui a créé le littéral :
// apprendre le nom plus tard aurait laissé le temps à une copie de naître
// (CLAUDE.md règle 6).
var filmCacheDirRE = regexp.MustCompile(`"film_manifests"|"film_chunks"|"film_facts"`)

// filmChunkNameAllowlist : fichiers où le nom de fichier d'un chunk est TOLÉRÉ (2026-09-26).
//   - filmcache.go : définition canonique (`chunkName`).
//   - film/internal/source/source.go et film/internal/grammar/film_packets.go : les deux
//     lecteurs d'un RÉPERTOIRE de chunks des couches de décodage (énumération `chunk_*.bin` et
//     lecture d'un chunk par numéro). `source` ne peut pas importer `filmcache` (qui l'importe),
//     et toucher l'une ou l'autre couche déplace son empreinte de révision, alors que J2 ne monte
//     aucune révision. Critère de retrait : la première montée de révision de `source` ou de
//     `grammar` qui fait descendre la déclaration du nom dans `source` (que `filmcache`
//     consommerait) retire les deux entrées.
var filmChunkNameAllowlist = map[string]bool{
	"internal/games/halo_infinite/film/filmcache/filmcache.go":           true,
	"internal/games/halo_infinite/film/internal/source/source.go":        true,
	"internal/games/halo_infinite/film/internal/grammar/film_packets.go": true,
}

// filmChunkNameRE — le nom de fichier d'un chunk sous toutes ses formes recopiables : format
// (`chunk_%02d.bin`), motif (`chunk_*.bin`), nom littéral (`chunk_00.bin"`) et préfixe (`"chunk_"`).
var filmChunkNameRE = regexp.MustCompile(`chunk_(%0?\d*d|\*|\d+)\.bin|"chunk_"`)

func TestNoHardcodedFilmCacheDirs(t *testing.T) {
	violations := litterauxHorsAllowlist(t, filmCacheDirRE, filmCacheDirAllowlist)
	if len(violations) > 0 {
		t.Errorf("littéral \"film_manifests\"/\"film_chunks\"/\"film_facts\" en dur interdit (lot hygiène 5.3, "+
			"L62) — passer par filmcache.ManifestsRoot/ChunksRoot/ManifestPath/ChunkDir, ou par "+
			"PathResolver.FilmFactsDir/FilmFactsPath pour \"film_facts\" :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

func TestNoHardcodedFilmChunkName(t *testing.T) {
	violations := litterauxHorsAllowlist(t, filmChunkNameRE, filmChunkNameAllowlist)
	if len(violations) > 0 {
		t.Errorf("nom de fichier de chunk (`chunk_NN.bin`) recopié interdit (J2.3, 2026-09-26) — "+
			"construire le chemin par filmcache.CheminDuChunk, lire un chunk du cache par "+
			"filmcache.LireChunk (validé par la taille du manifeste) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// litterauxHorsAllowlist walk tout le module (`internal/` + `cmd/`, hors tests et commentaires)
// et rend les lignes où `re` trouve une occurrence dans un fichier absent de `allowlist`.
func litterauxHorsAllowlist(t *testing.T, re *regexp.Regexp, allowlist map[string]bool) []string {
	t.Helper()
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
		if allowlist[rel] {
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
			if re.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", goAPIRoot, err)
	}
	return violations
}
