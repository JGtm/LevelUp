package config

// config_roots.go — la résolution UNIQUE des trois racines (ADR 0008, amendement « two roots »).
//
// Les racines se lisent dans l'ENVIRONNEMENT seul : un `.env.local` ne peut pas poser la racine
// dont il est lu (loadEnvLocal ignore ces quatre clés). Tout le reste du programme reçoit les
// racines par AppConfig.Roots / AppConfig.PathResolver, ou par le pont de
// title.InstallProcessRoots pour les services qui ne reçoivent qu'une racine nue.

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
//   - Logs : LEVELUP_LOGS_DIR, défaut <Resources>/logs ;
//   - FilmArchive : app_settings.json:film_archive_dir (chemin absolu), défaut <Data>/cache.
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
	settingsPath := os.Getenv("LEVELUP_APP_SETTINGS") // même surcharge que loadStatePaths
	if settingsPath == "" {
		settingsPath = filepath.Join(roots.StateFilesDir(), "app_settings.json")
	}
	roots.FilmArchive = filmArchiveFromSettings(settingsPath)
	return roots
}

// SettingFilmArchiveDir : la clé de app_settings.json qui pose l'emplacement de l'archive des
// films (setting du lot L14). Les racines sont résolues avant que le store des réglages n'existe :
// la clé se lit ici dans le JSON brut, une seule fois.
const SettingFilmArchiveDir = "film_archive_dir"

// filmArchiveFromSettings lit `film_archive_dir` dans app_settings.json. Un fichier absent ou sans
// la clé donne "" (disposition par défaut). Une valeur illisible ou non absolue est IGNORÉE avec
// un avertissement : un réglage corrompu ne doit pas envoyer l'archive dans le dossier courant.
func filmArchiveFromSettings(path string) string {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin d'état du programme
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		slog.Warn("film_archive_dir: app_settings.json illisible, archive à l'emplacement par défaut",
			"file", path, "err", err)
		return ""
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		slog.Warn("film_archive_dir: app_settings.json invalide, archive à l'emplacement par défaut",
			"file", path, "err", err)
		return ""
	}
	var dir string
	if v, ok := keys[SettingFilmArchiveDir]; !ok || json.Unmarshal(v, &dir) != nil {
		return ""
	}
	dir = strings.TrimSpace(dir)
	if dir != "" && !filepath.IsAbs(dir) {
		slog.Warn("film_archive_dir: chemin non absolu ignoré, archive à l'emplacement par défaut",
			"file", path, "film_archive_dir", dir)
		return ""
	}
	return dir
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

// InstallProcessRootsFromEnv résout les racines depuis l'environnement (ResolveRoots) et les
// installe comme racines du processus, pour un processus qui n'appelle pas Load : l'enfant de
// cuisson (-film-child), intercepté avant toute configuration. Sans cet appel, l'enfant bâtirait
// ses chemins sous la disposition par défaut de la racine de ressources et écrirait ses faits de
// film sous ResourcesRoot au lieu de DataRoot (amendement ADR 0008, A-5 et A-6).
func InstallProcessRootsFromEnv() title.Roots {
	roots := ResolveRoots()
	title.InstallProcessRoots(roots)
	return roots
}
