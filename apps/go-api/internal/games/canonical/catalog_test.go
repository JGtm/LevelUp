// Package canonical — catalog_test.go : tests pour les enums Experience +
// ModeCanonical du catalogue Playlists/Pairs/Maps (audit #10 coverage).
package canonical

import (
	"testing"
)

func TestIsKnownExperience_AllValid(t *testing.T) {
	t.Parallel()
	for _, e := range AllExperiences() {
		if !IsKnownExperience(e) {
			t.Errorf("IsKnownExperience(%q) returned false, expected true", e)
		}
	}
}

func TestIsKnownExperience_Invalid(t *testing.T) {
	t.Parallel()
	cases := []Experience{
		Experience(""),
		Experience("not_a_real_xp"),
		Experience("ranked_arena"), // pas dans l'enum (granularité plus fine)
	}
	for _, in := range cases {
		if IsKnownExperience(in) {
			t.Errorf("IsKnownExperience(%q) returned true, expected false", in)
		}
	}
}

func TestAllExperiences_LenAndContent(t *testing.T) {
	t.Parallel()
	all := AllExperiences()
	if len(all) != 8 {
		t.Errorf("AllExperiences() len = %d, want 8", len(all))
	}
	seen := make(map[Experience]struct{}, len(all))
	for _, e := range all {
		seen[e] = struct{}{}
	}
	required := []Experience{
		ExperienceRanked, ExperienceSocial, ExperienceBTB,
		ExperienceFirefight, ExperienceActionSack, ExperienceLimitedTime,
		ExperienceCustomBrowser, ExperienceUnknown,
	}
	for _, r := range required {
		if _, ok := seen[r]; !ok {
			t.Errorf("AllExperiences missing %q", r)
		}
	}
}

func TestExperienceValues_Stable(t *testing.T) {
	t.Parallel()
	// Les valeurs string sont stables (stockées en DB après catalog fetch).
	want := map[Experience]string{
		ExperienceRanked:        "ranked",
		ExperienceSocial:        "social",
		ExperienceBTB:           "btb",
		ExperienceFirefight:     "firefight",
		ExperienceActionSack:    "action_sack",
		ExperienceLimitedTime:   "limited_time",
		ExperienceCustomBrowser: "custom_browser",
		ExperienceUnknown:       "unknown",
	}
	for e, expected := range want {
		if string(e) != expected {
			t.Errorf("Experience value drift: %v should equal %q, got %q", e, expected, string(e))
		}
	}
}

func TestModeCanonicalValues_Stable(t *testing.T) {
	t.Parallel()
	// Les valeurs string sont stables (stockées en DB).
	want := map[ModeCanonical]string{
		ModeSlayer:        "slayer",
		ModeCTF:           "ctf",
		ModeOddball:       "oddball",
		ModeKOTH:          "koth",
		ModeStrongholds:   "strongholds",
		ModeExtraction:    "extraction",
		ModeFiesta:        "fiesta",
		ModeFirefightKOTR: "firefight_kotr",
		ModeAttrition:     "attrition",
		ModeStockpile:     "stockpile",
		ModeTotalControl:  "total_control",
		ModeUnknown:       "unknown",
	}
	for m, expected := range want {
		if string(m) != expected {
			t.Errorf("ModeCanonical value drift: %v should equal %q, got %q", m, expected, string(m))
		}
	}
}

func TestMatchTypeValues_Stable(t *testing.T) {
	t.Parallel()
	// Les valeurs MatchType sont stables.
	want := map[MatchType]string{
		MatchTypeRanked:    "ranked",
		MatchTypeSocial:    "social",
		MatchTypeCustom:    "custom",
		MatchTypeFirefight: "firefight",
		MatchTypeUnknownMT: "unknown",
	}
	for mt, expected := range want {
		if string(mt) != expected {
			t.Errorf("MatchType value drift: %v should equal %q, got %q", mt, expected, string(mt))
		}
	}
}

func TestRatingTypeValues_Stable(t *testing.T) {
	t.Parallel()
	want := map[RatingType]string{
		RatingTypeCSR:  "csr",
		RatingTypeLUSR: "lusr",
	}
	for rt, expected := range want {
		if string(rt) != expected {
			t.Errorf("RatingType value drift: %v should equal %q, got %q", rt, expected, string(rt))
		}
	}
}

func TestBucketValues_Stable(t *testing.T) {
	t.Parallel()
	want := map[Bucket]string{
		BucketDay:   "day",
		BucketWeek:  "week",
		BucketMonth: "month",
	}
	for b, expected := range want {
		if string(b) != expected {
			t.Errorf("Bucket value drift: %v should equal %q, got %q", b, expected, string(b))
		}
	}
}

func TestGroupByValues_Stable(t *testing.T) {
	t.Parallel()
	want := map[GroupBy]string{
		GroupByPlaylist: "playlist",
		GroupByMode:     "mode",
		GroupByMap:      "map",
	}
	for g, expected := range want {
		if string(g) != expected {
			t.Errorf("GroupBy value drift: %v should equal %q, got %q", g, expected, string(g))
		}
	}
}

func TestDefaultCareerHistoryLimit(t *testing.T) {
	t.Parallel()
	// La constante doit rester stable : changement = breaking pour caller.
	if DefaultCareerHistoryLimit != 50 {
		t.Errorf("DefaultCareerHistoryLimit = %d, want 50", DefaultCareerHistoryLimit)
	}
}

func TestOutcomeValues_Stable(t *testing.T) {
	t.Parallel()
	// Miroir test stable pour Outcome (déjà couvert IsKnownOutcome dans enums_test.go,
	// mais les literales en string ne sont pas frozen ailleurs).
	want := map[Outcome]string{
		OutcomeWin:  "win",
		OutcomeLoss: "loss",
		OutcomeTie:  "tie",
		OutcomeDNF:  "dnf",
	}
	for o, expected := range want {
		if string(o) != expected {
			t.Errorf("Outcome value drift: %v should equal %q, got %q", o, expected, string(o))
		}
	}
}
