//go:build integration

package main

// cmd_backfill_match_summaries_integration_test.go — LE RATTRAPAGE DES RÉSUMÉS PAR MATCH
// (`levelup backfill-match-summaries`, ADR 0039 décision 5) sur des bases temporaires : seuls les
// matchs dont la passe courante n'a pas de tête à jour sont écrits, un second passage ne trouve
// rien, `--dry-run` n'écrit rien, un échec se compte sans arrêter la passe.

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/testutil"
)

func baseResumesMigree(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

// ecrirePasseDeTest écrit une passe de deux morts (le tueur nommé de la première, un bot pour la
// seconde) par l'écrivain réel : sa tête et ses filles suivent dans la même transaction.
func ecrirePasseDeTest(t *testing.T, db *sql.DB, matchID string, morts int) {
	t.Helper()
	deaths := make([]persist.KillEventInsert, morts)
	for i := range deaths {
		deaths[i] = persist.KillEventInsert{
			TimeMS: 1000 * (i + 1), VictimGamertag: "Victime", VictimXUID: "2222",
			FeedKillerGamertag: "Tueur", FeedKillerXUID: "1111", FeedPresent: true, AssistKnown: true,
			ReadPath: killscope.ReadPathFilmWalk, ReadOrigin: "credit-concordant",
		}
	}
	err := persist.NewKillSourcePersister(db).PersistPass(context.Background(),
		persist.KillSourceBatch{MatchID: matchID, DecoderRev: "killsource-test", Publishable: true, Deaths: deaths})
	if err != nil {
		t.Fatalf("passe %s : %v", matchID, err)
	}
}

func compter(t *testing.T, db *sql.DB, requete string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(requete, args...).Scan(&n); err != nil {
		t.Fatalf("%s : %v", requete, err)
	}
	return n
}

// corpusARattraper : "sans-tete" (têtes effacées), "ancienne-rev" (tête à une autre révision),
// "passe-neuve" (deux passes, tête seulement pour la plus ancienne), "a-jour" (tête courante).
func corpusARattraper(t *testing.T) *sql.DB {
	t.Helper()
	db := baseResumesMigree(t)
	for _, id := range []string{"sans-tete", "ancienne-rev", "passe-neuve", "a-jour"} {
		ecrirePasseDeTest(t, db, id, 3)
	}
	ecrirePasseDeTest(t, db, "passe-neuve", 2)
	if _, err := db.Exec(`DELETE FROM match_kill_heads WHERE match_id = 'sans-tete'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE match_kill_heads SET summary_rev = 'ks0-ancienne' WHERE match_id = 'ancienne-rev'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM match_kill_heads WHERE match_id = 'passe-neuve'
		AND src_decode_pass = (SELECT any_value(decode_pass) FROM match_kill_events_latest
		                       WHERE match_id = 'passe-neuve')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRattraperResumes_SeulsLesMatchsSansTeteAJourSontEcrits(t *testing.T) {
	ctx := context.Background()
	db := corpusARattraper(t)
	avant := compter(t, db, `SELECT COUNT(*) FROM match_kill_heads WHERE match_id = 'a-jour'`)

	b, err := rattraperResumes(ctx, db, matchSummariesOptions{titleSlug: "halo_infinite", batch: 1})
	if err != nil {
		t.Fatalf("rattraperResumes : %v", err)
	}
	if b.aRattraper != 3 || b.ecrits != 3 || b.echecs != 0 || b.disparus != 0 || b.restants != 0 {
		t.Fatalf("bilan = %+v, attendu 3 à rattraper, 3 écrits, 0 reste", b)
	}
	for _, id := range []string{"sans-tete", "ancienne-rev", "passe-neuve", "a-jour"} {
		testutil.ExigerResumeDeLaPasseCourante(t, db, id)
	}
	if n := compter(t, db, `SELECT COUNT(*) FROM match_kill_heads WHERE match_id = 'a-jour'`); n != avant {
		t.Fatalf("a-jour : %d tête(s) en table, avait %d — un match à jour ne doit pas être réécrit", n, avant)
	}
	if n := compter(t, db, `SELECT COUNT(*) FROM match_kill_heads WHERE match_id = 'ancienne-rev'`); n != 2 {
		t.Fatalf("ancienne-rev : %d tête(s), attendu 2 (l'ancienne reste, rien n'est effacé)", n)
	}

	// Second passage : rien à faire.
	b2, err := rattraperResumes(ctx, db, matchSummariesOptions{titleSlug: "halo_infinite", batch: 250})
	if err != nil || b2.aRattraper != 0 || b2.ecrits != 0 || b2.restants != 0 {
		t.Fatalf("second passage = %+v (err %v), attendu rien à faire", b2, err)
	}
}

func TestRattraperResumes_DryRunNEcritRien(t *testing.T) {
	db := corpusARattraper(t)
	avant := compter(t, db, `SELECT COUNT(*) FROM match_kill_heads`)
	b, err := rattraperResumes(context.Background(), db, matchSummariesOptions{titleSlug: "halo_infinite", dryRun: true, batch: 250})
	if err != nil || b.aRattraper != 3 || b.ecrits != 0 {
		t.Fatalf("dry-run = %+v (err %v), attendu 3 à rattraper et rien d'écrit", b, err)
	}
	if n := compter(t, db, `SELECT COUNT(*) FROM match_kill_heads`); n != avant {
		t.Fatalf("%d têtes après --dry-run, avait %d", n, avant)
	}
	if n, err := persist.CountPendingKillSummaries(context.Background(), db); err != nil || n != 3 {
		t.Fatalf("jauge = %d (err %v), attendu 3 inchangée", n, err)
	}
}

// TestRattraperResumes_UnEchecSeCompteEtNeStoppePas : l'écriture des filles échoue pour chaque
// match (table retirée) ; la passe va au bout, compte les échecs, rend une erreur, et la jauge
// dit qu'il reste tout à faire.
func TestRattraperResumes_UnEchecSeCompteEtNeStoppePas(t *testing.T) {
	db := corpusARattraper(t)
	testutil.CasserLaTable(t, db, "match_kill_pairs")
	b, err := rattraperResumes(context.Background(), db, matchSummariesOptions{titleSlug: "halo_infinite", batch: 2})
	if err == nil || !strings.Contains(err.Error(), "en échec") {
		t.Fatalf("erreur = %v, attendu une erreur d'échec", err)
	}
	if b.aRattraper != 3 || b.echecs != 3 || b.ecrits != 0 {
		t.Fatalf("bilan = %+v, attendu 3 échecs comptés sans interruption", b)
	}
}

func TestErreurOuvertureEcriturePartagee_DitQueLaBaseEstTenue(t *testing.T) {
	err := erreurOuvertureEcriturePartagee("x.duckdb",
		errString("IO Error: Could not set lock on file \"x.duckdb\": Conflicting lock is held"))
	if !strings.Contains(err.Error(), "tenue par un autre processus") {
		t.Fatalf("message = %q, attendu l'explication de la base tenue", err)
	}
	if autre := erreurOuvertureEcriturePartagee("x.duckdb", errString("disque plein")); strings.Contains(autre.Error(), "tenue par un autre processus") {
		t.Fatalf("message = %q : une autre cause ne doit pas se dire « base tenue »", autre)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
