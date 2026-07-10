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
		// Rang 26 présent → 25 n'est PAS le dernier rang du catalog (sinon
		// buildHomeCareerRank le déduirait comme max → ProgressPct=100).
		{
			ID:    26,
			Title: map[string]string{"en": "Corporal", "fr": "Caporal"},
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

// TestBuildSpartanIdentity_DerivesMaxRankFromCatalog : l'API Halo ne marque pas
// toujours le dernier rang (Héros) comme max ; buildHomeCareerRank le déduit du
// catalog (rang présent sans rang suivant) → IsMaxRank=true + ProgressPct=100,
// même quand raw.IsMaxRank est false.
func TestBuildSpartanIdentity_DerivesMaxRankFromCatalog(t *testing.T) {
	raw := &domain.HomeSpartanIdentityRow{
		RankNumber:    272,
		CurrentXP:     5000,
		XPForNextRank: 10000,
		IsMaxRank:     false, // l'API n'a pas marqué le rang comme max
	}
	ranks := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 271, Title: map[string]string{"en": "Hero", "fr": "Héros"}},
		{ID: 272, Title: map[string]string{"en": "Hero", "fr": "Héros"}}, // dernier → pas de 273
	})

	id := analysis.BuildSpartanIdentity(raw, "fr", ranks)
	if id == nil || id.CareerRank == nil {
		t.Fatal("CareerRank: want non-nil")
	}
	if !id.CareerRank.IsMaxRank {
		t.Error("IsMaxRank: want true (rang 272 = dernier rang du catalog)")
	}
	if id.CareerRank.ProgressPct != 100 {
		t.Errorf("ProgressPct: want 100 (rang max), got %.2f", id.CareerRank.ProgressPct)
	}
	if id.CareerRank.NextRankTitle != "" {
		t.Errorf("NextRankTitle: want empty (rang max), got %q", id.CareerRank.NextRankTitle)
	}
}

// TestBuildSpartanIdentity_XPForNextFallbackFromCatalog : quand la source ne
// fournit pas xp_for_next (=0, cas cible Explorer suivie où career_progression
// renvoie 0), buildHomeCareerRank lit le seuil du catalog (RankEntry.XPRequired)
// pour que la barre ne reste pas bloquée à 0 % / "0 XP".
func TestBuildSpartanIdentity_XPForNextFallbackFromCatalog(t *testing.T) {
	raw := &domain.HomeSpartanIdentityRow{
		RankNumber:    25,
		CurrentXP:     8210,
		XPForNextRank: 0, // source locale sans seuil
	}
	ranks := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 25, Title: map[string]string{"fr": "Caporal"}, XPRequired: 16420},
		{ID: 26, Title: map[string]string{"fr": "Sergent"}}, // 25 n'est pas le dernier
	})
	id := analysis.BuildSpartanIdentity(raw, "fr", ranks)
	if id == nil || id.CareerRank == nil {
		t.Fatal("CareerRank: want non-nil")
	}
	if id.CareerRank.XPForNextRank != 16420 {
		t.Errorf("XPForNextRank: want 16420 (fallback catalog), got %d", id.CareerRank.XPForNextRank)
	}
	if id.CareerRank.ProgressPct != 50.0 {
		t.Errorf("ProgressPct: want 50 (8210/16420), got %.2f", id.CareerRank.ProgressPct)
	}
}

// TestBuildSpartanIdentity_TotalXPCumulative : l'XP de carrière cumulée =
// Σ XPRequired des rangs précédents + current_xp. Au rang max (current_xp=0)
// c'est ce total qu'on affiche (le « grand nombre »).
func TestBuildSpartanIdentity_TotalXPCumulative(t *testing.T) {
	raw := &domain.HomeSpartanIdentityRow{RankNumber: 3, CurrentXP: 50}
	ranks := mappings.NewRankCatalog("halo_infinite", []mappings.RankEntry{
		{ID: 1, Title: map[string]string{"fr": "R1"}, XPRequired: 100},
		{ID: 2, Title: map[string]string{"fr": "R2"}, XPRequired: 200},
		{ID: 3, Title: map[string]string{"fr": "R3"}, XPRequired: 300},
		{ID: 4, Title: map[string]string{"fr": "R4"}}, // rang 3 n'est pas le dernier
	})
	id := analysis.BuildSpartanIdentity(raw, "fr", ranks)
	if id == nil || id.CareerRank == nil {
		t.Fatal("CareerRank: want non-nil")
	}
	// cumulatif(rangs 1..2) = 100+200 = 300, + current_xp 50 = 350.
	if id.CareerRank.TotalXP != 350 {
		t.Errorf("TotalXP: want 350 (100+200+50), got %d", id.CareerRank.TotalXP)
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
