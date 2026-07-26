// Package archlint — ci_deploy_triggers_test.go : garde-rail des DÉCLENCHEURS
// GitHub Actions (D29, 2026-07-26).
//
// INVARIANT PROTÉGÉ : tout push qui déclenche le DÉPLOIEMENT doit AUSSI déclencher
// la CI. Depuis D29, deploy.yml n'envoie plus rien en prod avant d'avoir le verdict
// du run CI du MÊME commit (job `attente-ci`, boucle de sondage). Un push qui
// déclencherait le déploiement SANS déclencher la CI condamnerait ce job à sonder
// pendant 55 min un run qui n'existera jamais, puis à échouer : prod non déployée,
// aucun verdict rendu, diagnostic obscur (« pourquoi le deploy attend-il ? »).
//
// L'invariant se traduit en deux conditions vérifiables sur les deux fichiers :
//  1. l'ensemble `paths-ignore` de ci.yml est un SOUS-ENSEMBLE de celui de
//     deploy.yml — un chemin ignoré par la CI doit l'être aussi par le déploiement,
//     sinon il existe un push « déploie mais ne teste pas » ;
//  2. toute branche qui déclenche deploy.yml est couverte par le filtre `branches`
//     de ci.yml (aujourd'hui : main).
//
// PARSING : textuel et ANCRÉ (bloc `on:` → `push:` → clé), pas de dépendance à un
// parseur YAML dans le module go-api. Il tolère les variations d'indentation, les
// quotes simples/doubles, les commentaires et la forme flow `[a, b]` (celle que
// prettier produit pour les listes courtes). Il FAIT ÉCHOUER le test s'il ne trouve
// plus son ancre : un garde-rail qui a perdu sa cible doit rougir, pas se taire.
//
// COUVERTURE DE MOTIFS : volontairement CONSERVATRICE (égalité, `**`, préfixe
// `X/**`). Deux motifs équivalents écrits différemment seront signalés comme non
// couverts — c'est le bon sens de l'erreur : on aligne l'écriture des deux fichiers
// plutôt que d'embarquer un moteur de glob approximatif dans un garde-rail.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCIPathsIgnoreSubsetOfDeployPathsIgnore — condition 1 de l'invariant.
func TestCIPathsIgnoreSubsetOfDeployPathsIgnore(t *testing.T) {
	root := repoRootForWorkflows(t)

	ciIgnore := pushListValue(t, root, "ci.yml", "paths-ignore")
	deployIgnore := pushListValue(t, root, "deploy.yml", "paths-ignore")

	for _, ciPat := range ciIgnore {
		if patternCoveredByAny(ciPat, deployIgnore) {
			continue
		}
		t.Errorf(`paths-ignore de ci.yml NON couvert par deploy.yml : %q

  DEADLOCK CRÉÉ : un push qui ne toucherait que %q ne déclencherait PAS la CI
  (ci.yml l'ignore) mais DÉCLENCHERAIT le déploiement (deploy.yml ne l'ignore pas).
  Le job « attente-ci » de deploy.yml sonderait alors pendant 55 min un run CI qui
  n'existera jamais, puis échouerait : rien n'est déployé et aucun verdict n'est rendu.

  CORRECTION (une des deux) :
    - ajouter %q (ou un motif qui l'englobe, ex. « prefixe/** ») au paths-ignore de
      .github/workflows/deploy.yml — le déploiement ignore alors les mêmes chemins ;
    - ou retirer ce motif du paths-ignore de .github/workflows/ci.yml.

  paths-ignore ci.yml     : %v
  paths-ignore deploy.yml : %v`,
			ciPat, ciPat, ciPat, ciIgnore, deployIgnore)
	}
}

// TestDeployBranchesCoveredByCIBranches — condition 2 de l'invariant. Une branche
// qui déploie sans être testée produit le MÊME deadlock (aucun run CI à attendre).
func TestDeployBranchesCoveredByCIBranches(t *testing.T) {
	root := repoRootForWorkflows(t)

	ciBranches := pushListValue(t, root, "ci.yml", "branches")
	deployBranches := pushListValue(t, root, "deploy.yml", "branches")

	for _, dep := range deployBranches {
		if patternCoveredByAny(dep, ciBranches) {
			continue
		}
		t.Errorf(`branche déclenchant deploy.yml NON couverte par ci.yml : %q

  DEADLOCK CRÉÉ : un push sur cette branche déclencherait le déploiement sans
  qu'aucun run CI n'existe pour ce commit — le job « attente-ci » attendrait en vain.

  CORRECTION : ajouter cette branche (ou un motif l'englobant) au filtre « branches »
  du déclencheur push de .github/workflows/ci.yml.

  branches ci.yml     : %v
  branches deploy.yml : %v`, dep, ciBranches, deployBranches)
	}
}

// patternCoveredByAny — le motif est-il couvert par AU MOINS un motif de la liste ?
func patternCoveredByAny(pattern string, coverers []string) bool {
	for _, c := range coverers {
		if patternCovers(c, pattern) {
			return true
		}
	}
	return false
}

// patternCovers — « tout ce que matche `narrow` est-il matché par `wide` ? ».
// Règles volontairement peu nombreuses et sûres (cf. en-tête du fichier) :
//   - égalité stricte ;
//   - `**` / `**/*` : englobe tout ;
//   - `prefixe/**` : englobe tout motif commençant par `prefixe/`
//     (c'est le cas réel `.ai/**` de deploy.yml qui englobe `.ai/**.md` de ci.yml) ;
//   - `prefixe/*` : idem (moins large en réalité, mais toujours suffisant pour
//     l'usage « ce chemin est-il ignoré/déclenché aussi de l'autre côté »).
func patternCovers(wide, narrow string) bool {
	if wide == narrow {
		return true
	}
	if wide == "**" || wide == "**/*" {
		return true
	}
	for _, suffix := range []string{"/**", "/*"} {
		if prefix, ok := strings.CutSuffix(wide, suffix); ok && prefix != "" {
			if strings.HasPrefix(narrow, prefix+"/") {
				return true
			}
		}
	}
	return false
}

// ─── Lecture des workflows ────────────────────────────────────────────────────

// repoRootForWorkflows — racine du dépôt (celle qui contient .github/workflows/).
// On remonte depuis CE fichier : le test tourne depuis le dossier du package, et
// les workflows vivent DEUX niveaux au-dessus du module go-api.
func repoRootForWorkflows(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué — impossible de localiser la racine du dépôt")
	}
	start := filepath.Dir(thisFile)
	dir := start
	for range 12 {
		if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("racine du dépôt introuvable (.github/workflows/ci.yml) en remontant depuis %s", start)
	return ""
}

// pushListValue — liste YAML sous `on: push: <key>:` du workflow demandé.
func pushListValue(t *testing.T, root, workflow, key string) []string {
	t.Helper()
	lines := workflowLines(t, root, workflow)
	block := pushTriggerBlock(t, lines, workflow)
	return yamlListUnderKey(t, block, key, workflow)
}

func workflowLines(t *testing.T, root, workflow string) []string {
	t.Helper()
	path := filepath.Join(root, ".github", "workflows", workflow)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s impossible : %v (le garde-rail a perdu sa cible)", path, err)
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
}

// pushTriggerBlock — lignes du sous-bloc `push:` du déclencheur `on:`.
// L'ancrage sur `on:` est ce qui évite de confondre la clé de déclencheur avec un
// `push: false` d'un `with:` d'action (cas réel dans test-deploy-precheck.yml).
func pushTriggerBlock(t *testing.T, lines []string, workflow string) []string {
	t.Helper()

	onIdx := -1
	for i, line := range lines {
		if isBlankOrComment(line) || lineIndent(line) != 0 {
			continue
		}
		switch strings.TrimSpace(line) {
		case "on:", `"on":`, "'on':":
			onIdx = i
		}
		if onIdx >= 0 {
			break
		}
	}
	if onIdx < 0 {
		t.Fatalf("%s : déclencheur « on: » introuvable — le garde-rail des déclencheurs a perdu son ancre, le vérifier avant de le modifier", workflow)
	}

	pushIdx, pushIndent := -1, 0
	for i := onIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if isBlankOrComment(line) {
			continue
		}
		indent := lineIndent(line)
		if indent == 0 {
			break // fin du bloc on:
		}
		if strings.TrimSpace(line) == "push:" {
			pushIdx, pushIndent = i, indent
			break
		}
	}
	if pushIdx < 0 {
		t.Fatalf("%s : déclencheur « push: » introuvable sous « on: ». Si le workflow ne se déclenche plus sur push, ce garde-rail doit être revu (et l'invariant D29 réévalué), pas ignoré", workflow)
	}

	var block []string
	for i := pushIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if isBlankOrComment(line) {
			block = append(block, line)
			continue
		}
		if lineIndent(line) <= pushIndent {
			break
		}
		block = append(block, line)
	}
	if len(block) == 0 {
		t.Fatalf("%s : bloc « push: » vide", workflow)
	}
	return block
}

// yamlListUnderKey — items de la liste `key:` dans un bloc, forme bloc (« - x »)
// ou forme flow (« [a, b] », éventuellement étalée sur plusieurs lignes).
func yamlListUnderKey(t *testing.T, block []string, key, workflow string) []string {
	t.Helper()

	keyIdx, keyIndent, inline := -1, 0, ""
	for i, line := range block {
		if isBlankOrComment(line) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != key+":" && !strings.HasPrefix(trimmed, key+": ") {
			continue
		}
		keyIdx = i
		keyIndent = lineIndent(line)
		inline = strings.TrimSpace(strings.TrimPrefix(trimmed, key+":"))
		break
	}
	if keyIdx < 0 {
		t.Fatalf("%s : clé « %s: » introuvable dans le déclencheur push. Le garde-rail des déclencheurs (invariant D29) ne peut plus vérifier ce fichier : corriger le test EN MÊME TEMPS que le workflow, ne pas le neutraliser", workflow, key)
	}

	// Forme flow : sur la ligne de la clé, ou sur la première ligne significative
	// qui suit (style « clé: » puis « [ ... ] » que produit prettier).
	flowStart := inline
	next := keyIdx + 1
	if flowStart == "" {
		for i := keyIdx + 1; i < len(block); i++ {
			if isBlankOrComment(block[i]) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(block[i]), "[") {
				flowStart = strings.TrimSpace(block[i])
				next = i + 1
			}
			break
		}
	}
	if strings.HasPrefix(flowStart, "[") {
		buf := flowStart
		for !strings.Contains(buf, "]") && next < len(block) {
			if !isBlankOrComment(block[next]) {
				buf += " " + strings.TrimSpace(block[next])
			}
			next++
		}
		items := splitFlowSequence(buf)
		if len(items) == 0 {
			t.Fatalf("%s : liste « %s: » (forme flow) vide ou illisible : %q", workflow, key, buf)
		}
		return items
	}
	if inline != "" {
		t.Fatalf("%s : « %s: » porte une valeur scalaire (%q) là où une liste est attendue", workflow, key, inline)
	}

	// Forme bloc : items « - motif » (indentés au moins comme la clé — YAML autorise
	// les deux niveaux).
	var items []string
	for i := keyIdx + 1; i < len(block); i++ {
		line := block[i]
		if isBlankOrComment(line) {
			continue
		}
		if lineIndent(line) < keyIndent {
			break
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			break
		}
		if item := cleanScalar(strings.TrimPrefix(trimmed, "-")); item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		t.Fatalf("%s : liste « %s: » vide — un bloc vide vaut « aucun filtre » et casserait silencieusement l'invariant D29", workflow, key)
	}
	return items
}

// splitFlowSequence — « [a, b, c,] » -> {a, b, c}.
func splitFlowSequence(buf string) []string {
	open := strings.Index(buf, "[")
	closeIdx := strings.LastIndex(buf, "]")
	if open < 0 || closeIdx < open {
		return nil
	}
	var items []string
	for _, raw := range strings.Split(buf[open+1:closeIdx], ",") {
		if item := cleanScalar(raw); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// cleanScalar — retire l'indentation, les quotes et un éventuel commentaire de fin.
func cleanScalar(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if quote := s[0]; quote == '"' || quote == '\'' {
		if end := strings.IndexByte(s[1:], quote); end >= 0 {
			return s[1 : 1+end]
		}
	}
	if idx := strings.Index(s, " #"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	return strings.Trim(s, `"'`)
}

func lineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func isBlankOrComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}
