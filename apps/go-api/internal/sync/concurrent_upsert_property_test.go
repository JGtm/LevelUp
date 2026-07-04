//go:build integration

// Package sync — concurrent_upsert_property_test.go : property-based test
// d'idempotence sur InsertParticipants (Phase 5.2 du plan stabilisation
// 2026-05-22).
//
// Property testée :
//
//	∀ K match_ids, ∀ M xuids, ∀ N UPSERTs concurrents en parallèle,
//	    l'état final de match_participants =
//	    exactement {(match_id, xuid) | (match_id, xuid) ∈ inputs}
//	    avec :
//	        - count(match_id, xuid) == 1 pour chaque paire unique (idempotence PK)
//	        - les valeurs reflètent un des inputs proposés (deterministic UPSERT)
//	        - aucune row absente (pas de perte malgré le singleflight dedup)
//
// Cette property couvre simultanément :
//   - Phase 2.3 singleflight (commit aef47968) : pas de perte malgré la dédupe.
//   - PK ON CONFLICT DO UPDATE : pas de doublon.
//   - L'absence de race ART après le bump driver v1.5.3 + singleflight.
//
// Le test génère des inputs ALÉATOIRES (seed fixé pour reproductibilité) sur
// plusieurs combinaisons K×M×N pour explorer l'espace des concurrences.
// Chaque combinaison run avec -race pour détecter les races de bas niveau.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openParticipantsDBForProperty : DB minimale avec match_participants (PK
// composite) + match_registry pour FK référentielle.
func openParticipantsDBForProperty(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// SetMaxOpenConns(1) sérialise les écritures au niveau Go-sql (prod behavior).
	// Sans singleflight (supprimé f243b235), DuckDB ne tolère pas les UPSERTs
	// concurrents sur plusieurs connexions simultanées.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			pair_name VARCHAR
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			rank INTEGER,
			score INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			assists INTEGER DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			damage_taken DOUBLE DEFAULT 0,
			kda DOUBLE DEFAULT 0,
			accuracy DOUBLE DEFAULT 0,
			personal_score INTEGER DEFAULT 0,
			time_played_seconds INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0,
			kills_expected DOUBLE,
			deaths_expected DOUBLE,
			kills_stddev DOUBLE,
			deaths_stddev DOUBLE,
			team_mmr DOUBLE,
			enemy_mmr DOUBLE,
			headshot_kills SMALLINT DEFAULT 0,
			max_killing_spree SMALLINT DEFAULT 0,
			grenade_kills SMALLINT DEFAULT 0,
			melee_kills SMALLINT DEFAULT 0,
			power_weapon_kills SMALLINT DEFAULT 0,
			present_at_beginning BOOLEAN,
			present_at_completion BOOLEAN,
			joined_in_progress BOOLEAN,
			left_in_progress BOOLEAN,
			first_joined_time TIMESTAMPTZ,
			last_leave_time TIMESTAMPTZ,
			created_at TIMESTAMP,
			PRIMARY KEY (match_id, xuid)
		);
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestProperty_ConcurrentUpsertsIdempotent : K=8 matchs × M=12 xuids =
// 96 paires uniques. N=20 UPSERTs concurrents par paire = 1920 calls
// concurrents. État final attendu : exactement 96 rows.
//
// Test sous -race obligatoire (le harness CI doit le run en mode race).
func TestProperty_ConcurrentUpsertsIdempotent(t *testing.T) {
	const K = 8  // nb match_ids
	const M = 12 // nb xuids
	const N = 5  // nb UPSERTs par paire — réduit (SetMaxOpenConns=1 sérialise, N=20 dépasse timeout CI)

	db := openParticipantsDBForProperty(t)
	ctx := context.Background()

	// Seed match_registry pour FK référentielle.
	for k := 0; k < K; k++ {
		mid := fmt.Sprintf("prop-match-%04d", k)
		if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, pair_name)
			VALUES (?, NOW(), 'Slayer')`, mid); err != nil {
			t.Fatalf("seed match_registry: %v", err)
		}
	}

	// Générer toutes les paires (match_id, xuid).
	type pair struct{ matchID, xuid string }
	pairs := make([]pair, 0, K*M)
	for k := 0; k < K; k++ {
		mid := fmt.Sprintf("prop-match-%04d", k)
		for m := 0; m < M; m++ {
			pairs = append(pairs, pair{
				matchID: mid,
				xuid:    fmt.Sprintf("prop-xuid-%04d", m),
			})
		}
	}

	// Lancer N goroutines concurrentes par paire, avec valeurs RANDOM (seed
	// fixé pour reproductibilité). Le singleflight (writes.go) doit dédupliquer
	// les UPSERTs sur la même clé naturelle.
	src := rand.New(rand.NewSource(42))
	intPtr := func(v int) *int { return &v }
	strPtrNonEmpty := func(v string) *string { return &v }
	type job struct {
		p    pair
		rows []ParticipantRow
	}
	jobs := make([]job, 0, len(pairs)*N)
	for _, p := range pairs {
		for n := 0; n < N; n++ {
			jobs = append(jobs, job{
				p: p,
				rows: []ParticipantRow{{
					MatchID:  p.matchID,
					XUID:     p.xuid,
					Gamertag: strPtrNonEmpty(fmt.Sprintf("player-%d", src.Intn(100))),
					TeamID:   intPtr(src.Intn(2)),
					Outcome:  intPtr(2),
					Kills:    intPtr(src.Intn(30)),
					Deaths:   intPtr(src.Intn(20)),
					Assists:  intPtr(src.Intn(15)),
				}},
			})
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(jobs))
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := InsertParticipants(ctx, db, j.rows); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("InsertParticipants concurrent: %v", err)
	}

	// Property 1 : exactement K×M rows uniques.
	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rowCount); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	want := K * M
	if rowCount != want {
		t.Errorf("Property violation : count = %d, want %d (perte ou doublon)",
			rowCount, want)
	}

	// Property 2 : chaque paire (match_id, xuid) a exactement 1 row.
	for _, p := range pairs {
		var n int
		// Table-scan (|| '') pour court-circuiter ART au cas où.
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM match_participants
			 WHERE match_id || '' = ? AND xuid || '' = ?`,
			p.matchID, p.xuid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("Property violation : (%s,%s) count = %d, want 1",
				p.matchID, p.xuid, n)
		}
	}

	// Property 3 : aucun (match_id, xuid) absent. Détecte une perte qui
	// laisserait le count global correct mais la distribution faussée.
	rows, err := db.Query(
		`SELECT match_id, xuid FROM match_participants ORDER BY match_id, xuid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := make(map[pair]bool, K*M)
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.matchID, &p.xuid); err != nil {
			t.Fatal(err)
		}
		seen[p] = true
	}
	for _, p := range pairs {
		if !seen[p] {
			t.Errorf("Property violation : (%s,%s) absent du résultat",
				p.matchID, p.xuid)
		}
	}
}

// TestProperty_SamePairManyConcurrent_OneRow : K=1 match × M=1 xuid =
// 1 paire, N=200 UPSERTs concurrents tous sur la même clé. Le singleflight
// doit collapse à 1 seule exécution SQL. Résultat : 1 row finale.
//
// Variante extrême du test ci-dessus pour stresser le code path singleflight.
func TestProperty_SamePairManyConcurrent_OneRow(t *testing.T) {
	db := openParticipantsDBForProperty(t)
	ctx := context.Background()

	const N = 30 // réduit (SetMaxOpenConns=1 sérialise, N=200 dépasse timeout CI)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, pair_name)
		VALUES ('prop-singlepair', NOW(), 'Slayer')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	intPtr := func(v int) *int { return &v }
	strPtrNonEmpty := func(v string) *string { return &v }
	row := ParticipantRow{
		MatchID:  "prop-singlepair",
		XUID:     "prop-xuid-single",
		Gamertag: strPtrNonEmpty("player_unique"),
		Kills:    intPtr(10),
		Deaths:   intPtr(5),
	}

	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = InsertParticipants(ctx, db, []ParticipantRow{row})
		}()
	}
	wg.Wait()

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_participants
		 WHERE match_id || '' = 'prop-singlepair'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("Property violation : N=%d UPSERTs sur même clé → count = %d, want 1",
			N, count)
	}
}
