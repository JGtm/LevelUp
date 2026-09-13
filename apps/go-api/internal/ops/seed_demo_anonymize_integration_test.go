//go:build integration

// Package ops — seed_demo_anonymize_integration_test.go : `applyUniversalAnonymization`
// REMPLACE les identites reelles, et le fait sans UPDATE set-based.
//
// POURQUOI CE TEST EXISTE (2026-09-13, G6). Cette fonction est le dernier rempart entre les
// xuid/gamertags REELS du parc et un jeu de donnees publie publiquement, et elle n'avait
// aucun test direct — seul le bout-en-bout de SeedDemo la traversait. Elle vient de changer
// de forme d'ecriture (un UPDATE set-based joint a `_xuid_map` -> N UPDATE row-by-row a
// valeurs liees, pour sortir de la forme declencheuse du bug ART #23645) : un changement de
// chemin d'ecriture sur une fonction d'anonymisation sans test est exactement ce qu'on ne
// fait pas.
//
// CE QUE LE TEST EXIGE, ET DANS CET ORDRE D'IMPORTANCE :
//  1. plus AUCUN xuid ni gamertag reel dans les tables, sur TOUTES les colonnes d'identite
//     (y compris les trois paires de match_kill_events, dont l'assistant) ;
//  2. les identites de DEMO sont bien posees, et appariees (le bon gamertag avec le bon xuid) ;
//  3. le nombre de lignes est INCHANGE — une anonymisation qui ajoute une ligne laisserait
//     l'originale, donc la fuite, ce qui est le contresens que la conversion en INSERT-only
//     aurait produit ;
//  4. une identite HORS roster (bot, joueur non recense) est laissee telle quelle ;
//  5. une table absente (cas Halo 5 sur une demo Infinite) n'interrompt pas les autres.
package ops

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestApplyUniversalAnonymization_RemplaceEtNeLaisseAucuneIdentiteReelle(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	// Deux tables a une paire, une table a TROIS paires, et une table sans equivalent
	// Infinite volontairement NON creee (kill_positions) pour exercer errIsMissingTable.
	ddl := []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`CREATE TABLE match_kill_events (
			match_id VARCHAR,
			feed_killer_xuid VARCHAR, feed_killer_gamertag VARCHAR,
			victim_xuid VARCHAR, victim_gamertag VARCHAR,
			assist_xuid VARCHAR, assist_gamertag VARCHAR
		)`,
	}
	for _, q := range ddl {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("DDL %q: %v", q, err)
		}
	}

	const (
		reelA = "2533274801234567"
		reelB = "2533274809876543"
		botID = "bid(1.2)"
	)
	fixtures := []string{
		`INSERT INTO match_participants VALUES ('m1','` + reelA + `','VraiJoueurA')`,
		`INSERT INTO match_participants VALUES ('m1','` + reelB + `','VraiJoueurB')`,
		`INSERT INTO match_participants VALUES ('m1','` + botID + `','Bot Jackal')`,
		`INSERT INTO weapon_kills VALUES ('m1','` + reelA + `',7)`,
		`INSERT INTO weapon_kills VALUES ('m1','` + reelB + `',3)`,
		// L'ASSISTANT porte la troisieme identite : c'est celle qu'un remap partiel oublie.
		`INSERT INTO match_kill_events VALUES ('m1','` + reelA + `','VraiJoueurA','` +
			reelB + `','VraiJoueurB','` + reelA + `','VraiJoueurA')`,
	}
	for _, q := range fixtures {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("fixture %q: %v", q, err)
		}
	}

	roster := []demoRosterEntry{
		{SourceXUID: reelA, DemoXUID: "0000000000000000", DemoGamertag: "DemoPlayer", IsRosterMain: true},
		{SourceXUID: reelB, DemoXUID: "0000000000000001", DemoGamertag: "DemoPlayer2", IsRosterMain: true},
	}

	if err := applyUniversalAnonymization(ctx, db, roster); err != nil {
		t.Fatalf("applyUniversalAnonymization: %v", err)
	}

	// (1) AUCUNE identite reelle ne subsiste, sur aucune colonne d'aucune table.
	colonnes := map[string][]string{
		"match_participants": {"xuid", "gamertag"},
		"weapon_kills":       {"xuid"},
		"match_kill_events": {
			"feed_killer_xuid", "feed_killer_gamertag",
			"victim_xuid", "victim_gamertag",
			"assist_xuid", "assist_gamertag",
		},
	}
	for table, cols := range colonnes {
		for _, col := range cols {
			for _, fuite := range []string{reelA, reelB, "VraiJoueurA", "VraiJoueurB"} {
				var n int
				q := `SELECT count(*) FROM ` + table + ` WHERE ` + col + ` = ?`
				if err := db.QueryRowContext(ctx, q, fuite).Scan(&n); err != nil {
					t.Fatalf("%s.%s: %v", table, col, err)
				}
				if n != 0 {
					t.Errorf("FUITE : %s.%s porte encore %d fois l'identite reelle %q",
						table, col, n, fuite)
				}
			}
		}
	}

	// (2) Les identites de demo sont posees ET appariees.
	var gt string
	if err := db.QueryRowContext(ctx,
		`SELECT gamertag FROM match_participants WHERE xuid = '0000000000000000'`).Scan(&gt); err != nil {
		t.Fatalf("lecture de l'identite demo: %v", err)
	}
	if gt != "DemoPlayer" {
		t.Errorf("xuid demo 0 apparie au gamertag %q, attendu DemoPlayer", gt)
	}

	// (3) Le nombre de lignes est INCHANGE : on remplace, on n'ajoute pas.
	for table, attendu := range map[string]int{
		"match_participants": 3, "weapon_kills": 2, "match_kill_events": 1,
	} {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != attendu {
			t.Errorf("%s porte %d ligne(s), attendu %d — une anonymisation qui AJOUTE une "+
				"ligne laisse l'originale, donc la fuite", table, n, attendu)
		}
	}

	// (4) Une identite hors roster (bot) traverse intacte.
	var nBot int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM match_participants WHERE xuid = ?`, botID).Scan(&nBot); err != nil {
		t.Fatalf("count bot: %v", err)
	}
	if nBot != 1 {
		t.Errorf("le bot hors roster a %d ligne(s), attendu 1 (laisse tel quel)", nBot)
	}

	// (5) `kill_positions` n'a jamais ete creee : son absence ne doit pas avoir fait
	//     echouer l'appel (verifie par le t.Fatalf ci-dessus) ni empeche les tables
	//     suivantes d'etre traitees (verifie par les assertions (1) et (2)).
}
