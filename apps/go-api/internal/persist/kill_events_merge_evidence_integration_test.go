//go:build integration

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// LoadMergeEvidence : kills distincts par instant (les groupes en double exact comptent une
// fois) et morts officielles par joueur.
func TestLoadMergeEvidence(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, s := range []string{
		`CREATE TABLE highlight_events (match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, deaths INTEGER)`,
		`INSERT INTO highlight_events VALUES ('m','kill',100,'a'),('m','kill',100,'a'),('m','kill',100,'b'),
			('m','death',100,'c'),('m','kill',200,'a'),('autre','kill',100,'z')`,
		`INSERT INTO match_participants VALUES ('m','c',7),('m','d',NULL),('autre','c',99)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	ev, err := LoadMergeEvidence(context.Background(), db, "m")
	if err != nil {
		t.Fatal(err)
	}
	if !ev.Known || ev.APIKills[100] != 2 || ev.APIKills[200] != 1 || len(ev.APIKills) != 2 {
		t.Errorf("kills = %v, attendu {100:2, 200:1}", ev.APIKills)
	}
	if len(ev.OfficialDeaths) != 1 || ev.OfficialDeaths["c"] != 7 {
		t.Errorf("morts officielles = %v, attendu {c:7}", ev.OfficialDeaths)
	}
}
