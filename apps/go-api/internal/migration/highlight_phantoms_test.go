package migration

// highlight_phantoms_test.go — PurgeHighlightPhantoms retire les seules lignes fantomes, garde le
// schema, la sequence et les index, et est idempotente.

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedHighlightPhantoms(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range []string{
		`CREATE SEQUENCE highlight_events_id_seq START 1`,
		`CREATE TABLE highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR,
			type_hint INTEGER, raw_json VARCHAR)`,
		`CREATE INDEX idx_highlight_match ON highlight_events(match_id)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`,
		`INSERT INTO match_participants VALUES ('m1','real'),('m1','other'),('m2','real2')`,
		// m1 : kill@40500 emis sous le vrai ET sous un faux xuid ; kill@900 par un autre participant
		// (deux tueurs a la meme milliseconde : AUCUN fantome) ; medaille jumelle.
		`INSERT INTO highlight_events (match_id,event_type,time_ms,xuid,type_hint,raw_json) VALUES
			('m1','kill',40500,'real',50,NULL),('m1','kill',40500,'fake',50,NULL),
			('m1','kill',900,'real',50,NULL),('m1','kill',900,'other',50,NULL),
			('m1','medal',41000,'real',100,'{"m":1}'),('m1','medal',41000,'fake',100,'{"m":1}'),
			('m1','kill',70000,'fake',50,NULL)`,
		// m2 : le faux n a pas de jumeau participant -> conserve (rien n est retire dans le doute) ;
		// m3 : aucun participant charge -> rien n est retire.
		`INSERT INTO highlight_events (match_id,event_type,time_ms,xuid,type_hint,raw_json) VALUES
			('m2','kill',100,'ghost',50,NULL),('m3','kill',5,'a',50,NULL),('m3','kill',5,'b',50,NULL)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
	return db
}

func TestPurgeHighlightPhantoms(t *testing.T) {
	db := seedHighlightPhantoms(t)
	ctx := t.Context()
	ddl, _ := ddlDeTable(ctx, db, "highlight_events")
	index, _ := ddlDesIndex(ctx, db, "highlight_events")

	dry, err := PurgeHighlightPhantoms(ctx, db, true)
	if err != nil || dry.Rows != 2 || len(dry.MatchIDs) != 1 || dry.Purged {
		t.Fatalf("dry-run : %+v err=%v, attendu 2 lignes sur m1, rien retire", dry, err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM highlight_events`).Scan(&n)
	if n != 10 {
		t.Fatalf("le dry-run a ecrit : %d lignes", n)
	}

	rep, err := PurgeHighlightPhantoms(ctx, db, false)
	if err != nil || rep.Rows != 2 || !rep.Purged || rep.MatchIDs[0] != "m1" {
		t.Fatalf("purge : %+v err=%v", rep, err)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM highlight_events`).Scan(&n)
	if n != 8 {
		t.Errorf("%d lignes apres purge, attendu 8", n)
	}
	// Restent : le faux isole de m1@70000 (pas de jumeau), celui de m2, et les deux tueurs de m1@900.
	var fakes int
	_ = db.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE xuid = 'fake'`).Scan(&fakes)
	if fakes != 1 {
		t.Errorf("%d lignes 'fake' restantes, attendu 1 (celle sans jumeau)", fakes)
	}
	ddl2, _ := ddlDeTable(ctx, db, "highlight_events")
	index2, _ := ddlDesIndex(ctx, db, "highlight_events")
	if ddl != ddl2 || len(index) != len(index2) || len(index2) == 0 {
		t.Errorf("schema ou index modifies\navant %s %v\napres %s %v", ddl, index, ddl2, index2)
	}
	if _, err := db.Exec(`INSERT INTO highlight_events (match_id,event_type,time_ms,xuid,type_hint) VALUES ('m9','kill',1,'x',50)`); err != nil {
		t.Errorf("la sequence ne fournit plus d id : %v", err)
	}

	again, err := PurgeHighlightPhantoms(ctx, db, false)
	if err != nil || again.Rows != 0 || again.Purged {
		t.Errorf("seconde passe : %+v err=%v, attendu idempotente (0 ligne)", again, err)
	}
}
