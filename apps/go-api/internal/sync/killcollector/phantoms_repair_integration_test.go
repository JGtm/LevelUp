//go:build integration

package killcollector

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// baseDeReparation : `highlight_events`, `killer_victim_pairs` AVEC ses trois index ART herites (la
// forme des bases anciennes) et l annuaire de lecture.
func baseDeReparation(t *testing.T, seed ...string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range append([]string{
		`CREATE TABLE highlight_events (id INTEGER, match_id VARCHAR, event_type VARCHAR, time_ms INTEGER,
			xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR)`,
		`CREATE TABLE killer_victim_pairs (match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL, victim_gamertag VARCHAR,
			kill_count INTEGER DEFAULT 1, time_ms INTEGER, is_validated BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
		`CREATE INDEX idx_kv_match ON killer_victim_pairs(match_id)`,
		`CREATE INDEX idx_kv_killer ON killer_victim_pairs(killer_xuid)`,
		`CREATE INDEX idx_kv_victim ON killer_victim_pairs(victim_xuid)`,
		`CREATE TABLE v_gamertag_lookup (xuid VARCHAR, gamertag VARCHAR)`,
	}, seed...) {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

// reparerLesCouples : le chemin de la commande — calcul en lecture, ecriture par echange.
func reparerLesCouples(t *testing.T, db *sql.DB) StalePairsReport {
	t.Helper()
	ctx := context.Background()
	rep, err := StalePairs(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	parMatch := map[string][]migration.KillerVictimPair{}
	for id, ps := range rep.Pairs {
		out := []migration.KillerVictimPair{}
		for _, p := range ps {
			out = append(out, migration.KillerVictimPair{KillerXUID: p.KillerXUID, KillerGamertag: p.KillerGamertag,
				VictimXUID: p.VictimXUID, VictimGamertag: p.VictimGamertag, TimeMS: p.TimeMS})
		}
		parMatch[id] = out
	}
	if _, err := migration.ReplaceKillerVictimPairs(ctx, db, parMatch); err != nil {
		t.Fatal(err)
	}
	return rep
}

// Apres la purge des fantomes, les couples du match se recalculent depuis highlight_events assaini :
// un seul couple (Partycz -> Kin3tico), noms de l annuaire, les trois index gardes ; un second
// passage ne trouve plus rien (et c est ce qui rend une reparation interrompue reprenable).
func TestReparationDesCouplesApresPurge(t *testing.T) {
	db := baseDeReparation(t,
		`INSERT INTO killer_victim_pairs (match_id,killer_xuid,killer_gamertag,victim_xuid,victim_gamertag,time_ms)
			VALUES ('m','fake','Partycz','kin','Kin3tico',40500),('m','real','Partycz','real','Partycz',80000)`,
		`INSERT INTO highlight_events (id,match_id,event_type,time_ms,xuid,type_hint) VALUES
			(1,'m','kill',40500,'real',50),(2,'m','kill',40500,'real',50),(3,'m','death',40501,'kin',20)`,
		`INSERT INTO v_gamertag_lookup VALUES ('real','Partycz'),('kin','Kin3tico')`)
	if rep := reparerLesCouples(t, db); len(rep.MatchIDs) != 1 {
		t.Fatalf("matchs recalcules = %v, attendu [m]", rep.MatchIDs)
	}
	var c int
	var k, v, gk string
	if err := db.QueryRow(`SELECT COUNT(*), MAX(killer_xuid), MAX(victim_xuid), MAX(killer_gamertag)
		FROM killer_victim_pairs WHERE match_id = 'm'`).Scan(&c, &k, &v, &gk); err != nil {
		t.Fatal(err)
	}
	if c != 1 || k != "real" || v != "kin" || gk != "Partycz" {
		t.Errorf("%d couples %s->%s (%s), attendu 1 couple real->kin nomme Partycz", c, k, v, gk)
	}
	var idx int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'killer_victim_pairs'`).
		Scan(&idx); err != nil || idx != 3 {
		t.Errorf("index apres echange = %d (err=%v), attendu les 3 d avant", idx, err)
	}
	if rep, err := StalePairs(context.Background(), db); err != nil || len(rep.MatchIDs) != 0 {
		t.Errorf("second passage : %v err=%v, attendu aucun match", rep.MatchIDs, err)
	}
}

// Seuls les matchs dont les couples stockes different de l appariement courant sont recalcules :
// `m1` porte le couple de l ancien appariement glouton (le kill du bot a 593 093 avait pris la mort
// de Sprifi), `m2` est deja juste. Un joueur que ni l annuaire ni les couples ne nomment reste sans
// nom : jamais son xuid a la place.
func TestStalePairsNeRecalculeQueLesCouplesPerimes(t *testing.T) {
	db := baseDeReparation(t,
		`INSERT INTO highlight_events (id,match_id,event_type,time_ms,xuid,type_hint) VALUES
			(1,'m1','kill',593093,'jgtm',50),(2,'m1','death',593095,'sprifi',20),(3,'m1','kill',593095,'jgtm',50),
			(4,'m2','kill',1000,'a',50),(5,'m2','death',1001,'b',20)`,
		`INSERT INTO killer_victim_pairs (match_id,killer_xuid,killer_gamertag,victim_xuid,victim_gamertag,time_ms)
			VALUES ('m1','jgtm','JGtm','sprifi','sprifi',593093),('m2','a','A','b','B',1000)`)
	rep := reparerLesCouples(t, db)
	if rep.Examined != 2 || len(rep.MatchIDs) != 1 || rep.MatchIDs[0] != "m1" {
		t.Fatalf("rapport %+v, attendu 2 examines, [m1] recalcule", rep)
	}
	var ms int
	var gk, gv sql.NullString
	if err := db.QueryRow(`SELECT time_ms, killer_gamertag, victim_gamertag FROM killer_victim_pairs
		WHERE match_id = 'm1'`).Scan(&ms, &gk, &gv); err != nil {
		t.Fatal(err)
	}
	if ms != 593095 || gk.String != "JGtm" || gv.Valid {
		t.Errorf("couple de m1 : %d, %v, %v ; attendu 593 095, JGtm, victime sans nom (NULL)", ms, gk, gv)
	}
}
