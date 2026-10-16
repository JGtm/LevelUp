// Package archlint — ci_go_shards_test.go : garde-rails du découpage de la suite Go de la CI en lots
// (chantier b1016, étape E1.2).
//
// SOURCE UNIQUE : .github/ci/go-shards.txt (lots, listes de tests lourds, budgets de délai), lue par
// scripts/ci_go_shard.sh. Quatre pannes silencieuses sont fermées ici :
//
//  1. un lot défini mais absent de la matrice de ci.yml (ses paquets ne tourneraient jamais), ou
//     l'inverse (un job qui échouerait sur un lot inconnu) ;
//  2. un test « lourd » renommé ou supprimé : la liste nommée qui sert à la fois de `-run` et de
//     `-skip` vieillirait sans bruit et le découpage de `grammar` se déséquilibrerait ;
//  3. une définition qui perd ou duplique un paquet : le script la refuse (partition exacte contre
//     `go list -tags=integration ./...`) — ce test exécute le script sur la définition réelle ;
//  4. un script qui ne sait plus refuser : il est exécuté sur une définition défectueuse.
//
// PARSING : textuel, sans parseur YAML (cf. ci_deploy_triggers_test.go). Les tests d'exécution
// lancent bash ; hors Linux ils cherchent le bash de Git for Windows (jamais le lanceur WSL de
// System32, qui ne comprend pas les chemins Windows) et se sautent à défaut — la CI Linux les joue.
package archlint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// shardLot — une ligne `lot` de go-shards.txt.
type shardLot struct {
	name     string
	filter   string // "-", "run:<liste>" ou "skip:<liste>"
	patterns []string
}

// shardDefinition — go-shards.txt lu.
type shardDefinition struct {
	heavy map[string][]string
	lots  []shardLot
}

var heavyTestNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// parseShardDefinition lit la syntaxe documentée en tête de .github/ci/go-shards.txt.
func parseShardDefinition(text string) (shardDefinition, error) {
	def := shardDefinition{heavy: map[string][]string{}}
	for n, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := raw
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "heavy":
			if len(f) < 3 {
				return def, fmt.Errorf("ligne %d : heavy sans nom de liste ou sans test", n+1)
			}
			for _, name := range f[2:] {
				if !heavyTestNameRe.MatchString(name) {
					return def, fmt.Errorf("ligne %d : nom de test invalide %q dans la liste %s", n+1, name, f[1])
				}
			}
			def.heavy[f[1]] = f[2:]
		case "lot":
			if len(f) < 4 {
				return def, fmt.Errorf("ligne %d : lot incomplet", n+1)
			}
			def.lots = append(def.lots, shardLot{name: f[1], filter: f[2], patterns: f[3:]})
		case "timeout-defaut", "timeout":
			// Budgets : jugés par le script (durée valide, lots connus).
		default:
			return def, fmt.Errorf("ligne %d : directive inconnue %q", n+1, f[0])
		}
	}
	for _, lot := range def.lots {
		if lot.filter == "-" {
			continue
		}
		kind, list, ok := strings.Cut(lot.filter, ":")
		if !ok || (kind != "run" && kind != "skip") {
			return def, fmt.Errorf("lot %s : filtre %q inconnu", lot.name, lot.filter)
		}
		if _, known := def.heavy[list]; !known {
			return def, fmt.Errorf("lot %s : liste %q non déclarée par heavy", lot.name, list)
		}
	}
	return def, nil
}

// matrixLotsOfCoverageJob — la liste `lot: [...]` de la matrice du job `go-coverage`.
func matrixLotsOfCoverageJob(lines []string) ([]string, error) {
	block, ok := jobBlock(lines, "go-coverage")
	if !ok {
		return nil, fmt.Errorf("job go-coverage absent")
	}
	for _, line := range block {
		if isBlankOrComment(line) {
			continue
		}
		if value, found := strings.CutPrefix(strings.TrimSpace(line), "lot:"); found {
			items := splitFlowSequence(strings.TrimSpace(value))
			if len(items) == 0 {
				return nil, fmt.Errorf("matrice `lot:` vide ou en forme non flow : %q", value)
			}
			return items, nil
		}
	}
	return nil, fmt.Errorf("clé `lot:` absente de la matrice du job go-coverage")
}

// lotNameDifferences — noms présents d'un seul côté (définition, matrice), triés.
func lotNameDifferences(def shardDefinition, matrix []string) (onlyDef, onlyMatrix []string) {
	inDef := map[string]bool{}
	for _, lot := range def.lots {
		inDef[lot.name] = true
	}
	inMatrix := map[string]bool{}
	for _, m := range matrix {
		inMatrix[m] = true
		if !inDef[m] {
			onlyMatrix = append(onlyMatrix, m)
		}
	}
	for name := range inDef {
		if !inMatrix[name] {
			onlyDef = append(onlyDef, name)
		}
	}
	sort.Strings(onlyDef)
	sort.Strings(onlyMatrix)
	return onlyDef, onlyMatrix
}

func readShardDefinition(t *testing.T, root string) shardDefinition {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".github", "ci", "go-shards.txt"))
	if err != nil {
		t.Fatalf("lecture de .github/ci/go-shards.txt impossible : %v (le garde-rail a perdu sa cible)", err)
	}
	def, err := parseShardDefinition(string(data))
	if err != nil {
		t.Fatalf(".github/ci/go-shards.txt : %v", err)
	}
	if len(def.lots) < 2 {
		t.Fatalf(".github/ci/go-shards.txt : %d lot(s) — le découpage a disparu ou le fichier est illisible", len(def.lots))
	}
	return def
}

// TestCIGoShardsDefinitionAndMatrixAgree — la matrice de ci.yml est exactement la liste des lots.
func TestCIGoShardsDefinitionAndMatrixAgree(t *testing.T) {
	root := repoRootForWorkflows(t)
	def := readShardDefinition(t, root)
	matrix, err := matrixLotsOfCoverageJob(workflowLines(t, root, "ci.yml"))
	if err != nil {
		t.Fatalf("ci.yml : %v", err)
	}
	onlyDef, onlyMatrix := lotNameDifferences(def, matrix)
	if len(onlyDef) > 0 {
		t.Errorf("lots définis dans .github/ci/go-shards.txt mais absents de la matrice du job go-coverage (leurs paquets ne tourneraient JAMAIS) : %v", onlyDef)
	}
	if len(onlyMatrix) > 0 {
		t.Errorf("lots de la matrice du job go-coverage inconnus de .github/ci/go-shards.txt (le job échouerait) : %v", onlyMatrix)
	}
}

// TestCIGoShardsHeavyTestsExist — chaque test de la liste lourde existe encore, dans le paquet
// que filtre un lot `run:` / `skip:` de cette liste.
func TestCIGoShardsHeavyTestsExist(t *testing.T) {
	root := repoRootForWorkflows(t)
	def := readShardDefinition(t, root)
	goAPI := filepath.Join(root, "apps", "go-api")

	checked := 0
	for _, lot := range def.lots {
		_, list, ok := strings.Cut(lot.filter, ":")
		if !ok {
			continue
		}
		for _, pat := range lot.patterns {
			if strings.HasPrefix(pat, "!") || strings.HasSuffix(pat, "...") {
				continue
			}
			dir := filepath.Join(goAPI, filepath.FromSlash(strings.TrimPrefix(pat, "./")))
			src := packageTestSources(t, dir)
			for _, name := range def.heavy[list] {
				checked++
				if !regexp.MustCompile(`(?m)^func ` + regexp.QuoteMeta(name) + `\(t \*testing\.T\)`).MatchString(src) {
					t.Errorf(`liste lourde %q : le test %s n'existe plus dans %s

  Les lots qui l'utilisent (-run / -skip) se déséquilibreraient sans bruit. CORRECTION : mettre à
  jour la ligne « heavy %s » de .github/ci/go-shards.txt (nom renommé, ou test supprimé).`,
						list, name, pat, list)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucun test lourd vérifié : le garde-rail a perdu sa cible (ni lot run:/skip:, ni liste heavy)")
	}
}

// packageTestSources — concaténation des fichiers *_test.go du dossier (sans le tag `research`).
func packageTestSources(t *testing.T, dir string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun fichier de test dans %s (err=%v) : le lot filtré désigne un paquet disparu", dir, err)
	}
	var b strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String()
}

// bashForScripts — un bash capable d'exécuter les scripts du dépôt, ou saute le test.
func bashForScripts(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("`go` absent du PATH : le script énumère les paquets avec `go list`")
	}
	if runtime.GOOS == "windows" {
		for _, c := range []string{`C:\Program Files\Git\bin\bash.exe`, `C:\Program Files\Git\usr\bin\bash.exe`} {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
		t.Skip("bash de Git for Windows introuvable : le script ne s'exécute pas ici (la CI Linux le joue)")
	}
	path, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash absent du PATH")
	}
	return path
}

// runShardScript exécute scripts/ci_go_shard.sh (définition réelle, ou celle de defFile) et rend
// sa sortie standard, son erreur standard et son erreur d'exécution.
func runShardScript(t *testing.T, root, defFile string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(bashForScripts(t), append([]string{"scripts/ci_go_shard.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	if defFile != "" {
		cmd.Env = append(cmd.Env, "CI_GO_SHARDS_FILE="+filepath.ToSlash(defFile))
	}
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	return out.String(), errb.String(), err
}

// TestCIGoShardsPartitionIsExact — la définition réelle partitionne exactement `go list ./...`.
func TestCIGoShardsPartitionIsExact(t *testing.T) {
	root := repoRootForWorkflows(t)
	stdout, stderr, err := runShardScript(t, root, "", "--verifier")
	if err != nil {
		t.Fatalf("scripts/ci_go_shard.sh --verifier : %v\n%s%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "partition exacte") {
		t.Errorf("sortie inattendue du vérificateur :\n%s", stdout)
	}
}

// TestCIGoShardScriptRefusesBrokenDefinitions — le script sait refuser : une définition qui
// duplique ou perd un paquet n'est jamais rendue comme un lot valide.
func TestCIGoShardScriptRefusesBrokenDefinitions(t *testing.T) {
	root := repoRootForWorkflows(t)
	write := func(content string) string {
		path := filepath.Join(t.TempDir(), "go-shards.txt")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	const head = "timeout-defaut 600s\nheavy h TestA\n"

	if _, stderr, err := runShardScript(t, root, write(head+"lot a - ./internal/archlint\nlot r - @reste\n"), "--verifier"); err != nil {
		t.Fatalf("définition valide refusée (le harnais est faux) : %v\n%s", err, stderr)
	}

	for name, tc := range map[string]struct{ def, want string }{
		"paquet dans deux lots":  {head + "lot a - ./internal/archlint\nlot b - ./internal/archlint\nlot r - @reste\n", "paire complémentaire"},
		"motif périmé":           {head + "lot a - ./internal/n_existe_pas/...\nlot r - @reste\n", "ne désigne aucun paquet"},
		"filtre sans paire":      {head + "lot a run:h ./internal/archlint\nlot r - @reste\n", "sans lot complémentaire"},
		"complément vide":        {head + "lot a - ./...\nlot r - @reste\n", "complément"},
		"pas de complément":      {head + "lot a - ./internal/archlint\n", "complément"},
		"complément pas dernier": {head + "lot r - @reste\nlot a - ./internal/archlint\n", "dernier lot"},
	} {
		_, stderr, err := runShardScript(t, root, write(tc.def), "--verifier")
		if err == nil {
			t.Errorf("%s : définition défectueuse ACCEPTÉE par le script", name)
			continue
		}
		if !strings.Contains(stderr, tc.want) {
			t.Errorf("%s : refus sans le message attendu %q :\n%s", name, tc.want, stderr)
		}
	}
}

// TestCIGoShardsAnalysisDetectsDefects — l'analyse textuelle sait échouer.
func TestCIGoShardsAnalysisDetectsDefects(t *testing.T) {
	good := "timeout-defaut 600s\nheavy h TestA TestB # commentaire\nlot a run:h ./x\nlot b skip:h ./x\nlot r - @reste\n"
	def, err := parseShardDefinition(good)
	if err != nil || len(def.lots) != 3 || len(def.heavy["h"]) != 2 {
		t.Fatalf("définition conforme mal lue : lots=%d heavy=%v err=%v", len(def.lots), def.heavy, err)
	}

	for name, text := range map[string]string{
		"nom de test invalide": "heavy h Test-A\nlot r - @reste\n",
		"directive inconnue":   "lots a - ./x\n",
		"liste non déclarée":   "lot a run:zz ./x\nlot r - @reste\n",
		"filtre inconnu":       "lot a only:h ./x\nlot r - @reste\n",
		"heavy sans test":      "heavy h\n",
		"lot incomplet":        "lot a -\n",
	} {
		if _, err := parseShardDefinition(text); err == nil {
			t.Errorf("%s : définition défectueuse acceptée", name)
		}
	}

	workflow := splitWorkflow("jobs:\n  go-coverage:\n    strategy:\n      matrix:\n        lot: [a, b, c]\n    steps: []\n")
	matrix, err := matrixLotsOfCoverageJob(workflow)
	if err != nil || len(matrix) != 3 {
		t.Fatalf("matrice mal lue : %v err=%v", matrix, err)
	}
	onlyDef, onlyMatrix := lotNameDifferences(def, matrix)
	if fmt.Sprint(onlyDef) != "[r]" || fmt.Sprint(onlyMatrix) != "[c]" {
		t.Errorf("écarts définition/matrice : onlyDef=%v onlyMatrix=%v, attendu [r] et [c]", onlyDef, onlyMatrix)
	}
	if _, err := matrixLotsOfCoverageJob(splitWorkflow("jobs:\n  autre:\n    steps: []\n")); err == nil {
		t.Error("job go-coverage absent non détecté")
	}
}
