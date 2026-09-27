// config_demo_hermetic_test.go — le mode démo ne place AUCUN chemin d'exécution sous la
// racine du dépôt (backlog 2026-09-26, lot B5.7-1 ; décision D-7).
//
// Le harnais visuel (scripts/demo-visual-harness.sh) lance la démo avec
// LEVELUP_REPO_ROOT = le VRAI checkout. Avant B5, la configuration dérivait sessions, auth,
// db_profiles, app_settings et le reste de cette racine : une démo lue et écrivait les
// fichiers réels du poste. Le test pose un dépôt LEURRE et une racine démo distincte, et
// exige que chaque chemin d'exécution de cfg vive sous la racine démo.
//
// Depuis B-C10 (backlog 2026-09-26), la redirection appartient au SEUL processus serveur :
// ces tests passent par LoadServer ; Load (tout autre binaire) n'en hérite pas
// (config_load_state_test.go).
package config

import (
	"path/filepath"
	"strings"
	"testing"

	title "levelup/go-api/internal/domain/title"
)

// envDemoHermetique : variables qui pilotent les chemins de config.LoadServer. Vidées par défaut
// (getEnvOrDefault traite "" comme absent) pour que l'environnement du poste ne fausse rien.
var envDemoHermetique = []string{
	"LEVELUP_REPO_ROOT", "LEVELUP_DEMO_MODE", "LEVELUP_DEMO_FIXTURES_DIR",
	"LEVELUP_DB_PROFILES", "LEVELUP_APP_SETTINGS", "LEVELUP_SESSION_DIR",
	"LEVELUP_AUTH_DIR", "LEVELUP_BACKUP_DIR", "LEVELUP_LOGS_DIR",
	"LEVELUP_PERSIST_BATCH_ASYNC",
}

func poserEnvDemo(t *testing.T, leurre, demo string, demoMode bool) {
	t.Helper()
	for _, k := range envDemoHermetique {
		t.Setenv(k, "")
	}
	t.Setenv("LEVELUP_REPO_ROOT", leurre)
	t.Setenv("LEVELUP_DEMO_FIXTURES_DIR", demo)
	if demoMode {
		t.Setenv("LEVELUP_DEMO_MODE", "true")
	}
}

// cheminsDExecution : tous les chemins d'exécution que porte (ou rend) une AppConfig.
func cheminsDExecution(t *testing.T, cfg *AppConfig) map[string]string {
	t.Helper()
	rt := cfg.RuntimePaths()
	social, meta := cfg.PrestigeBundleDBPaths()
	out := map[string]string{
		"SessionDir":                     cfg.SessionDir,
		"AuthDir":                        cfg.AuthDir,
		"UsersFilePath":                  cfg.UsersFilePath(),
		"DBProfilesPath":                 cfg.DBProfilesPath,
		"AppSettingsPath":                cfg.AppSettingsPath,
		"Backup.BackupDir":               cfg.Backup.BackupDir,
		"WatcherTokensDir":               cfg.WatcherTokensDir(),
		"RuntimePaths.JobsCachePath":     rt.JobsCachePath(),
		"RuntimePaths.CacheRootDir":      rt.CacheRootDir(),
		"RuntimePaths.PlayerFriendsPath": rt.PlayerFriendsPath(),
		"RuntimePaths.AdminStateDir":     rt.AdminStateDir(),
		"RuntimePaths.GlobalMonitoring":  rt.GlobalMonitoringDB(),
		"RuntimePaths.ReplayArtifacts":   rt.ReplayArtifactsDir(title.DefaultSlug),
		"RuntimePaths.FilmFactsDir":      rt.FilmFactsDir(title.DefaultSlug),
		"TitleSettingsPath(défaut)":      cfg.TitleSettingsPath(title.DefaultSlug),
		"TitleSettingsPath(halo_5)":      cfg.TitleSettingsPath("halo_5"),
		"SharedDBPath(défaut)":           SharedDBPath(cfg, title.DefaultSlug),
		"SharedDBPath(halo_5)":           SharedDBPath(cfg, "halo_5"),
		"MetadataDBPath(défaut)":         MetadataDBPath(cfg, title.DefaultSlug),
		"PrestigeBundle.shared_social":   social,
		"PrestigeBundle.metadata":        meta,
	}
	if dir, ok := DemoLogsDir(); ok {
		out["DemoLogsDir"] = dir
	} else if cfg.DemoMode {
		out["DemoLogsDir"] = "(aucune redirection des logs en démo)"
	}
	return out
}

func sous(racine, chemin string) bool {
	rel, err := filepath.Rel(racine, chemin)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// TestLoadServer_DemoMode_AucunCheminDExecutionSousLeLeurre : en démo, sans variable explicite,
// chaque chemin d'exécution vit sous la racine démo, jamais sous le dépôt leurre.
func TestLoadServer_DemoMode_AucunCheminDExecutionSousLeLeurre(t *testing.T) {
	leurre := t.TempDir()
	demo := t.TempDir()
	poserEnvDemo(t, leurre, demo, true)

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if !cfg.DemoMode {
		t.Fatal("précondition : DemoMode attendu")
	}
	for nom, chemin := range cheminsDExecution(t, cfg) {
		if sous(leurre, chemin) {
			t.Errorf("%s = %s : sous le dépôt LEURRE en mode démo", nom, chemin)
		}
		if !sous(demo, chemin) {
			t.Errorf("%s = %s : hors de la racine démo %s", nom, chemin, demo)
		}
	}
	if cfg.PersistBatchAsync {
		t.Error("PersistBatchAsync = true en démo : la file persist écrirait un WAL et rejouerait RecoverPending")
	}
}

// TestLoadServer_HorsDemo_CheminsInchanges : hors démo, rien ne bouge (défauts sous le dépôt).
func TestLoadServer_HorsDemo_CheminsInchanges(t *testing.T) {
	racine := t.TempDir()
	demo := t.TempDir()
	poserEnvDemo(t, racine, demo, false)

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	pr := title.NewPathResolver(racine)
	attendus := map[string][2]string{
		"SessionDir":        {cfg.SessionDir, filepath.Join(racine, "data", "sessions")},
		"AuthDir":           {cfg.AuthDir, filepath.Join(racine, "data", "auth")},
		"DBProfilesPath":    {cfg.DBProfilesPath, filepath.Join(racine, "db_profiles.json")},
		"AppSettingsPath":   {cfg.AppSettingsPath, filepath.Join(racine, "app_settings.json")},
		"Backup.BackupDir":  {cfg.Backup.BackupDir, filepath.Join(racine, "data", "backups")},
		"WatcherTokensDir":  {cfg.WatcherTokensDir(), pr.WatcherTokensDir()},
		"RuntimePaths.root": {cfg.RuntimePaths().RepoRoot(), racine},
		"TitleSettingsPath": {cfg.TitleSettingsPath("halo_5"), pr.TitleSettingsPath("halo_5")},
	}
	for nom, v := range attendus {
		if v[0] != v[1] {
			t.Errorf("%s = %s, attendu %s (le comportement hors démo ne doit pas changer)", nom, v[0], v[1])
		}
	}
	if !cfg.PersistBatchAsync {
		t.Error("PersistBatchAsync = false hors démo : défaut inchangé attendu (true)")
	}
	if dir, ok := DemoLogsDir(); ok {
		t.Errorf("DemoLogsDir hors démo = %s, attendu aucune redirection", dir)
	}
}

// TestLoadServer_DemoMode_VariableExpliciteGagne : une variable posée explicitement garde la main,
// en démo comme ailleurs (contrat documenté de docs/CONFIGURATION.md).
func TestLoadServer_DemoMode_VariableExpliciteGagne(t *testing.T) {
	leurre := t.TempDir()
	demo := t.TempDir()
	poserEnvDemo(t, leurre, demo, true)
	explicite := filepath.Join(t.TempDir(), "sessions-explicites")
	t.Setenv("LEVELUP_SESSION_DIR", explicite)

	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer: %v", err)
	}
	if cfg.SessionDir != explicite {
		t.Errorf("SessionDir = %s, attendu la variable explicite %s", cfg.SessionDir, explicite)
	}
}
