// Package sync — schema.go : DDL bootstrap pour les DBs player et shared.
//
// Portage de _engine_schema.py (player) et du schéma _SHARED_SCHEMA_SQL
// (_engine_connections.py) Python. Toutes les instructions sont idempotentes
// (CREATE TABLE IF NOT EXISTS / CREATE SEQUENCE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).
package sync

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// playerSchemaSQL crée les tables minimales dans stats.duckdb d'un joueur.
// Portage de SYNC_SCHEMA_DDL (_engine_schema.py).
const playerSchemaSQL = `
CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq;
CREATE TABLE IF NOT EXISTS personal_score_awards (
    id         INTEGER   PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
    match_id   VARCHAR   NOT NULL,
    xuid       VARCHAR   NOT NULL,
    award_name VARCHAR   NOT NULL,
    award_category VARCHAR,
    award_count INTEGER  DEFAULT 1,
    award_score INTEGER  DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_psa_match    ON personal_score_awards(match_id);
CREATE INDEX IF NOT EXISTS idx_psa_xuid     ON personal_score_awards(xuid);
CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category);

CREATE TABLE IF NOT EXISTS player_match_enrichment (
    match_id               VARCHAR   PRIMARY KEY,
    performance_score      FLOAT,
    session_id             VARCHAR,
    session_label          VARCHAR,
    is_with_friends        BOOLEAN,
    teammates_signature    VARCHAR,
    known_teammates_count  SMALLINT,
    friends_xuids          VARCHAR,
    had_bot_teammate       BOOLEAN,
    created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pme_session ON player_match_enrichment(session_id);

CREATE TABLE IF NOT EXISTS sync_meta (
    key        VARCHAR PRIMARY KEY,
    value      VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

// sharedSchemaSQL crée les tables minimales dans shared_matches_v2.duckdb.
// Portage de _SHARED_SCHEMA_SQL (_engine_connections.py).
const sharedSchemaSQL = `
CREATE TABLE IF NOT EXISTS match_registry (
    match_id                  VARCHAR   PRIMARY KEY,
    start_time                TIMESTAMP NOT NULL,
    end_time                  TIMESTAMP,
    playlist_id               VARCHAR,
    playlist_name             VARCHAR,
    map_id                    VARCHAR,
    map_name                  VARCHAR,
    pair_id                   VARCHAR,
    pair_name                 VARCHAR,
    game_variant_id           VARCHAR,
    game_variant_name         VARCHAR,
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
    updated_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS match_participants (
    match_id     VARCHAR NOT NULL,
    xuid         VARCHAR NOT NULL,
    gamertag     VARCHAR,
    team_id      INTEGER,
    outcome      INTEGER,
    rank         SMALLINT,
    score        INTEGER,
    kills        SMALLINT,
    deaths       SMALLINT,
    assists      SMALLINT,
    shots_fired  INTEGER,
    shots_hit    INTEGER,
    damage_dealt FLOAT,
    damage_taken FLOAT,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
`

// EnsurePlayerSchema crée les tables player si elles n'existent pas.
// Idempotent — peut être appelé à chaque ouverture de connexion.
func EnsurePlayerSchema(db *sql.DB) error {
	return execScript(db, playerSchemaSQL)
}

// EnsureSharedSchema crée les tables shared si elles n'existent pas.
// Idempotent — peut être appelé à chaque ouverture de connexion.
func EnsureSharedSchema(db *sql.DB) error {
	return execScript(db, sharedSchemaSQL)
}

// OpenPlayerDB ouvre stats.duckdb d'un joueur en lecture/écriture.
// Crée le répertoire si nécessaire. Applique le schéma player si absent.
func OpenPlayerDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("OpenPlayerDB mkdir %s: %w", path, err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("OpenPlayerDB open %s: %w", path, err)
	}
	// DuckDB n'est pas conçu pour le multi-writer concurrent — 1 connexion active.
	db.SetMaxOpenConns(1)
	if err := EnsurePlayerSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("OpenPlayerDB schema %s: %w", path, err)
	}
	return db, nil
}

// OpenSharedDB ouvre shared_matches_v2.duckdb en lecture/écriture.
// Applique le schéma shared si absent.
func OpenSharedDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("OpenSharedDB mkdir %s: %w", path, err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("OpenSharedDB open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := EnsureSharedSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("OpenSharedDB schema %s: %w", path, err)
	}
	return db, nil
}

// execScript exécute un script SQL multi-instructions séparées par ";".
func execScript(db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.Exec(stmt); err != nil {
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
