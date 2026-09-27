package config

// config_demo.go — les chemins d'EXÉCUTION du mode démo (backlog 2026-09-26, lot B5 ;
// décision D-7 : « mode démo hermétique côté fichiers »).
//
// En démo, LEVELUP_REPO_ROOT reste le dépôt : on y lit la configuration VERSIONNÉE
// (config/titles/**, data/titles/*/reference/**, static/, .env.local qui porte
// LEVELUP_DEMO_MODE lui-même). Mais tout ce que le serveur lit comme ÉTAT (profils,
// réglages, auth, tokens) ou ÉCRIT en tournant (sessions, logs, caches, état admin) vit sous
// la racine démo, disposée par title.DemoLayout. Hors démo, chaque méthode ci-dessous rend
// EXACTEMENT le chemin d'avant.

import (
	"os"
	"path/filepath"
	"strings"

	title "levelup/go-api/internal/domain/title"
)

// DemoLayout rend la disposition de l'arbre démo (LEVELUP_DEMO_FIXTURES_DIR).
func (c *AppConfig) DemoLayout() title.DemoLayout {
	return title.NewDemoLayout(c.DemoFixturesDir)
}

// RuntimePaths rend le PathResolver des fichiers que le serveur ÉCRIT (ou relit comme
// état) en tournant : jobs.json, cache de l'aide, amis par joueur, état admin, artefacts de
// rejeu, faits de film. Hors démo : le PathResolver du dépôt (inchangé). En démo : un
// PathResolver enraciné sous `<démo>/runtime/`. Jamais pour lire une donnée de référence.
func (c *AppConfig) RuntimePaths() *title.PathResolver {
	if c.DemoMode {
		return c.DemoLayout().RuntimePaths()
	}
	return title.NewPathResolver(c.RepoRoot)
}

// WatcherTokensDir rend le magasin de tokens multi-utilisateur (ADR 0023). En démo : le
// magasin VIDE de la fixture (`<démo>/auth/watcher_tokens`) — la démo ne se connecte à rien
// et ne lit jamais les tokens réels du poste.
func (c *AppConfig) WatcherTokensDir() string {
	if c.DemoMode {
		return c.DemoLayout().WatcherTokensDir()
	}
	return title.NewPathResolver(c.RepoRoot).WatcherTokensDir()
}

// TitleSettingsPath rend l'overlay de réglages d'un titre : celui de la fixture en démo,
// `data/titles/<slug>/settings.json` du dépôt sinon.
func (c *AppConfig) TitleSettingsPath(slug string) string {
	if c.DemoMode {
		return c.DemoLayout().TitleSettingsPath(slug)
	}
	return title.NewPathResolver(c.RepoRoot).TitleSettingsPath(slug)
}

// demoModeEnabledValue : la seule valeur (insensible à la casse) de LEVELUP_DEMO_MODE qui
// active la démo — sémantique d'avant B5, inchangée (pas de "1" ni de "yes").
const demoModeEnabledValue = "true"

// demoModeFromEnv lit LEVELUP_DEMO_MODE (source unique de la lecture, partagée par Load et
// DemoLogsDir).
func demoModeFromEnv() bool {
	return strings.ToLower(getEnvOrDefault("LEVELUP_DEMO_MODE", "false")) == demoModeEnabledValue
}

// DemoLogsDir rend le dossier des logs du mode démo (`<démo>/runtime/logs`) quand le
// processus tourne en démo et que LEVELUP_LOGS_DIR n'est pas posé explicitement. Lu par
// cmd/server AVANT config.Load (le logger est monté le premier) : il relit donc l'env
// lui-même, avec les mêmes défauts que Load. ok=false : aucune redirection.
func DemoLogsDir() (string, bool) {
	if !demoModeFromEnv() || os.Getenv("LEVELUP_LOGS_DIR") != "" {
		return "", false
	}
	repoRoot := getEnvOrDefault("LEVELUP_REPO_ROOT", autoDetectRepoRoot())
	return title.NewDemoLayout(demoFixturesDirFromEnv(repoRoot)).LogsDir(), true
}

// statePaths résout les chemins d'état et d'exécution de Load (sessions, auth, profils,
// réglages, sauvegarde). Hors démo : les défauts du dépôt, inchangés. En démo : la
// disposition démo. Dans les deux cas, une variable d'environnement posée garde la main
// (contrat documenté de docs/CONFIGURATION.md) — même sémantique que getEnvOrDefault.
type statePaths struct {
	demo   bool
	layout title.DemoLayout
}

func newStatePaths(demoMode bool, fixturesDir string) statePaths {
	return statePaths{demo: demoMode, layout: title.NewDemoLayout(fixturesDir)}
}

func (s statePaths) path(envKey, prodDefault string, demoPath func(title.DemoLayout) string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if s.demo {
		return demoPath(s.layout)
	}
	return prodDefault
}

// demoBackupDir : dossier de sauvegarde de la démo, sous `<démo>/runtime/` (la sauvegarde
// est de toute façon refusée en démo, cf. handlers/settings_backup.go).
func demoBackupDir(l title.DemoLayout) string {
	return filepath.Join(l.RuntimeDir(), "backups")
}
