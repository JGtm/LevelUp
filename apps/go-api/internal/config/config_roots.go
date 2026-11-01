package config

// config_roots.go — la résolution UNIQUE des trois racines (ADR 0008, amendement « two roots »).
//
// Les racines se lisent dans l'ENVIRONNEMENT seul : un `.env.local` ne peut pas poser la racine
// dont il est lu (loadEnvLocal ignore ces quatre clés). Tout le reste du programme reçoit les
// racines par AppConfig.Roots / AppConfig.PathResolver, ou par le pont de
// title.InstallProcessRoots pour les services qui ne reçoivent qu'une racine nue.

import (
	"log/slog"
	"os"

	title "levelup/go-api/internal/domain/title"
)

// Variables d'environnement des racines. LEVELUP_REPO_ROOT reste le nom compatible de
// LEVELUP_RESOURCES_DIR, qui l'emporte quand les deux sont posées.
const (
	EnvResourcesDir = "LEVELUP_RESOURCES_DIR"
	EnvRepoRoot     = "LEVELUP_REPO_ROOT"
	EnvDataDir      = "LEVELUP_DATA_DIR"
	EnvLogsDir      = "LEVELUP_LOGS_DIR"
)

// rootEnvKeys : les clés qu'un `.env.local` n'a pas le droit de poser.
var rootEnvKeys = map[string]bool{
	EnvResourcesDir: true, EnvRepoRoot: true, EnvDataDir: true, EnvLogsDir: true,
}

// ResolveRoots lit l'environnement et rend les trois racines :
//   - Resources : LEVELUP_RESOURCES_DIR, puis LEVELUP_REPO_ROOT, puis la détection (autoDetectRepoRoot) ;
//   - Data : LEVELUP_DATA_DIR, défaut <Resources>/data ;
//   - Logs : LEVELUP_LOGS_DIR, défaut <Resources>/logs.
func ResolveRoots() title.Roots {
	resources := getEnvOrDefault(EnvResourcesDir, getEnvOrDefault(EnvRepoRoot, ""))
	if resources == "" {
		resources = autoDetectRepoRoot()
	}
	roots := title.DefaultRoots(resources)
	if v := os.Getenv(EnvDataDir); v != "" {
		roots.Data, roots.DataSet = v, true
	}
	if v := os.Getenv(EnvLogsDir); v != "" {
		roots.Logs = v
	}
	return roots
}

// Roots rend les trois racines de la configuration. Une AppConfig construite à la main (tests)
// ou dont RepoRoot a été réaffecté après le chargement retombe sur la disposition par défaut
// de RepoRoot.
func (c *AppConfig) Roots() title.Roots {
	if c.roots.Resources != "" && title.SameRoot(c.roots.Resources, c.RepoRoot) {
		return c.roots
	}
	return title.DefaultRoots(c.RepoRoot)
}

// PathResolver rend le résolveur de chemins des trois racines de la configuration.
func (c *AppConfig) PathResolver(registry ...*title.Registry) *title.PathResolver {
	return title.NewPathResolverRoots(c.Roots(), registry...)
}

// skipRootEnvKey dit si loadEnvLocal doit ignorer key ; il journalise l'ignorance.
func skipRootEnvKey(path, key string) bool {
	if !rootEnvKeys[key] {
		return false
	}
	slog.Warn("env_local: variable de racine ignorée (l'environnement du processus seul pose les racines)",
		"file", path, "key", key)
	return true
}
