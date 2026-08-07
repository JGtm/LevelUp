//go:build integration

package sync

// sync_pipeline_fixture_test.go — fixture complète simulant un pipeline de sync
// du début à la fin : highlights → weapon_kills → perf_score → LUSR →
// sessions → citations → engagement_score.
//
// Ce fichier est la source de vérité pour la non-régression de l'ensemble
// des calculs locaux. Si une formule change, les assertions ici doivent être
// mises à jour.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// ── constantes de la fixture ─────────────────────────────────────────────────

const (
	fixXUID           = "xuid_main_player"
	fixGamertag       = "MainPlayer"
	fixFriendXUID     = "xuid_friend_1"
	fixFriendGamertag = "FriendPlayer1"
	fixEnemy1XUID     = "xuid_enemy_1"
	fixEnemy2XUID     = "xuid_enemy_2"
	fixStrangerXUID   = "xuid_stranger_1"

	// 3 matchs de référence
	fixM1 = "match-fixture-001" // ranked, film présent (10 kills + médailles)
	fixM2 = "match-fixture-002" // social, pas de film, 40 min après m1 → même session
	fixM3 = "match-fixture-003" // social, pas de film, 7 jours après → session séparée

	// Médailles de test
	fixMedalBulltrue   = int64(12345)
	fixMedalTripleKill = int64(23456)
)

// ── pipelineFixture ──────────────────────────────────────────────────────────

type pipelineFixture struct {
	shared   *sql.DB
	player   *sql.DB
	metadata *sql.DB
}

// buildPipelineFixture construit 3 DB in-memory avec toutes les tables et données
// nécessaires pour exécuter l'ensemble du pipeline de calcul local.
func buildPipelineFixture(t *testing.T) *pipelineFixture {
	t.Helper()

	shared := openFixtureDB(t, buildSharedDDL())
	player := openFixtureDB(t, buildPlayerDDL())
	// Append-only #23046 : convertit player_match_enrichment + crée la vue _latest.
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(player); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	// Append-only #23046 (Phase 2) : convertit match_citations (generation_id) +
	// crée la vue match_citations_latest + les séquences.
	if err := migration.EnsureMatchCitationsAppendOnly(player); err != nil {
		t.Fatalf("EnsureMatchCitationsAppendOnly: %v", err)
	}
	metadata := openFixtureDB(t, buildMetadataDDL())

	f := &pipelineFixture{shared: shared, player: player, metadata: metadata}
	f.insertSharedData(t)
	f.insertPlayerData(t)
	f.insertMetadataData(t)
	return f
}

func openFixtureDB(t *testing.T, ddl string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatalf("execScript DDL: %v", err)
	}
	return db
}

// ── DDL ──────────────────────────────────────────────────────────────────────

func buildSharedDDL() string {
	return `
CREATE SEQUENCE highlight_events_id_seq;

CREATE TABLE match_registry (
    match_id            VARCHAR PRIMARY KEY,
    start_time          TIMESTAMP NOT NULL,
    end_time            TIMESTAMP,
    start_time_utc      TIMESTAMPTZ,
    end_time_utc        TIMESTAMPTZ,
    playlist_name       VARCHAR,
    pair_name           VARCHAR,
    game_variant_name   VARCHAR,
    mode_category       VARCHAR,
    is_ranked           BOOLEAN DEFAULT FALSE,
    is_firefight        BOOLEAN DEFAULT FALSE,
    duration_seconds    INTEGER,
    backfill_completed  INTEGER DEFAULT 0
);

CREATE TABLE match_participants (
    match_id            VARCHAR NOT NULL,
    xuid                VARCHAR NOT NULL,
    gamertag            VARCHAR,
    team_id             INTEGER,
    outcome             INTEGER,
    rank                INTEGER,
    score               INTEGER DEFAULT 0,
    kills               INTEGER DEFAULT 0,
    deaths              INTEGER DEFAULT 0,
    assists             INTEGER DEFAULT 0,
    kda                 DOUBLE DEFAULT 0,
    accuracy            DOUBLE DEFAULT 0,
    personal_score      INTEGER DEFAULT 0,
    time_played_seconds INTEGER DEFAULT 600,
    avg_life_seconds    DOUBLE DEFAULT 0,
    damage_dealt        DOUBLE DEFAULT 0,
    damage_taken        DOUBLE DEFAULT 0,
    headshot_kills      INTEGER DEFAULT 0,
    melee_kills         INTEGER DEFAULT 0,
    power_weapon_kills  INTEGER DEFAULT 0,
    grenade_kills       INTEGER DEFAULT 0,
    max_killing_spree   INTEGER DEFAULT 0,
    kills_expected      DOUBLE,
    deaths_expected     DOUBLE,
    team_mmr            DOUBLE,
    enemy_mmr           DOUBLE,
    present_at_beginning  BOOLEAN,
    present_at_completion BOOLEAN,
    joined_in_progress    BOOLEAN,
    left_in_progress      BOOLEAN,
    first_joined_time     TIMESTAMPTZ,
    last_leave_time       TIMESTAMPTZ,
    PRIMARY KEY (match_id, xuid)
);

CREATE TABLE medals_earned (
    match_id        VARCHAR NOT NULL,
    xuid            VARCHAR NOT NULL,
    medal_name_id   BIGINT NOT NULL,
    count           INTEGER,
    PRIMARY KEY (match_id, xuid, medal_name_id)
);

CREATE TABLE xuid_aliases (
    xuid        VARCHAR PRIMARY KEY,
    gamertag    VARCHAR,
    last_seen   TIMESTAMP
);

CREATE TABLE highlight_events (
    id          INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
    match_id    VARCHAR NOT NULL,
    event_type  VARCHAR NOT NULL,
    time_ms     INTEGER,
    xuid        VARCHAR,
    type_hint   INTEGER,
    raw_json    VARCHAR
);

CREATE TABLE killer_victim_pairs (
    match_id        VARCHAR NOT NULL,
    killer_xuid     VARCHAR NOT NULL,
    killer_gamertag VARCHAR,
    victim_xuid     VARCHAR NOT NULL,
    victim_gamertag VARCHAR,
    kill_count      INTEGER DEFAULT 1,
    time_ms         INTEGER
);

CREATE SEQUENCE IF NOT EXISTS weapon_kills_generation_seq START 1;
CREATE TABLE weapon_kills (
    match_id        VARCHAR NOT NULL,
    xuid            VARCHAR NOT NULL,
    time_ms         INTEGER NOT NULL,
    weapon_id       UBIGINT,
    reconciled_as   UBIGINT,
    delta_ms        INTEGER,
    confidence      VARCHAR DEFAULT 'none',
    attribution_path VARCHAR DEFAULT 'none',
    swap_detected   BOOLEAN DEFAULT FALSE,
    delayed_damage  BOOLEAN DEFAULT FALSE,
    player_index    INTEGER,
    generation_id   BIGINT DEFAULT 0
);

CREATE VIEW v_weapon_kills AS
SELECT * EXCLUDE (rk) FROM (
    SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
           DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
    FROM weapon_kills)
WHERE rk = 1;

CREATE SEQUENCE match_objective_stats_id_seq;
CREATE TABLE match_objective_stats (
    id                                    BIGINT PRIMARY KEY DEFAULT nextval('match_objective_stats_id_seq'),
    match_id                              VARCHAR NOT NULL,
    xuid                                  VARCHAR NOT NULL,
    flag_captures                         INTEGER,
    flag_capture_assists                  INTEGER,
    flag_grabs                            INTEGER,
    flag_secures                          INTEGER,
    flag_steals                           INTEGER,
    flag_returns                          INTEGER,
    flag_carriers_killed                  INTEGER,
    flag_returners_killed                 INTEGER,
    kills_as_flag_carrier                 INTEGER,
    kills_as_flag_returner                INTEGER,
    time_as_flag_carrier_seconds          DOUBLE,
    zone_captures                         INTEGER,
    zone_secures                          INTEGER,
    zone_offensive_kills                  INTEGER,
    zone_defensive_kills                  INTEGER,
    zone_scoring_ticks                    INTEGER,
    time_in_zones_seconds                 DOUBLE,
    kills_as_skull_carrier                INTEGER,
    skull_carriers_killed                 INTEGER,
    skull_grabs                           INTEGER,
    skull_scoring_ticks                   INTEGER,
    time_as_skull_carrier_seconds         DOUBLE,
    longest_time_as_skull_carrier_seconds DOUBLE,
    kills_as_power_seed_carrier           INTEGER,
    power_seed_carriers_killed            INTEGER,
    power_seeds_deposited                 INTEGER,
    power_seeds_stolen                    INTEGER,
    time_as_power_seed_carrier_seconds    DOUBLE,
    time_as_power_seed_driver_seconds     DOUBLE,
    extraction_conversions_completed      INTEGER,
    extraction_conversions_denied         INTEGER,
    extraction_initiations_completed      INTEGER,
    extraction_initiations_denied         INTEGER,
    successful_extractions                INTEGER,
    kills_as_vip                          INTEGER,
    vip_kills                             INTEGER,
    vip_assists                           INTEGER,
    times_selected_as_vip                 INTEGER,
    max_killing_spree_as_vip              INTEGER,
    time_as_vip_seconds                   DOUBLE,
    longest_time_as_vip_seconds           DOUBLE,
    written_at                            TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
` + migration.MatchObjectiveStatsLatestViewSQL("match_objective_stats") + `;
`
}

func buildPlayerDDL() string {
	return `
CREATE TABLE player_match_enrichment (
    match_id                    VARCHAR PRIMARY KEY,
    performance_score           DOUBLE,
    performance_chain           VARCHAR,
    session_id                  VARCHAR,
    session_label               VARCHAR,
    is_with_friends             BOOLEAN DEFAULT FALSE,
    teammates_signature         VARCHAR,
    engagement_score            DOUBLE,
    engagement_score_brut       DOUBLE,
    engagement_score_confidence VARCHAR,
    mode_category               VARCHAR,
    created_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);

CREATE SEQUENCE personal_score_awards_id_seq;
CREATE TABLE personal_score_awards (
    id              INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id        VARCHAR NOT NULL,
    xuid            VARCHAR NOT NULL,
    award_name      VARCHAR NOT NULL,
    award_category  VARCHAR,
    award_count     INTEGER DEFAULT 1,
    award_score     INTEGER DEFAULT 0
);

CREATE TABLE match_citations (
    match_id            VARCHAR NOT NULL,
    citation_name_norm  VARCHAR NOT NULL,
    value               INTEGER DEFAULT 1,
    created_at          TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    PRIMARY KEY (match_id, citation_name_norm)
);

CREATE TABLE match_skill_rank (
    match_id         VARCHAR PRIMARY KEY,
    rating_type      VARCHAR NOT NULL,
    rating_value     DOUBLE,
    rating_deviation DOUBLE,
    tier             VARCHAR,
    tier_fr          VARCHAR,
    sub_tier         INTEGER DEFAULT 0,
    tier_label       VARCHAR,
    rating_delta     DOUBLE,
    playlist_group   VARCHAR,
    expected_win_prob FLOAT,
    start_time       TIMESTAMP,
    created_at       TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at       TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);

CREATE TABLE sync_meta (
    key         VARCHAR PRIMARY KEY,
    value       VARCHAR,
    updated_at  TIMESTAMP
);
`
}

func buildMetadataDDL() string {
	return `
CREATE TABLE citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,
    citation_name_display VARCHAR NOT NULL,
    mapping_type          VARCHAR DEFAULT 'medal',
    category              VARCHAR,
    image_path            VARCHAR,
    description           VARCHAR,
    tier_targets          VARCHAR,
    medal_id              UBIGINT,
    medal_ids             VARCHAR,
    stat_name             VARCHAR,
    award_name            VARCHAR,
    custom_function       VARCHAR,
    composite_children    VARCHAR,
    enabled               BOOLEAN DEFAULT TRUE
);

CREATE TABLE weapon_labels (
    weapon_id  UBIGINT PRIMARY KEY,
    name_en    VARCHAR,
    name_fr    VARCHAR
);
`
}

// ── insertion des données ─────────────────────────────────────────────────────

func (f *pipelineFixture) insertSharedData(t *testing.T) {
	t.Helper()
	db := f.shared

	// Timestamps de référence (UTC explicite)
	m1Start := time.Date(2026, 1, 10, 20, 0, 0, 0, time.UTC)
	m2Start := time.Date(2026, 1, 10, 20, 40, 0, 0, time.UTC) // +40min → même session
	m3Start := time.Date(2026, 1, 17, 19, 0, 0, 0, time.UTC)  // +7j → session séparée

	type matchSpec struct {
		id     string
		start  time.Time
		ranked bool
		bits   int
	}
	matches := []matchSpec{
		{fixM1, m1Start, true, MBitEvents},
		{fixM2, m2Start, false, MBitEvents},
		{fixM3, m3Start, false, MBitEvents},
	}

	// match_registry
	for _, m := range matches {
		end := m.start.Add(600 * time.Second)
		mustExec(t, db, `
			INSERT INTO match_registry
				(match_id, start_time, end_time, start_time_utc, end_time_utc,
				 playlist_name, pair_name, is_ranked, duration_seconds, backfill_completed)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.id, m.start, end,
			m.start.Format(time.RFC3339), end.Format(time.RFC3339),
			"Ranked Arena", "Aquarius - Slayer",
			m.ranked, 600, m.bits,
		)
	}

	// Seed extra matches pour atteindre MinMatchesForRelative (10) pour perf_score.
	// Ces matchs sont joués avant m1, tous solo (pas de coéquipiers communs).
	seedSharedMatches(t, db, 12)

	// match_participants — m1 : ranked 4-joueurs (2v2)
	type partSpec struct {
		matchID  string
		xuid     string
		gamertag string
		team     int
		outcome  int
		kills    int
		deaths   int
		assists  int
		score    int
	}
	participants := []partSpec{
		{fixM1, fixXUID, fixGamertag, 0, 2, 10, 2, 3, 2000},
		{fixM1, fixFriendXUID, fixFriendGamertag, 0, 2, 7, 3, 4, 1500},
		{fixM1, fixEnemy1XUID, "Enemy1", 1, 3, 4, 6, 1, 800},
		{fixM1, fixEnemy2XUID, "Enemy2", 1, 3, 3, 7, 2, 700},

		{fixM2, fixXUID, fixGamertag, 0, 2, 8, 3, 2, 1800},
		{fixM2, fixFriendXUID, fixFriendGamertag, 0, 2, 6, 4, 3, 1300},
		{fixM2, fixEnemy1XUID, "Enemy1", 1, 3, 5, 5, 1, 900},
		{fixM2, fixEnemy2XUID, "Enemy2", 1, 3, 4, 6, 0, 750},

		{fixM3, fixXUID, fixGamertag, 0, 3, 5, 6, 1, 1200},
		{fixM3, fixStrangerXUID, "Stranger1", 0, 3, 4, 5, 2, 1100},
	}
	for _, p := range participants {
		mustExec(t, db, `
			INSERT INTO match_participants
				(match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
				 personal_score, time_played_seconds, kda, accuracy,
				 kills_expected, deaths_expected, team_mmr, enemy_mmr)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.matchID, p.xuid, p.gamertag, p.team, p.outcome,
			p.kills, p.deaths, p.assists, p.score, 600,
			float64(p.kills+p.assists/2)/float64(max1(p.deaths)), 0.45,
			float64(p.kills)*0.9, float64(p.deaths)*1.0, 1500.0, 1500.0,
		)
	}

	// medals_earned — m1 seulement
	mustExec(t, db, `INSERT INTO medals_earned VALUES (?, ?, ?, ?)`, fixM1, fixXUID, fixMedalBulltrue, 2)
	mustExec(t, db, `INSERT INTO medals_earned VALUES (?, ?, ?, ?)`, fixM1, fixXUID, fixMedalTripleKill, 1)

	// xuid_aliases
	type alias struct{ xuid, gt string }
	for _, a := range []alias{
		{fixXUID, fixGamertag},
		{fixFriendXUID, fixFriendGamertag},
		{fixEnemy1XUID, "Enemy1"},
		{fixEnemy2XUID, "Enemy2"},
		{fixStrangerXUID, "Stranger1"},
	} {
		mustExec(t, db, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`, a.xuid, a.gt)
	}

	// highlight_events — kill events pour chaque match
	// m1 : 10 kills du joueur principal
	for i := 0; i < 10; i++ {
		mustExec(t, db, `
			INSERT INTO highlight_events (match_id, event_type, time_ms, xuid)
			VALUES (?, 'kill', ?, ?)`,
			fixM1, (i+1)*30000, fixXUID,
		)
	}
	// m1 : quelques morts et médailles
	for i := 0; i < 2; i++ {
		mustExec(t, db, `INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'death', ?, ?)`,
			fixM1, (i+1)*120000, fixXUID)
	}
	// m2 : 8 kills (pour que batchComputeEngagementScores traite m2)
	for i := 0; i < 8; i++ {
		mustExec(t, db, `INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
			fixM2, (i+1)*35000, fixXUID)
	}
	// m3 : 5 kills
	for i := 0; i < 5; i++ {
		mustExec(t, db, `INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
			fixM3, (i+1)*40000, fixXUID)
	}

	// killer_victim_pairs — m1
	for i := 0; i < 5; i++ {
		mustExec(t, db, `
			INSERT INTO killer_victim_pairs
				(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fixM1, fixXUID, fixGamertag, fixEnemy1XUID, "Enemy1", 1, (i+1)*60000,
		)
		mustExec(t, db, `
			INSERT INTO killer_victim_pairs
				(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fixM1, fixXUID, fixGamertag, fixEnemy2XUID, "Enemy2", 1, (i+1)*60000+5000,
		)
	}
}

func (f *pipelineFixture) insertPlayerData(t *testing.T) {
	t.Helper()
	for _, mid := range []string{fixM1, fixM2, fixM3} {
		mustExec(t, f.player,
			`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid)
	}
	// Pré-peupler engagement_score_brut pour les 12 matchs seed (mode PvP_unranked).
	// Cela permet à loadHistoryForCategory d'avoir ≥HistoryMinPartial=10 entrées
	// quand les matchs m2/m3 (aussi PvP_unranked) sont traités.
	for i := 0; i < 12; i++ {
		mid := fmt.Sprintf("seed-match-%04d", i)
		brut := float64(i)*0.1 - 0.6 // valeurs synthétiques dans [-0.6, 0.5]
		mustExec(t, f.player, `
			INSERT INTO player_match_enrichment (match_id, mode_category, engagement_score_brut)
			VALUES (?, 'PvP_unranked', ?)`, mid, brut)
	}
}

func (f *pipelineFixture) insertMetadataData(t *testing.T) {
	t.Helper()
	mustExec(t, f.metadata, `
		INSERT INTO citation_mappings
			(citation_name_norm, citation_name_display, mapping_type, medal_id, enabled)
		VALUES (?, ?, ?, ?, ?)`,
		"bulltrue", "Bulltrue", "medal", uint64(fixMedalBulltrue), true,
	)
	mustExec(t, f.metadata, `
		INSERT INTO citation_mappings
			(citation_name_norm, citation_name_display, mapping_type, medal_id, enabled)
		VALUES (?, ?, ?, ?, ?)`,
		"triple_kill", "Triple Kill", "medal", uint64(fixMedalTripleKill), true,
	)
}

// seedSharedMatches insère n matchs sociaux fictifs dans match_registry +
// match_participants pour fixXUID. Utilisés pour dépasser MinMatchesForRelative.
func seedSharedMatches(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	base := time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("seed-match-%04d", i)
		start := base.Add(time.Duration(i) * 2 * time.Hour)
		end := start.Add(600 * time.Second)
		mustExec(t, db, `
			INSERT INTO match_registry
				(match_id, start_time, end_time, is_ranked, duration_seconds, backfill_completed)
			VALUES (?, ?, ?, FALSE, 600, 0)`, mid, start, end)
		mustExec(t, db, `
			INSERT INTO match_participants
				(match_id, xuid, team_id, outcome, kills, deaths, assists,
				 personal_score, time_played_seconds, kda, kills_expected, deaths_expected, team_mmr, enemy_mmr)
			VALUES (?, ?, 0, 2, ?, 4, 2, 1200, 600, ?, 8.0, 4.0, 1450.0, 1550.0)`,
			mid, fixXUID, 8+i%5,
			float64(8+i%5+1)/4.0,
		)
	}
}

// mustExec exécute une instruction SQL et fait échouer le test en cas d'erreur.
func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("mustExec %s: %v", query[:min60(query)], err)
	}
}

func min60(s string) int {
	if len(s) < 60 {
		return len(s)
	}
	return 60
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestPipelineFixture_SharedDataInserted vérifie que les données de base
// sont correctement insérées dans la shared DB.
func TestPipelineFixture_SharedDataInserted(t *testing.T) {
	f := buildPipelineFixture(t)

	var nMatches int
	f.shared.QueryRow("SELECT COUNT(*) FROM match_registry").Scan(&nMatches)
	if nMatches < 3 {
		t.Fatalf("expected ≥3 matches in registry, got %d", nMatches)
	}

	var nParticipants int
	f.shared.QueryRow(
		"SELECT COUNT(*) FROM match_participants WHERE xuid=?", fixXUID,
	).Scan(&nParticipants)
	if nParticipants < 3 {
		t.Fatalf("expected ≥3 participant rows for fixXUID, got %d", nParticipants)
	}

	var nMedals int
	f.shared.QueryRow(
		"SELECT COUNT(*) FROM medals_earned WHERE match_id=? AND xuid=?", fixM1, fixXUID,
	).Scan(&nMedals)
	if nMedals != 2 {
		t.Fatalf("expected 2 medal rows for m1/fixXUID, got %d", nMedals)
	}

	var nEvents int
	f.shared.QueryRow(
		"SELECT COUNT(*) FROM highlight_events WHERE match_id=? AND xuid=? AND event_type='kill'",
		fixM1, fixXUID,
	).Scan(&nEvents)
	if nEvents != 10 {
		t.Fatalf("expected 10 kill events for m1/fixXUID, got %d", nEvents)
	}

	var nKVP int
	f.shared.QueryRow(
		"SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id=? AND killer_xuid=?",
		fixM1, fixXUID,
	).Scan(&nKVP)
	if nKVP != 10 {
		t.Fatalf("expected 10 kvp rows for m1/fixXUID, got %d", nKVP)
	}

	var gt string
	f.shared.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid=?", fixXUID).Scan(&gt)
	if gt != fixGamertag {
		t.Fatalf("xuid_aliases: expected %s, got %s", fixGamertag, gt)
	}
}

// TestPipelineFixture_PlayerDataInserted vérifie que la player DB contient
// les lignes initiales player_match_enrichment.
func TestPipelineFixture_PlayerDataInserted(t *testing.T) {
	f := buildPipelineFixture(t)

	var n int
	f.player.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&n)
	if n < 3 {
		t.Fatalf("expected ≥3 pme rows (3 fixture + 12 seed), got %d", n)
	}
	// Les 3 matchs principaux doivent être présents
	var n3 int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE match_id IN (?, ?, ?)`, fixM1, fixM2, fixM3,
	).Scan(&n3)
	if n3 != 3 {
		t.Fatalf("expected 3 pme rows for fixture matches, got %d", n3)
	}
}

// TestPipelineFixture_MetadataInserted vérifie que citation_mappings est peuplé.
func TestPipelineFixture_MetadataInserted(t *testing.T) {
	f := buildPipelineFixture(t)

	var n int
	f.metadata.QueryRow("SELECT COUNT(*) FROM citation_mappings WHERE enabled=TRUE").Scan(&n)
	if n != 2 {
		t.Fatalf("expected 2 enabled citation mappings, got %d", n)
	}
}

// TestPipelineFixture_WeaponKills_FilmPresent simule BackfillWeaponKillsForMatchAll
// sur m1 avec un film "présent mais vide" (chunks binaires vides → 0 fire events).
// Vérifie que MBitWeaponKills est set et que le pipeline termine proprement.
func TestPipelineFixture_WeaponKills_FilmPresent(t *testing.T) {
	f := buildPipelineFixture(t)

	client := &weaponTestClient{
		filmPresent: true,
		filmChunks: map[int]FilmChunkData{
			0: {Data: []byte{}, StartMS: 0, DurationMS: 60000},
		},
	}

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), client, f.shared, fixM1)
	if err != nil {
		t.Fatalf("BackfillWeaponKillsForMatchAll m1: %v", err)
	}
	if !found {
		t.Fatal("expected found=true (film présent)")
	}

	var bits int
	f.shared.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id=?", fixM1).Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Fatalf("MBitWeaponKills non set après film présent (bits=%d)", bits)
	}
}

// TestPipelineFixture_WeaponKills_NoFilm vérifie que MBitWeaponKillsNoFilm est
// posé quand GetMatchFilm retourne filmPresent=false.
func TestPipelineFixture_WeaponKills_NoFilm(t *testing.T) {
	f := buildPipelineFixture(t)

	client := &weaponTestClient{filmPresent: false}

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), client, f.shared, fixM2)
	if err != nil {
		t.Fatalf("BackfillWeaponKillsForMatchAll m2: %v", err)
	}
	if found {
		t.Fatal("expected found=false (film absent)")
	}

	var bits int
	f.shared.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id=?", fixM2).Scan(&bits)
	if bits&int(MBitWeaponKillsNoFilm) == 0 {
		t.Fatalf("MBitWeaponKillsNoFilm non set (bits=%d)", bits)
	}
}

// TestPipelineFixture_WeaponKills_IdempotentRerun vérifie qu'un deuxième appel
// BackfillWeaponKillsForMatchAll ne crée pas de doublons dans weapon_kills.
func TestPipelineFixture_WeaponKills_IdempotentRerun(t *testing.T) {
	f := buildPipelineFixture(t)

	// Insérer des kill events dans highlight_events qui seront reconnus par le pipeline
	// (chunks vides → corrélation ne produit rien, mais le pipeline tourne)
	client := &weaponTestClient{
		filmPresent: true,
		filmChunks:  map[int]FilmChunkData{0: {Data: []byte{}, StartMS: 0, DurationMS: 60000}},
	}

	for i := 0; i < 2; i++ {
		_, err := BackfillWeaponKillsForMatchAll(context.Background(), client, f.shared, fixM1)
		if err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	// weapon_kills doit contenir exactement le même nombre de rows après 2 runs
	var count1, count2 int
	// Append-only #23046 (Phase 2) : idempotence LOGIQUE via v_weapon_kills (dernière
	// génération). Le physique croît à chaque run (nouvelle génération), mais la vue
	// reste stable → count1 == count2.
	f.shared.QueryRow("SELECT COUNT(*) FROM v_weapon_kills WHERE match_id=?", fixM1).Scan(&count1)
	BackfillWeaponKillsForMatchAll(context.Background(), client, f.shared, fixM1)
	f.shared.QueryRow("SELECT COUNT(*) FROM v_weapon_kills WHERE match_id=?", fixM1).Scan(&count2)

	if count1 != count2 {
		t.Fatalf("doublons logiques détectés (v_weapon_kills) : %d → %d rows après re-run", count1, count2)
	}
}

// TestPipelineFixture_PerformanceScore vérifie que batchComputePerformanceScores
// calcule des scores pour les matchs qui ont suffisamment d'historique.
// Avec 12 matchs seed + 3 matchs fixture = 15 matchs, les 5 derniers doivent
// recevoir un score (index >= MinMatchesForRelative=10).
func TestPipelineFixture_PerformanceScore(t *testing.T) {
	f := buildPipelineFixture(t)

	n, err := batchComputePerformanceScores(t.Context(), f.player, f.shared, fixXUID, nil, false)
	if err != nil {
		t.Fatalf("batchComputePerformanceScores: %v", err)
	}
	if n == 0 {
		t.Fatal("expected >0 matchs avec performance_score calculé")
	}

	// Vérifier que les 3 matchs fixture principaux ont au moins un score
	var scored int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE performance_score IS NOT NULL
		  AND match_id IN (?, ?, ?)`, fixM1, fixM2, fixM3,
	).Scan(&scored)
	if scored == 0 {
		t.Fatal("aucun des 3 matchs fixture principaux n'a de performance_score")
	}
}

// TestPipelineFixture_PerformanceScore_Idempotent vérifie qu'un double appel
// ne re-calcule pas les matchs déjà scorés (force=false).
func TestPipelineFixture_PerformanceScore_Idempotent(t *testing.T) {
	f := buildPipelineFixture(t)

	n1, err := batchComputePerformanceScores(t.Context(), f.player, f.shared, fixXUID, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	n2, err := batchComputePerformanceScores(t.Context(), f.player, f.shared, fixXUID, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Second run ne doit rien mettre à jour (déjà calculé, force=false)
	if n2 != 0 {
		t.Fatalf("second run (force=false) a mis à jour %d matchs au lieu de 0", n2)
	}
	// Force=true doit tout re-calculer
	n3, err := batchComputePerformanceScores(t.Context(), f.player, f.shared, fixXUID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if n3 < n1 {
		t.Fatalf("force=true a mis à jour %d matchs, attendu ≥%d", n3, n1)
	}
}

// TestPipelineFixture_LUSRRatings vérifie que batchComputeLUSR produit des
// ratings pour les matchs sociaux (is_ranked=false).
// m1 est ranked donc exclu. m2, m3 et les seed-matches sont sociaux.
func TestPipelineFixture_LUSRRatings(t *testing.T) {
	f := buildPipelineFixture(t)

	n, err := batchComputeLUSR(t.Context(), f.player, f.shared, fixXUID, nil, false)
	if err != nil {
		t.Fatalf("batchComputeLUSR: %v", err)
	}
	if n == 0 {
		t.Fatal("expected >0 LUSR ratings calculés (matchs sociaux présents)")
	}

	var rated int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM match_skill_rank
		WHERE rating_type = 'LUSR'
		  AND match_id IN (?, ?)`, fixM2, fixM3,
	).Scan(&rated)
	if rated == 0 {
		t.Fatal("aucun LUSR pour m2/m3")
	}
}

// TestPipelineFixture_LUSRRatings_RankedExcluded vérifie que m1 (ranked) n'a
// pas de rating LUSR (LUSR est réservé aux matchs non-ranked).
func TestPipelineFixture_LUSRRatings_RankedExcluded(t *testing.T) {
	f := buildPipelineFixture(t)

	_, err := batchComputeLUSR(t.Context(), f.player, f.shared, fixXUID, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	var hasM1 bool
	f.player.QueryRow(`
		SELECT COUNT(*) > 0 FROM match_skill_rank
		WHERE match_id = ? AND rating_type = 'LUSR'`, fixM1,
	).Scan(&hasM1)
	if hasM1 {
		t.Fatal("m1 (ranked) ne devrait pas avoir de rating LUSR")
	}
}

// TestPipelineFixture_SessionGrouping vérifie que l'algorithme de sessions
// regroupe m1+m2 dans la même session (écart 40 min < seuil 120 min) et
// place m3 dans une session séparée (écart 7 jours).
func TestPipelineFixture_SessionGrouping(t *testing.T) {
	f := buildPipelineFixture(t)

	opts := domain.SessionComputeOptions{
		GapMinutes:     120,
		Mode:           domain.SessionModeContext,
		TeamChangeMode: domain.TeamChangeModeIgnore,
	}
	n, err := recalculateSessionsInline(context.Background(), f.player, f.shared, fixXUID, opts, nil)
	if err != nil {
		t.Fatalf("recalculateSessionsInline: %v", err)
	}
	if n == 0 {
		t.Fatal("aucun match mis à jour par sessions")
	}

	// m1 et m2 doivent être dans la même session
	var sid1, sid2, sid3 sql.NullString
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM1).Scan(&sid1)
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM2).Scan(&sid2)
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM3).Scan(&sid3)

	if !sid1.Valid || !sid2.Valid || !sid3.Valid {
		t.Fatalf("session_id NULL — m1=%v, m2=%v, m3=%v", sid1, sid2, sid3)
	}
	if sid1.String != sid2.String {
		t.Fatalf("m1 et m2 devraient partager la même session (m1=%s, m2=%s)", sid1.String, sid2.String)
	}
	if sid1.String == sid3.String {
		t.Fatalf("m3 ne devrait pas être dans la même session que m1/m2")
	}
}

// TestPipelineFixture_SessionGrouping_WithFriends vérifie le mode TeamChangeModeFriends :
// m1 et m2 ont fixFriendXUID comme coéquipier → même session (friend présent).
// m3 n'a que fixStrangerXUID → nouvelle session (changement d'ami).
func TestPipelineFixture_SessionGrouping_WithFriends(t *testing.T) {
	f := buildPipelineFixture(t)

	opts := domain.SessionComputeOptions{
		GapMinutes:     120,
		Mode:           domain.SessionModeContext,
		TeamChangeMode: domain.TeamChangeModeFriends,
	}
	_, err := recalculateSessionsInline(
		context.Background(), f.player, f.shared, fixXUID, opts,
		[]string{fixFriendGamertag},
	)
	if err != nil {
		t.Fatalf("recalculateSessionsInline friends: %v", err)
	}

	var sid1, sid2, sid3 sql.NullString
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM1).Scan(&sid1)
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM2).Scan(&sid2)
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM3).Scan(&sid3)

	if sid1.String != sid2.String {
		t.Fatalf("m1/m2 devraient être dans la même session avec friends mode (m1=%s m2=%s)", sid1.String, sid2.String)
	}
	if sid1.String == sid3.String {
		t.Fatalf("m3 devrait être séparé (pas d'ami commun)")
	}
}

// TestPipelineFixture_Citations vérifie que BackfillMatchCitations produit des
// citations basées sur les médailles gagnées dans m1.
func TestPipelineFixture_Citations(t *testing.T) {
	f := buildPipelineFixture(t)

	err := BackfillMatchCitations(
		context.Background(),
		f.metadata, f.shared, f.player, nil,
		fixXUID,
		[]string{fixM1},
	)
	if err != nil {
		t.Fatalf("BackfillMatchCitations: %v", err)
	}

	var n int
	f.player.QueryRow(
		"SELECT COUNT(*) FROM match_citations_latest WHERE match_id=?", fixM1,
	).Scan(&n)
	if n == 0 {
		t.Fatal("aucune citation générée pour m1 malgré les médailles")
	}

	// Vérifier que bulltrue est présent (medal_id 12345, count 2)
	var bulltrue int
	f.player.QueryRow(
		"SELECT value FROM match_citations_latest WHERE match_id=? AND citation_name_norm='bulltrue'",
		fixM1,
	).Scan(&bulltrue)
	if bulltrue == 0 {
		t.Fatal("citation 'bulltrue' manquante ou value=0 pour m1")
	}
}

// TestPipelineFixture_Citations_Idempotent vérifie que deux appels successifs
// ne créent pas de doublons (DO NOTHING sur conflit).
func TestPipelineFixture_Citations_Idempotent(t *testing.T) {
	f := buildPipelineFixture(t)

	runCitations := func() {
		if err := BackfillMatchCitations(
			context.Background(),
			f.metadata, f.shared, f.player, nil, fixXUID, []string{fixM1},
		); err != nil {
			t.Fatalf("BackfillMatchCitations: %v", err)
		}
	}
	latestCount := func() int {
		var n int
		f.player.QueryRow("SELECT COUNT(*) FROM match_citations_latest WHERE match_id=?", fixM1).Scan(&n)
		return n
	}
	physCount := func() int {
		var n int
		f.player.QueryRow("SELECT COUNT(*) FROM match_citations WHERE match_id=?", fixM1).Scan(&n)
		return n
	}

	runCitations()
	n1, phys1 := latestCount(), physCount()
	if n1 == 0 {
		t.Fatal("aucune citation après le 1er appel")
	}
	runCitations()
	n2, phys2 := latestCount(), physCount()

	// Append-only #23046 — idempotence LOGIQUE : match_citations_latest stable
	// (zéro doublon visible) entre les deux runs.
	if n2 != n1 {
		t.Fatalf("doublon logique (match_citations_latest) : run1=%d run2=%d", n1, n2)
	}
	// Supersession GÉNÉRATIONNELLE : la table physique CROÎT (2e run = nouvelle
	// génération), preuve que ce n'est pas un DELETE+INSERT masqué mais bien de
	// l'append-only avec lecture par-génération.
	if phys2 <= phys1 {
		t.Fatalf("la table physique devrait croître (append-only) : run1=%d run2=%d", phys1, phys2)
	}
}

// TestPipelineFixture_Citations_NoMedals vérifie que BackfillMatchCitations
// ne produit aucune citation pour un match sans médailles (m3).
func TestPipelineFixture_Citations_NoMedals(t *testing.T) {
	f := buildPipelineFixture(t)

	err := BackfillMatchCitations(
		context.Background(),
		f.metadata, f.shared, f.player, nil,
		fixXUID,
		[]string{fixM3},
	)
	if err != nil {
		t.Fatalf("BackfillMatchCitations m3: %v", err)
	}

	var n int
	// Exclure le sentinel "_processed" écrit par writeCitations quand aucune
	// citation réelle n'est calculée (cf. citations.go:417-420). m3 n'a pas
	// de médailles → 0 citations réelles attendues (le sentinel marque juste
	// le match comme traité pour éviter une re-évaluation).
	f.player.QueryRow(
		"SELECT COUNT(*) FROM match_citations_latest WHERE match_id=? AND citation_name_norm <> '_processed'",
		fixM3,
	).Scan(&n)
	if n != 0 {
		t.Fatalf("expected 0 citations réelles pour m3 (pas de médailles), got %d", n)
	}
}

// TestPipelineFixture_EngagementScore vérifie que batchComputeEngagementScores
// calcule l'engagement_score pour les matchs sociaux (unranked).
//
// m1 (ranked) : aucun historique "PvP_ranked" → engagement_score=NULL mais
// engagement_score_brut est toujours posé.
// m2, m3 (unranked) : 12 entrées pré-seedées en historique "PvP_unranked" →
// ≥HistoryMinPartial=10 → engagement_score IS NOT NULL.
func TestPipelineFixture_EngagementScore(t *testing.T) {
	f := buildPipelineFixture(t)

	n, ups, err := batchComputeEngagementScores(context.Background(), f.player, f.shared, fixXUID, false)
	persistMatchIntensities(context.Background(), f.shared, ups)
	if err != nil {
		t.Fatalf("batchComputeEngagementScores: %v", err)
	}
	if n == 0 {
		t.Fatal("expected >0 matchs traités par batchComputeEngagementScores")
	}

	// engagement_score_brut est toujours posé (même sans historique suffisant)
	var brutScored int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE engagement_score_brut IS NOT NULL
		  AND match_id IN (?, ?, ?)`, fixM1, fixM2, fixM3,
	).Scan(&brutScored)
	if brutScored == 0 {
		t.Fatal("aucun engagement_score_brut calculé pour les 3 matchs fixture")
	}

	// m2 et m3 (unranked) ont 12 entrées d'historique → score percentile non-NULL
	var unrankedScored int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE engagement_score IS NOT NULL
		  AND match_id IN (?, ?)`, fixM2, fixM3,
	).Scan(&unrankedScored)
	if unrankedScored == 0 {
		t.Fatal("engagement_score NULL pour m2/m3 malgré 12 entrées d'historique unranked")
	}
}

// TestPipelineFixture_EngagementScore_Idempotent vérifie que force=false ne
// re-calcule pas les matchs déjà scorés (engagement_score IS NOT NULL).
// Applicable à m2/m3 (unranked, score non-NULL après premier run).
func TestPipelineFixture_EngagementScore_Idempotent(t *testing.T) {
	f := buildPipelineFixture(t)

	n1, ups1, err := batchComputeEngagementScores(context.Background(), f.player, f.shared, fixXUID, false)
	persistMatchIntensities(context.Background(), f.shared, ups1)
	if err != nil {
		t.Fatal(err)
	}
	if n1 == 0 {
		t.Fatal("premier run sans résultat")
	}
	// Cardinalité physique après le 1er run (append-only : 1 row stage='engagement'/match traité).
	var rows1 int
	f.player.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE stage = 'engagement'`).Scan(&rows1)

	// Append-only #23046 — IDEMPOTENCE STRICTE : le 2e run (force=false) doit skiper
	// TOUS les matchs déjà tentés, Y COMPRIS m1 (ranked, score=NULL insufficient_history).
	// Sans cela, m1 serait ré-INSÉRÉ à chaque cycle → croissance non bornée (bug audit
	// 2026-06-21). n2 == 0 ET la table ne grossit PAS.
	n2, ups2, err := batchComputeEngagementScores(context.Background(), f.player, f.shared, fixXUID, false)
	persistMatchIntensities(context.Background(), f.shared, ups2)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Fatalf("second run (force=false) a re-traité %d matchs, attendu 0 (idempotence : tout déjà tenté)", n2)
	}
	var rows2 int
	f.player.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE stage = 'engagement'`).Scan(&rows2)
	if rows2 != rows1 {
		t.Fatalf("croissance non bornée : rows stage='engagement' %d -> %d (insufficient_history ré-INSÉRÉ)", rows1, rows2)
	}
}

// TestPipelineFixture_KillerVictimPairs_Présent vérifie que la fixture a des
// killer_victim_pairs et que les gamertags correspondent à xuid_aliases.
func TestPipelineFixture_KillerVictimPairs_Present(t *testing.T) {
	f := buildPipelineFixture(t)

	var n int
	f.shared.QueryRow(`
		SELECT COUNT(*) FROM killer_victim_pairs
		WHERE match_id=? AND killer_xuid=?`, fixM1, fixXUID,
	).Scan(&n)
	if n != 10 {
		t.Fatalf("expected 10 kvp rows pour fixXUID dans m1, got %d", n)
	}

	// Vérifier qu'on peut rejoindre avec xuid_aliases
	var alliasedKills int
	f.shared.QueryRow(`
		SELECT COUNT(*) FROM killer_victim_pairs kvp
		JOIN xuid_aliases ka ON kvp.killer_xuid = ka.xuid
		WHERE kvp.match_id = ?`, fixM1,
	).Scan(&alliasedKills)
	if alliasedKills == 0 {
		t.Fatal("killer_victim_pairs ne peut pas joindre xuid_aliases")
	}
}

// TestPipelineFixture_XuidGamertag_Resolution vérifie la résolution xuid→gamertag
// depuis xuid_aliases (essentielle pour l'affichage des scores).
func TestPipelineFixture_XuidGamertag_Resolution(t *testing.T) {
	f := buildPipelineFixture(t)

	type tc struct {
		xuid string
		want string
	}
	cases := []tc{
		{fixXUID, fixGamertag},
		{fixFriendXUID, fixFriendGamertag},
		{fixEnemy1XUID, "Enemy1"},
		{fixStrangerXUID, "Stranger1"},
	}
	for _, c := range cases {
		var got string
		f.shared.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid=?", c.xuid).Scan(&got)
		if got != c.want {
			t.Errorf("xuid=%s: expected %q, got %q", c.xuid, c.want, got)
		}
	}
}

// TestPipelineFixture_FullSequence exécute toutes les étapes du pipeline dans
// l'ordre et vérifie que chaque étape produit un résultat cohérent.
// C'est le test de non-régression du pipeline complet.
func TestPipelineFixture_FullSequence(t *testing.T) {
	f := buildPipelineFixture(t)
	ctx := context.Background()

	// Étape 1 — weapon_kills (film présent pour m1)
	weaponClient := &weaponTestClient{
		filmPresent: true,
		filmChunks:  map[int]FilmChunkData{0: {Data: []byte{}, StartMS: 0, DurationMS: 60000}},
	}
	_, err := BackfillWeaponKillsForMatchAll(ctx, weaponClient, f.shared, fixM1)
	if err != nil {
		t.Fatalf("[étape 1] weapon_kills m1: %v", err)
	}
	// m2 et m3 sans film
	noFilmClient := &weaponTestClient{filmPresent: false}
	for _, mid := range []string{fixM2, fixM3} {
		if _, err := BackfillWeaponKillsForMatchAll(ctx, noFilmClient, f.shared, mid); err != nil {
			t.Fatalf("[étape 1] weapon_kills %s: %v", mid, err)
		}
	}

	// Étape 2 — performance_score
	if _, err := batchComputePerformanceScores(t.Context(), f.player, f.shared, fixXUID, nil, false); err != nil {
		t.Fatalf("[étape 2] performance_score: %v", err)
	}

	// Étape 3 — LUSR ratings
	if _, err := batchComputeLUSR(t.Context(), f.player, f.shared, fixXUID, nil, false); err != nil {
		t.Fatalf("[étape 3] LUSR: %v", err)
	}

	// Étape 4 — sessions
	opts := domain.SessionComputeOptions{
		GapMinutes:     120,
		Mode:           domain.SessionModeContext,
		TeamChangeMode: domain.TeamChangeModeIgnore,
	}
	if _, err := recalculateSessionsInline(ctx, f.player, f.shared, fixXUID, opts, nil); err != nil {
		t.Fatalf("[étape 4] sessions: %v", err)
	}

	// Étape 5 — citations
	if err := BackfillMatchCitations(ctx, f.metadata, f.shared, f.player, nil, fixXUID,
		[]string{fixM1, fixM2, fixM3}); err != nil {
		t.Fatalf("[étape 5] citations: %v", err)
	}

	// Étape 6 — engagement_score
	if _, _, err := batchComputeEngagementScores(ctx, f.player, f.shared, fixXUID, false); err != nil {
		t.Fatalf("[étape 6] engagement_score: %v", err)
	}

	// ── Assertions finales ────────────────────────────────────────────────────

	// m1 : MBitWeaponKills set
	var bitsM1 int
	f.shared.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id=?", fixM1).Scan(&bitsM1)
	if bitsM1&int(MBitWeaponKills) == 0 {
		t.Error("[full] MBitWeaponKills non set pour m1")
	}

	// m2 : MBitWeaponKillsNoFilm set
	var bitsM2 int
	f.shared.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id=?", fixM2).Scan(&bitsM2)
	if bitsM2&int(MBitWeaponKillsNoFilm) == 0 {
		t.Error("[full] MBitWeaponKillsNoFilm non set pour m2")
	}

	// sessions : m1 et m2 dans la même session
	var sid1, sid2 sql.NullString
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM1).Scan(&sid1)
	f.player.QueryRow("SELECT session_id FROM player_match_enrichment_latest WHERE match_id=?", fixM2).Scan(&sid2)
	if sid1.String == "" || sid1.String != sid2.String {
		t.Errorf("[full] m1/m2 doivent partager la même session (m1=%q m2=%q)", sid1.String, sid2.String)
	}

	// citations : m1 doit avoir au moins une citation
	var nCit int
	f.player.QueryRow("SELECT COUNT(*) FROM match_citations_latest WHERE match_id=?", fixM1).Scan(&nCit)
	if nCit == 0 {
		t.Error("[full] aucune citation pour m1")
	}

	// LUSR : au moins un rating calculé
	var nLUSR int
	f.player.QueryRow("SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='LUSR'").Scan(&nLUSR)
	if nLUSR == 0 {
		t.Error("[full] aucun rating LUSR")
	}

	// engagement_score_brut : toujours posé (score percentile dépend de l'historique)
	var engScored int
	f.player.QueryRow(`
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE engagement_score_brut IS NOT NULL AND match_id IN (?, ?, ?)`,
		fixM1, fixM2, fixM3,
	).Scan(&engScored)
	if engScored == 0 {
		t.Error("[full] aucun engagement_score_brut calculé")
	}
}
