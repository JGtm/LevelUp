//go:build integration

// Package duckdb — coordination_repo_test.go : QCoordinationAppuis sur DuckDB `:memory:`.
//
// Les frontières de la requête, et pourquoi chacune compte :
//   - `publishable` et `assist_known` écartent les lignes dont l'assistance n'est pas
//     mesurée ligne à ligne (elles restent dans la base officielle, QCoordinationFragsOfficiels) ;
//   - une mort sans assistant sort sans xuid ni gamertag d'assistant : l'état MESURÉ
//     « personne n'a assisté » ;
//   - les BOTS sortent : assistant nommé par son seul gamertag, tueur sans xuid, et la
//     victime renseignée quand assistant et tueur sont tous deux sans xuid.
package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

// seedCoordinationAppuis : le schéma minimal de la requête et un jeu couvrant les bornes.
//
//	m1  Me tue 2 fois sans assistant, 2 fois assisté par Ally, 1 fois par Bis, 1 fois par un
//	    BOT ; une ligne NON publiable et une ligne assist_known = FALSE sont ignorées ;
//	    un tueur bot assisté par Ally ; un tueur bot assisté par un bot, sur Foe.
//	m2  Ally tue 1 fois, assisté par Me — l'appui que JE distribue.
//	m3  hors périmètre demandé.
func seedCoordinationAppuis(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_kill_events_latest (
			match_id VARCHAR, publishable BOOLEAN, assist_known BOOLEAN,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR,
			assist_xuid VARCHAR, assist_gamertag VARCHAR, time_ms INTEGER)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`INSERT INTO match_kill_events_latest VALUES
			('m1',TRUE ,TRUE ,'Me' ,'Foe',NULL  ,NULL      ,1),
			('m1',TRUE ,TRUE ,'Me' ,'Foe',NULL  ,NULL      ,2),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Ally','AllyGT'  ,3),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Ally','AllyGT'  ,4),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Bis' ,'BisGT'   ,5),
			('m1',TRUE ,TRUE ,'Me' ,'Foe',NULL  ,'Bot Un'  ,10),
			('m1',FALSE,TRUE ,'Me' ,'Foe','Ally','AllyGT'  ,6),
			('m1',TRUE ,FALSE,'Me' ,'Foe',NULL  ,NULL      ,7),
			('m1',TRUE ,TRUE ,NULL ,'Foe','Ally','AllyGT'  ,8),
			('m1',TRUE ,TRUE ,NULL ,'Foe',NULL  ,'Bot Deux',9),
			('m2',TRUE ,TRUE ,'Ally','Foe','Me' ,'MeGT'    ,1),
			('m3',TRUE ,TRUE ,'Me' ,'Foe','Ally','AllyGT'  ,1)`,
		`INSERT INTO match_participants VALUES
			('m1','Me',9), ('m1','Ally',3), ('m2','Me',0), ('m2','Me',0), ('m3','Me',4),
			('m2','Ally',NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seedCoordinationAppuis: %v\nSQL: %s", err, stmt)
		}
	}
}

func TestCoordinationRepo_LoadAppuis(t *testing.T) {
	db := openMemDB(t)
	seedCoordinationAppuis(t, db.SQLDb())
	repo := NewCoordinationRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})

	got, err := repo.LoadAppuis(context.Background(), []string{"m1", "m2", "absent"})
	if err != nil {
		t.Fatalf("LoadAppuis: %v", err)
	}

	type cle struct{ match, assist, assistGT, killer, victim string }
	vu := map[cle]int{}
	for _, r := range got {
		vu[cle{r.MatchID, r.AssistXUID, r.AssistGamertag, r.KillerXUID, r.VictimXUID}] = r.Nombre
	}
	attendus := map[cle]int{
		// Deux frags MESURÉS sans assistant : ni xuid ni gamertag d'assistant.
		{"m1", "", "", "Me", ""}:          2,
		{"m1", "Ally", "", "Me", ""}:      2,
		{"m1", "Bis", "", "Me", ""}:       1,
		{"m1", "", "Bot Un", "Me", ""}:    1,
		{"m1", "Ally", "", "", ""}:        1,
		{"m1", "", "Bot Deux", "", "Foe"}: 1,
		{"m2", "Me", "", "Ally", ""}:      1,
	}
	if len(vu) != len(attendus) {
		t.Fatalf("%d lignes, attendu %d : %+v", len(vu), len(attendus), got)
	}
	for k, want := range attendus {
		if vu[k] != want {
			t.Errorf("%+v = %d, attendu %d — la ligne non publiable, celle dont l'assistance "+
				"n'est pas connue et le match hors périmètre ne doivent pas y entrer", k, vu[k], want)
		}
	}
}

// TestCoordinationRepo_LoadFragsOfficiels — la feuille de match du joueur, par match : un
// doublon de participant ne double rien, un compte NULL est absent, un autre joueur et un
// match hors périmètre n'entrent pas.
func TestCoordinationRepo_LoadFragsOfficiels(t *testing.T) {
	db := openMemDB(t)
	seedCoordinationAppuis(t, db.SQLDb())
	repo := NewCoordinationRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me", Gamertag: "MePlayer"})

	got, err := repo.LoadFragsOfficiels(context.Background(), "Me", []string{"m1", "m2", "absent"})
	if err != nil {
		t.Fatalf("LoadFragsOfficiels: %v", err)
	}
	if len(got) != 2 || got["m1"] != 9 || got["m2"] != 0 {
		t.Fatalf("frags officiels = %+v, attendu m1=9, m2=0", got)
	}
	ally, err := repo.LoadFragsOfficiels(context.Background(), "Ally", []string{"m1", "m2"})
	if err != nil || len(ally) != 1 || ally["m1"] != 3 {
		t.Fatalf("frags officiels d'Ally = %+v (%v), attendu m1=3 seul", ally, err)
	}
}

// TestCoordinationRepo_LoadAppuis_ScopeVide — aucune requête, aucune ligne.
func TestCoordinationRepo_LoadAppuis_ScopeVide(t *testing.T) {
	repo := NewCoordinationRepo(&PlayerDB{XUID: "Me"})
	got, err := repo.LoadAppuis(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("LoadAppuis(vide) = %v, %v — attendu aucune ligne et aucune erreur", got, err)
	}
}

// TestCoordinationRepo_LoadAppuis_TableAbsente — une base non migrée (ou un titre sans
// décodeur de film) dégrade en zéro ligne, jamais en erreur : ce n'est pas une panne.
func TestCoordinationRepo_LoadAppuis_TableAbsente(t *testing.T) {
	db := openMemDB(t)
	repo := NewCoordinationRepo(&PlayerDB{Player: db, Shared: db, XUID: "Me"})

	got, err := repo.LoadAppuis(context.Background(), []string{"m1"})
	if err != nil {
		t.Fatalf("LoadAppuis sur table absente = %v, attendu une dégradation silencieuse", err)
	}
	if len(got) != 0 {
		t.Fatalf("%d lignes sur table absente, attendu 0", len(got))
	}
}
