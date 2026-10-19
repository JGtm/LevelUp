package ops

// seed_demo_replay_picks_non_gere_test.go — la demo ne fige aucun rejeu d'un mode que le rejeu ne
// gere pas (Firefight, decision du 2026-10-15), quel que soit le nom de sa variante.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestLoadDemoReplayCandidates_ExclutLeModeNonGere(t *testing.T) {
	root := t.TempDir()
	sharedPath := filepath.Join(root, "shared.duckdb")
	db, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatal(err)
	}
	const xuid = "2533274000000001"
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, game_variant_name VARCHAR, map_name VARCHAR,
			start_time TIMESTAMP, start_time_utc TIMESTAMPTZ, is_firefight BOOLEAN)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER, present_at_beginning BOOLEAN)`,
		`CREATE TABLE match_vehicle_takes_latest (match_id VARCHAR, row_kind VARCHAR, family VARCHAR, takes INTEGER, measured BOOLEAN)`,
		`INSERT INTO match_registry VALUES
			('39910eb1-8c01-4e26-bc60-965cb2dcc789', 'CTF:Arena', 'Recharge', TIMESTAMP '2026-10-01 10:00:00', TIMESTAMP '2026-10-01 10:00:00', FALSE),
			('edc5daf6-0df8-4eb0-b578-b8eb892ce184', 'Battle of the Academy', 'Cole Protocol', TIMESTAMP '2026-10-02 10:00:00', TIMESTAMP '2026-10-02 10:00:00', TRUE)`,
		`INSERT INTO match_participants (match_id, xuid) VALUES
			('39910eb1-8c01-4e26-bc60-965cb2dcc789', '` + xuid + `'), ('edc5daf6-0df8-4eb0-b578-b8eb892ce184', '` + xuid + `')`,
		`CHECKPOINT`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	// Les deux films sont au cache : seul le registre doit departager.
	cache := titlePkg.NewPathResolver(root).CacheRootDir()
	if err := os.MkdirAll(filepath.Join(cache, "film_manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, short := range []string{"39910eb1", "edc5daf6"} {
		if err := os.WriteFile(filepath.Join(cache, "film_manifests", short+".json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cands, err := loadDemoReplayCandidates(context.Background(), SeedDemoOptions{
		RepoRoot: root, TitleSlug: titlePkg.DefaultSlug, SourceSharedDB: sharedPath, SourceXUID: xuid,
	})
	if err != nil {
		t.Fatalf("loadDemoReplayCandidates : %v", err)
	}
	if len(cands) != 1 || cands[0].matchID != "39910eb1-8c01-4e26-bc60-965cb2dcc789" {
		t.Fatalf("candidats = %+v, attendu le seul match gere", cands)
	}
}
