//go:build integration

// explorer_repo_recent_test.go — tests d'intégration ExplorerRepo.GetTargetRecentMatches
// (profil de combat Explorer). Valide : filtre PvP via is_firefight, ordre
// start_time DESC, cap LIMIT, perfect_kills via LEFT JOIN (COALESCE 0), rank NULL
// pour DNF (*int nil), cast damage DOUBLE→int.

package duckdb

import (
	"context"
	"testing"
)

const recentTgtXUID = "tgt-recent-xuid"

// seedRecentMatch insère un match_registry + un match_participants pour la cible.
func seedRecentMatch(t *testing.T, pdb *PlayerDB, ctx context.Context,
	matchID, startUTC, mapName, pairName string, isFirefight bool,
	outcome, kills, deaths, assists, score, dmgDealt, dmgTaken, spree int, rank interface{},
) {
	t.Helper()
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.match_registry (match_id, start_time_utc, map_name, pair_name, is_firefight)
		 VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)`,
		matchID, startUTC, mapName, pairName, isFirefight,
	); err != nil {
		t.Fatalf("seed registry %s: %v", matchID, err)
	}
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.match_participants
		   (match_id, xuid, outcome, kills, deaths, assists, kda, personal_score,
		    damage_dealt, damage_taken, max_killing_spree, rank)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		matchID, recentTgtXUID, outcome, kills, deaths, assists,
		float64(kills+assists)/2.0, score,
		float64(dmgDealt), float64(dmgTaken), spree, rank,
	); err != nil {
		t.Fatalf("seed participant %s: %v", matchID, err)
	}
}

func seedRecentDataset(t *testing.T, pdb *PlayerDB, ctx context.Context) {
	t.Helper()
	// mr1 = plus récent (win, rank 1) + 2 perfect kills.
	seedRecentMatch(t, pdb, ctx, "mr1", "2025-03-10 18:00:00+00", "Aquarius", "Slayer", false,
		2, 20, 8, 5, 1800, 4500, 3000, 7, 1)
	// mr2 = intermédiaire (loss, rank 5), AUCUNE médaille perfect.
	seedRecentMatch(t, pdb, ctx, "mr2", "2025-03-09 18:00:00+00", "Recharge", "Oddball", false,
		3, 10, 12, 3, 1200, 3000, 3500, 3, 5)
	// mr3 = plus ancien (DNF, rank NULL).
	seedRecentMatch(t, pdb, ctx, "mr3", "2025-03-08 18:00:00+00", "Streets", "CTF", false,
		4, 4, 6, 1, 600, 1500, 1800, 1, nil)
	// mff = Firefight (is_firefight TRUE) → doit être EXCLU.
	seedRecentMatch(t, pdb, ctx, "mff", "2025-03-11 18:00:00+00", "Firebase", "Firefight", true,
		2, 50, 2, 0, 5000, 9000, 1000, 20, 1)

	// 2 perfect-kill medals sur mr1 ; rien sur mr2/mr3 (valide LEFT JOIN / COALESCE 0).
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.medals_earned (medal_name_id, xuid, match_id, count) VALUES (?, ?, ?, ?)`,
		uint64(1512363953), recentTgtXUID, "mr1", 2,
	); err != nil {
		t.Fatalf("seed perfect medal: %v", err)
	}
}

func TestExplorerRepo_GetTargetRecentMatches_PvPOrderingAndPerfect(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	seedRecentDataset(t, pdb, ctx)

	repo := NewExplorerRepo(pdb, pTestXUID)
	rows, err := repo.GetTargetRecentMatches(ctx, recentTgtXUID, 20)
	if err != nil {
		t.Fatalf("GetTargetRecentMatches: %v", err)
	}

	// mff (Firefight) exclu → 3 matchs PvP.
	if len(rows) != 3 {
		t.Fatalf("len = %d, want 3 (Firefight exclu)", len(rows))
	}
	// Ordre start_time DESC : mr1, mr2, mr3.
	if rows[0].MatchID != "mr1" || rows[1].MatchID != "mr2" || rows[2].MatchID != "mr3" {
		t.Errorf("ordre = [%s,%s,%s], want [mr1,mr2,mr3]", rows[0].MatchID, rows[1].MatchID, rows[2].MatchID)
	}
	// mr1 : perfect_kills=2, rank=1, damage cast int, mode/map remontés.
	if rows[0].PerfectKills != 2 {
		t.Errorf("mr1 perfect_kills = %d, want 2", rows[0].PerfectKills)
	}
	if rows[0].Rank == nil || *rows[0].Rank != 1 {
		t.Errorf("mr1 rank = %v, want 1", rows[0].Rank)
	}
	if rows[0].DamageDealt != 4500 || rows[0].DamageTaken != 3000 {
		t.Errorf("mr1 damage = %d/%d, want 4500/3000", rows[0].DamageDealt, rows[0].DamageTaken)
	}
	if rows[0].ModeUI != "Slayer" || rows[0].MapUI != "Aquarius" {
		t.Errorf("mr1 mode/map = %q/%q, want Slayer/Aquarius", rows[0].ModeUI, rows[0].MapUI)
	}
	// mr2 : aucune médaille perfect → 0 via LEFT JOIN/COALESCE.
	if rows[1].PerfectKills != 0 {
		t.Errorf("mr2 perfect_kills = %d, want 0 (LEFT JOIN COALESCE)", rows[1].PerfectKills)
	}
	// mr3 : DNF → rank NULL → *int nil.
	if rows[2].Rank != nil {
		t.Errorf("mr3 rank = %v, want nil (DNF)", *rows[2].Rank)
	}
}

// TestExplorerRepo_GetTargetRecentMatches_H5AssetFallback vérifie le fallback
// libellés via asset_translations pour un match dont map_name/pair_name sont NULL
// (cas Halo 5 : 100 % du registre) mais qui porte map_id + game_variant_id.
// L'Explorer affichait des lignes vides ; map/mode doivent maintenant être résolus.
func TestExplorerRepo_GetTargetRecentMatches_H5AssetFallback(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// map_name / pair_name absents (NULL) ; map_id + game_variant_id présents.
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.match_registry (match_id, start_time_utc, map_id, game_variant_id, is_firefight)
		 VALUES (?, ?::TIMESTAMPTZ, ?, ?, ?)`,
		"h5m1", "2025-03-10 18:00:00+00", "d67fdcb9-map", "257a305e-gv", false,
	); err != nil {
		t.Fatalf("seed registry h5m1: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.match_participants
		   (match_id, xuid, outcome, kills, deaths, assists, kda, personal_score,
		    damage_dealt, damage_taken, max_killing_spree, rank)
		 VALUES (?, ?, 2, 15, 6, 4, 12.0, 1500, 4000.0, 2500.0, 5, 1)`,
		"h5m1", recentTgtXUID,
	); err != nil {
		t.Fatalf("seed participant h5m1: %v", err)
	}
	// asset_translations fr-FR pour la map et le game_variant (mode).
	for _, ins := range [][]any{
		{"d67fdcb9-map", assetTypeMap, "fr-FR", "Tidal"},
		{"257a305e-gv", assetTypeGameVariant, "fr-FR", "Assassin"},
	} {
		if _, err := pdb.Metadata.Exec(ctx,
			`INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
			 VALUES (?, ?, ?, ?, '', now())`, ins...); err != nil {
			t.Fatalf("seed asset_translations: %v", err)
		}
	}

	repo := NewExplorerRepo(pdb, pTestXUID)
	rows, err := repo.GetTargetRecentMatches(ctx, recentTgtXUID, 20)
	if err != nil {
		t.Fatalf("GetTargetRecentMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len = %d, want 1", len(rows))
	}
	if rows[0].MapUI != "Tidal" {
		t.Errorf("map_ui = %q, want Tidal (résolu via asset_translations map_id)", rows[0].MapUI)
	}
	if rows[0].ModeUI != "Assassin" {
		t.Errorf("mode_ui = %q, want Assassin (résolu via asset_translations game_variant)", rows[0].ModeUI)
	}
}

func TestExplorerRepo_GetTargetRecentMatches_LimitAndGuards(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	seedRecentDataset(t, pdb, ctx)
	repo := NewExplorerRepo(pdb, pTestXUID)

	// LIMIT 2 → les 2 plus récents PvP.
	rows, err := repo.GetTargetRecentMatches(ctx, recentTgtXUID, 2)
	if err != nil {
		t.Fatalf("GetTargetRecentMatches limit=2: %v", err)
	}
	if len(rows) != 2 || rows[0].MatchID != "mr1" || rows[1].MatchID != "mr2" {
		t.Errorf("limit=2 → %d rows, want [mr1,mr2]", len(rows))
	}

	// Gardes : xuid vide / limit<=0 → nil sans erreur.
	if r, err := repo.GetTargetRecentMatches(ctx, "", 20); err != nil || r != nil {
		t.Errorf("xuid vide → (%v,%v), want (nil,nil)", r, err)
	}
	if r, err := repo.GetTargetRecentMatches(ctx, recentTgtXUID, 0); err != nil || r != nil {
		t.Errorf("limit=0 → (%v,%v), want (nil,nil)", r, err)
	}

	// xuid sans match → slice vide (pas d'erreur).
	if r, err := repo.GetTargetRecentMatches(ctx, "unknown-xuid", 20); err != nil || len(r) != 0 {
		t.Errorf("xuid inconnu → (%d rows,%v), want (0,nil)", len(r), err)
	}
}
