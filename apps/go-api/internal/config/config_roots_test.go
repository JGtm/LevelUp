package config

import (
	"os"
	"path/filepath"
	"testing"

	title "levelup/go-api/internal/domain/title"
)

// clearRootEnv vide les quatre variables de racine pour la durée du test.
func clearRootEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{EnvResourcesDir, EnvRepoRoot, EnvDataDir, EnvLogsDir} {
		t.Setenv(k, "")
	}
	t.Cleanup(title.ResetProcessRoots)
}

func TestResolveRoots_Precedence(t *testing.T) {
	clearRootEnv(t)
	t.Setenv(EnvRepoRoot, "/repo")
	got := ResolveRoots()
	if got.Resources != "/repo" || got.Data != filepath.Join("/repo", "data") || got.Logs != filepath.Join("/repo", "logs") || got.DataSet {
		t.Fatalf("LEVELUP_REPO_ROOT seul : %+v", got)
	}

	t.Setenv(EnvResourcesDir, "/res")
	if got := ResolveRoots(); got.Resources != "/res" {
		t.Fatalf("LEVELUP_RESOURCES_DIR doit l'emporter sur LEVELUP_REPO_ROOT : %+v", got)
	}

	t.Setenv(EnvDataDir, "/donnees")
	t.Setenv(EnvLogsDir, "/journaux")
	got = ResolveRoots()
	if got.Data != "/donnees" || !got.DataSet || got.Logs != "/journaux" || got.Resources != "/res" {
		t.Fatalf("trois racines explicites : %+v", got)
	}
}

func TestResolveRoots_DetectionSansVariable(t *testing.T) {
	clearRootEnv(t)
	got := ResolveRoots()
	if got.Resources == "" || got.Data != filepath.Join(got.Resources, "data") {
		t.Fatalf("la détection doit rendre une racine et ses défauts : %+v", got)
	}
}

func TestLoadEnvLocal_IgnoreLesRacines(t *testing.T) {
	clearRootEnv(t)
	t.Setenv("LEVELUP_TEST_ROOTS_AUTRE", "")
	file := filepath.Join(t.TempDir(), ".env.local")
	body := EnvDataDir + "=/pirate\n" + EnvLogsDir + "=/pirate\n" + EnvRepoRoot + "=/pirate\n" +
		EnvResourcesDir + "=/pirate\nLEVELUP_TEST_ROOTS_AUTRE=ok\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	loadEnvLocal(file)
	for _, k := range []string{EnvDataDir, EnvLogsDir, EnvRepoRoot, EnvResourcesDir} {
		if v := os.Getenv(k); v != "" {
			t.Errorf("%s posée par .env.local (%q) : un fichier ne pose pas les racines", k, v)
		}
	}
	if os.Getenv("LEVELUP_TEST_ROOTS_AUTRE") != "ok" {
		t.Error("les autres variables de .env.local doivent rester chargées")
	}
}

func TestLoad_RacinesParDefautEtDistinctes(t *testing.T) {
	clearRootEnv(t)
	res := t.TempDir()
	t.Setenv(EnvRepoRoot, res)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// Défaut : les trois fichiers d'état en tête du dépôt, le reste sous data/.
	if cfg.DBProfilesPath != filepath.Join(res, "db_profiles.json") || cfg.AppSettingsPath != filepath.Join(res, "app_settings.json") {
		t.Errorf("fichiers d'état par défaut : %q %q", cfg.DBProfilesPath, cfg.AppSettingsPath)
	}
	if cfg.SessionDir != filepath.Join(res, "data", "sessions") || cfg.AuthDir != filepath.Join(res, "data", "auth") {
		t.Errorf("sessions/auth par défaut : %q %q", cfg.SessionDir, cfg.AuthDir)
	}

	data, logs := t.TempDir(), t.TempDir()
	t.Setenv(EnvResourcesDir, res)
	t.Setenv(EnvDataDir, data)
	t.Setenv(EnvLogsDir, logs)
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RepoRoot != res || cfg.Roots().Data != data || cfg.Roots().Logs != logs {
		t.Fatalf("trois racines distinctes : repo=%q roots=%+v", cfg.RepoRoot, cfg.Roots())
	}
	if cfg.DBProfilesPath != filepath.Join(data, "db_profiles.json") || cfg.AppSettingsPath != filepath.Join(data, "app_settings.json") {
		t.Errorf("avec LEVELUP_DATA_DIR, les fichiers d'état vont en tête de Data : %q %q", cfg.DBProfilesPath, cfg.AppSettingsPath)
	}
	if cfg.SessionDir != filepath.Join(data, "sessions") || cfg.AuthDir != filepath.Join(data, "auth") {
		t.Errorf("sessions/auth sous Data : %q %q", cfg.SessionDir, cfg.AuthDir)
	}
	// Le pont : un service qui ne reçoit que la racine de ressources retrouve Data.
	if got := title.NewPathResolver(res).SharedDBPath("halo_infinite"); got != filepath.Join(data, "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb") {
		t.Errorf("NewPathResolver(racine de ressources) doit suivre la racine de données du processus : %q", got)
	}
}

func TestAppConfigRoots_ConfigurationConstruiteALaMain(t *testing.T) {
	clearRootEnv(t)
	cfg := &AppConfig{RepoRoot: "/depot"}
	if r := cfg.Roots(); r.Data != filepath.Join("/depot", "data") || r.Logs != filepath.Join("/depot", "logs") {
		t.Fatalf("AppConfig nue : %+v", r)
	}
	// RepoRoot réaffecté après un chargement : la disposition suit RepoRoot, pas l'ancien chargement.
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	loaded.RepoRoot = other
	if got := loaded.Roots().Data; got != filepath.Join(other, "data") {
		t.Fatalf("RepoRoot réaffecté : Data = %q", got)
	}
}

// TestInstallProcessRootsFromEnv_EnfantDeCuisson — un processus qui n'appelle pas Load (enfant
// -film-child) obtient, par un simple appel, les racines transmises par son parent : un résolveur
// bâti sur la racine de ressources rend alors la racine de données explicite, pas <res>/data.
func TestInstallProcessRootsFromEnv_EnfantDeCuisson(t *testing.T) {
	res, data, logs := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv(EnvResourcesDir, res)
	t.Setenv(EnvRepoRoot, "")
	t.Setenv(EnvDataDir, data)
	t.Setenv(EnvLogsDir, logs)
	t.Cleanup(title.ResetProcessRoots)

	got := InstallProcessRootsFromEnv()
	if !title.SameRoot(got.Data, data) || !title.SameRoot(got.Logs, logs) {
		t.Fatalf("racines résolues %+v, attendu data=%s logs=%s", got, data, logs)
	}
	pr := title.NewPathResolver(res)
	if !title.SameRoot(pr.DataRoot(), data) {
		t.Fatalf("NewPathResolver(res).DataRoot() = %s, attendu la racine de données du parent %s", pr.DataRoot(), data)
	}
	if !title.SameRoot(pr.LogsRoot(), logs) {
		t.Fatalf("NewPathResolver(res).LogsRoot() = %s, attendu %s", pr.LogsRoot(), logs)
	}
}
