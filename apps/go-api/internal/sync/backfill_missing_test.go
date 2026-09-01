//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// openSharedForAll crée un shared DB avec toutes les colonnes utilisées
// par findMatchesInSharedAll.
func openSharedForAll(t *testing.T) *sql.DB {
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
			start_time TIMESTAMPTZ DEFAULT now(),
			events_loaded BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			backfill_completed INTEGER DEFAULT 0,
			playlist_name VARCHAR,
			map_name VARCHAR,
			pair_name VARCHAR,
			game_variant_name VARCHAR,
			playable_duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR,
			gamertag VARCHAR,
			kills INTEGER, deaths INTEGER,
			shots_fired INTEGER, shots_hit INTEGER,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			avg_life_seconds DOUBLE,
			accuracy DOUBLE,
			team_mmr DOUBLE,
			backfill_bits INTEGER DEFAULT 0,
			outcome INTEGER,
			kda DOUBLE
		);
		CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT, count INTEGER);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}

	// Insert 3 matches
	for i := 1; i <= 3; i++ {
		db.Exec("INSERT INTO match_registry (match_id) VALUES (?)", mid(i))
		db.Exec(`INSERT INTO match_participants (match_id, xuid, kills, deaths, accuracy, team_mmr, shots_fired, shots_hit)
			VALUES (?, 'xuid1', 10, 5, NULL, NULL, NULL, NULL)`, mid(i))
	}

	return db
}

func mid(i int) string { return "match" + string(rune('0'+i)) }

func openPlayerForAll(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR PRIMARY KEY,
		performance_score DOUBLE,
		session_id VARCHAR,
		session_label VARCHAR,
		is_with_friends BOOLEAN DEFAULT FALSE,
		teammates_signature VARCHAR,
		is_excluded BOOLEAN DEFAULT FALSE,
		updated_at TIMESTAMPTZ
	)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	db.Exec(`CREATE TABLE personal_score_awards (match_id VARCHAR PRIMARY KEY)`)
	return db
}

func TestFindMatchesMissingData_NilScope2(t *testing.T) {
	_, err := FindMatchesMissingData(t.Context(), nil, nil, "xuid1", nil)
	if err == nil {
		t.Fatal("expected error for nil scope")
	}
}

func TestFindMatchesMissingData_EmptyScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{}
	scope.Resolve()

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestFindMatchesMissingData_MedalsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Medals: true}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// All matches missing medals for xuid1
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
}

func TestFindMatchesMissingData_EventsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Events: true}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// events_loaded=FALSE on all matches
	if len(result) != 3 {
		t.Fatalf("expected 3 for events, got %d", len(result))
	}
}

func TestFindMatchesMissingData_AccuracyScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Accuracy: true}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// accuracy=NULL for all
	if len(result) != 3 {
		t.Fatalf("expected 3 for accuracy, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ShotsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Shots: true}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for shots, got %d", len(result))
	}
}

func TestFindMatchesMissingData_EnemyMMRScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{EnemyMMR: true}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for enemy_mmr, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ParticipantsScores_WithLocal(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{
		ParticipantsScores: true,
		Medals:             true,
	}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// Both should return something
	if len(result) == 0 {
		t.Fatal("expected some results")
	}
}

func TestFindMatchesMissingData_MaxMatches(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Medals: true, MaxMatches: 2}

	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) > 2 {
		t.Fatalf("expected max 2, got %d", len(result))
	}
}

func TestFindMatchesMissingData_SkillScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Skill: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for skill, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForceSkill(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Skill: true, ForceSkill: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_skill, got %d", len(result))
	}
}

// ForceEvents : sans force, events_loaded=TRUE doit éliminer les matchs.
// Avec force, ils sont tous retournés.
func TestFindMatchesMissingData_EventsLoadedExcludesWithoutForce(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	// Marquer les 3 matchs comme already loaded
	sdb.Exec("UPDATE match_registry SET events_loaded=TRUE")

	scope := &SyncScope{Events: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 (events already loaded), got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForceEvents(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	// Marquer les 3 matchs comme already loaded — sans force ils seraient ignorés
	sdb.Exec("UPDATE match_registry SET events_loaded=TRUE")

	scope := &SyncScope{Events: true, ForceEvents: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_events, got %d", len(result))
	}
}

// ForcePersonalScores : sans force, des entrées dans personal_score_awards
// (avec un match_id hex valide qui matche celui de match_registry)
// doivent éliminer les matchs. Avec force, ils sont tous retournés.
//
// Note : on utilise des UUIDs hex car playerDoneGuard valide les match_id via
// isValidMatchID (a-f, 0-9, -) avant de construire la clause NOT IN.
func TestFindMatchesMissingData_ForcePersonalScores(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)

	// Créer 3 matchs avec des match_id hex valides
	hexIDs := []string{"aaaaaaaa-1111", "bbbbbbbb-2222", "cccccccc-3333"}
	sdb.Exec("DELETE FROM match_registry")
	sdb.Exec("DELETE FROM match_participants")
	for _, id := range hexIDs {
		sdb.Exec("INSERT INTO match_registry (match_id) VALUES (?)", id)
		sdb.Exec(`INSERT INTO match_participants (match_id, xuid, kills, deaths, accuracy, team_mmr, shots_fired, shots_hit)
			VALUES (?, 'xuid1', 10, 5, NULL, NULL, NULL, NULL)`, id)
		// Marquer comme déjà fait dans le player DB
		pdb.Exec("INSERT INTO personal_score_awards (match_id) VALUES (?)", id)
	}

	// Sans force : tous filtrés par la guard
	scopeNoForce := &SyncScope{PersonalScores: true}
	resultNoForce, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scopeNoForce)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultNoForce) != 0 {
		t.Fatalf("expected 0 (personal_scores already present), got %d", len(resultNoForce))
	}

	// Avec force : tous retournés
	scopeForce := &SyncScope{PersonalScores: true, ForcePersonalScores: true}
	resultForce, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scopeForce)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultForce) != 3 {
		t.Fatalf("expected 3 for force_personal_scores, got %d", len(resultForce))
	}
}

func TestFindMatchesMissingData_AssetsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Assets: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// playlist_name is NULL on all matches → 3 results
	if len(result) != 3 {
		t.Fatalf("expected 3 for assets, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForceAssetsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Assets: true, ForceAssets: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_assets, got %d", len(result))
	}
}

func TestFindMatchesMissingData_PVEScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{PVEStats: true}
	// No firefight matches → should return 0
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 firefight matches, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForcePVEScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	// Set one match as firefight
	sdb.Exec("UPDATE match_registry SET is_firefight=TRUE WHERE match_id='match1'")
	scope := &SyncScope{PVEStats: true, ForcePVEStats: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 firefight match, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ParticipantsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Participants: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for participants, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForceParticipantsScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{Participants: true, ForceParticipants: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_participants, got %d", len(result))
	}
}

func TestFindMatchesMissingData_PlayableDurationScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{PlayableDuration: true}
	// All matches have NULL playable_duration_seconds → all returned
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for playable_duration, got %d", len(result))
	}
}

func TestFindMatchesMissingData_ForceAliases(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{ForceAliases: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_aliases, got %d", len(result))
	}
}

func TestFindMatchesMissingData_PerformanceForce(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{PerformanceScores: true, ForcePerformanceScores: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 for force_performance, got %d", len(result))
	}
}

func TestFindMatchesMissingData_PersonalScoresScope(t *testing.T) {
	pdb := openPlayerForAll(t)
	sdb := openSharedForAll(t)
	scope := &SyncScope{PersonalScores: true}
	result, err := FindMatchesMissingData(t.Context(), pdb, sdb, "xuid1", scope)
	if err != nil {
		t.Fatal(err)
	}
	// All matches not in personal_score_awards
	if len(result) != 3 {
		t.Fatalf("expected 3 for personal_scores, got %d", len(result))
	}
}
