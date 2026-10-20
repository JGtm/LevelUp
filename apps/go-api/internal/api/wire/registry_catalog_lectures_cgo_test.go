//go:build cgo

// Package api — registry_catalog_lectures_cgo_test.go : les tâches du catalogue (expansion des
// playlists, drain UGC) rendent la base partagée AVANT leurs accès réseau. Au démarrage du serveur
// local (2026-10-19, 22:38:29 → 22:41:11), l'expansion a tenu la base Infinite 2 min 42 s, le temps
// d'interroger 320 playlists : toutes les prises d'écriture de la base expiraient.
package wire

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// registreCatalogue : bases réelles sur disque (un match avec playlist versionnée), provider réel.
func registreCatalogue(t *testing.T) (*ServiceRegistry, sharedprovider.Provider) {
	t.Helper()
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	creerBase(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY,
			playlist_id VARCHAR, playlist_version_id VARCHAR, pair_id VARCHAR, pair_version_id VARCHAR,
			map_id VARCHAR, map_version_id VARCHAR, game_variant_id VARCHAR, game_variant_version_id VARCHAR)`,
		`INSERT INTO match_registry VALUES ('m1', 'pl-1', 'v1', 'pa-1', 'v1', 'map-1', 'v1', 'gv-1', 'v1')`)
	creerBase(t, pr.MetadataDBPath(titlePkg.DefaultSlug),
		`CREATE TABLE catalog_fetch_queue (title_slug VARCHAR, asset_type VARCHAR, asset_id VARCHAR,
			version_id VARCHAR, status VARCHAR DEFAULT 'pending', attempts INTEGER DEFAULT 0,
			last_error VARCHAR, fetched_at TIMESTAMP)`)
	prov, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	return &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: repoRoot, SharedProvider: prov}}, prov
}

// jetonsQuiEcrivent remplace la prise de jetons : à ce moment (premier accès réseau), tente une
// écriture de la base partagée et retient son issue ; aucun jeton n'est servi.
func jetonsQuiEcrivent(t *testing.T, prov sharedprovider.Provider) *[]error {
	t.Helper()
	var issues []error
	prev := jetonsCatalogue
	jetonsCatalogue = func(_ *ServiceRegistry, ctx context.Context, _ string) (*domain.HaloTokens, error) {
		wctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		w, err := prov.AcquireWriter(wctx)
		if err == nil {
			w.Release()
		}
		issues = append(issues, err)
		return nil, errors.New("aucun jeton (test)")
	}
	t.Cleanup(func() { jetonsCatalogue = prev })
	return &issues
}

func verifierEcritures(t *testing.T, tache string, issues []error) {
	t.Helper()
	if len(issues) == 0 {
		t.Fatalf("%s : la prise de jetons n'a pas eu lieu (fixture)", tache)
	}
	for _, err := range issues {
		if err != nil {
			t.Errorf("%s : écriture de la base partagée au premier accès réseau : %v", tache, err)
		}
	}
}

func TestExpandPlaylistChildren_RendLaBasePartageeAvantLeReseau(t *testing.T) {
	reg, prov := registreCatalogue(t)
	issues := jetonsQuiEcrivent(t, prov)
	if _, err := reg.ExpandPlaylistChildren(context.Background(), titlePkg.DefaultSlug); err == nil {
		t.Fatal("expansion sans jeton : erreur attendue")
	}
	verifierEcritures(t, "expansion des playlists", *issues)
}

func TestRunCatalogUGCDrain_RendLaBasePartageeAvantLeReseau(t *testing.T) {
	reg, prov := registreCatalogue(t)
	issues := jetonsQuiEcrivent(t, prov)
	res, err := reg.RunCatalogUGCDrain(context.Background(), titlePkg.DefaultSlug)
	if err == nil {
		t.Fatal("drain sans jeton : erreur attendue")
	}
	if res.Seeded == 0 {
		t.Errorf("recensement : 0 asset enfilé, attendu ceux du match (fixture)")
	}
	verifierEcritures(t, "drain UGC", *issues)
}
