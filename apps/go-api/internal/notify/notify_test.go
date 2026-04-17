// Package notify — tests unitaires pour les fonctions pures.
//
//go:build integration

package notify

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// discord.go — helpers
// ---------------------------------------------------------------------------

func TestBoolVal(t *testing.T) {
	m := map[string]any{"flag": true, "off": false}
	if !boolVal(m, "flag") {
		t.Error("expected true for 'flag'")
	}
	if boolVal(m, "off") {
		t.Error("expected false for 'off'")
	}
	if boolVal(m, "missing") {
		t.Error("expected false for missing key")
	}
}

func TestBoolValDefault(t *testing.T) {
	m := map[string]any{"flag": true}
	if !boolValDefault(m, "missing", true) {
		t.Error("expected default true")
	}
	if boolValDefault(m, "missing", false) {
		t.Error("expected default false")
	}
}

func TestStrVal(t *testing.T) {
	m := map[string]any{"name": "test"}
	if strVal(m, "name") != "test" {
		t.Error("expected 'test'")
	}
	if strVal(m, "missing") != "" {
		t.Error("expected empty string for missing key")
	}
}

func TestStrValDefault(t *testing.T) {
	m := map[string]any{}
	if strValDefault(m, "key", "default") != "default" {
		t.Error("expected 'default'")
	}
}

func TestT_UnknownKey(t *testing.T) {
	result := T("nonexistent_key_xyz", "fr")
	// Should return key or empty, not panic
	if result == "" {
		// Some implementations return the key itself
		result = T("nonexistent_key_xyz", "en")
	}
	_ = result // no panic is enough
}

// ---------------------------------------------------------------------------
// embeds.go — fonctions pures
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 10, 5, 30, 0, time.UTC)
	result := formatDuration(start, end)
	if result == "" {
		t.Error("expected non-empty duration string")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("short string should not be truncated")
	}
	result := truncate("hello world this is long", 5)
	if len(result) > 8 { // 5 + "..."
		t.Errorf("expected truncated string, got %q", result)
	}
}

func TestHasMissingData_Empty(t *testing.T) {
	if hasMissingData(nil) {
		t.Error("expected false for nil players")
	}
}

func TestAllIdle_Empty(t *testing.T) {
	if !allIdle(nil) {
		t.Error("expected true for nil players")
	}
}

func TestResolveOpLabel(t *testing.T) {
	label := resolveOpLabel("delta", "fr")
	if label == "" {
		t.Error("expected non-empty label for delta op")
	}
}

// ---------------------------------------------------------------------------
// version.go — parsing
// ---------------------------------------------------------------------------

func TestIsMajorMinorChange(t *testing.T) {
	tests := []struct {
		old, new string
		want     bool
	}{
		{"1.0.0", "1.0.1", false},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"", "1.0.0", false},
	}
	for _, tt := range tests {
		got := isMajorMinorChange(tt.old, tt.new)
		if got != tt.want {
			t.Errorf("isMajorMinorChange(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.want)
		}
	}
}

func TestParseMajorMinor(t *testing.T) {
	major, minor, ok := parseMajorMinor("2.5.3")
	if !ok || major != 2 || minor != 5 {
		t.Errorf("expected (2, 5, true), got (%d, %d, %v)", major, minor, ok)
	}

	_, _, ok = parseMajorMinor("invalid")
	if ok {
		t.Error("expected ok=false for invalid version")
	}
}

func TestBuildVersionEmbed(t *testing.T) {
	embed := BuildVersionEmbed("1.2.0", "New features", "fr")
	if embed.Title == "" {
		t.Error("expected non-empty embed title")
	}
}
