//go:build integration

// Package duckdb — coordination_repo_test.go : QCoordinationAppuis sur DuckDB `:memory:`.
//
// Les trois frontières de la requête, et pourquoi chacune compte :
//   - `publishable` et `assist_known` écartent les lignes dont l'assistance n'est pas
//     mesurée ligne à ligne : elles ne doivent entrer dans AUCUN dénominateur ;
//   - un assistant NULL sort en chaîne VIDE, et c'est l'état MESURÉ « personne n'a
//     assisté » — c'est lui qui porte le dénominateur « mes frags mesurés », que Q21d
//     écarte parce qu'une PAIRE ne sait pas le représenter ;
//   - un tueur sans xuid (bot) ne sort pas : l'agréger fusionnerait tous les bots en un
//     joueur fantôme.
package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

// seedCoordinationAppuis : le schéma minimal de la requête et un jeu couvrant les bornes.
//
//	m1  Me tue 2 fois sans assistant, 2 fois assisté par Ally, 1 fois par Bis ;
//	    une ligne NON publiable et une ligne assist_known = FALSE sont ignorées ;
//	    une ligne sans tueur identifié (bot) est ignorée.
//	m2  Ally tue 1 fois, assisté par Me — l'appui que JE distribue.
//	m3  hors périmètre demandé.
func seedCoordinationAppuis(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE match_kill_events_latest (
			match_id VARCHAR, publishable BOOLEAN, assist_known BOOLEAN,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR,
			assist_xuid VARCHAR, time_ms INTEGER)`,
		`INSERT INTO match_kill_events_latest VALUES
			('m1',TRUE ,TRUE ,'Me' ,'Foe',NULL  ,1),
			('m1',TRUE ,TRUE ,'Me' ,'Foe',NULL  ,2),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Ally',3),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Ally',4),
			('m1',TRUE ,TRUE ,'Me' ,'Foe','Bis' ,5),
			('m1',FALSE,TRUE ,'Me' ,'Foe','Ally',6),
			('m1',TRUE ,FALSE,'Me' ,'Foe',NULL  ,7),
			('m1',TRUE ,TRUE ,NULL ,'Foe','Ally',8),
			('m2',TRUE ,TRUE ,'Ally','Foe','Me' ,1),
			('m3',TRUE ,TRUE ,'Me' ,'Foe','Ally',1)`,
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

	type cle struct{ match, assist, killer string }
	vu := map[cle]int{}
	for _, r := range got {
		vu[cle{r.MatchID, r.AssistXUID, r.KillerXUID}] = r.Nombre
	}
	attendus := map[cle]int{
		// L'assistant NULL sort en chaîne VIDE : deux frags MESURÉS sans assistant.
		{"m1", "", "Me"}:     2,
		{"m1", "Ally", "Me"}: 2,
		{"m1", "Bis", "Me"}:  1,
		{"m2", "Me", "Ally"}: 1,
	}
	if len(vu) != len(attendus) {
		t.Fatalf("%d lignes, attendu %d : %+v", len(vu), len(attendus), got)
	}
	for k, want := range attendus {
		if vu[k] != want {
			t.Errorf("%+v = %d, attendu %d — la ligne non publiable, celle dont l'assistance "+
				"n'est pas connue, celle sans tueur identifié et le match hors périmètre "+
				"ne doivent pas y entrer", k, vu[k], want)
		}
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
