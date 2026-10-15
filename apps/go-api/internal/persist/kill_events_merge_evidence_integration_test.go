//go:build integration

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// LoadMergeEvidence : instants distincts des morts de l API par joueur (les groupes en double exact
// comptent une fois) et morts officielles ; preuves inconnues pour un match sans evenement.
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
			('m','death',100,'c'),('m','death',100,'c'),('m','death',104,'c'),('m','kill',200,'a'),('autre','kill',100,'z')`,
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
	if !ev.Known {
		t.Error("preuves inconnues pour un match qui porte des evenements")
	}
	if len(ev.OfficialDeaths) != 1 || ev.OfficialDeaths["c"] != 7 {
		t.Errorf("morts officielles = %v, attendu {c:7}", ev.OfficialDeaths)
	}
	if d := ev.APIDeaths["c"]; len(ev.APIDeaths) != 1 || len(d) != 2 || d[0] != 100 || d[1] != 104 {
		t.Errorf("morts de l API = %v, attendu {c:[100 104]} (doublon exact compte une fois)", ev.APIDeaths)
	}
	// Un match sans aucun evenement kill ni death : les preuves restent inconnues (la fusion ne
	// refuse alors aucune mort du film). Mutation qui doit rougir : `Known` pose sans condition.
	vide, err := LoadMergeEvidence(context.Background(), db, "sans-evenements")
	if err != nil || vide.Known {
		t.Errorf("match sans evenement : Known=%v err=%v, attendu inconnu", vide.Known, err)
	}
}
