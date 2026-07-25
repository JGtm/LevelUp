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
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
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
	// create_base_shared_schema (+ match_registry/match_participants/weapon_kills,
	// shared_add_participation_timestamps) sont title-owned depuis la relocation du
	// merge h5 : sans provider, RunForDB n'applique que les 3 migrations globales
	// restantes et weapon_kills n'existe pas. Poser le provider (cf. openPVETestDB).
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
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
	// Patch match_participants : first_joined_time / last_leave_time sont
	// ajoutés par shared_add_participation_timestamps (title-owned, halo_infinite/migrations/steps.go).
	// Le title steps provider n'est pas disponible ici → patch manuel.
	for _, col := range []string{"first_joined_time TIMESTAMPTZ", "last_leave_time TIMESTAMPTZ"} {
		if _, err := db.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_participants add %s: %v", col, err)
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

// ─── Test 7 : MatchIntensity + BackfillCompleted persistés ─────────────────

func TestSharedPersister_MatchIntensityAndBackfillCompleted_Persisted(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	intensity := 12.5
	bf := int64(0xFF00) // bitmask cumul events+killer_victim+pve+weapon_kills

	batch := helperBuildSampleBatch("m_bits_001", "1111", "Alice")
	batch.Shared.Match.MatchIntensity = &intensity
	batch.Shared.Match.BackfillCompleted = &bf

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var gotIntensity sql.NullFloat64
	var gotBF sql.NullInt64
	err := db.QueryRow(
		"SELECT match_intensity, backfill_completed FROM match_registry WHERE match_id = ?",
		"m_bits_001",
	).Scan(&gotIntensity, &gotBF)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !gotIntensity.Valid || gotIntensity.Float64 != intensity {
		t.Errorf("match_intensity = %+v, want %f", gotIntensity, intensity)
	}
	if !gotBF.Valid || gotBF.Int64 != bf {
		t.Errorf("backfill_completed = %+v, want %d", gotBF, bf)
	}
}

// ─── Test 8 : BackfillBits sur match_participants ──────────────────────────

func TestSharedPersister_ParticipantBackfillBits_Persisted(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	bf := 0x1FF // tous les bits stats remplis
	batch := helperBuildSampleBatch("m_pbits_001", "1111", "Alice")
	batch.Shared.Participants = []domain.MatchParticipantRow{
		{
			MatchID: "m_pbits_001", XUID: "1111", Gamertag: strPtr("Alice"),
			Kills: intPtr(10), Deaths: intPtr(5),
			BackfillBits: &bf,
		},
	}

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var got sql.NullInt64
	err := db.QueryRow(
		"SELECT backfill_bits FROM match_participants WHERE match_id = ? AND xuid = ?",
		"m_pbits_001", "1111",
	).Scan(&got)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !got.Valid || got.Int64 != int64(bf) {
		t.Errorf("backfill_bits = %+v, want %d", got, bf)
	}
}

// ─── Test 9 : MatchCSRs insert (lobby CSR context) ─────────────────────────

func TestSharedPersister_MatchCSRs_InsertsAllParticipantCSRs(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	float64Ptr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	batch := helperBuildSampleBatch("m_csr_001", "1111", "Alice")
	batch.Shared.MatchCSRs = []MatchCSRInsert{
		{
			MatchID: "m_csr_001", XUID: "1111", RatingType: "CSR",
			RatingValue: float64Ptr(1450), Tier: strPtr("Onyx"), SubTier: intPtr(0),
			TierLabel: strPtr("Onyx"), RatingDelta: float64Ptr(+18),
			SeasonID: strPtr("CsrSeason13-1"),
		},
		{
			MatchID: "m_csr_001", XUID: "9876543210", RatingType: "CSR",
			RatingValue: float64Ptr(1500), Tier: strPtr("Onyx"), SubTier: intPtr(0),
			TierLabel: strPtr("Onyx"), RatingDelta: float64Ptr(-15),
			SeasonID: strPtr("CsrSeason13-1"),
		},
	}

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs WHERE match_id = ?`, "m_csr_001").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("match_csrs : %d rows, want 2", n)
	}

	// Sanity sur la row du joueur sync
	var rating sql.NullFloat64
	var tier sql.NullString
	err := db.QueryRow(
		"SELECT rating_value, tier FROM match_csrs WHERE match_id = ? AND xuid = ?",
		"m_csr_001", "1111",
	).Scan(&rating, &tier)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !rating.Valid || rating.Float64 != 1450 {
		t.Errorf("rating_value = %+v, want 1450", rating)
	}
	if !tier.Valid || tier.String != "Onyx" {
		t.Errorf("tier = %+v, want Onyx", tier)
	}
}

// ─── Test 10 : MatchCSRs default RatingType = "CSR" ────────────────────────

func TestSharedPersister_MatchCSRs_DefaultRatingTypeCSR(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	float64Ptr := func(v float64) *float64 { return &v }
	batch := helperBuildSampleBatch("m_csr_dflt", "1111", "Alice")
	batch.Shared.MatchCSRs = []MatchCSRInsert{
		// RatingType laissé vide → doit défaulter à "CSR"
		{MatchID: "m_csr_dflt", XUID: "1111", RatingValue: float64Ptr(1200)},
	}

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var rt string
	if err := db.QueryRow(`SELECT rating_type FROM match_csrs WHERE match_id = ? AND xuid = ?`,
		"m_csr_dflt", "1111").Scan(&rt); err != nil {
		t.Fatal(err)
	}
	if rt != "CSR" {
		t.Errorf("rating_type = %q, want CSR (default)", rt)
	}
}

// ─── Test 11 : KillerVictim forme par-kill (gamertags + time_ms) ────────────
//
// Régression D3-1 : le chemin batch (prod par défaut) écrivait la forme dégradée
// (kill_count seul), laissant killer_gamertag/victim_gamertag/time_ms NULL — or
// le match-view (Q20KVPairs) les lit. On vérifie que la forme par-kill complète
// est persistée, à parité avec la complétion legacy.
func TestSharedPersister_KillerVictim_PerKillFormPersisted(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	batch := helperBuildSampleBatch("m_kv_001", "1111", "Alice")
	batch.Shared.KillerVictim = []KillerVictimInsert{
		{
			MatchID:    "m_kv_001",
			KillerXUID: "1111", KillerGamertag: "Alice",
			VictimXUID: "9876543210", VictimGamertag: "FriendA",
			Count: 1, TimeMS: 42000,
		},
	}

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var killerGT, victimGT sql.NullString
	var timeMS, killCount sql.NullInt64
	err := db.QueryRow(`
		SELECT killer_gamertag, victim_gamertag, time_ms, kill_count
		FROM killer_victim_pairs WHERE match_id = ?`, "m_kv_001",
	).Scan(&killerGT, &victimGT, &timeMS, &killCount)
	if err != nil {
		t.Fatalf("query killer_victim_pairs: %v", err)
	}
	if !killerGT.Valid || killerGT.String != "Alice" {
		t.Errorf("killer_gamertag = %+v, want Alice", killerGT)
	}
	if !victimGT.Valid || victimGT.String != "FriendA" {
		t.Errorf("victim_gamertag = %+v, want FriendA", victimGT)
	}
	if !timeMS.Valid || timeMS.Int64 != 42000 {
		t.Errorf("time_ms = %+v, want 42000", timeMS)
	}
	if !killCount.Valid || killCount.Int64 != 1 {
		t.Errorf("kill_count = %+v, want 1", killCount)
	}
}

// ─── Test golden : chemin batch complet + INSERT-only + idempotence ────────
//
// FIGE le chemin batch (prod par défaut) de bout en bout : un batch combat
// complet est persisté avec la forme par-kill (gamertags + time_ms) ET
// events_loaded=TRUE, puis un re-Persist du MÊME match est idempotent (aucun
// doublon, aucune mutation — INSERT-only gardé par le pré-check match_registry).
// Couvre les régressions D3-1 (KV dégradé) + D3-2 (events_loaded) + l'invariant
// INSERT-only/idempotence en un seul test caractérisant le comportement correct.
func TestSharedPersister_GoldenBatchPath_CompleteAndIdempotent(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)
	strPtr := func(s string) *string { return &s }

	batch := helperBuildSampleBatch("m_golden", "1111", "Alice")
	// Forme combat par-kill complète (2 kills) + highlight_events correspondants.
	batch.Shared.KillerVictim = []KillerVictimInsert{
		{MatchID: "m_golden", KillerXUID: "1111", KillerGamertag: "Alice",
			VictimXUID: "9876543210", VictimGamertag: "FriendA", Count: 1, TimeMS: 12000},
		{MatchID: "m_golden", KillerXUID: "1111", KillerGamertag: "Alice",
			VictimXUID: "9876543210", VictimGamertag: "FriendA", Count: 1, TimeMS: 34000},
	}
	batch.Shared.HighlightEvents = []HighlightEventInsert{
		{MatchID: "m_golden", XUID: strPtr("1111"), EventType: "Kill", TimeMS: 12000},
		{MatchID: "m_golden", XUID: strPtr("1111"), EventType: "Kill", TimeMS: 34000},
	}

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// (1) Forme combat par-kill : 2 rows, gamertags + time_ms NON NULL partout.
	rows, err := db.Query(`
		SELECT killer_gamertag, victim_gamertag, time_ms
		FROM killer_victim_pairs WHERE match_id = ? ORDER BY time_ms`, "m_golden")
	if err != nil {
		t.Fatalf("query KV: %v", err)
	}
	defer rows.Close()
	var gotTimes []int64
	for rows.Next() {
		var kgt, vgt sql.NullString
		var tms sql.NullInt64
		if err := rows.Scan(&kgt, &vgt, &tms); err != nil {
			t.Fatalf("scan KV: %v", err)
		}
		if !kgt.Valid || kgt.String == "" || !vgt.Valid || vgt.String == "" {
			t.Errorf("forme dégradée : gamertag NULL/vide (killer=%v victim=%v)", kgt, vgt)
		}
		if !tms.Valid {
			t.Errorf("forme dégradée : time_ms NULL")
		} else {
			gotTimes = append(gotTimes, tms.Int64)
		}
	}
	if len(gotTimes) != 2 || gotTimes[0] != 12000 || gotTimes[1] != 34000 {
		t.Errorf("time_ms par-kill = %v, want [12000 34000]", gotTimes)
	}

	// (2) events_loaded = TRUE.
	var evl bool
	if err := db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`,
		"m_golden").Scan(&evl); err != nil {
		t.Fatalf("query events_loaded: %v", err)
	}
	if !evl {
		t.Error("events_loaded doit être TRUE (batch avec highlight_events)")
	}

	// (3) Re-Persist du même match = idempotent (no-op via pré-check registry) :
	//     aucun doublon, aucune mutation.
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("re-Persist (idempotent attendu): %v", err)
	}
	for _, c := range []struct {
		table string
		want  int
	}{
		{"match_registry", 1},
		{"match_participants", 2},
		{"medals_earned", 2},
		{"killer_victim_pairs", 2},
		{"highlight_events", 2},
	} {
		if got := countRows(t, db, c.table, "match_id = ?", "m_golden"); got != c.want {
			t.Errorf("idempotence cassée — %s = %d rows après re-Persist, want %d", c.table, got, c.want)
		}
	}
}

// ─── Test 12 : events_loaded dérivé de la présence de highlight_events ──────
//
// Régression D3-2 : le chemin batch posait MBitEvents mais laissait
// events_loaded=FALSE → matchs éternellement re-candidats au backfill events
// (heals décommissionnés). On vérifie que events_loaded passe TRUE à l'INSERT
// quand le batch porte des highlight_events, et reste FALSE sinon.
func TestSharedPersister_EventsLoaded_DerivedFromHighlightEvents(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	// helperBuildSampleBatch ajoute 1 highlight event → events_loaded == TRUE.
	withEvents := helperBuildSampleBatch("m_evl_001", "1111", "Alice")
	if err := p.Persist(context.Background(), withEvents); err != nil {
		t.Fatalf("Persist (avec events): %v", err)
	}
	var evl bool
	if err := db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`,
		"m_evl_001").Scan(&evl); err != nil {
		t.Fatalf("query events_loaded: %v", err)
	}
	if !evl {
		t.Error("events_loaded doit être TRUE quand le batch porte des highlight_events")
	}

	// Batch SANS highlight_events → events_loaded reste FALSE (film pas prêt).
	noEvents := helperBuildSampleBatch("m_evl_002", "1111", "Alice")
	noEvents.Shared.HighlightEvents = nil
	if err := p.Persist(context.Background(), noEvents); err != nil {
		t.Fatalf("Persist (sans events): %v", err)
	}
	var evl2 bool
	if err := db.QueryRow(`SELECT events_loaded FROM match_registry WHERE match_id = ?`,
		"m_evl_002").Scan(&evl2); err != nil {
		t.Fatalf("query events_loaded 2: %v", err)
	}
	if evl2 {
		t.Error("events_loaded doit rester FALSE sans highlight_events")
	}
}

// ─── Test : weapon_kills.kill_kind — capture + vue + non-regression NULL ────────
//
// Couvre 3 exigences de la capture kill_kind (Halo 5, Phase 1) :
//   - migration : la colonne kill_kind existe ET remonte via la vue v_weapon_kills
//     (piege DuckDB `SELECT * EXCLUDE(rk)` : la vue est recreee par la migration) ;
//   - persist round-trip : un WeaponKillInsert.KillKind non vide est relu via la vue ;
//   - non-regression : un weapon_kill SANS kill_kind (chemin Infinite/film) => NULL.
func TestSharedPersister_WeaponKillKind_RoundTripAndNull(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	u64Ptr := func(v uint64) *uint64 { return &v }
	builder := NewBatchBuilder("halo_5", "Alice", "1111", "test")
	builder.SetMatch(&domain.MatchRegistryRow{
		MatchID:      "m_kk_001",
		StartTime:    time.Now().UTC(),
		ModeCategory: "PVP",
		FirstSyncBy:  "Alice",
	})
	builder.AddWeaponKills([]WeaponKillInsert{
		// H5 : mecanique capturee.
		{MatchID: "m_kk_001", XUID: "1111", TimeMS: 5000, WeaponID: u64Ptr(111),
			Confidence: "native", AttributionPath: "h5_native", KillKind: "melee"},
		// Infinite/film : aucune mecanique → doit rester NULL en base.
		{MatchID: "m_kk_001", XUID: "1111", TimeMS: 6000, WeaponID: u64Ptr(222),
			Confidence: "high", AttributionPath: "primary"},
	})
	if err := p.Persist(context.Background(), builder.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// Lecture via la VUE (prouve que la migration a recree v_weapon_kills en exposant
	// kill_kind — sans recreation, `SELECT *` figerait l'ancien jeu de colonnes).
	readKind := func(timeMS int) sql.NullString {
		var kk sql.NullString
		if err := db.QueryRow(
			`SELECT kill_kind FROM v_weapon_kills WHERE match_id = ? AND time_ms = ?`,
			"m_kk_001", timeMS,
		).Scan(&kk); err != nil {
			t.Fatalf("read kill_kind via vue (time_ms=%d): %v", timeMS, err)
		}
		return kk
	}

	if kk := readKind(5000); !kk.Valid || kk.String != "melee" {
		t.Errorf("kill_kind capture: got valid=%v %q, attendu \"melee\"", kk.Valid, kk.String)
	}
	if kk := readKind(6000); kk.Valid {
		t.Errorf("kill_kind Infinite: attendu NULL, got %q", kk.String)
	}
}

// TestPersistWeaponKillsNewGeneration_SupersedesWithKillKind — chemin du backfill kill_kind :
// PersistWeaponKillsNewGeneration ré-insère TOUS les kills d'un couple en NOUVELLE
// génération (INSERT-only). Prouve : ancienne génération intacte en table physique, la vue
// v_weapon_kills renvoie la nouvelle génération COMPLÈTE avec kill_kind, aucune perte.
func TestPersistWeaponKillsNewGeneration_SupersedesWithKillKind(t *testing.T) {
	db := openSharedTestDB(t)
	ctx := context.Background()
	u64Ptr := func(v uint64) *uint64 { return &v }

	// Génération legacy (kill_kind NULL) : 2 kills pour (mX, xA) via le persister normal.
	p := NewSharedPersister(db)
	b := NewBatchBuilder("halo_5", "Alice", "xA", "test")
	b.SetMatch(&domain.MatchRegistryRow{MatchID: "mX", StartTime: time.Now().UTC(), ModeCategory: "PVP", FirstSyncBy: "Alice"})
	b.AddWeaponKills([]WeaponKillInsert{
		{MatchID: "mX", XUID: "xA", TimeMS: 1000, WeaponID: u64Ptr(100), Confidence: "native", AttributionPath: "h5_native"},
		{MatchID: "mX", XUID: "xA", TimeMS: 2000, WeaponID: u64Ptr(100), Confidence: "native", AttributionPath: "h5_native"},
	})
	if err := p.Persist(ctx, b.Build()); err != nil {
		t.Fatalf("seed persist: %v", err)
	}

	// Backfill : ré-insertion COMPLÈTE du couple (3 kills désormais avec kill_kind) en
	// nouvelle génération (nextval strictement > gen legacy).
	if err := PersistWeaponKillsNewGeneration(ctx, db, []WeaponKillInsert{
		{MatchID: "mX", XUID: "xA", TimeMS: 1000, WeaponID: u64Ptr(100), Confidence: "native", AttributionPath: "h5_native", KillKind: "weapon"},
		{MatchID: "mX", XUID: "xA", TimeMS: 2000, WeaponID: u64Ptr(100), Confidence: "native", AttributionPath: "h5_native", KillKind: "melee"},
		{MatchID: "mX", XUID: "xA", TimeMS: 3000, WeaponID: u64Ptr(200), Confidence: "native", AttributionPath: "h5_native", KillKind: "weapon"},
	}); err != nil {
		t.Fatalf("PersistWeaponKillsNewGeneration: %v", err)
	}

	// Table physique : append pur → 2 (legacy) + 3 (backfill) = 5, rien supprimé.
	var phys int
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_kills WHERE match_id='mX'`).Scan(&phys); err != nil {
		t.Fatalf("count physique: %v", err)
	}
	if phys != 5 {
		t.Errorf("weapon_kills physique = %d, attendu 5 (2 legacy intacts + 3 backfill)", phys)
	}
	// Vue : uniquement la nouvelle génération (3 kills), tous avec kill_kind.
	var viewRows, kkNull int
	if err := db.QueryRow(`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='mX'`).Scan(&viewRows); err != nil {
		t.Fatalf("count vue: %v", err)
	}
	if viewRows != 3 {
		t.Errorf("v_weapon_kills = %d, attendu 3 (nouvelle génération supersède)", viewRows)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='mX' AND kill_kind IS NULL`).Scan(&kkNull); err != nil {
		t.Fatalf("count kill_kind NULL: %v", err)
	}
	if kkNull != 0 {
		t.Errorf("v_weapon_kills kill_kind NULL = %d, attendu 0 (génération legacy supersédée)", kkNull)
	}

	// Empty rows = no-op (aucune génération allouée, aucune erreur).
	if err := PersistWeaponKillsNewGeneration(ctx, db, nil); err != nil {
		t.Fatalf("PersistWeaponKillsNewGeneration(nil): %v", err)
	}
}

// ─── Test : ObjectiveStats INSERT-only + lecture via _latest + colonnes NULL ────
//
// FIGE le chemin objectif (V72-03) de bout en bout : un batch avec des rows CTF est
// persisté dans match_objective_stats (INSERT pur), relu via la vue _latest, et les
// colonnes d'un autre mode (zone_captures) restent NULL. Re-Persist du même match =
// no-op (idempotence via le pré-check match_registry, INSERT-only préservé).
func TestSharedPersister_ObjectiveStats_InsertsReadViaLatestAndNullOtherMode(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)
	intPtr := func(v int) *int { return &v }
	f := func(v float64) *float64 { return &v }

	batch := helperBuildSampleBatch("m_obj_001", "1111", "Alice")
	batch.Shared.ObjectiveStats = []ObjectiveStatsInsert{
		{
			MatchID: "m_obj_001", XUID: "1111",
			FlagCaptures: intPtr(1), FlagGrabs: intPtr(9), FlagSecures: intPtr(7),
			FlagSteals: intPtr(3), FlagReturns: intPtr(2), FlagCarriersKilled: intPtr(1),
			TimeAsFlagCarrierSeconds: f(140.0),
		},
		{
			MatchID: "m_obj_001", XUID: "9876543210",
			FlagReturns: intPtr(3), FlagSteals: intPtr(3), TimeAsFlagCarrierSeconds: f(18.8),
		},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	if n := countRows(t, db, "match_objective_stats", "match_id = ?", "m_obj_001"); n != 2 {
		t.Errorf("match_objective_stats = %d rows, want 2", n)
	}

	// Lecture via la vue _latest : valeurs CTF correctes + colonne d'un autre mode NULL.
	var flagCaptures, flagGrabs, zoneCaptures sql.NullInt64
	var timeCarrier sql.NullFloat64
	err := db.QueryRow(`
		SELECT flag_captures, flag_grabs, time_as_flag_carrier_seconds, zone_captures
		FROM match_objective_stats_latest WHERE match_id = ? AND xuid = ?`,
		"m_obj_001", "1111").Scan(&flagCaptures, &flagGrabs, &timeCarrier, &zoneCaptures)
	if err != nil {
		t.Fatalf("query _latest: %v", err)
	}
	if !flagCaptures.Valid || flagCaptures.Int64 != 1 {
		t.Errorf("flag_captures = %+v, want 1", flagCaptures)
	}
	if !flagGrabs.Valid || flagGrabs.Int64 != 9 {
		t.Errorf("flag_grabs = %+v, want 9", flagGrabs)
	}
	if !timeCarrier.Valid || timeCarrier.Float64 != 140.0 {
		t.Errorf("time_as_flag_carrier_seconds = %+v, want 140", timeCarrier)
	}
	if zoneCaptures.Valid {
		t.Errorf("zone_captures doit être NULL (mode CTF), got %d", zoneCaptures.Int64)
	}

	// Re-Persist = no-op idempotent (aucun doublon).
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("re-Persist (idempotent attendu): %v", err)
	}
	if n := countRows(t, db, "match_objective_stats", "match_id = ?", "m_obj_001"); n != 2 {
		t.Errorf("idempotence cassée — match_objective_stats = %d rows après re-Persist, want 2", n)
	}
}

// ─── Test : ObjectiveStats Stockpile + Extraction + VIP (V721-02) ─────────────
//
// Verrouille les 18 colonnes ajoutées par shared_objective_stats_add_stockpile_extraction
// de bout en bout : INSERT pur via SharedPersister → relecture par la vue _latest (donc
// vue bien RECRÉÉE après l'ALTER) → colonnes des autres modes NULL. L'INSERT de
// persistObjectiveStats aligne 43 colonnes / 43 placeholders / 43 arguments : un décalage
// d'un cran écrirait la valeur dans la mauvaise colonne — ce test l'attrape (les valeurs
// choisies sont toutes distinctes).
func TestSharedPersister_ObjectiveStats_StockpileAndExtraction(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)
	intPtr := func(v int) *int { return &v }
	f := func(v float64) *float64 { return &v }

	batch := helperBuildSampleBatch("m_obj_sp", "1111", "Alice")
	batch.Shared.ObjectiveStats = []ObjectiveStatsInsert{
		{
			MatchID: "m_obj_sp", XUID: "1111",
			KillsAsPowerSeedCarrier: intPtr(2), PowerSeedCarriersKilled: intPtr(1),
			PowerSeedsDeposited: intPtr(6), PowerSeedsStolen: intPtr(3),
			TimeAsPowerSeedCarrierSeconds: f(59.1), TimeAsPowerSeedDriverSeconds: f(64.2),
		},
		{
			MatchID: "m_obj_sp", XUID: "9876543210",
			ExtractionConversionsCompleted: intPtr(11), ExtractionConversionsDenied: intPtr(12),
			ExtractionInitiationsCompleted: intPtr(13), ExtractionInitiationsDenied: intPtr(14),
			SuccessfulExtractions: intPtr(15),
		},
		{
			MatchID: "m_obj_sp", XUID: "5555",
			KillsAsVip: intPtr(21), VipKills: intPtr(22), VipAssists: intPtr(23),
			TimesSelectedAsVip: intPtr(24), MaxKillingSpreeAsVip: intPtr(25),
			TimeAsVipSeconds: f(109.5), LongestTimeAsVipSeconds: f(48),
		},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// Ligne Stockpile : 6 valeurs distinctes + colonnes CTF/Extraction NULL.
	var killsCarrier, carriersKilled, deposited, stolen sql.NullInt64
	var carrierSecs, driverSecs sql.NullFloat64
	var flagCaptures, successfulExtractions sql.NullInt64
	err := db.QueryRow(`
		SELECT kills_as_power_seed_carrier, power_seed_carriers_killed, power_seeds_deposited,
		       power_seeds_stolen, time_as_power_seed_carrier_seconds, time_as_power_seed_driver_seconds,
		       flag_captures, successful_extractions
		FROM match_objective_stats_latest WHERE match_id = ? AND xuid = ?`,
		"m_obj_sp", "1111").Scan(&killsCarrier, &carriersKilled, &deposited, &stolen,
		&carrierSecs, &driverSecs, &flagCaptures, &successfulExtractions)
	if err != nil {
		t.Fatalf("query _latest (stockpile): %v", err)
	}
	for _, c := range []struct {
		got  sql.NullInt64
		want int64
		name string
	}{
		{killsCarrier, 2, "kills_as_power_seed_carrier"},
		{carriersKilled, 1, "power_seed_carriers_killed"},
		{deposited, 6, "power_seeds_deposited"},
		{stolen, 3, "power_seeds_stolen"},
	} {
		if !c.got.Valid || c.got.Int64 != c.want {
			t.Errorf("%s = %+v, want %d", c.name, c.got, c.want)
		}
	}
	if !carrierSecs.Valid || carrierSecs.Float64 != 59.1 {
		t.Errorf("time_as_power_seed_carrier_seconds = %+v, want 59.1", carrierSecs)
	}
	if !driverSecs.Valid || driverSecs.Float64 != 64.2 {
		t.Errorf("time_as_power_seed_driver_seconds = %+v, want 64.2", driverSecs)
	}
	if flagCaptures.Valid {
		t.Errorf("flag_captures doit être NULL (mode Stockpile), got %d", flagCaptures.Int64)
	}
	if successfulExtractions.Valid {
		t.Errorf("successful_extractions doit être NULL (mode Stockpile), got %d", successfulExtractions.Int64)
	}

	// Ligne Extraction : 5 compteurs distincts + toutes les durées NULL.
	var convDone, convDenied, initDone, initDenied, extractions sql.NullInt64
	var spCarrierSecs sql.NullFloat64
	err = db.QueryRow(`
		SELECT extraction_conversions_completed, extraction_conversions_denied,
		       extraction_initiations_completed, extraction_initiations_denied,
		       successful_extractions, time_as_power_seed_carrier_seconds
		FROM match_objective_stats_latest WHERE match_id = ? AND xuid = ?`,
		"m_obj_sp", "9876543210").Scan(&convDone, &convDenied, &initDone, &initDenied,
		&extractions, &spCarrierSecs)
	if err != nil {
		t.Fatalf("query _latest (extraction): %v", err)
	}
	for _, c := range []struct {
		got  sql.NullInt64
		want int64
		name string
	}{
		{convDone, 11, "extraction_conversions_completed"},
		{convDenied, 12, "extraction_conversions_denied"},
		{initDone, 13, "extraction_initiations_completed"},
		{initDenied, 14, "extraction_initiations_denied"},
		{extractions, 15, "successful_extractions"},
	} {
		if !c.got.Valid || c.got.Int64 != c.want {
			t.Errorf("%s = %+v, want %d", c.name, c.got, c.want)
		}
	}
	if spCarrierSecs.Valid {
		t.Errorf("time_as_power_seed_carrier_seconds doit être NULL (mode Extraction), got %v", spCarrierSecs.Float64)
	}

	// Ligne VIP : 5 compteurs + 2 durées, toutes distinctes ; colonnes des autres modes NULL.
	var killsAsVip, vipKills, vipAssists, timesSelected, maxSpree sql.NullInt64
	var vipSecs, longestVipSecs sql.NullFloat64
	var vipFlagCaptures sql.NullInt64
	err = db.QueryRow(`
		SELECT kills_as_vip, vip_kills, vip_assists, times_selected_as_vip,
		       max_killing_spree_as_vip, time_as_vip_seconds, longest_time_as_vip_seconds,
		       flag_captures
		FROM match_objective_stats_latest WHERE match_id = ? AND xuid = ?`,
		"m_obj_sp", "5555").Scan(&killsAsVip, &vipKills, &vipAssists, &timesSelected,
		&maxSpree, &vipSecs, &longestVipSecs, &vipFlagCaptures)
	if err != nil {
		t.Fatalf("query _latest (vip): %v", err)
	}
	for _, c := range []struct {
		got  sql.NullInt64
		want int64
		name string
	}{
		{killsAsVip, 21, "kills_as_vip"},
		{vipKills, 22, "vip_kills"},
		{vipAssists, 23, "vip_assists"},
		{timesSelected, 24, "times_selected_as_vip"},
		{maxSpree, 25, "max_killing_spree_as_vip"},
	} {
		if !c.got.Valid || c.got.Int64 != c.want {
			t.Errorf("%s = %+v, want %d", c.name, c.got, c.want)
		}
	}
	if !vipSecs.Valid || vipSecs.Float64 != 109.5 {
		t.Errorf("time_as_vip_seconds = %+v, want 109.5", vipSecs)
	}
	if !longestVipSecs.Valid || longestVipSecs.Float64 != 48 {
		t.Errorf("longest_time_as_vip_seconds = %+v, want 48", longestVipSecs)
	}
	if vipFlagCaptures.Valid {
		t.Errorf("flag_captures doit être NULL (mode VIP), got %d", vipFlagCaptures.Int64)
	}
}
