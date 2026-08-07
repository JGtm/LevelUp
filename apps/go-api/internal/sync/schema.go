// Package sync — schema.go : DDL bootstrap pour les DBs player et shared.
//
// Portage de _engine_schema.py (player) et du schéma _SHARED_SCHEMA_SQL
// (_engine_connections.py) Python. Toutes les instructions sont idempotentes
// (CREATE TABLE IF NOT EXISTS / CREATE SEQUENCE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/schemadrift"
)

// playerSchemaSQL — SOIN DE TRANSITION du schéma player (plus une autorité).
//
// AUTORITÉ DE SCHÉMA = la chaîne de migrations (décision 2026-08-05). Ce script reste
// rejoué à chaque OpenPlayerDB pour les player DB LEGACY qui n'ont pas encore convergé,
// mais il ne doit RIEN ajouter à une DB fraîchement migrée : l'invariant est verrouillé
// par internal/sync/schema_authority_test.go, et toute action réelle est journalisée
// (schema_drift_healed, cf. internal/sync/schemadrift). Les blocs personal_score_awards et
// player_csr_snapshots proviennent de la SOURCE UNIQUE côté migrations
// (migration.PlayerPersonalScoreAwardsDDL / PlayerCSRSnapshotsDDL) — toute évolution s'y
// fait, jamais ici. L'ordre de concaténation reproduit l'ordre historique du script.
var playerSchemaSQL = migration.PlayerPersonalScoreAwardsDDL +
	playerCoreSchemaSQL +
	migration.PlayerCSRSnapshotsDDL

// playerCoreSchemaSQL — tables player dont le DDL de soin n'est pas partagé avec un step
// de migration (leur création vit dans create_baseline_player_v1, title-owned).
const playerCoreSchemaSQL = `
-- player_match_enrichment : APPEND-ONLY (campagne ART #23046, 2026-06-21). La
-- table la PLUS écrite du projet (écritures incrémentales partielles perf/engagement/
-- session/friends/bot/exclusion/psa) ne peut plus naître avec PK(match_id) + index
-- ART mutés. PK technique id (séquence pme_seq) + colonne stage discriminant
-- l'étape d'écriture + written_at. Chaque writer INSÈRE une row partielle taguée
-- (perf/session/engagement/friends/bot/exclusion/psa/dominance/teammates/live).
-- Lecture via la vue player_match_enrichment_latest (merge-on-read par-groupe),
-- créée/rafraîchie par la migration player_append_only_match_enrichment_v1 (source
-- unique = buildPMELatestViewSQL). Aucun PK(match_id), aucun index ART muté.
CREATE SEQUENCE IF NOT EXISTS pme_seq START 1;
CREATE TABLE IF NOT EXISTS player_match_enrichment (
    id                          BIGINT  DEFAULT nextval('pme_seq') PRIMARY KEY,
    match_id                    VARCHAR NOT NULL,
    performance_score           FLOAT,
    performance_chain           VARCHAR,
    dominance_flag              TINYINT,
    session_id                  VARCHAR,
    session_label               VARCHAR,
    is_with_friends             BOOLEAN   DEFAULT FALSE,
    teammates_signature         VARCHAR,
    had_bot_teammate            BOOLEAN,
    is_excluded                 BOOLEAN   DEFAULT FALSE,
    -- Marqueur terminal de la convergence PSA (cf. convergePSA). NULL = jamais
    -- tente. Non-NULL = JSON match fetche et extraction tentee (meme si 0 award).
    -- Empeche le re-fetch infini des matchs sans PersonalScores extractibles.
    psa_checked_at              TIMESTAMP,
    engagement_score            DOUBLE,
    engagement_score_brut       DOUBLE,
    engagement_score_confidence VARCHAR,
    mode_category               VARCHAR,
    engagement_pace_player      DOUBLE,
    engagement_pace_team        DOUBLE,
    engagement_pace_lobby       DOUBLE,
    engagement_player_activity  INTEGER,
    stage                       VARCHAR   DEFAULT 'legacy',
    written_at                  TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    created_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at                  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
-- idx_pme_match_lookup(match_id, written_at) est créé par la migration append-only
-- (player_append_only_match_enrichment_v1), PAS ici : sur une DB legacy pré-existante,
-- CREATE TABLE IF NOT EXISTS no-ope et written_at n'existe pas encore → CREATE INDEX
-- échouerait. La migration le pose après le swap (written_at garanti).

CREATE TABLE IF NOT EXISTS sync_meta (
    key        VARCHAR PRIMARY KEY,
    value      VARCHAR,
    updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);

-- Schéma append-only (Phase 2.B/E du refactor ART) : PK technique sur id,
-- N versions par (match_id, rating_type), lecture via la vue
-- match_skill_rank_latest avec priorité CSR > LUSR.
CREATE SEQUENCE IF NOT EXISTS msr_seq START 1;
CREATE TABLE IF NOT EXISTS match_skill_rank (
    id                BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
    match_id          VARCHAR NOT NULL,
    rating_type       VARCHAR NOT NULL,
    rating_value      FLOAT,
    rating_deviation  FLOAT,
    tier              VARCHAR,
    tier_fr           VARCHAR,
    sub_tier          SMALLINT DEFAULT 0,
    tier_label        VARCHAR,
    rating_delta      FLOAT,
    playlist_group    VARCHAR,
    expected_win_prob FLOAT,
    start_time        TIMESTAMP,
    written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at);
CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type);
CREATE INDEX IF NOT EXISTS idx_msr_playlist    ON match_skill_rank(playlist_group);
CREATE OR REPLACE VIEW match_skill_rank_latest AS
    SELECT * FROM match_skill_rank
    QUALIFY ROW_NUMBER() OVER (
        PARTITION BY match_id
        ORDER BY
            CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
            start_time DESC NULLS LAST,
            written_at DESC,
            id DESC
    ) = 1;

CREATE SEQUENCE IF NOT EXISTS career_progression_id_seq;
CREATE TABLE IF NOT EXISTS career_progression (
    id              INTEGER PRIMARY KEY DEFAULT nextval('career_progression_id_seq'),
    xuid            VARCHAR NOT NULL,
    rank            INTEGER,
    rank_name       VARCHAR,
    rank_tier       VARCHAR,
    current_xp      INTEGER,
    xp_for_next_rank INTEGER,
    xp_total        INTEGER,
    is_max_rank     BOOLEAN DEFAULT FALSE,
    adornment_path  VARCHAR,
    spartan_id      VARCHAR,
    banner_image_url VARCHAR,
    emblem_image_url VARCHAR,
    backdrop_image_url VARCHAR,
    last_fetch_status VARCHAR,  -- Phase 6 PLAN_V2 : 'ok' / 'api_empty' / 'forbidden_403' / 'auth_missing' / 'failed'
    recorded_at     TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
-- ALTER additif idempotent pour migrer les player DBs existantes (post-refactor V2).
-- DuckDB ignore ADD COLUMN si déjà présente (≥ v0.10).
ALTER TABLE career_progression ADD COLUMN IF NOT EXISTS last_fetch_status VARCHAR;
-- idx_career_xuid : SUPPRIMÉ (décision 2026-08-05, arbitrage doctrinal option 2). Dans
-- une player DB, xuid est QUASI CONSTANT (une DB = un joueur) → sélectivité nulle,
-- l'index n'accélère aucun filtre mais porte la classe de corruption ART DuckDB #23046
-- (ADR 0019/0026), la plus chère de l'histoire du projet. La convergence des DB
-- EXISTANTES est assurée par le step drop_career_xuid_art_index_v1 (migration player).
`

// sharedSchemaSQL crée les tables minimales dans shared_matches_v2.duckdb.
// Portage de _SHARED_SCHEMA_SQL (_engine_connections.py).
const sharedSchemaSQL = `
CREATE TABLE IF NOT EXISTS match_registry (
    match_id                  VARCHAR   PRIMARY KEY,
    start_time                TIMESTAMP NOT NULL,
    end_time                  TIMESTAMP,
    start_time_utc            TIMESTAMPTZ,
    end_time_utc              TIMESTAMPTZ,
    playlist_id               VARCHAR,
    playlist_name             VARCHAR,
    playlist_version_id       VARCHAR,
    map_id                    VARCHAR,
    map_name                  VARCHAR,
    map_version_id            VARCHAR,
    pair_id                   VARCHAR,
    pair_name                 VARCHAR,
    pair_version_id           VARCHAR,
    game_variant_id           VARCHAR,
    game_variant_name         VARCHAR,
    game_variant_version_id   VARCHAR,
    mode_category             VARCHAR,
    is_ranked                 BOOLEAN DEFAULT FALSE,
    is_firefight              BOOLEAN DEFAULT FALSE,
    duration_seconds          INTEGER,
    playable_duration_seconds INTEGER,
    real_start_time           TIMESTAMP,
    team_0_score              SMALLINT,
    team_1_score              SMALLINT,
    team_0_ps_score           INTEGER,
    team_1_ps_score           INTEGER,
    backfill_completed        INTEGER  DEFAULT 0,
    participants_loaded       BOOLEAN  DEFAULT FALSE,
    events_loaded             BOOLEAN  DEFAULT FALSE,
    medals_loaded             BOOLEAN  DEFAULT FALSE,
    first_sync_by             VARCHAR,
    first_sync_at             TIMESTAMP,
    last_updated_at           TIMESTAMP,
    player_count              SMALLINT DEFAULT 0,
    created_at                TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at                TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    season_id                 VARCHAR
);

CREATE TABLE IF NOT EXISTS match_participants (
    match_id             VARCHAR NOT NULL,
    xuid                 VARCHAR NOT NULL,
    gamertag             VARCHAR,
    team_id              INTEGER,
    outcome              INTEGER,
    rank                 SMALLINT,
    score                INTEGER,
    kills                SMALLINT,
    deaths               SMALLINT,
    assists              SMALLINT,
    kda                  FLOAT,
    accuracy             FLOAT,
    shots_fired          INTEGER,
    shots_hit            INTEGER,
    damage_dealt         FLOAT,
    damage_taken         FLOAT,
    personal_score       INTEGER,
    time_played_seconds  INTEGER,
    avg_life_seconds     FLOAT,
    kills_expected       FLOAT,
    deaths_expected      FLOAT,
    kills_stddev         FLOAT,
    deaths_stddev        FLOAT,
    team_mmr             FLOAT,
    enemy_mmr            FLOAT,
    headshot_kills       SMALLINT DEFAULT 0,
    max_killing_spree    SMALLINT,
    grenade_kills        SMALLINT,
    melee_kills          SMALLINT,
    power_weapon_kills   SMALLINT,
    present_at_beginning  BOOLEAN,
    present_at_completion BOOLEAN,
    joined_in_progress    BOOLEAN,
    left_in_progress      BOOLEAN,
    first_joined_time     TIMESTAMPTZ,
    last_leave_time       TIMESTAMPTZ,
    created_at           TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    PRIMARY KEY (match_id, xuid)
);

CREATE TABLE IF NOT EXISTS medals_earned (
    match_id     VARCHAR  NOT NULL,
    xuid         VARCHAR  NOT NULL,
    medal_name_id BIGINT  NOT NULL,
    count        SMALLINT NOT NULL,
    created_at   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    PRIMARY KEY (match_id, xuid, medal_name_id)
);

CREATE TABLE IF NOT EXISTS xuid_aliases (
    xuid       VARCHAR PRIMARY KEY,
    gamertag   VARCHAR NOT NULL,
    last_seen  TIMESTAMP,
    source     VARCHAR,
    updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);

-- Schéma append-only (Phase 2.F du refactor ART) : PK technique sur id,
-- N versions par (match_id, xuid), lecture via la vue match_csrs_latest.
CREATE SEQUENCE IF NOT EXISTS mcsrs_seq START 1;
CREATE TABLE IF NOT EXISTS match_csrs (
    id                           BIGINT DEFAULT nextval('mcsrs_seq') PRIMARY KEY,
    match_id                     VARCHAR NOT NULL,
    xuid                         VARCHAR NOT NULL,
    rating_type                  VARCHAR NOT NULL DEFAULT 'CSR',
    rating_value                 FLOAT,
    tier                         VARCHAR,
    sub_tier                     SMALLINT DEFAULT 0,
    tier_label                   VARCHAR,
    rating_delta                 FLOAT,
    measurement_matches_remaining INTEGER DEFAULT 0,
    season_id                    VARCHAR,
    written_at                   TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    created_at                   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
    updated_at                   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE INDEX IF NOT EXISTS idx_match_csrs_lookup ON match_csrs(match_id, xuid, written_at);
CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid);
CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id);
CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id);
CREATE OR REPLACE VIEW match_csrs_latest AS
    SELECT * FROM match_csrs
    QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1;

-- killer_victim_pairs : historiquement créée par les migrations uniquement.
-- Remontée dans le schéma de base pour v_gamertag_lookup. Le résolveur lit
-- désormais match_kill_events (bascule du 2026-08-02), mais la table reste lue
-- par Q20 et une vingtaine de sites, et reste écrite par les persisters.
-- Schéma aligné sur la migration (steps_shared.go), CREATE IF NOT EXISTS idempotent.
CREATE TABLE IF NOT EXISTS killer_victim_pairs (
    match_id        VARCHAR NOT NULL,
    killer_xuid     VARCHAR NOT NULL,
    killer_gamertag VARCHAR,
    victim_xuid     VARCHAR NOT NULL,
    victim_gamertag VARCHAR,
    kill_count      INTEGER DEFAULT 1,
    time_ms         INTEGER,
    is_validated    BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
`

// EnsurePlayerSchema crée les tables player si elles n'existent pas.
// Idempotent — peut être appelé à chaque ouverture de connexion.
//
// C1 (revue 2026-07-17, findings M2/M3) — RÉPARATION PROACTIVE de player_csr_snapshots
// AVANT playerSchemaSQL : une player DB legacy (backup pré-2026-05-24) porte l'ANCIEN schéma
// PK(playlist_id, season_id) SANS colonnes id/written_at. playerSchemaSQL crée ensuite l'index
// idx_pcs_lookup(... written_at) puis la vue player_csr_snapshots_latest (QUALIFY ... written_at,
// id) : sur l'ancien schéma leur BIND échoue → OpenPlayerDB mort définitivement (aucun step de
// migration restant ne répare, la conversion ayant été squashée). La réparation par
// introspection des colonnes (jamais de sentinelle ; conversion CTAS transactionnelle
// idempotente, ADR 0026) DOIT donc précéder la création de la vue. No-op sur DB vierge ou déjà
// convertie.
// SOIN VISIBLE (2026-08-05) : le schéma est photographié avant/après ; toute création
// RÉELLE est journalisée en WARN `schema_drift_healed` (cf. sync/schemadrift). Sur une
// player DB à jour, cette fonction est un NO-OP intégral et silencieux — invariant
// verrouillé par schema_authority_test.go.
//
// PÉRIMÈTRE ÉLARGI (2026-08-05) : les 3 conversions append-only résiduelles vivaient
// APRÈS l'appel dans OpenPlayerDB, donc HORS de la photographie — leur soin était le
// dernier angle mort silencieux du provisioning player. Elles sont désormais appelées
// ICI, entre le snapshot et le report : le soin complet est visible sous une seule
// autorité, et l'invariant no-op couvre le chemin d'ouverture ENTIER.
func EnsurePlayerSchema(ctx context.Context, db *sql.DB) error {
	before := schemadrift.Snapshot(ctx, db)
	if err := migration.EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		return fmt.Errorf("EnsurePlayerSchema: réparation append-only player_csr_snapshots: %w", err)
	}
	if err := execScript(ctx, db, playerSchemaSQL); err != nil {
		if rerr := recoverPlayerSchemaBoot(ctx, db, err); rerr != nil {
			return rerr
		}
	}
	if err := ensurePlayerAppendOnlyTables(db); err != nil {
		return err
	}
	schemadrift.Report(ctx, db, before, "sync.EnsurePlayerSchema")
	return nil
}

// playerAppendOnlyCares — conversions append-only (#23046, ADR 0026) garanties à CHAQUE
// ouverture d'une player DB, en plus du DDL de soin. playerSchemaSQL crée les TABLES mais
// jamais les vues `_latest` (leur bind échouerait sur une table legacy non convertie) :
// sans ces appels, une player DB NEUVE ouverte hors chaîne de migrations aurait la table
// mais pas la vue, et tous les readers post-sync casseraient. Chacune est idempotente et
// no-ope si la table est absente.
//
// Le nom sert UNIQUEMENT au message d'erreur ; l'ordre est celui, historique, des appels
// qu'OpenPlayerDB portait avant le 2026-08-05.
var playerAppendOnlyCares = []struct {
	table string
	care  func(*sql.DB) error
}{
	{"player_match_enrichment", migration.EnsurePlayerMatchEnrichmentAppendOnly},
	{"personal_score_awards", migration.EnsurePersonalScoreAwardsAppendOnly},
	{"match_citations", migration.EnsureMatchCitationsAppendOnly},
}

func ensurePlayerAppendOnlyTables(db *sql.DB) error {
	for _, c := range playerAppendOnlyCares {
		if err := c.care(db); err != nil {
			return fmt.Errorf("EnsurePlayerSchema: conversion append-only %s: %w", c.table, err)
		}
	}
	return nil
}

// recoverPlayerSchemaBoot — filet convergent (C4, revue 2026-07-17, findings M2/M3) sur un échec
// de DDL/bind au boot player. Un tel échec ne doit JAMAIS rester un échec permanent SILENCIEUX
// (avant ce filet, l'erreur remontait telle quelle et re-cassait à chaque boot sans piste). On :
//  1. journalise la cause EXACTE (slog.ErrorContext : vue ciblée + erreur brute) ;
//  2. rejoue la réparation append-only idempotente (referme un schéma player_csr_snapshots legacy
//     résiduel qui aurait échappé au passage proactif) ;
//  3. rejoue le script UNE fois.
//
// Si l'échec PERSISTE après réparation → erreur EXPLICITE exploitable (intervention requise),
// jamais avalée. Jamais de panic. Sur le chemin nominal (C1 proactif a déjà converti), ce filet
// n'est pas atteint ; il couvre les échecs de bind RÉSIDUELS/imprévus (défense en profondeur,
// doctrine convergent sync D3).
func recoverPlayerSchemaBoot(ctx context.Context, db *sql.DB, cause error) error {
	slog.ErrorContext(ctx,
		"EnsurePlayerSchema: DDL/bind player échoué au boot — réparation append-only puis retry",
		"err", cause, "target", "player", "view", "player_csr_snapshots_latest")
	if err := migration.EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		return fmt.Errorf("EnsurePlayerSchema: DDL échoué (%v) ET réparation player_csr_snapshots échouée: %w", cause, err)
	}
	if err := execScript(ctx, db, playerSchemaSQL); err != nil {
		return fmt.Errorf("EnsurePlayerSchema: DDL player échoué au boot même après réparation append-only (schéma irréparable, intervention requise): %w", err)
	}
	slog.InfoContext(ctx, "EnsurePlayerSchema: schéma player rétabli après réparation append-only", "target", "player")
	return nil
}

// EnsureSharedSchema crée les tables shared + les vues SQL (v_gamertag_lookup,
// v_match_full) si elles n'existent pas. Idempotent — peut être appelé à
// chaque ouverture de connexion.
//
// Les vues sont créées ICI (boot, conn RW pure ouverte par OpenReadWrite)
// et NON depuis le post-sync runtime. Raison — fix bug 2026-05-27 :
//
//	En post-sync, le sharedDB writer est obtenu via Provider.AcquireWriter
//	APRÈS qu'une instance DuckDB RO ait existé (SharedReader). Même si le
//	Provider ferme la RO avant d'ouvrir la RW (swap), DuckDB peut auto-attacher
//	l'ancien handle sous le nom dérivé du fichier (`shared_matches_v2`). À
//	ce moment CREATE OR REPLACE VIEW v_gamertag_lookup AS SELECT FROM
//	xuid_aliases tombe sur la database attached RO (resolution `xuid_aliases`
//	→ `shared_matches_v2.xuid_aliases` car c'est la seule où la table existe)
//	→ "Cannot execute statement of type CREATE on database shared_matches_v2
//	which is attached in read-only mode!" (logs/sync.log 23:57:51 et+).
//
//	Au boot via OpenSharedDB, aucune instance RO n'existe encore → CREATE
//	VIEW passe sans souci. Les VIEW sont stockées dans la DB elle-même, donc
//	survivent au close/reopen. À chaque boot, CREATE OR REPLACE écrase
//	l'ancienne définition (idempotent si la query a évolué).
func EnsureSharedSchema(ctx context.Context, db *sql.DB) error {
	if err := execScript(ctx, db, sharedSchemaSQL); err != nil {
		return err
	}
	// match_kill_events : la dépendance de v_gamertag_lookup depuis la bascule du
	// 2026-08-02 (le résolveur y lit le kill-feed). DuckDB bind les vues à leur
	// création, donc la table passe AVANT — et son DDL n'existe qu'à un endroit,
	// dans la migration qui la possède.
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		return err
	}
	// v_gamertag_lookup : SOURCE UNIQUE DE VÉRITÉ (analysis.GamertagLookupViewSQL).
	// AVANT le fix 2026-05-30 ce boot recréait une version simplifiée (filtre
	// `WHERE gamertag IS NOT NULL`, pas de CASE bot, fallback xuid brut) qui
	// écrasait à chaque boot la version robuste posée par les migrations →
	// xuids bruts réapparaissaient à l'affichage. Désormais boot et migrations
	// partagent le même DDL.
	if err := execScript(ctx, db, analysis.GamertagLookupViewSQL()+";"); err != nil {
		return err
	}
	return execScript(ctx, db, sharedViewsSQL)
}

// sharedViewsSQL : vues SQL recréées à chaque boot via EnsureSharedSchema.
// Stable jusqu'à la prochaine évolution de query (recompilées au boot).
// AVANT le fix 2026-05-27 ces VIEW étaient recreées par refreshSharedViews
// dans le post-sync ; voir godoc EnsureSharedSchema pour le contexte.
// v_gamertag_lookup est posée séparément (cf. EnsureSharedSchema) car son DDL
// est généré par analysis.GamertagLookupViewSQL (source unique).
const sharedViewsSQL = `
CREATE OR REPLACE VIEW v_match_full AS
SELECT mr.* FROM match_registry mr;
`

// OpenPlayerDB ouvre stats.duckdb d'un joueur en lecture/écriture via le cache process-level.
// Crée le répertoire si nécessaire. Applique le schéma player si absent.
// Retourne un *duckdbpkg.DB ref-compté ; appeler .Close() quand terminé.
func OpenPlayerDB(path string) (*duckdbpkg.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("OpenPlayerDB mkdir %s: %w", path, err)
	}
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("OpenPlayerDB open %s: %w", path, err)
	}
	// EnsurePlayerSchema est une opération de boot idempotente (CREATE TABLE IF
	// NOT EXISTS). Pas de contexte caller au moment de l'ouverture du handle ;
	// les appels ne sont pas annulables côté caller — context.Background() est
	// le bon choix ici.
	//
	// Les 3 conversions append-only (pme / psa / match_citations) qu'OpenPlayerDB
	// appelait ici sont DANS EnsurePlayerSchema depuis le 2026-08-05 : hors d'elle,
	// elles échappaient à la photographie schemadrift (soin silencieux). Le soin
	// player est désormais entier sous une autorité unique et instrumentée.
	if err := EnsurePlayerSchema(context.Background(), handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenPlayerDB schema %s: %w", path, err)
	}
	return handle, nil
}

// OpenSharedDB ouvre shared_matches_v2.duckdb en lecture/écriture via le cache
// process-level. Applique le schéma shared si absent.
// Retourne un *duckdbpkg.DB ref-compté ; appeler .Close() quand terminé.
//
// Note : le store global xbox_aliases (openGlobalDB) a été supprimé le
// 2026-06-19 — consolidé dans shared.xuid_aliases (plus aucun lecteur ni
// écrivain). Cf. v_gamertag_lookup + chemin convergent upsertAliasesFromMatchJSON.
func OpenSharedDB(path string) (*duckdbpkg.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("OpenSharedDB mkdir %s: %w", path, err)
	}
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("OpenSharedDB open %s: %w", path, err)
	}
	// EnsureSharedSchema est une opération de boot idempotente (CREATE TABLE IF
	// NOT EXISTS). Pas de contexte caller — context.Background() OK.
	if err := EnsureSharedSchema(context.Background(), handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenSharedDB schema %s: %w", path, err)
	}
	return handle, nil
}

// execScript exécute un script SQL multi-instructions séparées par ";".
func execScript(ctx context.Context, db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execScript: %w (stmt=%q)", err, truncate(stmt, 80))
		}
	}
	return nil
}

// splitSQL découpe un script SQL en instructions individuelles (séparateur ";").
func splitSQL(script string) []string {
	var stmts []string
	var cur []byte
	for i := 0; i < len(script); i++ {
		ch := script[i]
		if ch == ';' {
			s := trimSpace(string(cur))
			if s != "" {
				stmts = append(stmts, s)
			}
			cur = cur[:0]
		} else {
			cur = append(cur, ch)
		}
	}
	if s := trimSpace(string(cur)); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
