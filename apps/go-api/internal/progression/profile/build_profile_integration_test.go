//go:build integration

// build_profile_integration_test.go — tests d'intégration de BuildProfile()
// avec DuckDB in-memory.

package profile

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	testXUID  = "xuid-profile-test"
	testGT    = "ProfileTester"
	testTitle = "halo_infinite"
)

// setupProfileEnv crée un PlayerDB minimal pour exercer BuildProfile :
//   - Player (stats.duckdb) avec shared attaché en RW
//   - Metadata (metadata.duckdb) pour les templates (catalogue vide en test
//     simple, ou seeded pour tester les suggestions)
//
// Pattern aligné sur post_sync_progression_test.go.setupProgressionEnv.
func setupProfileEnv(t *testing.T) *duckdb.PlayerDB {
	t.Helper()
	dir := t.TempDir()

	openAndMigrate := func(name string, target migration.TargetDB) *duckdb.DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		raw.Close()
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		return db
	}

	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	{
		raw, err := sql.Open("duckdb", sharedPath)
		if err != nil {
			t.Fatalf("open shared: %v", err)
		}
		if err := migration.RunForDB(raw, migration.TargetShared); err != nil {
			raw.Close()
			t.Fatalf("migrate shared: %v", err)
		}
		raw.Close()
	}

	meta := openAndMigrate("metadata", migration.TargetMetadata)
	player := openAndMigrate("stats", migration.TargetPlayer)

	ctx := context.Background()
	if _, err := player.Exec(ctx, "ATTACH '"+sharedPath+"' AS shared"); err != nil {
		t.Fatalf("attach shared: %v", err)
	}

	pdb := &duckdb.PlayerDB{
		Player: player, Metadata: meta,
		XUID: testXUID, Gamertag: testGT, TitleSlug: testTitle,
	}
	t.Cleanup(func() {
		player.Close()
		meta.Close()
	})
	return pdb
}

func seedMatchesForProfile(t *testing.T, pdb *duckdb.PlayerDB, now time.Time, count int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < count; i++ {
		matchID := zeropadProfile(i, 4)
		startTime := now.AddDate(0, 0, -i)
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO shared.match_registry (match_id, start_time)
			VALUES (?, ?)
		`, matchID, startTime); err != nil {
			t.Fatalf("match_registry %s: %v", matchID, err)
		}
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO shared.match_participants (
				match_id, xuid, gamertag, team_id, outcome,
				kills, deaths, assists, personal_score, max_killing_spree,
				time_played_seconds
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, matchID, testXUID, testGT, 1, 2, 12+i%4, 8, 3, 1500, 5, 600); err != nil {
			t.Fatalf("match_participants %s: %v", matchID, err)
		}
		mu := 1500.0 + float64(i%20)
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, start_time)
			VALUES (?, 'LUSR', ?, 120, ?)
		`, matchID, mu, startTime); err != nil {
			t.Fatalf("match_skill_rank %s: %v", matchID, err)
		}
	}
}

func zeropadProfile(n, w int) string {
	out := make([]byte, w)
	for i := w - 1; i >= 0; i-- {
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out)
}

// TestBuildProfile_EmptyDB_HasEnoughDataFalse vérifie le cas dégradé.
func TestBuildProfile_EmptyDB_HasEnoughDataFalse(t *testing.T) {
	pdb := setupProfileEnv(t)
	svc := NewServiceFromPlayerDB(pdb)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 30, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if prof.HasEnoughData {
		t.Errorf("HasEnoughData on empty DB: got true, want false")
	}
	if prof.MatchesAnalyzed != 0 {
		t.Errorf("MatchesAnalyzed: got %d, want 0", prof.MatchesAnalyzed)
	}
}

// TestBuildProfile_WithMatches_FillsSections vérifie qu'au-dessus de
// MinMatchesForProfile, les sections sont remplies.
func TestBuildProfile_WithMatches_FillsSections(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForProfile+5)

	svc := NewServiceFromPlayerDB(pdb)
	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if !prof.HasEnoughData {
		t.Errorf("HasEnoughData: want true, got false (matches=%d)", prof.MatchesAnalyzed)
	}
	if prof.MatchesAnalyzed < MinMatchesForProfile {
		t.Errorf("MatchesAnalyzed: got %d, want >= %d", prof.MatchesAnalyzed, MinMatchesForProfile)
	}
	if len(prof.RadarAxes) == 0 {
		t.Error("RadarAxes: empty")
	}
	if prof.LUSR.Mu == 0 {
		t.Error("LUSR.Mu: zero")
	}
	if prof.Tier.IsEmpty() {
		t.Error("Tier: empty")
	}
	if prof.EngagementSnap.Tier == "" {
		t.Error("EngagementSnap.Tier: empty")
	}
}

// TestLoad_V2Compat vérifie que Load() reste compatible avec le caller V2
// (post_sync_progression.go). MinMatchesForRating=10 → 10+ matchs requis.
func TestLoad_V2Compat(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForRating+5)

	svc := NewService(pdb.Player) // V2 path : constructor minimal
	prof, err := svc.Load(context.Background(), testXUID, testTitle, 30, now)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if prof.LUSR.MatchesCount < MinMatchesForRating {
		t.Errorf("MatchesCount: got %d, want >= %d", prof.LUSR.MatchesCount, MinMatchesForRating)
	}
	if prof.Tier.IsEmpty() {
		t.Error("Tier: empty after Load() with enough matches")
	}
}
