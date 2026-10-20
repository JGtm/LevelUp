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
//	m1  film porteur : Me tue 9 fois — sans assistant, assisté à 10 / 30 / 80 / 25 / 50 %
//	    (25 et 50 sont des parts MOYENNES : bornes incluses), assisté SANS part mesurée,
//	    assisté par un BOT (gamertag seul, 40 %), et une victime bot dont l'assistance n'est
//	    pas lue (frag de la base, hors des assistés). Une ligne NON publiable et une ligne
//	    où Me est la VICTIME sont ignorées.
//	m2  film porteur : Me tue 2 fois sans assistant → « zéro assisté ».
//	m3  le film ne porte aucune assistance (assist_known = FALSE partout) → absent.
//	m4  seul un AUTRE joueur y tue → absent (aucun frag lu pour Me).
//	m5  film porteur par la ligne d'un AUTRE tueur ; Me n'y tue qu'un bot, assistance non
//	    lue → présent, 1 frag lu, 0 assisté.
func seedHomeAssistedFrags(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_kill_events_latest (
			match_id VARCHAR, publishable BOOLEAN, assist_known BOOLEAN,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR,
			assist_xuid VARCHAR, assist_gamertag VARCHAR, assist_damage_pct UTINYINT, time_ms INTEGER)`,
		`INSERT INTO match_kill_events_latest VALUES
			('m1',TRUE,TRUE,'Me','Foe',NULL,NULL,NULL,1),
			('m1',TRUE,TRUE,'Me','Foe','Ally','AllyGT',10,2),
			('m1',TRUE,TRUE,'Me','Foe','Ally','AllyGT',30,3),
			('m1',TRUE,TRUE,'Me','Foe','Ally','AllyGT',25,8),
			('m1',TRUE,TRUE,'Me','Foe','Bis','BisGT',50,9),
			('m1',TRUE,TRUE,'Me','Foe','Bis','BisGT',80,4),
			('m1',TRUE,TRUE,'Me','Foe','Ally','AllyGT',NULL,5),
			('m1',TRUE,TRUE,'Me','Foe',NULL,'343 Oscar [bot]',40,10),
			('m1',TRUE,FALSE,'Me',NULL,NULL,NULL,NULL,11),
			('m1',FALSE,TRUE,'Me','Foe','Ally','AllyGT',40,6),
			('m1',TRUE,TRUE,'Foe','Me','Zed','ZedGT',60,7),
			('m2',TRUE,TRUE,'Me','Foe',NULL,NULL,NULL,1),
			('m2',TRUE,TRUE,'Me','Foe',NULL,NULL,NULL,2),
			('m3',TRUE,FALSE,'Me','Foe',NULL,NULL,NULL,1),
			('m4',TRUE,TRUE,'Ally','Foe','Me','MeGT',50,1),
			('m5',TRUE,TRUE,'Ally','Foe',NULL,NULL,NULL,1),
			('m5',TRUE,FALSE,'Me',NULL,NULL,NULL,NULL,2)`,
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

	got, err := repo.LoadMatchAssistedFrags(context.Background(), []string{"m1", "m2", "m3", "m4", "m5", "absent"})
	if err != nil {
		t.Fatalf("LoadMatchAssistedFrags: %v", err)
	}
	want := map[string]domain.MatchAssistedFrags{
		"m1": {FragsFilm: 9, Received: domain.AssistTiers{Total: 7, Low: 1, Mid: 4, High: 1}},
		"m2": {FragsFilm: 2},
		"m5": {FragsFilm: 1},
	}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("%s = %+v, want %+v", id, got[id], w)
		}
	}
	for _, id := range []string{"m3", "m4", "absent"} {
		if _, ok := got[id]; ok {
			t.Fatalf("%s doit être ABSENT : %+v", id, got[id])
		}
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
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
