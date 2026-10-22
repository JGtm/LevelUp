//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// seedRelationAssists : schéma minimal de Q28c et un jeu couvrant les frontières.
//
//	m1  film porteur : Me, Ally, Bis et un BOT (équipe 0) contre Foe (équipe 1).
//	    Me tue Foe 5 fois : assisté par Ally à 10 / 25 / 50 / 60 %, puis sans assistant ;
//	    Me tue aussi un bot dont l'assistance n'est pas lue, et la feuille lui compte un
//	    frag que le film ne lit pas : 7 frags officiels.
//	    Ally tue Foe 2 fois : assisté par Me à 80 %, puis sans assistant.
//	m2  film porteur : Me et Bis (équipe 0). Me et Bis tuent Foe une fois chacun,
//	    sans assistant → Bis « zéro assistance ».
//	m3  NON publiable : Me assisté par Ally → m3 hors de la mesure.
//	m4  le film ne porte aucune assistance (assist_known = FALSE) → m4 hors de la mesure.
func seedRelationAssists(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER, kills INTEGER)`,
		`INSERT INTO match_participants VALUES
			('m1','Me',0,7), ('m1','Ally',0,2), ('m1','Bis',0,0), ('m1','bid(1.0)',0,1), ('m1','Foe',1,1),
			('m2','Me',0,1), ('m2','Bis',0,1), ('m2','Foe',1,0),
			('m3','Me',0,3), ('m3','Ally',0,3),
			('m4','Me',0,4), ('m4','Ally',0,4)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seedRelationAssists: %v\nSQL: %s", err, stmt)
		}
	}
	// Le journal et ses résumés (ADR 0039) : schéma réel, une passe par match.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatalf("EnsureMatchKillEvents: %v", err)
	}
	ligne := func(match string, pub, known bool, killer, victim, assist string, pctAssist *uint8, ms int) ligneJournal {
		return ligneJournal{match: match, publiable: pub, assistKnown: known, tueur: killer, victime: victim,
			assistant: assist, assistantGT: assist, pctAssist: pctAssist, timeMS: ms}
	}
	poserJournal(t, db,
		ligne("m1", true, true, "Me", "Foe", "Ally", pct(10), 1),
		ligne("m1", true, true, "Me", "Foe", "Ally", pct(25), 2),
		ligne("m1", true, true, "Me", "Foe", "Ally", pct(50), 3),
		ligne("m1", true, true, "Me", "Foe", "Ally", pct(60), 4),
		ligne("m1", true, true, "Me", "Foe", "", nil, 5),
		ligne("m1", true, false, "Me", "", "", nil, 9),
		ligne("m1", true, true, "Ally", "Foe", "Me", pct(80), 6),
		ligne("m1", true, true, "Ally", "Foe", "", nil, 7),
		ligne("m1", true, true, "Foe", "Me", "", nil, 8),
		ligne("m2", true, true, "Me", "Foe", "", nil, 1),
		ligne("m2", true, true, "Bis", "Foe", "", nil, 2),
		ligne("m3", false, true, "Me", "Foe", "Ally", pct(40), 1),
		ligne("m4", true, false, "Me", "Foe", "", nil, 1),
	)
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
	// Bases = frags OFFICIELS des matchs porteurs : 7 pour Me sur m1 (frag sur un bot et
	// frag absent du film compris), jamais les 5 lignes dont l'assistance est lue.
	wantAlly := domain.RelationAssists{
		MyFrags: 7, PartnerFrags: 2,
		Received: domain.AssistTiers{Total: 4, Low: 1, Mid: 2, High: 1},
		Given:    domain.AssistTiers{Total: 1, High: 1},
	}
	if got["Ally"] != wantAlly {
		t.Fatalf("Ally = %+v, want %+v", got["Ally"], wantAlly)
	}
	wantBis := domain.RelationAssists{MyFrags: 8, PartnerFrags: 1}
	if got["Bis"] != wantBis {
		t.Fatalf("Bis = %+v, want %+v (zéro assistance)", got["Bis"], wantBis)
	}
	if _, ok := got["Foe"]; ok {
		t.Fatal("un adversaire n'échange pas d'assistances : Foe doit être absent")
	}
	if _, ok := got["bid(1.0)"]; ok {
		t.Fatal("un bot n'est pas une ligne de relation : bid(1.0) doit être absent")
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
	if len(scoped) != 1 || scoped["Bis"].MyFrags != 1 || scoped["Bis"].PartnerFrags != 1 {
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
	if len(ofMatch) != 1 || ofMatch["Bis"].MyFrags != 8 {
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
