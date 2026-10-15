//go:build integration

package killcollector

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Apres la purge, les couples du match se recalculent depuis highlight_events assaini : un seul
// couple (Partycz -> Kin3tico), idempotent a la seconde passe.
func TestRebuildPairsAfterPhantomPurge(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range []string{
		`CREATE TABLE highlight_events (id INTEGER, match_id VARCHAR, event_type VARCHAR, time_ms INTEGER,
			xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR)`,
		`CREATE TABLE killer_victim_pairs (match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
			kill_count INTEGER DEFAULT 1, time_ms INTEGER, is_validated BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		// Couples d avant la reparation : le fantome avait pris la mort.
		`INSERT INTO killer_victim_pairs (match_id,killer_xuid,killer_gamertag,victim_xuid,victim_gamertag,time_ms)
			VALUES ('m','fake','Partycz','kin','Kin3tico',40500),('m','real','Partycz','real','Partycz',80000)`,
		// highlight_events assaini : un kill reel, la mort de Kin3tico (les doublons exacts comptent une fois).
		`INSERT INTO highlight_events (id,match_id,event_type,time_ms,xuid,type_hint) VALUES
			(1,'m','kill',40500,'real',50),(2,'m','kill',40500,'real',50),(3,'m','death',40501,'kin',20)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	for pass := 1; pass <= 2; pass++ {
		n, err := RebuildPairsAfterPhantomPurge(context.Background(), db, []string{"m"})
		if err != nil || n != 1 {
			t.Fatalf("passe %d : n=%d err=%v", pass, n, err)
		}
		var k, v, gk string
		var c int
		if err := db.QueryRow(`SELECT COUNT(*), MAX(killer_xuid), MAX(victim_xuid), MAX(killer_gamertag)
			FROM killer_victim_pairs WHERE match_id = 'm'`).Scan(&c, &k, &v, &gk); err != nil {
			t.Fatal(err)
		}
		if c != 1 || k != "real" || v != "kin" || gk != "Partycz" {
			t.Errorf("passe %d : %d couples %s->%s (%s), attendu 1 couple real->kin nomme Partycz", pass, c, k, v, gk)
		}
	}
}
