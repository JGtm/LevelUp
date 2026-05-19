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

// TestPool_AttachShared_SurvivesSwapCycle_integration (T9 cible du commit 8)
// validera que la conn player peut faire SELECT shared.* après un swap RW.
//
// Actuellement skippé : nécessite que le pool migre vers un mécanisme
// DETACH-pre-swap / REATTACH-post-swap via Provider.Subscribe. La limitation
// actuelle est documentée par TestPool_AttachSharedConflictsWithSwap.
//
// L'implémentation cible au commit 8 sera quelque chose comme :
//
//	provider.Subscribe(func(evt SwapEvent) {
//	    if evt.Direction == DirectionRWToRO {
//	        _, _ = playerConn.Exec(ctx, "ATTACH '...' AS shared (READ_ONLY)")
//	    }
//	})
//	// + une notification PRE-SWAP (à ajouter au Provider) pour DETACH avant
//	// que AcquireWriter ne tente OpenReadWrite.
func TestPool_AttachShared_SurvivesSwapCycle_integration(t *testing.T) {
	t.Skip(
		"commit 7 — actuellement bloqué par 'Unique file handle conflict'. " +
			"Sera implémenté au commit 8 quand le pool migrera vers Provider " +
			"avec DETACH-pre-swap / REATTACH-post-swap via Subscribe. " +
			"Voir TestPool_AttachSharedConflictsWithSwap pour la régression actuelle.")

	// Code cible — décommentera au commit 8.
}
