//go:build integration

package sync

import (
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPerfDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kda DOUBLE, accuracy DOUBLE,
			time_played_seconds INTEGER,
			personal_score INTEGER, damage_dealt DOUBLE, damage_taken DOUBLE,
			rank INTEGER,
			team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE
		);
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			performance_chain VARCHAR,
			updated_at TIMESTAMPTZ
		);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	return db
}

func seedPerfMatches(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("m%04d", i)
		ts := fmt.Sprintf("2025-01-%02dT%02d:00:00Z", (i/24)+1, i%24)
		// pair_name "Arena:Slayer" → chaîne arena_slayer pour tous (cohérence test legacy).
		db.Exec(
			"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)",
			mid, ts, "Arena:Slayer", false, false)
		db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected) VALUES (?, 'xuid1', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mid, 2, 10+i, 5, 3, 1.5, 0.5, 600, 1000+i, 2000.0, 500.0, 1, 1500.0, 1500.0, 10.0, 5.0)
	}
}

func TestLoadHistoryForPerf_Empty(t *testing.T) {
	db := openPerfDB(t)
	rows, err := loadHistoryForPerf(t.Context(), db, "xuid_none")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0, got %d", len(rows))
	}
}

func TestLoadHistoryForPerf_WithData(t *testing.T) {
	db := openPerfDB(t)
	seedPerfMatches(t, db, 5)
	rows, err := loadHistoryForPerf(t.Context(), db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5, got %d", len(rows))
	}
}

// TestLoadHistoryForPerf_PopulatesChain garantit que loadHistoryForPerf
// dérive bien `Chain` depuis pair_name + is_ranked + is_firefight, en
// déléguant à GetPerformanceChain. Couvre tous les cas de figure : PvP non classé
// (chaînes LUSR), Firefight, pair_name NULL (fallback), et les DEUX familles du
// classé (scission D-A : ranked_slayer / ranked_objectif selon le sous-mode).
func TestLoadHistoryForPerf_PopulatesChain(t *testing.T) {
	db := openPerfDB(t)

	// 7 matchs avec des configurations différentes.
	insert := func(mid, ts, pair string, ranked, ff bool, useNullPair bool) {
		if useNullPair {
			db.Exec(
				"INSERT INTO match_registry (match_id, start_time, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?)",
				mid, ts, ranked, ff)
		} else {
			db.Exec(
				"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)",
				mid, ts, pair, ranked, ff)
		}
		db.Exec(
			"INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, time_played_seconds) VALUES (?, 'xuid1', 2, 5, 5, 600)",
			mid)
	}
	insert("c1", "2025-01-01T01:00:00Z", "BTB:Slayer", false, false, false)    // → btb
	insert("c2", "2025-01-01T02:00:00Z", "", true, false, false)               // → ranked_slayer (flag + pair vide → famille slayer)
	insert("c3", "2025-01-01T03:00:00Z", "Firefight:KOTH", false, true, false) // → firefight (flag wins)
	insert("c4", "2025-01-01T04:00:00Z", "", false, false, true)               // → fallback arena_slayer (pair NULL)
	insert("c5", "2025-01-01T05:00:00Z", "Ranked:Oddball", true, false, false) // → ranked_objectif (famille du sous-mode)
	insert("c6", "2025-01-01T06:00:00Z", "Ranked:Slayer", true, false, false)  // → ranked_slayer
	insert("c7", "2025-01-01T07:00:00Z", "", true, false, true)                // → ranked_slayer (ranked prime sur firefight)

	rows, err := loadHistoryForPerf(t.Context(), db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}

	want := map[string]string{
		"c1": LUSRChainBTB,
		"c2": PerfChainRankedSlayer,
		"c3": PerfChainFirefight,
		"c4": LUSRChainArenaSlayer, // fallback (pair_name NULL, no flag)
		"c5": PerfChainRankedObjectif,
		"c6": PerfChainRankedSlayer,
		"c7": PerfChainRankedSlayer,
	}
	for _, r := range rows {
		if r.Chain != want[r.MatchID] {
			t.Errorf("match %s : Chain=%q, want %q", r.MatchID, r.Chain, want[r.MatchID])
		}
	}
}

func TestBatchComputePerformanceScores_Empty(t *testing.T) {
	db := openPerfDB(t)
	n, err := batchComputePerformanceScores(t.Context(), db, db, "xuid_none", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestBatchComputePerformanceScores_WithData(t *testing.T) {
	db := openPerfDB(t)
	// Need >MinMatchesForRelative matches for any scoring
	seedPerfMatches(t, db, MinMatchesForRelative+10)
	n, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Some matches should get scored (those after MinMatchesForRelative)
	if n == 0 {
		t.Fatal("expected some matches to be scored")
	}
}

// seedPerfMatchesWithChain insère n matchs dans la chaîne spécifiée via
// (pair_name, is_ranked, is_firefight). startIdx permet de chaîner plusieurs
// chaînes dans une même DB sans collision d'IDs.
func seedPerfMatchesWithChain(t *testing.T, db *sql.DB, startIdx, n int, pairName string, isRanked, isFirefight bool) {
	t.Helper()
	for i := 0; i < n; i++ {
		idx := startIdx + i
		mid := fmt.Sprintf("m%04d", idx)
		// Espace les matchs par 1h pour garantir un ordre chronologique stable.
		ts := fmt.Sprintf("2025-01-%02dT%02d:00:00Z", (idx/24)+1, idx%24)
		db.Exec(
			"INSERT INTO match_registry (match_id, start_time, pair_name, is_ranked, is_firefight) VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)",
			mid, ts, pairName, isRanked, isFirefight)
		db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, personal_score, damage_dealt, damage_taken, rank, team_mmr, enemy_mmr, kills_expected, deaths_expected) VALUES (?, 'xuid1', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mid, 2, 10+i, 5, 3, 1.5, 0.5, 600, 1000+i, 2000.0, 500.0, 1, 1500.0, 1500.0, 10.0, 5.0)
		// Seed la row player_match_enrichment (posée à l'ingestion en prod).
		// Sans elle, BatchUpdateMulti UPDATE = no-op (0 rows affected).
		db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid)
	}
}

// TestBatchComputePerformanceScores_PartitionsByChain est la garantie clé du
// refactor "par chaîne" : un joueur qui a 5 matchs btb + 5 ranked + 12
// arena_slayer ne doit avoir des scores QUE sur arena_slayer (≥ 10), pas sur
// btb (< 10) ni sur ranked (< 10). Le seuil MinMatchesPerChainForRelative
// s'applique par chaîne, indépendamment du volume total.
//
// Vérifie aussi que la `performance_chain` stockée correspond bien à la
// chaîne d'appartenance du match (pas l'historique global).
func TestBatchComputePerformanceScores_PartitionsByChain(t *testing.T) {
	db := openPerfDB(t)

	// 5 matchs BTB (insuffisant pour scorer)
	seedPerfMatchesWithChain(t, db, 0, 5, "BTB:Slayer", false, false)
	// 5 matchs Ranked (insuffisant pour scorer)
	seedPerfMatchesWithChain(t, db, 100, 5, "Ranked:Slayer", true, false)
	// 12 matchs arena_slayer (10 premiers sans score, 2 derniers scorés)
	seedPerfMatchesWithChain(t, db, 200, 12, "Arena:Slayer", false, false)

	n, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 scored matches (12 arena_slayer - 10 threshold), got %d", n)
	}

	// Tous les scores stockés doivent être dans la chaîne arena_slayer.
	rows, err := db.Query(`
		SELECT match_id, performance_chain
		FROM player_match_enrichment_latest
		WHERE performance_score IS NOT NULL
		ORDER BY match_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type stored struct {
		matchID, chain string
	}
	var got []stored
	for rows.Next() {
		var s stored
		if err := rows.Scan(&s.matchID, &s.chain); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 stored rows, got %d (%v)", len(got), got)
	}
	for _, s := range got {
		if s.chain != LUSRChainArenaSlayer {
			t.Errorf("match %s : chain stockée = %q, want %q", s.matchID, s.chain, LUSRChainArenaSlayer)
		}
	}

	// Vérification finale : aucun match BTB ou Ranked n'a de score, peu importe
	// que d'autres matchs aient été scorés ailleurs.
	var btbScored, rankedScored int
	db.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment_latest pme
		JOIN match_registry mr ON pme.match_id = mr.match_id
		WHERE pme.performance_score IS NOT NULL AND mr.pair_name = 'BTB:Slayer'`).Scan(&btbScored)
	db.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment_latest pme
		JOIN match_registry mr ON pme.match_id = mr.match_id
		WHERE pme.performance_score IS NOT NULL AND mr.is_ranked = TRUE`).Scan(&rankedScored)
	if btbScored != 0 {
		t.Errorf("BTB matches should not have any score (5 < %d), got %d scored",
			MinMatchesPerChainForRelative, btbScored)
	}
	if rankedScored != 0 {
		t.Errorf("Ranked matches should not have any score (5 < %d), got %d scored",
			MinMatchesPerChainForRelative, rankedScored)
	}
}

// TestBatchComputePerformanceScores_SkipExistingPreservesChain vérifie qu'en
// mode !force, un match avec la MÊME chaîne déjà stockée n'est pas recomputé.
// Si la chaîne diffère (reclassification rare mais possible), recompute.
func TestBatchComputePerformanceScores_SkipExistingPreservesChain(t *testing.T) {
	db := openPerfDB(t)
	seedPerfMatchesWithChain(t, db, 0, 12, "Arena:Slayer", false, false)

	// 1er run : compute initial
	n1, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n1 != 2 {
		t.Fatalf("1er run : expected 2 scored, got %d", n1)
	}

	// 2e run sans force : devrait skipper tous (même chaîne).
	n2, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Errorf("2e run (!force, même chaîne) : expected 0 recomputed, got %d", n2)
	}

	// 3e run avec force=true : devrait tout recomputer.
	n3, err := batchComputePerformanceScores(t.Context(), db, db, "xuid1", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if n3 != 2 {
		t.Errorf("3e run (force=true) : expected 2 recomputed, got %d", n3)
	}
}
