package migration

// highlight_events_dedupe_test.go — ApplyHighlightEventsDedupe retire les seules lignes écrites deux
// fois (forme sans code doublée, match recopié en bloc), garde les répétitions légitimes, les
// lignes sans jumeau et le schéma, et ne réécrit rien sur une base saine.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openHighlightEventsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := ApplyHighlightEventsAutoincrement(db); err != nil {
		t.Fatalf("création: %v", err)
	}
	return db
}

type heRow struct {
	match, typ string
	ms         int
	xuid       string
	hint       any
	raw        any
}

func insertHE(t *testing.T, db *sql.DB, rows ...heRow) {
	t.Helper()
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint, raw_json)
			VALUES (?, ?, ?, ?, ?, ?)`, r.match, r.typ, r.ms, r.xuid, r.hint, r.raw); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func countHE(t *testing.T, db *sql.DB, match string) int64 {
	t.Helper()
	return scanInt(t, db, "SELECT COUNT(*) FROM highlight_events WHERE match_id = '"+match+"'")
}

// seedHighlightEventsDoubled pose cinq matchs :
//
//	bare  : forme sans code puis forme avec code (deux passes), dont deux médailles identiques
//	        légitimes au même instant dans chaque passe ;
//	copy  : tout le match recopié en bloc, dont une médaille légitimement répétée (4 lignes) ;
//	clean : un match sain avec deux médailles identiques légitimes et des lignes uniques ;
//	solo  : forme sans code SANS jumeau (seule trace du fait) ;
//	pair  : deux lignes identiques et rien d'autre, sans élimination (pas un match recopié).
func seedHighlightEventsDoubled(t *testing.T) *sql.DB {
	t.Helper()
	db := openHighlightEventsDB(t)
	medal := `{"medal_name":"Kong"}`
	insertHE(t, db,
		heRow{"bare", "kill", 1000, "x1", nil, nil}, heRow{"bare", "kill", 2000, "x2", nil, nil},
		heRow{"bare", "medal", 3000, "x1", nil, nil}, heRow{"bare", "medal", 3000, "x1", nil, nil},
		heRow{"bare", "mode", 4000, "x1", nil, nil},
		heRow{"bare", "kill", 1000, "x1", 50, nil}, heRow{"bare", "kill", 2000, "x2", 50, nil},
		heRow{"bare", "medal", 3000, "x1", 50, medal}, heRow{"bare", "medal", 3000, "x1", 50, medal},
		heRow{"bare", "mode", 4000, "x1", 10, nil})
	for i := 0; i < 2; i++ {
		insertHE(t, db,
			heRow{"copy", "kill", 1000, "x1", 50, nil}, heRow{"copy", "death", 1002, "x2", 1, nil},
			heRow{"copy", "medal", 3000, "x1", 50, medal}, heRow{"copy", "medal", 3000, "x1", 50, medal})
	}
	insertHE(t, db,
		heRow{"clean", "kill", 1000, "x1", 50, nil}, heRow{"clean", "kill", 5000, "x2", 50, nil},
		heRow{"clean", "medal", 3000, "x1", 50, medal}, heRow{"clean", "medal", 3000, "x1", 50, medal})
	insertHE(t, db, heRow{"solo", "kill", 1000, "x1", nil, nil}, heRow{"solo", "kill", 2000, "x2", nil, nil})
	insertHE(t, db, heRow{"pair", "mode", 700, "x1", 10, nil}, heRow{"pair", "mode", 700, "x1", 10, nil})
	return db
}

func TestApplyHighlightEventsDedupe_RetireLesSeulsDoublons(t *testing.T) {
	db := seedHighlightEventsDoubled(t)
	ctx := t.Context()
	ddl, err := ddlDeTable(ctx, db, "highlight_events")
	if err != nil {
		t.Fatal(err)
	}
	index, err := ddlDesIndex(ctx, db, "highlight_events")
	if err != nil {
		t.Fatal(err)
	}
	avant := map[string]int64{}
	for _, m := range []string{"bare", "copy", "clean", "solo", "pair"} {
		avant[m] = countHE(t, db, m)
	}
	idsClean := scanInt(t, db, "SELECT SUM(id) FROM highlight_events WHERE match_id = 'clean'")

	rep, err := DetectHighlightEventsDuplicates(ctx, db)
	if err != nil {
		t.Fatalf("détection: %v", err)
	}
	if rep.Bare != 5 || rep.Recopied != 4 || rep.Matches != 2 {
		t.Errorf("détection = %+v, attendu 5 lignes sans code, 4 recopiées, 2 matchs", rep)
	}
	if got := scanInt(t, db, "SELECT COUNT(*) FROM highlight_events"); got != rep.Before {
		t.Errorf("la détection a écrit : %d lignes, %d avant", got, rep.Before)
	}

	if err := ApplyHighlightEventsDedupe(db); err != nil {
		t.Fatalf("dédoublonnage: %v", err)
	}

	if got := countHE(t, db, "bare"); got != 5 {
		t.Errorf("bare : %d lignes, attendu 5", got)
	}
	if n := scanInt(t, db, "SELECT COUNT(*) FROM highlight_events WHERE match_id = 'bare' AND type_hint IS NULL"); n != 0 {
		t.Errorf("bare : %d ligne(s) sans code restantes", n)
	}
	if got := countHE(t, db, "copy"); got != 4 {
		t.Errorf("copy : %d lignes, attendu 4", got)
	}
	if n := scanInt(t, db, "SELECT COUNT(*) FROM highlight_events WHERE match_id = 'copy' AND event_type = 'medal'"); n != 2 {
		t.Errorf("copy : %d médailles, attendu 2 (la répétition légitime est gardée)", n)
	}
	for _, m := range []string{"clean", "solo", "pair"} {
		if got := countHE(t, db, m); got != avant[m] {
			t.Errorf("%s : %d lignes après, %d avant (match non doublé touché)", m, got, avant[m])
		}
	}
	if s := scanInt(t, db, "SELECT SUM(id) FROM highlight_events WHERE match_id = 'clean'"); s != idsClean {
		t.Errorf("clean : ids changés (%d -> %d)", idsClean, s)
	}
	if err := verifierSchemaIdentique(ctx, db, "highlight_events", ddl, index); err != nil {
		t.Errorf("schéma modifié : %v", err)
	}
	var next, max int64
	if err := db.QueryRow("INSERT INTO highlight_events (match_id, event_type) VALUES ('z', 'kill') RETURNING id").Scan(&next); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT MAX(id) FROM highlight_events WHERE match_id <> 'z'").Scan(&max); err != nil {
		t.Fatal(err)
	}
	if next <= max {
		t.Errorf("id suivant %d <= plus grand id gardé %d", next, max)
	}
}

func TestApplyHighlightEventsDedupe_Idempotente(t *testing.T) {
	db := seedHighlightEventsDoubled(t)
	if err := ApplyHighlightEventsDedupe(db); err != nil {
		t.Fatalf("premier passage: %v", err)
	}
	total := scanInt(t, db, "SELECT COUNT(*) FROM highlight_events")
	oid := tableOID(t, db, "highlight_events")
	if err := ApplyHighlightEventsDedupe(db); err != nil {
		t.Fatalf("second passage: %v", err)
	}
	if got := tableOID(t, db, "highlight_events"); got != oid {
		t.Errorf("second passage : table reconstruite (oid %d -> %d), aucun swap attendu", oid, got)
	}
	if got := scanInt(t, db, "SELECT COUNT(*) FROM highlight_events"); got != total {
		t.Errorf("second passage : %d lignes, %d avant", got, total)
	}
	rep, err := DetectHighlightEventsDuplicates(t.Context(), db)
	if err != nil || rep.Removed() != 0 {
		t.Errorf("doublons restants après dédoublonnage : %+v (err %v)", rep, err)
	}
}

func TestApplyHighlightEventsDedupe_SansTableOuSansDoublon(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ApplyHighlightEventsDedupe(db); err != nil {
		t.Fatalf("sans table: %v", err)
	}
	clean := openHighlightEventsDB(t)
	insertHE(t, clean, heRow{"m", "kill", 1000, "x1", 50, nil}, heRow{"m", "kill", 2000, "x2", 50, nil})
	oid := tableOID(t, clean, "highlight_events")
	if err := ApplyHighlightEventsDedupe(clean); err != nil {
		t.Fatalf("base saine: %v", err)
	}
	if got := tableOID(t, clean, "highlight_events"); got != oid {
		t.Errorf("base saine reconstruite (oid %d -> %d)", oid, got)
	}
}
