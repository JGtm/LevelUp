//go:build art_repro

// Package sync — csr_art_repro_test.go : Phase 1 du plan d'éradication ART
// (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md) — volet CSR.
//
// **Cible** : les deux chemins CSR qui utilisent `INSERT ... ON CONFLICT
// DO UPDATE` (pattern A du test art_upsert_patterns_test.go, identifié
// comme à risque ART) :
//
//   - UpsertCSRRow (csr_writes.go:166) → match_skill_rank player DB,
//     ON CONFLICT (match_id) DO UPDATE.
//   - UpsertSharedCSRs (csr_shared_writes.go:99) → match_csrs shared DB,
//     ON CONFLICT (match_id, xuid) DO UPDATE.
//
// Les SQL exacts sont répliqués ici pour tester les patterns en isolation,
// sans dépendre du setup complet d'un SyncEngine.
//
// **Lancement** :
//
//	go test -tags art_repro -run TestCSR_ARTRepro -v ./internal/sync/...
//
// Build tag `art_repro` exclut le test de la CI tant que Phase 2 n'est
// pas livrée.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	csrReproWorkers     = 20
	csrReproRowsPerLoop = 50
	csrReproIterations  = 5
)

// openCSRPlayerFileDB ouvre une DuckDB persistante sur fichier avec le
// schéma `match_skill_rank` réel (cf. steps_player.go:302).
func openCSRPlayerFileDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_csr_player.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open duckdb file: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE match_skill_rank (
			match_id                      VARCHAR PRIMARY KEY,
			rating_type                   VARCHAR NOT NULL,
			rating_value                  FLOAT,
			rating_deviation              FLOAT,
			tier                          VARCHAR,
			tier_fr                       VARCHAR,
			sub_tier                      SMALLINT DEFAULT 0,
			tier_label                    VARCHAR,
			rating_delta                  FLOAT,
			playlist_group                VARCHAR,
			start_time                    TIMESTAMP,
			measurement_matches_remaining INTEGER DEFAULT 0,
			created_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX idx_msr_rating_type ON match_skill_rank(rating_type);
		CREATE INDEX idx_msr_playlist    ON match_skill_rank(playlist_group);
	`); err != nil {
		t.Fatalf("create table match_skill_rank: %v", err)
	}
	return db
}

// openCSRSharedFileDB ouvre une DuckDB persistante avec le schéma
// `match_csrs` réel (cf. steps_shared_match_csrs.go).
func openCSRSharedFileDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_csr_shared.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open duckdb file: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE match_csrs (
			match_id                      VARCHAR NOT NULL,
			xuid                          VARCHAR NOT NULL,
			rating_type                   VARCHAR NOT NULL DEFAULT 'CSR',
			rating_value                  FLOAT,
			tier                          VARCHAR,
			sub_tier                      SMALLINT DEFAULT 0,
			tier_label                    VARCHAR,
			rating_delta                  FLOAT,
			measurement_matches_remaining INTEGER DEFAULT 0,
			season_id                     VARCHAR,
			created_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at                    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (match_id, xuid)
		);
		CREATE INDEX idx_match_csrs_xuid   ON match_csrs(xuid);
		CREATE INDEX idx_match_csrs_season ON match_csrs(season_id);
		CREATE INDEX idx_match_csrs_match  ON match_csrs(match_id);
	`); err != nil {
		t.Fatalf("create table match_csrs: %v", err)
	}
	return db
}

// upsertCSRPlayerRowRaw réplique le SQL exact de csr_writes.go:171 sans
// dépendre du type MatchCSRRow ni du setup SyncEngine.
func upsertCSRPlayerRowRaw(ctx context.Context, db *sql.DB, matchID string, ratingValue float64) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, rating_deviation,
			 tier, tier_fr, sub_tier, tier_label,
			 rating_delta, playlist_group, start_time,
			 measurement_matches_remaining,
			 created_at, updated_at)
		VALUES (?, 'CSR', ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (match_id) DO UPDATE SET
			rating_type                   = 'CSR',
			rating_value                  = EXCLUDED.rating_value,
			rating_deviation              = NULL,
			tier                          = EXCLUDED.tier,
			tier_fr                       = EXCLUDED.tier_fr,
			sub_tier                      = EXCLUDED.sub_tier,
			tier_label                    = EXCLUDED.tier_label,
			rating_delta                  = EXCLUDED.rating_delta,
			playlist_group                = EXCLUDED.playlist_group,
			start_time                    = EXCLUDED.start_time,
			measurement_matches_remaining = EXCLUDED.measurement_matches_remaining,
			updated_at                    = EXCLUDED.updated_at`,
		matchID, ratingValue,
		"Onyx", "Onyx", 0, "Onyx 1450",
		0.0, "ranked_arena", now,
		0,
		now, now,
	)
	return err
}

// upsertCSRSharedRowRaw réplique le SQL de csr_shared_writes.go:105.
func upsertCSRSharedRowRaw(ctx context.Context, db *sql.DB, matchID, xuid string, ratingValue float64) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_csrs (
			match_id, xuid, rating_type, rating_value,
			tier, sub_tier, tier_label, rating_delta,
			measurement_matches_remaining, season_id,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (match_id, xuid) DO UPDATE SET
			rating_type                   = EXCLUDED.rating_type,
			rating_value                  = EXCLUDED.rating_value,
			tier                          = EXCLUDED.tier,
			sub_tier                      = EXCLUDED.sub_tier,
			tier_label                    = EXCLUDED.tier_label,
			rating_delta                  = EXCLUDED.rating_delta,
			measurement_matches_remaining = EXCLUDED.measurement_matches_remaining,
			season_id                     = COALESCE(EXCLUDED.season_id, match_csrs.season_id),
			updated_at                    = EXCLUDED.updated_at`,
		matchID, xuid, "CSR", ratingValue,
		"Onyx", 0, "Onyx 1450",
		0.0,
		0, "season_1",
		now, now,
	)
	return err
}

// TestCSR_ARTRepro_PlayerDB — Best-effort repro ART sur le chemin
// `UpsertCSRRow` (player DB, ON CONFLICT (match_id) DO UPDATE).
//
// **Hypothèse Phase 2** : après bascule de `match_skill_rank` en schema
// append-only, ce test ne peut plus déclencher d'erreur de DELETE (puisque
// le pattern devient INSERT pur, jamais de DELETE implicite via ON CONFLICT).
//
// Loggue les erreurs observées sans assertion bloquante (le test sert
// d'observation empirique, comme TestLUSRUpsert_ARTReproOnFile).
func TestCSR_ARTRepro_PlayerDB(t *testing.T) {
	db := openCSRPlayerFileDB(t)
	ctx := context.Background()

	// Pré-seed N rows partagés entre workers
	for i := 0; i < csrReproRowsPerLoop; i++ {
		mid := fmt.Sprintf("csr_p_%03d", i)
		if err := upsertCSRPlayerRowRaw(ctx, db, mid, 1400.0); err != nil {
			t.Fatalf("pre-seed CSR player row %d: %v", i, err)
		}
	}

	var (
		wg          sync.WaitGroup
		errsMu      sync.Mutex
		firstArtErr error
		errCount    int
	)

	for w := 0; w < csrReproWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for iter := 0; iter < csrReproIterations; iter++ {
				for i := 0; i < csrReproRowsPerLoop; i++ {
					mid := fmt.Sprintf("csr_p_%03d", i)
					rating := float64(workerID*100 + iter*10 + i)
					if err := upsertCSRPlayerRowRaw(ctx, db, mid, rating); err != nil {
						errsMu.Lock()
						errCount++
						if firstArtErr == nil {
							firstArtErr = err
						}
						errsMu.Unlock()
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()

	if errCount > 0 {
		t.Logf("CSR player ART repro : %d errors sur %d workers x %d iters",
			errCount, csrReproWorkers, csrReproIterations)
		t.Logf("Première erreur : %v", firstArtErr)
	} else {
		t.Logf("CSR player ART repro : 0 erreur sur %d workers x %d iters x %d rows. Bug ART NON reproduit cette fois.",
			csrReproWorkers, csrReproIterations, csrReproRowsPerLoop)
		t.Logf("Pattern fragile par construction (ON CONFLICT DO UPDATE = DELETE+INSERT implicite). Migration append-only recommandée Phase 2.")
	}
}

// TestCSR_ARTRepro_SharedDB — Best-effort repro ART sur `UpsertSharedCSRs`
// (shared DB, ON CONFLICT (match_id, xuid) DO UPDATE).
//
// Pattern identique au précédent, sur table à PK composite — le crash
// observé historiquement sur shared.match_participants (PK composite
// VARCHAR) suggère que match_csrs présente le même risque.
func TestCSR_ARTRepro_SharedDB(t *testing.T) {
	db := openCSRSharedFileDB(t)
	ctx := context.Background()

	// Pré-seed N (match_id, xuid) pairs
	for i := 0; i < csrReproRowsPerLoop; i++ {
		mid := fmt.Sprintf("csr_s_%03d", i)
		xuid := fmt.Sprintf("xuid_%03d", i%5) // 5 xuids partagés → collisions inter-workers
		if err := upsertCSRSharedRowRaw(ctx, db, mid, xuid, 1400.0); err != nil {
			t.Fatalf("pre-seed CSR shared row %d: %v", i, err)
		}
	}

	var (
		wg          sync.WaitGroup
		errsMu      sync.Mutex
		firstArtErr error
		errCount    int
	)

	for w := 0; w < csrReproWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for iter := 0; iter < csrReproIterations; iter++ {
				for i := 0; i < csrReproRowsPerLoop; i++ {
					mid := fmt.Sprintf("csr_s_%03d", i)
					xuid := fmt.Sprintf("xuid_%03d", i%5)
					rating := float64(workerID*100 + iter*10 + i)
					if err := upsertCSRSharedRowRaw(ctx, db, mid, xuid, rating); err != nil {
						errsMu.Lock()
						errCount++
						if firstArtErr == nil {
							firstArtErr = err
						}
						errsMu.Unlock()
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()

	if errCount > 0 {
		t.Logf("CSR shared ART repro : %d errors sur %d workers x %d iters",
			errCount, csrReproWorkers, csrReproIterations)
		t.Logf("Première erreur : %v", firstArtErr)
	} else {
		t.Logf("CSR shared ART repro : 0 erreur. Bug ART NON reproduit cette fois (table PK composite).")
		t.Logf("Le bug a déjà été observé en prod sur cette classe de pattern (shared.match_participants). Migration append-only recommandée Phase 2.")
	}
}
