//go:build integration

// Tests TDD pour la fonction runtime RebuildMatchParticipantsART (Phase 4.1
// du plan stabilisation 2026-05-22). Cf. .ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md §4.1.
//
// Contrat de la fonction runtime (≠ migration applyRebuildMatchParticipants) :
//   - Pas de sentinel sync_meta : la fonction est IDEMPOTENTE PAR DESIGN
//     (peut être rappelée à chaque boot ou même périodiquement)
//   - Préserve toutes les rows (CTAS SELECT *)
//   - Préserve toutes les colonnes (PRAGMA table_info)
//   - Recrée la PRIMARY KEY (match_id, xuid)
//   - Recrée les 6 indexes idx_mp_*
//   - Recrée les vues dépendantes (v_gamertag_lookup, mv_player_matches, etc.)
//   - No-op gracieux si table absente
//
// **Mode TDD strict** (directive utilisateur) : ces tests définissent le
// contrat AVANT toute extraction. Ils doivent passer immédiatement après
// l'extraction de `RebuildMatchParticipantsART` depuis `applyRebuildMatchParticipants`.

package migration

import (
	"context"
	"database/sql"
	"testing"
)

// TestRebuildMatchParticipantsART_Idempotent_NoSentinel : appel 2x successif
// → tableau stable, pas d'erreur, pas de sentinel (contrairement à la
// version migration). Critère le plus important : pouvoir être rappelée
// au runtime quand BootARTGuard détecte une nouvelle divergence.
func TestRebuildMatchParticipantsART_Idempotent_NoSentinel(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	ctx := context.Background()

	// 1er appel : succès, rows préservés.
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild 1: %v", err)
	}
	var rows1 int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rows1); err != nil {
		t.Fatalf("count after rebuild 1: %v", err)
	}
	if rows1 != 10 {
		t.Errorf("rebuild 1: count = %d, want 10", rows1)
	}

	// 2e appel : succès aussi (idempotent par design, pas de no-op via sentinel).
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild 2: %v", err)
	}
	var rows2 int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rows2); err != nil {
		t.Fatalf("count after rebuild 2: %v", err)
	}
	if rows2 != rows1 {
		t.Errorf("rebuild 2: count = %d, want %d (stable)", rows2, rows1)
	}

	// Sentinel doit être ABSENT (la fonction runtime ne le pose pas).
	var marker sql.NullString
	err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = ?`,
		matchParticipantsRebuildMetaKey).Scan(&marker)
	if err == nil && marker.Valid {
		t.Errorf("la version runtime ne doit PAS poser le sentinel — got value=%q", marker.String)
	}
}

// TestRebuildMatchParticipantsART_RecreatesPrimaryKey : la PK est bien recréée
// après chaque appel (pas seulement le 1er).
func TestRebuildMatchParticipantsART_RecreatesPrimaryKey(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	ctx := context.Background()
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// INSERT dupliqué doit échouer (PK active).
	_, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists, created_at)
		VALUES ('match_alpha', 'match_alpha_xuid_1', 'dup', 0, 2, 1, 100, 1, 1, 1, NOW())
	`)
	if err == nil {
		t.Fatal("PK absente : INSERT dupliqué a réussi alors qu'il aurait dû violer la PK")
	}
}

// TestRebuildMatchParticipantsART_PreservesColumns : ordre et nombre identiques.
func TestRebuildMatchParticipantsART_PreservesColumns(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	ctx := context.Background()
	before, err := loadMatchParticipantsColumns(ctx, db)
	if err != nil {
		t.Fatalf("columns before: %v", err)
	}

	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	after, err := loadMatchParticipantsColumns(ctx, db)
	if err != nil {
		t.Fatalf("columns after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("expected %d columns, got %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("column[%d] mismatch: before=%s after=%s", i, before[i], after[i])
		}
	}
}

// TestRebuildMatchParticipantsART_NoTableNoOp : pas de crash si table absente.
// Contrairement à la version migration, on ne pose pas de sentinel.
func TestRebuildMatchParticipantsART_NoTableNoOp(t *testing.T) {
	db := openMemDB(t)
	// Pas de table match_participants — la fonction doit retourner sans erreur.
	ctx := context.Background()
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild (no table): %v", err)
	}
}

// TestRebuildMatchParticipantsART_RecreatesViews : applyResolutionViews est
// rappelée → v_gamertag_lookup existe post-rebuild.
func TestRebuildMatchParticipantsART_RecreatesViews(t *testing.T) {
	db := openMemDB(t)
	seedMatchParticipantsForRebuild(t, db)

	ctx := context.Background()
	if err := RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// v_gamertag_lookup doit exister (créée par applyResolutionViews).
	var viewCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.views
		WHERE table_schema = 'main' AND table_name = 'v_gamertag_lookup'
	`).Scan(&viewCount)
	if err != nil {
		t.Fatalf("query views: %v", err)
	}
	if viewCount != 1 {
		t.Errorf("v_gamertag_lookup absent post-rebuild (count=%d)", viewCount)
	}
}
