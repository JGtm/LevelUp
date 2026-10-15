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

// Le recalcul de tous les couples ne reecrit que les matchs dont les couples stockes different de
// l appariement courant : `m1` porte le couple de l ancien appariement glouton (le kill du bot a
// 593 093 avait pris la mort de Sprifi), `m2` est deja juste. Lecture seule d abord, puis ecriture,
// puis un second passage qui ne trouve plus rien.
func TestRebuildStalePairs(t *testing.T) {
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
		`INSERT INTO highlight_events (id,match_id,event_type,time_ms,xuid,type_hint) VALUES
			(1,'m1','kill',593093,'jgtm',50),(2,'m1','death',593095,'sprifi',20),(3,'m1','kill',593095,'jgtm',50),
			(4,'m2','kill',1000,'a',50),(5,'m2','death',1001,'b',20)`,
		`INSERT INTO killer_victim_pairs (match_id,killer_xuid,killer_gamertag,victim_xuid,victim_gamertag,time_ms)
			VALUES ('m1','jgtm','JGtm','sprifi','Sprifi',593093),('m2','a','A','b','B',1000)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	ctx := context.Background()
	rep, err := RebuildStalePairs(ctx, db, false)
	if err != nil || rep.Examined != 2 || len(rep.MatchIDs) != 1 || rep.MatchIDs[0] != "m1" || rep.Rewritten != 0 {
		t.Fatalf("lecture seule : %+v err=%v, attendu 2 examines, [m1] different, 0 reecrit", rep, err)
	}
	if rep, err = RebuildStalePairs(ctx, db, true); err != nil || rep.Rewritten != 1 {
		t.Fatalf("ecriture : %+v err=%v, attendu 1 reecrit", rep, err)
	}
	var ms int
	if err := db.QueryRow(`SELECT time_ms FROM killer_victim_pairs WHERE match_id = 'm1'`).Scan(&ms); err != nil || ms != 593095 {
		t.Errorf("couple de m1 a %d (err=%v), attendu 593 095 : la mort de Sprifi va au kill de l instant exact", ms, err)
	}
	if rep, err = RebuildStalePairs(ctx, db, false); err != nil || len(rep.MatchIDs) != 0 {
		t.Errorf("second passage : %+v err=%v, attendu aucun match different", rep, err)
	}
}
