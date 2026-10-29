// Package archlint — ci_web_reads_go_test.go : les sources Go que des tests web relisent sont
// couvertes par la classe `croisee` du filtre de chemins de ci.yml.
//
// POURQUOI. Sur une branche de travail, le filtre de chemins saute les jobs web quand le push ne
// touche que du Go. Or des tests Vitest lisent des sources Go (boot_gate.go, les balises `json:`
// du rejeu, les chargeurs de libellés…) pour tenir la parité des deux piles : un push Go seul
// modifiant ces fichiers ne les rejouait pas (revue adversariale E10.1, constat C4). Un fichier
// Go lu par un test web classe donc dans `croisee` (les deux piles tournent).
//
// RÈGLE. Tout chemin `go-api/...` ou `libs/...` (une brique Go, ADR 0041) écrit dans une ligne de
// code (hors commentaire) d'un test sous apps/web/src est couvert par un motif de la classe `croisee` de ci.yml. Un chemin sans extension
// désigne un dossier lu fichier par fichier : un fichier de ce dossier doit être couvert.
// Analyse textuelle, sans parseur YAML ; les fonctions pures sont testées sur des textes
// synthétiques défectueux (TestWebReadsGoGuardDetectsDefects).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// goPathInWebTestRe : un chemin sous go-api ou sous une brique de `libs/` écrit dans une chaîne de
// test.
var goPathInWebTestRe = regexp.MustCompile(`(?:go-api|\blibs)/[A-Za-z0-9_./-]*[A-Za-z0-9_]`)

// goPathsReadByWebTest rend les chemins (relatifs au dépôt : `apps/go-api/...`, `libs/...`) cités par les lignes
// de code d'un fichier de test web ; les lignes de commentaire sont ignorées.
func goPathsReadByWebTest(text string) []string {
	seen := map[string]bool{}
	var paths []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		for _, m := range goPathInWebTestRe.FindAllString(line, -1) {
			p := m
			if strings.HasPrefix(m, "go-api/") {
				p = "apps/" + m
			}
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// croiseePatterns rend les motifs de la classe `croisee` du filtre `dorny/paths-filter` : les
// lignes `- '...'` qui suivent la clé `croisee:` jusqu'à la clé suivante.
func croiseePatterns(ciText string) []string {
	var patterns []string
	in := false
	for _, line := range strings.Split(strings.ReplaceAll(ciText, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "croisee:":
			in = true
		case !in || trimmed == "":
		case strings.HasPrefix(trimmed, "- "):
			patterns = append(patterns, strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), `'"`))
		default:
			in = false
		}
	}
	return patterns
}

// globMatches dit si le chemin satisfait le motif (`**` = tout, `*` = tout sauf `/`).
func globMatches(pattern, path string) bool {
	var re strings.Builder
	re.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch {
		case strings.HasPrefix(pattern[i:], "**/"):
			re.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "**"):
			re.WriteString(".*")
			i++
		case pattern[i] == '*':
			re.WriteString("[^/]*")
		default:
			re.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	re.WriteString("$")
	return regexp.MustCompile(re.String()).MatchString(path)
}

// coveredByCroisee dit si le chemin lu (fichier, ou dossier sans extension) est couvert.
func coveredByCroisee(patterns []string, path string) bool {
	probe := path
	if !strings.Contains(filepath.Base(path), ".") {
		probe = path + "/x.go"
	}
	for _, p := range patterns {
		if globMatches(p, probe) {
			return true
		}
	}
	return false
}

// uncoveredGoReads rend, par chemin Go non couvert, les tests web qui le lisent.
func uncoveredGoReads(ciText string, webTests map[string]string) map[string][]string {
	patterns := croiseePatterns(ciText)
	out := map[string][]string{}
	for file, text := range webTests {
		for _, p := range goPathsReadByWebTest(text) {
			if !coveredByCroisee(patterns, p) {
				out[p] = append(out[p], file)
			}
		}
	}
	for _, files := range out {
		sort.Strings(files)
	}
	return out
}

func TestWebTestsReadingGoSourcesAreCoveredByCroisee(t *testing.T) {
	root := repoRootForWorkflows(t)
	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("lecture de ci.yml : %v", err)
	}
	if len(croiseePatterns(string(ci))) == 0 {
		t.Fatal("classe `croisee` introuvable dans ci.yml : le garde-rail a perdu sa cible")
	}
	webTests := map[string]string{}
	err = filepath.WalkDir(filepath.Join(root, "apps", "web", "src"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() || !(strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".test.tsx")) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		webTests[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de apps/web/src : %v", err)
	}
	reads := 0
	for _, text := range webTests {
		reads += len(goPathsReadByWebTest(text))
	}
	if reads < 5 {
		t.Fatalf("garde-rail vacant : %d chemin(s) Go lu(s) par les tests web (attendu au moins 5)", reads)
	}
	for path, files := range uncoveredGoReads(string(ci), webTests) {
		t.Errorf("%s est lu par %s mais n'est couvert par aucun motif de la classe `croisee` de ci.yml : "+
			"un push Go seul sauterait les jobs web. Ajouter le chemin sous `croisee:`.", path, strings.Join(files, ", "))
	}
}

// TestWebReadsGoGuardDetectsDefects : l'analyse voit un chemin non couvert et accepte les formes
// couvertes.
func TestWebReadsGoGuardDetectsDefects(t *testing.T) {
	const ci = `
          filters: |
            go:
              - 'apps/go-api/**'
            croisee:
              - '**/openapi.yaml'
              - 'apps/go-api/cmd/server/boot_gate.go'
              - 'apps/go-api/internal/film/replay/**'
              - 'apps/go-api/internal/mappings/loader_*.go'
              - 'libs/film/replay/**'
            autre:
              - '**'
`
	tests := map[string]string{
		"a.test.ts": "resolve(r, 'apps/go-api/cmd/server/boot_gate.go')\nresolve(r, 'apps/go-api/internal/film/replay')\n" +
			"resolve(r, 'apps/go-api/internal/mappings/loader_labels.go')\nfetch('apps/go-api/api/openapi.yaml')",
		"b.test.ts": "// apps/go-api/internal/ignore/commentaire.go\n * go-api/internal/ignore/doc.go\n",
		"d.test.ts": "resolve(REPO, 'libs/film/replay/document_chronicle.go')\nresolve(REPO, 'apps/web/src/lib/x.ts')",
	}
	if got := uncoveredGoReads(ci, tests); len(got) != 0 {
		t.Fatalf("faux positifs : %v", got)
	}
	tests["c.test.ts"] = "readFileSync(resolve(r, '../go-api/internal/analysis/weapon_range.go'))\n" +
		"readdirSync(r + 'apps/go-api/internal/film/grammar')\nresolve(REPO, 'libs/film/internal/grammar')"
	got := uncoveredGoReads(ci, tests)
	if len(got) != 3 || got["apps/go-api/internal/analysis/weapon_range.go"][0] != "c.test.ts" ||
		got["apps/go-api/internal/film/grammar"][0] != "c.test.ts" || got["libs/film/internal/grammar"][0] != "c.test.ts" {
		t.Fatalf("défauts vus : %v (le fichier et le dossier non couverts étaient attendus)", got)
	}
	if p := croiseePatterns(ci); len(p) != 5 {
		t.Fatalf("motifs croisee lus : %v (5 attendus, la classe suivante ne doit pas y entrer)", p)
	}
}
