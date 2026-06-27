// Tests de la résolution per-titre du SharedReader (B-swap multi-titre).
//
// Couvre :
//  1. HINF byte-identique : sharedReaderForTitle(DefaultSlug) retourne le MÊME
//     provider que cfg.SharedProvider (le provider boot, caché par path dans le
//     Manager) → aucune nouvelle conn DuckDB, comportement Infinite inchangé.
//  2. Sélection per-titre : un titre non-Infinite (halo_5) lit SON propre shared
//     (le match_id n'existe que dans le h5), PAS le shared Infinite.
//  3. Fallbacks : Manager nil (kill-switch) et mode démo → cfg.SharedProvider.
//
// Build tag cgo : ouvre des providers DuckDB réels via le Manager.

//go:build cgo

package config

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"

	_ "github.com/duckdb/duckdb-go/v2"
)

// makeSharedDBWithMatch crée le shared_matches_v2.duckdb au path title-aware du
// slug donné, avec une table match_registry minimale + une ligne matchID si
// non vide. Schéma minimal (pas de dépendance à internal/sync — cycle d'import).
func makeSharedDBWithMatch(t *testing.T, repoRoot, slug, matchID string) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).SharedDBPath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir warehouse %s: %v", slug, err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared %s: %v", slug, err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if _, err := db.SQLDb().ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS match_registry (
			match_id   VARCHAR PRIMARY KEY,
			start_time TIMESTAMP NOT NULL
		)`); err != nil {
		t.Fatalf("create match_registry %s: %v", slug, err)
	}
	if matchID != "" {
		if _, err := db.SQLDb().ExecContext(ctx,
			"INSERT INTO match_registry (match_id, start_time) VALUES (?, CURRENT_TIMESTAMP)",
			matchID); err != nil {
			t.Fatalf("insert match %s: %v", slug, err)
		}
	}
	return path
}

// readMatchExists teste si matchID est présent dans match_registry via un
// SharedReader (provider RO). Retourne (existe, erreur de lecture).
func readMatchExists(t *testing.T, reader duckdb.SharedReader, matchID string) (bool, error) {
	t.Helper()
	ctx := context.Background()
	db, release, err := reader.Get(ctx)
	if err != nil {
		return false, err
	}
	defer release()
	var n int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry WHERE match_id = ?", matchID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// TestSharedReaderForTitle_HINF_ByteIdentical (test #1) : pour DefaultSlug,
// sharedReaderForTitle retourne le provider boot lui-même (caché par path).
func TestSharedReaderForTitle_HINF_ByteIdentical(t *testing.T) {
	repoRoot := t.TempDir()
	hinfPath := makeSharedDBWithMatch(t, repoRoot, titlePkg.DefaultSlug, "")

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	// Reproduit le boot : provider du shared Infinite, injecté dans cfg.
	bootProvider, err := mgr.For(hinfPath)
	if err != nil {
		t.Fatalf("For(hinf): %v", err)
	}
	cfg := &AppConfig{
		RepoRoot:       repoRoot,
		SharedProvider: bootProvider,
		SharedManager:  mgr,
	}

	got := cfg.sharedReaderForTitle(titlePkg.DefaultSlug)
	if got != duckdb.SharedReader(bootProvider) {
		t.Errorf("sharedReaderForTitle(DefaultSlug) != provider boot (byte-identique attendu)")
	}
	// "" doit être normalisé vers DefaultSlug → même provider.
	if cfg.sharedReaderForTitle("") != duckdb.SharedReader(bootProvider) {
		t.Errorf("sharedReaderForTitle(\"\") != provider boot")
	}
}

// TestSharedReaderForTitle_PerTitle_Selection (test #2) : halo_5 lit son propre
// shared (match présent uniquement côté h5), Infinite lit le sien.
func TestSharedReaderForTitle_PerTitle_Selection(t *testing.T) {
	repoRoot := t.TempDir()
	const h5Match = "h5-only-match-1234"
	const hinfMatch = "hinf-only-match-5678"

	hinfPath := makeSharedDBWithMatch(t, repoRoot, titlePkg.DefaultSlug, hinfMatch)
	_ = makeSharedDBWithMatch(t, repoRoot, halo5.TitleSlug, h5Match)

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	bootProvider, err := mgr.For(hinfPath)
	if err != nil {
		t.Fatalf("For(hinf): %v", err)
	}
	cfg := &AppConfig{
		RepoRoot:       repoRoot,
		SharedProvider: bootProvider,
		SharedManager:  mgr,
	}

	// Le reader h5 voit h5Match et NE voit PAS hinfMatch.
	h5Reader := cfg.sharedReaderForTitle(halo5.TitleSlug)
	if ok, err := readMatchExists(t, h5Reader, h5Match); err != nil || !ok {
		t.Errorf("reader h5 doit voir %q (ok=%v err=%v)", h5Match, ok, err)
	}
	if ok, _ := readMatchExists(t, h5Reader, hinfMatch); ok {
		t.Errorf("reader h5 ne doit PAS voir le match Infinite %q (fuite cross-titre)", hinfMatch)
	}

	// Le reader Infinite voit hinfMatch et PAS h5Match.
	hinfReader := cfg.sharedReaderForTitle(titlePkg.DefaultSlug)
	if ok, err := readMatchExists(t, hinfReader, hinfMatch); err != nil || !ok {
		t.Errorf("reader Infinite doit voir %q (ok=%v err=%v)", hinfMatch, ok, err)
	}
	if ok, _ := readMatchExists(t, hinfReader, h5Match); ok {
		t.Errorf("reader Infinite ne doit PAS voir le match h5 %q (fuite cross-titre)", h5Match)
	}

	// Le provider h5 est DISTINCT du provider boot Infinite (paths différents).
	if h5Reader == duckdb.SharedReader(bootProvider) {
		t.Errorf("le reader h5 ne doit pas être le provider Infinite")
	}
}

// TestSharedReaderForTitle_Fallbacks : Manager nil (kill-switch) → cfg.SharedProvider ;
// démo titre par DÉFAUT → cfg.SharedProvider (provider boot, byte-identique).
func TestSharedReaderForTitle_Fallbacks(t *testing.T) {
	// Provider sentinel in-memory (identité comparable, pas de fichier).
	mem, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	defer func() { _ = mem.Close() }()
	sentinel := sharedprovider.FromInMemoryDB(mem, "sentinel")

	// Manager nil → fallback SharedProvider direct.
	cfg := &AppConfig{RepoRoot: t.TempDir(), SharedProvider: sentinel}
	if got := cfg.sharedReaderForTitle(halo5.TitleSlug); got != duckdb.SharedReader(sentinel) {
		t.Errorf("Manager nil : attendu fallback SharedProvider")
	}
	// Démo + titre par DÉFAUT → provider boot (cfg.SharedProvider), sans routage.
	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	cfgDemo := &AppConfig{RepoRoot: t.TempDir(), DemoMode: true, SharedProvider: sentinel, SharedManager: mgr}
	if got := cfgDemo.sharedReaderForTitle(titlePkg.DefaultSlug); got != duckdb.SharedReader(sentinel) {
		t.Errorf("démo défaut : attendu provider boot (cfg.SharedProvider)")
	}
}

// TestSharedReaderForTitle_DemoPerTitle : en démo, un titre ADDITIONNEL résout le
// provider de SON shared démo title-scopé (data/demo/titles/{slug}/warehouse/…),
// DISTINCT du provider boot du titre par défaut.
func TestSharedReaderForTitle_DemoPerTitle(t *testing.T) {
	fixtures := t.TempDir()
	const h5Match = "h5-demo-match-9999"

	// Shared démo H5 title-scopé : fixtures/titles/halo_5/warehouse/shared_matches_v2.duckdb.
	h5Dir := filepath.Join(fixtures, "titles", halo5.TitleSlug, "warehouse")
	if err := os.MkdirAll(h5Dir, 0o755); err != nil {
		t.Fatalf("mkdir h5 demo warehouse: %v", err)
	}
	h5Path := filepath.Join(h5Dir, "shared_matches_v2.duckdb")
	h5db, err := duckdb.OpenReadWrite(h5Path)
	if err != nil {
		t.Fatalf("OpenReadWrite h5 demo shared: %v", err)
	}
	ctx := context.Background()
	if _, err := h5db.SQLDb().ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	if _, err := h5db.SQLDb().ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES (?, CURRENT_TIMESTAMP)", h5Match); err != nil {
		t.Fatalf("insert h5 match: %v", err)
	}
	_ = h5db.Close()

	mem, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	defer func() { _ = mem.Close() }()
	sentinel := sharedprovider.FromInMemoryDB(mem, "boot")

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()
	cfg := &AppConfig{DemoMode: true, DemoFixturesDir: fixtures, SharedProvider: sentinel, SharedManager: mgr}

	h5Reader := cfg.sharedReaderForTitle(halo5.TitleSlug)
	if h5Reader == duckdb.SharedReader(sentinel) {
		t.Fatalf("démo H5 : doit résoudre un provider DISTINCT du provider boot")
	}
	if ok, err := readMatchExists(t, h5Reader, h5Match); err != nil || !ok {
		t.Errorf("démo H5 : le reader doit voir %q (ok=%v err=%v)", h5Match, ok, err)
	}
}
