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
	gosync "sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// reproBulkFill peuple massivement (INSERT ... SELECT) pour depasser la taille
// d'un row group DuckDB (122880 lignes) : plusieurs row groups => le vacuum de
// CHECKPOINT peut compacter et deplacer les row_id references par l'ART.
func reproBulkFill(t *testing.T, db *sql.DB, matches, awardsPerMatch int) {
	t.Helper()
	reproExec(t, db, fmt.Sprintf(`
		INSERT INTO personal_score_awards
			(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
		SELECT
			printf('%%08x-0000-4000-8000-%%012d', (m * 2654435761)%%4294967296, m),
			'2533274',
			printf('award_%%d', a),
			['objective','combat','support','vehicle','misc'][(a %% 5) + 1],
			1, 100 + a,
			m
		FROM range(0, %d) t1(m), range(0, %d) t2(a)`, matches, awardsPerMatch))
}

// ── S7 : gros volume (> 1 row group) + DELETE massif + CHECKPOINT (vacuum) ───

func TestPSAReproS7BulkVacuum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()
	reproCreateFresh(t, db)

	reproBulkFill(t, db, 30000, 12) // 360 000 lignes ~ 3 row groups
	reproExec(t, db, `CHECKPOINT`)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Logf("  volume initial = %d lignes", n)
	reproAssertIndexScan(t, db, reproMatchID(7))
	if d := reproCheckSample(t, db, "S7 apres bulk", 400); d > 0 {
		t.Errorf("S7 REPRODUIT (bulk) : %d cles en ecart", d)
	}

	// DELETE massif (85 %) : cible du vacuum de row group au CHECKPOINT.
	reproExec(t, db, `DELETE FROM personal_score_awards WHERE generation_id >= 4500`)
	reproExec(t, db, `CHECKPOINT`)
	if err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Logf("  apres DELETE massif + CHECKPOINT = %d lignes", n)
	if d := reproCheckSample(t, db, "S7 apres DELETE massif + CHECKPOINT", 400); d > 0 {
		t.Errorf("S7 REPRODUIT (vacuum) : %d cles en ecart", d)
	}
	db.Close()

	db = reproOpen(t, path)
	if d := reproCheckSample(t, db, "S7 apres reouverture", 400); d > 0 {
		t.Errorf("S7 REPRODUIT (reouverture) : %d cles en ecart", d)
	}
}

// ── S8 : vacuum_rebuild_indexes = 1 (experimental) ──────────────────────────

func TestPSAReproS8VacuumRebuildIndexes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db, oerr := sql.Open("duckdb", path+"?vacuum_rebuild_indexes=true")
	if oerr == nil {
		db.SetMaxOpenConns(1)
		oerr = db.Ping()
	}
	if oerr != nil {
		t.Skipf("S8 NON TESTABLE : vacuum_rebuild_indexes n_est pas reglable (%v). Reste OFF par defaut, donc hors cause pour les DB reelles.", truncate(oerr.Error(), 200))
	}
	defer db.Close()
	reproCreateFresh(t, db)
	reproBulkFill(t, db, 30000, 12)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, reproMatchID(7))
	if d := reproCheckSample(t, db, "S8 apres bulk", 400); d > 0 {
		t.Errorf("S8 REPRODUIT (bulk) : %d cles en ecart", d)
	}
	reproExec(t, db, `DELETE FROM personal_score_awards WHERE generation_id >= 4500`)
	reproExec(t, db, `CHECKPOINT`)
	if d := reproCheckSample(t, db, "S8 apres DELETE massif + CHECKPOINT", 400); d > 0 {
		t.Errorf("S8 REPRODUIT (vacuum rebuild) : %d cles en ecart", d)
	}
	db.Close()
	db = reproOpen(t, path+"?vacuum_rebuild_indexes=true")
	if d := reproCheckSample(t, db, "S8 apres reouverture", 400); d > 0 {
		t.Errorf("S8 REPRODUIT (reouverture) : %d cles en ecart", d)
	}
}

// reproCheckSample : meme controle que reproCheck mais borne a `limit` cles de
// l'axe match_id (le corpus bulk a des dizaines de milliers de cles distinctes).
func reproCheckSample(t *testing.T, db *sql.DB, phase string, limit int) int {
	t.Helper()
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT match_id || '', COUNT(*) FROM personal_score_awards GROUP BY match_id || '' LIMIT %d`, limit))
	if err != nil {
		t.Fatalf("[%s] scan: %v", phase, err)
	}
	type kc struct {
		k string
		n int
	}
	var refs []kc
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		refs = append(refs, kc{k, n})
	}
	rows.Close()
	stmt, err := db.PrepareContext(ctx, `SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	bad := 0
	for _, r := range refs {
		var got int
		if err := stmt.QueryRowContext(ctx, r.k).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != r.n {
			if bad < 5 {
				t.Logf("     ECART cle=%s scan=%d indexe=%d", r.k, r.n, got)
			}
			bad++
		}
	}
	if bad == 0 {
		t.Logf("  [%s] %d cles echantillonnees : OK (0 ecart)", phase, len(refs))
	} else {
		t.Logf("  [%s] %d cles echantillonnees : *** %d EN ECART ***", phase, len(refs), bad)
	}
	return bad
}

// ── S9 : ecritures CONCURRENTES (pool multi-connexions) sur la meme DB ───────
//
// Le chemin player DB de prod est serialise (MaxOpenConns=1), mais on teste
// l'aggravant « plusieurs connexions ecrivent en parallele » pour savoir s'il
// suffit a desynchroniser l'ART.
func TestPSAReproS9Concurrent(t *testing.T) {
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
	ids := reproIDs(1100)

	ctx := context.Background()
	var wg gosync.WaitGroup
	errs := make(chan error, 16)
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := w; i < len(ids); i += 8 {
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					errs <- err
					return
				}
				var gen int64
				if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
					_ = tx.Rollback()
					errs <- err
					return
				}
				for a := 0; a < 4; a++ {
					if _, err := tx.ExecContext(ctx, `
						INSERT INTO personal_score_awards
							(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
						VALUES (?, ?, ?, ?, ?, ?, ?)`,
						ids[i], "2533274", fmt.Sprintf("award_%d", a),
						reproCategories[a%len(reproCategories)], 1, 100+a, gen); err != nil {
						_ = tx.Rollback()
						errs <- err
						return
					}
				}
				if err := tx.Commit(); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	nerr := 0
	for e := range errs {
		nerr++
		if nerr <= 3 {
			t.Logf("  erreur ecriture concurrente: %v", truncate(e.Error(), 180))
		}
	}
	t.Logf("  erreurs d'ecriture concurrente = %d", nerr)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S9 apres ecritures concurrentes")
	reproExec(t, db, `CHECKPOINT`)
	if n := reproReport(t, db, "S9 apres CHECKPOINT"); n > 0 {
		t.Errorf("S9 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S10 : process TUE PENDANT les ecritures et les CHECKPOINT ───────────────
//
// Scenario le plus proche du poste reel : le serveur Go est tue (taskkill /F)
// alors qu'il ecrit ET checkpointe. L'enfant boucle « vagues + CHECKPOINT »
// jusqu'a ce que le parent le tue a un instant arbitraire.
func TestPSAReproS10KillDuringCheckpoint(t *testing.T) {
	if os.Getenv("PSA_REPRO_KILLLOOP") != "" {
		reproKillLoopChild()
		return
	}
	dir, err := os.MkdirTemp("", "psarepro10")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "stats.duckdb")

	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S10 socle")
	db.Close()

	delays := []time.Duration{150 * time.Millisecond, 400 * time.Millisecond, 900 * time.Millisecond,
		1700 * time.Millisecond, 2600 * time.Millisecond, 3500 * time.Millisecond}
	for cycle, d := range delays {
		cmd := exec.Command(os.Args[0], "-test.run", "TestPSAReproS10KillDuringCheckpoint")
		cmd.Env = append(os.Environ(), "PSA_REPRO_KILLLOOP="+path)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start enfant: %v", err)
		}
		time.Sleep(d)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		db = reproOpen(t, path)
		n := reproReport(t, db, fmt.Sprintf("S10 cycle %d (tue apres %v)", cycle, d))
		reproExec(t, db, `CHECKPOINT`)
		n += reproReport(t, db, fmt.Sprintf("S10 cycle %d + CHECKPOINT", cycle))
		db.Close()
		if n > 0 {
			t.Errorf("S10 REPRODUIT au cycle %d (kill apres %v) : %d cles en ecart", cycle, d, n)
			return
		}
	}
}

// reproKillLoopChild : ecrit et checkpointe en boucle jusqu'a etre tue.
func reproKillLoopChild() {
	path := os.Getenv("PSA_REPRO_KILLLOOP")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		os.Exit(3)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		os.Exit(4)
	}
	ctx := context.Background()
	ids := reproIDs(1100)
	for round := 0; ; round++ {
		for _, mid := range ids {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				os.Exit(5)
			}
			var gen int64
			if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
				os.Exit(6)
			}
			for a := 0; a < 4; a++ {
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO personal_score_awards
						(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
					VALUES (?, ?, ?, ?, ?, ?, ?)`,
					mid, "2533274", fmt.Sprintf("award_%d", a),
					reproCategories[a%len(reproCategories)], 1, 100+a, gen); err != nil {
					os.Exit(7)
				}
			}
			if err := tx.Commit(); err != nil {
				os.Exit(8)
			}
		}
		if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
			os.Exit(9)
		}
	}
}
