//go:build integration

// POC isolé pour diagnostiquer la limitation driver DuckDB-Go découverte
// au commit 8f WIP.
//
// Question fondamentale : peut-on faire OpenReadWrite sur un fichier qui
// vient d'être Closed depuis un autre handle dans le même process ?
//
// 4 scénarios testés, du plus simple au plus complexe :
//
//	S1 : Open RO → Close → Open RW (sanity check)
//	S2 : Open RO #1 (kept open) → Open RO #2 (cache hit) → Close #2 → Open RW
//	S3 : RO + ATTACH RW dans player → Close + Reopen player → Open RW
//	S4 : Comme S3 mais avec un Sleep pour laisser le driver C purger
package duckdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// preparePOCShared crée un fichier shared.duckdb avec schéma et ferme la
// conn de bootstrap. Retourne le path.
func preparePOCShared(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	boot, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("bootstrap OpenReadWrite: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(boot.SQLDb()); err != nil {
		_ = boot.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := boot.Close(); err != nil {
		t.Fatalf("bootstrap Close: %v", err)
	}
	return sharedPath
}

// TestPOCSwap_S1_OpenROCloseOpenRW vérifie le scénario le plus basique :
// après avoir fermé proprement une conn RO, peut-on ouvrir RW ?
func TestPOCSwap_S1_OpenROCloseOpenRW(t *testing.T) {
	sharedPath := preparePOCShared(t)
	ctx := context.Background()

	// Open RO via sql.Open direct (pas via duckdb.OpenReadOnly qui utilise un cache).
	roDB, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("S1 sql.Open RO: %v", err)
	}
	var v string
	if err := roDB.QueryRowContext(ctx, "SELECT version()").Scan(&v); err != nil {
		t.Fatalf("S1 ping RO: %v", err)
	}
	if err := roDB.Close(); err != nil {
		t.Fatalf("S1 Close RO: %v", err)
	}

	// Maintenant Open RW.
	rwDB, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatalf("S1 sql.Open RW: %v", err)
	}
	defer func() { _ = rwDB.Close() }()

	if _, err := rwDB.ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES ('s1', NOW())"); err != nil {
		t.Errorf("S1 INSERT RW after RO close: %v", err)
	} else {
		t.Logf("S1 OK : RW après RO fermé fonctionne")
	}
}

// TestPOCSwap_S2_TwoROHandlesCache vérifie le scénario du cache process-level
// de duckdb : 2 conn RO sur le même fichier → 1 close → 1 reste → OpenRW ?
func TestPOCSwap_S2_TwoROHandlesCache(t *testing.T) {
	sharedPath := preparePOCShared(t)

	roDB1, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		t.Fatalf("S2 OpenReadOnly #1: %v", err)
	}
	defer func() { _ = roDB1.Close() }()

	roDB2, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		t.Fatalf("S2 OpenReadOnly #2 (cache hit): %v", err)
	}

	// Close #2 : refCount 2→1.
	if err := roDB2.Close(); err != nil {
		t.Errorf("S2 Close #2: %v", err)
	}

	// Open RW : devrait FAIL car #1 tient encore.
	_, err = duckdb.OpenReadWrite(sharedPath)
	if err == nil {
		t.Error("S2 OpenReadWrite a réussi alors qu'une RO existe (cache refCount=1)")
	} else if !strings.Contains(err.Error(), "different configuration") &&
		!strings.Contains(err.Error(), "Unique file handle") {
		t.Errorf("S2 erreur attendue different config / unique file handle, obtenu : %v", err)
	} else {
		t.Logf("S2 OK : OpenReadWrite refusé tant qu'une RO existe — comme attendu")
	}

	// Maintenant Close #1 aussi (refCount 1→0).
	if err := roDB1.Close(); err != nil {
		t.Errorf("S2 Close #1: %v", err)
	}

	// Re-tente Open RW.
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Errorf("S2 OpenReadWrite après Close #1 (refCount=0): %v", err)
	} else {
		t.Logf("S2 OK : OpenReadWrite réussit après Close de toutes les RO")
		_ = rwDB.Close()
	}
}

// TestPOCSwap_S3_PlayerWithAttach vérifie le scénario réel : une conn player
// RW avec ATTACH RO sur shared. Reopen player + Close shared. OpenRW ?
func TestPOCSwap_S3_PlayerWithAttach(t *testing.T) {
	sharedPath := preparePOCShared(t)
	dir := filepath.Dir(sharedPath)
	playerPath := filepath.Join(dir, "player.duckdb")
	ctx := context.Background()

	// Player en RW.
	playerDB, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("S3 player OpenReadWrite: %v", err)
	}
	defer func() { _ = playerDB.Close() }()

	// Shared en RO.
	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		t.Fatalf("S3 shared OpenReadOnly: %v", err)
	}

	// ATTACH RO de shared sur player.
	if _, err := playerDB.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)); err != nil {
		t.Fatalf("S3 ATTACH: %v", err)
	}

	// Maintenant : Reopen player (libère ATTACH) puis Close shared.
	if err := playerDB.Reopen(); err != nil {
		t.Fatalf("S3 Reopen player: %v", err)
	}
	if err := sharedDB.Close(); err != nil {
		t.Errorf("S3 Close shared: %v", err)
	}

	// Tenter OpenReadWrite. EXPECTED FAIL — c'est l'enseignement du POC :
	// Reopen ne libère pas l'ATTACH côté driver. C'est ce qui motive la
	// stratégie DETACH dans PrepareForSharedSwap. On ASSERT la présence
	// de l'erreur pour ancrer cette limitation dans la suite (pattern T1).
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if rwDB != nil {
		_ = rwDB.Close()
	}
	if err == nil {
		t.Fatal("S3 attendait une erreur Reopen-ne-libère-pas-ATTACH, mais OpenReadWrite a réussi. " +
			"Si ce test échoue, c'est que duckdb-go a corrigé la sémantique de Reopen " +
			"— on peut simplifier PrepareForSharedSwap (retirer DETACH explicite).")
	}
	if !strings.Contains(err.Error(), "Unique file handle") {
		t.Errorf("S3 erreur attendue contenant 'Unique file handle', obtenu : %v", err)
	}
	t.Logf("S3 OK (documente la limitation Reopen-ne-libère-pas-ATTACH) : %v", err)
}

// TestPOCSwap_S4_PlayerWithAttachAndSleep teste si un Sleep après Close
// permet au driver C de purger ses caches internes.
func TestPOCSwap_S4_PlayerWithAttachAndSleep(t *testing.T) {
	sharedPath := preparePOCShared(t)
	dir := filepath.Dir(sharedPath)
	playerPath := filepath.Join(dir, "player.duckdb")
	ctx := context.Background()

	playerDB, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("S4 player OpenReadWrite: %v", err)
	}
	defer func() { _ = playerDB.Close() }()

	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		t.Fatalf("S4 shared OpenReadOnly: %v", err)
	}

	if _, err := playerDB.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)); err != nil {
		t.Fatalf("S4 ATTACH: %v", err)
	}

	if err := playerDB.Reopen(); err != nil {
		t.Fatalf("S4 Reopen player: %v", err)
	}
	if err := sharedDB.Close(); err != nil {
		t.Errorf("S4 Close shared: %v", err)
	}

	// Sleep pour laisser le driver C purger.
	time.Sleep(500 * time.Millisecond)

	// EXPECTED FAIL — Sleep ne suffit pas, la limitation Reopen est
	// structurelle, pas temporaire. Documente que le cache C de DuckDB-Go
	// n'est pas purgé par un simple wait.
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if rwDB != nil {
		_ = rwDB.Close()
	}
	if err == nil {
		t.Fatal("S4 attendait que Sleep ne suffise pas — mais OpenReadWrite a réussi. " +
			"Hypothèse cache-C invalidée, à investiguer.")
	}
	t.Logf("S4 OK (documente que Sleep ne libère pas non plus) : %v", err)
}

// TestPOCSwap_S5_DetachExplicit vérifie un DETACH explicite côté player
// avant le Reopen. Peut-être que DuckDB-Go propage l'ATTACH au cache C
// même après Reopen, et un DETACH explicite est nécessaire.
func TestPOCSwap_S5_DetachExplicit(t *testing.T) {
	sharedPath := preparePOCShared(t)
	dir := filepath.Dir(sharedPath)
	playerPath := filepath.Join(dir, "player.duckdb")
	ctx := context.Background()

	playerDB, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("S5 player OpenReadWrite: %v", err)
	}
	defer func() { _ = playerDB.Close() }()

	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		t.Fatalf("S5 shared OpenReadOnly: %v", err)
	}

	if _, err := playerDB.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)); err != nil {
		t.Fatalf("S5 ATTACH: %v", err)
	}

	// DETACH explicite AVANT Close + Reopen.
	if _, err := playerDB.Exec(ctx, "DETACH shared"); err != nil {
		t.Logf("S5 DETACH explicite échoué (peut être normal) : %v", err)
	}

	if err := sharedDB.Close(); err != nil {
		t.Errorf("S5 Close shared: %v", err)
	}

	// PAS de Reopen — on garde la conn player ouverte avec DETACH appliqué.
	rwDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Errorf("S5 OpenReadWrite après DETACH explicite + Close shared: %v", err)
	} else {
		t.Logf("S5 OK : DETACH explicite suffit à libérer shared")
		_ = rwDB.Close()
	}
}
