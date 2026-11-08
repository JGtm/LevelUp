package sync

// lusr_full_recompute_held_writes_test.go — le rejeu complet du LUSR n'a pas de cycle
// suivant : une écriture canonical tenue remonte (skill.ErrCanonicalWriteHeld) avec le
// décompte des matchs traités, au lieu de laisser croire à un rejeu complet.

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/sync/skill"
)

// openRefusingPlayerDB ouvre une base joueur en mémoire dont match_skill_rank refuse toute
// écriture (CHECK impossible) : l'écriture canonical de chaque match échoue.
func openRefusingPlayerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const ddl = `
		CREATE SEQUENCE match_skill_rank_id_seq;
		CREATE TABLE match_skill_rank (
			id BIGINT DEFAULT nextval('match_skill_rank_id_seq') PRIMARY KEY,
			match_id VARCHAR NOT NULL, rating_type VARCHAR NOT NULL, rating_value FLOAT,
			rating_deviation FLOAT, tier VARCHAR, tier_fr VARCHAR, sub_tier SMALLINT,
			tier_label VARCHAR, rating_delta FLOAT, playlist_group VARCHAR,
			expected_win_prob FLOAT, start_time TIMESTAMP,
			written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			CHECK (FALSE)
		);
		CREATE VIEW match_skill_rank_latest AS
		  SELECT * FROM match_skill_rank
		  QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC) = 1;
		CREATE VIEW match_skill_rank_latest_by_type AS
		  SELECT * FROM match_skill_rank
		  QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC, id DESC) = 1;`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("DDL player: %v", err)
	}
	return db
}

func TestRecomputeLUSRCanonical_EcritureTenue_RemonteErreurEtDecompte(t *testing.T) {
	sharedPath := filepath.Join(t.TempDir(), "shared_matches_v2.duckdb")
	seedLUSRShared(t, sharedPath, 3)
	shared, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer shared.Close()
	player := openRefusingPlayerDB(t)

	n, err := RecomputeLUSRCanonicalForPlayer(context.Background(), player, shared.SQLDb(), recomputerTestXUID)
	if !errors.Is(err, skill.ErrCanonicalWriteHeld) {
		t.Fatalf("err = %v, want skill.ErrCanonicalWriteHeld (écritures refusées par la base joueur)", err)
	}
	if n != 0 {
		t.Errorf("processed = %d, want 0 (aucune note écrite) rendu avec l'erreur", n)
	}
}
