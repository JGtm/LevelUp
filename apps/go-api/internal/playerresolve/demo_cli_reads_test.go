//go:build cgo

package playerresolve

// demo_cli_reads_test.go — un binaire opérateur chargé par Load() sous LEVELUP_DEMO_MODE=true opère sur les
// VRAIES données : la résolution d'un joueur et du lecteur partagé d'un titre suit DemoState() et non la valeur
// brute de DemoMode (suite de config/config_demo_cli_reads_test.go).

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// cfgCLIEnDemo charge la configuration d'un binaire opérateur sous LEVELUP_DEMO_MODE=true, avec un profil réel
// dans le dépôt et une fixture démo vide.
func cfgCLIEnDemo(t *testing.T) (cfg *config.AppConfig, depot, demo string) {
	t.Helper()
	depot, demo = t.TempDir(), t.TempDir()
	for _, k := range []string{
		"LEVELUP_REPO_ROOT", "LEVELUP_DEMO_MODE", "LEVELUP_DEMO_FIXTURES_DIR",
		"LEVELUP_DB_PROFILES", "LEVELUP_APP_SETTINGS", "LEVELUP_SESSION_DIR",
		"LEVELUP_AUTH_DIR", "LEVELUP_BACKUP_DIR", "LEVELUP_LOGS_DIR",
		"LEVELUP_PERSIST_BATCH_ASYNC",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("LEVELUP_REPO_ROOT", depot)
	t.Setenv("LEVELUP_DEMO_FIXTURES_DIR", demo)
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	profils := `{"version":"3.0","profiles":{"halo_infinite":{"JoueurReel":{"xuid":"2535000000000001"}}}}`
	if err := os.WriteFile(filepath.Join(depot, "db_profiles.json"), []byte(profils), 0o644); err != nil {
		t.Fatalf("db_profiles du dépôt : %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if !cfg.DemoMode {
		t.Fatal("précondition : DemoMode attendu (sémantique d'avant B5)")
	}
	return cfg, depot, demo
}

func TestLoad_Demo_ResolvePlayerNeServPasLaFixture(t *testing.T) {
	cfg, _, _ := cfgCLIEnDemo(t)
	_, err := ResolvePlayer(context.Background(), cfg, "demo-player", "")
	if !errors.Is(err, config.ErrPlayerNotFound) {
		t.Fatalf("ResolvePlayer(demo-player) = %v, attendu config.ErrPlayerNotFound (résolution réelle)", err)
	}
}

func TestLoad_Demo_SharedReaderDuTitreLitLeDepot(t *testing.T) {
	cfg, depot, _ := cfgCLIEnDemo(t)
	const h5Match = "h5-reel-match-1"
	makeSharedDBWithMatch(t, depot, halo5.TitleSlug, h5Match)
	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	cfg.SharedManager = mgr

	reader := SharedReaderForTitle(cfg, halo5.TitleSlug)
	if reader == nil {
		t.Fatal("aucun reader : le shared du titre a été cherché sous la fixture démo et non sous le dépôt")
	}
	if ok, err := readMatchExists(t, reader, h5Match); err != nil || !ok {
		t.Fatalf("le reader du titre doit voir %q dans le shared du dépôt (ok=%v err=%v)", h5Match, ok, err)
	}
}
