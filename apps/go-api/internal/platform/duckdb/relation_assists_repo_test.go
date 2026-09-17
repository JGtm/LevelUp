//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
)

// seedRelationAssists : schéma minimal de Q28c et un jeu couvrant les frontières.
//
//	m1  mesuré, publiable : Me, Ally, Bis (équipe 0) contre Foe (équipe 1).
//	    Me tue Foe 5 fois : assisté par Ally à 10 / 25 / 50 / 60 %, puis sans assistant.
//	    Ally tue Foe 2 fois : assisté par Me à 80 %, puis sans assistant.
//	m2  mesuré, publiable : Me et Bis (équipe 0). Me et Bis tuent Foe une fois chacun,
//	    sans assistant → Bis « mesuré, zéro ».
//	m3  NON publiable : Me assisté par Ally → exclu, m3 non compté.
//	m4  assistance inconnue (assist_known = FALSE) → exclu, m4 non compté.
func seedRelationAssists(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER)`,
		`CREATE TABLE match_kill_events_latest (
			match_id VARCHAR, publishable BOOLEAN, assist_known BOOLEAN,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR,
			assist_xuid VARCHAR, assist_damage_pct UTINYINT, time_ms INTEGER)`,
		`INSERT INTO match_participants VALUES
			('m1','Me',0), ('m1','Ally',0), ('m1','Bis',0), ('m1','Foe',1),
			('m2','Me',0), ('m2','Bis',0), ('m2','Foe',1),
			('m3','Me',0), ('m3','Ally',0),
			('m4','Me',0), ('m4','Ally',0)`,
		`INSERT INTO match_kill_events_latest VALUES
			('m1',TRUE,TRUE,'Me','Foe','Ally',10,1),
			('m1',TRUE,TRUE,'Me','Foe','Ally',25,2),
			('m1',TRUE,TRUE,'Me','Foe','Ally',50,3),
			('m1',TRUE,TRUE,'Me','Foe','Ally',60,4),
			('m1',TRUE,TRUE,'Me','Foe',NULL,NULL,5),
			('m1',TRUE,TRUE,'Ally','Foe','Me',80,6),
			('m1',TRUE,TRUE,'Ally','Foe',NULL,NULL,7),
			('m1',TRUE,TRUE,'Foe','Me',NULL,NULL,8),
			('m2',TRUE,TRUE,'Me','Foe',NULL,NULL,1),
			('m2',TRUE,TRUE,'Bis','Foe',NULL,NULL,2),
			('m3',FALSE,TRUE,'Me','Foe','Ally',40,1),
			('m4',TRUE,FALSE,'Me','Foe',NULL,NULL,1)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seedRelationAssists: %v\nSQL: %s", err, stmt)
		}
	}
}

func openAssistsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	seedRelationAssists(t, db)
	return db
}

func TestQueryRelationAssists_AllHistory(t *testing.T) {
	db := openAssistsDB(t)
	got, err := queryRelationAssists(context.Background(), db, assistExchangeQuery{me: "Me"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	wantAlly := domain.RelationAssists{
		MatchesMeasured: 1, MyFrags: 5, PartnerFrags: 2,
		Received: domain.AssistTiers{Total: 4, Low: 1, Mid: 2, High: 1},
		Given:    domain.AssistTiers{Total: 1, High: 1},
	}
	if got["Ally"] != wantAlly {
		t.Fatalf("Ally = %+v, want %+v", got["Ally"], wantAlly)
	}
	wantBis := domain.RelationAssists{MatchesMeasured: 2, MyFrags: 6, PartnerFrags: 1}
	if got["Bis"] != wantBis {
		t.Fatalf("Bis = %+v, want %+v (mesuré, zéro assistance)", got["Bis"], wantBis)
	}
	if _, ok := got["Foe"]; ok {
		t.Fatal("un adversaire n'échange pas d'assistances : Foe doit être absent")
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (%+v)", len(got), got)
	}
}

func TestQueryRelationAssists_ScopeAndMatchPartners(t *testing.T) {
	db := openAssistsDB(t)
	ctx := context.Background()

	scoped, err := queryRelationAssists(ctx, db, assistExchangeQuery{me: "Me", scope: []string{"m2"}})
	if err != nil {
		t.Fatalf("scoped: %v", err)
	}
	if len(scoped) != 1 || scoped["Bis"].MatchesMeasured != 1 || scoped["Bis"].MyFrags != 1 {
		t.Fatalf("scope m2 = %+v", scoped)
	}

	empty, err := queryRelationAssists(ctx, db, assistExchangeQuery{me: "Me", scope: []string{}})
	if err != nil || len(empty) != 0 {
		t.Fatalf("scope vide = %+v, err %v", empty, err)
	}

	ofMatch, err := queryRelationAssists(ctx, db, assistExchangeQuery{me: "Me", partnersOfMatch: "m2"})
	if err != nil {
		t.Fatalf("partners of m2: %v", err)
	}
	// Bis est dans m2 : tout son historique (m1 + m2). Ally n'y est pas.
	if len(ofMatch) != 1 || ofMatch["Bis"].MatchesMeasured != 2 {
		t.Fatalf("partners of m2 = %+v", ofMatch)
	}
}

func TestCareerRepo_GetRelationAssists(t *testing.T) {
	db := openMemDB(t)
	seedRelationAssists(t, db.SQLDb())
	repo := NewCareerRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})
	got, err := repo.GetRelationAssists(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetRelationAssists: %v", err)
	}
	if got["Ally"].Received.Total != 4 {
		t.Fatalf("Ally = %+v", got["Ally"])
	}
}
