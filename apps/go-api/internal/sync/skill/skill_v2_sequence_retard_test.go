//go:build cgo

package skill

// skill_v2_sequence_retard_test.go — le chemin du post-sync V2 (cycle auto_sync du serveur)
// n'écrit plus le LUSR canonique en « Duplicate key » sur une base joueur dont la séquence de
// match_skill_rank est en retard. Ce chemin ouvre la base par duckdb.OpenReadWrite (ouvreur
// playerDBOpenerRW de cmd/server/sync_v2_wiring.go), sans OpenPlayerDB ni EnsurePlayerSchema :
// l'alignement vient de l'ouverture physique (internal/platform/duckdb/physical_open.go).

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// msrRetardSQL — match_skill_rank append-only dont la séquence rend 1 alors que les ids 1..3
// sont pris (forme de la base de Chocoboflor, séquence recréée à START 1 par une reconstruction).
var msrRetardSQL = []string{
	`CREATE SEQUENCE match_skill_rank_id_seq START 1`,
	`CREATE TABLE match_skill_rank (
		id BIGINT DEFAULT nextval('match_skill_rank_id_seq') PRIMARY KEY,
		match_id VARCHAR NOT NULL, rating_type VARCHAR NOT NULL, rating_value FLOAT,
		rating_deviation FLOAT, tier VARCHAR, tier_fr VARCHAR, sub_tier SMALLINT, tier_label VARCHAR,
		rating_delta FLOAT, playlist_group VARCHAR, expected_win_prob FLOAT, start_time TIMESTAMP,
		written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`,
	`CREATE VIEW match_skill_rank_latest AS SELECT * FROM match_skill_rank
		QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC) = 1`,
	`INSERT INTO match_skill_rank (id, match_id, rating_type)
		SELECT range + 1, 'legacy-' || range, 'CSR' FROM range(3)`,
}

// preparePlayerDBMSRRetard crée, hors du cache du paquet duckdb, la base joueur en retard,
// vérifie la collision, puis ferme le fichier.
func preparePlayerDBMSRRetard(t *testing.T) string {
	t.Helper()
	path := title.NewPathResolver(t.TempDir()).PlayerDBPath(title.DefaultSlug, "Chocoboflor")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range msrRetardSQL {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("préparation %q: %v", s, err)
		}
	}
	_, err = db.Exec(`INSERT INTO match_skill_rank (match_id, rating_type) VALUES ('sonde', 'LUSR')`)
	if err == nil || !strings.Contains(err.Error(), "Duplicate key") {
		t.Fatalf("prémisse : une insertion sans id doit collisionner avant l'ouverture, err=%v", err)
	}
	return path
}

func TestRunLUSRV2ShadowOwnerOnly_CheminV2_SequenceEnRetardAligneeAvantLEcriture(t *testing.T) {
	t.Setenv(lusrV2EnvFlag, "1")
	t.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")
	sharedDB := openShadowTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_retard", time.Date(2025, 4, 1, 14, 0, 0, 0, time.UTC))

	handle, err := duckdb.OpenReadWrite(preparePlayerDBMSRRetard(t))
	if err != nil {
		t.Fatalf("ouverture (ouvreur du post-sync V2): %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })

	processed, err := RunLUSRV2ShadowOwnerOnly(context.Background(), handle.SQLDb(),
		newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, attendu 1 (écriture canonique tenue : séquence non alignée ?)", processed)
	}
	var n int
	if err := handle.SQLDb().QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest
		WHERE match_id = 'm_retard' AND rating_type = 'LUSR'`).Scan(&n); err != nil {
		t.Fatalf("lecture LUSR: %v", err)
	}
	if n != 1 {
		t.Fatalf("lignes LUSR de m_retard = %d, attendu 1", n)
	}
}
