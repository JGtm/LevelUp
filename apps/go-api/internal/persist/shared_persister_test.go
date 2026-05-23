//go:build integration

// Package persist — shared_persister_test.go : tests TDD pour SharedPersister.
//
// Contrat à valider AVANT impl :
//
//  1. Persist d'un batch nouveau → toutes les rows INSERTées dans
//     match_registry, match_participants, medals_earned, weapon_kills,
//     killer_victim_pairs, xuid_aliases, highlight_events.
//  2. Persist d'un batch dont match_id existe déjà → no-op (idempotent),
//     pas d'erreur, pas de DELETE/UPDATE.
//  3. Persist d'un batch avec Shared.Match == nil → no-op.
//  4. Persist d'un batch avec xuid_aliases qui pre-existent → pas d'erreur
//     (INSERT OR IGNORE / ON CONFLICT DO NOTHING sur xuid_aliases uniquement).
//  5. Atomicité : si une INSERT échoue mid-batch → ROLLBACK complet
//     (aucune row partielle dans aucune table).
//  6. INSERT-only : aucun UPDATE émis sur les tables (vérifié via property
//     que ré-persist d'un même match_id ne modifie pas les valeurs existantes).

package persist

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// ─── Helpers ───────────────────────────────────────────────────────────────

// openSharedTestDB ouvre une DuckDB in-memory avec le schéma shared appliqué.
//
// Patch weapon_kills : le `create_base_shared_schema` crée la table en forme
// aggregée (4 cols : match_id, xuid, weapon_id, kills) mais `add_weapon_kills`
// (la forme per-kill avec time_ms, delta_ms, confidence, etc.) est skip car
// `CREATE TABLE IF NOT EXISTS` no-op. En prod, le schéma per-kill provient
// d'une migration Python ancienne — pas répliqué côté Go. À traiter dans un
// fix migration dédié (cf. thought_log 2026-05-23). Patch test-local en
// attendant pour ne pas bloquer le refactor Collect→Persist.
func openSharedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	for _, col := range []string{
		"time_ms INTEGER",
		"delta_ms INTEGER",
		"confidence VARCHAR DEFAULT 'none'",
		"swap_detected BOOLEAN DEFAULT FALSE",
		"delayed_damage BOOLEAN DEFAULT FALSE",
	} {
		if _, err := db.Exec("ALTER TABLE weapon_kills ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch weapon_kills add %s: %v", col, err)
		}
	}
	return db
}

// helperBuildSampleBatch construit un batch de test complet avec 1 match,
// 2 participants, 1 médaille, 1 weapon_kill, 1 killer_victim, 2 xuid_aliases.
func helperBuildSampleBatch(matchID, xuid, gamertag string) *MatchBatch {
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }
	u64Ptr := func(v uint64) *uint64 { return &v }
	tNow := time.Now().UTC()

	builder := NewBatchBuilder("halo_infinite", gamertag, xuid, "test")
	builder.SetMatch(&domain.MatchRegistryRow{
		MatchID:         matchID,
		StartTime:       tNow,
		ModeCategory:    "PVP",
		IsRanked:        false,
		IsFirefight:     false,
		DurationSeconds: intPtr(600),
		FirstSyncBy:     gamertag,
	})
	builder.AddParticipants([]domain.MatchParticipantRow{
		{
			MatchID: matchID, XUID: xuid, Gamertag: strPtr(gamertag),
			TeamID: intPtr(0), Outcome: intPtr(2),
			Kills: intPtr(10), Deaths: intPtr(5), Assists: intPtr(3),
		},
		{
			MatchID: matchID, XUID: "9876543210", Gamertag: strPtr("FriendA"),
			TeamID: intPtr(1), Outcome: intPtr(3),
			Kills: intPtr(7), Deaths: intPtr(8), Assists: intPtr(2),
		},
	})
	builder.AddMedals([]domain.MedalRow{
		{MatchID: matchID, XUID: xuid, MedalNameID: 12345, Count: 2},
		{MatchID: matchID, XUID: xuid, MedalNameID: 67890, Count: 1},
	})
	builder.AddWeaponKills([]WeaponKillInsert{
		{
			MatchID: matchID, XUID: xuid,
			TimeMS:          5000,
			WeaponID:        u64Ptr(1234567890),
			Confidence:      "high",
			AttributionPath: "primary",
		},
	})
	builder.AddKillerVictim([]KillerVictimInsert{
		{MatchID: matchID, KillerXUID: xuid, VictimXUID: "9876543210", Count: 1},
	})
	builder.AddXUIDAliases([]XUIDAliasInsert{
		{XUID: xuid, Gamertag: gamertag, LastSeen: tNow},
		{XUID: "9876543210", Gamertag: "FriendA", LastSeen: tNow},
	})
	builder.AddHighlightEvents([]HighlightEventInsert{
		{MatchID: matchID, XUID: strPtr(xuid), EventType: "Kill", TimeMS: 5000},
	})
	return builder.Build()
}

// countRows compte les rows d'une table avec un WHERE simple.
func countRows(t *testing.T, db *sql.DB, table, where string, args ...any) int {
	t.Helper()
	var n int
	q := "SELECT COUNT(*) FROM " + table
	if where != "" {
		q += " WHERE " + where
	}
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// ─── Test 1 : NewMatch_InsertsAllRows ──────────────────────────────────────

func TestSharedPersister_NewMatch_InsertsAllRows(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	batch := helperBuildSampleBatch("m_new_001", "1111", "Alice")
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	checks := []struct {
		table, where string
		args         []any
		want         int
	}{
		{"match_registry", "match_id = ?", []any{"m_new_001"}, 1},
		{"match_participants", "match_id = ?", []any{"m_new_001"}, 2},
		{"medals_earned", "match_id = ?", []any{"m_new_001"}, 2},
		{"weapon_kills", "match_id = ?", []any{"m_new_001"}, 1},
		{"killer_victim_pairs", "match_id = ?", []any{"m_new_001"}, 1},
		{"xuid_aliases", "xuid IN (?, ?)", []any{"1111", "9876543210"}, 2},
		{"highlight_events", "match_id = ?", []any{"m_new_001"}, 1},
	}
	for _, c := range checks {
		got := countRows(t, db, c.table, c.where, c.args...)
		if got != c.want {
			t.Errorf("%s WHERE %s : got %d, want %d", c.table, c.where, got, c.want)
		}
	}
}

// ─── Test 2 : EmptyMatch_NoOp ──────────────────────────────────────────────

func TestSharedPersister_EmptyMatch_NoOp(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	// Batch sans Match → SharedPersister doit no-op silently.
	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	batch := builder.Build()

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist empty: %v", err)
	}

	if n := countRows(t, db, "match_registry", ""); n != 0 {
		t.Errorf("match_registry doit être vide (no-op), got %d rows", n)
	}
}

// ─── Test 3 : DuplicateMatchID_Idempotent ──────────────────────────────────

func TestSharedPersister_DuplicateMatchID_Idempotent(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	batch1 := helperBuildSampleBatch("m_dup_001", "1111", "Alice")
	if err := p.Persist(context.Background(), batch1); err != nil {
		t.Fatalf("1er Persist: %v", err)
	}

	// 2e batch — même match_id, kills modifiés (devrait NE PAS écraser)
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }
	batch2 := helperBuildSampleBatch("m_dup_001", "1111", "Alice")
	batch2.Shared.Participants = []domain.MatchParticipantRow{
		{
			MatchID: "m_dup_001", XUID: "1111", Gamertag: strPtr("Alice"),
			Kills: intPtr(99999), // valeur sentinelle, doit pas écraser le 10 initial
		},
	}

	if err := p.Persist(context.Background(), batch2); err != nil {
		t.Fatalf("2e Persist (idempotent attendu): %v", err)
	}

	// Vérifier que la valeur initiale (kills=10) n'a pas été modifiée.
	var kills int
	err := db.QueryRow(
		"SELECT kills FROM match_participants WHERE match_id = ? AND xuid = ?",
		"m_dup_001", "1111",
	).Scan(&kills)
	if err != nil {
		t.Fatalf("query kills: %v", err)
	}
	if kills != 10 {
		t.Errorf("kills doit rester 10 (INSERT-only), got %d", kills)
	}

	// Vérifier qu'il n'y a toujours qu'1 row registry + 2 participants
	if n := countRows(t, db, "match_registry", "match_id = ?", "m_dup_001"); n != 1 {
		t.Errorf("match_registry doit avoir 1 row, got %d", n)
	}
	if n := countRows(t, db, "match_participants", "match_id = ?", "m_dup_001"); n != 2 {
		t.Errorf("match_participants doit avoir 2 rows, got %d", n)
	}
}

// ─── Test 4 : XUIDAliasesPreexisting_NoFail ────────────────────────────────

func TestSharedPersister_XUIDAliasesPreexisting_NoFail(t *testing.T) {
	db := openSharedTestDB(t)

	// Pré-seed xuid_aliases avec un xuid qui sera dans le batch.
	_, err := db.Exec(`
		INSERT INTO xuid_aliases (xuid, gamertag, last_seen)
		VALUES (?, ?, ?)
	`, "1111", "AliceOld", time.Now().Add(-24*time.Hour).UTC())
	if err != nil {
		t.Fatalf("pre-seed xuid_aliases: %v", err)
	}

	p := NewSharedPersister(db)
	batch := helperBuildSampleBatch("m_alias_001", "1111", "Alice")
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist avec xuid_aliases pre-existant: %v", err)
	}

	// La row originale doit toujours être présente (pas d'écrasement),
	// et le 2e xuid (9876543210) doit avoir été inséré.
	if n := countRows(t, db, "xuid_aliases", "xuid = ?", "1111"); n != 1 {
		t.Errorf("xuid 1111 doit avoir 1 row, got %d", n)
	}
	if n := countRows(t, db, "xuid_aliases", "xuid = ?", "9876543210"); n != 1 {
		t.Errorf("xuid 9876543210 (nouveau) doit avoir été inséré, got %d", n)
	}

	// Sanity : le gamertag du xuid pré-existant doit rester "AliceOld" (INSERT-only).
	var gt string
	if err := db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "1111").Scan(&gt); err != nil {
		t.Fatalf("query gamertag: %v", err)
	}
	if gt != "AliceOld" {
		t.Errorf("xuid_aliases doit rester INSERT-only, gamertag = %q, want AliceOld", gt)
	}
}

// ─── Test 5 : AtomicityOnFailure_RollsBackAll ──────────────────────────────

func TestSharedPersister_AtomicityOnFailure_RollsBackAll(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	// Construire un batch qui va échouer mid-transaction :
	// 2 participants AVEC LA MEME (match_id, xuid) → PK conflict sur le 2e INSERT.
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }
	batch := helperBuildSampleBatch("m_atomic_001", "1111", "Alice")
	batch.Shared.Participants = []domain.MatchParticipantRow{
		{MatchID: "m_atomic_001", XUID: "1111", Gamertag: strPtr("Alice"), Kills: intPtr(10)},
		{MatchID: "m_atomic_001", XUID: "1111", Gamertag: strPtr("Alice"), Kills: intPtr(20)}, // PK conflict
	}

	err := p.Persist(context.Background(), batch)
	if err == nil {
		t.Fatal("Persist devrait échouer sur PK conflict participants, mais a réussi")
	}

	// Atomicité : AUCUNE row ne doit avoir été committée.
	tables := []string{
		"match_registry",
		"match_participants",
		"medals_earned",
		"weapon_kills",
		"killer_victim_pairs",
		"highlight_events",
	}
	for _, table := range tables {
		if n := countRows(t, db, table, "match_id = ?", "m_atomic_001"); n != 0 {
			t.Errorf("atomicité cassée — %s a %d rows pour le batch foiré", table, n)
		}
	}
}

// ─── Test 6 : NilBatch_ReturnsError ────────────────────────────────────────

func TestSharedPersister_NilBatch_ReturnsError(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	if err := p.Persist(context.Background(), nil); err == nil {
		t.Error("Persist(nil) devrait retourner une erreur (defensive)")
	}
}
