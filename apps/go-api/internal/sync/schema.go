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
)

// playerSchemaSQL crée les tables minimales dans stats.duckdb d'un joueur.
// Portage de SYNC_SCHEMA_DDL (_engine_schema.py).
const playerSchemaSQL = `
CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
CREATE SEQUENCE IF NOT EXISTS psa_generation_seq START 1;
CREATE TABLE IF NOT EXISTS personal_score_awards (
    id         INTEGER   PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id   VARCHAR   NOT NULL,
    xuid       VARCHAR   NOT NULL,
    award_name VARCHAR   NOT NULL,
    award_category VARCHAR,
    award_count INTEGER  DEFAULT 1,
    award_score INTEGER  DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- APPEND-ONLY (campagne ART #23046, Phase 2). Plus de DELETE+INSERT sur les 4
    -- index ART. Chaque ecriture INSERE pur avec UN generation_id (sequence
    -- psa_generation_seq, partage par le batch) + written_at + is_tombstone.
    -- Lecture via personal_score_awards_latest (DENSE_RANK, generation MAX,
    -- tombstones exclus). Detail dans steps_player_append_only_personal_score_awards.go
    generation_id BIGINT NOT NULL DEFAULT 0,
    written_at TIMESTAMP DEFAULT now(),
    is_tombstone BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_psa_match    ON personal_score_awards(match_id);
CREATE INDEX IF NOT EXISTS idx_psa_xuid     ON personal_score_awards(xuid);
CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category);
CREATE INDEX IF NOT EXISTS idx_psa_gen      ON personal_score_awards(match_id, xuid, generation_id);

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
    written_at                  TIMESTAMP NOT NULL DEFAULT now(),
    created_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- idx_pme_match_lookup(match_id, written_at) est créé par la migration append-only
-- (player_append_only_match_enrichment_v1), PAS ici : sur une DB legacy pré-existante,
-- CREATE TABLE IF NOT EXISTS no-ope et written_at n'existe pas encore → CREATE INDEX
-- échouerait. La migration le pose après le swap (written_at garanti).

CREATE TABLE IF NOT EXISTS sync_meta (
    key        VARCHAR PRIMARY KEY,
    value      VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    written_at        TIMESTAMP NOT NULL DEFAULT now(),
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    recorded_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- ALTER additif idempotent pour migrer les player DBs existantes (post-refactor V2).
-- DuckDB ignore ADD COLUMN si déjà présente (≥ v0.10).
ALTER TABLE career_progression ADD COLUMN IF NOT EXISTS last_fetch_status VARCHAR;
CREATE INDEX IF NOT EXISTS idx_career_xuid ON career_progression(xuid);

-- Schéma append-only (Phase 2.G du refactor ART) : PK technique sur id,
-- N versions par (playlist_id, season_id), lecture via la vue
-- player_csr_snapshots_latest.
CREATE SEQUENCE IF NOT EXISTS pcs_seq START 1;
CREATE TABLE IF NOT EXISTS player_csr_snapshots (
    id                               BIGINT DEFAULT nextval('pcs_seq') PRIMARY KEY,
    playlist_id                      VARCHAR NOT NULL,
    playlist_name                    VARCHAR,
    queue                            VARCHAR,
    input                            VARCHAR,
    season_id                        VARCHAR NOT NULL,
    current_value                    FLOAT,
    current_tier                     VARCHAR,
    current_sub_tier                 SMALLINT,
    current_measurement_remaining    INTEGER,
    season_value                     FLOAT,
    season_tier                      VARCHAR,
    season_sub_tier                  SMALLINT,
    alltime_value                    FLOAT,
    alltime_tier                     VARCHAR,
    alltime_sub_tier                 SMALLINT,
    fetched_at                       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    written_at                       TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at);
CREATE OR REPLACE VIEW player_csr_snapshots_latest AS
    SELECT * FROM player_csr_snapshots
    QUALIFY ROW_NUMBER() OVER (PARTITION BY playlist_id, season_id ORDER BY written_at DESC, id DESC) = 1;
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
    created_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid)
);

CREATE TABLE IF NOT EXISTS medals_earned (
    match_id     VARCHAR  NOT NULL,
    xuid         VARCHAR  NOT NULL,
    medal_name_id BIGINT  NOT NULL,
    count        SMALLINT NOT NULL,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (match_id, xuid, medal_name_id)
);

CREATE TABLE IF NOT EXISTS xuid_aliases (
    xuid       VARCHAR PRIMARY KEY,
    gamertag   VARCHAR NOT NULL,
    last_seen  TIMESTAMP,
    source     VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    written_at                   TIMESTAMP NOT NULL DEFAULT now(),
    created_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
func EnsurePlayerSchema(ctx context.Context, db *sql.DB) error {
	if err := migration.EnsurePlayerCSRSnapshotsAppendOnly(db); err != nil {
		return fmt.Errorf("EnsurePlayerSchema: réparation append-only player_csr_snapshots: %w", err)
	}
	if err := execScript(ctx, db, playerSchemaSQL); err != nil {
		return recoverPlayerSchemaBoot(ctx, db, err)
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
	if err := EnsurePlayerSchema(context.Background(), handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenPlayerDB schema %s: %w", path, err)
	}
	// Append-only #23046 : garantir la vue player_match_enrichment_latest + la
	// conversion append-only de la table. EnsurePlayerSchema crée la TABLE mais PAS
	// la vue (créée par la migration player_append_only_match_enrichment_v1). Sans
	// cela, une player DB NEUVE ouverte hors du pool (1er sync) aurait la table mais
	// pas la vue → tous les readers post-sync (_latest) échoueraient. Idempotent.
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenPlayerDB append-only pme %s: %w", path, err)
	}
	// Append-only #23046 (Phase 2) : idem pour personal_score_awards — garantir la
	// vue personal_score_awards_latest + la conversion generation_id. Sans cela un
	// reader _latest casserait sur une player DB neuve. Idempotent.
	if err := migration.EnsurePersonalScoreAwardsAppendOnly(handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenPlayerDB append-only psa %s: %w", path, err)
	}
	// Append-only #23046 (Phase 2) : idem pour match_citations — garantir la vue
	// match_citations_latest + la conversion generation_id. Idempotent.
	if err := migration.EnsureMatchCitationsAppendOnly(handle.SQLDb()); err != nil {
		handle.Close()
		return nil, fmt.Errorf("OpenPlayerDB append-only mc %s: %w", path, err)
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
