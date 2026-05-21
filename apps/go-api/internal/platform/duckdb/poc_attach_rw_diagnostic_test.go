//go:build integration

// POC diagnostique commit 8j+ : peut-on ATTACHer shared en READ_WRITE
// depuis la conn player (qui est elle-même RW), permettant au sync engine
// d'écrire directement via cette conn — sans avoir besoin du swap RW
// séparé qui crée la fenêtre de Catalog Errors ?
//
// Si ce POC PASS, c'est une solution radicalement plus simple que B3 :
// pas de Provider, pas de Subscribe, pas de DETACH, pas de Catalog Error.
package duckdb_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// TestPOC_AttachRWFromPlayerConn vérifie si on peut écrire dans shared.*
// via une conn player après un ATTACH explicite SANS READ_ONLY.
func TestPOC_AttachRWFromPlayerConn(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player.duckdb")
	ctx := context.Background()

	// Setup shared avec schéma.
	boot, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("bootstrap shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), boot.SQLDb()); err != nil {
		_ = boot.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := boot.Close(); err != nil {
		t.Fatalf("bootstrap Close: %v", err)
	}

	// Player en RW (comme le pool).
	playerDB, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("player RW: %v", err)
	}
	defer func() { _ = playerDB.Close() }()

	// ATTACH RW (sans READ_ONLY) sur la conn player.
	attachStmt := fmt.Sprintf("ATTACH '%s' AS shared", sharedPath)
	if _, err := playerDB.Exec(ctx, attachStmt); err != nil {
		t.Fatalf("ATTACH RW: %v", err)
	}

	// Tentative d'INSERT via la conn player dans shared.match_registry.
	if _, err := playerDB.Exec(ctx,
		"INSERT INTO shared.match_registry (match_id, start_time) VALUES ('poc-rw-1', NOW())"); err != nil {
		t.Fatalf("INSERT via player conn dans shared.* : %v (ATTACH RW ne permet pas l'écriture ?)", err)
	}

	// Vérifier le SELECT.
	var count int
	if err := playerDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM shared.match_registry WHERE match_id = 'poc-rw-1'").Scan(&count); err != nil {
		t.Fatalf("SELECT post-INSERT: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, attendu 1", count)
	} else {
		t.Logf("POC OK : ATTACH RW + INSERT via conn player fonctionne — voie radicalement plus simple que B3")
	}
}

// TestPOC_AttachRWWithSecondConnRO vérifie qu'avec une conn player ATTACH RW
// shared, une autre conn (handler HTTP par ex) peut faire un OpenReadOnly
// shared en parallèle (cas typique : 2 joueurs ouvrent shared simultanément).
func TestPOC_AttachRWWithSecondConnRO(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player.duckdb")
	ctx := context.Background()

	boot, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("bootstrap shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), boot.SQLDb()); err != nil {
		_ = boot.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = boot.Close()

	playerDB, err := duckdb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("player RW: %v", err)
	}
	defer func() { _ = playerDB.Close() }()

	if _, err := playerDB.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared", sharedPath)); err != nil {
		t.Fatalf("ATTACH RW: %v", err)
	}

	// Maintenant tenter OpenReadOnly sur shared depuis ailleurs (=
	// simule main.go boot ou autre joueur).
	roDB, err := duckdb.OpenReadOnly(sharedPath)
	if err == nil {
		_ = roDB.Close()
		t.Logf("POC OK : OpenReadOnly réussit même avec ATTACH RW actif côté player")
	} else {
		t.Logf("POC ATTENTION : OpenReadOnly refusé avec ATTACH RW player : %v", err)
	}
}

// TestPOC_AttachRWWithSecondPlayerConn vérifie le cas multi-joueur : 2 conns
// player distinctes essayent toutes les deux d'attach shared en RW. Si ça
// échoue, l'approche ATTACH RW ne marche pas en multi-joueur.
func TestPOC_AttachRWWithSecondPlayerConn(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath1 := filepath.Join(dir, "player1.duckdb")
	playerPath2 := filepath.Join(dir, "player2.duckdb")
	ctx := context.Background()

	boot, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("bootstrap shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), boot.SQLDb()); err != nil {
		_ = boot.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = boot.Close()

	player1, err := duckdb.OpenReadWrite(playerPath1)
	if err != nil {
		t.Fatalf("player1 RW: %v", err)
	}
	defer func() { _ = player1.Close() }()

	player2, err := duckdb.OpenReadWrite(playerPath2)
	if err != nil {
		t.Fatalf("player2 RW: %v", err)
	}
	defer func() { _ = player2.Close() }()

	// Player1 ATTACH RW shared.
	if _, err := player1.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared", sharedPath)); err != nil {
		t.Fatalf("player1 ATTACH RW: %v", err)
	}

	// Player2 ATTACH RW shared (DEUXIÈME conn dans le process).
	if _, err := player2.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared", sharedPath)); err != nil {
		t.Logf("POC NOTE : 2e ATTACH RW depuis player2 échoue : %v", err)
		// On essaie un fallback : ATTACH RO sur player2 (puisque player1 a déjà RW).
		// POC informatif uniquement — le résultat documente le comportement
		// driver, pas un contrat. Pas de t.Errorf si ça échoue.
		if _, err := player2.Exec(ctx,
			fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath)); err != nil {
			t.Logf("POC NOTE : fallback ATTACH RO sur player2 échoue aussi : %v", err)
			t.Logf("POC CONCLUSION : ATTACH RW multi-conn impossible — confirme que B1 reste la voie unique")
			return // Skip le reste du test, pas pertinent.
		}
		t.Logf("POC OK : ATTACH RO fonctionne sur player2 quand player1 tient RW")
	} else {
		t.Logf("POC OK : 2 conns player avec ATTACH RW shared coexistent !")
	}

	// Player1 INSERT
	if _, err := player1.Exec(ctx,
		"INSERT INTO shared.match_registry (match_id, start_time) VALUES ('poc-mp-1', NOW())"); err != nil {
		t.Fatalf("INSERT via player1: %v", err)
	}

	// Player2 SELECT (devrait voir l'INSERT — MVCC partagé via ATTACH multi-conn)
	var count int
	if err := player2.QueryRow(ctx,
		"SELECT COUNT(*) FROM shared.match_registry WHERE match_id = 'poc-mp-1'").Scan(&count); err != nil {
		t.Logf("POC NOTE : player2 SELECT post-INSERT player1 : %v", err)
	} else if count != 1 {
		t.Logf("POC NOTE : player2 voit %d rows (attendu 1) — visibility MVCC à vérifier", count)
	} else {
		t.Logf("POC OK : player2 voit l'INSERT player1 immédiatement (MVCC cross-conn fonctionne)")
	}
}
