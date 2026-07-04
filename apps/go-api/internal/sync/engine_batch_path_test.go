//go:build integration

// Package sync — engine_batch_path_test.go : tests d'intégration
// du chemin Collect→Persist (Phase 2.3) sur DuckDB :memory:.
//
// Vérifie que persistFetchedMatch utilise le path INSERT-only via
// persist.{Shared,Player}Persister et que les données arrivent en DB.

package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
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
		// Patch match_participants : first_joined_time / last_leave_time ajoutés
		// par shared_add_participation_timestamps (title-owned, hors registre global).
		for _, col := range []string{"first_joined_time TIMESTAMPTZ", "last_leave_time TIMESTAMPTZ"} {
			_, _ = db.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col)
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
	}

	intPtr := func(v int) *int { return &v }
	strPtrNonEmpty := func(v string) *string { return &v }

	fm := &fetchedMatch{
		MatchID: "m_e2e_001",
		Registry: &MatchRegistryRow{
			MatchID:      "m_e2e_001",
			StartTime:    time.Now().UTC(),
			ModeCategory: "PVP",
			FirstSyncBy:  "Alice",
		},
		Participants: []domain.MatchParticipantRow{
			{MatchID: "m_e2e_001", XUID: "1111", Gamertag: strPtrNonEmpty("Alice"), Kills: intPtr(10), Deaths: intPtr(5)},
			{MatchID: "m_e2e_001", XUID: "2222", Gamertag: strPtrNonEmpty("Bob"), Kills: intPtr(7), Deaths: intPtr(8)},
		},
		Medals: []domain.MedalRow{
			{MatchID: "m_e2e_001", XUID: "1111", MedalNameID: 1234, Count: 2},
		},
	}

	result := &domain.SyncResult{}
	if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm); err != nil {
		t.Fatalf("persistFetchedMatch: %v", err)
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

// ─── Test E2E Phase 2.4 : fetchMatchData → submitMatchAsBatch → DB ────────
//
// Exerce le pipeline complet du chemin Collect→Persist avec mockHaloClient :
// HTTP API mocked → fetchMatchData (parse JSON, build fetchedMatch) →
// persistFetchedMatch (batchMode=true) → SharedPersister + PlayerPersister →
// rows en DB. Aucune dépendance réseau.

// patchSharedSchemaForBatch ajoute les colonnes du Phase 2.1+ schema qui
// sont créées par migration mais absentes de sharedSchemaSQL (EnsureSharedSchema).
// Mirror du patch test-local appliqué dans openSharedTestDB du package persist.
func patchSharedSchemaForBatch(t *testing.T, sharedDB *sql.DB) {
	t.Helper()
	for _, col := range []string{
		"match_intensity DOUBLE",
		"backfill_completed BIGINT DEFAULT 0",
	} {
		if _, err := sharedDB.Exec("ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_registry %s: %v", col, err)
		}
	}
	// backfill_bits + colonnes mécaniques de kill natives Halo 5
	// (add_h5_kill_mechanics_columns, steps_shared_core.go) : créées par migration
	// title-owned mais absentes de sharedSchemaSQL statique. SharedPersister.Persist
	// les écrit → sans ce patch : Binder Error "column assassination_kills".
	for _, col := range []string{
		"backfill_bits INTEGER",
		"assassination_kills SMALLINT DEFAULT 0",
		"ground_pound_kills SMALLINT DEFAULT 0",
		"shoulder_bash_kills SMALLINT DEFAULT 0",
	} {
		if _, err := sharedDB.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_participants %s: %v", col, err)
		}
	}
}

func TestE2ECollectPersist_FetchThenBatchSubmit(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	patchSharedSchemaForBatch(t, sharedDB)

	// 2 matchs avec ~2 joueurs chacun + médailles.
	matchIDs := []string{
		"aabbccdd-0000-4000-8000-000000000001",
		"aabbccdd-0000-4000-8000-000000000002",
	}
	statsBody := map[string]map[string]any{}
	for _, id := range matchIDs {
		statsBody[id] = makeMatchJSON(id, 2)
	}
	mock := &mockHaloClient{
		history:   makeHistory(matchIDs...),
		statsBody: statsBody,
	}

	e := &SyncEngine{
		gamertag: "Player0", xuid: "0000000000000000",
		titleSlug: "halo_infinite",
	}

	opts := domain.SyncOptions{
		MatchType:        "matchmaking",
		MaxMatches:       10,
		WithParticipants: true,
		WithMedals:       true,
	}

	result := &domain.SyncResult{}
	for _, id := range matchIDs {
		fm, err := e.fetchMatchData(context.Background(), mock, id, opts)
		if err != nil {
			t.Fatalf("fetchMatchData(%s): %v", id, err)
		}
		if fm == nil {
			t.Fatalf("fetchMatchData retourne nil pour %s", id)
		}
		if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm); err != nil {
			t.Fatalf("persistFetchedMatch(%s): %v", id, err)
		}
	}

	// Compteurs SyncResult
	if result.MatchesInserted != 2 {
		t.Errorf("MatchesInserted = %d, want 2", result.MatchesInserted)
	}

	// Shared DB
	if n := countRows(t, sharedDB, "match_registry"); n != 2 {
		t.Errorf("match_registry : %d, want 2", n)
	}
	if n := countRows(t, sharedDB, "match_participants"); n != 4 {
		t.Errorf("match_participants : %d, want 4 (2 matchs x 2 joueurs)", n)
	}
	if n := countRows(t, sharedDB, "medals_earned"); n < 1 {
		t.Errorf("medals_earned : %d, want >= 1", n)
	}

	// Player DB — enrichment placeholder pour chaque match
	if n := countRows(t, playerDB, "player_match_enrichment"); n != 2 {
		t.Errorf("player_match_enrichment : %d, want 2", n)
	}
}

// ─── Test E2E ASYNC : queue + worker + Drain ───────────────────────────

// Vérifie que le path async (batchQueue non-nil) Submit → worker persiste →
// Drain bloque jusqu'à ACK complet → DB peuplée. Vérifie aussi que les WAL
// files sont nettoyés post-ACK.
func TestE2ECollectPersist_AsyncQueuePath_DrainBlocksUntilPersisted(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	patchSharedSchemaForBatch(t, sharedDB)

	// Setup BatchQueue + Worker pour le shared DB (path async).
	walDir := t.TempDir()
	q, err := persist.NewBatchQueue(persist.BatchQueueConfig{WALDir: walDir, ChanBufSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = q.Close() })

	// Worker shared : consomme la queue + persiste via SharedPersister.
	sharedP := persist.NewSharedPersister(sharedDB)
	playerP := persist.NewPlayerPersister(playerDB)
	combined := &combinedPersister{shared: sharedP, player: playerP}
	w := persist.NewWorker("test-async", q, persist.TargetShared, combined)

	wCtx, wCancel := context.WithCancel(context.Background())
	defer wCancel()
	go func() { _ = w.Run(wCtx) }()

	// SyncEngine en mode batchMode=true + batchQueue (chemin async).
	mock := &mockHaloClient{
		history:   makeHistory("aabbccdd-0000-4000-8000-000000000088"),
		statsBody: map[string]map[string]any{"aabbccdd-0000-4000-8000-000000000088": makeMatchJSON("aabbccdd-0000-4000-8000-000000000088", 2)},
	}
	e := &SyncEngine{
		gamertag: "Player0", xuid: "0000000000000000",
		titleSlug:  "halo_infinite",
		batchQueue: q, // ← active le path async
	}
	opts := domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 10,
		WithParticipants: true, WithMedals: true,
	}
	fm, err := e.fetchMatchData(context.Background(), mock, "aabbccdd-0000-4000-8000-000000000088", opts)
	if err != nil {
		t.Fatal(err)
	}
	result := &domain.SyncResult{}
	if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm); err != nil {
		t.Fatalf("persistFetchedMatch async: %v", err)
	}

	// Drain : attendre que le worker ait persisté.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()
	if err := q.Drain(drainCtx); err != nil {
		t.Fatalf("Drain: %v", err)
	}

	// Post-Drain : le batch doit être persisté en DB ET les WAL files supprimés.
	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("match_registry post-async : %d, want 1", n)
	}
	if pending, _ := q.PendingCount(); pending != 0 {
		t.Errorf("PendingCount post-Drain : %d, want 0", pending)
	}
}

// combinedPersister combine SharedPersister + PlayerPersister en 1 BatchPersister
// pour le test async (le Worker n'a qu'un seul persister).
type combinedPersister struct {
	shared *persist.SharedPersister
	player *persist.PlayerPersister
}

func (c *combinedPersister) Persist(ctx context.Context, batch *persist.MatchBatch) error {
	if err := c.shared.Persist(ctx, batch); err != nil {
		return err
	}
	return c.player.Persist(ctx, batch)
}

// ─── Property INSERT-only : re-submit même match n'écrase pas ────────────

func TestE2ECollectPersist_ReSubmitMatch_IdempotentNoOverwrite(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	patchSharedSchemaForBatch(t, sharedDB)

	id := "aabbccdd-0000-4000-8000-000000000099"
	stats := map[string]map[string]any{id: makeMatchJSON(id, 2)}
	mock := &mockHaloClient{
		history:   makeHistory(id),
		statsBody: stats,
	}

	e := &SyncEngine{
		gamertag: "Player0", xuid: "0000000000000000",
		titleSlug: "halo_infinite",
	}
	opts := domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 10,
		WithParticipants: true, WithMedals: true,
	}

	// 1ère sync
	fm1, err := e.fetchMatchData(context.Background(), mock, id, opts)
	if err != nil {
		t.Fatal(err)
	}
	result := &domain.SyncResult{}
	if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm1); err != nil {
		t.Fatal(err)
	}

	// Capture l'état initial — kills du joueur "xuid(0000000000000000)"
	var initialKills int
	err = sharedDB.QueryRow(
		"SELECT kills FROM match_participants WHERE match_id = ? AND xuid LIKE ?",
		id, "%0000000000000000%").Scan(&initialKills)
	if err != nil {
		t.Fatal(err)
	}

	// 2e submit du MÊME match (simule retry / re-sync)
	fm2, _ := e.fetchMatchData(context.Background(), mock, id, opts)
	if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm2); err != nil {
		t.Fatalf("2e persistFetchedMatch: %v", err)
	}

	// La row doit toujours avoir 1 entrée registry + N participants
	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("registry : %d après re-submit, want 1 (INSERT-only skip)", n)
	}
	// kills initial préservé (pas d'UPDATE)
	var afterKills int
	_ = sharedDB.QueryRow(
		"SELECT kills FROM match_participants WHERE match_id = ? AND xuid LIKE ?",
		id, "%0000000000000000%").Scan(&afterKills)
	if afterKills != initialKills {
		t.Errorf("kills modifié post-re-submit : initial=%d after=%d (INSERT-only viole)", initialKills, afterKills)
	}
}
