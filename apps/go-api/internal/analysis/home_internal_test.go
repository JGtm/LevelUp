package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
)

// srRankCatalog reproduit la forme du catalog SR Halo 5 (halo5.BuildSpartanRankCatalog,
// non importable ici — cycle analysis→halo_5→canonical→analysis) : entrées « SR N »
// avec un seuil XP par rang. Suffit pour valider que buildHomeCareerRank résout le
// label SR via le catalog au lieu du fallback générique « Rang N ».
func srRankCatalog() *mappings.RankCatalog {
	const maxSR = 152
	entries := make([]mappings.RankEntry, 0, maxSR)
	for n := 1; n <= maxSR; n++ {
		label := "SR " + itoaHome(n)
		xp := 0
		if n < maxSR {
			xp = 50000 // seuil fictif uniforme : ne sert qu'au fallback xp_for_next
		}
		entries = append(entries, mappings.RankEntry{
			ID:         n,
			Title:      map[string]string{mappings.LocaleEN: label, mappings.LocaleFR: label},
			Subtitle:   map[string]string{},
			Tier:       map[string]string{},
			XPRequired: xp,
		})
	}
	return mappings.NewRankCatalog("halo_5", entries)
}

func itoaHome(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestBuildHomeCareerRank_SRCatalog_Label147 : avec un catalog de style SR (Halo 5),
// buildHomeCareerRank résout « SR 147 » (via lookupRankLabel→FullLabel), is_max=false
// (Next(147) existe), et N'ÉCRASE PAS l'xp_for_next déjà peuplé en DB (2 950 000).
// C'est exactement le chemin de la Home Halo 5 après injection du catalog SR.
func TestBuildHomeCareerRank_SRCatalog_Label147(t *testing.T) {
	cat := srRankCatalog()
	raw := &domain.HomeSpartanIdentityRow{
		RankNumber:    147,
		RankName:      nil, // career_progression.rank_name NULL (donnée existante)
		CurrentXP:     1234,
		XPForNextRank: 2950000, // valeur DB peuplée → ne doit pas être écrasée
	}

	got := buildHomeCareerRank(raw, "fr", cat)
	if got == nil {
		t.Fatal("buildHomeCareerRank = nil")
	}
	if got.RankTitle != "SR 147" {
		t.Errorf("RankTitle = %q, want 'SR 147' (pas le fallback 'Rang 147')", got.RankTitle)
	}
	if got.IsMaxRank {
		t.Error("IsMaxRank = true, want false (Next(147) existe)")
	}
	if got.XPForNextRank != 2950000 {
		t.Errorf("XPForNextRank = %d, want 2950000 (DB non écrasée par le fallback catalog)", got.XPForNextRank)
	}
	if got.NextRankTitle != "SR 148" {
		t.Errorf("NextRankTitle = %q, want 'SR 148'", got.NextRankTitle)
	}
}

// TestBuildHomeCareerRank_NoCatalog_FallbackRangN : sans catalog (ranks nil) et sans
// rank_name, la Home retombe sur le fallback générique « Rang N » — c'était l'état
// Halo 5 AVANT l'injection du catalog SR (régression à empêcher).
func TestBuildHomeCareerRank_NoCatalog_FallbackRangN(t *testing.T) {
	raw := &domain.HomeSpartanIdentityRow{RankNumber: 147}
	got := buildHomeCareerRank(raw, "fr", nil)
	if got == nil {
		t.Fatal("buildHomeCareerRank = nil")
	}
	if got.RankTitle != "Rang 147" {
		t.Errorf("RankTitle = %q, want 'Rang 147' (fallback sans catalog)", got.RankTitle)
	}
}

func TestRound1(t *testing.T) {
	if round1(1.55) != 1.6 {
		t.Errorf("round1(1.55) = %f, want 1.6", round1(1.55))
	}
}

func TestRound2_Home(t *testing.T) {
	if round2(1.555) != 1.56 {
		t.Errorf("round2(1.555) = %f, want 1.56", round2(1.555))
	}
}

func TestRound3(t *testing.T) {
	if round3(1.5555) != 1.556 {
		t.Errorf("round3(1.5555) = %f, want 1.556", round3(1.5555))
	}
}

func TestRound4(t *testing.T) {
	if round4(1.55555) != 1.5556 {
		t.Errorf("round4(1.55555) = %f, want 1.5556", round4(1.55555))
	}
}

func TestMeanRatio_Empty(t *testing.T) {
	if meanRatio(nil) != nil {
		t.Error("expected nil")
	}
}

func TestMeanRatio_WithValues(t *testing.T) {
	r1, r2 := 2.0, 4.0
	matches := []legacymatch.HomeMatchRow{
		{Ratio: &r1},
		{Ratio: &r2},
		{Ratio: nil},
	}
	got := meanRatio(matches)
	if got == nil || *got != 3.0 {
		t.Errorf("meanRatio = %v, want 3.0", got)
	}
}

func TestMeanAccuracy_Empty(t *testing.T) {
	if meanAccuracy(nil) != nil {
		t.Error("expected nil")
	}
}

func TestWinRate_Home_Empty(t *testing.T) {
	if winRate(nil) != 0 {
		t.Error("expected 0")
	}
}

func TestWinRate_Home_WithMatches(t *testing.T) {
	matches := []legacymatch.HomeMatchRow{
		{Outcome: homeOutcomeWin},
		{Outcome: homeOutcomeLoss},
		{Outcome: homeOutcomeWin},
	}
	wr := winRate(matches)
	if wr < 0.66 || wr > 0.67 {
		t.Errorf("winRate = %f, want ~0.667", wr)
	}
}

func TestBestRatioMatch_Empty(t *testing.T) {
	if bestRatioMatch(nil) != nil {
		t.Error("expected nil")
	}
}

func TestBestRatioMatch_AllNil(t *testing.T) {
	matches := []legacymatch.HomeMatchRow{{Ratio: nil}, {Ratio: nil}}
	if bestRatioMatch(matches) != nil {
		t.Error("expected nil")
	}
}

func TestBestRatioMatch_FindsBest(t *testing.T) {
	r1, r2, r3 := 1.5, 3.0, 2.0
	matches := []legacymatch.HomeMatchRow{
		{MatchID: "m1", Ratio: &r1},
		{MatchID: "m2", Ratio: &r2},
		{MatchID: "m3", Ratio: &r3},
	}
	best := bestRatioMatch(matches)
	if best == nil || best.MatchID != "m2" {
		t.Errorf("bestRatioMatch = %v, want m2", best)
	}
}

func TestOutcomeLabel_Win(t *testing.T) {
	l := outcomeLabel(homeOutcomeWin)
	if l == "" || l == "DNF" {
		t.Errorf("outcomeLabel(WIN) = %q", l)
	}
}

func TestOutcomeLabel_Unknown(t *testing.T) {
	if outcomeLabel(99) != "DNF" {
		t.Error("expected DNF for unknown")
	}
}

func TestOutcomeTone_Win(t *testing.T) {
	tone := outcomeTone(homeOutcomeWin)
	if tone == "" || tone == "dnf" {
		t.Errorf("outcomeTone(WIN) = %q", tone)
	}
}

func TestOutcomeTone_Unknown(t *testing.T) {
	if outcomeTone(99) != "dnf" {
		t.Error("expected dnf for unknown")
	}
}

func TestLatestSessionLabel_Empty(t *testing.T) {
	if latestSessionLabel(nil) != "" {
		t.Error("expected empty")
	}
}

func TestLatestSessionLabel_WithSessions(t *testing.T) {
	t1 := time.Now().Add(-time.Hour)
	t2 := time.Now()
	s1 := "Session 1"
	s2 := "Session 2"
	sessions := []legacymatch.HomeSessionRow{
		{SessionLabel: &s1, StartTime: &t1},
		{SessionLabel: &s2, StartTime: &t2},
	}
	got := latestSessionLabel(sessions)
	if got != "Session 2" {
		t.Errorf("latestSessionLabel = %q, want Session 2", got)
	}
}

func TestEarliestStartTime_Empty(t *testing.T) {
	if earliestStartTime(nil) != nil {
		t.Error("expected nil")
	}
}

func TestEarliestStartTime_FindsEarliest(t *testing.T) {
	t1 := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 3, 1, 8, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	matches := []legacymatch.HomeMatchRow{
		{StartTime: t1},
		{StartTime: t2},
		{StartTime: t3},
	}
	got := earliestStartTime(matches)
	if got == nil || !got.Equal(t2) {
		t.Errorf("earliestStartTime = %v, want %v", got, t2)
	}
}

func TestBuildRecentMedia_Empty(t *testing.T) {
	if BuildRecentMedia(nil, 5) != nil {
		t.Error("expected nil")
	}
}

func TestBuildRecentMedia_Limit(t *testing.T) {
	m1, m2, m3 := "m1", "m2", "m3"
	now := time.Now()
	media := []domain.HomeMediaRow{
		{FileName: "a.png", MatchID: &m1, MatchStartTime: &now},
		{FileName: "b.png", MatchID: &m2, MatchStartTime: &now},
		{FileName: "c.png", MatchID: &m3, MatchStartTime: &now},
	}
	result := BuildRecentMedia(media, 2)
	if len(result) != 2 {
		t.Errorf("expected 2, got %d", len(result))
	}
}

func TestBuildRecentMedia_SkipsEmpty(t *testing.T) {
	m1, m2 := "m1", "m2"
	now := time.Now()
	media := []domain.HomeMediaRow{
		{FileName: "", MatchID: &m1},
		{FileName: "a.png", MatchID: &m2, MatchStartTime: &now},
	}
	result := BuildRecentMedia(media, 5)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}
