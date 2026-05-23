//go:build integration

// Tests TDD pour RebuildPlayerMatchEnrichmentART (Phase 4.1 follow-up
// 2026-05-23). Pattern identique aux tests de RebuildMatchParticipantsART
// (steps_shared_runtime_art_rebuild_test.go) mais sur la table
// player_match_enrichment (PK simple match_id, 8 colonnes).

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedPlayerMatchEnrichmentForRebuild(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_signature VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	for i := 0; i < 15; i++ {
		mid := fmt.Sprintf("pme-%04d", i)
		if _, err := db.Exec(`
			INSERT INTO player_match_enrichment
				(match_id, performance_score, session_id, session_label,
				 is_with_friends, teammates_signature, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			mid, float64(50+i), fmt.Sprintf("s%d", i/3),
			fmt.Sprintf("session %d", i/3), i%2 == 0,
			fmt.Sprintf("sig%d", i)); err != nil {
			t.Fatalf("insert pme[%d]: %v", i, err)
		}
	}
}

// TestRebuildPlayerMatchEnrichmentART_PreservesAllRows : 15 rows seed →
// 15 rows après. Pas de perte sur le swap CTAS.
func TestRebuildPlayerMatchEnrichmentART_PreservesAllRows(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)

	ctx := context.Background()
	var before int
	db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&before)
	if before != 15 {
		t.Fatalf("seed expected 15 rows, got %d", before)
	}

	if err := RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	var after int
	db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&after)
	if after != before {
		t.Errorf("rows changé : before=%d after=%d", before, after)
	}
}

// TestRebuildPlayerMatchEnrichmentART_RecreatesPrimaryKey : la PK est
// reconstruite (INSERT dupliqué doit fail).
func TestRebuildPlayerMatchEnrichmentART_RecreatesPrimaryKey(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)
	if err := RebuildPlayerMatchEnrichmentART(context.Background(), db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	_, dupErr := db.Exec(`
		INSERT INTO player_match_enrichment
			(match_id, performance_score, session_id, session_label,
			 is_with_friends, teammates_signature, created_at, updated_at)
		VALUES ('pme-0000', 0, 's', 'l', FALSE, 'sig', NOW(), NOW())`)
	if dupErr == nil {
		t.Error("PK absente : INSERT dupliqué a réussi")
	}
}

// TestRebuildPlayerMatchEnrichmentART_PreservesColumns : ordre et nombre
// identiques.
func TestRebuildPlayerMatchEnrichmentART_PreservesColumns(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)

	ctx := context.Background()
	before, err := loadTableColumns(ctx, db, "player_match_enrichment")
	if err != nil {
		t.Fatalf("cols before: %v", err)
	}
	if err := RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	after, err := loadTableColumns(ctx, db, "player_match_enrichment")
	if err != nil {
		t.Fatalf("cols after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("cols count : before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("col[%d] : before=%s after=%s", i, before[i], after[i])
		}
	}
}

// TestRebuildPlayerMatchEnrichmentART_NoTableNoOp : pas de crash si la
// table est absente (player DB jamais initialisée).
func TestRebuildPlayerMatchEnrichmentART_NoTableNoOp(t *testing.T) {
	db := openMemDB(t)
	if err := RebuildPlayerMatchEnrichmentART(context.Background(), db); err != nil {
		t.Fatalf("rebuild (no table): %v", err)
	}
}

// TestRebuildPlayerMatchEnrichmentART_Idempotent : 2 appels successifs
// produisent le même résultat.
func TestRebuildPlayerMatchEnrichmentART_Idempotent(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)
	ctx := context.Background()

	if err := RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		t.Fatalf("rebuild 1: %v", err)
	}
	var n1 int
	db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n1)

	if err := RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		t.Fatalf("rebuild 2: %v", err)
	}
	var n2 int
	db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n2)

	if n1 != n2 {
		t.Errorf("idempotence violée : n1=%d n2=%d", n1, n2)
	}
}
