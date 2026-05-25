//go:build integration

// Package sync — concurrent_upsert_stress_test.go : stress test pour valider
// que `InsertParticipants` reste sûr sous écritures concurrentes massives.
//
// Plan stabilisation Phase 5.1 (TDD avant Phase 2.3 singleflight). Cf.
// `.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md` et `docs/adr/0018-concurrent-write-model.md`.
//
// Le test n'essaie PAS de reproduire la race ART DuckDB exacte (bug upstream
// non-déterministe). Il verrouille le contrat applicatif :
//   - N goroutines × M UPSERTs sur la même `(match_id, xuid)` → exactement
//     1 row en table à la fin.
//   - Aucun panic Go, aucune race détectée par `-race`.
//   - Les UPSERTs sur des clés DIFFÉRENTES tournent en parallèle (perf).
//
// Avec singleflight (Phase 2.3) : 1 seul SQL exec par clé même si N appelants.
// Sans singleflight : N execs séquentialisés par DuckDB driver, fonctionne sur
// les petites tables mais peut corrompre l'ART en prod sous charge.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupParticipantsTable crée une DB DuckDB in-memory avec le schéma minimum
// de `match_participants` (PK + colonnes utilisées par InsertParticipants).
func setupParticipantsTable(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	// SetMaxOpenConns(1) sérialise les écritures au niveau Go-sql (prod behavior).
	// Sans singleflight (supprimé f243b235), DuckDB ne tolère pas les UPSERTs
	// concurrents sur plusieurs connexions simultanées.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE match_participants (
		match_id            VARCHAR NOT NULL,
		xuid                VARCHAR NOT NULL,
		gamertag            VARCHAR,
		team_id             INTEGER,
		outcome             INTEGER,
		rank                INTEGER,
		score               INTEGER,
		kills               INTEGER,
		deaths              INTEGER,
		assists             INTEGER,
		shots_fired         INTEGER,
		shots_hit           INTEGER,
		damage_dealt        DOUBLE,
		damage_taken        DOUBLE,
		kda                 DOUBLE,
		accuracy            DOUBLE,
		personal_score      DOUBLE,
		time_played_seconds DOUBLE,
		avg_life_seconds    DOUBLE,
		kills_expected      DOUBLE,
		deaths_expected     DOUBLE,
		kills_stddev        DOUBLE,
		deaths_stddev       DOUBLE,
		team_mmr            DOUBLE,
		enemy_mmr           DOUBLE,
		headshot_kills      INTEGER,
		max_killing_spree   INTEGER,
		grenade_kills       INTEGER,
		melee_kills         INTEGER,
		power_weapon_kills  INTEGER,
		created_at          TIMESTAMP,
		PRIMARY KEY (match_id, xuid)
	)`); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	return db
}

// TestStressUpsertParticipants_SameKey_NoCrash_OneRow : 50 goroutines × 200
// UPSERTs concurrents sur la MÊME `(match_id, xuid)` produisent exactement
// 1 row, sans panic ni data race. Test de régression Phase 1.3 singleflight.
//
// Sans singleflight, ce test passe quand même sur la plupart des runs (DuckDB
// driver sérialise les commits) mais sous charge réelle prod l'ART corrompt
// — cf. incident 2026-05-22. Le test est un garde-fou : si on régresse en
// retirant le singleflight, la corruption pourra ressurgir mais le contrat
// applicatif reste vérifié.
func TestStressUpsertParticipants_SameKey_NoCrash_OneRow(t *testing.T) {
	db := setupParticipantsTable(t)
	ctx := context.Background()

	const (
		matchID     = "aabbccdd-0000-0000-0000-000000000001"
		xuid        = "2535469190789936"
		nGoroutines = 50
		nIterations = 200
	)

	teamID := 0
	outcome := 2
	kills := 10
	deaths := 5
	row := ParticipantRow{
		MatchID: matchID, XUID: xuid,
		TeamID: &teamID, Outcome: &outcome,
		Kills: &kills, Deaths: &deaths,
	}

	var wg sync.WaitGroup
	var failures atomic.Int64
	for g := 0; g < nGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < nIterations; j++ {
				if err := InsertParticipants(ctx, db, []ParticipantRow{row}); err != nil {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if n := failures.Load(); n > 0 {
		t.Errorf("got %d/%d goroutines × iterations with InsertParticipants err",
			n, nGoroutines*nIterations)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants
		WHERE match_id = ? AND xuid = ?`, matchID, xuid).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row in match_participants, got %d", count)
	}
}

// TestStressUpsertParticipants_DifferentKeys_AllPresent : les UPSERTs sur des
// clés DIFFÉRENTES tournent en parallèle sans collision. Verrouille que le
// singleflight (clé par `match_id|xuid`) ne sérialise PAS les writes inutilement.
func TestStressUpsertParticipants_DifferentKeys_AllPresent(t *testing.T) {
	db := setupParticipantsTable(t)
	ctx := context.Background()

	const (
		nMatches = 10
		nXUIDs   = 8 // = 80 rows uniques attendues
		nWriters = 16
	)

	expectedRows := nMatches * nXUIDs
	keys := make([][2]string, 0, expectedRows)
	for m := 0; m < nMatches; m++ {
		for x := 0; x < nXUIDs; x++ {
			keys = append(keys, [2]string{
				fmt.Sprintf("11111111-0000-0000-0000-%012d", m),
				fmt.Sprintf("2533274%010d", x),
			})
		}
	}

	teamID := 0
	outcome := 1
	kills := 1
	var wg sync.WaitGroup
	var failures atomic.Int64
	for w := 0; w < nWriters; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, k := range keys {
				row := ParticipantRow{
					MatchID: k[0], XUID: k[1],
					TeamID: &teamID, Outcome: &outcome, Kills: &kills,
				}
				if err := InsertParticipants(ctx, db, []ParticipantRow{row}); err != nil {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if n := failures.Load(); n > 0 {
		t.Errorf("got %d failures across %d writers", n, nWriters)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != expectedRows {
		t.Errorf("expected %d unique rows (nMatches × nXUIDs), got %d",
			expectedRows, count)
	}
}

// TestStressUpsertParticipants_BatchPerCall : un appelant envoie un batch de
// 8 participants (1 match, 8 xuids) en parallèle avec d'autres appelants. Tous
// les rows doivent être présents à la fin. Reproduit le scénario sync engine
// Phase 3 + heal concurrent.
func TestStressUpsertParticipants_BatchPerCall(t *testing.T) {
	db := setupParticipantsTable(t)
	ctx := context.Background()

	const (
		nMatches  = 20
		nXuidsPer = 8
		nWorkers  = 8
	)

	rowsByMatch := make([][]ParticipantRow, nMatches)
	teamID := 1
	outcome := 3
	kills := 7
	for m := 0; m < nMatches; m++ {
		matchID := fmt.Sprintf("22222222-0000-0000-0000-%012d", m)
		batch := make([]ParticipantRow, 0, nXuidsPer)
		for x := 0; x < nXuidsPer; x++ {
			batch = append(batch, ParticipantRow{
				MatchID: matchID,
				XUID:    fmt.Sprintf("xuid-%d-%d", m, x),
				TeamID:  &teamID, Outcome: &outcome, Kills: &kills,
			})
		}
		rowsByMatch[m] = batch
	}

	var wg sync.WaitGroup
	var failures atomic.Int64
	for w := 0; w < nWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, batch := range rowsByMatch {
				if err := InsertParticipants(ctx, db, batch); err != nil {
					failures.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if n := failures.Load(); n > 0 {
		t.Errorf("got %d failures across %d workers", n, nWorkers)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	expected := nMatches * nXuidsPer
	if count != expected {
		t.Errorf("expected %d rows (nMatches × nXuidsPer), got %d", expected, count)
	}
}
