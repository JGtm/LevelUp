package notify

import (
	"testing"
)

// ── T (i18n lookup) ──────────────────────────────────────────────────────────

func TestT_KnownKeyFR(t *testing.T) {
	got := T("discord_version_footer", "fr")
	if got == "discord_version_footer" {
		t.Fatal("expected translated text, got key itself")
	}
	if got == "" {
		t.Fatal("expected non-empty translation")
	}
}

func TestT_KnownKeyEN(t *testing.T) {
	got := T("discord_version_footer", "en")
	if got == "discord_version_footer" {
		t.Fatal("expected translated text, got key itself")
	}
}

func TestT_UnknownKeyReturnsKey(t *testing.T) {
	got := T("totally_missing_key", "fr")
	if got != "totally_missing_key" {
		t.Fatalf("expected key itself, got %q", got)
	}
}

func TestT_WithArgs(t *testing.T) {
	got := T("discord_version_title", "fr", "version", "1.2.3")
	if got == "" || got == "discord_version_title" {
		t.Fatalf("expected formatted string, got %q", got)
	}
}

func TestT_EmptyLangDefaultsFR(t *testing.T) {
	got := T("discord_version_footer", "")
	if got == "discord_version_footer" {
		t.Fatal("expected translation with empty lang (should default to fr)")
	}
}

func TestT_OddArgsIgnored(t *testing.T) {
	// Odd number of args: should not substitute
	got := T("discord_version_title", "fr", "version")
	if got == "" {
		t.Fatal("expected non-empty")
	}
}

// ── boolVal / boolValDefault ─────────────────────────────────────────────────

func TestBoolVal_True(t *testing.T) {
	m := map[string]any{"a": true}
	if !boolVal(m, "a") {
		t.Fatal("expected true")
	}
}

func TestBoolVal_False(t *testing.T) {
	m := map[string]any{"a": false}
	if boolVal(m, "a") {
		t.Fatal("expected false")
	}
}

func TestBoolVal_Missing(t *testing.T) {
	m := map[string]any{}
	if boolVal(m, "x") {
		t.Fatal("expected false for missing")
	}
}

func TestBoolVal_WrongType(t *testing.T) {
	m := map[string]any{"a": "yes"}
	if boolVal(m, "a") {
		t.Fatal("expected false for non-bool type")
	}
}

func TestBoolValDefault_MissingUsesDefault(t *testing.T) {
	m := map[string]any{}
	if !boolValDefault(m, "x", true) {
		t.Fatal("expected true default")
	}
	if boolValDefault(m, "x", false) {
		t.Fatal("expected false default")
	}
}

func TestBoolValDefault_WrongTypeUsesDefault(t *testing.T) {
	m := map[string]any{"a": 123}
	if !boolValDefault(m, "a", true) {
		t.Fatal("expected true default for non-bool")
	}
}

func TestBoolValDefault_PresentsOverridesDefault(t *testing.T) {
	m := map[string]any{"a": false}
	if boolValDefault(m, "a", true) {
		t.Fatal("expected false (actual value) not default true")
	}
}

// ── strVal / strValDefault ───────────────────────────────────────────────────

func TestStrVal_Found(t *testing.T) {
	m := map[string]any{"k": "v"}
	if strVal(m, "k") != "v" {
		t.Fatal("expected v")
	}
}

func TestStrVal_Missing(t *testing.T) {
	m := map[string]any{}
	if strVal(m, "k") != "" {
		t.Fatal("expected empty")
	}
}

func TestStrVal_WrongType(t *testing.T) {
	m := map[string]any{"k": 42}
	if strVal(m, "k") != "" {
		t.Fatal("expected empty for non-string")
	}
}

func TestStrValDefault_Empty(t *testing.T) {
	m := map[string]any{}
	if strValDefault(m, "k", "def") != "def" {
		t.Fatal("expected default")
	}
}

func TestStrValDefault_Present(t *testing.T) {
	m := map[string]any{"k": "val"}
	if strValDefault(m, "k", "def") != "val" {
		t.Fatal("expected val")
	}
}

// ── parseMajorMinor ──────────────────────────────────────────────────────────

func TestParseMajorMinor_Valid(t *testing.T) {
	maj, min, ok := parseMajorMinor("1.2.3")
	if !ok || maj != 1 || min != 2 {
		t.Fatalf("expected (1,2,true), got (%d,%d,%v)", maj, min, ok)
	}
}

func TestParseMajorMinor_WithV(t *testing.T) {
	maj, min, ok := parseMajorMinor("v3.4.5")
	if !ok || maj != 3 || min != 4 {
		t.Fatalf("expected (3,4,true), got (%d,%d,%v)", maj, min, ok)
	}
}

func TestParseMajorMinor_TwoParts(t *testing.T) {
	maj, min, ok := parseMajorMinor("1.2")
	if !ok || maj != 1 || min != 2 {
		t.Fatalf("expected (1,2,true), got (%d,%d,%v)", maj, min, ok)
	}
}

func TestParseMajorMinor_OnePart(t *testing.T) {
	_, _, ok := parseMajorMinor("1")
	if ok {
		t.Fatal("expected false for single part")
	}
}

func TestParseMajorMinor_Invalid(t *testing.T) {
	_, _, ok := parseMajorMinor("abc.def")
	if ok {
		t.Fatal("expected false for non-numeric")
	}
}

// ── isMajorMinorChange ───────────────────────────────────────────────────────

func TestIsMajorMinorChange_SameVersion(t *testing.T) {
	if isMajorMinorChange("1.2.3", "1.2.4") {
		t.Fatal("expected false for patch only")
	}
}

func TestIsMajorMinorChange_MinorBump(t *testing.T) {
	if !isMajorMinorChange("1.2.3", "1.3.0") {
		t.Fatal("expected true for minor bump")
	}
}

func TestIsMajorMinorChange_MajorBump(t *testing.T) {
	if !isMajorMinorChange("1.2.3", "2.0.0") {
		t.Fatal("expected true for major bump")
	}
}

func TestIsMajorMinorChange_EmptyOld(t *testing.T) {
	if isMajorMinorChange("", "1.0.0") {
		t.Fatal("expected false for empty old")
	}
}

func TestIsMajorMinorChange_InvalidFallback(t *testing.T) {
	// Invalid versions fallback to string comparison
	if !isMajorMinorChange("abc", "def") {
		t.Fatal("expected true for different invalid versions")
	}
}

// ── BuildVersionEmbed ────────────────────────────────────────────────────────

func TestBuildVersionEmbed_EN(t *testing.T) {
	e := BuildVersionEmbed("1.2.0", "New features!", "en")
	if e.Title == "" {
		t.Fatal("expected non-empty title")
	}
	if e.Description != "New features!" {
		t.Fatalf("expected description, got %q", e.Description)
	}
	if e.Color != colorVersion {
		t.Fatalf("expected colorVersion, got %d", e.Color)
	}
}

func TestBuildVersionEmbed_EmptyChangelogFR(t *testing.T) {
	e := BuildVersionEmbed("v2.0.0", "", "fr")
	if e.Description == "" {
		t.Fatal("expected fallback description")
	}
}
