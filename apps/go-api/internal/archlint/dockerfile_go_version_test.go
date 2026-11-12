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
