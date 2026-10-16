//go:build integration

package duckdb

import (
	"context"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain"
)

// maxLimitAllocBytes : budget d'allocation d'un appel au repo avec une limite
// démesurée. Une entrée occupe plusieurs centaines d'octets : allouer 1<<20 entrées
// d'avance en consommerait des centaines de Mio.
const maxLimitAllocBytes = 32 << 20

// allocatedBy mesure les octets alloués (cumul du tas) par fn.
func allocatedBy(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// Une limite démesurée ne dimensionne aucune allocation : le repo la plafonne à
// domain.MaxLeaderboardLimit avant d'allouer.
func TestLeaderboardRepo_HugeLimitDoesNotSizeAllocations(t *testing.T) {
	const huge = 1 << 20
	ctx := context.Background()

	t.Run("csr-world", func(t *testing.T) {
		shared := openMemDB(t)
		applyWorldLeaderboardMigration(t, shared.SQLDb())
		if _, err := shared.SQLDb().ExecContext(ctx, `
			INSERT INTO world_csr_leaderboard_snapshots
				(season_id, playlist_id, rank, gamertag, csr_value, title_slug, fetched_at)
			VALUES ('s1','pl1',1,'P1',2000,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00')`); err != nil {
			t.Fatalf("insert snapshot: %v", err)
		}
		repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
		var got []domain.LeaderboardEntry
		var err error
		alloc := allocatedBy(func() {
			got, err = repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "s1", "pl1", huge)
		})
		if err != nil || len(got) != 1 {
			t.Fatalf("GetCSRWorldLeaderboard : %d entrées, err=%v", len(got), err)
		}
		if alloc > maxLimitAllocBytes {
			t.Errorf("limit=%d : %d octets alloués, budget %d", huge, alloc, maxLimitAllocBytes)
		}
	})

	t.Run("stats", func(t *testing.T) {
		shared := openMemDB(t)
		seedStatLeaderboardData(t, shared)
		repo := NewLeaderboardRepo(&PlayerDB{Shared: shared, XUID: "xa"})
		var got []domain.LeaderboardEntry
		var err error
		alloc := allocatedBy(func() {
			got, err = repo.GetStatLeaderboard(ctx, "halo_infinite", domain.LeaderboardKills, "", "", huge)
		})
		if err != nil || len(got) != 2 {
			t.Fatalf("GetStatLeaderboard : %d entrées, err=%v", len(got), err)
		}
		if alloc > maxLimitAllocBytes {
			t.Errorf("limit=%d : %d octets alloués, budget %d", huge, alloc, maxLimitAllocBytes)
		}
	})
}
