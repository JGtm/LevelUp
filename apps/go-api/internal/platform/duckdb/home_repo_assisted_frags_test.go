//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// seedHomeAssistedFrags : schéma minimal de Q26l et un jeu couvrant les frontières.
//
//	m1  film porteur : Me tue 9 fois — sans assistant, assisté à 10 / 30 / 80 / 25 / 50 %
//	    (25 et 50 sont des parts MOYENNES : bornes incluses), assisté SANS part mesurée,
//	    assisté par un BOT (gamertag seul, 40 %), et une victime bot dont l'assistance n'est
//	    pas lue (frag de la base, hors des assistés). Une ligne où Me est la VICTIME est
//	    ignorée.
//	m2  film porteur : Me tue 2 fois sans assistant → « zéro assisté ».
//	m3  le film ne porte aucune assistance (assist_known = FALSE partout) → absent.
//	m4  seul un AUTRE joueur y tue → absent (aucun frag lu pour Me).
//	m5  film porteur par la ligne d'un AUTRE tueur ; Me n'y tue qu'un bot, assistance non
//	    lue → présent, 1 frag lu, 0 assisté.
//	m6  passe NON publiable (Me tue, assisté) → absent.
func seedHomeAssistedFrags(t *testing.T, db *sql.DB) {
	t.Helper()
	// Le journal et ses résumés (ADR 0039) : schéma réel, une passe par match.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatalf("EnsureMatchKillEvents: %v", err)
	}
	// ligne : (match, publiable, assistance lue, tueur, victime, assistant xuid, assistant gamertag, part, instant).
	ligne := func(match string, pub, known bool, killer, victim, assistX, assistGT string, p *uint8, ms int) ligneJournal {
		return ligneJournal{match: match, publiable: pub, assistKnown: known, tueur: killer, victime: victim,
			assistant: assistX, assistantGT: assistGT, pctAssist: p, timeMS: ms}
	}
	poserJournal(t, db,
		ligne("m1", true, true, "Me", "Foe", "", "", nil, 1),
		ligne("m1", true, true, "Me", "Foe", "Ally", "AllyGT", pct(10), 2),
		ligne("m1", true, true, "Me", "Foe", "Ally", "AllyGT", pct(30), 3),
		ligne("m1", true, true, "Me", "Foe", "Ally", "AllyGT", pct(25), 8),
		ligne("m1", true, true, "Me", "Foe", "Bis", "BisGT", pct(50), 9),
		ligne("m1", true, true, "Me", "Foe", "Bis", "BisGT", pct(80), 4),
		ligne("m1", true, true, "Me", "Foe", "Ally", "AllyGT", nil, 5),
		ligne("m1", true, true, "Me", "Foe", "", "343 Oscar [bot]", pct(40), 10),
		ligne("m1", true, false, "Me", "", "", "", nil, 11),
		ligne("m1", true, true, "Foe", "Me", "Zed", "ZedGT", pct(60), 7),
		ligne("m2", true, true, "Me", "Foe", "", "", nil, 1),
		ligne("m2", true, true, "Me", "Foe", "", "", nil, 2),
		ligne("m3", true, false, "Me", "Foe", "", "", nil, 1),
		ligne("m4", true, true, "Ally", "Foe", "Me", "MeGT", pct(50), 1),
		ligne("m5", true, true, "Ally", "Foe", "", "", nil, 1),
		ligne("m5", true, false, "Me", "", "", "", nil, 2),
		// m6 : passe NON publiable (la publiabilité est celle de la passe) → absent.
		ligne("m6", false, true, "Me", "Foe", "Ally", "AllyGT", pct(40), 1),
	)
}

func TestHomeRepo_LoadMatchAssistedFrags(t *testing.T) {
	db := openMemDB(t)
	seedHomeAssistedFrags(t, db.SQLDb())
	repo := NewHomeRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})

	got, err := repo.LoadMatchAssistedFrags(context.Background(), []string{"m1", "m2", "m3", "m4", "m5", "m6", "absent"})
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
	for _, id := range []string{"m3", "m4", "m6", "absent"} {
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
