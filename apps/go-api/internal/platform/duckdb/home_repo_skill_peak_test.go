// Package duckdb — home_repo_skill_peak_test.go : tests unitaires pour les
// helpers pures de skill peak (audit #10 coverage extension).
//
// Cible : 0 accès DB, juste les normalizers, builders d'URL, comparateurs.
package duckdb

import (
	"database/sql"
	"testing"
	"time"

	titlepkg "levelup/go-api/internal/domain/title"
)

// TestBuildHomeSkillPeakBadgeURL_CSRResolverTitleAware : le résolveur CSR title-aware
// override l'insigne pour un titre additionnel (Halo 5) et laisse HINF intact.
func TestBuildHomeSkillPeakBadgeURL_CSRResolverTitleAware(t *testing.T) {
	SetCSRBadgeResolver(func(titleSlug, tier string, subTier int) string {
		if titleSlug == "halo_5" && tier == "Diamond" && subTier == 5 {
			return "https://cdn/h5-diamond5.png"
		}
		return ""
	})
	defer SetCSRBadgeResolver(nil) // ne pas polluer les autres tests

	// Halo 5 → URL officielle du résolveur.
	if got := buildHomeSkillPeakBadgeURLForThreshold("Diamond", "", 5, "halo_5", 0, 10); got == nil || *got != "https://cdn/h5-diamond5.png" {
		t.Errorf("badge h5 = %v, want l'URL du résolveur", got)
	}
	// Halo Infinite → résolveur renvoie "" → chemin static HINF (jamais l'URL h5).
	if got := buildHomeSkillPeakBadgeURLForThreshold("Diamond", "", 5, "halo_infinite", 0, 10); got == nil || *got == "https://cdn/h5-diamond5.png" {
		t.Errorf("badge HINF ne doit PAS passer par le résolveur h5, got %v", got)
	}
}

// ─── canonicalHomeSkillTierName ─────────────────────────────────────────

func TestCanonicalHomeSkillTierName_AllKnown(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"bronze":   "Bronze",
		"silver":   "Silver",
		"gold":     "Gold",
		"platinum": "Platinum",
		"diamond":  "Diamond",
		"onyx":     "Onyx",
	}
	for in, want := range cases {
		if got := canonicalHomeSkillTierName(in); got != want {
			t.Errorf("canonicalHomeSkillTierName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCanonicalHomeSkillTierName_CaseInsensitive(t *testing.T) {
	t.Parallel()
	cases := []string{"GOLD", "Gold", "gold", "GoLd"}
	for _, in := range cases {
		if got := canonicalHomeSkillTierName(in); got != "Gold" {
			t.Errorf("canonicalHomeSkillTierName(%q) = %q, want Gold", in, got)
		}
	}
}

func TestCanonicalHomeSkillTierName_TrimsWhitespace(t *testing.T) {
	t.Parallel()
	if got := canonicalHomeSkillTierName("  diamond  "); got != "Diamond" {
		t.Errorf("canonicalHomeSkillTierName(  diamond  ) = %q, want Diamond", got)
	}
}

func TestCanonicalHomeSkillTierName_UnknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	cases := []string{"", "  ", "unknown", "platine", "uber"}
	for _, in := range cases {
		if got := canonicalHomeSkillTierName(in); got != "" {
			t.Errorf("canonicalHomeSkillTierName(%q) = %q, want empty", in, got)
		}
	}
}

// ─── parseHomeSkillPeakSubTier ──────────────────────────────────────────

func TestParseHomeSkillPeakSubTier_Numeric(t *testing.T) {
	t.Parallel()
	cases := map[string]int{
		"1": 1, "2": 2, "3": 3, "4": 4, "5": 5, "6": 6,
		"  3  ": 3,
		"42":    42, // accepte les valeurs hors range (caller filtre)
	}
	for in, want := range cases {
		if got := parseHomeSkillPeakSubTier(in); got != want {
			t.Errorf("parseHomeSkillPeakSubTier(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseHomeSkillPeakSubTier_RomanNumerals(t *testing.T) {
	t.Parallel()
	cases := map[string]int{
		"I":   1,
		"II":  2,
		"III": 3,
		"IV":  4,
		"V":   5,
		"VI":  6,
		"vi":  6, // strings.ToUpper côté impl
	}
	for in, want := range cases {
		if got := parseHomeSkillPeakSubTier(in); got != want {
			t.Errorf("parseHomeSkillPeakSubTier(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseHomeSkillPeakSubTier_Empty(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "   ", "\t"} {
		if got := parseHomeSkillPeakSubTier(in); got != 0 {
			t.Errorf("parseHomeSkillPeakSubTier(%q) = %d, want 0", in, got)
		}
	}
}

func TestParseHomeSkillPeakSubTier_Unknown(t *testing.T) {
	t.Parallel()
	cases := []string{"abc", "VII", "XX", "I.I"}
	for _, in := range cases {
		if got := parseHomeSkillPeakSubTier(in); got != 0 {
			t.Errorf("parseHomeSkillPeakSubTier(%q) = %d, want 0", in, got)
		}
	}
}

// ─── normalizeHomeSkillPeakBadgeParts ───────────────────────────────────

func TestNormalizeHomeSkillPeakBadgeParts_TierExplicit(t *testing.T) {
	t.Parallel()
	tier, sub := normalizeHomeSkillPeakBadgeParts("gold", "", 3)
	if tier != "Gold" || sub != 3 {
		t.Errorf("(tier, sub) = (%q, %d), want (Gold, 3)", tier, sub)
	}
}

func TestNormalizeHomeSkillPeakBadgeParts_TierFromLabel(t *testing.T) {
	t.Parallel()
	// Tier vide → extraire le tier depuis tierLabel.
	tier, sub := normalizeHomeSkillPeakBadgeParts("", "platinum 5", 0)
	if tier != "Platinum" || sub != 5 {
		t.Errorf("(tier, sub) = (%q, %d), want (Platinum, 5)", tier, sub)
	}
}

func TestNormalizeHomeSkillPeakBadgeParts_SubTierFromLabel(t *testing.T) {
	t.Parallel()
	// Tier connu, sub-tier 0 → extraire sub-tier depuis tierLabel.
	tier, sub := normalizeHomeSkillPeakBadgeParts("diamond", "Diamond IV", 0)
	if tier != "Diamond" || sub != 4 {
		t.Errorf("(tier, sub) = (%q, %d), want (Diamond, 4)", tier, sub)
	}
}

func TestNormalizeHomeSkillPeakBadgeParts_OnyxNoSubTier(t *testing.T) {
	t.Parallel()
	// Onyx n'a pas de sub-tier → label "Onyx" sans nombre.
	tier, sub := normalizeHomeSkillPeakBadgeParts("onyx", "Onyx", 0)
	if tier != "Onyx" || sub != 0 {
		t.Errorf("(tier, sub) = (%q, %d), want (Onyx, 0)", tier, sub)
	}
}

func TestNormalizeHomeSkillPeakBadgeParts_AllEmpty(t *testing.T) {
	t.Parallel()
	tier, sub := normalizeHomeSkillPeakBadgeParts("", "", 0)
	if tier != "" || sub != 0 {
		t.Errorf("(tier, sub) = (%q, %d), want (empty, 0)", tier, sub)
	}
}

func TestNormalizeHomeSkillPeakBadgeParts_PrefersExplicitSubTier(t *testing.T) {
	t.Parallel()
	// subTier > 0 ne doit pas être overridé par tierLabel.
	tier, sub := normalizeHomeSkillPeakBadgeParts("gold", "Gold 5", 3)
	if tier != "Gold" || sub != 3 {
		t.Errorf("(tier, sub) = (%q, %d), want (Gold, 3) — explicit subTier prioritaire", tier, sub)
	}
}

// ─── unrankedBadgeURL ───────────────────────────────────────────────────

func TestUnrankedBadgeURL_BasicHaloSlug(t *testing.T) {
	t.Parallel()
	got := unrankedBadgeURL(5, "halo_infinite")
	want := "/static/ranks/halo_infinite/unranked_5.png"
	if got == nil || *got != want {
		t.Errorf("unrankedBadgeURL(5, halo_infinite) = %v, want %q", got, want)
	}
}

func TestUnrankedBadgeURL_EmptySlugFallback(t *testing.T) {
	t.Parallel()
	// titleSlug == "" → fallback sur titlepkg.DefaultSlug.
	got := unrankedBadgeURL(3, "")
	want := "/static/ranks/" + titlepkg.DefaultSlug + "/unranked_3.png"
	if got == nil || *got != want {
		t.Errorf("unrankedBadgeURL(3, empty) = %v, want %q", got, want)
	}
}

func TestUnrankedBadgeURL_NegativeClampsToZero(t *testing.T) {
	t.Parallel()
	got := unrankedBadgeURL(-5, "halo_infinite")
	want := "/static/ranks/halo_infinite/unranked_0.png"
	if got == nil || *got != want {
		t.Errorf("unrankedBadgeURL(-5) = %v, want unranked_0", got)
	}
}

func TestUnrankedBadgeURL_OverNineClampsTo9(t *testing.T) {
	t.Parallel()
	got := unrankedBadgeURL(100, "halo_infinite")
	want := "/static/ranks/halo_infinite/unranked_9.png"
	if got == nil || *got != want {
		t.Errorf("unrankedBadgeURL(100) = %v, want unranked_9", got)
	}
}

func TestUnrankedBadgeURL_BordersZeroAndNine(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 9} {
		got := unrankedBadgeURL(n, "halo_infinite")
		if got == nil {
			t.Errorf("unrankedBadgeURL(%d) = nil", n)
			continue
		}
	}
}

// Régression G3 : un titre additionnel (halo_5) n'a pas de dossier ranks/ propre.
// Le badge de placement unranked_N.png doit se résoudre sous le titre par DÉFAUT
// (static/ranks/halo_infinite/), sinon /static/ranks/halo_5/unranked_0.png → 404 →
// le front affiche « ? » au lieu de l'image unranked_0.
func TestUnrankedBadgeURL_AdditionalTitleUsesDefaultFolder(t *testing.T) {
	t.Parallel()
	got := unrankedBadgeURL(0, "halo_5")
	want := "/static/ranks/" + titlepkg.DefaultSlug + "/unranked_0.png"
	if got == nil || *got != want {
		t.Errorf("unrankedBadgeURL(0, halo_5) = %v, want %q", got, want)
	}
}

// ─── buildHomeSkillPeakBadgeURL ─────────────────────────────────────────

func TestBuildHomeSkillPeakBadgeURL_OnyxIgnoresSubTier(t *testing.T) {
	t.Parallel()
	got := buildHomeSkillPeakBadgeURL("onyx", "", 0, "halo_infinite", 0)
	want := "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"
	if got == nil || *got != want {
		t.Errorf("buildHomeSkillPeakBadgeURL(onyx) = %v, want %q", got, want)
	}
}

func TestBuildHomeSkillPeakBadgeURL_GoldWithSubTier(t *testing.T) {
	t.Parallel()
	got := buildHomeSkillPeakBadgeURL("gold", "", 3, "halo_infinite", 0)
	want := "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png"
	if got == nil || *got != want {
		t.Errorf("buildHomeSkillPeakBadgeURL(gold, 3) = %v, want %q", got, want)
	}
}

func TestBuildHomeSkillPeakBadgeURL_PlacementWithRemaining(t *testing.T) {
	t.Parallel()
	// Pas de tier + remaining > 0 → URL unranked.
	got := buildHomeSkillPeakBadgeURL("", "", 0, "halo_infinite", 7)
	// remaining=7 → completed=10-7=3 → unranked_3.png
	want := "/static/ranks/halo_infinite/unranked_3.png"
	if got == nil || *got != want {
		t.Errorf("buildHomeSkillPeakBadgeURL(placement remaining=7) = %v, want %q", got, want)
	}
}

func TestBuildHomeSkillPeakBadgeURL_NoTierNoRemainingReturnsNil(t *testing.T) {
	t.Parallel()
	// Tier vide + remaining = 0 → nil (rien à afficher).
	if got := buildHomeSkillPeakBadgeURL("", "", 0, "halo_infinite", 0); got != nil {
		t.Errorf("buildHomeSkillPeakBadgeURL(no tier no remaining) = %v, want nil", got)
	}
}

func TestBuildHomeSkillPeakBadgeURL_SubTierOutOfRangeReturnsNil(t *testing.T) {
	t.Parallel()
	// Tier connu mais subTier hors [1..6] (non-onyx) → nil.
	for _, st := range []int{0, -1, 7} {
		if got := buildHomeSkillPeakBadgeURL("gold", "", st, "halo_infinite", 0); got != nil {
			t.Errorf("buildHomeSkillPeakBadgeURL(gold, subTier=%d) = %v, want nil", st, got)
		}
	}
}

func TestBuildHomeSkillPeakBadgeURL_EmptyTitleSlugStripsSlug(t *testing.T) {
	t.Parallel()
	// titleSlug "" (LUSR cross-titre) → l'URL ne contient PAS le slug.
	got := buildHomeSkillPeakBadgeURL("gold", "", 3, "", 0)
	if got == nil {
		t.Fatal("got nil, want URL without slug")
	}
	// Doit contenir /static/ranks/120px... (sans halo_infinite/).
	if *got != "/static/ranks/120px-HINF-CSR_Gold3.png" {
		t.Errorf("got %q, want /static/ranks/120px-HINF-CSR_Gold3.png (no slug)", *got)
	}
}

func TestBuildHomeSkillPeakBadgeURL_TierLabelFallback(t *testing.T) {
	t.Parallel()
	// Tier vide + tierLabel "Platinum 5" → la fonction extrait tier+sub.
	got := buildHomeSkillPeakBadgeURL("", "Platinum 5", 0, "halo_infinite", 0)
	want := "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png"
	if got == nil || *got != want {
		t.Errorf("got %v, want %q", got, want)
	}
}

// ─── classifyPeakType ───────────────────────────────────────────────────

func TestClassifyPeakType_RegistryRankedFlag(t *testing.T) {
	t.Parallel()
	registry := map[string]peakRegistryInfo{
		"m1": {isRanked: true, playlistName: "Quick Play", pairName: "Slayer"},
	}
	pr := peakRow{matchID: "m1"}
	if got := classifyPeakType(pr, registry); got != "CSR" {
		t.Errorf("registry isRanked=true → CSR, got %q", got)
	}
}

func TestClassifyPeakType_RegistryRankedInPlaylistName(t *testing.T) {
	t.Parallel()
	registry := map[string]peakRegistryInfo{
		"m1": {isRanked: false, playlistName: "Ranked Arena", pairName: ""},
	}
	if got := classifyPeakType(peakRow{matchID: "m1"}, registry); got != "CSR" {
		t.Errorf("registry playlistName contains ranked → CSR, got %q", got)
	}
}

func TestClassifyPeakType_RegistryRankedInPairName(t *testing.T) {
	t.Parallel()
	registry := map[string]peakRegistryInfo{
		"m1": {isRanked: false, playlistName: "Quick Play", pairName: "Slayer Ranked"},
	}
	if got := classifyPeakType(peakRow{matchID: "m1"}, registry); got != "CSR" {
		t.Errorf("registry pairName contains ranked → CSR, got %q", got)
	}
}

func TestClassifyPeakType_RegistryUnrankedReturnsLUSR(t *testing.T) {
	t.Parallel()
	registry := map[string]peakRegistryInfo{
		"m1": {isRanked: false, playlistName: "Quick Play", pairName: "Slayer"},
	}
	if got := classifyPeakType(peakRow{matchID: "m1"}, registry); got != "LUSR" {
		t.Errorf("registry unranked → LUSR, got %q", got)
	}
}

func TestClassifyPeakType_NoRegistryFallbackRatingTypeCSR(t *testing.T) {
	t.Parallel()
	// Pas d'entrée dans le registry → fallback sur pr.ratingType.
	pr := peakRow{matchID: "unknown", ratingType: "CSR"}
	if got := classifyPeakType(pr, map[string]peakRegistryInfo{}); got != "CSR" {
		t.Errorf("no registry, ratingType CSR → CSR, got %q", got)
	}
}

func TestClassifyPeakType_NoRegistryFallbackRatingTypeLowercase(t *testing.T) {
	t.Parallel()
	pr := peakRow{matchID: "unknown", ratingType: "csr"}
	if got := classifyPeakType(pr, map[string]peakRegistryInfo{}); got != "CSR" {
		t.Errorf("no registry, ratingType csr → CSR (case-insensitive), got %q", got)
	}
}

func TestClassifyPeakType_NoRegistryDefaultLUSR(t *testing.T) {
	t.Parallel()
	pr := peakRow{matchID: "unknown", ratingType: ""}
	if got := classifyPeakType(pr, map[string]peakRegistryInfo{}); got != "LUSR" {
		t.Errorf("no registry, empty ratingType → LUSR, got %q", got)
	}
}

func TestClassifyPeakType_CaseInsensitiveRanked(t *testing.T) {
	t.Parallel()
	// Le code utilise strings.ToLower donc "RANKED" doit matcher.
	registry := map[string]peakRegistryInfo{
		"m1": {isRanked: false, playlistName: "RANKED ARENA", pairName: ""},
	}
	if got := classifyPeakType(peakRow{matchID: "m1"}, registry); got != "CSR" {
		t.Errorf("uppercase RANKED → CSR, got %q", got)
	}
}

// ─── isBetterPeak ───────────────────────────────────────────────────────

func TestIsBetterPeak_HigherRatingWins(t *testing.T) {
	t.Parallel()
	cand := peakRow{ratingValue: 1500}
	curr := peakRow{ratingValue: 1200}
	if !isBetterPeak(cand, curr) {
		t.Error("1500 > 1200 should be better")
	}
	if isBetterPeak(curr, cand) {
		t.Error("1200 < 1500 should not be better")
	}
}

func TestIsBetterPeak_EqualRatingMoreRecentWins(t *testing.T) {
	t.Parallel()
	earlier := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cand := peakRow{ratingValue: 1500, recency: sql.NullTime{Time: later, Valid: true}}
	curr := peakRow{ratingValue: 1500, recency: sql.NullTime{Time: earlier, Valid: true}}
	if !isBetterPeak(cand, curr) {
		t.Error("later recency should be better with equal rating")
	}
}

func TestIsBetterPeak_EqualRatingValidRecencyWinsOverInvalid(t *testing.T) {
	t.Parallel()
	cand := peakRow{ratingValue: 1500, recency: sql.NullTime{Time: time.Now(), Valid: true}}
	curr := peakRow{ratingValue: 1500, recency: sql.NullTime{Valid: false}}
	if !isBetterPeak(cand, curr) {
		t.Error("valid recency should beat invalid (nil) recency")
	}
}

func TestIsBetterPeak_EqualEverythingHigherSubTierWins(t *testing.T) {
	t.Parallel()
	// Même rating, même recency invalide → sub_tier décide.
	cand := peakRow{ratingValue: 1500, subTier: 4}
	curr := peakRow{ratingValue: 1500, subTier: 2}
	if !isBetterPeak(cand, curr) {
		t.Error("higher subTier should win with equal rating+recency")
	}
}

func TestIsBetterPeak_TieBreakerMatchID(t *testing.T) {
	t.Parallel()
	// Tout égal → matchID DESC (lexicographique).
	cand := peakRow{matchID: "m-zzz", ratingValue: 1500, subTier: 3}
	curr := peakRow{matchID: "m-aaa", ratingValue: 1500, subTier: 3}
	if !isBetterPeak(cand, curr) {
		t.Error("m-zzz > m-aaa should win tie-break")
	}
}

func TestIsBetterPeak_AllIdenticalNotBetter(t *testing.T) {
	t.Parallel()
	// Strictement identique → false (pas strictement supérieur sur matchID).
	row := peakRow{matchID: "m1", ratingValue: 1500, subTier: 3}
	if isBetterPeak(row, row) {
		t.Error("identical rows should not be 'better'")
	}
}

func TestIsBetterPeak_SameRecencyTimeUsesSubTier(t *testing.T) {
	t.Parallel()
	// Quand les 2 recency sont valides et exactement égales, on tombe sur subTier.
	now := time.Now()
	cand := peakRow{matchID: "m1", ratingValue: 1500, subTier: 5, recency: sql.NullTime{Time: now, Valid: true}}
	curr := peakRow{matchID: "m1", ratingValue: 1500, subTier: 3, recency: sql.NullTime{Time: now, Valid: true}}
	if !isBetterPeak(cand, curr) {
		t.Error("equal recency time → fallback subTier")
	}
}
