//go:build psarepro

// Suite du harnais d_enquete jetable psa_index_repro_test.go (build tag
// `psarepro`, EXCLU des gates). Voir ce fichier pour la methode et les helpers.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	gosync "sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── S11 : ecriture OPTIMISTE (> 1 row group en UNE transaction) + crash sans
// CHECKPOINT => rejeu WAL ReplayRowGroupData. Mecanisme de duckdb/duckdb PR
// #24744 (« the inserts that should have been in the index weren't there, hence
// the index was corrupted »), mergee APRES v1.5.5 donc ABSENTE de notre version.
func TestPSAReproS11OptimisticWriteWalReplay(t *testing.T) {
	if os.Getenv("PSA_REPRO_OPTIMISTIC") != "" {
		reproOptimisticChild()
		return
	}
	dir, err := os.MkdirTemp("", "psarepro11")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "stats.duckdb")

	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	reproBulkFill(t, db, 1100, 4)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, reproMatchID(7))
	reproReport(t, db, "S11 socle")
	db.Close()

	for cycle := 0; cycle < 3; cycle++ {
		cmd := exec.Command(os.Args[0], "-test.run", "TestPSAReproS11OptimisticWriteWalReplay")
		cmd.Env = append(os.Environ(), "PSA_REPRO_OPTIMISTIC="+path)
		out, _ := cmd.CombinedOutput()
		t.Logf("  cycle %d enfant: %s", cycle, truncate(strings.TrimSpace(string(out)), 200))

		db = reproOpen(t, path)
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&n)
		t.Logf("  cycle %d : %d lignes apres rejeu WAL", cycle, n)
		bad := reproCheckSample(t, db, fmt.Sprintf("S11 cycle %d apres rejeu WAL", cycle), 400)
		reproExec(t, db, `CHECKPOINT`)
		bad += reproCheckSample(t, db, fmt.Sprintf("S11 cycle %d + CHECKPOINT", cycle), 400)
		db.Close()
		if bad > 0 {
			t.Errorf("S11 REPRODUIT au cycle %d : %d cles en ecart", cycle, bad)
			return
		}
	}
}

// reproOptimisticChild : UNE transaction qui appende > 1 row group (ecriture
// optimiste directe sur disque), puis mort brutale sans CHECKPOINT.
func reproOptimisticChild() {
	path := os.Getenv("PSA_REPRO_OPTIMISTIC")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		os.Exit(3)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		fmt.Println("ping:", err)
		os.Exit(4)
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		os.Exit(5)
	}
	// 200 000 lignes en UNE transaction : depasse le row group (122 880).
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO personal_score_awards
			(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
		SELECT printf('%08x-0000-4000-8000-%012d', (m * 2654435761)%4294967296, m),
		       '2533274', printf('award_%d', a),
		       ['objective','combat','support','vehicle','misc'][(a % 5) + 1],
		       1, 100 + a, 900000 + m
		FROM range(0, 1100) t1(m), range(0, 182) t2(a)`); err != nil {
		fmt.Println("insert:", err)
		os.Exit(6)
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("commit:", err)
		os.Exit(7)
	}
	fmt.Println("ok, mort brutale sans CHECKPOINT")
	os.Exit(0)
}

// ── S12 : ROLLBACK d'un append partiel PENDANT un CHECKPOINT concurrent.
// Mecanisme de duckdb/duckdb PR #24755 (« Fix checkpoint delta revert », TOUJOURS
// OUVERTE) : « rollback deleted from the wrong index. The entries remained in the
// delta and were later merged into the main index as stale entries. »
func TestPSAReproS12RollbackDuringCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	reproCreateFresh(t, db)
	reproBulkFill(t, db, 1100, 4)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, reproMatchID(7))
	reproReport(t, db, "S12 socle")

	ctx := context.Background()
	stop := make(chan struct{})
	var wg gosync.WaitGroup

	// Ecrivains : append partiel puis ROLLBACK, en boucle.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					continue
				}
				_, _ = tx.ExecContext(ctx, fmt.Sprintf(`
					INSERT INTO personal_score_awards
						(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
					SELECT printf('%%08x-0000-4000-8000-%%012d', (m * 2654435761)%%4294967296, m),
					       '2533274', printf('rb_%d_%d_%%d', a),
					       'combat', 1, 1, 800000 + m
					FROM range(0, 1100) t1(m), range(0, 20) t2(a)`, w, i))
				_ = tx.Rollback()
			}
		}(w)
	}
	// Checkpointeurs concurrents.
	for c := 0; c < 2; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = db.ExecContext(ctx, `CHECKPOINT`)
			}
		}()
	}
	// Ecrivains COMMITTES en parallele (les lignes qui doivent rester indexees).
	wg.Add(1)
	go func() {
		defer wg.Done()
		ids := reproIDs(1100)
		for r := 0; ; r++ {
			select {
			case <-stop:
				return
			default:
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				continue
			}
			for _, mid := range ids[:50] {
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO personal_score_awards
						(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
					VALUES (?, ?, ?, 'combat', 1, 1, ?)`, mid, "2533274", fmt.Sprintf("ok_%d", r), 700000+r)
			}
			_ = tx.Commit()
		}
	}()

	time.Sleep(12 * time.Second)
	close(stop)
	wg.Wait()

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&n)
	t.Logf("  lignes finales = %d", n)
	bad := reproCheckSample(t, db, "S12 apres rollback/checkpoint concurrents", 400)
	reproExec(t, db, `CHECKPOINT`)
	bad += reproCheckSample(t, db, "S12 + CHECKPOINT final", 400)
	db.Close()
	db2 := reproOpen(t, path)
	defer db2.Close()
	bad += reproCheckSample(t, db2, "S12 apres reouverture", 400)
	if bad > 0 {
		t.Errorf("S12 REPRODUIT : %d cles en ecart", bad)
	}
}
