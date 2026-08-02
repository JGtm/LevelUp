// Package config — env_centralization_test.go : garde-rail anti-régression CR A6.
//
// Après D1e, les flags de déploiement suivants ont UNE source de lecture unique :
// config.AppConfig (résolue au boot) ou le helper config.DiscordWebhookURLFromEnv.
// Ce test scanne les sources Go de apps/go-api et échoue si un `os.Getenv` de l'un
// de ces noms réapparaît HORS du package internal/config — ce qui réintroduirait la
// divergence entre deux lecteurs d'un même flag (le défaut CR A6 : le handler
// field-mappings et les résolveurs Discord relisaient l'env en bypassant la
// précédence settings/LEVELUP_*).
//
// Règle contributeur : lire ces flags via cfg.<Champ> (injecté au boot) ou, pour le
// webhook Discord, via config.DiscordWebhookURLFromEnv(). Ne jamais rajouter un
// os.Getenv dispersé. Sans build tag — exécuté en CI normale (grep sur sources).
package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// centralizedEnvReadPattern détecte un os.Getenv d'un flag centralisé au boot.
// L'alternance à `"` fermant distingue LEVELUP_EVENTS_CONVERGENCE de son suffixe
// _MAX (deux flags distincts, tous deux centralisés).
var centralizedEnvReadPattern = regexp.MustCompile(
	// MULTI_TITLE_API_ENABLED est sorti de cette liste le 2026-08-02 : le flag a été
	// SUPPRIMÉ (item 3.3), plus aucun lecteur ne doit exister nulle part — c'est le
	// smoke test internal/api/multi_title_smoke_test.go qui garde cet invariant.
	`Getenv\(\s*"(DISCORD_WEBHOOK_URL|LEVELUP_DISCORD_WEBHOOK_URL|` +
		`LEVELUP_PERSIST_BATCH_ASYNC|LEVELUP_EVENTS_CONVERGENCE|LEVELUP_EVENTS_CONVERGENCE_MAX)"\s*\)`)

// allowedCentralizedEnvReaders : sites AUTORISÉS à lire ces flags via os.Getenv.
// Seul internal/config est légitime (source unique) — exclu par chemin ci-dessous,
// donc ce map reste vide. Toute entrée future doit porter une justification écrite.
var allowedCentralizedEnvReaders = map[string]string{}

func TestEnvCentralization_NoDispersedFlagReaders(t *testing.T) {
	apiRoot := findGoAPIRoot(t)

	var violations []string
	err := filepath.Walk(apiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)

		// internal/config est la source unique légitime : exclue par chemin.
		if strings.HasPrefix(rel, "internal/config/") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !centralizedEnvReadPattern.Match(content) {
			return nil
		}
		if _, ok := allowedCentralizedEnvReaders[rel]; ok {
			return nil
		}
		violations = append(violations,
			"lecture dispersée d'un flag centralisé (CR A6): "+rel+
				" — lire via cfg.<Champ> (boot) ou config.DiscordWebhookURLFromEnv()")
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("REGRESSION CR A6 : %d lecteur(s) d'env dispersé(s) détecté(s) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// findGoAPIRoot remonte depuis le répertoire du test jusqu'à trouver le go.mod du
// module apps/go-api.
func findGoAPIRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 12; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("go.mod (apps/go-api) introuvable en remontant depuis %s", dir)
	return ""
}
