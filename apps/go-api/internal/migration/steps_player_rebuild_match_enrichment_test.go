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

// TestRebuildPlayerMatchEnrichmentART_AppendOnlyAfterRebuild : append-only #23046 —
// le rebuild délègue à la conversion append-only. Après rebuild : colonne id présente,
// vue player_match_enrichment_latest queryable, et PLUS de PK(match_id) (les match_id
// dupliqués sont permis — 1 row par stage).
func TestRebuildPlayerMatchEnrichmentART_AppendOnlyAfterRebuild(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)
	if err := RebuildPlayerMatchEnrichmentART(context.Background(), db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if hasID, _ := columnExists(db, "player_match_enrichment", "id"); !hasID {
		t.Error("colonne id absente après rebuild append-only")
	}
	if _, err := db.Exec(`SELECT COUNT(*) FROM player_match_enrichment_latest`); err != nil {
		t.Errorf("vue _latest non queryable après rebuild: %v", err)
	}
	// PLUS de PK(match_id) : un INSERT du même match_id (stage distinct) doit réussir.
	if _, err := db.Exec(
		`INSERT INTO player_match_enrichment (match_id, dominance_flag, stage) VALUES ('pme-0000', 1, 'dominance')`); err != nil {
		t.Errorf("append-only : INSERT match_id dupliqué (stage distinct) devrait réussir, got: %v", err)
	}
}

// TestRebuildPlayerMatchEnrichmentART_PreservesOriginalColumns : append-only #23046 —
// les colonnes d'origine sont PRÉSERVÉES (after ⊇ before) ; le rebuild AJOUTE
// id/stage/written_at + les colonnes canoniques (ensurePMEColumns).
func TestRebuildPlayerMatchEnrichmentART_PreservesOriginalColumns(t *testing.T) {
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
	afterSet := make(map[string]bool, len(after))
	for _, c := range after {
		afterSet[c] = true
	}
	for _, c := range before {
		if !afterSet[c] {
			t.Errorf("colonne d'origine %q perdue après rebuild append-only", c)
		}
	}
	for _, c := range []string{"id", "stage", "written_at"} {
		if !afterSet[c] {
			t.Errorf("colonne append-only %q absente après rebuild", c)
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

// TestRebuildPlayerMatchEnrichmentART_EradicatesARTIndexes : append-only #23046 —
// INVERSION de doctrine (vs le fix 2026-06-19 qui les rejouait). Les 3 ex-index ART
// mutés (idx_pme_engagement_history/_paces/_session) sont ÉRADIQUÉS par le swap
// append-only et ne doivent JAMAIS revenir (seul idx_pme_match_lookup est toléré).
func TestRebuildPlayerMatchEnrichmentART_EradicatesARTIndexes(t *testing.T) {
	db := openMemDB(t)
	seedPlayerMatchEnrichmentForRebuild(t, db)
	ctx := context.Background()

	// Simule une vieille DB portant les 3 index ART.
	for _, s := range []string{
		`ALTER TABLE player_match_enrichment ADD COLUMN engagement_score_brut DOUBLE`,
		`ALTER TABLE player_match_enrichment ADD COLUMN mode_category VARCHAR`,
		`CREATE INDEX idx_pme_engagement_history ON player_match_enrichment(mode_category, engagement_score_brut)`,
		`CREATE INDEX idx_pme_engagement_paces ON player_match_enrichment(mode_category)`,
		`CREATE INDEX idx_pme_session ON player_match_enrichment(session_id)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed index ddl %q: %v", s, err)
		}
	}

	if err := RebuildPlayerMatchEnrichmentART(ctx, db); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// Aucun des 3 index ART ne doit subsister après le rebuild append-only.
	for _, idx := range []string{"idx_pme_engagement_history", "idx_pme_engagement_paces", "idx_pme_session"} {
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, idx).Scan(&n); err != nil {
			t.Fatalf("count index %s: %v", idx, err)
		}
		if n != 0 {
			t.Errorf("index ART %s toujours présent après rebuild (doit être éradiqué)", idx)
		}
	}
}
