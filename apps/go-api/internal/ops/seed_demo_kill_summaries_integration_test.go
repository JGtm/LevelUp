//go:build integration

package ops

// seed_demo_kill_summaries_integration_test.go — LES RÉSUMÉS PAR MATCH DU JOURNAL (ADR 0039)
// TRAVERSENT LA DÉMO ANONYMISÉS, avec la déclaration réelle (`killSummaryTables`) : aucun xuid ni
// gamertag réel ne reste sur S1 ni S2, le gamertag d'un assistant ou d'un tueur suit son xuid,
// un bot hors roster traverse tel quel, et la tête (sans identité) est copiée avec ses colonnes
// techniques.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func sourceResumesAnonymisation(t *testing.T, ctx context.Context) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "src.duckdb")
	src, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer src.Close()
	for _, q := range []string{
		`CREATE TABLE match_kill_heads (id BIGINT, match_id VARCHAR, summary_pass VARCHAR,
			summary_rev VARCHAR, src_decode_pass VARCHAR, written_at TIMESTAMP, publishable BOOLEAN,
			n_rows INTEGER, n_assist_known INTEGER, n_killer_xuid INTEGER, n_src_killer INTEGER)`,
		`CREATE TABLE match_kill_pairs (id BIGINT, match_id VARCHAR, summary_pass VARCHAR,
			feed_killer_xuid VARCHAR, victim_xuid VARCHAR, kills INTEGER)`,
		`CREATE TABLE match_assist_pairs (id BIGINT, match_id VARCHAR, summary_pass VARCHAR,
			assist_xuid VARCHAR, assist_gamertag VARCHAR, feed_killer_xuid VARCHAR,
			feed_killer_gamertag VARCHAR, victim_xuid VARCHAR, n INTEGER)`,
		`INSERT INTO match_kill_heads VALUES (7, 'm1', 'g1', 'ks1', 'p1', TIMESTAMP '2026-10-21 12:00:00',
			TRUE, 2, 2, 2, 0)`,
		`INSERT INTO match_kill_pairs VALUES (8, 'm1', 'g1', '` + anonReelA + `', '` + anonReelB + `', 2),
			(9, 'hors_corpus', 'g9', '` + anonReelA + `', '` + anonReelB + `', 1)`,
		`INSERT INTO match_assist_pairs VALUES
			(10, 'm1', 'g1', '` + anonReelB + `', 'VraiJoueurB', '` + anonReelA + `', 'VraiJoueurA', NULL, 1),
			(11, 'm1', 'g1', NULL, 'Bot Jackal', NULL, 'Bot Elite', '` + anonReelA + `', 1)`,
	} {
		if _, err := src.ExecContext(ctx, q); err != nil {
			t.Fatalf("fixture %q: %v", q, err)
		}
	}
	return path
}

func TestCopyAnonymizedTables_ResumesDuJournal(t *testing.T) {
	ctx := context.Background()
	srcPath := sourceResumesAnonymisation(t, ctx)
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "demo.duckdb"))
	if err != nil {
		t.Fatalf("open dst: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // l'ATTACH vit sur la connexion
	if _, err := db.ExecContext(ctx, "ATTACH '"+srcPath+"' AS src (READ_ONLY)"); err != nil {
		t.Fatalf("attach: %v", err)
	}
	roster := []demoRosterEntry{
		{SourceXUID: anonReelA, DemoXUID: "0000000000000000", DemoGamertag: "DemoPlayer", IsRosterMain: true},
		{SourceXUID: anonReelB, DemoXUID: "0000000000000001", DemoGamertag: "DemoPlayer2", IsRosterMain: true},
	}
	counts, err := copyAnonymizedTables(ctx, db, killSummaryTables, formatIDsLiteral([]string{"m1"}),
		rosterRemaps(roster), false)
	if err != nil {
		t.Fatalf("copyAnonymizedTables: %v", err)
	}
	for table, attendu := range map[string]int{tableMatchKillHeads: 1, tableMatchKillPairs: 1, tableMatchAssistPairs: 2} {
		if counts[table] != attendu {
			t.Errorf("%s : %d ligne(s) copiée(s), attendu %d", table, counts[table], attendu)
		}
	}
	for table, cols := range map[string][]string{
		tableMatchKillPairs:   {colFeedKillerXUID, colVictimXUID},
		tableMatchAssistPairs: {colAssistXUID, colAssistGamertag, colFeedKillerXUID, colFeedKillerGamertag, colVictimXUID},
	} {
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
	var assistant, tueur string
	if err := db.QueryRowContext(ctx, `SELECT assist_gamertag, feed_killer_gamertag FROM match_assist_pairs
		WHERE assist_xuid = '0000000000000001' AND feed_killer_xuid = '0000000000000000'`).Scan(&assistant, &tueur); err != nil {
		t.Fatalf("appui nommé : %v", err)
	}
	if assistant != "DemoPlayer2" || tueur != "DemoPlayer" {
		t.Errorf("gamertags démo (%q, %q), attendu appariés à leur xuid (DemoPlayer2, DemoPlayer)", assistant, tueur)
	}
	var botA, botK, victime string
	if err := db.QueryRowContext(ctx, `SELECT assist_gamertag, feed_killer_gamertag, victim_xuid
		FROM match_assist_pairs WHERE assist_xuid IS NULL`).Scan(&botA, &botK, &victime); err != nil {
		t.Fatalf("appui bot-bot : %v", err)
	}
	if botA != "Bot Jackal" || botK != "Bot Elite" || victime != "0000000000000000" {
		t.Errorf("appui bot-bot = (%q, %q, %q), attendu les bots intacts et la victime remappée", botA, botK, victime)
	}
	var id int64
	var rev string
	if err := db.QueryRowContext(ctx, `SELECT id, summary_rev FROM match_kill_heads`).Scan(&id, &rev); err != nil || id != 7 || rev != "ks1" {
		t.Errorf("tête copiée (%d, %q, err %v), attendu (7, ks1) avec ses colonnes techniques", id, rev, err)
	}
}
