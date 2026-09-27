// demo_paths_test.go — ratchet : le mode démo sert la FIXTURE et n'écrit que sous sa racine,
// quoi qu'il arrive.
//
// Régression 2026-08-05 : la bascule vers la fixture était conditionnée à
// l'absence de la DB de production. Sur un poste ayant ses données réelles — donc
// tout poste de développement — la démo servait la production, `demo-player`
// n'y avait aucun match, et le harnais visuel skippait 6 pages sur 7 en annonçant
// « données absentes » (faux vert : zéro diff sur des pages vides). Le lancement
// de démo appliquait en prime les migrations de boot aux bases de production.
//
// Étendu le 2026-09-27 (backlog, lot B5.7-3) à TOUS les chemins de boot et de cfg : le
// harnais visuel lance la démo avec LEVELUP_REPO_ROOT = le vrai checkout, et sessions,
// logs, état admin, tokens… se lisaient et s'écrivaient encore sous ce checkout.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
)

// envCheminsDeBoot : variables qui pilotent les chemins de boot. Vidées (getEnvOrDefault
// traite "" comme absent) pour que l'environnement du poste ne fausse rien.
var envCheminsDeBoot = []string{
	"LEVELUP_REPO_ROOT", "LEVELUP_DEMO_MODE", "LEVELUP_DEMO_FIXTURES_DIR",
	"LEVELUP_DB_PROFILES", "LEVELUP_APP_SETTINGS", "LEVELUP_SESSION_DIR",
	"LEVELUP_AUTH_DIR", "LEVELUP_BACKUP_DIR", "LEVELUP_LOGS_DIR", "LEVELUP_LOGS_ENABLED",
	"LEVELUP_PERSIST_BATCH_ASYNC",
}

// chargerCfg charge une AppConfig sur un dépôt `depot` et une racine démo `demo`.
func chargerCfg(t *testing.T, depot, demo string, demoMode bool) *config.AppConfig {
	t.Helper()
	for _, k := range envCheminsDeBoot {
		t.Setenv(k, "")
	}
	t.Setenv("LEVELUP_REPO_ROOT", depot)
	t.Setenv("LEVELUP_DEMO_FIXTURES_DIR", demo)
	if demoMode {
		t.Setenv("LEVELUP_DEMO_MODE", "true")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

func estSous(racine, chemin string) bool {
	rel, err := filepath.Rel(racine, chemin)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func TestDemoBootDBPaths_IgnoresProductionDBPresence(t *testing.T) {
	depot := t.TempDir()
	demo := t.TempDir()
	cfg := chargerCfg(t, depot, demo, true)
	pr := title.NewPathResolver(depot)

	prodShared := pr.SharedDBPath(title.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(prodShared), 0o755); err != nil {
		t.Fatalf("mkdir prod: %v", err)
	}

	// Cas 1 : DB de production ABSENTE.
	absent := resolveBootDBPaths(cfg, pr)

	// Cas 2 : DB de production PRÉSENTE — c'est le cas qui régressait.
	if err := os.WriteFile(prodShared, []byte("donnees de production"), 0o600); err != nil {
		t.Fatalf("write prod db: %v", err)
	}
	present := resolveBootDBPaths(cfg, pr)

	if absent != present {
		t.Fatalf("la résolution démo dépend de l'existence de la DB de production :\n  absente = %+v\n  présente = %+v",
			absent, present)
	}
	want := filepath.Join(demo, "warehouse", "shared_matches_v2.duckdb")
	if present.shared != want {
		t.Errorf("chemin démo attendu %s, obtenu %s", want, present.shared)
	}
	if estSous(depot, present.shared) {
		t.Errorf("le mode démo pointe l'arborescence de PRODUCTION : %s", present.shared)
	}
}

// TestDemoBootDBPaths_AllWarehouseDBs : les quatre bases du warehouse doivent être
// traduites. `shared_social` et `shared_pve` étaient absents de la liste d'origine — la
// démo lisait et MIGRAIT donc les bases sociales et PvE de production même quand
// shared/metadata étaient correctement redirigés.
func TestDemoBootDBPaths_AllWarehouseDBs(t *testing.T) {
	depot := t.TempDir()
	demo := t.TempDir()
	cfg := chargerCfg(t, depot, demo, true)

	got := resolveBootDBPaths(cfg, title.NewPathResolver(depot))
	for nom, v := range map[string][2]string{
		"shared_matches_v2.duckdb": {got.shared, "shared_matches_v2.duckdb"},
		"metadata.duckdb":          {got.metadata, "metadata.duckdb"},
		"shared_social.duckdb":     {got.sharedSocial, "shared_social.duckdb"},
		"shared_pve.duckdb":        {got.pve, "shared_pve.duckdb"},
	} {
		if want := filepath.Join(demo, "warehouse", v[1]); v[0] != want {
			t.Errorf("%s : attendu %s, obtenu %s", nom, want, v[0])
		}
	}
}

// cheminsDeBootEtDeCfg : chaque chemin que le boot du serveur lit comme état ou écrit.
func cheminsDeBootEtDeCfg(t *testing.T, cfg *config.AppConfig, depot string) map[string]string {
	t.Helper()
	db := resolveBootDBPaths(cfg, title.NewPathResolver(depot))
	rt := cfg.RuntimePaths()
	return map[string]string{
		"db.shared":                       db.shared,
		"db.metadata":                     db.metadata,
		"db.shared_social":                db.sharedSocial,
		"db.shared_pve":                   db.pve,
		"fixture joueur":                  demoFixturePlayerDBPath(cfg),
		"logs":                            bootLogsConfig(depot).LogsDir,
		"cfg.SessionDir":                  cfg.SessionDir,
		"cfg.AuthDir":                     cfg.AuthDir,
		"cfg.UsersFilePath":               cfg.UsersFilePath(),
		"cfg.DBProfilesPath":              cfg.DBProfilesPath,
		"cfg.AppSettingsPath":             cfg.AppSettingsPath,
		"cfg.Backup.BackupDir":            cfg.Backup.BackupDir,
		"cfg.WatcherTokensDir":            cfg.WatcherTokensDir(),
		"cfg.TitleSettingsPath":           cfg.TitleSettingsPath(title.DefaultSlug),
		"runtime.PostSyncSnapshotPath":    rt.PostSyncSnapshotPath(),
		"runtime.ActionJournalPath":       rt.ActionJournalPath(),
		"runtime.DiskWatchStatePath":      rt.DiskWatchStatePath(),
		"runtime.PlayerFriendsPath":       rt.PlayerFriendsPath(),
		"runtime.JobsCachePath":           rt.JobsCachePath(),
		"runtime.CacheRootDir":            rt.CacheRootDir(),
		"runtime.WALDir":                  rt.WALDir(),
		"runtime.SyncCacheDir":            rt.SyncCacheDir(),
		"runtime.ReplayArtifactsDir":      rt.ReplayArtifactsDir(title.DefaultSlug),
		"runtime.TacticalRasterDir":       rt.TacticalRasterDir(title.DefaultSlug),
		"runtime.FilmFactsDir":            rt.FilmFactsDir(title.DefaultSlug),
		"config.SharedDBPath(halo_5)":     config.SharedDBPath(cfg, "halo_5"),
		"config.MetadataDBPath(halo_5)":   config.MetadataDBPath(cfg, "halo_5"),
		"cfg.TitleSettingsPath(halo_5)":   cfg.TitleSettingsPath("halo_5"),
		"cfg.DemoLayout.PlayerDBPath(h5)": cfg.DemoLayout().PlayerDBPath("halo_5", "DEMO"),
	}
}

// TestDemoBootPaths_TousSousLaRacineDemo : en démo, sans variable explicite, aucun chemin de
// boot ni de cfg ne vit sous le dépôt ; tous vivent sous la racine démo.
func TestDemoBootPaths_TousSousLaRacineDemo(t *testing.T) {
	depot := t.TempDir()
	demo := t.TempDir()
	cfg := chargerCfg(t, depot, demo, true)

	for nom, chemin := range cheminsDeBootEtDeCfg(t, cfg, depot) {
		if estSous(depot, chemin) {
			t.Errorf("%s = %s : sous le DÉPÔT en mode démo", nom, chemin)
		}
		if !estSous(demo, chemin) {
			t.Errorf("%s = %s : hors de la racine démo", nom, chemin)
		}
	}
}

// TestBootPaths_HorsDemo_Inchanges : hors démo, les chemins de boot sont ceux d'avant.
func TestBootPaths_HorsDemo_Inchanges(t *testing.T) {
	depot := t.TempDir()
	demo := t.TempDir()
	cfg := chargerCfg(t, depot, demo, false)
	pr := title.NewPathResolver(depot)

	got := resolveBootDBPaths(cfg, pr)
	want := bootDBPaths{
		shared:       pr.SharedDBPath(title.DefaultSlug),
		metadata:     pr.MetadataDBPath(title.DefaultSlug),
		sharedSocial: pr.SharedSocialDBPath(title.DefaultSlug),
		pve:          pr.SharedPVEDBPath(title.DefaultSlug),
	}
	if got != want {
		t.Errorf("bases de boot hors démo = %+v, attendu %+v", got, want)
	}
	if l, w := bootLogsConfig(depot).LogsDir, logging.LoadConfig(depot).LogsDir; l != w {
		t.Errorf("logs hors démo = %s, attendu %s", l, w)
	}
	if rt := cfg.RuntimePaths(); rt.RepoRoot() != depot {
		t.Errorf("RuntimePaths hors démo enraciné en %s, attendu le dépôt %s", rt.RepoRoot(), depot)
	}
}
