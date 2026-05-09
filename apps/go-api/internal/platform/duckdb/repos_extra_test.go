//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// ── CompareRepo ──────────────────────────────────────────────────────────────

func TestCompareRepo_ResolveXUID_Found(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuid001", Gamertag: "AlphaPlayer"}
	repo := NewCompareRepo(pdb)

	xuid, err := repo.ResolveXUID(context.Background(), "AlphaPlayer")
	if err != nil {
		t.Fatal(err)
	}
	if xuid != "xuid001" {
		t.Fatalf("expected xuid001, got %s", xuid)
	}
}

func TestCompareRepo_ResolveXUID_NotFound(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewCompareRepo(pdb)

	xuid, err := repo.ResolveXUID(context.Background(), "NoSuchPlayer")
	if err != nil {
		t.Fatal(err)
	}
	if xuid != "" {
		t.Fatalf("expected empty, got %s", xuid)
	}
}

func TestCompareRepo_ResolveXUID_CaseInsensitive(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewCompareRepo(pdb)

	xuid, err := repo.ResolveXUID(context.Background(), "alphaplayer")
	if err != nil {
		t.Fatal(err)
	}
	if xuid != "xuid001" {
		t.Fatalf("expected xuid001, got %s", xuid)
	}
}

func TestCompareRepo_GetLocalStats(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE IF NOT EXISTS shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			outcome INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
			damage_taken DOUBLE DEFAULT 0, max_killing_spree SMALLINT DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0, headshot_kills SMALLINT DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS shared.medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	inserts := []string{
		// match_id, xuid, gamertag, kills, deaths, assists, outcome, accuracy, damage_dealt,
		// damage_taken, max_killing_spree, avg_life_seconds, headshot_kills
		`INSERT INTO shared.match_participants VALUES
			('m1', 'x1', 'Player1', 20, 5, 10, 2, 55.0, 3000.0, 1500.0, 10, 60.0, 5),
			('m2', 'x1', 'Player1', 10, 10, 5, 3, 45.0, 2000.0, 1000.0, 5, 45.0, 3)`,
		// Médaille "Tir parfait" (medal_name_id=1512363953) : 2 en m1, 0 en m2.
		`INSERT INTO shared.medals_earned VALUES ('m1', 'x1', 1512363953, 2)`,
		`INSERT INTO shared.xuid_aliases VALUES ('x1', 'Player1')`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "x1"}
	repo := NewCompareRepo(pdb)

	stats, err := repo.GetLocalStats(ctx, "x1", "halo_infinite")
	if err != nil {
		t.Fatalf("GetLocalStats: %v", err)
	}
	if stats.Matches != 2 {
		t.Errorf("Matches: got %d, want 2", stats.Matches)
	}
	if stats.Gamertag != "Player1" {
		t.Errorf("Gamertag: got %q, want Player1", stats.Gamertag)
	}
	// 1 win sur 2 → 50 %
	if stats.WinRate != 0.5 {
		t.Errorf("WinRate: got %f, want 0.5", stats.WinRate)
	}
	if stats.KillsPerGame != 15.0 {
		t.Errorf("KillsPerGame: got %f, want 15.0", stats.KillsPerGame)
	}
	// accuracy stockée en % dans match_participants → normalisée /100 → 0..1
	if stats.Accuracy != 0.5 {
		t.Errorf("Accuracy: got %f, want 0.5 (normalized from 50%%)", stats.Accuracy)
	}
	// Phase 2 — nouvelles métriques.
	wantDamageTaken := 1250.0 // avg(1500, 1000)
	if stats.DamageTakenPerGame != wantDamageTaken {
		t.Errorf("DamageTakenPerGame: got %f, want %f", stats.DamageTakenPerGame, wantDamageTaken)
	}
	if stats.MaxKillingSpree != 10 {
		t.Errorf("MaxKillingSpree: got %d, want 10", stats.MaxKillingSpree)
	}
	wantLifeSecs := 52.5 // avg(60, 45)
	if stats.AvgLifeSecs != wantLifeSecs {
		t.Errorf("AvgLifeSecs: got %f, want %f", stats.AvgLifeSecs, wantLifeSecs)
	}
	wantHeadshots := 4.0 // avg(5, 3)
	if stats.HeadshotKillsPerGame != wantHeadshots {
		t.Errorf("HeadshotKillsPerGame: got %f, want %f", stats.HeadshotKillsPerGame, wantHeadshots)
	}
	// Tirs parfaits : 2 en m1, 0 en m2 → avg = 1.0
	if stats.PerfectKillsPerGame != 1.0 {
		t.Errorf("PerfectKillsPerGame: got %f, want 1.0", stats.PerfectKillsPerGame)
	}
}

func TestCompareRepo_GetLocalStats_NotFound(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE IF NOT EXISTS shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			outcome INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
			damage_taken DOUBLE DEFAULT 0, max_killing_spree SMALLINT DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0, headshot_kills SMALLINT DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS shared.medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewCompareRepo(pdb)

	_, err := repo.GetLocalStats(ctx, "xuid_unknown", "halo_infinite")
	if err == nil {
		t.Fatal("expected error for unknown XUID, got nil")
	}
}

func TestCompareRepo_GetPlayerATH(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	ddls := []string{
		`CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY, performance_score DOUBLE,
			session_id INTEGER, is_with_friends BOOLEAN, is_excluded BOOLEAN DEFAULT FALSE,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE match_skill_rank (
			match_id VARCHAR PRIMARY KEY, rating_type VARCHAR, rating_value DOUBLE,
			rating_deviation DOUBLE, tier VARCHAR, tier_fr VARCHAR, sub_tier SMALLINT,
			tier_label VARCHAR, rating_delta DOUBLE, playlist_group VARCHAR,
			start_time TIMESTAMPTZ, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`,
		`CREATE TABLE career_progression (
			rank INTEGER, current_xp INTEGER, recorded_at TIMESTAMPTZ,
			rank_name VARCHAR, rank_tier VARCHAR, xp_for_next_rank INTEGER,
			xp_total INTEGER, is_max_rank BOOLEAN DEFAULT FALSE)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL: %v\nSQL: %s", err, q)
		}
	}

	inserts := []string{
		`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m1', 72.5), ('m2', 55.0)`,
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, start_time)
			VALUES ('m1', 'CSR', 1650.0, '2024-01-02 00:00:00+00'),
			       ('m2', 'CSR', 1500.0, '2024-01-01 00:00:00+00'),
			       ('m3', 'LUSR', 1800.0, '2024-01-03 00:00:00+00')`,
		`INSERT INTO career_progression (rank, recorded_at) VALUES (42, '2024-01-01 00:00:00+00'), (55, '2024-02-01 00:00:00+00')`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("INSERT: %v\nSQL: %s", err, q)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewCompareRepo(pdb)

	ath, err := repo.GetPlayerATH(ctx)
	if err != nil {
		t.Fatalf("GetPlayerATH: %v", err)
	}
	// CSR le plus récent = m1 → 1650
	if ath.CSRCurrent != 1650 {
		t.Errorf("CSRCurrent: got %d, want 1650", ath.CSRCurrent)
	}
	// CareerRank le plus récent = 55 (enregistrement le plus récent)
	if ath.CareerRank != 55 {
		t.Errorf("CareerRank: got %d, want 55", ath.CareerRank)
	}
	// PerfATH = MAX(performance_score) = 72.5
	if ath.PerfATH != 72.5 {
		t.Errorf("PerfATH: got %f, want 72.5", ath.PerfATH)
	}
	// LusrATH = MAX(rating_value WHERE rating_type='LUSR') = 1800
	if ath.LusrATH != 1800.0 {
		t.Errorf("LusrATH: got %f, want 1800.0", ath.LusrATH)
	}
}

func TestCompareRepo_GetFavoriteWeapon(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.weapon_kills (
			match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT,
			reconciled_as UBIGINT, attribution_path VARCHAR, player_index INTEGER)`,
		`CREATE VIEW shared.v_weapon_kills AS
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
			FROM shared.weapon_kills`,
		// weapon_labels dans le même DB (Metadata = db dans le test).
		`CREATE TABLE weapon_labels (
			weapon_id UBIGINT PRIMARY KEY, name_fr VARCHAR, name_en VARCHAR)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL: %v\nSQL: %s", err, q)
		}
	}

	inserts := []string{
		// x1 a tué 3 fois avec weapon 111 et 1 fois avec weapon 222.
		`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES
			('m1', 'x1', 111), ('m2', 'x1', 111), ('m3', 'x1', 111),
			('m4', 'x1', 222)`,
		// Label pour weapon 111.
		`INSERT INTO weapon_labels VALUES (111, 'Fusil à plasma', 'Plasma Rifle')`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("INSERT: %v\nSQL: %s", err, q)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db, Metadata: db, XUID: "x1"}
	repo := NewCompareRepo(pdb)

	wh, err := repo.GetFavoriteWeapon(ctx, "x1")
	if err != nil {
		t.Fatalf("GetFavoriteWeapon: %v", err)
	}
	if wh == nil {
		t.Fatal("expected WeaponHighlight, got nil")
	}
	if wh.Kills != 3 {
		t.Errorf("Kills: got %d, want 3", wh.Kills)
	}
	if wh.LabelFR != "Fusil à plasma" {
		t.Errorf("LabelFR: got %q, want %q", wh.LabelFR, "Fusil à plasma")
	}
	if wh.LabelEN != "Plasma Rifle" {
		t.Errorf("LabelEN: got %q, want %q", wh.LabelEN, "Plasma Rifle")
	}
}

func TestCompareRepo_GetFavoriteWeapon_NoData(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	// Pas de table weapon_kills du tout — best-effort doit retourner nil sans erreur.
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "x1"}
	repo := NewCompareRepo(pdb)

	wh, err := repo.GetFavoriteWeapon(ctx, "x1")
	if err != nil {
		t.Fatalf("expected nil error (best-effort), got %v", err)
	}
	if wh != nil {
		t.Errorf("expected nil WeaponHighlight for empty DB, got %+v", wh)
	}
}

// ── FanoutRepo ───────────────────────────────────────────────────────────────

func TestFanoutRepo_CountCommonMatches_Empty(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	count, err := repo.CountCommonMatchesForXUID(context.Background(), "xuid001", nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestFanoutRepo_CountCommonMatches(t *testing.T) {
	db := openMemDB(t)
	seedShared(t, db)

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	count, err := repo.CountCommonMatchesForXUID(context.Background(), "xuid001", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatal(err)
	}
	// xuid001 is in m1 and m2
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestFanoutRepo_LoadExistingEnrichments_Empty(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	result, err := repo.LoadExistingEnrichments(ctx, []string{"m1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestFanoutRepo_LoadExistingEnrichments(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO player_match_enrichment VALUES ('m1'), ('m3')`); err != nil {
		t.Fatal(err)
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewFanoutRepo(pdb)

	result, err := repo.LoadExistingEnrichments(ctx, []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatal(err)
	}
	if !result["m1"] || result["m2"] || !result["m3"] {
		t.Fatalf("unexpected result: %v", result)
	}
}

// ── MatchExclusionRepo ──────────────────────────────────────────────────────

func TestMatchExclusionRepo_ListExcluded_Empty(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		// start_time TIMESTAMP + start_time_utc TIMESTAMPTZ : pattern canonique
		// requis par MatchExclusionRepo.ListExcluded.
		`CREATE TABLE shared.match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, map_name VARCHAR, pair_name VARCHAR)`,
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, is_excluded BOOLEAN, updated_at TIMESTAMPTZ)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	result, err := repo.ListExcluded(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestMatchExclusionRepo_ListExcluded(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, map_name VARCHAR, pair_name VARCHAR)`,
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, is_excluded BOOLEAN, updated_at TIMESTAMPTZ)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	inserts := []string{
		`INSERT INTO shared.match_registry VALUES ('m1', '2025-01-10 14:00:00', '2025-01-10 14:00:00+00', 'Arena', 'Slayer')`,
		`INSERT INTO player_match_enrichment VALUES ('m1', TRUE, '2025-06-01 00:00:00+00')`,
		`INSERT INTO player_match_enrichment VALUES ('m2', FALSE, '2025-06-01 00:00:00+00')`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewMatchExclusionRepo(pdb)

	result, err := repo.ListExcluded(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 excluded, got %d", len(result))
	}
	if result[0].MatchID != "m1" {
		t.Fatalf("expected m1, got %s", result[0].MatchID)
	}
}
