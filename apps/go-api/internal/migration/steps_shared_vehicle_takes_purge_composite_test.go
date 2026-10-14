package migration

// steps_shared_vehicle_takes_purge_composite_test.go — shared_purge_composite_vehicle_takes_v1
// retire les seules lignes à match_id composite, sans toucher aux autres ni au schéma.

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedVehicleTakes(t *testing.T, withComposite bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := applyMatchVehicleTakes(db); err != nil {
		t.Fatalf("création: %v", err)
	}
	ins := `INSERT INTO match_vehicle_takes (match_id, decode_pass, written_at, row_kind, camp, xuid,
		family, takes, aboard_ms, episodes, proximity_episodes, frags, measured, unmeasured_reason,
		doc_schema, episodes_read, episodes_unnamed, episodes_no_camp, frags_read, frags_reason,
		frags_total, frags_unmatched)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, 1000, 1, 0, 0, TRUE, '', 71, 1, 0, 0, TRUE, '', 0, 0)`
	rows := [][]any{
		{"11111111-aaaa-4000-8000-000000000001", "p1", "2026-10-01 01:00:00", "match", -1, "", ""},
		{"11111111-aaaa-4000-8000-000000000001", "p2", "2026-10-02 01:00:00", "match", -1, "", ""},
		{"11111111-aaaa-4000-8000-000000000001", "p2", "2026-10-02 01:00:00", "take", 0, "x1", "warthog"},
	}
	if withComposite {
		rows = append(rows,
			[]any{"fccc61cd,879a4dba", "p9", "2026-10-01 02:37:00", "match", -1, "", ""},
			[]any{"fccc61cd,879a4dba", "p9", "2026-10-01 02:37:00", "take", 1, "x2", "falcon"})
	}
	for _, r := range rows {
		if _, err := db.Exec(ins, r...); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	return db
}

func TestPurgeCompositeVehicleTakes_RetireLesSeulesLignesComposites(t *testing.T) {
	db := seedVehicleTakes(t, true)
	ctx := t.Context()
	ddl, err := ddlDeTable(ctx, db, vehicleTakesTable)
	if err != nil {
		t.Fatal(err)
	}
	index, err := ddlDesIndex(ctx, db, vehicleTakesTable)
	if err != nil {
		t.Fatal(err)
	}
	attendue, err := empreinteRestreinte(ctx, db, vehicleTakesView, vehicleTakesKeepClause)
	if err != nil {
		t.Fatal(err)
	}

	if err := applyPurgeCompositeVehicleTakes(db); err != nil {
		t.Fatalf("purge: %v", err)
	}

	if n := scanInt(t, db, `SELECT COUNT(*) FROM match_vehicle_takes WHERE strpos(match_id, ',') > 0`); n != 0 {
		t.Errorf("%d ligne(s) composite(s) restantes", n)
	}
	if n := scanInt(t, db, `SELECT COUNT(*) FROM match_vehicle_takes`); n != 3 {
		t.Errorf("lignes gardées = %d, attendu 3 (les deux passes du vrai match)", n)
	}
	if s := scanInt(t, db, `SELECT SUM(id) FROM match_vehicle_takes`); s != 6 {
		t.Errorf("somme des id gardés = %d, attendu 6 (ids 1,2,3 conservés)", s)
	}
	if err := verifierSchemaIdentique(ctx, db, vehicleTakesTable, ddl, index); err != nil {
		t.Errorf("schéma modifié : %v", err)
	}
	if apres, err := empreinteDeVue(ctx, db, vehicleTakesView); err != nil || apres != attendue {
		t.Errorf("vue après purge %+v (err %v), attendu %+v", apres, err, attendue)
	}
	// Séquence intacte : l'id suivant continue après le plus grand jamais tiré (5).
	var next int64
	if err := db.QueryRow(`INSERT INTO match_vehicle_takes (match_id, decode_pass, row_kind, camp, xuid,
		family, takes, aboard_ms, episodes, proximity_episodes, frags, measured, unmeasured_reason,
		doc_schema, episodes_read, episodes_unnamed, episodes_no_camp, frags_read, frags_reason,
		frags_total, frags_unmatched) VALUES ('m', 'p', 'match', -1, '', '', 0, 0, 0, 0, 0, TRUE, '',
		71, 0, 0, 0, TRUE, '', 0, 0) RETURNING id`).Scan(&next); err != nil {
		t.Fatalf("insert après purge: %v", err)
	}
	if next != 6 {
		t.Errorf("id suivant = %d, attendu 6 (séquence non touchée)", next)
	}
	if err := applyPurgeCompositeVehicleTakes(db); err != nil {
		t.Errorf("second passage: %v", err)
	}
}

func TestPurgeCompositeVehicleTakes_BaseSaineNonTouchee(t *testing.T) {
	db := seedVehicleTakes(t, false)
	before := scanInt(t, db, `SELECT COUNT(*) FROM match_vehicle_takes`)
	oid := tableOID(t, db, vehicleTakesTable)
	if err := applyPurgeCompositeVehicleTakes(db); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if got := tableOID(t, db, vehicleTakesTable); got != oid {
		t.Errorf("base saine reconstruite (oid %d -> %d) : aucun swap attendu", oid, got)
	}
	if after := scanInt(t, db, `SELECT COUNT(*) FROM match_vehicle_takes`); after != before {
		t.Errorf("base saine : %d lignes avant, %d après", before, after)
	}
}
