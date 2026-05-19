//go:build integration

// T9 du plan SharedDBProvider B-swap — découverte commit 7.
//
// Test en `package duckdb_test` (black-box) — cycle d'import interdit avec
// `package duckdb` car sharedprovider importe déjà duckdb.
package duckdb_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// setupSharedAndPlayer crée un shared.duckdb avec schéma + une player DB RW
// avec ATTACH RO sur shared. Reproduit la topologie du pool joueur.
func setupSharedAndPlayer(t *testing.T) (sharedPath string, playerDB *duckdb.DB) {
	t.Helper()
	sharedPath = filepath.Join(t.TempDir(), "shared.duckdb")

	boot, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("bootstrap OpenReadWrite shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(boot.SQLDb()); err != nil {
		_ = boot.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := boot.Close(); err != nil {
		t.Fatalf("bootstrap Close shared: %v", err)
	}

	playerPath := filepath.Join(t.TempDir(), "player.duckdb")
	playerDB, err = duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("player OpenReadWrite: %v", err)
	}

	attachStmt := fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)
	if _, err := playerDB.Exec(context.Background(), attachStmt); err != nil {
		_ = playerDB.Close()
		t.Fatalf("ATTACH initial: %v", err)
	}
	return sharedPath, playerDB
}

// TestPool_AttachSharedConflictsWithSwap_integration (T9 — découverte commit 7)
// documente la régression critique trouvée lors du commit 7 : tant que le pool
// joueur (ou n'importe quelle conn) maintient un ATTACH RO sur shared, le
// Provider ne peut PAS exécuter un swap RW.
//
// Symptôme observé :
//
//	Binder Error: Unique file handle conflict: Cannot attach "shared" -
//	the database file "..." is already attached by database "shared"
//
// Ce test est l'analogue de T1 (baseline rouge) pour le commit 7 : il PASSE
// tant que la régression existe ; il échouera (rouge) si DuckDB-Go résout
// cette limitation OU si le pool est migré pour DETACH avant chaque swap.
//
// Le commit 8 doit implémenter le DETACH/REATTACH via Subscribe pour que
// TestPool_AttachShared_SurvivesSwapCycle (ci-dessous, actuellement skippé)
// devienne le test cible.
func TestPool_AttachSharedConflictsWithSwap_integration(t *testing.T) {
	sharedPath, playerDB := setupSharedAndPlayer(t)
	defer func() { _ = playerDB.Close() }()

	provider, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	// La conn player a un ATTACH RO actif sur shared. Si on tente un
	// AcquireWriter sur le Provider, l'OpenReadWrite interne échoue car
	// le fichier est tenu par la conn player.
	_, err = provider.AcquireWriter(ctx)
	if err == nil {
		t.Fatal(
			"AcquireWriter aurait dû échouer (Unique file handle conflict). " +
				"Si ce test échoue, c'est soit que DuckDB-Go a corrigé la " +
				"limitation, soit que le pool n'ATTACH plus shared. Revoir " +
				"la stratégie du commit 8.")
	}
	if !strings.Contains(err.Error(), "Unique file handle conflict") {
		t.Errorf(
			"erreur attendue contenant 'Unique file handle conflict', "+
				"signature inattendue : %v", err)
	}
}

// TestPool_AttachShared_SurvivesSwapCycle_integration (T9 cible, activé
// commit 8h) — l'objectif initial du plan : la conn player avec ATTACH RO
// shared doit pouvoir faire SELECT shared.* après un cycle complet de
// swap RW du Provider.
//
// Implémentation : utilise OnSharedSwap (le helper de prod câblé dans
// main.go) pour exercer le chemin complet Subscribe → DETACH/Close →
// OpenReadWrite → re-ATTACH/Open. La preuve que main.go en mode flag-on
// résout le bug "different configuration" / "Unique file handle conflict"
// observé sur auto_sync RunDelta.
//
// Variante de TestPool_PrepareAndRestoreSharedSwap : ce test passe par
// OnSharedSwap (qui itère le globalPool) au lieu d'appeler Prepare/Restore
// directement. Couverture supplémentaire du code de prod.
func TestPool_AttachShared_SurvivesSwapCycle_integration(t *testing.T) {
	// Sprint B1 commit 9c.4 : isole le globalPool des autres tests.
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)

	sharedPath, provider, pdb := setupPoolFixturesForSwap(t)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()

	// Subscribe : utiliser le helper de prod OnSharedSwap (équivalent au
	// câblage main.go commit 8g).
	unsubscribe := provider.Subscribe(func(evt sharedprovider.SwapEvent) {
		switch evt.Direction {
		case sharedprovider.DirectionPreSwapToRW:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirPreSwapToRW)
		case sharedprovider.DirectionRWToRO:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirRWToRO)
		case sharedprovider.DirectionErrorToRO:
			duckdb.OnSharedSwap(ctx, duckdb.SwapDirErrorToRO)
		}
	})
	defer unsubscribe()

	// Sprint B1 commit 9c.4 : attachShared retiré de pdb.Player. Le test
	// utilise désormais SharedReader.Get (path Provider) à la place de
	// pdb.Player.QueryRow("shared.X"). Le cycle DETACH/REATTACH testé reste
	// pertinent pour pdb.SharedSocial (media_repo) et pour valider qu'aucune
	// régression ne se glisse côté Provider.

	// Sanity : query initiale via Provider — count initial 0.
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

	// Cycle complet : Provider.AcquireWriter → INSERT → Release.
	// OnSharedSwap est invoqué via Subscribe sur PreSwap puis RWToRO.
	w, err := provider.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter (chaîne complète) : %v", err)
	}
	if _, err := w.DB().ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES ('t9-cible-1', NOW())"); err != nil {
		w.Release()
		t.Fatalf("INSERT via writer: %v", err)
	}
	w.Release()

	// ASSERTION CRITIQUE : la conn Provider doit voir l'INSERT après le cycle.
	rdb, release, err = pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("SharedReadDB().Get post-cycle: %v", err)
	}
	if err := rdb.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM match_registry").Scan(&count); err != nil {
		release()
		t.Fatalf("query post-cycle via SharedReader: %v (chaîne B-swap cassée?)", err)
	}
	release()
	if count != 1 {
		t.Errorf("count post-cycle = %d, attendu 1 (INSERT pas visible — swap stale?)", count)
	}

	t.Logf("T9 cible OK : cycle B-swap complet sur shared = %s", sharedPath)
}
