//go:build integration

// lusrdb_helpers_test.go — helper de test openLUSRDB copié depuis skill_rating_loaders_test.go
// lors de l'extraction du package skill (K3c). Utilisé par art_rebuild_e2e_test.go (sync).
// Duplication test-only : le helper vivait dans un fichier de test qui a migré vers skill ;
// sync ne peut pas importer les helpers de test d'un autre package. execScript = version
// sync/schema.go.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openLUSRDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			start_time_utc TIMESTAMPTZ,
			playlist_name VARCHAR,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			accuracy DOUBLE,
			team_id INTEGER
		);
		CREATE SEQUENCE msr_seq START 1;
		CREATE TABLE match_skill_rank (
			id               BIGINT DEFAULT nextval('msr_seq') PRIMARY KEY,
			match_id         VARCHAR NOT NULL,
			rating_type      VARCHAR NOT NULL,
			rating_value     DOUBLE,
			rating_deviation DOUBLE,
			tier             VARCHAR,
			tier_fr          VARCHAR,
			sub_tier         SMALLINT DEFAULT 0,
			tier_label       VARCHAR,
			rating_delta     DOUBLE,
			playlist_group   VARCHAR,
			expected_win_prob FLOAT,
			start_time       TIMESTAMPTZ,
			written_at       TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			created_at       TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at       TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at);
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 ELSE 1 END,
					written_at DESC,
					id DESC
			) = 1;
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}
