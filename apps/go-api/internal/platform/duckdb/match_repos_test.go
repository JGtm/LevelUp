//go:build integration

// Package duckdb — match_repos_test.go : tests FiltersRepo, MatchHistoryRepo,
// CitationsRepo, ExplorerRepo, MatchViewRepo, SquadRepo.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
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

// Le nom de citation est résolu selon la locale du contexte : EN →
// citation_name_display_en, FR (défaut) → citation_name_display. Câble la
// traduction des citations Infinite (copies de commendations H5).
func TestCitationsRepo_LoadCitationMappings_LocaleAware(t *testing.T) {
	repo := NewCitationsRepo(newTestPlayerDB(t))

	fr, err := repo.LoadCitationMappings(context.Background())
	if err != nil || len(fr) != 1 {
		t.Fatalf("FR: err=%v n=%d", err, len(fr))
	}
	if fr[0].NameDisplay != "Killing Spree" {
		t.Errorf("FR NameDisplay = %q, want 'Killing Spree'", fr[0].NameDisplay)
	}

	en, err := repo.LoadCitationMappings(ctxkeys.WithLocale(context.Background(), "en"))
	if err != nil || len(en) != 1 {
		t.Fatalf("EN: err=%v n=%d", err, len(en))
	}
	if en[0].NameDisplay != "Killing Spree (EN)" {
		t.Errorf("EN NameDisplay = %q, want 'Killing Spree (EN)'", en[0].NameDisplay)
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

func TestExplorerRepo_GetParticipantStatsForMatches(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	const targetXUID = "xuid_target_555"
	// Insert 3 lignes participant pour le target sur m1, m2 (m3 non listé).
	// outcome : 2=win, 3=loss, 1=draw.
	inserts := []struct {
		matchID                         string
		outcome, kills, deaths, assists int
		shotsFired, shotsHit            int
		dmgDealt, dmgTaken              float64
		hsKills, melee, power, grenade  int
	}{
		{"m1", 2, 15, 5, 3, 100, 50, 1800, 1200, 5, 1, 3, 2},
		{"m2", 3, 8, 12, 1, 80, 30, 1100, 1500, 2, 0, 1, 0},
	}
	for _, in := range inserts {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.match_participants
			 (match_id, xuid, gamertag, outcome, kills, deaths, assists,
			  shots_fired, shots_hit, damage_dealt, damage_taken,
			  headshot_kills, melee_kills, power_weapon_kills, grenade_kills, team_id)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			in.matchID, targetXUID, "TargetPlayer", in.outcome,
			in.kills, in.deaths, in.assists,
			in.shotsFired, in.shotsHit, in.dmgDealt, in.dmgTaken,
			in.hsKills, in.melee, in.power, in.grenade, 1)
		if err != nil {
			t.Fatalf("insert %s: %v", in.matchID, err)
		}
	}
	repo := NewExplorerRepo(pdb, pTestXUID)

	t.Run("agrégat sur 2 matchs présents", func(t *testing.T) {
		agg, err := repo.GetParticipantStatsForMatches(ctx, targetXUID, []string{"m1", "m2"})
		if err != nil {
			t.Fatalf("GetParticipantStatsForMatches: %v", err)
		}
		if agg == nil {
			t.Fatal("agg attendu non-nil")
		}
		if agg.Kills != 23 || agg.Deaths != 17 || agg.Assists != 4 {
			t.Errorf("K/D/A = %d/%d/%d, want 23/17/4", agg.Kills, agg.Deaths, agg.Assists)
		}
		if agg.Wins != 1 || agg.Losses != 1 || agg.Draws != 0 {
			t.Errorf("W/L/D = %d/%d/%d, want 1/1/0", agg.Wins, agg.Losses, agg.Draws)
		}
		if agg.ShotsFired != 180 || agg.ShotsHit != 80 {
			t.Errorf("shots = %d/%d, want 180/80", agg.ShotsFired, agg.ShotsHit)
		}
		if agg.DamageDealt != 2900 || agg.DamageTaken != 2700 {
			t.Errorf("damage = %.0f/%.0f, want 2900/2700", agg.DamageDealt, agg.DamageTaken)
		}
		if agg.HeadshotKills != 7 || agg.MeleeKills != 1 || agg.PowerWeaponKills != 4 || agg.GrenadeKills != 2 {
			t.Errorf("kill types = HS:%d Me:%d Pwr:%d Gr:%d, want 7/1/4/2",
				agg.HeadshotKills, agg.MeleeKills, agg.PowerWeaponKills, agg.GrenadeKills)
		}
	})

	t.Run("matchIDs vide → nil", func(t *testing.T) {
		agg, err := repo.GetParticipantStatsForMatches(ctx, targetXUID, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if agg != nil {
			t.Errorf("attendu nil pour matchIDs vide, got %+v", agg)
		}
	})

	t.Run("xuid sans participants → agrégat zéro", func(t *testing.T) {
		agg, err := repo.GetParticipantStatsForMatches(ctx, "xuid_inconnu", []string{"m1", "m2"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		// SUM retourne 0 par défaut (pas une row absente), donc l'aggregate est non-nil avec des zéros.
		if agg == nil {
			t.Fatal("attendu non-nil (zéros), got nil")
		}
		if agg.Kills != 0 || agg.Deaths != 0 {
			t.Errorf("xuid inconnu doit retourner 0/0, got %d/%d", agg.Kills, agg.Deaths)
		}
	})

	t.Run("filtrage match_ids respecté", func(t *testing.T) {
		// Demande uniquement m1.
		agg, err := repo.GetParticipantStatsForMatches(ctx, targetXUID, []string{"m1"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if agg.Kills != 15 {
			t.Errorf("filtré sur m1 seul, attendu Kills=15, got %d", agg.Kills)
		}
	})
}

func TestExplorerRepo_GetMedalCountsForMatches(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	const targetXUID = "xuid_target_555"
	// Insert dans shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count).
	medals := []struct {
		medalNameID uint64
		matchID     string
		count       int
	}{
		{100, "m1", 3},  // medal 100 sur m1 ×3
		{200, "m1", 2},  // medal 200 sur m1 ×2
		{100, "m2", 1},  // medal 100 sur m2 ×1 (même type → unique pour total)
		{300, "m2", 5},  // medal 300 sur m2 ×5
		{999, "m3", 10}, // medal 999 sur m3 — exclu si on filtre m1/m2
	}
	for _, m := range medals {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (?,?,?,?,?)`,
			m.medalNameID, m.medalNameID, targetXUID, m.matchID, m.count)
		if err != nil {
			t.Fatalf("insert medal: %v", err)
		}
	}
	repo := NewExplorerRepo(pdb, pTestXUID)

	t.Run("agrégat sur 2 matchs : total+unique corrects, m3 exclu", func(t *testing.T) {
		agg, err := repo.GetMedalCountsForMatches(ctx, targetXUID, []string{"m1", "m2"})
		if err != nil {
			t.Fatalf("GetMedalCountsForMatches: %v", err)
		}
		if agg == nil {
			t.Fatal("agg attendu non-nil")
		}
		// Total = 3+2+1+5 = 11. Unique = {100,200,300} = 3.
		if agg.Total != 11 {
			t.Errorf("Total = %d, want 11", agg.Total)
		}
		if agg.Unique != 3 {
			t.Errorf("Unique = %d, want 3 (medals 100/200/300, m3 exclu)", agg.Unique)
		}
	})

	t.Run("matchIDs vide → nil", func(t *testing.T) {
		agg, err := repo.GetMedalCountsForMatches(ctx, targetXUID, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if agg != nil {
			t.Errorf("attendu nil pour matchIDs vide, got %+v", agg)
		}
	})

	t.Run("xuid sans médailles → zéros", func(t *testing.T) {
		agg, err := repo.GetMedalCountsForMatches(ctx, "xuid_inconnu", []string{"m1", "m2"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if agg == nil {
			t.Fatal("attendu non-nil (zéros)")
		}
		if agg.Total != 0 || agg.Unique != 0 {
			t.Errorf("attendu 0/0, got %d/%d", agg.Total, agg.Unique)
		}
	})
}

// TestExplorerRepo_GetMedalCountsForMatches_Halo5_AggregatesPerfectKillIDs valide
// que perfect_kills agrège bien le SET de médailles « frag parfait » du titre h5
// (6 ids) et pas un seul littéral. On seed deux ids perfect-kill h5 DIFFÉRENTS
// (1080468863 ×2 + 3653057799 ×1) → perfect_kills attendu = 3. Une 3e médaille
// non-perfect-kill (et « Perfection » 3592822316, exclue du set) ne doivent PAS
// compter.
func TestExplorerRepo_GetMedalCountsForMatches_Halo5_AggregatesPerfectKillIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	pdb.TitleSlug = "halo_5" // discriminant title-aware du set perfect-kill
	ctx := context.Background()
	const targetXUID = "xuid_h5_777"

	seed := []struct {
		medalNameID uint64
		matchID     string
		count       int
	}{
		{1080468863, "m1", 2}, // perfect-kill h5 #1 ×2
		{3653057799, "m1", 1}, // perfect-kill h5 #2 ×1 (id DIFFÉRENT)
		{3592822316, "m1", 4}, // « Perfection » h5 — EXCLUE du set perfect-kill
		{42, "m1", 9},         // médaille quelconque non perfect-kill
		{1512363953, "m1", 5}, // perfect-kill HINF — NE doit PAS compter en h5
	}
	for _, m := range seed {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (?,?,?,?,?)`,
			m.medalNameID, m.medalNameID, targetXUID, m.matchID, m.count)
		if err != nil {
			t.Fatalf("insert medal: %v", err)
		}
	}

	repo := NewExplorerRepo(pdb, pTestXUID)
	agg, err := repo.GetMedalCountsForMatches(ctx, targetXUID, []string{"m1"})
	if err != nil {
		t.Fatalf("GetMedalCountsForMatches: %v", err)
	}
	if agg == nil {
		t.Fatal("agg attendu non-nil")
	}
	// 2 (#1080468863) + 1 (#3653057799) = 3, agrégation des 6 ids h5.
	if agg.PerfectKills != 3 {
		t.Errorf("PerfectKills = %d, want 3 (agrégation des ids perfect-kill h5 ; "+
			"Perfection + HINF id + médaille quelconque exclus)", agg.PerfectKills)
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

// ---------------------------------------------------------------------------
// SquadRepo
// ---------------------------------------------------------------------------

func TestSquadRepo_LookupXUIDByGamertag(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Seed xuid_aliases avec un gamertag spécifique sur pdb.Shared (où
	// SharedReader pointe). Cas testés : found, not-found, casse-insensible
	// (ILIKE), trim espaces.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag, last_seen) VALUES (?, ?, ?)`,
		"xuid_lookup_001", "LookupTarget", "2026-04-01 10:00:00+00")
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag, last_seen) VALUES (?, ?, ?)`,
		"xuid_lookup_002", "LookupTarget", "2026-04-15 10:00:00+00") // plus récent

	repo := NewSquadRepo(pdb)

	// Cas 1 : trouvé (case-insensitive via ILIKE + dernier last_seen)
	xuid, ok, err := repo.LookupXUIDByGamertag(ctx, "lookuptarget")
	if err != nil {
		t.Fatalf("LookupXUIDByGamertag found: %v", err)
	}
	if !ok || xuid != "xuid_lookup_002" {
		t.Errorf("expected xuid_lookup_002 (most recent), got %q ok=%v", xuid, ok)
	}

	// Cas 2 : non trouvé (best-effort, pas d'erreur)
	xuid, ok, err = repo.LookupXUIDByGamertag(ctx, "DoesNotExist")
	if err != nil {
		t.Fatalf("LookupXUIDByGamertag not-found: %v", err)
	}
	if ok || xuid != "" {
		t.Errorf("expected empty/false, got %q ok=%v", xuid, ok)
	}

	// Cas 3 : gamertag vide → early return (pas d'erreur, pas de query)
	xuid, ok, err = repo.LookupXUIDByGamertag(ctx, "   ")
	if err != nil || ok || xuid != "" {
		t.Errorf("empty gamertag: expected ('', false, nil), got (%q, %v, %v)", xuid, ok, err)
	}
}

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
