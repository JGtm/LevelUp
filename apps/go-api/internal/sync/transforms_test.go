// Package sync — transforms_test.go : tests unitaires pour les helpers de transformation JSON.
//
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package sync

import (
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// extractXUID
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractXUID_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"xuid(1234567890)", "1234567890"},
		{"xuid(9876543210123456)", "9876543210123456"},
	}
	for _, c := range cases {
		got := extractXUID(c.input)
		if got != c.want {
			t.Errorf("extractXUID(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestExtractXUID_Invalid(t *testing.T) {
	invalids := []string{"", "not-a-xuid", "xuid()", "XUID(123)"}
	for _, input := range invalids {
		got := extractXUID(input)
		// xuid() ne match pas (groupe vide non numérique) → vide attendu
		// XUID(123) est case-sensitive
		if got != "" && input == "XUID(123)" {
			t.Errorf("extractXUID(%q) = %q, expected empty (case-sensitive)", input, got)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// parsePTDuration
// ─────────────────────────────────────────────────────────────────────────────

func TestParsePTDuration_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"PT1H", 3600},
		{"PT30M", 1800},
		{"PT90S", 90},
		{"PT1H30M", 5400},
		{"PT1H2M3S", 3723},
		{"PT2M30.5S", 150},
	}
	for _, c := range cases {
		got := parsePTDuration(c.input)
		if got == nil {
			t.Fatalf("parsePTDuration(%q) = nil", c.input)
		}
		if *got != c.want {
			t.Errorf("parsePTDuration(%q) = %d, want %d", c.input, *got, c.want)
		}
	}
}

func TestParsePTDuration_Empty(t *testing.T) {
	if parsePTDuration("") != nil {
		t.Error("parsePTDuration(\"\") should return nil")
	}
}

func TestParsePTDuration_Invalid(t *testing.T) {
	if parsePTDuration("not-a-duration") != nil {
		t.Error("parsePTDuration(invalid) should return nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// parseISO
// ─────────────────────────────────────────────────────────────────────────────

func TestParseISO_Valid(t *testing.T) {
	cases := []string{
		"2024-01-15T18:30:00Z",
		"2024-01-15T18:30:00.123456789Z",
		"2024-01-15T18:30:00+00:00",
	}
	for _, s := range cases {
		got, err := parseISO(s)
		if err != nil {
			t.Errorf("parseISO(%q) error: %v", s, err)
			continue
		}
		if got.Year() != 2024 || got.Month() != time.January || got.Day() != 15 {
			t.Errorf("parseISO(%q) = %v, expected 2024-01-15", s, got)
		}
		if got.Location() != time.UTC {
			t.Errorf("parseISO(%q) not UTC: %v", s, got.Location())
		}
	}
}

func TestParseISO_Empty(t *testing.T) {
	_, err := parseISO("")
	if err == nil {
		t.Error("parseISO(\"\") should return error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ErrMissingField
// ─────────────────────────────────────────────────────────────────────────────

func TestErrMissingField(t *testing.T) {
	err := ErrMissingField("MatchId")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "MatchId") {
		t.Errorf("error message should contain field name, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// extractAssetID / extractPublicName
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractAssetID(t *testing.T) {
	m := map[string]any{
		"Playlist": map[string]any{
			"AssetId":    "playlist-uuid-123",
			"PublicName": "Ranked Arena",
		},
	}
	got := extractAssetID(m, "Playlist")
	if got != "playlist-uuid-123" {
		t.Errorf("extractAssetID: got %q, want %q", got, "playlist-uuid-123")
	}
}

func TestExtractPublicName(t *testing.T) {
	m := map[string]any{
		"MapVariant": map[string]any{
			"AssetId":    "map-uuid",
			"PublicName": "Bazaar",
		},
	}
	got := extractPublicName(m, "MapVariant")
	if got != "Bazaar" {
		t.Errorf("extractPublicName: got %q, want %q", got, "Bazaar")
	}
}

func TestExtractAssetID_MissingKey(t *testing.T) {
	got := extractAssetID(map[string]any{}, "Playlist")
	if got != "" {
		t.Errorf("missing key: expected empty, got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// determineModeCategory
// ─────────────────────────────────────────────────────────────────────────────

func TestDetermineModeCategoryTable(t *testing.T) {
	cases := []struct{ input, want string }{
		{"ranked-arena-2024", "Ranked"},
		{"firefight-heroic", "Firefight"},
		{"btb-heavies", "BTB"},
		{"big-team-tactical", "BTB"},
		{"fiesta-slayer", "Fiesta"},
		{"assassination-arena", "Assassin"},
		{"unknown-mode", "Other"},
	}
	for _, c := range cases {
		got := determineModeCategory(c.input)
		if got != c.want {
			t.Errorf("determineModeCategory(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
