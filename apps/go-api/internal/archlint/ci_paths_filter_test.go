// Package archlint — ci_paths_filter_test.go : garde-rails du filtre de chemins de ci.yml
// (chantier b1016, étape E1.6) et de la règle d'annulation des runs périmés.
//
// INVARIANT PROTÉGÉ : le filtre de chemins ne s'applique qu'aux branches de travail. Sur `main`,
// `feat/v75` et les PR, TOUS les jobs tournent, inconditionnellement — sinon un défaut du filtre
// laisserait du code non testé jusqu'au déploiement (deploy.yml attend le verdict du run de `main`).
// Concrètement : tout job de ci.yml qui dépend de `changes` porte dans son `if:` le contournement
// (PR, main, feat/v75) AVANT toute lecture de `needs.changes.outputs.*`.
//
// PARSING : textuel et ANCRÉ, sans parseur YAML (même parti que ci_deploy_triggers_test.go, dont les
// helpers sont réutilisés). Les fonctions d'analyse sont pures et testées sur des textes synthétiques
// défectueux (TestCIPathsFilterAnalysisDetectsDefects) : elles prouvent qu'elles savent échouer.
package archlint

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// requiredBypass — les trois contournements, dans l'ordre où ils doivent précéder la lecture
// du filtre.
var requiredBypass = []string{
	"github.event_name == 'pull_request'",
	"github.ref == 'refs/heads/main'",
	"github.ref == 'refs/heads/feat/v75'",
}

// jobTopLevelNames — noms des jobs de `jobs:` (clés à 2 espaces d'indentation), dans l'ordre.
func jobTopLevelNames(lines []string) []string {
	var names []string
	inJobs := false
	for _, line := range lines {
		if isBlankOrComment(line) {
			continue
		}
		indent := lineIndent(line)
		if indent == 0 {
			inJobs = strings.TrimSpace(line) == "jobs:"
			continue
		}
		if inJobs && indent == 2 && strings.HasSuffix(strings.TrimSpace(line), ":") {
			names = append(names, strings.TrimSuffix(strings.TrimSpace(line), ":"))
		}
	}
	return names
}

// jobLevelScalar — valeur de la clé `key:` posée au niveau du job (4 espaces), sur UNE ligne, ou
// "" si absente. Pour `needs:` la forme flow `[a, b]` est rendue telle quelle.
func jobLevelScalar(block []string, key string) string {
	for _, line := range block[1:] {
		if isBlankOrComment(line) || lineIndent(line) != 4 {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(trimmed, key+":"); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// jobNeeds — jobs listés dans `needs:` (forme scalaire, flow ou bloc).
func jobNeeds(block []string) []string {
	for i, line := range block {
		if i == 0 || isBlankOrComment(line) || lineIndent(line) != 4 {
			continue
		}
		trimmed := strings.TrimSpace(line)
		value, ok := strings.CutPrefix(trimmed, "needs:")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch {
		case strings.HasPrefix(value, "["):
			return splitFlowSequence(value)
		case value != "":
			return []string{cleanScalar(value)}
		}
		var needs []string
		for _, next := range block[i+1:] {
			if isBlankOrComment(next) {
				continue
			}
			t := strings.TrimSpace(next)
			if lineIndent(next) <= 4 || !strings.HasPrefix(t, "-") {
				break
			}
			needs = append(needs, cleanScalar(strings.TrimPrefix(t, "-")))
		}
		return needs
	}
	return nil
}

// bypassProblems — écarts d'une expression `if:` de job par rapport au contournement exigé :
// chaque littéral doit y figurer, et PRÉCÉDER la première lecture de `needs.changes.`.
func bypassProblems(ifExpr string) []string {
	if ifExpr == "" {
		return []string{"aucun `if:` : un job qui dépend de `changes` DOIT porter le contournement (PR, main, feat/v75) devant la lecture du filtre"}
	}
	readAt := strings.Index(ifExpr, "needs.changes.")
	var problems []string
	for _, lit := range requiredBypass {
		at := strings.Index(ifExpr, lit)
		switch {
		case at < 0:
			problems = append(problems, fmt.Sprintf("contournement manquant : %s", lit))
		case readAt >= 0 && at > readAt:
			problems = append(problems, fmt.Sprintf("contournement %q placé APRÈS la lecture de needs.changes", lit))
		}
	}
	return problems
}

// pathsFilterProblems — écarts du texte d'un workflow : chaque job dépendant de `changes` porte le
// contournement. Rend aussi la liste des jobs concernés (ancre : un détecteur qui ne voit plus aucun
// job dépendant a perdu sa cible).
func pathsFilterProblems(lines []string) (dependents []string, problems []string) {
	for _, name := range jobTopLevelNames(lines) {
		block, ok := jobBlock(lines, name)
		if !ok {
			continue
		}
		dependsOnChanges := false
		for _, n := range jobNeeds(block) {
			if n == "changes" {
				dependsOnChanges = true
			}
		}
		if !dependsOnChanges {
			continue
		}
		dependents = append(dependents, name)
		for _, p := range bypassProblems(jobLevelScalar(block, "if")) {
			problems = append(problems, fmt.Sprintf("job %q : %s", name, p))
		}
	}
	return dependents, problems
}

// TestCIJobsDependingOnChangesCarryBypass — le contournement main / feat/v75 / PR sur tout job
// qui lit le filtre de chemins.
func TestCIJobsDependingOnChangesCarryBypass(t *testing.T) {
	root := repoRootForWorkflows(t)
	lines := workflowLines(t, root, "ci.yml")

	dependents, problems := pathsFilterProblems(lines)
	if len(dependents) < 5 {
		t.Fatalf("ci.yml : seulement %d job(s) dépendent de `changes` (%v) — le garde-rail a perdu sa cible, ou le filtre a été retiré sans que ce test le soit", len(dependents), dependents)
	}
	for _, p := range problems {
		t.Errorf(`ci.yml : %s

  Sur `+"`main`, `feat/v75` et les PR"+`, tout doit tourner sans condition : un job conditionné par le filtre
  de chemins sans ce contournement serait sauté par un défaut du filtre, et du code non testé
  atteindrait la production.

  CORRECTION : `+"`if: ${{ github.event_name == 'pull_request' || github.ref == 'refs/heads/main' || github.ref == 'refs/heads/feat/v75' || needs.changes.outputs.<pile> == 'true' }}`", p)
	}
}

// TestCIChangesJobPinsPathsFilterBySHA — le job `changes` existe et son action tierce est épinglée
// par SHA de commit (ci.yml ne porte pas de secret : ci_workflows_security_test.go ne le couvre pas).
func TestCIChangesJobPinsPathsFilterBySHA(t *testing.T) {
	root := repoRootForWorkflows(t)
	lines := workflowLines(t, root, "ci.yml")

	block, ok := jobBlock(lines, "changes")
	if !ok {
		t.Fatal("ci.yml : job `changes` absent")
	}
	found := false
	for _, line := range block {
		ref, isUses := usesReference(line)
		if !isUses || !strings.HasPrefix(ref, "dorny/paths-filter@") {
			continue
		}
		found = true
		if !pinnedUsesRe.MatchString(ref) {
			t.Errorf("ci.yml, job changes : « uses: %s » n'est pas épinglé par SHA de 40 caractères (git ls-remote https://github.com/dorny/paths-filter 'refs/tags/v4*')", ref)
		}
	}
	if !found {
		t.Error("ci.yml, job changes : l'action dorny/paths-filter n'y figure plus — le filtre de chemins n'est plus alimenté")
	}
}

// TestCIConcurrencyCancelsStaleRunsExceptOnProtectedBranches — D-5 : un run par branche, les runs
// périmés sont annulés, sauf sur main et feat/v75 (chaque commit y rend son verdict).
func TestCIConcurrencyCancelsStaleRunsExceptOnProtectedBranches(t *testing.T) {
	root := repoRootForWorkflows(t)
	lines := workflowLines(t, root, "ci.yml")
	for _, p := range concurrencyProblems(lines) {
		t.Errorf("ci.yml : %s", p)
	}
}

// concurrencyProblems — écarts du bloc `concurrency:` racine.
func concurrencyProblems(lines []string) []string {
	var block []string
	in := false
	for _, line := range lines {
		if isBlankOrComment(line) {
			continue
		}
		if lineIndent(line) == 0 {
			in = strings.HasPrefix(line, "concurrency:")
			continue
		}
		if in {
			block = append(block, strings.TrimSpace(line))
		}
	}
	if len(block) == 0 {
		return []string{"bloc `concurrency:` racine absent : les runs périmés ne sont plus annulés"}
	}
	var group, cancel string
	for _, l := range block {
		if v, ok := strings.CutPrefix(l, "group:"); ok {
			group = v
		}
		if v, ok := strings.CutPrefix(l, "cancel-in-progress:"); ok {
			cancel = v
		}
	}
	var problems []string
	if !strings.Contains(group, "github.ref") {
		problems = append(problems, "`concurrency.group` ne porte pas github.ref : tous les runs se disputeraient le même groupe")
	}
	// GitHub ne garde qu'un run EN ATTENTE par groupe, même sans annulation du run en cours : sur
	// une branche protégée, le groupe doit être propre au run, sinon un push rapproché annule le run
	// en attente d'un commit dont le déploiement attend le verdict.
	if !strings.Contains(group, "github.run_id") {
		problems = append(problems, "`concurrency.group` ne porte pas github.run_id : sur main et feat/v75, un run en attente serait annulé par le push suivant")
	}
	for _, protected := range []string{"refs/heads/main", "refs/heads/feat/v75"} {
		if !regexp.MustCompile(`github\.ref\s*==\s*'` + regexp.QuoteMeta(protected) + `'`).MatchString(group) {
			problems = append(problems, fmt.Sprintf("`concurrency.group` n'isole pas les runs de %s par github.run_id", protected))
		}
		if !regexp.MustCompile(`github\.ref\s*!=\s*'` + regexp.QuoteMeta(protected) + `'`).MatchString(cancel) {
			problems = append(problems, fmt.Sprintf("`cancel-in-progress` n'exclut pas %s : un run en cours y serait annulé par le push suivant", protected))
		}
	}
	return problems
}

// TestCIPathsFilterAnalysisDetectsDefects — preuve que les analyseurs détectent leur défaut sur un
// texte synthétique et laissent passer le texte conforme.
func TestCIPathsFilterAnalysisDetectsDefects(t *testing.T) {
	const bypass = "github.event_name == 'pull_request' || github.ref == 'refs/heads/main' || github.ref == 'refs/heads/feat/v75'"
	job := func(ifLine string) string {
		text := "jobs:\n  changes:\n    runs-on: x\n  lourd:\n    needs: changes\n"
		if ifLine != "" {
			text += "    " + ifLine + "\n"
		}
		return text + "    steps: []\n  libre:\n    runs-on: x\n"
	}

	t.Run("conforme", func(t *testing.T) {
		deps, problems := pathsFilterProblems(splitWorkflow(job("if: ${{ " + bypass + " || needs.changes.outputs.go == 'true' }}")))
		if len(problems) != 0 || len(deps) != 1 || deps[0] != "lourd" {
			t.Fatalf("deps=%v problems=%v, attendu [lourd] sans écart", deps, problems)
		}
	})
	t.Run("aucun if", func(t *testing.T) {
		if _, problems := pathsFilterProblems(splitWorkflow(job(""))); len(problems) != 1 {
			t.Fatalf("problems=%v, attendu 1", problems)
		}
	})
	t.Run("filtre seul, sans contournement", func(t *testing.T) {
		if _, problems := pathsFilterProblems(splitWorkflow(job("if: ${{ needs.changes.outputs.go == 'true' }}"))); len(problems) != len(requiredBypass) {
			t.Fatalf("problems=%v, attendu %d", problems, len(requiredBypass))
		}
	})
	t.Run("feat/v75 oublié", func(t *testing.T) {
		expr := "if: ${{ github.event_name == 'pull_request' || github.ref == 'refs/heads/main' || needs.changes.outputs.go == 'true' }}"
		if _, problems := pathsFilterProblems(splitWorkflow(job(expr))); len(problems) != 1 {
			t.Fatalf("problems=%v, attendu 1", problems)
		}
	})
	t.Run("contournement après la lecture du filtre", func(t *testing.T) {
		expr := "if: ${{ needs.changes.outputs.go == 'true' || " + bypass + " }}"
		if _, problems := pathsFilterProblems(splitWorkflow(job(expr))); len(problems) != len(requiredBypass) {
			t.Fatalf("problems=%v, attendu %d", problems, len(requiredBypass))
		}
	})
	t.Run("needs en forme flow et en bloc", func(t *testing.T) {
		flow := "jobs:\n  a:\n    needs: [x, changes]\n    steps: []\n"
		block := "jobs:\n  a:\n    needs:\n      - x\n      - changes\n    steps: []\n"
		for name, text := range map[string]string{"flow": flow, "bloc": block} {
			if deps, _ := pathsFilterProblems(splitWorkflow(text)); len(deps) != 1 {
				t.Errorf("needs en forme %s : dépendants=%v, attendu [a]", name, deps)
			}
		}
	})
	t.Run("un job qui ne dépend pas de changes n'est pas jugé", func(t *testing.T) {
		deps, problems := pathsFilterProblems(splitWorkflow("jobs:\n  a:\n    needs: autre\n    steps: []\n"))
		if len(deps) != 0 || len(problems) != 0 {
			t.Fatalf("deps=%v problems=%v, attendu aucun", deps, problems)
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		const groupOK = "${{ github.workflow }}-${{ github.ref }}-${{ (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/feat/v75') && github.run_id || 'travail' }}"
		good := splitWorkflow("concurrency:\n  group: " + groupOK + "\n  cancel-in-progress: ${{ github.ref != 'refs/heads/main' && github.ref != 'refs/heads/feat/v75' }}\njobs:\n  a:\n    steps: []\n")
		if p := concurrencyProblems(good); len(p) != 0 {
			t.Fatalf("bloc conforme refusé : %v", p)
		}
		for name, text := range map[string]string{
			"absent":                  "jobs:\n  a:\n    steps: []\n",
			"sans ref":                "concurrency:\n  group: ci\n  cancel-in-progress: ${{ github.ref != 'refs/heads/main' && github.ref != 'refs/heads/feat/v75' }}\n",
			"annule main":             "concurrency:\n  group: ${{ github.ref }}\n  cancel-in-progress: ${{ github.ref != 'refs/heads/feat/v75' }}\n",
			"annule v75":              "concurrency:\n  group: ${{ github.ref }}\n  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}\n",
			"annule partout":          "concurrency:\n  group: ${{ github.ref }}\n  cancel-in-progress: true\n",
			"groupe partagé sur main": "concurrency:\n  group: ${{ github.workflow }}-${{ github.ref }}\n  cancel-in-progress: ${{ github.ref != 'refs/heads/main' && github.ref != 'refs/heads/feat/v75' }}\n",
		} {
			if p := concurrencyProblems(splitWorkflow(text)); len(p) == 0 {
				t.Errorf("bloc défectueux (%s) non détecté", name)
			}
		}
	})
}
