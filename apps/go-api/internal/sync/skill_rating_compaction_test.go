//go:build integration

// Package sync — skill_rating_compaction_test.go : régression D6-2.
//
// La compaction collapse l'historique de versions append-only de
// match_skill_rank à la plus récente par (match_id, rating_type), bornant la
// croissance après un recompute force (toggle exclusion, rebuild ART).
package sync

import (
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestCompactMatchSkillRankSuperseded(t *testing.T) {
	db := openLUSRDB(t)
	ctx := t.Context()

	insert := func(matchID, rt string, val float64, wat string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, `
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, written_at)
			VALUES (?, ?, ?, ?::TIMESTAMP)`, matchID, rt, val, wat); err != nil {
			t.Fatalf("insert %s/%s: %v", matchID, rt, err)
		}
	}
	// 3 versions LUSR de m1 (force répété) + 1 LUSR m2 + 1 CSR m1 (type distinct).
	insert("m1", "LUSR", 1500, "2025-01-01 10:00:00")
	insert("m1", "LUSR", 1520, "2025-01-01 11:00:00")
	insert("m1", "LUSR", 1540, "2025-01-01 12:00:00") // latest LUSR m1 (id le + grand)
	insert("m2", "LUSR", 1600, "2025-01-01 10:00:00") // latest LUSR m2
	insert("m1", "CSR", 1450, "2025-01-01 09:00:00")  // type distinct → conservé

	var before int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 5 {
		t.Fatalf("setup: %d rows, want 5", before)
	}

	deleted, err := compactMatchSkillRankSuperseded(ctx, db)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2 (les 2 vieilles versions LUSR de m1)", deleted)
	}

	// Reste : latest LUSR m1 (1540), latest LUSR m2 (1600), CSR m1 (1450) = 3.
	var after int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 3 {
		t.Errorf("après compaction = %d rows, want 3", after)
	}

	// La version conservée par (match_id, rating_type) est la plus récente.
	var m1lusr float64
	if err := db.QueryRow(
		`SELECT rating_value FROM match_skill_rank WHERE match_id='m1' AND rating_type='LUSR'`,
	).Scan(&m1lusr); err != nil {
		t.Fatal(err)
	}
	if m1lusr != 1540 {
		t.Errorf("LUSR m1 conservé = %.0f, want 1540 (le plus récent)", m1lusr)
	}

	// La vue latest reste correcte (priorité CSR pour m1).
	var viewRT string
	if err := db.QueryRow(
		`SELECT rating_type FROM match_skill_rank_latest WHERE match_id='m1'`,
	).Scan(&viewRT); err != nil {
		t.Fatal(err)
	}
	if viewRT != "CSR" {
		t.Errorf("vue latest m1 rating_type = %q, want CSR (priorité)", viewRT)
	}

	// Idempotent : 2e compaction ne supprime rien.
	d2, err := compactMatchSkillRankSuperseded(ctx, db)
	if err != nil {
		t.Fatalf("2e compact: %v", err)
	}
	if d2 != 0 {
		t.Errorf("2e compaction deleted = %d, want 0 (idempotent)", d2)
	}
}
