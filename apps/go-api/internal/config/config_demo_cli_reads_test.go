//go:build cgo

package config

// config_demo_cli_reads_test.go — un binaire opérateur chargé par Load() sous
// LEVELUP_DEMO_MODE=true opère sur les VRAIES données : toute lecture de profils, de chemins
// de bases et de joueur suit demoState() et non la valeur brute de DemoMode.
//
// Avant : six lectures de cfg.DemoMode (LoadPlayers, ResolvePlayer, SharedDBPath,
// MetadataDBPath, PrestigeBundleDBPaths, sharedReaderForTitle) renvoyaient la fixture démo à
// `levelup recompute-friends`, alors que Load() redirige déjà les chemins d'état vers le dépôt.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// cfgCLIEnDemo charge la configuration d'un binaire opérateur sous LEVELUP_DEMO_MODE=true,
// avec un profil réel dans le dépôt et une fixture démo vide.
func cfgCLIEnDemo(t *testing.T) (cfg *AppConfig, depot, demo string) {
	t.Helper()
	depot, demo = t.TempDir(), t.TempDir()
	poserEnvDemo(t, depot, demo, true)
	profils := `{"version":"3.0","profiles":{"halo_infinite":{"JoueurReel":{"xuid":"2535000000000001"}}}}`
	if err := os.WriteFile(filepath.Join(depot, "db_profiles.json"), []byte(profils), 0o644); err != nil {
		t.Fatalf("db_profiles du dépôt : %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if !cfg.DemoMode {
		t.Fatal("précondition : DemoMode attendu (sémantique d'avant B5)")
	}
	return cfg, depot, demo
}

func TestLoad_Demo_LoadPlayersLitLesProfilsDuDepot(t *testing.T) {
	cfg, _, _ := cfgCLIEnDemo(t)
	players, err := cfg.LoadPlayers()
	if err != nil {
		t.Fatalf("LoadPlayers : %v", err)
	}
	if len(players) != 1 || players[0].Gamertag != "JoueurReel" {
		t.Fatalf("LoadPlayers = %+v, attendu le profil réel du dépôt (et non le roster démo)", players)
	}
}

func TestLoad_Demo_CheminsDeBasesDuDepot(t *testing.T) {
	cfg, depot, _ := cfgCLIEnDemo(t)
	pr := titlePkg.NewPathResolver(depot)
	social, meta := cfg.PrestigeBundleDBPaths()
	for nom, chemin := range map[string]string{
		"SharedDBPath":            SharedDBPath(cfg, titlePkg.DefaultSlug),
		"SharedDBPath(halo_5)":    SharedDBPath(cfg, halo5.TitleSlug),
		"MetadataDBPath":          MetadataDBPath(cfg, titlePkg.DefaultSlug),
		"PrestigeBundle.social":   social,
		"PrestigeBundle.metadata": meta,
	} {
		if !sous(depot, chemin) {
			t.Errorf("%s = %q : hors du dépôt, la fixture démo est lue par un binaire opérateur", nom, chemin)
		}
	}
	if got, want := SharedDBPath(cfg, titlePkg.DefaultSlug), pr.SharedDBPath(titlePkg.DefaultSlug); got != want {
		t.Errorf("SharedDBPath = %q, attendu %q", got, want)
	}
}

func TestLoad_Demo_ResolvePlayerNeServPasLaFixture(t *testing.T) {
	cfg, _, _ := cfgCLIEnDemo(t)
	_, err := ResolvePlayer(context.Background(), cfg, "demo-player", "")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("ResolvePlayer(demo-player) = %v, attendu ErrPlayerNotFound (résolution réelle)", err)
	}
}

func TestLoad_Demo_SharedReaderDuTitreLitLeDepot(t *testing.T) {
	cfg, depot, _ := cfgCLIEnDemo(t)
	const h5Match = "h5-reel-match-1"
	makeSharedDBWithMatch(t, depot, halo5.TitleSlug, h5Match)
	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	cfg.SharedManager = mgr

	reader := cfg.sharedReaderForTitle(halo5.TitleSlug)
	if reader == nil {
		t.Fatal("aucun reader : le shared du titre a été cherché sous la fixture démo et non sous le dépôt")
	}
	if ok, err := readMatchExists(t, reader, h5Match); err != nil || !ok {
		t.Fatalf("le reader du titre doit voir %q dans le shared du dépôt (ok=%v err=%v)", h5Match, ok, err)
	}
}
