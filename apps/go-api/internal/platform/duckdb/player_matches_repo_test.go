//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// resetPlayerMatchesTables nettoie les tables avant d'inserer nos fixtures.
// Necessaire car newTestPlayerDB() insere une row "m1" via seedPlayerSchema()
// que nos tests ne controlent pas.
func resetPlayerMatchesTables(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, "DELETE FROM player_match_enrichment"); err != nil {
		t.Fatalf("reset (player_match_enrichment): %v", err)
	}
	execOnSharedDBs(t, pdb, ctx, "DELETE FROM shared.match_participants")
	execOnSharedDBs(t, pdb, ctx, "DELETE FROM shared.match_registry")
}

// seedPlayerMatchesFixtures insere un jeu de fixtures couvrant tous les cas
// utilises par les tests : 3 maps, 4 outcomes, ranked / firefight / social,
// avec et sans bot teammate, scores varies, friend xuids.
func seedPlayerMatchesFixtures(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	resetPlayerMatchesTables(t, pdb)
	ctx := context.Background()

	// 5 matchs avec start_time decroissants. Les match_id sont prefixes pour
	// faciliter le matching dans les assertions.
	type fix struct {
		matchID, mapID, mapName, mapNameFR string
		pairName, playlistName             string
		isFirefight, isRanked              bool
		startTime                          string
		duration                           int
		// p1 (joueur principal)
		outcome                          int
		kills, deaths, assists, headshot int
		kda                              float64
		timePlayed                       int
		// pme
		hadBotTeammate, isWithFriends bool
		dominanceFlag                 int
		performanceScore              float64
		// teammate xuid present (pour ExcludeFriendsXUIDs)
		friendXUID string
	}
	fixtures := []fix{
		{
			matchID: "m_recent_win", mapID: "bazaar", mapName: "Bazaar", mapNameFR: "Bazar",
			pairName: "Arena:Slayer", playlistName: "Quick Play",
			startTime: "2026-04-26T10:00:00Z", duration: 600,
			outcome: 2 /* WIN */, kills: 20, deaths: 5, assists: 3, headshot: 8,
			kda: 4.6, timePlayed: 600, dominanceFlag: 1, /* Domination */
			performanceScore: 87.5, friendXUID: "friend1",
		},
		{
			matchID: "m_old_loss", mapID: "aquarius", mapName: "Aquarius", mapNameFR: "Verseau",
			pairName: "Arena:CTF", playlistName: "Ranked Arena", isRanked: true,
			startTime: "2026-01-01T10:00:00Z", duration: 800,
			outcome: 3 /* LOSS */, kills: 5, deaths: 12, assists: 2, headshot: 1,
			kda: 0.5, timePlayed: 800, performanceScore: 35,
		},
		{
			matchID: "m_btb_tie", mapID: "fragmentation", mapName: "Fragmentation", mapNameFR: "Fragmentation",
			pairName: "BTB:Slayer", playlistName: "Big Team Battle",
			startTime: "2026-03-15T10:00:00Z", duration: 700,
			outcome: 1 /* TIE */, kills: 10, deaths: 10, assists: 5, headshot: 4,
			kda: 1.5, timePlayed: 700, performanceScore: 60,
		},
		{
			matchID: "m_dnf", mapID: "bazaar", mapName: "Bazaar", mapNameFR: "Bazar",
			pairName: "Arena:Slayer", playlistName: "Quick Play",
			startTime: "2026-04-20T10:00:00Z", duration: 90,
			outcome: 4 /* DNF */, kills: 0, deaths: 1, assists: 0, headshot: 0,
			timePlayed: 90, hadBotTeammate: true,
		},
		{
			matchID: "m_ff", mapID: "ff_map", mapName: "Outpost", mapNameFR: "Avant-poste",
			pairName: "Firefight", playlistName: "Firefight Bookmark", isFirefight: true,
			startTime: "2026-04-22T10:00:00Z", duration: 1200,
			outcome: 2 /* WIN */, kills: 50, deaths: 3, assists: 0, headshot: 20,
			kda: 17.0, timePlayed: 1200, performanceScore: 95,
		},
	}

	for _, f := range fixtures {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
			(match_id, start_time, map_id, map_name, map_name_fr, pair_name,
			 playlist_name, is_firefight, is_ranked, duration_seconds)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.matchID, f.startTime, f.mapID, f.mapName, f.mapNameFR, f.pairName,
			f.playlistName, f.isFirefight, f.isRanked, f.duration)
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, outcome, kills, deaths, assists, kda,
			 time_played_seconds, headshot_kills, team_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.matchID, pTestXUID, pTestGamertag, f.outcome, f.kills, f.deaths,
			f.assists, f.kda, f.timePlayed, f.headshot, 0)
		if f.friendXUID != "" {
			execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
				(match_id, xuid, gamertag, team_id) VALUES (?, ?, ?, ?)`,
				f.matchID, f.friendXUID, "FriendOne", 0)
		}
		// player_match_enrichment reste sur pdb.Player uniquement (table player).
		_, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment
			(match_id, performance_score, dominance_flag, had_bot_teammate, is_with_friends)
			VALUES (?, ?, ?, ?, ?)`,
			f.matchID, f.performanceScore, f.dominanceFlag, f.hadBotTeammate, f.isWithFriends)
		if err != nil {
			t.Fatalf("insert pme %s: %v", f.matchID, err)
		}
	}
}

// TestPlayerMatchesRepo_LobbySizesAtCompletion : compte des participants présents
// à la fin (present_at_completion=TRUE, bots inclus), résistant au churn.
func TestPlayerMatchesRepo_LobbySizesAtCompletion(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetPlayerMatchesTables(t, pdb)
	ctx := context.Background()
	// Le schéma de test de match_participants est minimal ; la colonne existe en
	// prod (schema.go) mais pas ici → on l'ajoute pour rendre le test self-contained.
	execOnSharedDBs(t, pdb, ctx,
		`ALTER TABLE shared.match_participants ADD COLUMN IF NOT EXISTS present_at_completion BOOLEAN`)

	insP := func(matchID, xuid string, present bool) {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, team_id, present_at_completion)
			VALUES (?, ?, ?, ?, ?)`, matchID, xuid, xuid, 0, present)
	}
	// lobby1 : 3 présents à la fin (dont 1 bot) + 1 quitter (exclu).
	insP("lobby1", pTestXUID, true)
	insP("lobby1", "p2", true)
	insP("lobby1", "bid(0123456789)", true) // bot → compté (bots inclus)
	insP("lobby1", "quitter", false)        // parti avant la fin → exclu
	// lobby2 : 2 présents.
	insP("lobby2", pTestXUID, true)
	insP("lobby2", "p3", true)

	repo := NewPlayerMatchesRepo(pdb)
	sizes, err := repo.LobbySizesAtCompletion(ctx, []string{"lobby1", "lobby2"})
	if err != nil {
		t.Fatalf("LobbySizesAtCompletion: %v", err)
	}
	if sizes["lobby1"] != 3 {
		t.Fatalf("lobby1: want 3 (présents à la fin, bots inclus), got %d", sizes["lobby1"])
	}
	if sizes["lobby2"] != 2 {
		t.Fatalf("lobby2: want 2, got %d", sizes["lobby2"])
	}
	// matchIDs vide → map vide, pas d'erreur.
	empty, err := repo.LobbySizesAtCompletion(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty input: want {}, got %v (err=%v)", empty, err)
	}
}

func TestPlayerMatchesRepo_Load_NoFilter(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 5 {
		t.Errorf("want 5 rows, got %d", len(rows))
	}
	// Default order : start_time DESC -> m_recent_win first
	if rows[0].Summary.MatchID != "m_recent_win" {
		t.Errorf("default order should be DESC : first row %s", rows[0].Summary.MatchID)
	}
	// Verify identity propagated
	if rows[0].Self.Identity.XUID != pTestXUID {
		t.Errorf("Self.Identity.XUID: %s", rows[0].Self.Identity.XUID)
	}
}

func TestPlayerMatchesRepo_Load_FilterByPeriod1M(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	period := temporal.Period1M
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{Period: &period})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 1M par rapport a now (2026-04-27) : 2026-03-27 -> 2026-04-27
	// m_recent_win, m_dnf, m_ff sont dans la fenetre. m_old_loss et m_btb_tie sont hors.
	for _, row := range rows {
		if row.Summary.StartedAtUTC.Before(time.Now().AddDate(0, -1, -1)) {
			t.Errorf("row %s outside 1M window: %v", row.Summary.MatchID, row.Summary.StartedAtUTC)
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterByOutcome(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{
		OutcomeIn: []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Summary.Outcome != canonical.OutcomeWin && row.Summary.Outcome != canonical.OutcomeLoss {
			t.Errorf("row %s outcome %s should be win or loss", row.Summary.MatchID, row.Summary.Outcome)
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterHadBotTeammate(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	hadBot := false
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{HadBotTeammate: &hadBot})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Enrichment.HadBotTeammate {
			t.Errorf("row %s should be HadBotTeammate=false", row.Summary.MatchID)
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterFirefight(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	ff := true
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{IsFirefight: &ff})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 1 || rows[0].Summary.MatchID != "m_ff" {
		t.Errorf("expected only m_ff, got %v", rows)
	}
}

func TestPlayerMatchesRepo_Load_FilterRanked(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	r := true
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{IsRanked: &r})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 1 || rows[0].Summary.MatchID != "m_old_loss" {
		t.Errorf("expected only m_old_loss (ranked), got %v", rows)
	}
}

func TestPlayerMatchesRepo_Load_FilterMinTimePlayed(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	min := 180
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{MinTimePlayedSeconds: &min})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		// m_dnf a timePlayed=90 -> doit etre exclu
		if row.Summary.MatchID == "m_dnf" {
			t.Errorf("m_dnf should be excluded (timePlayed=90 < 180)")
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterBTBExcluded(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{BTBExcluded: true})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Summary.MatchID == "m_btb_tie" {
			t.Errorf("m_btb_tie should be excluded by BTBExcluded")
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterExcludeFriends(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{
		ExcludeFriendsXUIDs: []string{"friend1"},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Summary.MatchID == "m_recent_win" {
			t.Errorf("m_recent_win should be excluded (friend1 was a participant)")
		}
	}
}

func TestPlayerMatchesRepo_Load_FilterMapIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{
		MapIDs: []string{"bazaar"},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Summary.Map == nil || row.Summary.Map.ID != "bazaar" {
			t.Errorf("only bazaar expected, got %v", row.Summary.Map)
		}
	}
}

func TestPlayerMatchesRepo_Load_OrderByPerformance(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{
		OrderBy: "performance_score DESC",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// m_ff (95) > m_recent_win (87.5) > m_btb_tie (60) > m_old_loss (35) > m_dnf (0)
	if rows[0].Summary.MatchID != "m_ff" {
		t.Errorf("perf desc: first should be m_ff (95), got %s", rows[0].Summary.MatchID)
	}
}

func TestPlayerMatchesRepo_Load_LimitApplied(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("limit 2: got %d rows", len(rows))
	}
}

func TestPlayerMatchesRepo_Load_PlaylistKindRanked(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	kind := "ranked"
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{PlaylistKind: &kind})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 1 || rows[0].Summary.MatchID != "m_old_loss" {
		t.Errorf("playlist_kind=ranked should match m_old_loss, got %v", rows)
	}
}

func TestPlayerMatchesRepo_Load_PlaylistKindBTB(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	kind := "btb"
	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{PlaylistKind: &kind})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 1 || rows[0].Summary.MatchID != "m_btb_tie" {
		t.Errorf("playlist_kind=btb should match m_btb_tie, got %v", rows)
	}
}

func TestPlayerMatchesRepo_Load_UnknownPlaylistKind(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	kind := "not-a-real-kind"
	_, err := repo.Load(context.Background(), port.PlayerMatchFilters{PlaylistKind: &kind})
	if err == nil {
		t.Fatal("expected error for unknown PlaylistKind")
	}
}

func TestPlayerMatchesRepo_Load_UnknownOrderBy(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	_, err := repo.Load(context.Background(), port.PlayerMatchFilters{OrderBy: "drop_table"})
	if err == nil {
		t.Fatal("expected error for unknown OrderBy")
	}
}

func TestPlayerMatchesRepo_Load_DominanceFlagPropagated(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, row := range rows {
		if row.Summary.MatchID == "m_recent_win" {
			if row.Enrichment.DominanceFlag != canonical.DominanceDomination {
				t.Errorf("m_recent_win DominanceFlag: want Domination, got %d",
					row.Enrichment.DominanceFlag)
			}
		}
	}
}

func TestPlayerMatchesRepo_Load_MapAndPlaylistAssetsHydrated(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	rows, err := repo.Load(context.Background(), port.PlayerMatchFilters{Limit: 1})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	r := rows[0]
	if r.Summary.Map == nil || r.Summary.Map.Labels["fr"] == "" {
		t.Errorf("Map FR label missing: %+v", r.Summary.Map)
	}
	if r.Summary.Playlist == nil || r.Summary.Playlist.DefaultLabel == "" {
		t.Errorf("Playlist DefaultLabel missing: %+v", r.Summary.Playlist)
	}
}

func TestPlayerMatchesRepo_Load_InvalidFiltersRejected(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	repo := NewPlayerMatchesRepo(pdb)

	_, err := repo.Load(context.Background(), port.PlayerMatchFilters{Limit: -5})
	if err == nil {
		t.Fatal("expected error for negative Limit")
	}
}

// TestPlayerMatchesRepo_Load_T0MsComputed vérifie le calcul de l'offset T0
// (Match Timeline T0, Phase 3) : un match avec real_start_time renseigné expose
// T0Ms = epoch_ms(real_start_time UTC) − epoch_ms(start_time_utc) ; un match
// sans real_start_time expose T0Ms nil (fallback runtime T0=0).
func TestPlayerMatchesRepo_Load_T0MsComputed(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetPlayerMatchesTables(t, pdb)
	ctx := context.Background()

	// m_t0 : countdown de 28s entre start_time officiel et début gameplay.
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
		(match_id, start_time, start_time_utc, real_start_time, map_name, pair_name,
		 playlist_name, duration_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"m_t0", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00+00",
		"2026-01-01T00:00:28", "Bazaar", "Slayer", "Quick Play", 600)
	// m_no_t0 : real_start_time absent → T0Ms doit rester nil.
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
		(match_id, start_time, start_time_utc, map_name, pair_name, playlist_name, duration_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"m_no_t0", "2026-01-02T00:00:00Z", "2026-01-02T00:00:00+00",
		"Bazaar", "Slayer", "Quick Play", 600)

	for _, mid := range []string{"m_t0", "m_no_t0"} {
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, ?, ?)`,
			mid, pTestXUID, pTestGamertag, 2, 0)
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment
			(match_id) VALUES (?)`, mid); err != nil {
			t.Fatalf("insert pme %s: %v", mid, err)
		}
	}

	repo := NewPlayerMatchesRepo(pdb)
	rows, err := repo.Load(ctx, port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	byID := make(map[string]canonical.MatchSummary, len(rows))
	for _, r := range rows {
		byID[r.Summary.MatchID] = r.Summary
	}

	t0 := byID["m_t0"]
	if t0.T0Ms == nil {
		t.Fatalf("m_t0: T0Ms should be non-nil (real_start_time set)")
	}
	if *t0.T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000ms, got %d", *t0.T0Ms)
	}
	if got := byID["m_no_t0"]; got.T0Ms != nil {
		t.Errorf("m_no_t0: T0Ms want nil (no real_start_time), got %d", *got.T0Ms)
	}
}

// TestPlayerMatchesRepo_Load_PairModeHydrated vérifie que PairMode est peuplé
// depuis pair_name (EN) et pair_name_fr (COALESCE), aligné avec filtersResolve.
func TestPlayerMatchesRepo_Load_PairModeHydrated(t *testing.T) {
	pdb := newTestPlayerDB(t)
	resetPlayerMatchesTables(t, pdb)
	ctx := context.Background()

	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
		(match_id, start_time, map_id, map_name, pair_name, pair_name_fr,
		 playlist_name, duration_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"pm_test", "2026-01-01T00:00:00Z", "bazaar", "Bazaar",
		"Slayer", "Assassin", "Quick Play", 600)
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
		(match_id, xuid, gamertag, outcome, team_id)
		VALUES (?, ?, ?, ?, ?)`,
		"pm_test", pTestXUID, pTestGamertag, 2, 0)
	_, err := pdb.Player.Exec(ctx, `INSERT INTO player_match_enrichment
		(match_id) VALUES (?)`, "pm_test")
	if err != nil {
		t.Fatalf("insert player_match_enrichment: %v", err)
	}

	repo := NewPlayerMatchesRepo(pdb)
	rows, loadErr := repo.Load(ctx, port.PlayerMatchFilters{})
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	pm := rows[0].Summary.PairMode
	if pm == nil {
		t.Fatal("PairMode should be non-nil when pair_name is set")
	}
	if pm.DefaultLabel != "Slayer" {
		t.Errorf("PairMode.DefaultLabel want %q, got %q", "Slayer", pm.DefaultLabel)
	}
	if got := pm.Labels["fr"]; got != "Assassin" {
		t.Errorf("PairMode.Labels[fr] want %q (pair_name_fr), got %q", "Assassin", got)
	}
	if got := pm.Labels["en"]; got != "Slayer" {
		t.Errorf("PairMode.Labels[en] want %q (pair_name), got %q", "Slayer", got)
	}
}
