// Package archlint — ci_workflows_security_test.go : garde-rails de la chaîne CI/CD
// (registre d'audit sécurité 2026-10-15, constats C9-2 et C9-3).
//
// TROIS CLASSES DE DÉFAUT, UN TEST CHACUNE :
//
//  1. Une action tierce référencée par un tag mobile (`@v1`) dans un workflow qui porte
//     un secret ou une permission d'écriture : un tag amont déplacé ou compromis exécute
//     son code avec ce secret. Dans un tel workflow, tout `uses:` hors `actions/` est
//     épinglé par SHA de commit (40 hexadécimaux) — Dependabot `github-actions` maintient
//     le SHA à partir du commentaire de version.
//  2. L'absence de contrôle bloquant sur les vulnérabilités de dépendances : `ci.yml`
//     porte un job `govulncheck` et un job `npm-audit`, inconditionnels, sans échappatoire
//     (`|| true`, `continue-on-error`). Le hook local reste indicatif ; la CI est l'autorité.
//  3. Un `contents: write` accordé à tout un workflow : l'écriture se déclare sur le seul
//     job qui publie.
//
// PARSING : textuel et ANCRÉ, sans parseur YAML (même parti que ci_deploy_triggers_test.go,
// dont les helpers sont réutilisés). Un garde-rail qui perd sa cible rougit : les tests
// exigent que deploy.yml et release.yml soient reconnus comme sensibles et que les jobs
// attendus existent. Les fonctions d'analyse sont pures et testées sur des textes
// synthétiques (TestWorkflowSecurityAnalysisDetectsDefects) : elles prouvent qu'elles
// savent échouer indépendamment de l'état du dépôt.
package archlint

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// Référence d'action épinglée : `owner/repo[/chemin]@<sha de 40 hexadécimaux>`.
	pinnedUsesRe = regexp.MustCompile(`^[^@\s]+@[0-9a-f]{40}$`)
	// `contents: write` (au niveau workflow ou job).
	contentsWriteRe = regexp.MustCompile(`^contents:\s*write\b`)
	// Lecture d'un secret dans une expression `${{ secrets.X }}`.
	secretsReadRe = regexp.MustCompile(`\bsecrets\.[A-Za-z_]`)
	// Installation de govulncheck à une version exacte.
	govulncheckInstallRe = regexp.MustCompile(`golang\.org/x/vuln/cmd/govulncheck@v\d+\.\d+\.\d+\b`)
	// Exécution de govulncheck sur tout le module (le binaire peut être appelé par chemin).
	govulncheckRunRe = regexp.MustCompile(`govulncheck"?\s+\./\.\.\.`)
)

// jobEscapeHatches — marques qui rendent un job non bloquant ou conditionnel.
var jobEscapeHatches = []string{"|| true", "|| exit 0", "continue-on-error"}

// ─── Classe 1 : actions tierces épinglées dans les workflows sensibles ───────

// TestSensitiveWorkflowsPinThirdPartyActionsBySHA — un workflow qui lit `secrets.` ou
// accorde `contents: write` n'utilise que des actions `actions/*`, locales, ou épinglées
// par SHA de 40 caractères.
func TestSensitiveWorkflowsPinThirdPartyActionsBySHA(t *testing.T) {
	root := repoRootForWorkflows(t)
	files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun workflow trouvé sous .github/workflows (err=%v) : le garde-rail a perdu sa cible", err)
	}

	sensitive := map[string]bool{}
	for _, file := range files {
		name := filepath.Base(file)
		lines := workflowLines(t, root, name)
		isSensitive, violations := unpinnedUsesInSensitiveWorkflow(lines)
		sensitive[name] = isSensitive
		for _, v := range violations {
			t.Errorf(`%s : %s

  Un tag mobile d'action tierce dans un workflow qui porte un secret ou "contents: write"
  permet à l'amont (déplacement ou compromission du tag) d'exécuter son code avec ce secret.

  CORRECTION : épingler par SHA de commit, version en commentaire :
      uses: owner/action@<sha de 40 caractères> # vX.Y.Z
  (git ls-remote https://github.com/owner/action 'refs/tags/vX*' ; si une ligne se termine
  par ^{}, c'est le SHA de cette ligne qui est le commit.)`, name, v)
		}
	}

	// Ancres : ces deux workflows portent les clés SSH de production et la publication
	// de release. Qu'ils cessent d'être reconnus comme sensibles = le détecteur est cassé.
	for _, anchor := range []string{"deploy.yml", "release.yml"} {
		if !sensitive[anchor] {
			t.Errorf("%s n'est plus reconnu comme workflow sensible (secrets. ou contents: write) : "+
				"le détecteur a perdu son ancre, le corriger avant de continuer", anchor)
		}
	}
}

// unpinnedUsesInSensitiveWorkflow — (le workflow est-il sensible ?, les `uses:` tiers non
// épinglés avec leur numéro de ligne). Un workflow non sensible ne produit aucune violation.
func unpinnedUsesInSensitiveWorkflow(lines []string) (bool, []string) {
	sensitive := false
	for _, line := range lines {
		if isBlankOrComment(line) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if secretsReadRe.MatchString(trimmed) || contentsWriteRe.MatchString(trimmed) {
			sensitive = true
			break
		}
	}
	if !sensitive {
		return false, nil
	}

	var violations []string
	for i, line := range lines {
		ref, ok := usesReference(line)
		if !ok || isFirstPartyOrLocalUses(ref) || pinnedUsesRe.MatchString(ref) {
			continue
		}
		violations = append(violations, fmt.Sprintf("ligne %d : « uses: %s » n'est pas épinglé par SHA de 40 caractères", i+1, ref))
	}
	return true, violations
}

// usesReference — valeur d'une ligne `uses:` (ou `- uses:`), hors commentaire de fin.
func usesReference(line string) (string, bool) {
	if isBlankOrComment(line) {
		return "", false
	}
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
	value, found := strings.CutPrefix(trimmed, "uses:")
	if !found {
		return "", false
	}
	ref := cleanScalar(value)
	return ref, ref != ""
}

// isFirstPartyOrLocalUses — actions GitHub officielles (`actions/*`) et actions locales
// du dépôt (`./…`) : hors périmètre de l'épinglage.
func isFirstPartyOrLocalUses(ref string) bool {
	return strings.HasPrefix(ref, "actions/") || strings.HasPrefix(ref, "./")
}

// ─── Classe 2 : jobs de vulnérabilités bloquants dans ci.yml ────────────────

// TestCIHasBlockingVulnerabilityJobs — ci.yml porte les jobs `govulncheck` et
// `npm-audit`, bloquants (aucune échappatoire) et inconditionnels.
func TestCIHasBlockingVulnerabilityJobs(t *testing.T) {
	root := repoRootForWorkflows(t)
	lines := workflowLines(t, root, "ci.yml")

	for _, spec := range []blockingJobSpec{
		{
			job: "govulncheck",
			mustMatch: map[string]*regexp.Regexp{
				"installation de govulncheck à une version exacte (go install golang.org/x/vuln/cmd/govulncheck@vX.Y.Z)": govulncheckInstallRe,
				"exécution de govulncheck sur tout le module (govulncheck ./...)":                                        govulncheckRunRe,
			},
		},
		{
			job: "npm-audit",
			mustMatch: map[string]*regexp.Regexp{
				"npm audit --omit=dev --audit-level=high": regexp.MustCompile(`npm audit --omit=dev --audit-level=high\b`),
			},
		},
	} {
		for _, problem := range blockingJobProblems(lines, spec) {
			t.Errorf("ci.yml, job %q : %s\n\n  Les contrôles de vulnérabilités de dépendances doivent bloquer le merge : ni échappatoire (%s), ni condition `if:`.",
				spec.job, problem, strings.Join(jobEscapeHatches, ", "))
		}
	}
}

// blockingJobSpec — ce qu'un job bloquant doit contenir.
type blockingJobSpec struct {
	job       string
	mustMatch map[string]*regexp.Regexp // description -> motif exigé dans le job
}

// blockingJobProblems — écarts du job `spec.job` par rapport à un contrôle bloquant.
func blockingJobProblems(lines []string, spec blockingJobSpec) []string {
	block, ok := jobBlock(lines, spec.job)
	if !ok {
		return []string{"job absent"}
	}

	var problems []string
	body := make([]string, 0, len(block))
	for _, line := range block {
		if !isBlankOrComment(line) {
			body = append(body, strings.TrimSpace(line))
		}
	}
	for _, line := range body {
		for _, hatch := range jobEscapeHatches {
			if strings.Contains(line, hatch) {
				problems = append(problems, fmt.Sprintf("échappatoire %q : %s", hatch, line))
			}
		}
		if strings.HasPrefix(line, "if:") || strings.HasPrefix(line, "- if:") {
			problems = append(problems, fmt.Sprintf("condition sur le job ou une étape : %s", line))
		}
	}
	joined := strings.Join(body, "\n")
	for desc, re := range spec.mustMatch {
		if !re.MatchString(joined) {
			problems = append(problems, "manque : "+desc)
		}
	}
	return problems
}

// jobBlock — lignes du job `name` sous `jobs:` (clé à 2 espaces d'indentation), jusqu'au
// job suivant. Faux si le job n'existe pas.
func jobBlock(lines []string, name string) ([]string, bool) {
	inJobs, start := false, -1
	for i, line := range lines {
		if isBlankOrComment(line) {
			continue
		}
		indent := lineIndent(line)
		if indent == 0 {
			inJobs = strings.TrimSpace(line) == "jobs:"
			continue
		}
		if inJobs && indent == 2 && strings.TrimSpace(line) == name+":" {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}
	block := []string{lines[start]}
	for _, line := range lines[start+1:] {
		if !isBlankOrComment(line) && lineIndent(line) <= 2 {
			break
		}
		block = append(block, line)
	}
	return block, true
}

// ─── Classe 3 : pas de `contents: write` au niveau workflow ─────────────────

// TestNoWorkflowGrantsContentsWriteAtWorkflowLevel — `contents: write` ne se déclare que
// sur le job qui publie, jamais sur le bloc `permissions:` racine d'un workflow.
func TestNoWorkflowGrantsContentsWriteAtWorkflowLevel(t *testing.T) {
	root := repoRootForWorkflows(t)
	files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun workflow trouvé sous .github/workflows (err=%v)", err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		for _, v := range workflowLevelContentsWrite(workflowLines(t, root, name)) {
			t.Errorf("%s : %s\n\n  CORRECTION : permissions racine `contents: read`, et `contents: write` sur le seul job de publication.", name, v)
		}
	}

	// Ancre : release.yml publie une release, donc doit déclarer l'écriture quelque part.
	if !workflowDeclaresContentsWrite(workflowLines(t, root, "release.yml")) {
		t.Error("release.yml ne déclare plus `contents: write` nulle part : la publication de release échouerait, ou le détecteur a perdu son ancre")
	}
}

// workflowLevelContentsWrite — violations d'un bloc `permissions:` racine (indentation 0).
func workflowLevelContentsWrite(lines []string) []string {
	var violations []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isBlankOrComment(line) || lineIndent(line) != 0 || !strings.HasPrefix(line, "permissions:") {
			continue
		}
		if inline := cleanScalar(strings.TrimPrefix(line, "permissions:")); inline == "write-all" {
			violations = append(violations, fmt.Sprintf("ligne %d : `permissions: write-all` au niveau workflow", i+1))
		}
		for j := i + 1; j < len(lines); j++ {
			if isBlankOrComment(lines[j]) {
				continue
			}
			if lineIndent(lines[j]) == 0 {
				break
			}
			if contentsWriteRe.MatchString(strings.TrimSpace(lines[j])) {
				violations = append(violations, fmt.Sprintf("ligne %d : `contents: write` au niveau workflow", j+1))
			}
		}
	}
	return violations
}

// workflowDeclaresContentsWrite — `contents: write` apparaît-il (à un niveau quelconque) ?
func workflowDeclaresContentsWrite(lines []string) bool {
	for _, line := range lines {
		if !isBlankOrComment(line) && contentsWriteRe.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

// ─── Les analyseurs savent échouer ──────────────────────────────────────────

// TestWorkflowSecurityAnalysisDetectsDefects — preuve que les trois analyseurs détectent
// leur défaut sur un texte synthétique et laissent passer le texte conforme.
func TestWorkflowSecurityAnalysisDetectsDefects(t *testing.T) {
	const sha = "0ff4204d59e8e51228ff73bce53f80d53301dee2"

	t.Run("tag mobile dans un workflow à secret", func(t *testing.T) {
		lines := splitWorkflow("jobs:\n  d:\n    steps:\n      - uses: appleboy/ssh-action@v1\n        with:\n          key: ${{ secrets.K }}\n")
		sensitive, violations := unpinnedUsesInSensitiveWorkflow(lines)
		if !sensitive || len(violations) != 1 {
			t.Fatalf("sensitive=%v violations=%v, attendu sensible avec 1 violation", sensitive, violations)
		}
	})
	t.Run("tag mobile dans un workflow à contents: write", func(t *testing.T) {
		lines := splitWorkflow("jobs:\n  p:\n    permissions:\n      contents: write\n    steps:\n      - uses: softprops/action-gh-release@v2\n")
		if _, violations := unpinnedUsesInSensitiveWorkflow(lines); len(violations) != 1 {
			t.Fatalf("violations=%v, attendu 1", violations)
		}
	})
	t.Run("SHA court ou branche refusés", func(t *testing.T) {
		for _, ref := range []string{"o/a@abc1234", "o/a@main", "o/a@" + strings.ToUpper(sha), "o/a@" + sha[:39]} {
			lines := splitWorkflow("x: ${{ secrets.K }}\n- uses: " + ref + "\n")
			if _, violations := unpinnedUsesInSensitiveWorkflow(lines); len(violations) != 1 {
				t.Errorf("%s : violations=%v, attendu 1", ref, violations)
			}
		}
	})
	t.Run("conforme : SHA, actions/, locale, workflow non sensible", func(t *testing.T) {
		ok := splitWorkflow("x: ${{ secrets.K }}\n- uses: appleboy/ssh-action@" + sha + " # v1.2.5\n- uses: actions/checkout@v6\n- uses: ./.github/actions/local\n")
		if _, violations := unpinnedUsesInSensitiveWorkflow(ok); len(violations) != 0 {
			t.Fatalf("violations=%v, attendu aucune", violations)
		}
		plain := splitWorkflow("permissions:\n  contents: read\njobs:\n  a:\n    steps:\n      - uses: golangci/golangci-lint-action@v9\n")
		if sensitive, violations := unpinnedUsesInSensitiveWorkflow(plain); sensitive || len(violations) != 0 {
			t.Fatalf("sensitive=%v violations=%v, attendu non sensible", sensitive, violations)
		}
	})
	t.Run("commentaire seul ne rend pas sensible", func(t *testing.T) {
		lines := splitWorkflow("# lit secrets.K et contents: write\n- uses: o/a@v1\n")
		if sensitive, _ := unpinnedUsesInSensitiveWorkflow(lines); sensitive {
			t.Fatal("un commentaire ne doit pas rendre le workflow sensible")
		}
	})

	t.Run("job bloquant : échappatoires, condition, absence", func(t *testing.T) {
		spec := blockingJobSpec{job: "govulncheck", mustMatch: map[string]*regexp.Regexp{"run": govulncheckRunRe, "install": govulncheckInstallRe}}
		good := "jobs:\n  govulncheck:\n    steps:\n      # note : pas de || true\n      - run: go install golang.org/x/vuln/cmd/govulncheck@v1.9.0\n      - run: govulncheck ./...\n  autre:\n    steps: []\n"
		if p := blockingJobProblems(splitWorkflow(good), spec); len(p) != 0 {
			t.Fatalf("job conforme refusé : %v", p)
		}
		for name, mutate := range map[string]func(string) string{
			"|| true": func(s string) string { return strings.Replace(s, "govulncheck ./...", "govulncheck ./... || true", 1) },
			"continue-on-error": func(s string) string {
				return strings.Replace(s, "    steps:", "    continue-on-error: true\n    steps:", 1)
			},
			"if":               func(s string) string { return strings.Replace(s, "    steps:", "    if: false\n    steps:", 1) },
			"install flottant": func(s string) string { return strings.Replace(s, "@v1.9.0", "@latest", 1) },
			"run absent":       func(s string) string { return strings.Replace(s, "govulncheck ./...", "echo skip", 1) },
		} {
			if p := blockingJobProblems(splitWorkflow(mutate(good)), spec); len(p) == 0 {
				t.Errorf("mutation %q non détectée", name)
			}
		}
		if p := blockingJobProblems(splitWorkflow("jobs:\n  autre:\n    steps: []\n"), spec); len(p) == 0 {
			t.Error("job absent non détecté")
		}
	})

	t.Run("contents: write au niveau workflow", func(t *testing.T) {
		bad := splitWorkflow("permissions:\n  contents: write\njobs:\n  a:\n    steps: []\n")
		if v := workflowLevelContentsWrite(bad); len(v) != 1 {
			t.Fatalf("violations=%v, attendu 1", v)
		}
		if v := workflowLevelContentsWrite(splitWorkflow("permissions: write-all\n")); len(v) != 1 {
			t.Fatalf("write-all : violations=%v, attendu 1", v)
		}
		good := splitWorkflow("permissions:\n  contents: read\njobs:\n  p:\n    permissions:\n      contents: write\n    steps: []\n")
		if v := workflowLevelContentsWrite(good); len(v) != 0 {
			t.Fatalf("écriture sur le job seul refusée : %v", v)
		}
	})
}

func splitWorkflow(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}
