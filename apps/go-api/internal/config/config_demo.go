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

// Load charge la configuration de TOUT binaire autre que le serveur : outils opérateurs
// (cmd/levelup, cmd/token-capture, cmd/restore…) et tests. Chemins d'état du DÉPÔT, que
// LEVELUP_DEMO_MODE soit posé ou non : ces binaires opèrent sur les vraies données (seed-demo
// PRODUIT la démo à partir des vrais profils), et la CI lance toute la suite Go avec
// LEVELUP_DEMO_MODE=true. DemoMode garde la valeur de l'environnement (sémantique d'avant B5).
// Lots B-C9 puis B-C10 du backlog 2026-09-26.
func Load() (*AppConfig, error) {
	cfg, err := load(true)
	if cfg != nil {
		cfg.stateFromRepo = true
	}
	return cfg, err
}

// LoadServer charge la configuration du processus SERVEUR (cmd/server, et lui seul). En
// démo, sans variable explicite, les chemins d'état et d'exécution suivent la disposition
// démo et la file persist asynchrone est coupée (lot B5, D-7) : la redirection appartient au
// seul serveur démo. Tout ce qui tourne dans le serveur reçoit ce cfg par injection.
func LoadServer() (*AppConfig, error) { return load(false) }

// demoState dit si les chemins d'état et d'exécution suivent la disposition démo : en démo,
// sauf pour une configuration chargée par Load (binaire autre que le serveur). Une AppConfig
// construite à la main (tests) avec DemoMode suit la disposition, comme le serveur.
func (c *AppConfig) demoState() bool { return c.DemoMode && !c.stateFromRepo }

// DemoLayout rend la disposition de l'arbre démo (LEVELUP_DEMO_FIXTURES_DIR).
func (c *AppConfig) DemoLayout() title.DemoLayout {
	return title.NewDemoLayout(c.DemoFixturesDir)
}

// RuntimePaths rend le PathResolver des fichiers que le serveur ÉCRIT (ou relit comme
// état) en tournant : jobs.json, cache de l'aide, amis par joueur, état admin, artefacts de
// rejeu, faits de film. Hors démo : le PathResolver du dépôt (inchangé). En démo : un
// PathResolver enraciné sous `<démo>/runtime/`. Jamais pour lire une donnée de référence.
func (c *AppConfig) RuntimePaths() *title.PathResolver {
	if c.demoState() {
		return c.DemoLayout().RuntimePaths()
	}
	return title.NewPathResolver(c.RepoRoot)
}

// WatcherTokensDir rend le magasin de tokens multi-utilisateur (ADR 0023). En démo : le
// magasin VIDE de la fixture (`<démo>/auth/watcher_tokens`) — la démo ne se connecte à rien
// et ne lit jamais les tokens réels du poste.
func (c *AppConfig) WatcherTokensDir() string {
	if c.demoState() {
		return c.DemoLayout().WatcherTokensDir()
	}
	return title.NewPathResolver(c.RepoRoot).WatcherTokensDir()
}

// TitleSettingsPath rend l'overlay de réglages d'un titre : celui de la fixture en démo,
// `data/titles/<slug>/settings.json` du dépôt sinon.
func (c *AppConfig) TitleSettingsPath(slug string) string {
	if c.demoState() {
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
	demo        bool // les chemins suivent la disposition démo
	demoMode    bool // LEVELUP_DEMO_MODE, tel que l'environnement le pose
	fixturesDir string
	layout      title.DemoLayout
}

// loadStatePaths lit le mode démo (LEVELUP_DEMO_MODE) et la racine démo
// (LEVELUP_DEMO_FIXTURES_DIR) de l'environnement, pour load. fromRepo (Load, hors serveur) : aucune
// redirection démo des chemins d'état.
func loadStatePaths(repoRoot string, fromRepo bool) statePaths {
	fixturesDir := demoFixturesDirFromEnv(repoRoot)
	demoMode := demoModeFromEnv()
	return statePaths{demo: demoMode && !fromRepo, demoMode: demoMode, fixturesDir: fixturesDir,
		layout: title.NewDemoLayout(fixturesDir)}
}

// demoFixturesDirFromEnv lit LEVELUP_DEMO_FIXTURES_DIR (défaut `<repoRoot>/data/demo`).
func demoFixturesDirFromEnv(repoRoot string) string {
	return getEnvOrDefault("LEVELUP_DEMO_FIXTURES_DIR", filepath.Join(repoRoot, "data", "demo"))
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

// backupConfig lit la configuration de sauvegarde (loadBackupConfig) ; en démo, sans
// LEVELUP_BACKUP_DIR explicite, le dossier de sauvegarde va sous `<démo>/runtime/`.
func (s statePaths) backupConfig(repoRoot, appSettingsPath string) BackupConfig {
	b := loadBackupConfig(repoRoot, appSettingsPath)
	b.BackupDir = s.path("LEVELUP_BACKUP_DIR", b.BackupDir, demoBackupDir)
	return b
}

// persistBatchAsync : file persist asynchrone (LEVELUP_PERSIST_BATCH_ASYNC, active sauf "0").
// COUPÉE en démo (B5, D-7) : pas de WAL, pas de RecoverPending.
func (s statePaths) persistBatchAsync() bool {
	return !s.demo && getEnvOrDefault("LEVELUP_PERSIST_BATCH_ASYNC", "") != "0"
}
