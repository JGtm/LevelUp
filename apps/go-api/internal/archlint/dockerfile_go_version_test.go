// Package archlint — dockerfile_go_version_test.go : la version de l'image Go du Dockerfile
// égale la ligne `go` de apps/go-api/go.mod (plan sécurité 2026-10-15, lot L10, item L10-6,
// découverte D-L0-3).
//
// POURQUOI. Le Dockerfile compile les binaires de production dans `golang:X.Y.Z-…`. Une
// image Go fixe GOTOOLCHAIN=local : si elle est plus ancienne que `go.mod`, le build échoue
// (alerte immédiate) ; si elle est plus récente, il passe en silence et la production tourne
// sur une bibliothèque standard que le reste de la chaîne (CI, govulncheck, binaires de
// release) ne teste pas. Le correctif de bibliothèque standard exigé par govulncheck vit dans
// la ligne `go` de go.mod : l'image doit la suivre exactement. Dependabot (écosystème
// `docker`) propose le bump de l'image ; ce test impose de monter go.mod avec.
//
// CE QUI EST EXIGÉ : au moins un `FROM golang:…` ; chaque tag commence par un correctif exact
// `X.Y.Z` (ni `1.26`, ni `latest`, ni `${ARG}`) ; ce `X.Y.Z` est égal à la ligne `go` de
// go.mod. L'analyse est pure et testée sur des textes synthétiques
// (TestDockerfileGoVersionAnalysisDetectsDefects).
package archlint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

var (
	// `FROM [--platform=…] golang:<tag> [AS nom]`, début de ligne.
	dockerfileGolangFromRe = regexp.MustCompile(`(?im)^[ \t]*FROM[ \t]+(?:--\S+[ \t]+)*golang:(\S+)`)
	// Tag épinglé sur un correctif exact : `1.26.9`, `1.26.9-bookworm`.
	dockerfileExactTagRe = regexp.MustCompile(`^(\d+\.\d+\.\d+)(?:-\S+)?$`)
	// Ligne `go X.Y[.Z]` de go.mod.
	goModGoLineRe = regexp.MustCompile(`(?m)^go[ \t]+(\d+\.\d+(?:\.\d+)?)[ \t\r]*$`)
)

// TestDockerfileGoImageMatchesGoMod — l'image Go du Dockerfile suit go.mod.
func TestDockerfileGoImageMatchesGoMod(t *testing.T) {
	root := repoRootForWorkflows(t)
	dockerfile := mustReadRepoFile(t, filepath.Join(root, "Dockerfile"))
	goMod := mustReadRepoFile(t, filepath.Join(root, "apps", "go-api", "go.mod"))
	for _, v := range dockerfileGoVersionViolations(dockerfile, goMod) {
		t.Errorf(`Dockerfile / apps/go-api/go.mod : %s

  CORRECTION : monter ensemble la ligne "go X.Y.Z" de apps/go-api/go.mod et le tag de
  "FROM golang:X.Y.Z-bookworm" du Dockerfile (vérifier que le tag existe sur Docker Hub).`, v)
	}
}

// TestDockerfileGoVersionAnalysisDetectsDefects — l'analyse sait échouer : une divergence
// (image plus récente ou plus ancienne que go.mod) et chaque autre défaut de la classe,
// injectés dans une paire saine, sont signalés.
func TestDockerfileGoVersionAnalysisDetectsDefects(t *testing.T) {
	const healthyDockerfile = "FROM node:22-slim AS web\nFROM golang:1.26.9-bookworm AS go-builder\nFROM debian:bookworm-slim\n"
	const healthyGoMod = "module levelup/go-api\n\ngo 1.26.9\n"
	if got := dockerfileGoVersionViolations(healthyDockerfile, healthyGoMod); len(got) != 0 {
		t.Fatalf("paire saine signalée : %v", got)
	}
	cases := map[string][2]string{
		"image plus récente que go.mod":    {"FROM golang:1.26.10-bookworm\n", healthyGoMod},
		"image plus ancienne que go.mod":   {"FROM golang:1.26.8-bookworm\n", healthyGoMod},
		"go.mod monté sans l'image":        {healthyDockerfile, "go 1.27.0\n"},
		"tag flottant X.Y":                 {"FROM golang:1.26-bookworm\n", healthyGoMod},
		"tag latest":                       {"FROM golang:latest\n", healthyGoMod},
		"tag par argument de build":        {"FROM golang:${GO_VERSION}-bookworm\n", healthyGoMod},
		"aucune image golang (vacuité)":    {"FROM debian:bookworm-slim\n", healthyGoMod},
		"go.mod sans ligne go":             {healthyDockerfile, "module levelup/go-api\n"},
		"seconde image golang en écart":    {healthyDockerfile + "FROM golang:1.25.0-bookworm\n", healthyGoMod},
		"plateforme précisée, en écart":    {"FROM --platform=linux/amd64 golang:1.26.1-bookworm\n", healthyGoMod},
		"go.mod en deux composantes (X.Y)": {healthyDockerfile, "go 1.26\n"},
	}
	for name, c := range cases {
		if got := dockerfileGoVersionViolations(c[0], c[1]); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}
	// Une fin de ligne CRLF et une plateforme précisée, version égale : pas de faux positif.
	crlf := "FROM --platform=linux/amd64 golang:1.26.9-bookworm AS b\r\n"
	if got := dockerfileGoVersionViolations(crlf, "go 1.26.9\r\n"); len(got) != 0 {
		t.Errorf("faux positif sur CRLF / --platform : %v", got)
	}
}

// TestDockerfileImagesAreWatchedByDependabot — `.github/dependabot.yml` surveille les images
// du Dockerfile (écosystème `docker`, répertoire racine) : sans lui, l'épinglage exact de
// l'image Go ne se met à jour qu'à la main (L10-5, D-L0-3).
func TestDockerfileImagesAreWatchedByDependabot(t *testing.T) {
	src := mustReadRepoFile(t, filepath.Join(repoRootForWorkflows(t), ".github", "dependabot.yml"))
	if !dependabotWatchesDockerfile(src) {
		t.Error(`.github/dependabot.yml : aucun bloc package-ecosystem: "docker" avec directory: "/"`)
	}
}

func TestDockerfileDependabotAnalysisDetectsDefects(t *testing.T) {
	healthy := "version: 2\nupdates:\n  - package-ecosystem: docker\n    directory: /\n"
	if !dependabotWatchesDockerfile(healthy) {
		t.Fatal("fichier sain non reconnu")
	}
	for name, doc := range map[string]string{
		"écosystème absent":  "version: 2\nupdates:\n  - package-ecosystem: gomod\n    directory: /\n",
		"mauvais répertoire": "version: 2\nupdates:\n  - package-ecosystem: docker\n    directory: /apps\n",
		"YAML illisible":     "updates: [",
	} {
		if dependabotWatchesDockerfile(doc) {
			t.Errorf("%s : défaut non signalé", name)
		}
	}
}

func dependabotWatchesDockerfile(src string) bool {
	var cfg struct {
		Updates []struct {
			Ecosystem string `yaml:"package-ecosystem"`
			Directory string `yaml:"directory"`
		} `yaml:"updates"`
	}
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return false
	}
	for _, u := range cfg.Updates {
		if u.Ecosystem == "docker" && u.Directory == "/" {
			return true
		}
	}
	return false
}

// dockerfileGoVersionViolations rend les manquements de la paire (Dockerfile, go.mod).
func dockerfileGoVersionViolations(dockerfile, goMod string) []string {
	var out []string
	want := ""
	if m := goModGoLineRe.FindStringSubmatch(goMod); m == nil {
		out = append(out, "go.mod : ligne `go X.Y.Z` introuvable (le garde-rail a perdu sa cible)")
	} else {
		want = m[1]
	}
	froms := dockerfileGolangFromRe.FindAllStringSubmatch(dockerfile, -1)
	if len(froms) == 0 {
		return append(out, "Dockerfile : aucun `FROM golang:…` trouvé (le garde-rail a perdu sa cible)")
	}
	for _, f := range froms {
		tag := f[1]
		m := dockerfileExactTagRe.FindStringSubmatch(tag)
		if m == nil {
			out = append(out, fmt.Sprintf("Dockerfile : `FROM golang:%s` n'est pas épinglé sur un correctif exact X.Y.Z", tag))
			continue
		}
		if want != "" && m[1] != want {
			out = append(out, fmt.Sprintf("Dockerfile : `FROM golang:%s` (Go %s) diffère de la ligne `go %s` de go.mod", tag, m[1], want))
		}
	}
	return out
}

func mustReadRepoFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s impossible : %v (le garde-rail a perdu sa cible)", path, err)
	}
	return string(data)
}

// TestGoVersionIdentiqueDansLEspaceDeTravail — la ligne `go` de `go.work` et de chaque `go.mod`
// de `libs/**` égale celle de `apps/go-api/go.mod` (ADR 0041, plan D1 lot 3).
//
// POURQUOI. L'image du Dockerfile, le job `govulncheck` et le build de release lisent UNE ligne
// `go` (celle d'`apps/go-api/go.mod`, via `go-version-file`). Dans un espace de travail, la ligne
// `go` du `go.work` doit être au moins la plus haute des modules ; une brique qui déclarerait une
// ligne plus basse serait compilée et testée par un toolchain que personne n'a validé pour elle,
// plus haute elle exigerait un toolchain que l'image n'a pas (`GOTOOLCHAIN=local`). Une seule
// valeur partout : la montée de version est un seul geste, et ce test refuse le geste partiel.
func TestGoVersionIdentiqueDansLEspaceDeTravail(t *testing.T) {
	depot := workspaceRoot(t)
	modules, err := filepath.Glob(filepath.Join(depot, "libs", "*", "go.mod"))
	if err != nil || len(modules) == 0 {
		t.Fatalf("aucun libs/*/go.mod trouvé (%v) : le garde-rail a perdu sa cible", err)
	}
	fichiers := map[string]string{
		"apps/go-api/go.mod": mustReadRepoFile(t, filepath.Join(depot, "apps", "go-api", "go.mod")),
		"go.work":            mustReadRepoFile(t, filepath.Join(depot, "go.work")),
	}
	for _, m := range modules {
		rel, _ := filepath.Rel(depot, m)
		fichiers[filepath.ToSlash(rel)] = mustReadRepoFile(t, m)
	}
	for _, v := range goVersionsEspaceDeTravailViolations(fichiers) {
		t.Error(v)
	}
}

// TestGoVersionEspaceDeTravailDetectsDefects — l'analyse sait échouer : une brique en écart, un
// `go.work` en écart et un fichier sans ligne `go` sont signalés ; un jeu cohérent passe.
func TestGoVersionEspaceDeTravailDetectsDefects(t *testing.T) {
	sain := map[string]string{
		"apps/go-api/go.mod":    "module levelup/go-api\n\ngo 1.26.9\n",
		"go.work":               "// commentaire\ngo 1.26.9\n\nuse ./apps/go-api\n",
		"libs/gamefiles/go.mod": "module x/gamefiles\n\ngo 1.26.9\r\n",
	}
	if got := goVersionsEspaceDeTravailViolations(sain); len(got) != 0 {
		t.Fatalf("jeu sain signalé : %v", got)
	}
	for nom, mutation := range map[string]func(map[string]string){
		"brique en écart":       func(m map[string]string) { m["libs/gamefiles/go.mod"] = "go 1.26.8\n" },
		"go.work en écart":      func(m map[string]string) { m["go.work"] = "go 1.27.0\n" },
		"brique sans ligne go":  func(m map[string]string) { m["libs/gamefiles/go.mod"] = "module x\n" },
		"référence sans ligne":  func(m map[string]string) { m["apps/go-api/go.mod"] = "module levelup/go-api\n" },
		"go.work sans ligne go": func(m map[string]string) { m["go.work"] = "use ./apps/go-api\n" },
	} {
		cas := map[string]string{}
		for k, v := range sain {
			cas[k] = v
		}
		mutation(cas)
		if got := goVersionsEspaceDeTravailViolations(cas); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", nom)
		}
	}
}

// goVersionsEspaceDeTravailViolations compare la ligne `go` de chaque fichier à celle de
// `apps/go-api/go.mod` (la référence, lue par `go-version-file` en CI).
func goVersionsEspaceDeTravailViolations(fichiers map[string]string) []string {
	var out []string
	ref := goModGoLineRe.FindStringSubmatch(fichiers["apps/go-api/go.mod"])
	if ref == nil {
		return []string{"apps/go-api/go.mod : ligne `go X.Y.Z` introuvable (le garde-rail a perdu sa cible)"}
	}
	noms := make([]string, 0, len(fichiers))
	for nom := range fichiers {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		m := goModGoLineRe.FindStringSubmatch(fichiers[nom])
		switch {
		case m == nil:
			out = append(out, fmt.Sprintf("%s : ligne `go X.Y.Z` introuvable", nom))
		case m[1] != ref[1]:
			out = append(out, fmt.Sprintf("%s : `go %s` diffère de `go %s` d'apps/go-api/go.mod — "+
				"monter ensemble go.work, chaque libs/*/go.mod et l'image du Dockerfile", nom, m[1], ref[1]))
		}
	}
	return out
}

// TestModulesLibsSurveillesParDependabot — chaque module `libs/*/go.mod` a son bloc `gomod` dans
// `.github/dependabot.yml` : sans lui, ses dépendances externes ne reçoivent jamais de bump
// (Dependabot ne descend pas dans un module qu'on ne lui nomme pas).
func TestModulesLibsSurveillesParDependabot(t *testing.T) {
	depot := workspaceRoot(t)
	modules, err := filepath.Glob(filepath.Join(depot, "libs", "*", "go.mod"))
	if err != nil || len(modules) == 0 {
		t.Fatalf("aucun libs/*/go.mod trouvé (%v) : le garde-rail a perdu sa cible", err)
	}
	src := mustReadRepoFile(t, filepath.Join(depot, ".github", "dependabot.yml"))
	for _, m := range modules {
		rel, _ := filepath.Rel(depot, filepath.Dir(m))
		dir := "/" + filepath.ToSlash(rel)
		if !dependabotWatchesGomod(src, dir) {
			t.Errorf(".github/dependabot.yml : aucun bloc package-ecosystem: \"gomod\" avec directory: %q", dir)
		}
	}
}

func TestDependabotGomodAnalysisDetectsDefects(t *testing.T) {
	sain := "version: 2\nupdates:\n  - package-ecosystem: gomod\n    directory: /libs/x\n"
	if !dependabotWatchesGomod(sain, "/libs/x") {
		t.Fatal("fichier sain non reconnu")
	}
	for nom, doc := range map[string]string{
		"répertoire absent":  "version: 2\nupdates:\n  - package-ecosystem: gomod\n    directory: /apps/go-api\n",
		"mauvais écosystème": "version: 2\nupdates:\n  - package-ecosystem: docker\n    directory: /libs/x\n",
		"YAML illisible":     "updates: [",
	} {
		if dependabotWatchesGomod(doc, "/libs/x") {
			t.Errorf("%s : défaut non signalé", nom)
		}
	}
}

func dependabotWatchesGomod(src, directory string) bool {
	var cfg struct {
		Updates []struct {
			Ecosystem string `yaml:"package-ecosystem"`
			Directory string `yaml:"directory"`
		} `yaml:"updates"`
	}
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		return false
	}
	for _, u := range cfg.Updates {
		if u.Ecosystem == "gomod" && u.Directory == directory {
			return true
		}
	}
	return false
}
