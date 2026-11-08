package duckdb

// career_repo_lusr_exclusion_test.go — un match exclu disparaît du graphe LUSR / CSR de
// la Carrière, et l'écart d'un point se calcule entre les matchs RETENUS.
//
// Fixture sur fichiers (t.TempDir) : player et shared portent le schéma des migrations
// réelles, pas une DDL recopiée.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// openMigratedFile ouvre une base de test sur fichier, migrée pour la cible donnée.
func openMigratedFile(t *testing.T, name string, target migration.TargetDB) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	sqlDB, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migration.RunForDB(sqlDB, target); err != nil {
		t.Fatalf("RunForDB(%s): %v", target, err)
	}
	return newTestDB(sqlDB, path)
}

// seedLUSRCareerMatch sème un match noté LUSR (player) et son registre (shared).
func seedLUSRCareerMatch(t *testing.T, pdb *PlayerDB, matchID string, value float64, start time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx,
		`INSERT INTO match_registry (match_id, start_time, start_time_utc) VALUES (?, ?, ?)`,
		matchID, start, start); err != nil {
		t.Fatalf("insert registry %s: %v", matchID, err)
	}
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group, start_time)
		 VALUES (?, 'LUSR', ?, 'arena_slayer', ?)`,
		matchID, value, start); err != nil {
		t.Fatalf("insert msr %s: %v", matchID, err)
	}
}

// markExcluded pose l'exclusion d'un match, par l'INSERT d'une ligne d'enrichissement de
// l'étape qui porte is_excluded (la vue _latest ne lit la colonne que de cette étape).
func markExcluded(t *testing.T, pdb *PlayerDB, matchID string) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, is_excluded, stage) VALUES (?, TRUE, 'exclusion')`,
		matchID); err != nil {
		t.Fatalf("insert exclusion %s: %v", matchID, err)
	}
}

func newLUSRCareerPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	return &PlayerDB{
		Player: openMigratedFile(t, "stats.duckdb", migration.TargetPlayer),
		Shared: openMigratedFile(t, "shared.duckdb", migration.TargetShared),
		XUID:   "xuid_lusr_exclusion",
	}
}

func TestCareerRepo_GetLUSRHistory_DropsExcludedMatch(t *testing.T) {
	pdb := newLUSRCareerPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedLUSRCareerMatch(t, pdb, "m1", 1000, day)
	seedLUSRCareerMatch(t, pdb, "m2", 1100, day.Add(24*time.Hour))
	seedLUSRCareerMatch(t, pdb, "m3", 1300, day.Add(48*time.Hour))
	markExcluded(t, pdb, "m2")

	got, err := NewCareerRepo(pdb).GetLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("GetLUSRHistory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("points = %d (%+v), want 2 (m2 exclu)", len(got), got)
	}
	if got[0].MatchID != "m1" || got[1].MatchID != "m3" {
		t.Fatalf("matchs servis = %s, %s ; want m1, m3", got[0].MatchID, got[1].MatchID)
	}
	if got[0].RatingDelta != nil {
		t.Errorf("m1 : delta = %v, want nil (premier point retenu)", *got[0].RatingDelta)
	}
	if got[1].RatingDelta == nil || *got[1].RatingDelta != 300 {
		t.Errorf("m3 : delta = %v, want 300 (écart avec m1, pas avec le match exclu m2 : 200)", got[1].RatingDelta)
	}
}

func TestCareerRepo_GetLUSRHistory_AllMatchesExcluded(t *testing.T) {
	pdb := newLUSRCareerPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedLUSRCareerMatch(t, pdb, "m1", 1000, day)
	markExcluded(t, pdb, "m1")

	got, err := NewCareerRepo(pdb).GetLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("GetLUSRHistory: %v", err)
	}
	if got != nil {
		t.Fatalf("historique = %+v, want nil quand plus aucun match n'est retenu", got)
	}
}
