//go:build integration

// Package ops — seed_demo_anonymize_integration_test.go : `copyAnonymizedTables` REMPLACE les
// identités réelles PENDANT la copie, sans jamais réécrire une table après coup.
//
// Cette fonction est le dernier rempart entre les xuid/gamertags RÉELS du parc et un jeu de
// données publié publiquement. CE QUE LE TEST EXIGE, dans cet ordre d'importance :
//  1. plus AUCUN xuid ni gamertag réel dans les tables copiées, sur TOUTES les colonnes
//     d'identité déclarées (y compris les trois paires de match_kill_events, dont l'assistant,
//     et le killer_xuid de kill_positions, table append-only) ;
//  2. les identités de DÉMO sont posées, et appariées (le bon gamertag avec le bon xuid) ;
//  3. le nombre de lignes est celui de la source — l'original n'entre jamais dans la copie ;
//  4. une identité HORS roster (bot, joueur non recensé) traverse telle quelle ;
//  5. une table source absente n'interrompt pas les autres ;
//  6. les colonnes techniques d'une table append-only copiée telle quelle (id, written_at,
//     decode_pass) survivent, et la table de correspondance (xuid RÉELS) ne reste pas.
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	anonReelA = "2533274801234567"
	anonReelB = "2533274809876543"
	anonBotID = "bid(1.2)"
)

// sourceAnonymisation : une base SOURCE minimale portant des identités réelles.
func sourceAnonymisation(t *testing.T, ctx context.Context) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "src.duckdb")
	src, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer src.Close()
	for _, q := range []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`CREATE TABLE match_kill_events (match_id VARCHAR,
			feed_killer_xuid VARCHAR, feed_killer_gamertag VARCHAR,
			victim_xuid VARCHAR, victim_gamertag VARCHAR,
			assist_xuid VARCHAR, assist_gamertag VARCHAR)`,
		`CREATE TABLE kill_positions (id BIGINT, match_id VARCHAR, killer_xuid VARCHAR,
			decode_pass VARCHAR, written_at TIMESTAMP)`,
		`INSERT INTO match_participants VALUES ('m1','` + anonReelA + `','VraiJoueurA'),
			('m1','` + anonReelB + `','VraiJoueurB'), ('m1','` + anonBotID + `','Bot Jackal'),
			('hors_corpus','` + anonReelA + `','VraiJoueurA')`,
		`INSERT INTO weapon_kills VALUES ('m1','` + anonReelA + `',7), ('m1','` + anonReelB + `',3)`,
		// L'ASSISTANT porte la troisième identité : c'est celle qu'un remap partiel oublie.
		`INSERT INTO match_kill_events VALUES ('m1','` + anonReelA + `','VraiJoueurA','` +
			anonReelB + `','VraiJoueurB','` + anonReelA + `','VraiJoueurA')`,
		`INSERT INTO kill_positions VALUES (41, 'm1', '` + anonReelB + `', 'p1', TIMESTAMP '2026-10-01 12:00:00')`,
	} {
		if _, err := src.ExecContext(ctx, q); err != nil {
			t.Fatalf("fixture %q: %v", q, err)
		}
	}
	return path
}

func TestCopyAnonymizedTables_RemplaceALaCopieSansIdentiteReelle(t *testing.T) {
	ctx := context.Background()
	srcPath := sourceAnonymisation(t, ctx)
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "demo.duckdb"))
	if err != nil {
		t.Fatalf("open dst: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // l'ATTACH vit sur la connexion
	if _, err := db.ExecContext(ctx, "ATTACH '"+srcPath+"' AS src (READ_ONLY)"); err != nil {
		t.Fatalf("attach: %v", err)
	}

	tables := []extractTable{
		{name: "match_participants", where: matchIDInClause, identity: [][2]string{{"xuid", "gamertag"}}},
		{name: "weapon_kills", where: matchIDInClause, identity: [][2]string{{"xuid", ""}}},
		{name: "match_kill_events", where: matchIDInClause, identity: [][2]string{
			{"feed_killer_xuid", "feed_killer_gamertag"}, {"victim_xuid", "victim_gamertag"},
			{"assist_xuid", "assist_gamertag"}}},
		// Absente de la source : (5).
		{name: "match_commendations", where: matchIDInClause, identity: [][2]string{{"xuid", ""}}},
		{name: "kill_positions", where: matchIDInClause, identity: [][2]string{{"killer_xuid", ""}}},
	}
	roster := []demoRosterEntry{
		{SourceXUID: anonReelA, DemoXUID: "0000000000000000", DemoGamertag: "DemoPlayer", IsRosterMain: true},
		{SourceXUID: anonReelB, DemoXUID: "0000000000000001", DemoGamertag: "DemoPlayer2", IsRosterMain: true},
	}
	counts, err := copyAnonymizedTables(ctx, db, tables, formatIDsLiteral([]string{"m1"}), rosterRemaps(roster), false)
	if err != nil {
		t.Fatalf("copyAnonymizedTables: %v", err)
	}

	verifierAucuneFuite(t, ctx, db)

	// (2) Identités de démo posées ET appariées.
	var gt string
	if err := db.QueryRowContext(ctx,
		`SELECT gamertag FROM match_participants WHERE xuid = '0000000000000000'`).Scan(&gt); err != nil || gt != "DemoPlayer" {
		t.Errorf("xuid démo 0 apparié au gamertag %q (err %v), attendu DemoPlayer", gt, err)
	}

	// (3) Nombre de lignes de la source (filtrée sur le corpus), (5) table absente comptée 0.
	for table, attendu := range map[string]int{
		"match_participants": 3, "weapon_kills": 2, "match_kill_events": 1, "kill_positions": 1,
		"match_commendations": 0,
	} {
		if counts[table] != attendu {
			t.Errorf("%s : %d ligne(s) copiée(s), attendu %d", table, counts[table], attendu)
		}
	}

	// (4) Le bot hors roster traverse intact.
	var nBot int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM match_participants WHERE xuid = ? AND gamertag = 'Bot Jackal'`, anonBotID).Scan(&nBot); err != nil || nBot != 1 {
		t.Errorf("bot hors roster : %d ligne(s) (err %v), attendu 1", nBot, err)
	}

	// (6) Colonnes techniques de kill_positions conservées ; table de correspondance retirée.
	var id int64
	var pass, killer string
	if err := db.QueryRowContext(ctx,
		`SELECT id, decode_pass, killer_xuid FROM kill_positions WHERE written_at IS NOT NULL`).Scan(&id, &pass, &killer); err != nil {
		t.Fatalf("kill_positions : %v", err)
	}
	if id != 41 || pass != "p1" || killer != "0000000000000001" {
		t.Errorf("kill_positions = (%d, %q, %q), attendu (41, p1, 0000000000000001)", id, pass, killer)
	}
	var nMap int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM duckdb_tables() WHERE table_name = ?`, demoXUIDMapTable).Scan(&nMap); err != nil || nMap != 0 {
		t.Errorf("table de correspondance encore présente (%d, err %v) : elle porte les xuid RÉELS", nMap, err)
	}
}

// verifierAucuneFuite : (1) aucune identité réelle sur aucune colonne d'identité copiée.
func verifierAucuneFuite(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	colonnes := map[string][]string{
		"match_participants": {"xuid", "gamertag"},
		"weapon_kills":       {"xuid"},
		"match_kill_events": {
			"feed_killer_xuid", "feed_killer_gamertag", "victim_xuid", "victim_gamertag",
			"assist_xuid", "assist_gamertag",
		},
		"kill_positions": {"killer_xuid"},
	}
	for table, cols := range colonnes {
		for _, col := range cols {
			for _, fuite := range []string{anonReelA, anonReelB, "VraiJoueurA", "VraiJoueurB"} {
				var n int
				if err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+table+` WHERE `+col+` = ?`, fuite).Scan(&n); err != nil {
					t.Fatalf("%s.%s: %v", table, col, err)
				}
				if n != 0 {
					t.Errorf("FUITE : %s.%s porte %d fois l'identité réelle %q", table, col, n, fuite)
				}
			}
		}
	}
}
