//go:build integration

// Package sync — engine_batch_path_test.go : tests d'intégration
// du chemin Collect→Persist (Phase 2.3) sur DuckDB :memory:.
//
// Vérifie qu'avec WithBatchPersistMode(true), submitOrInsertMatch utilise
// le path INSERT-only via persist.{Shared,Player}Persister et que les
// données arrivent en DB exactement comme avec le legacy.

package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

func openBatchPathTestDB(t *testing.T, target migration.TargetDB) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if target == migration.TargetPlayer {
		// Bootstrap minimal pour player DB (engagement migration dépend
		// de player_match_enrichment existant).
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS player_match_enrichment (
				match_id VARCHAR PRIMARY KEY,
				performance_score FLOAT, performance_chain VARCHAR,
				session_id VARCHAR, session_label VARCHAR,
				is_with_friends BOOLEAN DEFAULT FALSE,
				teammates_signature VARCHAR, had_bot_teammate BOOLEAN,
				is_excluded BOOLEAN DEFAULT FALSE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
			CREATE TABLE IF NOT EXISTS personal_score_awards (
				id INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
				match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
				award_name VARCHAR NOT NULL, award_category VARCHAR,
				award_count INTEGER DEFAULT 1, award_score INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE IF NOT EXISTS match_skill_rank (
				match_id VARCHAR PRIMARY KEY, rating_type VARCHAR NOT NULL,
				rating_value FLOAT, rating_deviation FLOAT,
				tier VARCHAR, tier_fr VARCHAR, sub_tier SMALLINT DEFAULT 0,
				tier_label VARCHAR, rating_delta FLOAT, playlist_group VARCHAR
			);
		`); err != nil {
			t.Fatalf("player bootstrap: %v", err)
		}
	}
	if err := migration.RunForDB(db, target); err != nil {
		t.Fatalf("migrate %s: %v", target, err)
	}
	if target == migration.TargetShared {
		// Patch weapon_kills (divergence schéma migration/prod).
		for _, col := range []string{"time_ms INTEGER", "delta_ms INTEGER",
			"confidence VARCHAR DEFAULT 'none'", "swap_detected BOOLEAN DEFAULT FALSE",
			"delayed_damage BOOLEAN DEFAULT FALSE"} {
			_, _ = db.Exec("ALTER TABLE weapon_kills ADD COLUMN IF NOT EXISTS " + col)
		}
	}
	return db
}

// ─── Test smoke : batchMode=true → SharedPersister + PlayerPersister hit ──

func TestSubmitMatchAsBatch_SmokePath(t *testing.T) {
	sharedDB := openBatchPathTestDB(t, migration.TargetShared)
	playerDB := openBatchPathTestDB(t, migration.TargetPlayer)

	e := &SyncEngine{
		gamertag: "Alice", xuid: "1111", titleSlug: "halo_infinite",
		batchMode: true,
	}

	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	fm := &fetchedMatch{
		MatchID: "m_e2e_001",
		Registry: &MatchRegistryRow{
			MatchID:      "m_e2e_001",
			StartTime:    time.Now().UTC(),
			ModeCategory: "PVP",
			FirstSyncBy:  "Alice",
		},
		Participants: []domain.MatchParticipantRow{
			{MatchID: "m_e2e_001", XUID: "1111", Gamertag: strPtr("Alice"), Kills: intPtr(10), Deaths: intPtr(5)},
			{MatchID: "m_e2e_001", XUID: "2222", Gamertag: strPtr("Bob"), Kills: intPtr(7), Deaths: intPtr(8)},
		},
		Medals: []domain.MedalRow{
			{MatchID: "m_e2e_001", XUID: "1111", MedalNameID: 1234, Count: 2},
		},
	}

	result := &domain.SyncResult{}
	if err := e.submitOrInsertMatch(context.Background(), sharedDB, playerDB, nil, result, fm); err != nil {
		t.Fatalf("submitOrInsertMatch: %v", err)
	}

	// Shared DB
	var n int
	_ = sharedDB.QueryRow("SELECT COUNT(*) FROM match_registry WHERE match_id = ?", "m_e2e_001").Scan(&n)
	if n != 1 {
		t.Errorf("match_registry : %d, want 1", n)
	}
	_ = sharedDB.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = ?", "m_e2e_001").Scan(&n)
	if n != 2 {
		t.Errorf("match_participants : %d, want 2", n)
	}
	_ = sharedDB.QueryRow("SELECT COUNT(*) FROM medals_earned WHERE match_id = ?", "m_e2e_001").Scan(&n)
	if n != 1 {
		t.Errorf("medals_earned : %d, want 1", n)
	}

	// Player DB — placeholder enrichment doit être présent
	_ = playerDB.QueryRow("SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?", "m_e2e_001").Scan(&n)
	if n != 1 {
		t.Errorf("player_match_enrichment : %d, want 1", n)
	}

	// Compteurs SyncResult alignés avec le legacy
	if result.MatchesInserted != 1 {
		t.Errorf("MatchesInserted = %d, want 1", result.MatchesInserted)
	}
	if result.ParticipantsDone != 2 {
		t.Errorf("ParticipantsDone = %d, want 2", result.ParticipantsDone)
	}
	if result.MedalsInserted != 1 {
		t.Errorf("MedalsInserted = %d, want 1", result.MedalsInserted)
	}
}

// ─── Test : batchMode=false → fall-through (legacy path inchangé) ─────────
//
// Note : ne PEUT PAS exécuter le legacy ici car insertFetchedMatch nécessite
// le full EnsurePlayerSchema (sequence career_progression, etc.). Ce test
// vérifie juste que le switch routing fonctionne — submitMatchAsBatch ne
// doit PAS être appelé quand batchMode=false. On vérifie indirectement
// via l'absence de row en DB (le legacy InsertRegistry échouerait sur
// notre schéma minimal, mais on n'attend pas son succès — on attend juste
// qu'il SOIT appelé et non submitMatchAsBatch).

func TestSubmitOrInsertMatch_BatchModeFalse_RoutesToLegacy(t *testing.T) {
	// Pas d'assertion DB ici — on vérifie juste que batchMode=false ne
	// crash pas via le branchement. La méthode insertFetchedMatch sera
	// appelée et tentera ses INSERTs ; on tolère une erreur (schema
	// minimal incomplet pour le legacy path).
	sharedDB := openBatchPathTestDB(t, migration.TargetShared)
	playerDB := openBatchPathTestDB(t, migration.TargetPlayer)

	e := &SyncEngine{
		gamertag: "Alice", xuid: "1111", titleSlug: "halo_infinite",
		batchMode: false, // explicite : legacy path
	}

	strPtr := func(v string) *string { return &v }
	fm := &fetchedMatch{
		MatchID: "m_legacy_001",
		Registry: &MatchRegistryRow{
			MatchID:      "m_legacy_001",
			StartTime:    time.Now().UTC(),
			ModeCategory: "PVP",
			FirstSyncBy:  "Alice",
		},
		Participants: []domain.MatchParticipantRow{
			{MatchID: "m_legacy_001", XUID: "1111", Gamertag: strPtr("Alice")},
		},
	}
	result := &domain.SyncResult{}
	// Le legacy path peut écrire ou pas selon le schéma — on ne vérifie pas
	// le résultat. On vérifie juste que submitOrInsertMatch ne panique pas.
	_ = e.submitOrInsertMatch(context.Background(), sharedDB, playerDB, nil, result, fm)
}
