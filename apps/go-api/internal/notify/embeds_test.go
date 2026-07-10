package notify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const helloLiteral = "hello"

// ─── embeds.go ───────────────────────────────────────────────────────────────

func TestFormatDuration_Seconds(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(45 * time.Second)
	got := formatDuration(start, end)
	if got != "45s" {
		t.Errorf("formatDuration = %q, want 45s", got)
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(3*time.Minute + 12*time.Second)
	got := formatDuration(start, end)
	if got != "3m12s" {
		t.Errorf("formatDuration = %q, want 3m12s", got)
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(2*time.Hour + 5*time.Minute + 30*time.Second)
	got := formatDuration(start, end)
	if got != "2h05m30s" {
		t.Errorf("formatDuration = %q, want 2h05m30s", got)
	}
}

func TestFormatDuration_Negative(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(-5 * time.Second)
	got := formatDuration(start, end)
	if got != "0s" {
		t.Errorf("formatDuration negative = %q, want 0s", got)
	}
}

func TestTruncate_Short(t *testing.T) {
	got := truncate(helloLiteral, 10)
	if got != helloLiteral {
		t.Errorf("truncate short = %q", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	got := truncate("hello world", 5)
	if got != helloLiteral {
		t.Errorf("truncate long = %q, want 'hello'", got)
	}
}

func TestHasMissingData_Empty(t *testing.T) {
	if hasMissingData(nil) {
		t.Error("empty players should not have missing data")
	}
}

func TestHasMissingData_WithMissing(t *testing.T) {
	players := []PlayerSyncResult{
		{Gamertag: "Player1", MissingData: 5},
	}
	if !hasMissingData(players) {
		t.Error("should detect missing data")
	}
}

func TestHasMissingData_WithError(t *testing.T) {
	players := []PlayerSyncResult{
		{Gamertag: "Player1", Error: "boom"},
	}
	if !hasMissingData(players) {
		t.Error("should detect error as missing data")
	}
}

func TestAllIdle_Empty(t *testing.T) {
	if !allIdle(nil) {
		t.Error("empty players should all be idle")
	}
}

func TestAllIdle_NoMatches(t *testing.T) {
	players := []PlayerSyncResult{
		{Gamertag: "Player1", MatchesSynced: 0},
	}
	if !allIdle(players) {
		t.Error("0 matches should be idle")
	}
}

func TestAllIdle_WithMatches(t *testing.T) {
	players := []PlayerSyncResult{
		{Gamertag: "Player1", MatchesSynced: 5},
	}
	if allIdle(players) {
		t.Error("5 matches should not be idle")
	}
}

func TestResolveOpLabel(t *testing.T) {
	tests := []struct {
		op   string
		want string
	}{
		{"sync_delta", T("discord_op_sync_delta", "en")},
		{"delta", T("discord_op_sync_delta", "en")},
		{"sync_full", T("discord_op_sync_full", "en")},
		{"full", T("discord_op_sync_full", "en")},
		{"backfill_medals", T("discord_op_backfill", "en")},
		{"custom_op", "custom_op"},
	}
	for _, tt := range tests {
		got := resolveOpLabel(tt.op, "en")
		if got != tt.want {
			t.Errorf("resolveOpLabel(%q) = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestBackfillLines_Empty(t *testing.T) {
	lines := backfillLines(BackfillCounts{}, "en")
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty backfill, got %d", len(lines))
	}
}

func TestBackfillLines_WithCounts(t *testing.T) {
	bc := BackfillCounts{
		MedalsInserted: 10,
		EventsInserted: 5,
	}
	lines := backfillLines(bc, "en")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestBuildSyncEmbed_Basic(t *testing.T) {
	start := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Minute)
	players := []PlayerSyncResult{
		{Gamertag: "TestPlayer", MatchesSynced: 5},
	}
	embed := BuildSyncEmbed("sync_delta", start, end, players, true, "en")
	if embed.Title == "" {
		t.Error("expected non-empty title")
	}
	if embed.Color != colorSuccess {
		t.Errorf("expected green color for success, got %d", embed.Color)
	}
	if len(embed.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(embed.Fields))
	}
}

func TestBuildSyncEmbed_Failure(t *testing.T) {
	start := time.Now()
	end := start.Add(30 * time.Second)
	players := []PlayerSyncResult{
		{Gamertag: "P1", Error: "timeout"},
	}
	embed := BuildSyncEmbed("sync_full", start, end, players, false, "fr")
	if embed.Color != colorError {
		t.Errorf("expected error color, got %d", embed.Color)
	}
}

func TestBuildSyncEmbed_Warning(t *testing.T) {
	start := time.Now()
	end := start.Add(1 * time.Minute)
	players := []PlayerSyncResult{
		{Gamertag: "P1", MatchesSynced: 3, MissingData: 2},
	}
	embed := BuildSyncEmbed("sync_delta", start, end, players, true, "en")
	if embed.Color != colorWarning {
		t.Errorf("expected warning color for missing data, got %d", embed.Color)
	}
}

func TestLastMatchLines(t *testing.T) {
	lm := &LastMatchInfo{
		MapName:      "Aquarius",
		PlaylistName: "Ranked",
		VariantName:  "Slayer",
		IsRanked:     true,
		StartTime:    time.Date(2024, 6, 1, 14, 30, 0, 0, time.UTC),
		Kills:        15,
		Deaths:       8,
		Assists:      4,
		Outcome:      2,
	}
	lines := lastMatchLines(lm, "en", HaloLabels())
	if len(lines) == 0 {
		t.Error("expected at least one line")
	}
}

func TestLastMatchLines_WithSquad(t *testing.T) {
	lm := &LastMatchInfo{
		MapName:      "Streets",
		PlaylistName: "Quick Play",
		VariantName:  "CTF",
		Outcome:      3,
		Kills:        5,
		Deaths:       10,
		Assists:      2,
		SquadFriends: []string{"Friend1", "Friend2"},
	}
	lines := lastMatchLines(lm, "en", HaloLabels())
	if len(lines) < 2 {
		t.Errorf("expected squad lines, got %d", len(lines))
	}
}

func TestBuildPlayerField(t *testing.T) {
	p := PlayerSyncResult{
		Gamertag:      "TestPlayer",
		MatchesSynced: 3,
		LastMatch: &LastMatchInfo{
			MapName: "Bazaar",
			Outcome: 2,
			Kills:   10,
			Deaths:  5,
			Assists: 3,
		},
	}
	name, value := buildPlayerField(p, "sync_delta", "en", HaloLabels())
	if name == "" || value == "" {
		t.Error("expected non-empty field")
	}
}

// ─── version.go ──────────────────────────────────────────────────────────────

func TestIsMajorMinorChange(t *testing.T) {
	tests := []struct {
		old, new string
		want     bool
	}{
		{"1.2.3", "1.2.4", false},   // patch only
		{"1.2.3", "1.3.0", true},    // minor bump
		{"1.2.3", "2.0.0", true},    // major bump
		{"", "1.0.0", false},        // first start
		{"v1.2.3", "v1.2.4", false}, // v prefix
		{"v1.2.3", "v1.3.0", true},  // v prefix + minor
		{"invalid", "1.0.0", true},  // fallback string compare
	}
	for _, tt := range tests {
		got := isMajorMinorChange(tt.old, tt.new)
		if got != tt.want {
			t.Errorf("isMajorMinorChange(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.want)
		}
	}
}

func TestParseMajorMinor(t *testing.T) {
	tests := []struct {
		v   string
		maj int
		min int
		ok  bool
	}{
		{"1.2.3", 1, 2, true},
		{"v2.5.0", 2, 5, true},
		{"invalid", 0, 0, false},
		{"1", 0, 0, false},
		{"1.abc", 0, 0, false},
	}
	for _, tt := range tests {
		maj, min, ok := parseMajorMinor(tt.v)
		if ok != tt.ok || maj != tt.maj || min != tt.min {
			t.Errorf("parseMajorMinor(%q) = (%d,%d,%v), want (%d,%d,%v)", tt.v, maj, min, ok, tt.maj, tt.min, tt.ok)
		}
	}
}

func TestBuildVersionEmbed(t *testing.T) {
	embed := BuildVersionEmbed("1.5.0", "New features!", "en")
	if embed.Title == "" {
		t.Error("expected non-empty title")
	}
	if embed.Color != colorVersion {
		t.Errorf("expected version color, got %d", embed.Color)
	}
	if embed.Description != "New features!" {
		t.Errorf("description = %q, want 'New features!'", embed.Description)
	}
}

func TestBuildVersionEmbed_LongChangelog(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "changelog "
	}
	embed := BuildVersionEmbed("2.0.0", long, "en")
	if len([]rune(embed.Description)) > maxDiscordBody+10 {
		t.Error("description should be truncated")
	}
}

func TestReadWriteLastNotifiedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// Write initial
	if err := os.WriteFile(path, []byte(`{"lang":"fr"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Read — no last_notified_version
	v := readLastNotifiedVersion(path)
	if v != "" {
		t.Errorf("expected empty, got %q", v)
	}

	// Write version
	if err := writeLastNotifiedVersion(path, "1.5.0"); err != nil {
		t.Fatal(err)
	}

	// Read back
	v = readLastNotifiedVersion(path)
	if v != "1.5.0" {
		t.Errorf("expected '1.5.0', got %q", v)
	}

	// Verify other keys preserved
	raw, _ := os.ReadFile(path)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["lang"] != "fr" {
		t.Error("other keys should be preserved")
	}
}

func TestReadLastNotifiedVersion_MissingFile(t *testing.T) {
	v := readLastNotifiedVersion("/nonexistent/settings.json")
	if v != "" {
		t.Errorf("expected empty for missing file, got %q", v)
	}
}

// ─── discord.go ──────────────────────────────────────────────────────────────

func TestT_KnownKey(t *testing.T) {
	// T should return a string for known keys
	result := T("discord_last_match", "en")
	if result == "" {
		t.Error("expected non-empty string for known key")
	}
}

// TestDiscordFooterText garantit que le footer reste byte-identique au libellé
// historique « LevelUp · Halo Infinite Stats » pour les libellés par défaut (nil
// → Halo), et qu'il SUIT le nom du titre quand des labels title-aware sont fournis.
func TestDiscordFooterText(t *testing.T) {
	if got := discordFooterText(nil); got != "LevelUp · Halo Infinite Stats" {
		t.Errorf("discordFooterText(nil) = %q, want \"LevelUp · Halo Infinite Stats\"", got)
	}
	titled := LabelsFor(fakeOutcomeSrc{set: nil}, "Game Two")
	if got := discordFooterText(titled); got != "LevelUp · Game Two Stats" {
		t.Errorf("discordFooterText(titled) = %q, want \"LevelUp · Game Two Stats\"", got)
	}
}

func TestT_UnknownKey(t *testing.T) {
	result := T("totally_unknown_key_xyz", "en")
	// Should return the key itself as fallback
	if result != "totally_unknown_key_xyz" {
		t.Errorf("expected key as fallback, got %q", result)
	}
}

func TestBoolVal(t *testing.T) {
	m := map[string]any{"flag": true, "str": "not a bool"}
	if !boolVal(m, "flag") {
		t.Error("expected true")
	}
	if boolVal(m, "str") {
		t.Error("expected false for non-bool")
	}
	if boolVal(m, "missing") {
		t.Error("expected false for missing key")
	}
}

func TestBoolValDefault(t *testing.T) {
	m := map[string]any{"flag": false}
	if boolValDefault(m, "flag", true) {
		t.Error("expected false from map value")
	}
	if !boolValDefault(m, "missing", true) {
		t.Error("expected true as default")
	}
}

func TestStrVal(t *testing.T) {
	m := map[string]any{"name": helloLiteral}
	if strVal(m, "name") != helloLiteral {
		t.Error("expected hello")
	}
	if strVal(m, "missing") != "" {
		t.Error("expected empty for missing")
	}
}

func TestStrValDefault(t *testing.T) {
	m := map[string]any{"name": helloLiteral}
	if strValDefault(m, "name", "default") != helloLiteral {
		t.Error("expected hello")
	}
	if strValDefault(m, "missing", "fallback") != "fallback" {
		t.Error("expected fallback")
	}
}
