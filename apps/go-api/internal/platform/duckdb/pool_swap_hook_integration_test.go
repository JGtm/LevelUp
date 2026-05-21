//go:build integration

// Tests pour la mécanique B-swap pool-side (commit 8f).
//
// Valide PrepareForSharedSwap + RestoreSharedAfterSwap en mode manuel
// (sans passer par Subscribe — c'est 8g qui câblera). Si ces tests passent,
// T9 cible (TestPool_AttachShared_SurvivesSwapCycle) devient activable au
// commit 8g.
package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// setupPoolFixturesForSwap prépare shared + metadata + un PlayerDB avec un
// Provider injecté (mode B-swap).
func setupPoolFixturesForSwap(t *testing.T) (sharedPath string, provider sharedprovider.Provider, pdb *duckdb.PlayerDB) {
	t.Helper()
	dir := t.TempDir()
	// IMPORTANT : utiliser le filename réel de prod (shared_matches_v2.duckdb)
	// pour que l'auto-attach DuckDB-Go utilise l'alias `shared_matches_v2`,
	// distinct de l'alias `shared` utilisé par attachShared. Sinon, conflit
	// "already attached by database 'shared'" artificiel.
	sharedPath = filepath.Join(dir, "shared_matches_v2.duckdb")
	metaPath := filepath.Join(dir, "metadata.duckdb")
	playerPath := filepath.Join(dir, "player.duckdb")

	// Schéma shared.
	sb, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), sb.SQLDb()); err != nil {
		_ = sb.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = sb.Close()

	// Metadata vide.
	mb, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	_ = mb.Close()

	// Provider sur shared.
	provider, err = sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}

	// PlayerDB avec Provider injecté (mode B-swap).
	cfg := duckdb.PlayerPoolConfig{
		Gamertag:     "swap-test",
		XUID:         "999",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
		SharedReader: provider,
	}
	pdb, err = duckdb.GetOrOpen(context.Background(), cfg)
	if err != nil {
		_ = provider.Close()
		t.Fatalf("GetOrOpen: %v", err)
	}
	return sharedPath, provider, pdb
}

// TestPool_PrepareAndRestoreSharedSwap_integration (commit 8f) valide la
// mécanique B3 en mode manuel : on simule à la main les callbacks que le
// Provider invoquera depuis Subscribe.
//
// Scénario :
//  1. Initial : ATTACH RO sur player conn fonctionne (SELECT shared.* OK)
//  2. PrepareForSharedSwap → conn player Reopened sans ATTACH, Shared closed
//  3. Provider.AcquireWriter réussit (file libéré, plus de "Unique file handle conflict")
//  4. INSERT via writer
//  5. WriterHandle.Release
//  6. RestoreSharedAfterSwap → Shared rouvert + ATTACH reposé
//  7. Final : SELECT shared.* voit l'INSERT
//
// Si ce test passe : la mécanique B3 est fonctionnelle. Reste à câbler
// via Subscribe au commit 8g.
func TestPool_PrepareAndRestoreSharedSwap_integration(t *testing.T) {
	// isole le globalPool des autres tests pour éviter
	// de récupérer un PlayerDB cached avec un Provider fermé.
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	_, provider, pdb := setupPoolFixturesForSwap(t)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	// Subscribe — câble la mécanique B3 sur les events du Provider.
	// Le timing exact (PreSwap en Phase 3, après Close du handle Provider)
	// ne peut être respecté QUE via Subscribe — pas en mode manuel.
	unsubscribe := provider.Subscribe(func(_ context.Context, evt sharedprovider.SwapEvent) {
		switch evt.Direction {
		case sharedprovider.DirectionPreSwapToRW:
			if err := pdb.PrepareForSharedSwap(ctx); err != nil {
				t.Errorf("PrepareForSharedSwap (callback): %v", err)
			}
		case sharedprovider.DirectionRWToRO, sharedprovider.DirectionErrorToRO:
			if err := pdb.RestoreSharedAfterSwap(ctx); err != nil {
				t.Errorf("RestoreSharedAfterSwap (callback): %v", err)
			}
		}
	})
	defer unsubscribe()

	// attachShared retiré de pdb.Player. Le test
	// utilise désormais SharedReader.Get pour valider que le cycle Prepare/
	// Restore ne perturbe pas les lectures via Provider.

	// Étape 1 : query via SharedReader — count initial 0.
	rdb, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("SharedReadDB().Get initial: %v", err)
	}
	var count int
	if err := rdb.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&count); err != nil {
		release()
		t.Fatalf("query initial via SharedReader: %v", err)
	}
	release()
	if count != 0 {
		t.Errorf("count initial = %d, attendu 0", count)
	}

	// Étape 2 : AcquireWriter. Le callback Subscribe fera Prepare au bon
	// moment (Phase 3, après que Provider.handle soit fermé).
	w, err := provider.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v (mécanisme B3 cassé ?)", err)
	}

	// Étape 3 : INSERT.
	if _, err := w.DB().ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES ('swap-test-1', NOW())"); err != nil {
		w.Release()
		t.Fatalf("INSERT via writer: %v", err)
	}

	// Étape 4 : Release. Le callback Subscribe fera Restore après le
	// swap RW → RO complet.
	w.Release()

	// Étape 5 : la conn Provider doit voir l'INSERT après cycle.
	rdb, release, err = pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("SharedReadDB().Get post-cycle: %v", err)
	}
	if err := rdb.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&count); err != nil {
		release()
		t.Fatalf("query post-cycle via SharedReader: %v", err)
	}
	release()
	if count != 1 {
		t.Errorf("count post-cycle = %d, attendu 1 (INSERT pas visible — swap stale?)", count)
	}

	// Bonus : 2e cycle pour valider la stabilité.
	w2, err := provider.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter #2: %v", err)
	}
	if _, err := w2.DB().ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES ('swap-test-2', NOW())"); err != nil {
		w2.Release()
		t.Fatalf("INSERT #2: %v", err)
	}
	w2.Release()

	rdb, release, err = pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("SharedReadDB().Get final: %v", err)
	}
	if err := rdb.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&count); err != nil {
		release()
		t.Fatalf("query final via SharedReader: %v", err)
	}
	release()
	if count != 2 {
		t.Errorf("count final = %d, attendu 2 (cycle 2 visible?)", count)
	}
}

// TestPool_PrepareForSharedSwap_NoopWithoutSharedReader_integration vérifie
// que PrepareForSharedSwap est no-op si pdb.SharedReader est nil (mode
// legacy). Garantit qu'on ne casse pas les PlayerDB construits sans Provider.
func TestPool_PrepareForSharedSwap_NoopWithoutSharedReader_integration(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared.duckdb")
	metaPath := filepath.Join(dir, "metadata.duckdb")
	playerPath := filepath.Join(dir, "player.duckdb")

	sb, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), sb.SQLDb()); err != nil {
		_ = sb.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = sb.Close()

	mb, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	_ = mb.Close()

	cfg := duckdb.PlayerPoolConfig{
		Gamertag:     "swap-test-legacy",
		XUID:         "888",
		TitleSlug:    "halo_infinite",
		PlayerDBPath: playerPath,
		SharedDBPath: sharedPath,
		MetaDBPath:   metaPath,
		// SharedReader nil → mode legacy
	}
	pdb, err := duckdb.GetOrOpen(context.Background(), cfg)
	if err != nil {
		t.Fatalf("GetOrOpen: %v", err)
	}

	// Capture le pointeur Shared avant prepare.
	sharedBefore := pdb.Shared

	// PrepareForSharedSwap doit être no-op en mode legacy.
	if err := pdb.PrepareForSharedSwap(context.Background()); err != nil {
		t.Fatalf("PrepareForSharedSwap (mode legacy): %v", err)
	}

	// Shared pointer inchangé, conns toujours fonctionnelles.
	if pdb.Shared != sharedBefore {
		t.Error("Shared pointer changé en mode legacy — devrait être no-op")
	}
	var v string
	if err := pdb.Shared.QueryRow(context.Background(), "SELECT version()").Scan(&v); err != nil {
		t.Errorf("Shared inutilisable après PrepareForSharedSwap en mode legacy: %v", err)
	}
}
