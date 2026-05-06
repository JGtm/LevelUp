//go:build integration

// Package sync — writes_test.go : tests d'intégration des helpers d'écriture DuckDB.
package sync_test

import (
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	intsync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/testutil"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestInsertRegistryIfNotExists(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	mapName := "Recharge"
	plName := "Ranked Arena"
	row := intsync.MatchRegistryRow{
		MatchID:      "test-match-001",
		StartTime:    time.Now(),
		MapName:      &mapName,
		PlaylistName: &plName,
		FirstSyncBy:  "test-player",
	}
	if err := intsync.InsertRegistryIfNotExists(db, row); err != nil {
		t.Fatalf("InsertRegistryIfNotExists: %v", err)
	}

	// Vérifier l'insertion
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE match_id = ?", "test-match-001").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	// INSERT OR IGNORE → pas de doublon
	if err := intsync.InsertRegistryIfNotExists(db, row); err != nil {
		t.Fatalf("second insert should not fail: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE match_id = ?", "test-match-001").Scan(&count)
	if count != 1 {
		t.Errorf("expected still 1 row after duplicate insert, got %d", count)
	}
}

func TestUpsertXUIDAlias(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.UpsertXUIDAlias(db, "xuid-001", "PlayerOne"); err != nil {
		t.Fatalf("UpsertXUIDAlias: %v", err)
	}

	var gamertag string
	if err := db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "xuid-001").Scan(&gamertag); err != nil {
		t.Fatal(err)
	}
	if gamertag != "PlayerOne" {
		t.Errorf("expected PlayerOne, got %s", gamertag)
	}

	// Upsert avec un nouveau gamertag → mise à jour
	if err := intsync.UpsertXUIDAlias(db, "xuid-001", "PlayerOneRenamed"); err != nil {
		t.Fatalf("UpsertXUIDAlias update: %v", err)
	}
	_ = db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "xuid-001").Scan(&gamertag)
	if gamertag != "PlayerOneRenamed" {
		t.Errorf("expected PlayerOneRenamed after upsert, got %s", gamertag)
	}
}

func TestInsertMedals(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	rows := []intsync.MedalRow{
		{MatchID: "m1", XUID: "x1", MedalNameID: 100, Count: 3},
		{MatchID: "m1", XUID: "x1", MedalNameID: 200, Count: 1},
	}
	if err := intsync.InsertMedals(db, rows); err != nil {
		t.Fatalf("InsertMedals: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM medals_earned WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 medals, got %d", count)
	}
}

func TestSetSyncMeta(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.SetSyncMeta(db, "last_sync", "2025-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	var val string
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-01-01T00:00:00Z" {
		t.Errorf("expected 2025-01-01T00:00:00Z, got %s", val)
	}

	// Upsert
	_ = intsync.SetSyncMeta(db, "last_sync", "2025-06-01T00:00:00Z")
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-06-01T00:00:00Z" {
		t.Errorf("expected 2025-06-01T00:00:00Z after upsert, got %s", val)
	}
}

func TestInsertParticipants(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	teamMMR := 1500.0
	enemyMMR := 1480.0
	teamID0, teamID1 := 0, 1
	outcome2, outcome3 := 2, 3
	kills15, kills8 := 15, 8
	deaths10, deaths12 := 10, 12
	assists5, assists3 := 5, 3
	kda20, kda092 := 2.0, 0.92
	rows := []intsync.ParticipantRow{
		{
			MatchID:  "m1",
			XUID:     "x1",
			TeamID:   &teamID0,
			Outcome:  &outcome2,
			Kills:    &kills15,
			Deaths:   &deaths10,
			Assists:  &assists5,
			KDA:      &kda20,
			TeamMMR:  &teamMMR,
			EnemyMMR: &enemyMMR,
		},
		{
			MatchID: "m1",
			XUID:    "x2",
			TeamID:  &teamID1,
			Outcome: &outcome3,
			Kills:   &kills8,
			Deaths:  &deaths12,
			Assists: &assists3,
			KDA:     &kda092,
		},
	}
	if err := intsync.InsertParticipants(db, rows); err != nil {
		t.Fatalf("InsertParticipants: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 participants, got %d", count)
	}

	// Verify MMR propagated
	var mmr float64
	_ = db.QueryRow("SELECT team_mmr FROM match_participants WHERE xuid = 'x1'").Scan(&mmr)
	if mmr != 1500.0 {
		t.Errorf("expected team_mmr=1500.0, got %f", mmr)
	}

	// Idempotence: re-insert should not fail or duplicate
	if err := intsync.InsertParticipants(db, rows); err != nil {
		t.Fatalf("second InsertParticipants should not fail: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected still 2 participants after duplicate insert, got %d", count)
	}

	// Empty slice should no-op
	if err := intsync.InsertParticipants(db, nil); err != nil {
		t.Fatalf("InsertParticipants(nil) should not fail: %v", err)
	}
}

// TestInsertParticipants_UpsertFillsNullSkill vérifie qu'un re-sync avec des
// données skill (team_mmr/enemy_mmr/kills_expected) remplit les colonnes
// laissées à NULL au premier sync — c'est le mécanisme qui permet de combler
// les matchs où le skill endpoint avait initialement échoué.
func TestInsertParticipants_UpsertFillsNullSkill(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Premier sync : pas de skill data (tous les champs MMR/expected à NULL).
	teamID, outcome, kills, deaths := 0, 2, 15, 10
	first := []intsync.ParticipantRow{
		{
			MatchID: "m1", XUID: "x1",
			TeamID:  &teamID,
			Outcome: &outcome,
			Kills:   &kills,
			Deaths:  &deaths,
		},
	}
	if err := intsync.InsertParticipants(db, first); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	var teamMMR sql.NullFloat64
	_ = db.QueryRow("SELECT team_mmr FROM match_participants WHERE xuid = 'x1'").Scan(&teamMMR)
	if teamMMR.Valid {
		t.Fatalf("expected team_mmr NULL after first sync, got %v", teamMMR.Float64)
	}

	// Second sync : skill API a répondu cette fois. Les champs skill doivent
	// remplir les NULL existants.
	tm, em, ke := 1500.0, 1450.0, 12.5
	second := []intsync.ParticipantRow{
		{
			MatchID: "m1", XUID: "x1",
			TeamID:        &teamID,
			Outcome:       &outcome,
			Kills:         &kills,
			Deaths:        &deaths,
			TeamMMR:       &tm,
			EnemyMMR:      &em,
			KillsExpected: &ke,
		},
	}
	if err := intsync.InsertParticipants(db, second); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var got struct {
		TeamMMR       sql.NullFloat64
		EnemyMMR      sql.NullFloat64
		KillsExpected sql.NullFloat64
	}
	row := db.QueryRow("SELECT team_mmr, enemy_mmr, kills_expected FROM match_participants WHERE xuid = 'x1'")
	if err := row.Scan(&got.TeamMMR, &got.EnemyMMR, &got.KillsExpected); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !got.TeamMMR.Valid || got.TeamMMR.Float64 != tm {
		t.Errorf("team_mmr: want %.0f, got %v", tm, got.TeamMMR)
	}
	if !got.EnemyMMR.Valid || got.EnemyMMR.Float64 != em {
		t.Errorf("enemy_mmr: want %.0f, got %v", em, got.EnemyMMR)
	}
	if !got.KillsExpected.Valid || got.KillsExpected.Float64 != ke {
		t.Errorf("kills_expected: want %.1f, got %v", ke, got.KillsExpected)
	}

	// Pas de doublons.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = 'm1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row after upsert, got %d", count)
	}
}

// TestInsertParticipants_UpsertPreservesNonNull vérifie qu'un sync postérieur
// avec des champs nil ne détruit PAS les valeurs déjà persistées (COALESCE).
func TestInsertParticipants_UpsertPreservesNonNull(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Sync avec skill data complète.
	teamID, outcome, kills, deaths := 0, 2, 15, 10
	tm, em := 1500.0, 1450.0
	first := []intsync.ParticipantRow{
		{
			MatchID: "m1", XUID: "x1",
			TeamID: &teamID, Outcome: &outcome,
			Kills: &kills, Deaths: &deaths,
			TeamMMR: &tm, EnemyMMR: &em,
		},
	}
	if err := intsync.InsertParticipants(db, first); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Re-sync sans skill data (skill endpoint en panne) — les MMR doivent
	// être préservés.
	second := []intsync.ParticipantRow{
		{
			MatchID: "m1", XUID: "x1",
			TeamID: &teamID, Outcome: &outcome,
			Kills: &kills, Deaths: &deaths,
			// TeamMMR / EnemyMMR : nil
		},
	}
	if err := intsync.InsertParticipants(db, second); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var teamMMR, enemyMMR sql.NullFloat64
	_ = db.QueryRow("SELECT team_mmr, enemy_mmr FROM match_participants WHERE xuid = 'x1'").
		Scan(&teamMMR, &enemyMMR)
	if !teamMMR.Valid || teamMMR.Float64 != tm {
		t.Errorf("team_mmr should be preserved %.0f, got %v", tm, teamMMR)
	}
	if !enemyMMR.Valid || enemyMMR.Float64 != em {
		t.Errorf("enemy_mmr should be preserved %.0f, got %v", em, enemyMMR)
	}
}

func TestUpsertPlayerEnrichment(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Insert new enrichment
	if err := intsync.UpsertPlayerEnrichment(db, "m1", "sig-abc"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}

	var sig string
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-abc" {
		t.Errorf("expected sig-abc, got %s", sig)
	}

	// Upsert with new signature
	if err := intsync.UpsertPlayerEnrichment(db, "m1", "sig-def"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment update: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-def" {
		t.Errorf("expected sig-def after upsert, got %s", sig)
	}

	// Upsert with empty string → should preserve existing via COALESCE
	if err := intsync.UpsertPlayerEnrichment(db, "m1", ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment empty: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-def" {
		t.Errorf("expected sig-def preserved after empty upsert, got %s", sig)
	}
}

func TestInsertWeaponKills(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Seed a match for FK reference
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	wid1 := uint64(1234)
	wid2 := uint64(5678)
	rec := uint64(9999)
	delta := 150
	pidx := 3
	attrs := []intsync.WeaponKillRow{
		{TimeMS: 1000, WeaponID: &wid1, Confidence: "high", AttributionPath: "direct"},
		{TimeMS: 2000, WeaponID: &wid2, ReconciledAs: &rec, DeltaMS: &delta, Confidence: "medium", PlayerIndex: &pidx},
	}
	if err := intsync.InsertWeaponKills(db, "m1", "x1", attrs); err != nil {
		t.Fatalf("InsertWeaponKills: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 weapon_kills, got %d", count)
	}

	// Re-insert should replace (DELETE + INSERT)
	attrs2 := []intsync.WeaponKillRow{
		{TimeMS: 3000, WeaponID: &wid1, Confidence: "low", AttributionPath: "swap"},
	}
	if err := intsync.InsertWeaponKills(db, "m1", "x1", attrs2); err != nil {
		t.Fatalf("InsertWeaponKills replace: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 weapon_kill after replace, got %d", count)
	}
}

func TestMarkWeaponKillsDone(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	// Create minimal match_registry schema (same as openPveDB)
	_, err = db.Exec("CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, backfill_completed INTEGER DEFAULT 0)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Seed a match
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	if err := intsync.MarkWeaponKillsDone(db, "m1", false); err != nil {
		t.Fatalf("MarkWeaponKillsDone: %v", err)
	}

	var bits int
	_ = db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'").Scan(&bits)
	if bits&intsync.MBitWeaponKills == 0 {
		t.Errorf("expected MBitWeaponKills set, got bits=%d", bits)
	}

	// Mark no-film variant
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m2')")
	if err := intsync.MarkWeaponKillsDone(db, "m2", true); err != nil {
		t.Fatalf("MarkWeaponKillsDone noFilm: %v", err)
	}
	_ = db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm2'").Scan(&bits)
	if bits&intsync.MBitWeaponKillsNoFilm == 0 {
		t.Errorf("expected MBitWeaponKillsNoFilm set, got bits=%d", bits)
	}
}

func TestWriteSessionAssignments_UpdatesRows(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Seed deux matchs dans player_match_enrichment.
	_, err := db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('m1'), ('m2'), ('m3')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m2", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m3", SessionID: 2, SessionLabel: "Session 2"},
	}
	n, err := intsync.WriteSessionAssignments(db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 rows updated, got %d", n)
	}

	// Vérifier les valeurs écrites.
	var sid, slabel string
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sid, &slabel)
	if sid != "1" || slabel != "Session 1" {
		t.Errorf("m1: got session_id=%q session_label=%q, want 1 / 'Session 1'", sid, slabel)
	}
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm3'").Scan(&sid, &slabel)
	if sid != "2" || slabel != "Session 2" {
		t.Errorf("m3: got session_id=%q session_label=%q, want 2 / 'Session 2'", sid, slabel)
	}
}

func TestWriteSessionAssignments_EmptySlice(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)
	n, err := intsync.WriteSessionAssignments(db, nil)
	if err != nil {
		t.Fatalf("WriteSessionAssignments(nil): %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestWriteSessionAssignments_MissingMatchID_Zero(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Aucune ligne dans la table — l'UPDATE ne doit pas échouer, juste 0 lignes affectées.
	assignments := []domain.SessionAssignment{
		{MatchID: "nonexistent", SessionID: 1, SessionLabel: "S1"},
	}
	n, err := intsync.WriteSessionAssignments(db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 rows updated for missing match_id, got %d", n)
	}
}
