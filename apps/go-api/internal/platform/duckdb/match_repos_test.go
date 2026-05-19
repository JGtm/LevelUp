//go:build integration

// Package duckdb — match_repos_test.go : tests FiltersRepo, MatchHistoryRepo,
// CitationsRepo, ExplorerRepo, MatchViewRepo, SquadRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// FiltersRepo
// ---------------------------------------------------------------------------

func TestFiltersRepo_LoadMatchesForFilters_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewFiltersRepo(pdb)
	rows, err := repo.LoadMatchesForFilters(context.Background())
	if err != nil {
		t.Fatalf("LoadMatchesForFilters: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
	if rows[0].MatchID != "m1" {
		t.Errorf("match_id = %q, want m1", rows[0].MatchID)
	}
}

func TestFiltersRepo_LoadMatchesForFilters_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, "DELETE FROM shared.match_participants")
	repo := NewFiltersRepo(pdb)
	rows, err := repo.LoadMatchesForFilters(ctx)
	if err != nil {
		t.Fatalf("LoadMatchesForFilters empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestFiltersRepo_GetMatchCount(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewFiltersRepo(pdb)
	count, err := repo.GetMatchCount(context.Background())
	if err != nil {
		t.Fatalf("GetMatchCount: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1, obtenu %d", count)
	}
}

func TestFiltersRepo_GetPlayerMatchCount(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewFiltersRepo(pdb)
	count, err := repo.GetPlayerMatchCount(context.Background())
	if err != nil {
		t.Fatalf("GetPlayerMatchCount: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1, obtenu %d", count)
	}
}

func TestFiltersRepo_GetAvailablePlaylists(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewFiltersRepo(pdb)
	playlists, err := repo.GetAvailablePlaylists(context.Background())
	if err != nil {
		t.Fatalf("GetAvailablePlaylists: %v", err)
	}
	if len(playlists) != 1 {
		t.Errorf("attendu 1 playlist, obtenu %d", len(playlists))
	}
}

func TestFiltersRepo_GetAvailableMaps(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewFiltersRepo(pdb)
	maps, err := repo.GetAvailableMaps(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableMaps: %v", err)
	}
	if len(maps) != 1 {
		t.Errorf("attendu 1 carte, obtenu %d", len(maps))
	}
}

// ---------------------------------------------------------------------------
// MatchHistoryRepo
// ---------------------------------------------------------------------------

func TestMatchHistoryRepo_LoadAll_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, "DELETE FROM shared.match_participants")
	repo := NewMatchHistoryRepo(pdb)
	rows, err := repo.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestMatchHistoryRepo_LoadAll_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchHistoryRepo(pdb)
	rows, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestMatchHistoryRepo_LoadMapWinRates(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchHistoryRepo(pdb)
	rates, err := repo.LoadMapWinRates(context.Background())
	if err != nil {
		t.Fatalf("LoadMapWinRates: %v", err)
	}
	if len(rates) != 1 {
		t.Errorf("attendu 1 carte, obtenu %d", len(rates))
	}
}

// ---------------------------------------------------------------------------
// CitationsRepo
// ---------------------------------------------------------------------------

func TestCitationsRepo_LoadCitationMappings_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCitationsRepo(pdb)
	rows, err := repo.LoadCitationMappings(context.Background())
	if err != nil {
		t.Fatalf("LoadCitationMappings: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestCitationsRepo_LoadCitationTotals_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCitationsRepo(pdb)
	rows, err := repo.LoadCitationTotals(context.Background())
	if err != nil {
		t.Fatalf("LoadCitationTotals: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

func TestCitationsRepo_LoadMedalTotals_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCitationsRepo(pdb)
	rows, err := repo.LoadMedalTotals(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadMedalTotals: %v", err)
	}
	// shared.medals_earned vide → 0 résultats
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// ExplorerRepo
// ---------------------------------------------------------------------------

func TestExplorerRepo_ResolveXUIDByGamertag_Found(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewExplorerRepo(pdb, pTestXUID)
	xuid, err := repo.ResolveXUIDByGamertag(context.Background(), pTestGamertag)
	if err != nil {
		t.Fatalf("ResolveXUIDByGamertag: %v", err)
	}
	if xuid != pTestXUID {
		t.Errorf("xuid = %q, want %q", xuid, pTestXUID)
	}
}

func TestExplorerRepo_GetCommonMatches_Same(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewExplorerRepo(pdb, pTestXUID)
	// m1 a uniquement pTestXUID → pas de matchs communs avec un autre xuid
	rows, err := repo.GetCommonMatches(context.Background(), pTestXUID, "xuid_other_999")
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 matchs communs, obtenu %d", len(rows))
	}
}

func TestExplorerRepo_GetCommonMatches_WithSharedMatch(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Ajouter un second joueur sur m1
	_, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.match_participants
		 (match_id,xuid,gamertag,outcome,kills,deaths,team_id) VALUES (?,?,?,?,?,?,?)`,
		"m1", "xuid_other_999", "OtherPlayer", 2, 5, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewExplorerRepo(pdb, pTestXUID)
	rows, err := repo.GetCommonMatches(ctx, pTestXUID, "xuid_other_999")
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 match commun, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// MatchViewRepo
// ---------------------------------------------------------------------------

func TestMatchViewRepo_GetMatchMeta_Found(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	meta, err := repo.GetMatchMeta(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.MatchID != "m1" {
		t.Errorf("match_id = %q, want m1", meta.MatchID)
	}
}

func TestMatchViewRepo_GetMatchMeta_NotFound(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	_, err := repo.GetMatchMeta(context.Background(), "nonexistent")
	if err == nil {
		t.Error("attendu une erreur pour match inexistant")
	}
}

func TestMatchViewRepo_GetPlayerMatchStats_NotFound(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	// Joueur absent → stats vides (pas d'erreur selon le code)
	stats, err := repo.GetPlayerMatchStats(context.Background(), "xuid_absent", "m1")
	if err != nil {
		t.Fatalf("GetPlayerMatchStats absent: %v", err)
	}
	if stats == nil {
		t.Error("stats ne doit pas être nil")
	}
}

func TestMatchViewRepo_GetMatchEnrichment_NotFound(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM player_match_enrichment"); err != nil {
		t.Fatal(err)
	}
	repo := NewMatchViewRepo(pdb, pTestXUID)
	// Sans enrichissement → retourne struct vide, pas d'erreur
	enrichment, err := repo.GetMatchEnrichment(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchEnrichment absent: %v", err)
	}
	if enrichment == nil {
		t.Error("enrichment ne doit pas être nil")
	}
}

func TestMatchViewRepo_GetMatchSkillRank_ClassifiesRankedMatchAsCSR(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `UPDATE match_skill_rank SET rating_type = 'LUSR' WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("UPDATE match_skill_rank: %v", err)
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	rank, err := repo.GetMatchSkillRank(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchSkillRank: %v", err)
	}
	if rank == nil {
		t.Fatal("expected non-nil rank")
	}
	if rank.RatingType != "CSR" {
		t.Fatalf("RatingType = %q, want CSR", rank.RatingType)
	}
	if rank.TierLabel == nil || *rank.TierLabel != "Gold 3" {
		t.Fatalf("TierLabel = %v, want Gold 3", rank.TierLabel)
	}
	if rank.RatingValue == nil || *rank.RatingValue != 1250.5 {
		t.Fatalf("RatingValue = %v, want 1250.5", rank.RatingValue)
	}
}

func TestMatchViewRepo_GetMatchSkillRank_InfersCSRFromRankedPlaylistName(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `UPDATE match_skill_rank SET rating_type = 'LUSR' WHERE match_id = ?`, "m1"); err != nil {
		t.Fatalf("UPDATE match_skill_rank: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx, `
		UPDATE shared.match_registry
		SET is_ranked = FALSE,
			playlist_name = 'Ranked Arena',
			pair_name = 'Arena'
		WHERE match_id = ?
	`, "m1"); err != nil {
		t.Fatalf("UPDATE shared.match_registry: %v", err)
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	rank, err := repo.GetMatchSkillRank(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchSkillRank: %v", err)
	}
	if rank == nil {
		t.Fatal("expected non-nil rank")
	}
	if rank.RatingType != "CSR" {
		t.Fatalf("RatingType = %q, want CSR", rank.RatingType)
	}
}

func TestMatchViewRepo_GetMatchScoreboard_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	rows, err := repo.GetMatchScoreboard(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchScoreboard: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 joueur, obtenu %d", len(rows))
	}
}

func TestMatchViewRepo_GetMatchMedals_WithMetadataLabels(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	_, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (?,?,?,?,?)`,
		uint64(1001), uint64(1001), pTestXUID, "m1", 2,
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	rows, err := repo.GetMatchMedals(ctx, pTestXUID, "m1")
	if err != nil {
		t.Fatalf("GetMatchMedals: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 médaille, obtenu %d", len(rows))
	}
	if rows[0].Label != "Killing Spree" {
		t.Errorf("label = %q, want %q", rows[0].Label, "Killing Spree")
	}
}

func TestMatchViewRepo_GetMatchWeaponKills_WithMetadataLabels(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Q16WeaponKills agrege via COUNT(*) sur shared.v_weapon_kills :
	// chaque row de la table weapon_kills represente 1 kill effectif.
	for i := 0; i < 4; i++ {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.weapon_kills (match_id, xuid, weapon_id) VALUES (?,?,?)`,
			"m1", pTestXUID, uint64(42),
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	rows, err := repo.GetMatchWeaponKills(ctx, pTestXUID, "m1")
	if err != nil {
		t.Fatalf("GetMatchWeaponKills: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 arme, obtenu %d", len(rows))
	}
	if rows[0].WeaponLabel != "BR75" {
		t.Errorf("weapon_label = %q, want %q", rows[0].WeaponLabel, "BR75")
	}
	if rows[0].Kills != 4 {
		t.Errorf("kills = %d, want 4", rows[0].Kills)
	}
}

// ---------------------------------------------------------------------------
// SquadRepo
// ---------------------------------------------------------------------------

func TestSquadRepo_LoadTopTeammates_NoFriends(t *testing.T) {
	pdb := newTestPlayerDB(t)
	// is_with_friends = FALSE → Q29 filtre sur pme.is_with_friends = TRUE → 0 résultats
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadTopTeammates(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadTopTeammates: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 coéquipiers, obtenu %d", len(rows))
	}
}

func TestSquadRepo_LoadSquadMatches_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	// Pas de match en commun avec "xuid_other"
	rows, err := repo.LoadSquadMatches(context.Background(), pTestXUID, "xuid_other_999")
	if err != nil {
		t.Fatalf("LoadSquadMatches: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 matchs, obtenu %d", len(rows))
	}
}

func TestSquadRepo_LoadSquadMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants
		 (match_id,xuid,gamertag,outcome,kills,deaths,team_id,kda,accuracy,time_played_seconds,team_mmr)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"m1", "xuid_mate_002", "TeamMate", 2, 8, 4, 1, 1.2, 0.55, 600, 1200.0)
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadSquadMatches(ctx, pTestXUID, "xuid_mate_002")
	if err != nil {
		t.Fatalf("LoadSquadMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1 match commun, obtenu %d", len(rows))
	}
}
