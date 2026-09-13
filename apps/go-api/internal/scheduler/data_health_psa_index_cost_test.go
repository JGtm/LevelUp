//go:build psarepro

// data_health_psa_index_cost_test.go — MESURE DU COÛT de la garde index PSA, sur
// un corpus de taille réaliste (1 100 matchs, ~15 awards/match). Build tag
// `psarepro` : hors gates normaux (le chiffre sert au rapport, pas au CI).
//
//	go test -tags=psarepro ./internal/scheduler/ -run TestPSAIndexGuardCost -v
package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func TestPSAIndexGuardCost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := migration.ExecScript(db, migration.PlayerPersonalScoreAwardsDDL); err != nil {
		t.Fatal(err)
	}
	if err := migration.EnsurePersonalScoreAwardsAppendOnly(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO personal_score_awards
			(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
		SELECT printf('%08x-0000-4000-8000-%012d', (m * 2654435761)%4294967296, m),
		       '2533274', printf('award_%d', a),
		       ['objective','combat','support','vehicle','misc'][(a % 5) + 1],
		       1, 100 + a, m
		FROM range(0, 1100) t1(m), range(0, 15) t2(a)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards`).Scan(&rows); err != nil {
		t.Fatal(err)
	}

	for _, sample := range []int{50, 200, 1000} {
		var total time.Duration
		const runs = 5
		var rep psaIndexReport
		for i := 0; i < runs; i++ {
			start := time.Now()
			r, err := scanPSAIndexDesync(ctx, db, sample)
			if err != nil {
				t.Fatalf("scan (sample=%d): %v", sample, err)
			}
			total += time.Since(start)
			rep = r
		}
		fmt.Printf("COUT sample=%-5d rows=%d cles=%d ecarts=%d  moyenne=%v\n",
			sample, rows, rep.KeysSampled, rep.KeysDiverging, (total / runs).Round(time.Millisecond))
	}
}
