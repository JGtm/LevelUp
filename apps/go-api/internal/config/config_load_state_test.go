package config

// config_load_state_test.go — la redirection démo des chemins d'état appartient au SEUL
// processus serveur (lots B-C9 puis B-C10 du backlog 2026-09-26).
//
// La CI lance toute la suite Go avec LEVELUP_DEMO_MODE=true, et tout binaire opérateur
// (cmd/levelup, cmd/token-capture, cmd/restore…) opère sur les VRAIES données. Load, que
// tous appellent, rend donc en démo EXACTEMENT les chemins d'état hors démo (sémantique
// d'avant B5). LoadServer, appelé par cmd/server et lui seul, applique les redirections de
// B5.2 et B5.5 (config_demo_hermetic_test.go).

import (
	"testing"

	title "levelup/go-api/internal/domain/title"
)

// cheminsDEtat : les chemins que B5.2 et B5.5 redirigent pour le serveur démo (hors bases
// warehouse, dont la résolution démo précède B5 et reste liée à DemoMode).
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

func TestLoad_DemoMode_CheminsDEtatDuDepot(t *testing.T) {
	depot, demo := t.TempDir(), t.TempDir()

	poserEnvDemo(t, depot, demo, false)
	horsDemo, err := Load()
	if err != nil {
		t.Fatalf("Load hors démo : %v", err)
	}
	poserEnvDemo(t, depot, demo, true)
	enDemo, err := Load()
	if err != nil {
		t.Fatalf("Load en démo : %v", err)
	}

	if !enDemo.DemoMode {
		t.Error("Load en démo : DemoMode doit refléter l'environnement (sémantique d'avant B5)")
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
		t.Errorf("PersistBatchAsync : %v en démo, %v hors démo — seul le serveur démo coupe la file",
			enDemo.PersistBatchAsync, horsDemo.PersistBatchAsync)
	}
}

// Témoin : le SERVEUR démo garde les redirections de B5.2 et la coupure de la file persist.
func TestLoadServer_DemoMode_Redirige(t *testing.T) {
	depot, demo := t.TempDir(), t.TempDir()
	poserEnvDemo(t, depot, demo, true)
	cfg, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer : %v", err)
	}
	for nom, chemin := range cheminsDEtat(cfg) {
		if !sous(demo, chemin) {
			t.Errorf("serveur démo : %s = %q hors de la racine démo", nom, chemin)
		}
	}
	if cfg.PersistBatchAsync {
		t.Error("serveur démo : PersistBatchAsync = true, attendu false (file persist coupée, B5.3)")
	}
}
