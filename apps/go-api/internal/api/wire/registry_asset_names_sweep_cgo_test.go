//go:build cgo

// Package api — registry_asset_names_sweep_cgo_test.go : le balayage périodique des noms
// d'assets FAIT CONVERGER le registre. Bases DuckDB réelles sur disque, SharedProvider réel
// (B-swap RO↔RW) ; le pool de tokens est factice et ne sert rien : la traduction est déjà en
// metadata, seule la réinscription dans match_registry est éprouvée.
package wire

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// poolSansJeton : aucun token disponible — la passe de résolution réseau se dégrade (journal),
// la convergence du registre doit avoir lieu quand même.
type poolSansJeton struct{ pool.Pool }

func (poolSansJeton) Acquire(context.Context, pool.AcquirePolicy, string) (*pool.Lease, error) {
	return nil, errors.New("aucun jeton (test)")
}

// creerBase crée un fichier DuckDB et y joue les instructions.
func creerBase(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	for _, s := range stmts {
		if _, err := db.Exec(context.Background(), s); err != nil {
			_ = db.Close()
			t.Fatalf("exec %.50s: %v", s, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

func TestResolveUnresolvedAssetNames_FaitConvergerLeRegistre(t *testing.T) {
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	creerBase(t, sharedPath,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY,
			playlist_id VARCHAR, playlist_name VARCHAR, playlist_version_id VARCHAR,
			map_id VARCHAR, map_name VARCHAR, map_version_id VARCHAR,
			pair_id VARCHAR, pair_name VARCHAR, pair_version_id VARCHAR,
			game_variant_id VARCHAR, game_variant_name VARCHAR, game_variant_version_id VARCHAR,
			mode_category VARCHAR)`,
		`INSERT INTO match_registry (match_id, map_id, map_name, map_version_id, mode_category)
			VALUES ('m1', 'map-x', NULL, 'v1', 'other'), ('m2', 'map-x', 'map-x', 'v1', 'other')`)
	creerBase(t, pr.MetadataDBPath(titlePkg.DefaultSlug),
		`CREATE TABLE asset_translations (asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR,
			name VARCHAR, description VARCHAR, fetched_at TIMESTAMP)`,
		`INSERT INTO asset_translations VALUES ('map-x', 'map', 'en-US', 'Streets', '', now())`)

	prov, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = prov.Close() })
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: repoRoot, SharedProvider: prov}}

	ctx := context.Background()
	if _, err := reg.ResolveUnresolvedAssetNames(ctx, titlePkg.DefaultSlug, poolSansJeton{}); err != nil {
		t.Fatalf("ResolveUnresolvedAssetNames: %v", err)
	}

	db, release, err := prov.Get(ctx)
	if err != nil {
		t.Fatalf("prov.Get: %v", err)
	}
	defer release()
	for _, id := range []string{"m1", "m2"} {
		var nom, categorie sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT map_name, mode_category FROM match_registry WHERE match_id = ?`, id).
			Scan(&nom, &categorie); err != nil {
			t.Fatalf("lecture %s: %v", id, err)
		}
		if nom.String != "Streets" {
			t.Errorf("%s map_name = %q, want Streets (convergence après balayage)", id, nom.String)
		}
		if categorie.String != "other" {
			t.Errorf("%s mode_category = %q, want inchangée", id, categorie.String)
		}
	}
}
