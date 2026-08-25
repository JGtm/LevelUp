// Package sync — transforms_helpers_unit_test.go : tests unitaires des helpers purs.
//
// Couvre les fonctions stateless qui étaient à 0% : findCoreStats, isRankedPlaylist,
// isFirefightMatch, ExtractTeamScoresByID, asString, strPtrNonEmpty, coalesceStrPtr,
// intPtrFrom, floatPtrFrom, intFrom, int64From.
package sync

import (
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// asString
// ─────────────────────────────────────────────────────────────────────────────

func TestAsString_Nil(t *testing.T) {
	if got := asString(nil); got != "" {
		t.Errorf("asString(nil) = %q, want empty", got)
	}
}

func TestAsString_Valid(t *testing.T) {
	if got := asString("hello"); got != "hello" {
		t.Errorf("asString(\"hello\") = %q", got)
	}
}

func TestAsString_NonString(t *testing.T) {
	if got := asString(42); got != "" {
		t.Errorf("asString(42) = %q, want empty", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// strPtrNonEmpty
// ─────────────────────────────────────────────────────────────────────────────

func TestStrPtr_Empty(t *testing.T) {
	if got := strPtrNonEmpty(""); got != nil {
		t.Errorf("strPtrNonEmpty(\"\") should be nil")
	}
}

func TestStrPtr_NonEmpty(t *testing.T) {
	got := strPtrNonEmpty("abc")
	if got == nil || *got != "abc" {
		t.Errorf("strPtrNonEmpty(\"abc\") = %v, want ptr to \"abc\"", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// coalesceStrPtr
// ─────────────────────────────────────────────────────────────────────────────

func TestCoalesceStrPtr_FirstNonEmpty(t *testing.T) {
	a := "first"
	b := "second"
	got := coalesceStrPtr(&a, &b)
	if got == nil || *got != "first" {
		t.Error("should return first non-empty")
	}
}

func TestCoalesceStrPtr_FirstEmpty(t *testing.T) {
	a := ""
	b := "second"
	got := coalesceStrPtr(&a, &b)
	if got == nil || *got != "second" {
		t.Error("should return second when first is empty")
	}
}

func TestCoalesceStrPtr_FirstNil(t *testing.T) {
	b := "fallback"
	got := coalesceStrPtr(nil, &b)
	if got == nil || *got != "fallback" {
		t.Error("should return second when first is nil")
	}
}

func TestCoalesceStrPtr_BothNil(t *testing.T) {
	got := coalesceStrPtr(nil, nil)
	if got != nil {
		t.Error("should return nil when both nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// intPtrFrom / floatPtrFrom / intFrom / int64From
// ─────────────────────────────────────────────────────────────────────────────

func TestIntPtrFrom_Present(t *testing.T) {
	m := map[string]any{"kills": float64(15)}
	got := intPtrFrom(m, "kills")
	if got == nil || *got != 15 {
		t.Errorf("intPtrFrom should return 15, got %v", got)
	}
}

func TestIntPtrFrom_Missing(t *testing.T) {
	m := map[string]any{}
	got := intPtrFrom(m, "kills")
	if got != nil {
		t.Error("intPtrFrom on missing key should return nil")
	}
}

func TestIntPtrFrom_WrongType(t *testing.T) {
	m := map[string]any{"kills": "not-a-number"}
	got := intPtrFrom(m, "kills")
	if got != nil {
		t.Error("intPtrFrom on string should return nil")
	}
}

func TestFloatPtrFrom_Present(t *testing.T) {
	m := map[string]any{"dmg": float64(1234.5)}
	got := floatPtrFrom(m, "dmg")
	if got == nil || *got != 1234.5 {
		t.Errorf("floatPtrFrom should return 1234.5, got %v", got)
	}
}

func TestFloatPtrFrom_Missing(t *testing.T) {
	got := floatPtrFrom(map[string]any{}, "dmg")
	if got != nil {
		t.Error("floatPtrFrom on missing key should return nil")
	}
}

func TestIntFrom_Present(t *testing.T) {
	m := map[string]any{"score": float64(42)}
	if got := intFrom(m, "score"); got != 42 {
		t.Errorf("intFrom = %d, want 42", got)
	}
}

func TestIntFrom_Missing(t *testing.T) {
	if got := intFrom(map[string]any{}, "score"); got != 0 {
		t.Errorf("intFrom missing = %d, want 0", got)
	}
}

func TestInt64From_Present(t *testing.T) {
	m := map[string]any{"medal_id": float64(123456789)}
	if got := int64From(m, "medal_id"); got != 123456789 {
		t.Errorf("int64From = %d, want 123456789", got)
	}
}

func TestInt64From_Missing(t *testing.T) {
	if got := int64From(map[string]any{}, "x"); got != 0 {
		t.Errorf("int64From missing = %d, want 0", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// findCoreStats
// ─────────────────────────────────────────────────────────────────────────────

func TestFindCoreStats_Valid(t *testing.T) {
	player := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{
				"Stats": map[string]any{
					"CoreStats": map[string]any{
						"Kills":  float64(10),
						"Deaths": float64(5),
					},
				},
			},
		},
	}
	core := findCoreStats(player)
	if core == nil {
		t.Fatal("findCoreStats returned nil")
	}
	if intFrom(core, "Kills") != 10 {
		t.Errorf("Kills = %d, want 10", intFrom(core, "Kills"))
	}
}

func TestFindCoreStats_NoPlayerTeamStats(t *testing.T) {
	player := map[string]any{}
	if core := findCoreStats(player); core != nil {
		t.Error("expected nil when no PlayerTeamStats")
	}
}

func TestFindCoreStats_EmptyArray(t *testing.T) {
	player := map[string]any{"PlayerTeamStats": []any{}}
	if core := findCoreStats(player); core != nil {
		t.Error("expected nil when empty PlayerTeamStats")
	}
}

func TestFindCoreStats_NoStats(t *testing.T) {
	player := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{"TeamId": float64(0)},
		},
	}
	if core := findCoreStats(player); core != nil {
		t.Error("expected nil when Stats missing")
	}
}

func TestFindCoreStats_NoCoreStats(t *testing.T) {
	player := map[string]any{
		"PlayerTeamStats": []any{
			map[string]any{
				"Stats": map[string]any{
					"OtherStats": map[string]any{},
				},
			},
		},
	}
	if core := findCoreStats(player); core != nil {
		t.Error("expected nil when CoreStats missing")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// isRankedPlaylist
// ─────────────────────────────────────────────────────────────────────────────

func TestIsRankedPlaylist_ByName(t *testing.T) {
	info := map[string]any{
		"Playlist": map[string]any{
			"PublicName": "Ranked Arena",
		},
	}
	if !isRankedPlaylist(info) {
		t.Error("should detect ranked by name")
	}
}

func TestIsRankedPlaylist_ByTag(t *testing.T) {
	info := map[string]any{
		"Playlist": map[string]any{
			"PublicName": "Competitive",
			"Tags":       []any{"Ranked"},
		},
	}
	if !isRankedPlaylist(info) {
		t.Error("should detect ranked by tag")
	}
}

func TestIsRankedPlaylist_NoPlaylist(t *testing.T) {
	info := map[string]any{}
	if isRankedPlaylist(info) {
		t.Error("should not detect ranked without playlist")
	}
}

func TestIsRankedPlaylist_CaseInsensitive(t *testing.T) {
	info := map[string]any{
		"Playlist": map[string]any{
			"PublicName": "RANKED arena",
		},
	}
	if !isRankedPlaylist(info) {
		t.Error("should be case-insensitive")
	}
}

func TestIsRankedPlaylist_UnrankedPlaylist(t *testing.T) {
	info := map[string]any{
		"Playlist": map[string]any{
			"PublicName": "Quick Play",
			"Tags":       []any{"social"},
		},
	}
	if isRankedPlaylist(info) {
		t.Error("Quick Play should not be ranked")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// isFirefightMatch
// ─────────────────────────────────────────────────────────────────────────────

func TestIsFirefightMatch_ByCategory(t *testing.T) {
	for _, cat := range []float64{22, 32, 40, 41, 42} {
		info := map[string]any{"GameVariantCategory": cat}
		if !isFirefightMatch(info) {
			t.Errorf("category %v should be firefight", cat)
		}
	}
}

func TestIsFirefightMatch_ByVariantName(t *testing.T) {
	info := map[string]any{
		"UgcGameVariant": map[string]any{
			"PublicName": "Firefight Heroic",
		},
	}
	if !isFirefightMatch(info) {
		t.Error("should detect firefight by variant name")
	}
}

func TestIsFirefightMatch_NotFirefight(t *testing.T) {
	info := map[string]any{
		"GameVariantCategory": float64(6),
		"UgcGameVariant": map[string]any{
			"PublicName": "Slayer",
		},
	}
	if isFirefightMatch(info) {
		t.Error("Slayer should not be firefight")
	}
}

func TestIsFirefightMatch_Empty(t *testing.T) {
	if isFirefightMatch(map[string]any{}) {
		t.Error("empty map should not be firefight")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ExtractTeamScoresByID
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractTeamScoresByID_TwoTeams(t *testing.T) {
	match := map[string]any{
		"Teams": []any{
			map[string]any{
				"TeamId": float64(0),
				"Stats": map[string]any{
					"CoreStats": map[string]any{"Score": float64(50)},
				},
			},
			map[string]any{
				"TeamId": float64(1),
				"Stats": map[string]any{
					"CoreStats": map[string]any{"Score": float64(47)},
				},
			},
		},
	}
	t0, t1 := ExtractTeamScoresByID(match)
	if t0 == nil || *t0 != 50 {
		t.Errorf("team0 = %v, want 50", t0)
	}
	if t1 == nil || *t1 != 47 {
		t.Errorf("team1 = %v, want 47", t1)
	}
}

func TestExtractTeamScoresByID_NoTeams(t *testing.T) {
	t0, t1 := ExtractTeamScoresByID(map[string]any{})
	if t0 != nil || t1 != nil {
		t.Error("should return nil, nil when no teams")
	}
}

func TestExtractTeamScoresByID_PartialTeam(t *testing.T) {
	match := map[string]any{
		"Teams": []any{
			map[string]any{
				"TeamId": float64(0),
				"Stats": map[string]any{
					"CoreStats": map[string]any{"Score": float64(25)},
				},
			},
		},
	}
	t0, t1 := ExtractTeamScoresByID(match)
	if t0 == nil || *t0 != 25 {
		t.Errorf("team0 = %v, want 25", t0)
	}
	if t1 != nil {
		t.Error("team1 should be nil when missing")
	}
}

func TestExtractTeamScoresByID_MissingStats(t *testing.T) {
	match := map[string]any{
		"Teams": []any{
			map[string]any{"TeamId": float64(0)},
		},
	}
	t0, t1 := ExtractTeamScoresByID(match)
	if t0 != nil || t1 != nil {
		t.Error("should return nil when Stats missing")
	}
}
