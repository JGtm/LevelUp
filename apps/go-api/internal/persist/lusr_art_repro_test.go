//go:build art_repro

// Package persist — lusr_art_repro_test.go : Phase 1 du plan d'éradication
// ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Tests rouges** : ces tests prouvent que le chemin actuel de
// PostSyncLUSRPersister.Upsert (DELETE + INSERT en transaction) est à risque
// ART et doit basculer vers un schema append-only.
//
// - TestLUSRUpsert_AppendOnlySemantics : assert que Upsert n'efface jamais
//   les rows existantes. ROUGE aujourd'hui (Upsert DELETE), VERT après
//   Phase 2 du plan (append-only + vue latest).
//
// - TestLUSRUpsert_ARTReproOnFile : tente de reproduire le bug ART
//   "Failed to delete all rows from index" sur fichier persistant + N
//   workers concurrents. Best-effort, non-déterministe (le bug ne se
//   reproduit pas systématiquement). Logue le résultat sans assertion
//   bloquante — sert d'observation empirique.
//
// **Lancement** :
//
//	go test -tags art_repro ./internal/persist/... -run LUSRUpsert_ -v
//
// **Build tag** `art_repro` exclut ces tests de la suite par défaut (la CI
// reste verte) tant que Phase 2 n'est pas mergée. Une fois Phase 2 livrée,
// retirer le build tag pour intégrer le test #1 à la suite `integration`.

package persist

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	artReproWorkers     = 20
	artReproRowsPerLoop = 50
	artReproIterations  = 5
)

// openLUSRFileDB ouvre une DuckDB persistante sur fichier (pas :memory:)
// avec le schéma match_skill_rank actuel. Le bug ART nécessite un fichier
// pour se déclencher de façon réaliste.
func openLUSRFileDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_lusr.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open duckdb file: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE match_skill_rank (
			match_id          VARCHAR PRIMARY KEY,
			rating_type       VARCHAR NOT NULL,
			rating_value      FLOAT,
			rating_deviation  FLOAT,
			tier              VARCHAR,
			tier_fr           VARCHAR,
			sub_tier          SMALLINT DEFAULT 0,
			tier_label        VARCHAR,
			rating_delta      FLOAT,
			playlist_group    VARCHAR,
			start_time        TIMESTAMP,
			created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db, path
}

// TestLUSRUpsert_AppendOnlySemantics — Test ROUGE jusqu'à Phase 2.
//
// **Sémantique cible (append-only)** : Upsert ne fait QUE des INSERT, jamais
// de DELETE. Donc après pré-seed de 3 rows + Upsert de 2 rows avec
// match_id en collision, le total physique attendu est 5 (les 3 originales
// préservées + 2 nouvelles), pas 3 (DELETE + INSERT actuel).
//
// **Pourquoi ce test est important** : c'est l'invariant qui rend le bug
// ART impossible par construction. Tant que ce test échoue (count=3), le
// chemin Upsert peut déclencher l'ART crash observé en prod 20:41:04.
//
// ROUGE aujourd'hui (count=3), VERT après bascule schema append-only.
func TestLUSRUpsert_AppendOnlySemantics(t *testing.T) {
	db, _ := openLUSRFileDB(t)

	// Pré-seed 3 LUSR rows distinctes
	for _, mid := range []string{"m1", "m2", "m3"} {
		if _, err := db.Exec(`
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', 20.0, 'arena_slayer')
		`, mid); err != nil {
			t.Fatal(err)
		}
	}

	p := NewPostSyncLUSRPersister(db)
	// Upsert avec collision sur m1 et m2 (m3 non touché)
	rows := []LUSRRatingInsert{
		{MatchID: "m1", RatingValue: 25.0, PlaylistGroup: "arena_slayer"},
		{MatchID: "m2", RatingValue: 26.0, PlaylistGroup: "arena_slayer"},
	}
	if err := p.Upsert(context.Background(), rows); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Total physique attendu APRÈS append-only : 5 (3 originaux préservés + 2 nouveaux)
	// Actuel (DELETE + INSERT) : 3 (m1 et m2 remplacés in-place, m3 intact)
	var totalPhysical int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'`).Scan(&totalPhysical); err != nil {
		t.Fatal(err)
	}
	if totalPhysical != 5 {
		t.Errorf("total LUSR rows physiques = %d, want 5 (append-only : 3 originales préservées + 2 nouvelles)", totalPhysical)
		t.Logf("Si vous voyez count=3, c'est que Upsert fait toujours DELETE+INSERT. C'est exactement le chemin qui a crashé en prod 20:41:04.")
		t.Logf("Phase 2 du plan (.ai/PLAN_LUSR_ART_HOME_CRASH.md) bascule ce schema en append-only avec une vue match_skill_rank_latest.")
	}

	// Vérifier qu'une vue/query "latest" renverrait bien 3 rows fonctionnelles
	// (1 par match_id, la plus récente). Si la vue n'existe pas encore (avant
	// Phase 2), on simule via ROW_NUMBER() OVER pour valider la requête.
	var latestCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT match_id FROM match_skill_rank
			WHERE rating_type='LUSR'
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id ORDER BY updated_at DESC) = 1
		)
	`).Scan(&latestCount)
	if err != nil {
		t.Logf("query latest échouée (attendu sans written_at column) : %v", err)
	} else if latestCount != 3 {
		t.Errorf("latest LUSR rows = %d, want 3 (1 par match_id)", latestCount)
	}
}

// TestLUSRUpsert_ARTReproOnFile — Best-effort de reproduction du bug ART.
//
// Stratégie : ouverte fichier DuckDB + N workers concurrents qui exécutent
// chacun M itérations de Upsert avec match_ids en collision. Le pattern
// DELETE+INSERT en transaction sur des PKs en contention est le cas
// précis observé en prod (20:41:04 sur Chocoboflor).
//
// Le bug ART est **non-déterministe** — il ne se reproduit pas
// systématiquement, dépend du timing interne DuckDB + état de l'index ART.
// Ce test ne fail pas si le bug ne se reproduit pas : il logue l'observation
// pour documentation. Si le crash est reproduit, l'erreur est loggée et le
// test passe quand même (l'observation est l'objectif, pas le crash CI).
//
// **Hypothèse Phase 2** : après bascule append-only, ce test ne peut PLUS
// reproduire le bug car il n'y a plus de DELETE. À ce moment, ce test
// devient une régression — l'INSERT ne déclenche pas l'ART quelle que
// soit la concurrence.
func TestLUSRUpsert_ARTReproOnFile(t *testing.T) {
	db, dbPath := openLUSRFileDB(t)
	t.Logf("repro DB file: %s", dbPath)

	// Pré-seed N rows en collision potentielle (1 PK partagé entre workers)
	for i := 0; i < artReproRowsPerLoop; i++ {
		mid := fmt.Sprintf("seed_%03d", i)
		if _, err := db.Exec(`
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
			VALUES (?, 'LUSR', 20.0, 'arena_slayer')
		`, mid); err != nil {
			t.Fatal(err)
		}
	}

	p := NewPostSyncLUSRPersister(db)
	var (
		wg          sync.WaitGroup
		errsMu      sync.Mutex
		firstArtErr error
		errCount    int
	)

	ctx := context.Background()
	for w := 0; w < artReproWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for iter := 0; iter < artReproIterations; iter++ {
				// Batch de rows avec match_ids partagés entre workers → collision PK
				rows := make([]LUSRRatingInsert, 0, artReproRowsPerLoop)
				for i := 0; i < artReproRowsPerLoop; i++ {
					rows = append(rows, LUSRRatingInsert{
						MatchID:       fmt.Sprintf("seed_%03d", i),
						RatingValue:   float64(workerID*100 + iter),
						PlaylistGroup: "arena_slayer",
					})
				}
				if err := p.Upsert(ctx, rows); err != nil {
					errsMu.Lock()
					errCount++
					if firstArtErr == nil {
						firstArtErr = err
					}
					errsMu.Unlock()
					return // worker stop sur première erreur
				}
			}
		}(w)
	}
	wg.Wait()

	if errCount > 0 {
		t.Logf("ART repro : %d errors sur %d workers (%d iters chacun)",
			errCount, artReproWorkers, artReproIterations)
		t.Logf("Première erreur : %v", firstArtErr)
		// Si le bug ART est reproduit, on logue mais on ne fail pas
		// (le test sert d'observation, pas d'assertion CI).
		// Après Phase 2 (append-only), ce log devrait passer à "0 errors".
	} else {
		t.Logf("ART repro : 0 erreur sur %d workers x %d iters x %d rows. Bug ART NON reproduit cette fois.",
			artReproWorkers, artReproIterations, artReproRowsPerLoop)
		t.Logf("Non-reproduction ne prouve pas la sûreté. Le bug est observé en prod (sync.log 20:41:04 Chocoboflor).")
	}
}
