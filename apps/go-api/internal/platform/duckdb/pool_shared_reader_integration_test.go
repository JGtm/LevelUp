//go:build integration

// Tests pour la migration commit 8a : PlayerDB.SharedReader.
//
// Vérifie que PlayerDB expose toujours un SharedReader valide, en mode
// legacy (cfg.SharedReader nil → LegacySharedReader auto) ET en mode
// B-swap (cfg.SharedReader injecté).
package duckdb_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// setupPoolFixtures crée les fichiers shared + metadata + player nécessaires
// à openPlayerDB. Retourne les paths.
func setupPoolFixtures(t *testing.T) (sharedPath, metaPath, playerPath string) {
	t.Helper()
	dir := t.TempDir()
	sharedPath = filepath.Join(dir, "shared.duckdb")
	metaPath = filepath.Join(dir, "metadata.duckdb")
	playerPath = filepath.Join(dir, "player.duckdb")

	// Shared : appliquer schéma.
	sb, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(sb.SQLDb()); err != nil {
		_ = sb.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = sb.Close()

	// Metadata : créer fichier vide (le pool s'attend à ce qu'il existe et
	// applique des migrations idempotentes via OpenReadWriteShared).
	mb, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	_ = mb.Close()

	// Player : créé à la volée par openPlayerDB (ensurePlayerDBMigrations
	// gère l'open initial).
	return sharedPath, metaPath, playerPath
}

// TestOpenPlayerDB_LegacySharedReader vérifie que sans cfg.SharedReader,
// PlayerDB.SharedReader est un LegacySharedReader autour de pdb.Shared —
// donc Get retourne le *sql.DB de pdb.Shared.
func TestOpenPlayerDB_LegacySharedReader(t *testing.T) {
	sharedPath, metaPath, playerPath := setupPoolFixtures(t)

	cfg := duckdb.PlayerPoolConfig{
		Gamertag:     "test-legacy",
		XUID:         "123",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
		// cfg.SharedReader vide → mode legacy
	}

	ctx := context.Background()
	pdb, err := duckdb.GetOrOpen(ctx, cfg)
	if err != nil {
		t.Fatalf("GetOrOpen: %v", err)
	}

	if pdb.SharedReader == nil {
		t.Fatal("PlayerDB.SharedReader doit être initialisé même en mode legacy")
	}

	// Le SharedReader doit retourner le *sql.DB de pdb.Shared.
	db, release, err := pdb.SharedReader.Get(ctx)
	if err != nil {
		t.Fatalf("SharedReader.Get: %v", err)
	}
	defer release()

	if db == nil {
		t.Fatal("SharedReader.Get a retourné un *sql.DB nil")
	}
	if db != pdb.Shared.SQLDb() {
		t.Errorf("legacy SharedReader doit pointer vers pdb.Shared.SQLDb(), %p vs %p",
			db, pdb.Shared.SQLDb())
	}

	// Sanity ping.
	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping SharedReader: %v", err)
	}
}

// TestOpenPlayerDB_InjectedSharedReader vérifie que cfg.SharedReader (un
// sharedprovider.Provider) est exposé tel quel via PlayerDB.SharedReader.
// Pas de wrapper, pas d'ouverture de conn supplémentaire.
func TestOpenPlayerDB_InjectedSharedReader(t *testing.T) {
	sharedPath, metaPath, playerPath := setupPoolFixtures(t)

	provider, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	defer func() { _ = provider.Close() }()

	cfg := duckdb.PlayerPoolConfig{
		Gamertag:     "test-bswap",
		XUID:         "456",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
		SharedReader: provider, // mode B-swap
	}

	ctx := context.Background()
	pdb, err := duckdb.GetOrOpen(ctx, cfg)
	if err != nil {
		t.Fatalf("GetOrOpen: %v", err)
	}

	// PlayerDB.SharedReader doit être exactement le Provider injecté
	// (pas un wrapper supplémentaire).
	var reader interface{} = pdb.SharedReader
	if reader != interface{}(provider) {
		t.Errorf("pdb.SharedReader doit être le Provider injecté tel quel")
	}

	// Le SharedReader (Provider) doit fonctionner.
	db, release, err := pdb.SharedReader.Get(ctx)
	if err != nil {
		t.Fatalf("Provider.Get via SharedReader: %v", err)
	}
	defer release()

	if db == nil {
		t.Fatal("Provider.Get a retourné un *sql.DB nil")
	}

	var v string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Errorf("ping Provider via SharedReader: %v", err)
	}
}

// TestOpenPlayerDB_NoSharedSchemaOnPoolConns garde-fou ADR 0016.
//
// Le commit 9c.5 (retrait final attachShared) a supprimé tout ATTACH `shared`
// du pool joueur. Aucune conn `pdb.Player`, `pdb.SharedSocial`, ou
// `pdb.Metadata` ne doit avoir accès au schéma `shared` — toutes les requêtes
// shared passent via `pdb.SharedReadDB().Get(ctx)` qui retourne une conn
// pointée directement sur la DB shared (sans préfixe `shared.`).
//
// Ce test échouera si :
//   - quelqu'un réintroduit un `ATTACH ... AS shared` sur l'une des conns
//     pool (régression vers le bug "different configuration" éliminé par
//     l'ADR 0016) ;
//   - un nouveau site applicatif exécute `shared.X` sur `pdb.Player` /
//     `pdb.SharedSocial` / `pdb.Metadata` (le bug latent dans
//     `LoadMediaFiles` est précisément de cette nature).
//
// L'invariant vérifié : `SELECT 1 FROM shared.match_registry` DOIT échouer
// sur les conns du pool, mais réussir sur la conn obtenue via SharedReader
// (en root-level, sans préfixe).
func TestOpenPlayerDB_NoSharedSchemaOnPoolConns(t *testing.T) {
	sharedPath, metaPath, playerPath := setupPoolFixtures(t)

	cfg := duckdb.PlayerPoolConfig{
		Gamertag:     "test-sentinel",
		XUID:         "789",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
	}

	ctx := context.Background()
	pdb, err := duckdb.GetOrOpen(ctx, cfg)
	if err != nil {
		t.Fatalf("GetOrOpen: %v", err)
	}

	probe := "SELECT 1 FROM shared.match_registry LIMIT 1"

	poolConns := []struct {
		name string
		db   *duckdb.DB
	}{
		{"Player", pdb.Player},
		{"Metadata", pdb.Metadata},
	}
	if pdb.SharedSocial != nil {
		poolConns = append(poolConns, struct {
			name string
			db   *duckdb.DB
		}{"SharedSocial", pdb.SharedSocial})
	}

	for _, c := range poolConns {
		if c.db == nil {
			continue
		}
		var n int
		err := c.db.QueryRow(ctx, probe).Scan(&n)
		if err == nil {
			t.Errorf("conn %s: %q a réussi alors qu'aucun ATTACH shared ne doit "+
				"être posé sur les conns du pool (ADR 0016). Régression probable.",
				c.name, probe)
			continue
		}
		// L'erreur DuckDB attendue mentionne "schema" et "shared" (libellé
		// peut varier : "schema 'shared' does not exist", "Catalog Error:
		// Table with name ... does not exist because schema ... does not
		// exist", etc.). On accepte tout message qui contient les deux mots.
		msg := err.Error()
		if !containsAll(msg, "shared", "schema") && !containsAll(msg, "shared.match_registry", "does not exist") {
			t.Errorf("conn %s: erreur inattendue %v (attendu : message catalog mentionnant 'shared' et 'schema')",
				c.name, err)
		}
	}

	// Contre-épreuve : la même table accessible SANS préfixe via SharedReader.
	sharedDB, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("SharedReader.Get: %v", err)
	}
	defer release()

	var n int
	if err := sharedDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&n); err != nil {
		t.Errorf("SELECT FROM match_registry via SharedReader doit fonctionner : %v", err)
	}
}

// containsAll retourne true si s contient toutes les sous-chaînes (case-insensitive).
func containsAll(s string, subs ...string) bool {
	low := strings.ToLower(s)
	for _, sub := range subs {
		if !strings.Contains(low, strings.ToLower(sub)) {
			return false
		}
	}
	return true
}
