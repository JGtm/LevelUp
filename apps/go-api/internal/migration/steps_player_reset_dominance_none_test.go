//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestResetDominanceNone — seuls les flags 0 repassent à NULL dans la vue _latest
// (recalcul post-sync) ; les badges non nuls et les matchs jamais calculés sont
// intacts ; aucune row existante n'est modifiée (append-only) ; rejouable.
func TestResetDominanceNone(t *testing.T) {
	db := setupLegacyMatchEnrichment(t, nil)
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		t.Fatalf("append-only: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, dominance_flag, stage) VALUES
			('m_zero', 0, 'dominance'),
			('m_dom', 1, 'dominance'),
			('m_null', NULL, 'dominance')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := applyResetDominanceNone(db); err != nil {
			t.Fatalf("reset (passe %d): %v", i+1, err)
		}
	}
	want := map[string]sql.NullInt64{
		"m_zero": {},
		"m_dom":  {Int64: 1, Valid: true},
		"m_null": {},
	}
	for id, w := range want {
		var got sql.NullInt64
		if err := db.QueryRow(
			`SELECT dominance_flag FROM player_match_enrichment_latest WHERE match_id = ?`, id,
		).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		if got != w {
			t.Errorf("%s : dominance_flag = %v, attendu %v", id, got, w)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 4 {
		t.Errorf("rows = %d, attendu 4 (3 seeds + 1 reset ; la 2e passe ne réinsère rien)", n)
	}
}
