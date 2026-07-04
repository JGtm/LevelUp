//go:build integration

// Package sync — concurrent_multiplayer_e2e_test.go : test E2E multi-player
// pour la concurrence des writes shared.match_participants (Phase 5.3 du
// plan stabilisation 2026-05-22).
//
// Scénario : 3 joueurs syncent EN PARALLÈLE sur la même DB shared (mode
// scheduler post-Phase 3.4). 5 matchs sont PARTAGÉS entre les 3 joueurs
// (mêmes match_id) + 5 solos par joueur (match_id unique). Total :
//   - 5 partagés × 3 joueurs = 15 UPSERTs participants (3 par match
//     partagé, 5 matchs)
//   - 5 solos × 3 joueurs = 15 UPSERTs participants (1 par match solo)
//   - Total : 30 (match_id, xuid) uniques.
//
// Phase 2.3 singleflight + PK ON CONFLICT garantissent qu'il n'y a aucun
// doublon malgré la concurrence inter-joueur sur les matchs partagés.
//
// Le test :
// 1. Lance 3 goroutines (1 par joueur) qui font les 10 UPSERTs chacun en
//    parallèle (avec sync.WaitGroup).
// 2. Itère le scenario 100× pour stresser sous -race.
// 3. Vérifie après chaque cycle : 30 rows uniques, pas de doublon par PK,
//    chaque match partagé a 3 xuids, chaque match solo a 1 xuid.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupSharedDBE2E : DB minimale match_participants + match_registry pour
// le scénario multi-joueur partagé.
func setupSharedDBE2E(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
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

// player simule un joueur avec sa liste de matchs solo + références aux
// matchs partagés. Le sync simule l'écriture concurrente.
type playerE2E struct {
	xuid      string
	gamertag  string
	soloMatch []string // 5 match_ids uniques à ce joueur
}

// syncOnePlayer simule la phase "écriture participants" d'un sync :
//   - Pour chaque match partagé : insert un participant pour ce xuid.
//   - Pour chaque match solo : insert un participant pour ce xuid.
//
// Tous les inserts passent par InsertParticipants → singleflight protégé.
func syncOnePlayer(ctx context.Context, db *sql.DB, p playerE2E, sharedMatches []string) error {
	intPtr := func(v int) *int { return &v }
	strPtrNonEmpty := func(v string) *string { return &v }

	rows := make([]ParticipantRow, 0, len(sharedMatches)+len(p.soloMatch))
	for _, mid := range sharedMatches {
		rows = append(rows, ParticipantRow{
			MatchID:  mid,
			XUID:     p.xuid,
			Gamertag: strPtrNonEmpty(p.gamertag),
			TeamID:   intPtr(0),
			Outcome:  intPtr(2),
			Kills:    intPtr(10),
			Deaths:   intPtr(5),
			Assists:  intPtr(3),
		})
	}
	for _, mid := range p.soloMatch {
		rows = append(rows, ParticipantRow{
			MatchID:  mid,
			XUID:     p.xuid,
			Gamertag: strPtrNonEmpty(p.gamertag),
			TeamID:   intPtr(0),
			Outcome:  intPtr(2),
			Kills:    intPtr(8),
			Deaths:   intPtr(6),
			Assists:  intPtr(2),
		})
	}
	return InsertParticipants(ctx, db, rows)
}

// TestE2E_ConcurrentMultiPlayerSync_SharedAndSoloMatches : scénario
// canonique 3 joueurs × (5 partagés + 5 solos) en parallèle.
// 100 itérations sous -race.
func TestE2E_ConcurrentMultiPlayerSync_SharedAndSoloMatches(t *testing.T) {
	const N = 100 // 100 cycles consécutifs pour stresser la concurrence

	ctx := context.Background()

	// 5 matchs partagés (présents pour les 3 joueurs).
	sharedMatches := []string{
		"shared-001", "shared-002", "shared-003", "shared-004", "shared-005",
	}

	// 3 joueurs avec 5 matchs solos chacun.
	players := []playerE2E{
		{
			xuid:     "xuid-alice",
			gamertag: "Alice",
			soloMatch: []string{
				"alice-001", "alice-002", "alice-003", "alice-004", "alice-005",
			},
		},
		{
			xuid:     "xuid-bob",
			gamertag: "Bob",
			soloMatch: []string{
				"bob-001", "bob-002", "bob-003", "bob-004", "bob-005",
			},
		},
		{
			xuid:     "xuid-carol",
			gamertag: "Carol",
			soloMatch: []string{
				"carol-001", "carol-002", "carol-003", "carol-004", "carol-005",
			},
		},
	}

	// Calcul invariants attendus.
	// 5 partagés × 3 xuids = 15 rows participants pour les partagés.
	// 5 solos × 3 joueurs × 1 xuid = 15 rows pour les solos.
	// Total : 30 rows uniques par (match_id, xuid).
	const wantTotalRows = 15 + 15
	const wantSharedRowsPerMatch = 3 // 1 par xuid sur les 3 joueurs
	const wantSoloRowsPerMatch = 1

	for cycle := 0; cycle < N; cycle++ {
		db := setupSharedDBE2E(t)

		// Seed match_registry pour tous les matchs (partagés + solos).
		allMatches := append([]string{}, sharedMatches...)
		for _, p := range players {
			allMatches = append(allMatches, p.soloMatch...)
		}
		for _, mid := range allMatches {
			if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, pair_name)
				VALUES (?, NOW(), 'Slayer')`, mid); err != nil {
				t.Fatalf("cycle %d: seed registry %s: %v", cycle, mid, err)
			}
		}

		// 3 goroutines parallèles, 1 par joueur, qui font les UPSERTs.
		var wg sync.WaitGroup
		errs := make(chan error, len(players))
		for _, p := range players {
			p := p
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := syncOnePlayer(ctx, db, p, sharedMatches); err != nil {
					errs <- fmt.Errorf("player %s: %w", p.gamertag, err)
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatalf("cycle %d: %v", cycle, err)
		}

		// Vérif 1 : total rows == 30.
		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&total); err != nil {
			t.Fatal(err)
		}
		if total != wantTotalRows {
			t.Errorf("cycle %d: count = %d, want %d", cycle, total, wantTotalRows)
		}

		// Vérif 2 : chaque match partagé a exactement 3 xuids distincts.
		for _, mid := range sharedMatches {
			var n int
			if err := db.QueryRow(
				`SELECT COUNT(DISTINCT xuid) FROM match_participants
				 WHERE match_id || '' = ?`, mid).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != wantSharedRowsPerMatch {
				t.Errorf("cycle %d: match partagé %s a %d xuids, want %d",
					cycle, mid, n, wantSharedRowsPerMatch)
			}
		}

		// Vérif 3 : chaque match solo a exactement 1 xuid.
		for _, p := range players {
			for _, mid := range p.soloMatch {
				var n int
				if err := db.QueryRow(
					`SELECT COUNT(*) FROM match_participants
					 WHERE match_id || '' = ?`, mid).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n != wantSoloRowsPerMatch {
					t.Errorf("cycle %d: match solo %s (%s) a %d rows, want %d",
						cycle, mid, p.gamertag, n, wantSoloRowsPerMatch)
				}
			}
		}

		// Vérif 4 : aucun doublon (match_id, xuid). Si singleflight + PK
		// fonctionnent, c'est garanti — le test échoue si l'un casse.
		var dupCount int
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM (
				SELECT match_id, xuid, COUNT(*) AS c
				FROM match_participants
				GROUP BY match_id, xuid
				HAVING COUNT(*) > 1
			)`).Scan(&dupCount); err != nil {
			t.Fatal(err)
		}
		if dupCount != 0 {
			t.Errorf("cycle %d: %d paires (match_id, xuid) dupliquées (singleflight cassé ?)",
				cycle, dupCount)
		}

		db.Close()
	}
}
