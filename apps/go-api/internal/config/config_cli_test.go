package config

// config_cli_test.go — la CLI opérateur (cmd/levelup) lit les chemins d'état du DÉPÔT, que
// LEVELUP_DEMO_MODE soit posé ou non (lot B-C9 du backlog 2026-09-26).
//
// La CI lance toute la suite Go avec LEVELUP_DEMO_MODE=true. Depuis B5.2, config.Load y
// redirige profils, réglages, auth, sessions, sauvegarde et caches vers la racine démo :
// c'est voulu pour le SERVEUR démo, pas pour seed-demo, qui produit la démo à partir des
// vrais profils. LoadForCLI en démo doit rendre EXACTEMENT les chemins d'état hors démo.

import (
	"testing"

	title "levelup/go-api/internal/domain/title"
)

// cheminsDEtat : les chemins que B5.2 et B5.5 redirigent en démo (hors bases warehouse,
// dont la résolution démo précède B5 et reste liée à DemoMode).
func cheminsDEtat(cfg *AppConfig) map[string]string {
	rt := cfg.RuntimePaths()
	return map[string]string{
		"SessionDir":                 cfg.SessionDir,
		"AuthDir":                    cfg.AuthDir,
		"UsersFilePath":              cfg.UsersFilePath(),
		"DBProfilesPath":             cfg.DBProfilesPath,
		"AppSettingsPath":            cfg.AppSettingsPath,
		"Backup.BackupDir":           cfg.Backup.BackupDir,
		"WatcherTokensDir":           cfg.WatcherTokensDir(),
		"RuntimePaths.JobsCachePath": rt.JobsCachePath(),
		"RuntimePaths.CacheRootDir":  rt.CacheRootDir(),
		"RuntimePaths.AdminStateDir": rt.AdminStateDir(),
		"RuntimePaths.FilmFacts":     rt.FilmFactsDir(title.DefaultSlug),
		"TitleSettingsPath(halo_5)":  cfg.TitleSettingsPath("halo_5"),
	}
}

func TestLoadForCLI_DemoMode_CheminsDEtatDuDepot(t *testing.T) {
	depot, demo := t.TempDir(), t.TempDir()

	poserEnvDemo(t, depot, demo, false)
	horsDemo, err := LoadForCLI()
	if err != nil {
		t.Fatalf("LoadForCLI hors démo : %v", err)
	}
	poserEnvDemo(t, depot, demo, true)
	enDemo, err := LoadForCLI()
	if err != nil {
		t.Fatalf("LoadForCLI en démo : %v", err)
	}

	if !enDemo.DemoMode {
		t.Error("LoadForCLI en démo : DemoMode doit refléter l'environnement (sémantique d'avant B5)")
	}
	attendu := cheminsDEtat(horsDemo)
	for nom, chemin := range cheminsDEtat(enDemo) {
		if chemin != attendu[nom] {
			t.Errorf("%s : %q en démo, attendu %q (identique hors démo)", nom, chemin, attendu[nom])
		}
		if sous(demo, chemin) {
			t.Errorf("%s : %q sous la racine démo", nom, chemin)
		}
	}
	if enDemo.PersistBatchAsync != horsDemo.PersistBatchAsync {
		t.Errorf("PersistBatchAsync : %v en démo, %v hors démo — la CLI ne suit pas la coupure du serveur",
			enDemo.PersistBatchAsync, horsDemo.PersistBatchAsync)
	}
}

// Témoin : le SERVEUR garde les redirections de B5.2 (Load, inchangé).
func TestLoad_DemoMode_ServeurToujoursRedirige(t *testing.T) {
	depot, demo := t.TempDir(), t.TempDir()
	poserEnvDemo(t, depot, demo, true)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	for nom, chemin := range cheminsDEtat(cfg) {
		if !sous(demo, chemin) {
			t.Errorf("serveur démo : %s = %q hors de la racine démo", nom, chemin)
		}
	}
}
