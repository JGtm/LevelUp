// Package main — sync_v2_wiring_known_test.go : la base partagée servie à l'ensemble connu V2
// par le câblage (knownSetSharedBorrower) est celle du provider B-swap du titre ; sans provider
// (kill-switch legacy), la connexion en cache.
package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// newKnownWiringProvider : provider B-swap réel sur un fichier DuckDB vide.
func newKnownWiringProvider(t *testing.T) sharedprovider.Provider {
	t.Helper()
	duckdbpkg.CloseAll()
	t.Cleanup(duckdbpkg.CloseAll)
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	prov, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	return prov
}

// TestKnownSetSharedBorrower_ProviderDuTitre : avec un provider, l'emprunt en RO est celui du
// provider (connexion du provider, lecteur suivi), pas la connexion en cache passée au câblage.
func TestKnownSetSharedBorrower_ProviderDuTitre(t *testing.T) {
	prov := newKnownWiringProvider(t)
	provDB, provRelease, err := prov.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	provRelease()
	autre := &sql.DB{}
	deps := SyncV2WiringDeps{
		Cfg:       &config.AppConfig{RepoRoot: t.TempDir(), SharedProvider: prov},
		TitleSlug: titlePkg.DefaultSlug,
	}

	db, release, err := knownSetSharedBorrower(deps, func() *sql.DB { return autre })(context.Background())
	if err != nil {
		t.Fatalf("emprunt : %v", err)
	}
	defer release()
	if db != provDB {
		t.Fatal("emprunt ≠ connexion du provider du titre : le câblage ignore le provider")
	}
}

// TestKnownSetSharedBorrower_SansProvider_ConnexionEnCache : kill-switch legacy (aucun provider)
// — la connexion en cache est servie.
func TestKnownSetSharedBorrower_SansProvider_ConnexionEnCache(t *testing.T) {
	enCache := &sql.DB{}
	deps := SyncV2WiringDeps{
		Cfg:       &config.AppConfig{RepoRoot: t.TempDir()},
		TitleSlug: titlePkg.DefaultSlug,
	}
	db, release, err := knownSetSharedBorrower(deps, func() *sql.DB { return enCache })(context.Background())
	if err != nil {
		t.Fatalf("emprunt : %v", err)
	}
	defer release()
	if db != enCache {
		t.Fatal("emprunt ≠ connexion en cache en mode legacy")
	}
}
