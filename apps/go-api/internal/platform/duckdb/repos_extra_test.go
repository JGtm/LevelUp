//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/migration"
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

	// ajout des vues root-level pour aligner sur le
	// contrat SharedReader (queries sans préfixe `shared.`).
	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE IF NOT EXISTS shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			outcome INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
			damage_taken DOUBLE DEFAULT 0, max_killing_spree SMALLINT DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0, headshot_kills SMALLINT DEFAULT 0,
			kda DOUBLE, team_id INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS shared.medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE IF NOT EXISTS shared.killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER DEFAULT 1
		)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		// Vues root-level pour les queries via SharedReader.
		`CREATE VIEW match_participants AS SELECT * FROM shared.match_participants`,
		`CREATE VIEW medals_earned AS SELECT * FROM shared.medals_earned`,
		`CREATE VIEW xuid_aliases AS SELECT * FROM shared.xuid_aliases`,
		`CREATE VIEW killer_victim_pairs AS SELECT * FROM shared.killer_victim_pairs`,
		`CREATE VIEW v_gamertag_lookup AS SELECT * FROM shared.v_gamertag_lookup`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	inserts := []string{
		`INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, kills, deaths, assists, outcome, accuracy,
			 damage_dealt, damage_taken, max_killing_spree, avg_life_seconds, headshot_kills)
			VALUES
			('m1', 'x1', 'Player1', 20, 5, 10, 2, 55.0, 3000.0, 1500.0, 10, 60.0, 5),
			('m2', 'x1', 'Player1', 10, 10, 5, 3, 45.0, 2000.0, 1000.0, 5, 45.0, 3)`,
		// Médaille "Tir parfait" (medal_name_id=1512363953) : 2 en m1, 0 en m2.
		`INSERT INTO shared.medals_earned VALUES ('m1', 'x1', 1512363953, 2)`,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES ('x1', 'Player1')`,
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

	// ajout des vues root-level pour aligner sur le
	// contrat SharedReader (queries sans préfixe `shared.`).
	ddls := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE IF NOT EXISTS shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			outcome INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
			damage_taken DOUBLE DEFAULT 0, max_killing_spree SMALLINT DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0, headshot_kills SMALLINT DEFAULT 0,
			kda DOUBLE, team_id INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS shared.medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE IF NOT EXISTS shared.killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER DEFAULT 1
		)`,
		`CREATE VIEW shared.v_gamertag_lookup AS SELECT xuid, gamertag FROM shared.xuid_aliases`,
		// Vues root-level pour les queries via SharedReader.
		`CREATE VIEW match_participants AS SELECT * FROM shared.match_participants`,
		`CREATE VIEW medals_earned AS SELECT * FROM shared.medals_earned`,
		`CREATE VIEW xuid_aliases AS SELECT * FROM shared.xuid_aliases`,
		`CREATE VIEW killer_victim_pairs AS SELECT * FROM shared.killer_victim_pairs`,
		`CREATE VIEW v_gamertag_lookup AS SELECT * FROM shared.v_gamertag_lookup`,
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
		// Vue _latest (ADR 0026) : le reader CompareRepo.GetPlayerATH lit
		// match_skill_rank_latest, jamais la table brute. Sur ce fixture
		// mono-version, un simple passthrough suffit.
		`CREATE VIEW match_skill_rank_latest AS SELECT * FROM match_skill_rank`,
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
	// Append-only #23046 : convertit player_match_enrichment (id PK + stage +
	// written_at) et crée la vue player_match_enrichment_latest (lue par le repo).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
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

// TestCompareRepo_GetEncounterStats_WithData : seed 2 joueurs sur 2 matchs
// (1 ally, 1 enemy) + killer_victim_pairs croisés. Vérifie les agrégats.
func TestCompareRepo_GetEncounterStats_WithData(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	// Sprint follow-up B1 Phase 1 (9g.1) : test pour GetEncounterStats.
	// Naming root-level + tables shared minimales.
	for _, ddl := range []string{
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER)`,
		`CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER)`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("DDL: %v", err)
		}
	}

	// m1 : xuidA+xuidB ally (team 0), main win
	// m2 : xuidA+xuidB enemy (team 0 vs team 1), main loss
	// kv : xuidA tué xuidB 5× ; xuidB tué xuidA 3×
	for _, ins := range []string{
		`INSERT INTO match_participants VALUES
			('m1', 'xuidA', 0, 2), ('m1', 'xuidB', 0, 2),
			('m2', 'xuidA', 0, 3), ('m2', 'xuidB', 1, 2)`,
		`INSERT INTO killer_victim_pairs VALUES
			('m1', 'xuidA', 'xuidB', 5),
			('m2', 'xuidB', 'xuidA', 3)`,
	} {
		if _, err := db.Exec(ctx, ins); err != nil {
			t.Fatalf("INSERT: %v", err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidA"}
	repo := NewCompareRepo(pdb)

	enc, err := repo.GetEncounterStats(ctx, "xuidA", "xuidB")
	if err != nil {
		t.Fatalf("GetEncounterStats: %v", err)
	}
	if enc == nil {
		t.Fatal("expected non-nil EncounterStats")
	}
	if enc.TotalEncounters != 2 {
		t.Errorf("TotalEncounters: got %d, want 2", enc.TotalEncounters)
	}
	if enc.AllyCount != 1 {
		t.Errorf("AllyCount: got %d, want 1", enc.AllyCount)
	}
	if enc.EnemyCount != 1 {
		t.Errorf("EnemyCount: got %d, want 1", enc.EnemyCount)
	}
	if enc.KillsDealt != 5 {
		t.Errorf("KillsDealt: got %d, want 5", enc.KillsDealt)
	}
	if enc.DeathsSuffered != 3 {
		t.Errorf("DeathsSuffered: got %d, want 3", enc.DeathsSuffered)
	}
}

// TestCompareRepo_GetEncounterStats_NoCommonMatches : 0 match commun → nil best-effort.
func TestCompareRepo_GetEncounterStats_NoCommonMatches(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE match_participants (
		match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `CREATE TABLE killer_victim_pairs (
		match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER)`); err != nil {
		t.Fatal(err)
	}

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidA"}
	repo := NewCompareRepo(pdb)

	enc, err := repo.GetEncounterStats(ctx, "xuidA", "xuidUnknown")
	if err != nil {
		t.Fatalf("GetEncounterStats: %v", err)
	}
	if enc != nil {
		t.Errorf("expected nil for no common matches, got %+v", enc)
	}
}

// TestCompareRepo_GetCrossMatchSample_WithData : agrégats limités aux matchs
// communs xuidA + xuidB.
func TestCompareRepo_GetCrossMatchSample_WithData(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, team_id INTEGER, max_killing_spree INTEGER,
			avg_life_seconds DOUBLE, headshot_kills INTEGER)`,
		`CREATE TABLE medals_earned (
			match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("DDL: %v", err)
		}
	}
	// m1 + m2 : both xuidA + xuidB. m3 : xuidA seul (exclu).
	for _, ins := range []string{
		`INSERT INTO match_participants VALUES
			('m1', 'xuidA', 0, 5, 30.0, 3),
			('m1', 'xuidB', 0, 8, 45.0, 5),
			('m2', 'xuidA', 0, 4, 25.0, 2),
			('m2', 'xuidB', 0, 6, 50.0, 4),
			('m3', 'xuidA', 0, 3, 20.0, 1)`,
		`INSERT INTO medals_earned VALUES
			('m1', 'xuidB', 1512363953, 2),
			('m2', 'xuidB', 1512363953, 4)`,
	} {
		if _, err := db.Exec(ctx, ins); err != nil {
			t.Fatalf("INSERT: %v", err)
		}
	}

	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidA"}
	repo := NewCompareRepo(pdb)

	sample, err := repo.GetCrossMatchSample(ctx, "xuidA", "xuidB")
	if err != nil {
		t.Fatalf("GetCrossMatchSample: %v", err)
	}
	if sample == nil {
		t.Fatal("expected non-nil sample")
	}
	// 2 matchs communs (m1, m2 — m3 exclu car xuidB absent)
	if sample.MatchesCount != 2 {
		t.Errorf("MatchesCount: got %d, want 2", sample.MatchesCount)
	}
	// MAX(xuidB.max_killing_spree) = MAX(8, 6) = 8
	if sample.MaxKillingSpree != 8 {
		t.Errorf("MaxKillingSpree: got %d, want 8", sample.MaxKillingSpree)
	}
	// AVG(xuidB.avg_life_seconds) = (45+50)/2 = 47.5
	if sample.AvgLifeSecs < 47.4 || sample.AvgLifeSecs > 47.6 {
		t.Errorf("AvgLifeSecs: got %f, want ~47.5", sample.AvgLifeSecs)
	}
	// AVG(xuidB.headshot_kills) = (5+4)/2 = 4.5
	if sample.HeadshotKillsPerGame < 4.4 || sample.HeadshotKillsPerGame > 4.6 {
		t.Errorf("HeadshotKillsPerGame: got %f, want ~4.5", sample.HeadshotKillsPerGame)
	}
	// AVG(perfect_count) = (2+4)/2 = 3.0
	if sample.PerfectKillsPerGame < 2.9 || sample.PerfectKillsPerGame > 3.1 {
		t.Errorf("PerfectKillsPerGame: got %f, want ~3.0", sample.PerfectKillsPerGame)
	}
}

// TestCompareRepo_GetPlayerATHFor_NotInPool : pool vide → nil sans erreur.
func TestCompareRepo_GetPlayerATHFor_NotInPool(t *testing.T) {
	CloseAll() // isole le globalPool
	t.Cleanup(CloseAll)

	db := openMemDB(t)
	pdb := &PlayerDB{Player: db, Shared: db}
	repo := NewCompareRepo(pdb)

	ath, err := repo.GetPlayerATHFor(context.Background(), "AbsentPlayer", "halo_infinite")
	if err != nil {
		t.Fatalf("GetPlayerATHFor not-in-pool: %v", err)
	}
	if ath != nil {
		t.Errorf("expected nil for absent player, got %+v", ath)
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
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
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
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO player_match_enrichment (match_id) VALUES ('m1'), ('m3')`); err != nil {
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
		// Vue root-level pour le pipeline split (P7-4) : match_registry sans préfixe.
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, is_excluded BOOLEAN, updated_at TIMESTAMPTZ)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
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
		// Vue root-level pour le pipeline split (P7-4) : match_registry sans préfixe.
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, is_excluded BOOLEAN, updated_at TIMESTAMPTZ)`,
	}
	for _, q := range ddls {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	inserts := []string{
		`INSERT INTO shared.match_registry VALUES ('m1', '2025-01-10 14:00:00', '2025-01-10 14:00:00+00', 'Arena', 'Slayer')`,
		`INSERT INTO player_match_enrichment (match_id, is_excluded, updated_at) VALUES ('m1', TRUE, '2025-06-01 00:00:00+00')`,
		`INSERT INTO player_match_enrichment (match_id, is_excluded, updated_at) VALUES ('m2', FALSE, '2025-06-01 00:00:00+00')`,
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
