//go:build integration

// Tests pour le sync.Once process-level sur attachGlobalXuidAliases
// (Phase 3 plan stabilisation 2026-05-22).
//
// Objectif : valider que des appels multiples à attachGlobalXuidAliases pour
// le même globalPath ne déclenchent qu'une seule fois l'ATTACH côté DuckDB
// (sinon "Unique file handle conflict" — cf. AUDIT_DUCKDB_ATTACH_2026-05-21 §1).
package duckdb

import (
	"context"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestAttachGlobal_IdempotentMultipleCalls : 2 appels successifs sur la même
// path produisent au plus 1 erreur (la 1ère persiste, les suivantes la
// retournent depuis sync.Once.Do).
func TestAttachGlobal_IdempotentMultipleCalls(t *testing.T) {
	// Reset état pour isoler ce test.
	resetGlobalAttachState()

	tmp := t.TempDir()
	globalPath := filepath.Join(tmp, "xbox_aliases.duckdb")
	playerPath := filepath.Join(tmp, "player_stats.duckdb")

	playerDB, err := OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = playerDB.Close() })

	ctx := context.Background()

	// 1er appel : doit créer la global DB + attacher.
	err1 := attachGlobalXuidAliases(ctx, playerDB, globalPath)
	if err1 != nil {
		t.Fatalf("1er attach: %v", err1)
	}

	// 2e appel : doit être no-op via sync.Once.
	err2 := attachGlobalXuidAliases(ctx, playerDB, globalPath)
	if err2 != nil {
		t.Errorf("2e attach (devrait être no-op): %v", err2)
	}

	// 3e appel : idem.
	err3 := attachGlobalXuidAliases(ctx, playerDB, globalPath)
	if err3 != nil {
		t.Errorf("3e attach (devrait être no-op): %v", err3)
	}

	// Vérifier que `global.xuid_aliases` est accessible.
	var count int
	if err := playerDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM global.xuid_aliases").Scan(&count); err != nil {
		t.Errorf("query global.xuid_aliases après attach: %v", err)
	}
}

// TestAttachGlobal_DifferentDBsSamePath : plusieurs *DB ouvrent la même
// global path — sync.Once garantit qu'une seule tentative d'ATTACH est faite,
// même si les *DB diffèrent.
func TestAttachGlobal_DifferentDBsSamePath(t *testing.T) {
	resetGlobalAttachState()

	tmp := t.TempDir()
	globalPath := filepath.Join(tmp, "xbox_aliases.duckdb")
	player1Path := filepath.Join(tmp, "player1_stats.duckdb")
	player2Path := filepath.Join(tmp, "player2_stats.duckdb")

	db1, err := OpenReadWrite(player1Path)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	t.Cleanup(func() { _ = db1.Close() })
	db2, err := OpenReadWrite(player2Path)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	t.Cleanup(func() { _ = db2.Close() })

	ctx := context.Background()

	// Le 1er attach (sur db1) crée la global DB et attache.
	if err := attachGlobalXuidAliases(ctx, db1, globalPath); err != nil {
		t.Fatalf("db1 attach: %v", err)
	}
	// Le 2e attach (sur db2) devrait être no-op via sync.Once.
	// Avant le fix Phase 3 : "Unique file handle conflict" car ATTACH process-wide.
	if err := attachGlobalXuidAliases(ctx, db2, globalPath); err != nil {
		t.Errorf("db2 attach après db1 sur même global path: %v (régression sync.Once)", err)
	}
}
