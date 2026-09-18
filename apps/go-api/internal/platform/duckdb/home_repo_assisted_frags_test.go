//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
)

// seedHomeAssistedFrags : schéma minimal de Q26l et un jeu couvrant les frontières.
//
//	m1  mesuré : Me tue 7 fois — sans assistant, puis assisté à 10 / 30 / 80 / 25 / 50 %
//	    (25 et 50 sont des parts MOYENNES : bornes incluses), puis
//	    assisté SANS part mesurée (assist_xuid renseigné, pct NULL). Une ligne NON
//	    publiable (assist à 40 %) et une ligne où Me est la VICTIME sont ignorées.
//	m2  mesuré : Me tue 2 fois sans assistant → « mesuré, zéro ».
//	m3  aucune ligne mesurée pour Me (assist_known = FALSE) → absent de la map.
//	m4  seul un AUTRE joueur y est mesuré → absent de la map.
func seedHomeAssistedFrags(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_kill_events_latest (
			match_id VARCHAR, publishable BOOLEAN, assist_known BOOLEAN,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR,
			assist_xuid VARCHAR, assist_damage_pct UTINYINT, time_ms INTEGER)`,
		`INSERT INTO match_kill_events_latest VALUES
			('m1',TRUE,TRUE,'Me','Foe',NULL,NULL,1),
			('m1',TRUE,TRUE,'Me','Foe','Ally',10,2),
			('m1',TRUE,TRUE,'Me','Foe','Ally',30,3),
			('m1',TRUE,TRUE,'Me','Foe','Ally',25,8),
			('m1',TRUE,TRUE,'Me','Foe','Bis',50,9),
			('m1',TRUE,TRUE,'Me','Foe','Bis',80,4),
			('m1',TRUE,TRUE,'Me','Foe','Ally',NULL,5),
			('m1',FALSE,TRUE,'Me','Foe','Ally',40,6),
			('m1',TRUE,TRUE,'Foe','Me','Zed',60,7),
			('m2',TRUE,TRUE,'Me','Foe',NULL,NULL,1),
			('m2',TRUE,TRUE,'Me','Foe',NULL,NULL,2),
			('m3',TRUE,FALSE,'Me','Foe',NULL,NULL,1),
			('m4',TRUE,TRUE,'Ally','Foe','Me',50,1)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seedHomeAssistedFrags: %v\nSQL: %s", err, stmt)
		}
	}
}

func TestHomeRepo_LoadMatchAssistedFrags(t *testing.T) {
	db := openMemDB(t)
	seedHomeAssistedFrags(t, db.SQLDb())
	repo := NewHomeRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})

	got, err := repo.LoadMatchAssistedFrags(context.Background(), []string{"m1", "m2", "m3", "m4", "absent"})
	if err != nil {
		t.Fatalf("LoadMatchAssistedFrags: %v", err)
	}
	want1 := domain.MatchAssistedFrags{
		FragsMeasured: 7,
		Received:      domain.AssistTiers{Total: 6, Low: 1, Mid: 3, High: 1},
	}
	if got["m1"] != want1 {
		t.Fatalf("m1 = %+v, want %+v", got["m1"], want1)
	}
	want2 := domain.MatchAssistedFrags{FragsMeasured: 2}
	if got["m2"] != want2 {
		t.Fatalf("m2 = %+v, want %+v (mesuré, zéro assistance)", got["m2"], want2)
	}
	for _, id := range []string{"m3", "m4", "absent"} {
		if _, ok := got[id]; ok {
			t.Fatalf("%s doit être ABSENT (aucune ligne mesurée pour Me) : %+v", id, got[id])
		}
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (%+v)", len(got), got)
	}
}

func TestHomeRepo_LoadMatchAssistedFrags_EmptyBatch(t *testing.T) {
	db := openMemDB(t)
	seedHomeAssistedFrags(t, db.SQLDb())
	repo := NewHomeRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})
	got, err := repo.LoadMatchAssistedFrags(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("lot vide = %+v, err %v", got, err)
	}
}
