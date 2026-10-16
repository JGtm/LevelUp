package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestLoadFilmStats(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if got := loadFilmStats(ctx, db, "m1", "x"); got != nil {
		t.Fatalf("vue absente : nil attendu, obtenu %v", got)
	}
	if _, err := db.Exec(`
		CREATE TABLE t (match_id VARCHAR, xuid VARCHAR, pad_pickups INTEGER, shieldless_kills INTEGER);
		CREATE VIEW match_usage_players_latest AS SELECT * FROM t;
		INSERT INTO t VALUES ('m1', 'mesure', 4, 2), ('m1', 'non-mesure', 3, NULL);`); err != nil {
		t.Fatal(err)
	}
	got := loadFilmStats(ctx, db, "m1", "mesure")
	if got[filmStatPadPickups] != 4 || got[filmStatShieldlessKills] != 2 {
		t.Fatalf("ligne mesuree : %v", got)
	}
	got = loadFilmStats(ctx, db, "m1", "non-mesure")
	if _, ok := got[filmStatShieldlessKills]; ok || got[filmStatPadPickups] != 3 {
		t.Fatalf("frags non mesures : la cle doit etre absente, obtenu %v", got)
	}
	if got := loadFilmStats(ctx, db, "m2", "mesure"); got != nil {
		t.Fatalf("match sans ligne : nil attendu, obtenu %v", got)
	}
}
