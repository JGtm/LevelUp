package analysis_test

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeHomeMatch(matchID string, outcome int, ratio, accuracy *float64, isWithFriends bool) legacymatch.HomeMatchRow { //nolint:unparam
	t := time.Now()
	return legacymatch.HomeMatchRow{
		MatchID:       matchID,
		StartTime:     t,
		MapName:       "Recharge",
		PairName:      "Slayer",
		PlaylistName:  "Ranked",
		Outcome:       outcome,
		IsWithFriends: isWithFriends,
		Ratio:         ratio,
		Accuracy:      accuracy,
	}
}

func homeMatchAt(matchID string, outcome int, ratio *float64, t time.Time) legacymatch.HomeMatchRow {
	return legacymatch.HomeMatchRow{
		MatchID:   matchID,
		StartTime: t,
		MapName:   "Bazaar",
		PairName:  "Slayer",
		Outcome:   outcome,
		Ratio:     ratio,
	}
}

func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }

// ---------------------------------------------------------------------------
// ComputeKPIs
// ---------------------------------------------------------------------------

func TestComputeKPIs_Empty(t *testing.T) {
	kpis := analysis.ComputeKPIs(nil, 0)
	if kpis.TotalMatches != 0 || kpis.WinRate != 0 {
		t.Errorf("empty: got %+v", kpis)
	}
}

func TestComputeKPIs_WithMatches(t *testing.T) {
	matches := []legacymatch.HomeMatchRow{
		makeHomeMatch("m1", 2, fp(2.0), fp(50.0), false), // win
		makeHomeMatch("m2", 3, fp(0.5), fp(30.0), false), // loss
		makeHomeMatch("m3", 2, fp(1.5), nil, false),      // win, no accuracy
	}
	kpis := analysis.ComputeKPIs(matches, len(matches))
	if kpis.TotalMatches != 3 {
		t.Errorf("TotalMatches: want 3, got %d", kpis.TotalMatches)
	}
	if kpis.Wins != 2 {
		t.Errorf("Wins: want 2, got %d", kpis.Wins)
	}
	if kpis.Losses != 1 {
		t.Errorf("Losses: want 1, got %d", kpis.Losses)
	}
	wantWR := 2.0 / 3.0
	if abs64(kpis.WinRate-wantWR) > 1e-6 {
		t.Errorf("WinRate: want %.4f, got %.4f", wantWR, kpis.WinRate)
	}
	// GlobalRatio = (2 + 0.5 + 1.5) / 3 = 1.33
	if kpis.GlobalRatio == nil {
		t.Fatal("GlobalRatio: want non-nil")
	}
	if abs64(*kpis.GlobalRatio-1.33) > 0.01 {
		t.Errorf("GlobalRatio: want ~1.33, got %.2f", *kpis.GlobalRatio)
	}
	// AvgAccuracy = (50 + 30) / 2 = 40
	if kpis.AvgAccuracy == nil {
		t.Fatal("AvgAccuracy: want non-nil")
	}
	if abs64(*kpis.AvgAccuracy-40.0) > 0.1 {
		t.Errorf("AvgAccuracy: want 40, got %.1f", *kpis.AvgAccuracy)
	}
}

// ---------------------------------------------------------------------------
// ComputeTrend
// ---------------------------------------------------------------------------

func TestComputeTrend_NotEnoughMatches(t *testing.T) {
	matches := []legacymatch.HomeMatchRow{
		makeHomeMatch("m1", 2, fp(1.5), nil, false),
	}
	trend := analysis.ComputeTrend(matches, 5)
	if trend != nil {
		t.Errorf("should be nil with only 1 match")
	}
}

func TestComputeTrend_WithData(t *testing.T) {
	// 10 matchs : 5 rÃ©cents (ratio=2.0), 5 prÃ©cÃ©dents (ratio=1.0).
	var matches []legacymatch.HomeMatchRow
	for i := 0; i < 5; i++ {
		matches = append(matches, makeHomeMatch("r"+string(rune('a'+i)), 2, fp(2.0), nil, false))
	}
	for i := 0; i < 5; i++ {
		matches = append(matches, makeHomeMatch("p"+string(rune('a'+i)), 2, fp(1.0), nil, false))
	}
	trend := analysis.ComputeTrend(matches, 5)
	if trend == nil {
		t.Fatal("trend: want non-nil")
	}
	if trend.RatioDelta == nil {
		t.Fatal("RatioDelta: want non-nil")
	}
	// 2.0 - 1.0 = 1.0
	if abs64(*trend.RatioDelta-1.0) > 0.01 {
		t.Errorf("RatioDelta: want 1.0, got %.3f", *trend.RatioDelta)
	}
}

// ---------------------------------------------------------------------------
// BuildRecentMatches
// ---------------------------------------------------------------------------

func TestBuildRecentMatches_Empty(t *testing.T) {
	items := analysis.BuildRecentMatches(nil, 6)
	if len(items) != 0 {
		t.Errorf("want empty, got %d", len(items))
	}
}

func TestBuildRecentMatches_Limit(t *testing.T) {
	var matches []legacymatch.HomeMatchRow
	for i := 0; i < 10; i++ {
		matches = append(matches, makeHomeMatch("m"+string(rune('a'+i)), 2, fp(1.0), fp(55.0), false))
	}
	items := analysis.BuildRecentMatches(matches, 6)
	if len(items) != 6 {
		t.Errorf("want 6, got %d", len(items))
	}
	if items[0].OutcomeLabel != "Victoire" {
		t.Errorf("outcome label: want Victoire, got %s", items[0].OutcomeLabel)
	}
	if items[0].OutcomeTone != "win" {
		t.Errorf("outcome tone: want win, got %s", items[0].OutcomeTone)
	}
}

func TestBuildRecentMatches_NormalizesModeLabel(t *testing.T) {
	now := time.Now()
	items := analysis.BuildRecentMatches([]legacymatch.HomeMatchRow{{
		MatchID:      "m1",
		StartTime:    now,
		MapName:      "Aquarius",
		MapNameFR:    "Aquarius",
		PairName:     "Slayer on Aquarius",
		PairNameFR:   "Slayer sur Aquarius",
		PlaylistName: "Ranked Arena",
		Outcome:      2,
	}}, 6)

	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].ModeUI == nil || *items[0].ModeUI != "Slayer" {
		t.Fatalf("ModeUI: want Slayer, got %v", items[0].ModeUI)
	}
	if items[0].MapUI == nil || *items[0].MapUI != "Aquarius" {
		t.Fatalf("MapUI: want Aquarius, got %v", items[0].MapUI)
	}
	if items[0].PlaylistUI == nil || *items[0].PlaylistUI != "Ranked Arena" {
		t.Fatalf("PlaylistUI: want Ranked Arena, got %v", items[0].PlaylistUI)
	}
}

func TestBuildRecentMatchesForLocale_UsesRequestedLanguage(t *testing.T) {
	now := time.Now()
	match := legacymatch.HomeMatchRow{
		MatchID:           "m-locale",
		StartTime:         now,
		MapName:           "Bazaar",
		MapNameFR:         "Bazaar",
		PairName:          "Team Slayer on Bazaar",
		PairNameFR:        "Slayer en Ã©quipe sur Bazaar",
		GameVariantName:   "Arena:Slayer",
		GameVariantNameFR: "Assassin : ArÃ¨ne",
		PlaylistName:      "Quick Play",
		PlaylistNameFR:    "Partie rapide",
		Outcome:           2,
	}

	itemsFR := analysis.BuildRecentMatchesForLocale([]legacymatch.HomeMatchRow{match}, 6, "fr")
	if itemsFR[0].ModeUI == nil || *itemsFR[0].ModeUI != "Slayer en Ã©quipe" {
		t.Fatalf("FR ModeUI: want Slayer en Ã©quipe, got %v", itemsFR[0].ModeUI)
	}
	if itemsFR[0].PlaylistUI == nil || *itemsFR[0].PlaylistUI != "Partie rapide" {
		t.Fatalf("FR PlaylistUI: want Partie rapide, got %v", itemsFR[0].PlaylistUI)
	}
	if itemsFR[0].OutcomeLabel != "Victoire" {
		t.Fatalf("FR OutcomeLabel: want Victoire, got %q", itemsFR[0].OutcomeLabel)
	}

	itemsEN := analysis.BuildRecentMatchesForLocale([]legacymatch.HomeMatchRow{match}, 6, "en")
	if itemsEN[0].ModeUI == nil || *itemsEN[0].ModeUI != "Team Slayer" {
		t.Fatalf("EN ModeUI: want Team Slayer, got %v", itemsEN[0].ModeUI)
	}
	if itemsEN[0].PlaylistUI == nil || *itemsEN[0].PlaylistUI != "Quick Play" {
		t.Fatalf("EN PlaylistUI: want Quick Play, got %v", itemsEN[0].PlaylistUI)
	}
	if itemsEN[0].OutcomeLabel != "Victory" {
		t.Fatalf("EN OutcomeLabel: want Victory, got %q", itemsEN[0].OutcomeLabel)
	}
}

func TestBuildRecentMatches_UsesLocalStaticMapImageAndStripsExperiencePrefix(t *testing.T) {
	now := time.Now()
	items := analysis.BuildRecentMatches([]legacymatch.HomeMatchRow{{
		MatchID:       "m2",
		StartTime:     now,
		MapID:         "3e1e4cec-4f2c-44c6-b8d2-96b85c66c702",
		MapName:       "Bazaar",
		PairName:      "Quick Play: Slayer on Bazaar",
		PlaylistName:  "Quick Play",
		TeamID:        1,
		Team0Score:    1,
		Team1Score:    3,
		DominanceFlag: 3,
		Outcome:       3,
	}}, 6)

	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].ModeUI == nil || *items[0].ModeUI != "Slayer" {
		t.Fatalf("ModeUI: want Slayer, got %v", items[0].ModeUI)
	}
	if items[0].MapImageURL == nil || *items[0].MapImageURL != "/static/maps/halo_infinite/Bazaar.png" {
		t.Fatalf("MapImageURL: want /static/maps/halo_infinite/Bazaar.png, got %v", items[0].MapImageURL)
	}
	if items[0].ScoreLabel == nil || *items[0].ScoreLabel != "3-1" {
		t.Fatalf("ScoreLabel: want 3-1, got %v", items[0].ScoreLabel)
	}
	if len(items[0].NarrativeBadges) != 1 || items[0].NarrativeBadges[0] != "remontada" {
		t.Fatalf("NarrativeBadges: want [remontada], got %#v", items[0].NarrativeBadges)
	}
}

func TestBuildRecentMatches_MapsDominanceBadge(t *testing.T) {
	now := time.Now()
	items := analysis.BuildRecentMatches([]legacymatch.HomeMatchRow{{
		MatchID:       "m-domination",
		StartTime:     now,
		MapName:       "Recharge",
		PairName:      "Slayer",
		PlaylistName:  "Quick Play",
		Outcome:       2,
		TeamID:        0,
		Team0Score:    50,
		Team1Score:    13,
		DominanceFlag: 1,
	}}, 6)

	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if len(items[0].NarrativeBadges) != 1 || items[0].NarrativeBadges[0] != "dominant" {
		t.Fatalf("NarrativeBadges: want [dominant], got %#v", items[0].NarrativeBadges)
	}
}

func TestBuildSpartanIdentity_UsesRequestedLanguage(t *testing.T) {
	raw := &domain.HomeSpartanIdentityRow{
		SpartanID:         sp("JGTM"),
		BannerImageURL:    sp("https://example.test/banner.png"),
		EmblemImageURL:    sp("https://example.test/emblem.png"),
		BackdropImageURL:  sp("https://example.test/backdrop.png"),
		RankNumber:        25,
		RankName:          sp("Platinum 1"),
		RankTier:          sp("Platinum"),
		RankImageURL:      sp("https://example.test/career-rank.png"),
		AdornmentImageURL: sp("https://example.test/career-adornment.png"),
		CurrentXP:         5000,
		XPForNextRank:     10000,
	}

	ranks := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{
			ID:    25,
			Title: map[string]string{"en": "Lance Corporal", "fr": "Caporal-chef"},
		},
	})

	identityFR := analysis.BuildSpartanIdentity(raw, "fr", ranks)
	if identityFR == nil || identityFR.SpartanID == nil || *identityFR.SpartanID != "JGTM" {
		t.Fatalf("FR SpartanID: got %#v", identityFR)
	}
	if identityFR.CareerRank == nil {
		t.Fatal("FR CareerRank: want non-nil")
	}
	if identityFR.CareerRank.RankTitle != "Caporal-chef" {
		t.Fatalf("FR RankTitle: want Caporal-chef, got %q", identityFR.CareerRank.RankTitle)
	}
	if identityFR.CareerRank.ProgressPct != 50.0 {
		t.Fatalf("FR ProgressPct: want 50, got %.2f", identityFR.CareerRank.ProgressPct)
	}
	if identityFR.EmblemImageURL == nil || *identityFR.EmblemImageURL != "https://example.test/emblem.png" {
		t.Fatalf("FR EmblemImageURL: got %#v", identityFR.EmblemImageURL)
	}
	if identityFR.BannerImageURL == nil || *identityFR.BannerImageURL != "https://example.test/banner.png" {
		t.Fatalf("FR BannerImageURL: got %#v", identityFR.BannerImageURL)
	}
	if identityFR.BackdropImageURL == nil || *identityFR.BackdropImageURL != "https://example.test/backdrop.png" {
		t.Fatalf("FR BackdropImageURL: got %#v", identityFR.BackdropImageURL)
	}
	if identityFR.CareerRank.RankImageURL == nil || *identityFR.CareerRank.RankImageURL != "https://example.test/career-rank.png" {
		t.Fatalf("FR RankImageURL: got %#v", identityFR.CareerRank.RankImageURL)
	}
	if identityFR.CareerRank.AdornmentImageURL == nil || *identityFR.CareerRank.AdornmentImageURL != "https://example.test/career-adornment.png" {
		t.Fatalf("FR AdornmentImageURL: got %#v", identityFR.CareerRank.AdornmentImageURL)
	}

	identityEN := analysis.BuildSpartanIdentity(raw, "en", ranks)
	if identityEN == nil || identityEN.CareerRank == nil {
		t.Fatal("EN CareerRank: want non-nil")
	}
	if identityEN.CareerRank.RankTitle != "Lance Corporal" {
		t.Fatalf("EN RankTitle: want Lance Corporal, got %q", identityEN.CareerRank.RankTitle)
	}
}

// ---------------------------------------------------------------------------
// BuildRecentMedia
// ---------------------------------------------------------------------------

func TestBuildRecentMedia_Empty(t *testing.T) {
	items := analysis.BuildRecentMedia(nil, 4)
	if len(items) != 0 {
		t.Errorf("want empty, got %d", len(items))
	}
}

func TestBuildRecentMedia_WithData(t *testing.T) {
	media := []domain.HomeMediaRow{
		{FileName: "clip1.mp4", MatchID: sp("match-1")},
		{FileName: "clip2.mp4"},
	}
	items := analysis.BuildRecentMedia(media, 4)
	if len(items) != 2 {
		t.Errorf("want 2, got %d", len(items))
	}
	if items[0].Basename != "clip1.mp4" {
		t.Errorf("basename: want clip1.mp4, got %s", items[0].Basename)
	}
	if items[0].MatchID == nil || *items[0].MatchID != "match-1" {
		t.Errorf("match_id: want match-1")
	}
	if items[1].MatchID != nil {
		t.Errorf("match_id: want nil for clip2")
	}
}

// ---------------------------------------------------------------------------
// BuildSessionSummaries
// ---------------------------------------------------------------------------

func TestBuildSessionSummaries_RetourneNSessionsTrieesDesc(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(-1 * time.Hour)
	t3 := t1.Add(-2 * time.Hour)
	l1, l2, l3 := "session-1", "session-2", "session-3"

	sessions := []legacymatch.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &l1, IsWithFriends: false, StartTime: &t1},
		{MatchID: "m2", SessionLabel: &l2, IsWithFriends: false, StartTime: &t2},
		{MatchID: "m3", SessionLabel: &l3, IsWithFriends: false, StartTime: &t3},
	}
	matches := []legacymatch.HomeMatchRow{
		homeMatchAt("m1", 2, fp(1.5), t1),
		homeMatchAt("m2", 3, fp(0.8), t2),
		homeMatchAt("m3", 2, fp(2.0), t3),
	}

	result := analysis.BuildSessionSummaries(matches, sessions, false, 10)
	if len(result) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(result))
	}
	if result[0].SessionLabel != l1 {
		t.Errorf("first: want %s, got %s", l1, result[0].SessionLabel)
	}
	if result[1].SessionLabel != l2 {
		t.Errorf("second: want %s, got %s", l2, result[1].SessionLabel)
	}
}

func TestBuildSessionSummaries_LimitRespectee(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(-1 * time.Hour)
	t3 := t1.Add(-2 * time.Hour)
	l1, l2, l3 := "session-1", "session-2", "session-3"

	sessions := []legacymatch.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &l1, IsWithFriends: false, StartTime: &t1},
		{MatchID: "m2", SessionLabel: &l2, IsWithFriends: false, StartTime: &t2},
		{MatchID: "m3", SessionLabel: &l3, IsWithFriends: false, StartTime: &t3},
	}
	matches := []legacymatch.HomeMatchRow{
		homeMatchAt("m1", 2, fp(1.5), t1),
		homeMatchAt("m2", 3, fp(0.8), t2),
		homeMatchAt("m3", 2, fp(2.0), t3),
	}

	result := analysis.BuildSessionSummaries(matches, sessions, false, 2)
	if len(result) != 2 {
		t.Fatalf("limit 2: want 2, got %d", len(result))
	}
}

func TestBuildSessionSummaries_FiltreEscouade(t *testing.T) {
	now := time.Now()
	soloLabel := "solo-session"
	squadLabel := "squad-session"

	sessions := []legacymatch.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &soloLabel, IsWithFriends: false, StartTime: &now},
		{MatchID: "m2", SessionLabel: &squadLabel, IsWithFriends: true, StartTime: &now},
	}
	matches := []legacymatch.HomeMatchRow{
		homeMatchAt("m1", 2, fp(1.0), now),
		homeMatchAt("m2", 3, fp(0.5), now),
	}

	soloResult := analysis.BuildSessionSummaries(matches, sessions, false, 5)
	squadResult := analysis.BuildSessionSummaries(matches, sessions, true, 5)

	if len(soloResult) != 1 || soloResult[0].SessionLabel != soloLabel {
		t.Errorf("solo: want [%s], got %v", soloLabel, soloResult)
	}
	if len(squadResult) != 1 || squadResult[0].SessionLabel != squadLabel {
		t.Errorf("squad: want [%s], got %v", squadLabel, squadResult)
	}
}

func TestBuildSessionSummaries_Vide(t *testing.T) {
	result := analysis.BuildSessionSummaries(nil, nil, false, 5)
	if result != nil {
		t.Errorf("want nil, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// BuildSessionSummary
// ---------------------------------------------------------------------------

func TestBuildSessionSummary_Solo(t *testing.T) {
	now := time.Now()
	before := now.Add(-2 * time.Hour)
	label := "12/04/2025 20:00â€“22:00 (3)"

	sessions := []legacymatch.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &label, IsWithFriends: false, StartTime: &now},
		{MatchID: "m2", SessionLabel: &label, IsWithFriends: false, StartTime: &before},
		{MatchID: "m3", SessionLabel: &label, IsWithFriends: false, StartTime: &before},
	}
	matches := []legacymatch.HomeMatchRow{
		homeMatchAt("m1", 2, fp(2.0), now),
		homeMatchAt("m2", 3, fp(0.5), before),
		homeMatchAt("m3", 2, fp(1.5), before),
	}

	summary := analysis.BuildSessionSummary(matches, sessions, false)
	if summary == nil {
		t.Fatal("want non-nil summary")
	}
	if summary.MatchCount != 3 {
		t.Errorf("MatchCount: want 3, got %d", summary.MatchCount)
	}
	if summary.SessionLabel != label {
		t.Errorf("SessionLabel: want %s, got %s", label, summary.SessionLabel)
	}
	wantWR := 2.0 / 3.0
	if abs64(summary.WinRate-wantWR) > 1e-6 {
		t.Errorf("WinRate: want %.4f, got %.4f", wantWR, summary.WinRate)
	}
}

func TestBuildSessionSummary_SquadModeFiltering(t *testing.T) {
	now := time.Now()
	label := "solo-session"
	labelSquad := "squad-session"

	sessions := []legacymatch.HomeSessionRow{
		{MatchID: "m1", SessionLabel: &label, IsWithFriends: false, StartTime: &now},
		{MatchID: "m2", SessionLabel: &labelSquad, IsWithFriends: true, StartTime: &now},
	}
	matches := []legacymatch.HomeMatchRow{
		homeMatchAt("m1", 2, fp(1.0), now),
		homeMatchAt("m2", 3, fp(0.5), now),
	}

	solo := analysis.BuildSessionSummary(matches, sessions, false)
	squad := analysis.BuildSessionSummary(matches, sessions, true)

	if solo == nil {
		t.Fatal("solo summary: want non-nil")
	}
	if solo.SessionLabel != label {
		t.Errorf("solo label: want %s, got %s", label, solo.SessionLabel)
	}
	if squad == nil {
		t.Fatal("squad summary: want non-nil")
	}
	if squad.SessionLabel != labelSquad {
		t.Errorf("squad label: want %s, got %s", labelSquad, squad.SessionLabel)
	}
}

// ---------------------------------------------------------------------------
// BuildHighlights â€” SÃ©rie (slide tile)
// ---------------------------------------------------------------------------

func TestBuildHighlights_SerieTile(t *testing.T) {
	// FenÃªtre : 5 matchs avec un label de session, 2 maps, streak de 3 victoires,
	// plus haute folie meurtriÃ¨re = 7.
	now := time.Now()
	label := "s1"
	spree := func(v int) *int { return &v }
	mk := func(id, mapID, mapName string, outcome int, killingSpree *int, offset time.Duration) legacymatch.HomeMatchRow {
		lbl := label
		return legacymatch.HomeMatchRow{
			MatchID:         id,
			StartTime:       now.Add(offset),
			MapID:           mapID,
			MapName:         mapName,
			MapNameFR:       mapName,
			PairName:        "Slayer",
			PairNameFR:      "Massacre",
			SessionLabel:    &lbl,
			Outcome:         outcome,
			MaxKillingSpree: killingSpree,
		}
	}
	// Order disque (DESC) : m5=W(spree 7), m4=W, m3=W, m2=L, m1=W.
	// Ordre chrono (ASC)  : m1=W, m2=L, m3=W, m4=W, m5=W â†’ plus longue sÃ©rie = 3.
	matches := []legacymatch.HomeMatchRow{
		mk("m5", "map-a", "Aquarius", 2, spree(7), 4*time.Minute), // plus rÃ©cent
		mk("m4", "map-a", "Aquarius", 2, spree(4), 3*time.Minute),
		mk("m3", "map-a", "Aquarius", 2, spree(5), 2*time.Minute),
		mk("m2", "map-b", "Streets", 3, spree(3), 1*time.Minute),
		mk("m1", "map-b", "Streets", 2, spree(2), 0), // plus ancien
	}

	got := analysis.BuildHighlights(matches)

	var serie *domain.HighlightItem
	for i := range got {
		if got[i].TitleKey == "highlight.title.serie" {
			serie = &got[i]
			break
		}
	}
	if serie == nil {
		t.Fatalf("want SÃ©rie highlight, got titleKeys=%v", titleKeys(got))
	}
	if len(serie.Slides) != 3 {
		t.Fatalf("want 3 slides, got %d", len(serie.Slides))
	}
	// Slide 1 : Folie meurtriÃ¨re max = 7.
	if serie.Slides[0].LabelKey != "highlight.slide.killing_spree_max" || serie.Slides[0].Value != "7" {
		t.Errorf("slide 0 spree: got key=%q value=%q", serie.Slides[0].LabelKey, serie.Slides[0].Value)
	}
	// Slide 2 : Victoires consÃ©cutives = 3 (w,w,w en dÃ©but chronologique).
	if serie.Slides[1].LabelKey != "highlight.slide.win_streak" || serie.Slides[1].Value != "3" {
		t.Errorf("slide 1 streak: got key=%q value=%q", serie.Slides[1].LabelKey, serie.Slides[1].Value)
	}
	if serie.Slides[1].DetailKey != "highlight.detail.win_streak" {
		t.Errorf("slide 1 streak detailKey: got %q", serie.Slides[1].DetailKey)
	}
	if count, _ := serie.Slides[1].DetailParams["count"].(int); count != 3 {
		t.Errorf("slide 1 streak count param: want 3, got %v", serie.Slides[1].DetailParams["count"])
	}
	// Slide 3 : Carte fÃ©tiche = Aquarius (3V sur 3 parties = 100%, vs Streets 1V/2).
	if serie.Slides[2].LabelKey != "highlight.slide.favorite_map" || serie.Slides[2].Value != "Aquarius" {
		t.Errorf("slide 2 map: got key=%q value=%q", serie.Slides[2].LabelKey, serie.Slides[2].Value)
	}
	if serie.Slides[2].DetailKey != "highlight.detail.favorite_map" {
		t.Errorf("slide 2 map detailKey: got %q", serie.Slides[2].DetailKey)
	}
	// Value top-level copie slide 0.
	if serie.Value != serie.Slides[0].Value {
		t.Errorf("top value: want %q, got %q", serie.Slides[0].Value, serie.Value)
	}
}

func TestBuildHighlights_MaitriseTile(t *testing.T) {
	now := time.Now()
	label := "s1"
	acc := func(v float64) *float64 { return &v }
	mk := func(id string, outcome, hs, perf int, a *float64, offset time.Duration) legacymatch.HomeMatchRow {
		lbl := label
		return legacymatch.HomeMatchRow{
			MatchID:       id,
			StartTime:     now.Add(offset),
			MapID:         "map-a",
			MapName:       "Aquarius",
			MapNameFR:     "Aquarius",
			PairName:      "Slayer",
			PairNameFR:    "Massacre",
			SessionLabel:  &lbl,
			Outcome:       outcome,
			HeadshotKills: hs,
			PerfectKills:  perf,
			Accuracy:      a,
		}
	}
	// 3 matchs, prÃ©cisions 60, 50, 40 â†’ moyenne 50 â†’ "warning" (â‰¥ 40, â‰¤ 55).
	matches := []legacymatch.HomeMatchRow{
		mk("m3", 2, 5, 1, acc(60), 2*time.Minute),
		mk("m2", 2, 3, 0, acc(50), 1*time.Minute),
		mk("m1", 3, 2, 0, acc(40), 0),
	}

	got := analysis.BuildHighlights(matches)
	var m *domain.HighlightItem
	for i := range got {
		if got[i].TitleKey == "highlight.title.mastery" {
			m = &got[i]
			break
		}
	}
	if m == nil {
		t.Fatalf("want MaÃ®trise highlight, got titleKeys=%v", titleKeys(got))
	}
	if len(m.Slides) != 3 {
		t.Fatalf("want 3 slides, got %d", len(m.Slides))
	}
	if m.Slides[0].LabelKey != "highlight.slide.headshots" || m.Slides[0].Value != "10" {
		t.Errorf("slide HS: got key=%q value=%q", m.Slides[0].LabelKey, m.Slides[0].Value)
	}
	if m.Slides[1].LabelKey != "highlight.slide.perfect_kills" || m.Slides[1].Value != "1" {
		t.Errorf("slide perfects: got key=%q value=%q", m.Slides[1].LabelKey, m.Slides[1].Value)
	}
	if m.Slides[2].LabelKey != "highlight.slide.accuracy" || m.Slides[2].Value != "50%" {
		t.Errorf("slide accuracy: got key=%q value=%q", m.Slides[2].LabelKey, m.Slides[2].Value)
	}
	if m.Slides[2].ValueColor != "warning" {
		t.Errorf("slide accuracy color: want warning, got %q", m.Slides[2].ValueColor)
	}
}

func TestBuildHighlights_PerMinuteTile(t *testing.T) {
	now := time.Now()
	label := "s1"
	secs := func(v int) *int { return &v }
	mk := func(id string, outcome, k, d, a int, tsecs *int, offset time.Duration) legacymatch.HomeMatchRow {
		lbl := label
		return legacymatch.HomeMatchRow{
			MatchID:        id,
			StartTime:      now.Add(offset),
			MapID:          "map-a",
			MapName:        "Aquarius",
			MapNameFR:      "Aquarius",
			PairName:       "Slayer",
			PairNameFR:     "Massacre",
			SessionLabel:   &lbl,
			Outcome:        outcome,
			Kills:          k,
			Deaths:         d,
			Assists:        a,
			TimePlayedSecs: tsecs,
		}
	}
	// 2 matchs : 30 kills / 10 morts / 20 assists sur 600s = 10 min â†’ 3.00 / 1.00 / 2.00 par minute.
	matches := []legacymatch.HomeMatchRow{
		mk("m2", 2, 20, 5, 12, secs(360), 1*time.Minute),
		mk("m1", 2, 10, 5, 8, secs(240), 0),
	}

	got := analysis.BuildHighlights(matches)
	var h *domain.HighlightItem
	for i := range got {
		if got[i].TitleKey == "highlight.title.per_minute" {
			h = &got[i]
			break
		}
	}
	if h == nil {
		t.Fatalf("want Stats par min. highlight, got titleKeys=%v", titleKeys(got))
	}
	if len(h.Slides) != 3 {
		t.Fatalf("want 3 slides, got %d", len(h.Slides))
	}
	if h.Slides[0].LabelKey != "highlight.slide.kills" || h.Slides[0].Value != "3.00" {
		t.Errorf("slide kpm: got key=%q value=%q", h.Slides[0].LabelKey, h.Slides[0].Value)
	}
	if h.Slides[1].LabelKey != "highlight.slide.deaths" || h.Slides[1].Value != "1.00" {
		t.Errorf("slide dpm: got key=%q value=%q", h.Slides[1].LabelKey, h.Slides[1].Value)
	}
	if h.Slides[2].LabelKey != "highlight.slide.assists" || h.Slides[2].Value != "2.00" {
		t.Errorf("slide apm: got key=%q value=%q", h.Slides[2].LabelKey, h.Slides[2].Value)
	}
}

func titleKeys(items []domain.HighlightItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.TitleKey
	}
	return out
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
