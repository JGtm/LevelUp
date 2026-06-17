//go:build integration

package duckdb

// leaderboard_title_isolation_test.go — PMT-7 oracle (b) : GetCSRWorldLeaderboard
// filtre RÉELLEMENT par title_slug — deux titres dans la même table ne fuient pas
// l'un dans l'autre. Prouve que le seam route (pas juste un paramètre cosmétique).

import (
	"context"
	"testing"
)

func TestGetCSRWorldLeaderboard_TitleIsolation(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	// Deux titres, même season/playlist/rang — seul title_slug les distingue.
	if _, err := shared.SQLDb().ExecContext(ctx, `
		INSERT INTO world_csr_leaderboard_snapshots
			(season_id, playlist_id, rank, gamertag, csr_value, title_slug, fetched_at)
		VALUES
			('s1','pl1',1,'HaloGuy',2000,'halo_infinite',     TIMESTAMP '2026-01-01 00:00:00'),
			('s1','pl1',1,'SynthGuy',1500,'synthetic_title_b', TIMESTAMP '2026-01-01 00:00:00');
	`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})

	halo, err := repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "s1", "pl1", 10)
	if err != nil {
		t.Fatalf("halo: %v", err)
	}
	if len(halo) != 1 || halo[0].Gamertag != "HaloGuy" {
		t.Errorf("halo_infinite → %+v, want 1 entrée (HaloGuy) — fuite cross-titre", halo)
	}

	synth, err := repo.GetCSRWorldLeaderboard(ctx, "synthetic_title_b", "s1", "pl1", 10)
	if err != nil {
		t.Fatalf("synthetic: %v", err)
	}
	if len(synth) != 1 || synth[0].Gamertag != "SynthGuy" {
		t.Errorf("synthetic_title_b → %+v, want 1 entrée (SynthGuy) — le filtre title_slug ne route pas", synth)
	}
}
