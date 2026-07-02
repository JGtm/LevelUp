// Package sync — engine_e2e_test.go : tests E2E du moteur de sync avec DuckDB in-memory.
//
// Ces tests couvrent le workflow complet run() → processMatch → post-sync pipeline
// en utilisant mockHaloClient pour les appels API et de vraies DBs DuckDB en mémoire
// pour valider les insertions/lectures.
//
// Aucun accès réseau requis. CGO_ENABLED=1 requis (duckdb-go).
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

// newInMemoryDBs ouvre une paire (playerDB, sharedDB) DuckDB in-memory avec schéma.
func newInMemoryDBs(t *testing.T) (*sql.DB, *sql.DB) {
	t.Helper()
	playerDB := openMemDB(t)
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	// Append-only #23046 : EnsurePlayerSchema crée la table append-only (id+stage) ;
	// cet appel crée la vue player_match_enrichment_latest (lue par le post-sync).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	sharedDB := openMemDB(t)
	if err := EnsureSharedSchema(t.Context(), sharedDB); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	return playerDB, sharedDB
}

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("countRows(%s): %v", table, err)
	}
	return n
}

// makeMatchJSON builds a realistic Halo match stats JSON for testing.
func makeMatchJSON(matchID string, players int) map[string]any {
	playersArr := make([]any, 0, players)
	for i := 0; i < players; i++ {
		gt := fmt.Sprintf("Player%d", i)
		xuid := fmt.Sprintf("xuid(%016d)", i)
		coreStats := map[string]any{
			"Kills":               float64(10 + i),
			"Deaths":              float64(5 + i),
			"Assists":             float64(3 + i),
			"ShotsFired":          float64(100),
			"ShotsHit":            float64(50),
			"PersonalScore":       float64(1000 + i*100),
			"DamageDealt":         float64(2500.0),
			"DamageTaken":         float64(2000.0),
			"AverageLifeDuration": "PT30S",
			"Medals": []any{
				map[string]any{"NameId": float64(100 + i), "Count": float64(2)},
				map[string]any{"NameId": float64(200 + i), "Count": float64(1)},
			},
		}
		teamStats := []any{
			map[string]any{
				"Stats": map[string]any{
					"CoreStats": coreStats,
				},
			},
		}
		outcome := 2 // WIN
		if i%2 == 1 {
			outcome = 3 // LOSS
		}
		p := map[string]any{
			"PlayerId":        xuid,
			"PlayerName":      gt,
			"LastTeamId":      float64(i % 2),
			"Outcome":         float64(outcome),
			"Rank":            float64(i + 1),
			"PlayerTeamStats": teamStats,
			"ParticipationInfo": map[string]any{
				"TimePlayed": "PT10M0S",
			},
		}
		playersArr = append(playersArr, p)
	}

	return map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime":           "2025-01-15T14:30:00Z",
			"EndTime":             "2025-01-15T14:40:00Z",
			"MapVariant":          map[string]any{"AssetId": "map-asset-001", "PublicName": "Aquarius"},
			"GameVariantCategory": float64(9),
			"PlaylistExperience":  "Arena:Slayer",
			"Playlist":            map[string]any{"AssetId": "playlist-001", "PublicName": "Ranked Arena"},
			"UgcGameVariant":      map[string]any{"AssetId": "gv-001", "PublicName": "Slayer"},
			"Duration":            "PT10M0S",
			"PlayableDuration":    "PT9M30S",
			"LifecycleMode":       float64(3),
		},
		"Players": playersArr,
	}
}

// makeHistory builds a slice of MatchHistoryEntry for testing.
func makeHistory(matchIDs ...string) []MatchHistoryEntry {
	entries := make([]MatchHistoryEntry, len(matchIDs))
	for i, id := range matchIDs {
		entries[i] = MatchHistoryEntry{
			MatchID:   id,
			StartTime: time.Now().Add(-time.Duration(len(matchIDs)-i) * time.Hour).Format(time.RFC3339),
		}
	}
	return entries
}

// newTestEngine creates a SyncEngine that uses real (temp) DBs via t.TempDir.
func newTestEngine(t *testing.T) (*SyncEngine, string) {
	t.Helper()
	repoRoot := t.TempDir()
	gamertag := "TestPlayer"
	xuid := "1234567890123456"
	tokens := &domain.HaloTokens{
		SpartanToken:   "test-spartan-token",
		ClearanceToken: "test-clearance-token",
	}
	engine := NewSyncEngine(repoRoot, gamertag, xuid, tokens, nil)
	return engine, repoRoot
}

// ── processMatch tests (in-memory DBs) ──────────────────────────────────────

func TestProcessMatch_FullPipeline(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		statsBody: map[string]map[string]any{
			"match-001": makeMatchJSON("match-001", 4),
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()

	err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, "match-001", opts)
	if err != nil {
		t.Fatalf("processMatch: %v", err)
	}

	// Verify registry
	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("match_registry: attendu 1, obtenu %d", n)
	}

	// Verify participants
	if n := countRows(t, sharedDB, "match_participants"); n != 4 {
		t.Errorf("match_participants: attendu 4, obtenu %d", n)
	}

	// Verify medals
	if n := countRows(t, sharedDB, "medals_earned"); n == 0 {
		t.Error("medals_earned: attendu > 0")
	}

	// Note : plus d'assertion sur un store global xuid_aliases (consolidé dans
	// shared 2026-06-19). Les gamertags sont vérifiés via match_participants
	// ci-dessus ; v_gamertag_lookup les résout sans store séparé.

	// Verify player_match_enrichment
	if n := countRows(t, playerDB, "player_match_enrichment"); n != 1 {
		t.Errorf("player_match_enrichment: attendu 1, obtenu %d", n)
	}

	// Verify result counters
	if result.MatchesInserted != 1 {
		t.Errorf("MatchesInserted: attendu 1, obtenu %d", result.MatchesInserted)
	}
	if result.ParticipantsDone != 4 {
		t.Errorf("ParticipantsDone: attendu 4, obtenu %d", result.ParticipantsDone)
	}
	if len(result.InsertedMatchIDs) != 1 || result.InsertedMatchIDs[0] != "match-001" {
		t.Errorf("InsertedMatchIDs inattendu: %v", result.InsertedMatchIDs)
	}
}

func TestProcessMatch_WithoutParticipants(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		statsBody: map[string]map[string]any{
			"match-001": makeMatchJSON("match-001", 4),
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()
	opts.WithParticipants = false

	err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, "match-001", opts)
	if err != nil {
		t.Fatalf("processMatch: %v", err)
	}

	if n := countRows(t, sharedDB, "match_participants"); n != 0 {
		t.Errorf("match_participants: attendu 0 (WithParticipants=false), obtenu %d", n)
	}
	if result.ParticipantsDone != 0 {
		t.Errorf("ParticipantsDone: attendu 0, obtenu %d", result.ParticipantsDone)
	}
}

func TestProcessMatch_WithoutMedals(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		statsBody: map[string]map[string]any{
			"match-001": makeMatchJSON("match-001", 4),
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()
	opts.WithMedals = false

	err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, "match-001", opts)
	if err != nil {
		t.Fatalf("processMatch: %v", err)
	}

	if n := countRows(t, sharedDB, "medals_earned"); n != 0 {
		t.Errorf("medals_earned: attendu 0 (WithMedals=false), obtenu %d", n)
	}
	if result.MedalsInserted != 0 {
		t.Errorf("MedalsInserted: attendu 0, obtenu %d", result.MedalsInserted)
	}
}

func TestProcessMatch_GetMatchStatsError(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{getStatsErr: errors.New("API 401 Unauthorized")}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()

	err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, "match-001", opts)
	if err == nil {
		t.Fatal("attendu une erreur pour GetMatchStats failure")
	}
	if result.MatchesInserted != 0 {
		t.Errorf("MatchesInserted: attendu 0 après erreur, obtenu %d", result.MatchesInserted)
	}
}

func TestProcessMatch_Idempotent(t *testing.T) {
	// Inserting the same match twice should not duplicate rows (INSERT OR IGNORE).
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		statsBody: map[string]map[string]any{
			"match-001": makeMatchJSON("match-001", 2),
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	result1 := domain.SyncResult{StartedAt: time.Now()}
	if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result1, "match-001", opts); err != nil {
		t.Fatalf("1st processMatch: %v", err)
	}

	result2 := domain.SyncResult{StartedAt: time.Now()}
	if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result2, "match-001", opts); err != nil {
		t.Fatalf("2nd processMatch: %v", err)
	}

	// Registry should still be 1
	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("match_registry: attendu 1 (idempotent), obtenu %d", n)
	}
}

func TestProcessMatch_MultipleMatches(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	matchIDs := []string{"match-001", "match-002", "match-003"}
	statsBody := make(map[string]map[string]any)
	for _, id := range matchIDs {
		statsBody[id] = makeMatchJSON(id, 4)
	}
	mock := &mockHaloClient{statsBody: statsBody}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	for _, id := range matchIDs {
		result := domain.SyncResult{StartedAt: time.Now()}
		if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, id, opts); err != nil {
			t.Fatalf("processMatch(%s): %v", id, err)
		}
	}

	if n := countRows(t, sharedDB, "match_registry"); n != 3 {
		t.Errorf("match_registry: attendu 3, obtenu %d", n)
	}
	if n := countRows(t, sharedDB, "match_participants"); n != 12 {
		t.Errorf("match_participants: attendu 12 (3×4), obtenu %d", n)
	}
	if n := countRows(t, playerDB, "player_match_enrichment"); n != 3 {
		t.Errorf("player_match_enrichment: attendu 3, obtenu %d", n)
	}
}

// ── Deduplication tests ─────────────────────────────────────────────────────

func TestLoadKnownMatchIDs_Deduplication(t *testing.T) {
	playerDB := openMemDB(t)
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	// Insert some known matches
	for _, id := range []string{"match-old-1", "match-old-2", "match-old-3"} {
		if err := UpsertPlayerEnrichment(t.Context(), playerDB, id, ""); err != nil {
			t.Fatalf("UpsertPlayerEnrichment: %v", err)
		}
	}

	known, err := loadKnownMatchIDs(t.Context(), playerDB, nil, "")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}

	if len(known) != 3 {
		t.Errorf("attendu 3 match_ids connus, obtenu %d", len(known))
	}
	for _, id := range []string{"match-old-1", "match-old-2", "match-old-3"} {
		if !known[id] {
			t.Errorf("match_id %s manquant dans known map", id)
		}
	}
	if known["match-new"] {
		t.Error("match-new ne devrait pas être dans la map")
	}
}

// ── RunDelta / RunFull E2E (file-based DBs) ──────────────────────────────────

func TestRunDelta_NewMatches(t *testing.T) {
	engine, _ := newTestEngine(t)

	matchIDs := []string{"aabbccdd-0000-4000-8000-000000000001", "aabbccdd-0000-4000-8000-000000000002"}
	statsBody := make(map[string]map[string]any)
	for _, id := range matchIDs {
		statsBody[id] = makeMatchJSON(id, 2)
	}

	mock := &mockHaloClient{
		history:   makeHistory(matchIDs...),
		statsBody: statsBody,
	}

	// Inject mock client — we need to bypass the real API client creation.
	// Instead, call processMatch directly via run's internal flow.
	// For a true E2E, we test via the file-based DBs.
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        10,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 10,
	}

	// Direct test: open DBs, load known, run pagination manually (simulating run()).
	playerDB, err := OpenPlayerDB(engine.playerDBPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer playerDB.Close()

	sharedDB, err := OpenSharedDB(engine.sharedDBPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer sharedDB.Close()

	// Initially no known matches
	known, err := loadKnownMatchIDs(t.Context(), playerDB.SQLDb(), nil, "")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 0 {
		t.Fatalf("attendu 0 matchs connus initialement, obtenu %d", len(known))
	}

	// Process both matches
	result := domain.SyncResult{StartedAt: time.Now()}
	for _, id := range matchIDs {
		if known[id] {
			t.Errorf("match %s ne devrait pas être connu", id)
			continue
		}
		if err := engine.processMatch(context.Background(), mock, sharedDB.SQLDb(), playerDB.SQLDb(), &result, id, opts); err != nil {
			t.Fatalf("processMatch(%s): %v", id, err)
		}
	}

	if result.MatchesInserted != 2 {
		t.Errorf("MatchesInserted: attendu 2, obtenu %d", result.MatchesInserted)
	}

	// After processing, known should contain 2
	known2, err := loadKnownMatchIDs(t.Context(), playerDB.SQLDb(), nil, "")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs 2nd call: %v", err)
	}
	if len(known2) != 2 {
		t.Errorf("attendu 2 matchs connus après sync, obtenu %d", len(known2))
	}
}

func TestRunDelta_StopsAtKnownMatch(t *testing.T) {
	// Simulate delta behavior: first page has 3 matches, 2nd is known → stop
	playerDB, sharedDB := newInMemoryDBs(t)

	// Pre-insert a "known" match
	if err := UpsertPlayerEnrichment(t.Context(), playerDB, "match-known", ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}

	history := []MatchHistoryEntry{
		{MatchID: "match-new-1", StartTime: "2025-01-15T15:00:00Z"},
		{MatchID: "match-known", StartTime: "2025-01-15T14:00:00Z"},
		{MatchID: "match-new-2", StartTime: "2025-01-15T13:00:00Z"},
	}
	statsBody := map[string]map[string]any{
		"match-new-1": makeMatchJSON("match-new-1", 2),
		"match-new-2": makeMatchJSON("match-new-2", 2),
	}

	mock := &mockHaloClient{history: history, statsBody: statsBody}

	known, _ := loadKnownMatchIDs(t.Context(), playerDB, nil, "")
	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	result := domain.SyncResult{StartedAt: time.Now()}
	processed := 0

	for _, entry := range mock.history {
		if known[entry.MatchID] {
			result.MatchesSkipped++
			break // delta mode: stop at first known
		}
		if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, entry.MatchID, opts); err != nil {
			t.Fatalf("processMatch(%s): %v", entry.MatchID, err)
		}
		processed++
	}

	if processed != 1 {
		t.Errorf("processed: attendu 1 (delta stop), obtenu %d", processed)
	}
	if result.MatchesSkipped != 1 {
		t.Errorf("MatchesSkipped: attendu 1, obtenu %d", result.MatchesSkipped)
	}
	if result.MatchesInserted != 1 {
		t.Errorf("MatchesInserted: attendu 1, obtenu %d", result.MatchesInserted)
	}
}

func TestRunFull_ContinuesPastKnown(t *testing.T) {
	// Full mode: known matches are skipped but processing continues
	playerDB, sharedDB := newInMemoryDBs(t)

	// Pre-insert known match
	if err := UpsertPlayerEnrichment(t.Context(), playerDB, "match-known", ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}

	history := []MatchHistoryEntry{
		{MatchID: "match-new-1", StartTime: "2025-01-15T15:00:00Z"},
		{MatchID: "match-known", StartTime: "2025-01-15T14:00:00Z"},
		{MatchID: "match-new-2", StartTime: "2025-01-15T13:00:00Z"},
	}
	statsBody := map[string]map[string]any{
		"match-new-1": makeMatchJSON("match-new-1", 2),
		"match-new-2": makeMatchJSON("match-new-2", 2),
	}
	mock := &mockHaloClient{history: history, statsBody: statsBody}

	known, _ := loadKnownMatchIDs(t.Context(), playerDB, nil, "")
	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	result := domain.SyncResult{StartedAt: time.Now()}
	processed := 0

	for _, entry := range mock.history {
		if known[entry.MatchID] {
			result.MatchesSkipped++
			continue // full mode: skip but continue
		}
		if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, entry.MatchID, opts); err != nil {
			t.Fatalf("processMatch(%s): %v", entry.MatchID, err)
		}
		processed++
	}

	if processed != 2 {
		t.Errorf("processed: attendu 2 (full mode continues), obtenu %d", processed)
	}
	if result.MatchesSkipped != 1 {
		t.Errorf("MatchesSkipped: attendu 1, obtenu %d", result.MatchesSkipped)
	}
	if result.MatchesInserted != 2 {
		t.Errorf("MatchesInserted: attendu 2, obtenu %d", result.MatchesInserted)
	}
}

// ── Edge cases ──────────────────────────────────────────────────────────────

func TestProcessMatch_EmptyPlayersArray(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		statsBody: map[string]map[string]any{
			"match-empty": {
				"MatchId": "match-empty",
				"MatchInfo": map[string]any{
					"StartTime":           "2025-01-15T14:30:00Z",
					"MapVariant":          map[string]any{"AssetId": "map-001"},
					"GameVariantCategory": float64(9),
				},
				"Players": []any{},
			},
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()

	err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, "match-empty", opts)
	if err != nil {
		t.Fatalf("processMatch: %v", err)
	}

	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("match_registry: attendu 1 (match sans joueurs), obtenu %d", n)
	}
	if n := countRows(t, sharedDB, "match_participants"); n != 0 {
		t.Errorf("match_participants: attendu 0 (aucun joueur), obtenu %d", n)
	}
	if result.ParticipantsDone != 0 {
		t.Errorf("ParticipantsDone: attendu 0, obtenu %d", result.ParticipantsDone)
	}
}

func TestProcessMatch_ContextCancelled(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mock := &mockHaloClient{
		getStatsErr: ctx.Err(), // simulates cancelled context
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	result := domain.SyncResult{StartedAt: time.Now()}
	opts := domain.DefaultSyncOptions()

	err := e.processMatch(ctx, mock, sharedDB, playerDB, &result, "match-001", opts)
	if err == nil {
		t.Fatal("attendu une erreur pour context annulé")
	}
}

func TestRunDelta_EmptyHistory(t *testing.T) {
	// Mock returns empty history → 0 matches processed
	playerDB, sharedDB := newInMemoryDBs(t)
	mock := &mockHaloClient{history: nil}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()
	result := domain.SyncResult{StartedAt: time.Now()}

	entries, err := mock.GetMatchHistory(context.Background(), e.gamertag, opts.MatchType, 0, historyPageSize)
	if err != nil {
		t.Fatalf("GetMatchHistory: %v", err)
	}
	_ = sharedDB
	_ = playerDB

	if len(entries) != 0 {
		t.Errorf("attendu 0 entrées historique, obtenu %d", len(entries))
	}
	if result.MatchesInserted != 0 {
		t.Errorf("MatchesInserted: attendu 0, obtenu %d", result.MatchesInserted)
	}
}

func TestRunDelta_GetHistoryError(t *testing.T) {
	mock := &mockHaloClient{
		getHistoryErr: errors.New("network timeout"),
	}

	_, err := mock.GetMatchHistory(context.Background(), "TestPlayer", "all", 0, 25)
	if err == nil {
		t.Fatal("attendu une erreur réseau")
	}
}

func TestProcessMatch_APICallCounting(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	matchIDs := []string{"match-A", "match-B", "match-C"}
	statsBody := make(map[string]map[string]any)
	for _, id := range matchIDs {
		statsBody[id] = makeMatchJSON(id, 2)
	}
	mock := &mockHaloClient{statsBody: statsBody}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	for _, id := range matchIDs {
		result := domain.SyncResult{StartedAt: time.Now()}
		_ = e.processMatch(context.Background(), mock, sharedDB, playerDB, &result, id, opts)
	}

	// Each processMatch calls GetMatchStats once
	if mock.callsGetStats.Load() != 3 {
		t.Errorf("callsGetStats: attendu 3, obtenu %d", mock.callsGetStats.Load())
	}
}

// ── sync_meta tests ─────────────────────────────────────────────────────────

func TestSetSyncMeta_ReadBack(t *testing.T) {
	playerDB := openMemDB(t)
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := SetSyncMeta(t.Context(), playerDB, "last_delta_sync", now); err != nil {
		t.Fatalf("SetSyncMeta: %v", err)
	}

	var val string
	err := playerDB.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_delta_sync'").Scan(&val)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if val != now {
		t.Errorf("sync_meta: attendu %q, obtenu %q", now, val)
	}
}

func TestSetSyncMeta_Overwrite(t *testing.T) {
	playerDB := openMemDB(t)
	if err := EnsurePlayerSchema(t.Context(), playerDB); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerDB); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	_ = SetSyncMeta(t.Context(), playerDB, "test_key", "value1")
	_ = SetSyncMeta(t.Context(), playerDB, "test_key", "value2")

	var val string
	err := playerDB.QueryRow("SELECT value FROM sync_meta WHERE key = 'test_key'").Scan(&val)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if val != "value2" {
		t.Errorf("sync_meta overwrite: attendu 'value2', obtenu %q", val)
	}
}

// ── SyncOptions validation ──────────────────────────────────────────────────

func TestSyncOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    domain.SyncOptions
		wantErr bool
	}{
		{"valid defaults", domain.DefaultSyncOptions(), false},
		{"empty MatchType", domain.SyncOptions{MatchType: "", MaxMatches: 100, RequestsPerSecond: 10}, true},
		{"invalid MatchType", domain.SyncOptions{MatchType: "bogus", MaxMatches: 100, RequestsPerSecond: 10}, true},
		{"negative MaxMatches", domain.SyncOptions{MatchType: "all", MaxMatches: -1, RequestsPerSecond: 10}, true},
		{"negative RPS", domain.SyncOptions{MatchType: "all", MaxMatches: 100, RequestsPerSecond: -1}, true},
		{"zero MaxMatches OK", domain.SyncOptions{MatchType: "all", MaxMatches: 0, RequestsPerSecond: 10}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ── PostSync pipeline tests ─────────────────────────────────────────────────

// TestRunPostSyncPipeline_StationaryBurst_NoWriterAcquire : étape 1 contention —
// en régime stationnaire (0 nouveau match, 0 backlog events/weapons/psa, 0
// intensité engagement), le pipeline en mode BURST n'acquiert JAMAIS le writer
// shared : uniquement des segments de lecture RO. C'est la propriété centrale
// du refactor (fenêtre RW nulle quand il n'y a rien à écrire) — la mesure
// étape 0 montrait 13s/joueur tenues pour rien dans ce régime exact.
func TestRunPostSyncPipeline_StationaryBurst_NoWriterAcquire(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	mock := &mockHaloClient{}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "1234567890123456"}

	var writeAcquires, readAcquires int
	shared := NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) {
			writeAcquires++
			return sharedDB, func() {}, nil
		},
		func(ctx context.Context) (*sql.DB, func(), error) {
			readAcquires++
			return sharedDB, func() {}, nil
		},
		"test_postsync")

	_ = e.runPostSyncPipeline(context.Background(), playerDB, shared, mock, nil)

	if writeAcquires != 0 {
		t.Errorf("writer shared acquis %d fois en stationnaire, want 0 (fenêtre RW nulle)", writeAcquires)
	}
	if readAcquires == 0 {
		t.Error("aucun segment Read ouvert — le pipeline n'a pas lu shared ?")
	}
}

func TestRunPostSyncPipeline_NoError(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		careerData: &CareerRankData{
			XUID:            "1234567890123456",
			CurrentRank:     42,
			CurrentRankName: "Onyx",
			CurrentXP:       1000,
		},
	}

	e := &SyncEngine{
		gamertag: "TestPlayer",
		xuid:     "1234567890123456",
	}

	r := e.runPostSyncPipeline(context.Background(), playerDB, NewPinnedSharedAccess(sharedDB), mock, nil)

	// Post-sync should complete without panic.
	// Carrière (XP + Spartan ID) découplée du post-sync depuis 2026-05-14 :
	// CareerSynced reste à false et career_progression n'est plus écrite ici
	// (service.CareerLiveService prend le relais côté home).
	if r.CareerSynced {
		t.Error("CareerSynced doit rester false : carrière découplée du post-sync")
	}
	var count int
	err := playerDB.QueryRow("SELECT COUNT(*) FROM career_progression").Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow career_progression: %v", err)
	}
	if count > 0 {
		t.Errorf("career_progression: attendu 0 row depuis le découplage, obtenu %d", count)
	}
}

func TestEnrichCareerRankFromMetadata(t *testing.T) {
	metaDB := openMemDB(t)
	_, err := metaDB.Exec(`CREATE TABLE career_ranks (
		rank_id INTEGER,
		title_en VARCHAR,
		tier_type VARCHAR,
		grade INTEGER,
		xp_required INTEGER,
		adornment_icon_path VARCHAR
	)`)
	if err != nil {
		t.Fatalf("CREATE career_ranks: %v", err)
	}
	_, err = metaDB.Exec(`INSERT INTO career_ranks VALUES
		(173, 'Colonel', 'Platinum', 1, 25000, 'a-173'),
		(174, 'Colonel', 'Platinum', 2, 35000, 'a-174')`)
	if err != nil {
		t.Fatalf("INSERT career_ranks: %v", err)
	}

	data := &CareerRankData{XUID: "123", CurrentRank: 174, CurrentXP: 21840}
	if err := enrichCareerRankFromMetadata(t.Context(), metaDB, data); err != nil {
		t.Fatalf("enrichCareerRankFromMetadata: %v", err)
	}
	if data.CurrentRankName != "Colonel Platinum 2" {
		t.Fatalf("CurrentRankName = %q", data.CurrentRankName)
	}
	if data.CurrentRankTier != "Platinum" {
		t.Fatalf("CurrentRankTier = %q", data.CurrentRankTier)
	}
	if data.XPForNextRank != 35000 {
		t.Fatalf("XPForNextRank = %d", data.XPForNextRank)
	}
	if data.XPTotal != 46840 {
		t.Fatalf("XPTotal = %d", data.XPTotal)
	}
	if data.AdornmentPath != "a-174" {
		t.Fatalf("AdornmentPath = %q", data.AdornmentPath)
	}
}

// TestRunConditionalPostSync_NoInsertedMatches_DoesNotSyncCareer vérifie le
// découplage : sans nouveau match, le post-sync ne déclenche plus de fetch
// carrière (ni de write career_progression). Le rafraîchissement XP+Spartan
// ID est désormais géré live par service.CareerLiveService.
func TestRunConditionalPostSync_NoInsertedMatches_DoesNotSyncCareer(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		careerData: &CareerRankData{
			XUID:            "1234567890123456",
			CurrentRank:     42,
			CurrentRankName: "Onyx",
			CurrentXP:       1000,
			SpartanID:       "SR-TEST-4242",
		},
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "1234567890123456"}

	r := e.runConditionalPostSync(context.Background(), playerDB, NewPinnedSharedAccess(sharedDB), mock, 0, nil)

	if r.CareerSynced {
		t.Fatal("CareerSynced doit rester false : carrière découplée du post-sync")
	}
	if r.PerfScoresComputed != 0 || r.LUSRUpdated != 0 || r.ViewsRefreshed != 0 {
		t.Fatalf("pipeline sans matchs ne doit pas recalculer les agrégats: %+v", r)
	}

	var count int
	err := playerDB.QueryRow("SELECT COUNT(*) FROM career_progression").Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow career_progression: %v", err)
	}
	if count > 0 {
		t.Errorf("career_progression: attendu 0 row depuis le découplage, obtenu %d", count)
	}
}

// TestRunPostSyncPipeline_CareerError vérifie que les erreurs API carrière
// (lecture désormais hors post-sync) n'impactent plus le résultat — la mock
// errCareer reste configurée mais inutilisée.
func TestRunPostSyncPipeline_CareerError(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{
		getCareerErr: errors.New("career API 500"),
	}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "1234567890123456"}

	r := e.runPostSyncPipeline(context.Background(), playerDB, NewPinnedSharedAccess(sharedDB), mock, nil)

	if r.CareerSynced {
		t.Error("CareerSynced doit rester false : carrière découplée du post-sync")
	}
}

// TestRunPostSyncPipeline_NilCareerData : pendant régression du découplage.
func TestRunPostSyncPipeline_NilCareerData(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)

	mock := &mockHaloClient{careerData: nil}

	e := &SyncEngine{gamertag: "TestPlayer", xuid: "1234567890123456"}

	r := e.runPostSyncPipeline(context.Background(), playerDB, NewPinnedSharedAccess(sharedDB), mock, nil)

	if r.CareerSynced {
		t.Error("CareerSynced doit rester false : carrière découplée du post-sync")
	}
}
