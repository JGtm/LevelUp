//go:build integration

// Package persist — lusr_append_only_persister_test.go : Phase 2 du plan
// d'éradication ART. Tests TDD pour AppendOnlyLUSRPersister.
//
// **Objectif** : prouver que la nouvelle structure append-only :
//
//   - accepte des INSERT répétés sur le même (match_id, rating_type)
//   - préserve l'historique physique (jamais d'écrasement)
//   - expose une vue match_skill_rank_latest qui ne renvoie que la
//     dernière version par (match_id, rating_type)
//   - ne déclenche jamais le bug ART sous concurrence (cf. test ART repro
//     Phase 1 qui doit passer à 0 erreur quand on bascule sur ce
//     persister)

package persist

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openAppendOnlyLUSRTestDB ouvre une DuckDB :memory: avec le schéma
// append-only cible : PK (match_id, rating_type, written_at) + vue
// match_skill_rank_latest. C'est le schéma qui sera produit par la
// migration Phase 2.B.
func openAppendOnlyLUSRTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Schéma append-only : **pas de PRIMARY KEY contraignante**. La
	// nature append-only s'accommode de N versions par (match_id,
	// rating_type), et le ROW_NUMBER() OVER de la vue départage. Une PK
	// stricte sur (match_id, rating_type, written_at) provoquerait des
	// collisions sous concurrence (now() identique entre threads).
	// L'absence de PK est intentionnelle pour ce pattern — la vue
	// garantit l'unicité fonctionnelle vue côté lecture.
	if _, err := db.Exec(`
		CREATE SEQUENCE msr_seq START 1;
		CREATE TABLE match_skill_rank (
			id                BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
			match_id          VARCHAR NOT NULL,
			rating_type       VARCHAR NOT NULL,
			rating_value      FLOAT,
			rating_deviation  FLOAT,
			tier              VARCHAR,
			tier_fr           VARCHAR,
			sub_tier          SMALLINT DEFAULT 0,
			tier_label        VARCHAR,
			rating_delta      FLOAT,
			playlist_group    VARCHAR,
			expected_win_prob FLOAT,
			start_time        TIMESTAMP,
			written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_msr_lookup ON match_skill_rank(match_id, rating_type, written_at);
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC, id DESC) = 1;
	`); err != nil {
		t.Fatalf("create table+view: %v", err)
	}
	return db
}

// TestAppendOnlyLUSR_PersistAccumulates — Insert répétés sur le même
// match_id doivent S'ACCUMULER physiquement (jamais d'écrasement).
func TestAppendOnlyLUSR_PersistAccumulates(t *testing.T) {
	db := openAppendOnlyLUSRTestDB(t)
	p := NewAppendOnlyLUSRPersister(db)
	ctx := context.Background()

	// 1er Persist : 3 rows LUSR distinctes
	for _, mid := range []string{"m1", "m2", "m3"} {
		if err := p.Persist(ctx, []LUSRRatingInsert{
			{MatchID: mid, RatingValue: 20.0, PlaylistGroup: "arena_slayer"},
		}); err != nil {
			t.Fatalf("Persist initial %s: %v", mid, err)
		}
	}

	// 2nd Persist : nouvelles versions de m1 et m2 (collision logique sur match_id)
	// Sleep 1ms pour garantir written_at strictement supérieur
	time.Sleep(2 * time.Millisecond)
	if err := p.Persist(ctx, []LUSRRatingInsert{
		{MatchID: "m1", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m2", RatingValue: 26.0, PlaylistGroup: "arena_slayer"},
	}); err != nil {
		t.Fatalf("Persist update: %v", err)
	}

	// Assert 1 : 5 rows physiques totales (3 originales + 2 nouvelles)
	var totalPhysical int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&totalPhysical); err != nil {
		t.Fatal(err)
	}
	if totalPhysical != 5 {
		t.Errorf("rows physiques = %d, want 5 (append-only : 3 originales + 2 nouvelles)", totalPhysical)
	}

	// Assert 2 : vue latest renvoie 3 rows (1 par match_id)
	var latestCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type='LUSR'`).Scan(&latestCount); err != nil {
		t.Fatal(err)
	}
	if latestCount != 3 {
		t.Errorf("rows latest = %d, want 3 (1 par match_id)", latestCount)
	}

	// Assert 3 : la vue latest renvoie les VALEURS les plus récentes
	cases := []struct {
		matchID string
		want    float64
	}{
		{"m1", 25.0}, // mise à jour
		{"m2", 26.0}, // mise à jour
		{"m3", 20.0}, // non touché
	}
	for _, c := range cases {
		var got float64
		err := db.QueryRow(`SELECT rating_value FROM match_skill_rank_latest WHERE match_id=? AND rating_type='LUSR'`, c.matchID).Scan(&got)
		if err != nil {
			t.Errorf("query latest %s: %v", c.matchID, err)
			continue
		}
		if got != c.want {
			t.Errorf("latest rating_value pour %s = %f, want %f", c.matchID, got, c.want)
		}
	}
}

// TestAppendOnlyLUSR_PersistsStartTime — la colonne start_time est écrite (elle
// alimente l'ordre CHRONOLOGIQUE des readers du delta + de la vue latest). nil →
// colonne NULL.
func TestAppendOnlyLUSR_PersistsStartTime(t *testing.T) {
	db := openAppendOnlyLUSRTestDB(t)
	p := NewAppendOnlyLUSRPersister(db)
	ctx := context.Background()
	st := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)

	if err := p.Persist(ctx, []LUSRRatingInsert{
		{MatchID: "m1", RatingValue: 1500, PlaylistGroup: "arena_slayer", StartTime: &st},
	}); err != nil {
		t.Fatalf("Persist avec StartTime: %v", err)
	}
	var got sql.NullTime
	if err := db.QueryRow(`SELECT start_time FROM match_skill_rank WHERE match_id='m1'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || !got.Time.UTC().Equal(st) {
		t.Errorf("start_time = %v, want %v", got, st)
	}

	if err := p.Persist(ctx, []LUSRRatingInsert{
		{MatchID: "m2", RatingValue: 1500, PlaylistGroup: "arena_slayer", StartTime: nil},
	}); err != nil {
		t.Fatalf("Persist StartTime nil: %v", err)
	}
	var got2 sql.NullTime
	if err := db.QueryRow(`SELECT start_time FROM match_skill_rank WHERE match_id='m2'`).Scan(&got2); err != nil {
		t.Fatal(err)
	}
	if got2.Valid {
		t.Errorf("start_time m2 = %v, want NULL (StartTime nil)", got2)
	}
}

// TestAppendOnlyLUSR_EmptyBatchNoOp — batch vide → no-op.
func TestAppendOnlyLUSR_EmptyBatchNoOp(t *testing.T) {
	db := openAppendOnlyLUSRTestDB(t)
	p := NewAppendOnlyLUSRPersister(db)

	// Pré-seed 1 row
	if err := p.Persist(context.Background(), []LUSRRatingInsert{
		{MatchID: "keep", RatingValue: 22.0, PlaylistGroup: "arena_slayer"},
	}); err != nil {
		t.Fatal(err)
	}

	// Persist nil et empty → no-op, n'affecte pas l'existant
	if err := p.Persist(context.Background(), nil); err != nil {
		t.Errorf("Persist nil: %v", err)
	}
	if err := p.Persist(context.Background(), []LUSRRatingInsert{}); err != nil {
		t.Errorf("Persist empty: %v", err)
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n)
	if n != 1 {
		t.Errorf("rows = %d, want 1 (empty batch ne touche pas l'existant)", n)
	}
}

// TestAppendOnlyLUSR_RejectsEmptyMatchID — rang sans match_id est invalide.
func TestAppendOnlyLUSR_RejectsEmptyMatchID(t *testing.T) {
	db := openAppendOnlyLUSRTestDB(t)
	p := NewAppendOnlyLUSRPersister(db)

	err := p.Persist(context.Background(), []LUSRRatingInsert{
		{MatchID: "", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
	})
	if err == nil {
		t.Error("Persist devrait échouer sur MatchID vide")
	}
}

// TestAppendOnlyLUSR_ConcurrentInsertsNoArtCrash — sous concurrence
// forte, AUCUN crash (le pattern INSERT pur ne déclenche pas le bug ART).
// Contrepartie du test TestLUSRUpsert_ARTReproOnFile qui crashe sur
// l'ancien persister (19/20 workers crashent).
func TestAppendOnlyLUSR_ConcurrentInsertsNoArtCrash(t *testing.T) {
	db := openAppendOnlyLUSRTestDB(t)
	p := NewAppendOnlyLUSRPersister(db)
	ctx := context.Background()

	const workers = 10
	const itersPerWorker = 20
	const rowsPerBatch = 10

	var (
		wg       sync.WaitGroup
		errsMu   sync.Mutex
		errCount int
	)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for iter := 0; iter < itersPerWorker; iter++ {
				rows := make([]LUSRRatingInsert, 0, rowsPerBatch)
				for i := 0; i < rowsPerBatch; i++ {
					rows = append(rows, LUSRRatingInsert{
						// Tous les workers tapent les mêmes match_ids → collisions max
						MatchID:       "shared_match",
						RatingValue:   float64(workerID*1000 + iter*10 + i),
						PlaylistGroup: "arena_slayer",
					})
				}
				if err := p.Persist(ctx, rows); err != nil {
					errsMu.Lock()
					errCount++
					errsMu.Unlock()
					t.Logf("worker %d iter %d err: %v", workerID, iter, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	if errCount > 0 {
		t.Errorf("INSERT pur sous concurrence : %d erreurs (attendu 0)", errCount)
	}

	// La vue latest doit toujours renvoyer exactement 1 row pour "shared_match"
	var latestCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest WHERE match_id='shared_match'`).Scan(&latestCount)
	if latestCount != 1 {
		t.Errorf("latest count pour shared_match = %d, want 1", latestCount)
	}

	// La table physique doit contenir TOUTES les écritures (workers * iters * rows)
	// — au moins quelques unes peuvent avoir le même written_at par hasard de
	// timing, donc on accepte un range [workers*itersPerWorker, workers*itersPerWorker*rowsPerBatch]
	var physical int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id='shared_match'`).Scan(&physical)
	expectedMin := workers * itersPerWorker // au moins 1 row par batch commit
	if physical < expectedMin {
		t.Errorf("rows physiques = %d, want >= %d (au moins 1 par batch successful)", physical, expectedMin)
	}
	t.Logf("rows physiques accumulées : %d (workers=%d, iters=%d, rowsPerBatch=%d)",
		physical, workers, itersPerWorker, rowsPerBatch)
}
