package scheduler_test

// data_health_kill_summaries_e2e_test.go — LA JAUGE « PASSE COURANTE SANS RÉSUMÉS À JOUR »
// (ADR 0039, décision 6) du contrôle de santé des données : nulle quand chaque match a sa tête,
// non nulle pour une tête absente, à une ancienne révision ou qui résume une passe supplantée,
// publiée en expvar, et ramenée à zéro par la réécriture des résumés.

import (
	"context"
	"expvar"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// ecrirePasseSante écrit une passe de `morts` morts par l'écrivain réel (tête et filles incluses).
func ecrirePasseSante(t *testing.T, db *duckdb.DB, matchID string, morts int) {
	t.Helper()
	deaths := make([]persist.KillEventInsert, morts)
	for i := range deaths {
		deaths[i] = persist.KillEventInsert{
			TimeMS: 1000 * (i + 1), VictimGamertag: "Victime", VictimXUID: "2222",
			FeedKillerGamertag: "Tueur", FeedKillerXUID: "1111", FeedPresent: true, AssistKnown: true,
			ReadPath: killscope.ReadPathFilmWalk, ReadOrigin: "credit-concordant",
		}
	}
	err := persist.NewKillSourcePersister(db.SQLDb()).PersistPass(context.Background(),
		persist.KillSourceBatch{MatchID: matchID, DecoderRev: "killsource-test", Publishable: true, Deaths: deaths})
	if err != nil {
		t.Fatalf("passe %s : %v", matchID, err)
	}
}

// avecBasePartagee ouvre la base partagée en écriture le temps de `f`, puis la ferme (le contrôle
// de santé la rouvre en lecture seule).
func avecBasePartagee(t *testing.T, repoRoot string, f func(db *duckdb.DB)) {
	t.Helper()
	path := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open shared rw: %v", err)
	}
	defer db.Close()
	f(db)
}

func jaugeResumes(t *testing.T) int64 {
	t.Helper()
	m, ok := expvar.Get("levelup.kill_summaries_pending").(*expvar.Map)
	if !ok {
		t.Fatal("expvar levelup.kill_summaries_pending absent")
	}
	v, ok := m.Get("halo_infinite").(*expvar.Int)
	if !ok {
		t.Fatal("jauge halo_infinite non publiée")
	}
	return v.Value()
}

func TestHealthScheduler_E2E_ResumesDuJournal_Jauge(t *testing.T) {
	ctx := context.Background()
	repoRoot := healthE2ESetup(t)
	avecBasePartagee(t, repoRoot, func(db *duckdb.DB) {
		for _, id := range []string{"a-jour", "sans-tete", "ancienne-rev", "passe-supplantee"} {
			ecrirePasseSante(t, db, id, 2)
		}
		ecrirePasseSante(t, db, "passe-supplantee", 3)
	})
	sched := scheduler.NewDataHealthScheduler(repoRoot)

	res := sched.RunOnce(ctx)
	if res.ProbeErrors != 0 || res.KillSummariesPending != 0 || jaugeResumes(t) != 0 {
		t.Fatalf("base complète : sondes en échec %d, jauge %d / expvar %d, attendu 0 partout",
			res.ProbeErrors, res.KillSummariesPending, jaugeResumes(t))
	}

	avecBasePartagee(t, repoRoot, func(db *duckdb.DB) {
		for _, q := range []string{
			`DELETE FROM match_kill_heads WHERE match_id = 'sans-tete'`,
			`UPDATE match_kill_heads SET summary_rev = 'ks0-ancienne' WHERE match_id = 'ancienne-rev'`,
			`DELETE FROM match_kill_heads WHERE match_id = 'passe-supplantee'
			   AND src_decode_pass = (SELECT any_value(decode_pass) FROM match_kill_events_latest
			                          WHERE match_id = 'passe-supplantee')`,
		} {
			if _, err := db.Exec(ctx, q); err != nil {
				t.Fatalf("%s : %v", q, err)
			}
		}
	})
	res = sched.RunOnce(ctx)
	if res.KillSummariesPending != 3 || jaugeResumes(t) != 3 {
		t.Fatalf("jauge %d / expvar %d, attendu 3 (tête absente, ancienne révision, passe supplantée)",
			res.KillSummariesPending, jaugeResumes(t))
	}
	if res.WarningsTotal != 0 {
		t.Fatalf("WarningsTotal = %d : la jauge est un signal distinct, hors du total", res.WarningsTotal)
	}

	avecBasePartagee(t, repoRoot, func(db *duckdb.DB) {
		pending, err := persist.PendingKillSummaryMatches(ctx, db.SQLDb())
		if err != nil || len(pending) != 3 {
			t.Fatalf("matchs à rattraper = %v (err %v), attendu 3", pending, err)
		}
		for _, id := range pending {
			if ecrit, err := persist.PersistLatestKillPassSummaries(ctx, db.SQLDb(), id); err != nil || !ecrit {
				t.Fatalf("rattrapage %s : écrit=%v err=%v", id, ecrit, err)
			}
		}
	})
	res = sched.RunOnce(ctx)
	if res.KillSummariesPending != 0 || jaugeResumes(t) != 0 {
		t.Fatalf("après rattrapage : jauge %d / expvar %d, attendu 0", res.KillSummariesPending, jaugeResumes(t))
	}
}
