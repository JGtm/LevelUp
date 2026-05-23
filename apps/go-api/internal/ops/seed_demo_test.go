// Package ops — seed_demo_test.go : tests unitaires (sans CGO/DuckDB).
//
// Couvre les helpers stateless : formatIDsLiteral, validateSeedDemoOpts,
// ResolveSourceXUIDFromProfiles, copyMetadataFile, writeDemoConfigs,
// classifyMediaKind, parseRegistryTime, derefStr, buildDemoMediaRoot,
// loadMediaRegistry, saveMediaRegistry.
//
// Tests d'intégration avec DuckDB live dans seed_demo_integration_test.go
// (build tag integration).
package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── formatIDsLiteral ─────────────────────────────────────────────────────────

func TestFormatIDsLiteral(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"empty", nil, "''"},
		{"single", []string{"m1"}, "'m1'"},
		{"multi", []string{"m1", "m2", "m3"}, "'m1', 'm2', 'm3'"},
		{
			"reject_sqli_apostrophe",
			[]string{"m1", "m2'; DROP TABLE--", "m3"},
			"'m1', 'm3'",
		},
		{
			"reject_sqli_semicolon",
			[]string{"m1;", "m2"},
			"'m2'",
		},
		{"all_rejected_returns_empty_marker", []string{"'", ";"}, "''"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatIDsLiteral(tc.in)
			if got != tc.want {
				t.Errorf("formatIDsLiteral(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ── validateSeedDemoOpts ─────────────────────────────────────────────────────

func TestValidateSeedDemoOpts_AppliesDefaults(t *testing.T) {
	// Crée 3 fichiers source factices pour passer os.Stat.
	dir := t.TempDir()
	srcPlayer := filepath.Join(dir, "src_player.duckdb")
	srcShared := filepath.Join(dir, "src_shared.duckdb")
	srcMeta := filepath.Join(dir, "src_meta.duckdb")
	for _, p := range []string{srcPlayer, srcShared, srcMeta} {
		if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	opts := &SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     "1234567890",
		OutDir:         filepath.Join(dir, "out"),
		// MaxMatches / DemoXUID / DemoGamertag / MaxMedia laissés vides
	}
	if err := validateSeedDemoOpts(opts); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if opts.MaxMatches != DefaultMaxMatches {
		t.Errorf("MaxMatches default = %d, want %d", opts.MaxMatches, DefaultMaxMatches)
	}
	if opts.DemoXUID != DefaultDemoXUID {
		t.Errorf("DemoXUID default = %q, want %q", opts.DemoXUID, DefaultDemoXUID)
	}
	if opts.DemoGamertag != DefaultDemoGamertag {
		t.Errorf("DemoGamertag default = %q, want %q", opts.DemoGamertag, DefaultDemoGamertag)
	}
	if opts.MaxMedia != DefaultMaxMedia {
		t.Errorf("MaxMedia default = %d, want %d", opts.MaxMedia, DefaultMaxMedia)
	}
}

func TestValidateSeedDemoOpts_RequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		opts    SeedDemoOptions
		wantErr string
	}{
		{"missing_source_player", SeedDemoOptions{SourceSharedDB: "x", SourceMetaDB: "x", SourceXUID: "1", OutDir: "o"}, "source paths required"},
		{"missing_xuid", SeedDemoOptions{SourcePlayerDB: "x", SourceSharedDB: "x", SourceMetaDB: "x", OutDir: "o"}, "SourceXUID required"},
		{"missing_out", SeedDemoOptions{SourcePlayerDB: "x", SourceSharedDB: "x", SourceMetaDB: "x", SourceXUID: "1"}, "OutDir required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSeedDemoOpts(&tc.opts)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateSeedDemoOpts_SourceFileMissing(t *testing.T) {
	opts := &SeedDemoOptions{
		SourcePlayerDB: "/nonexistent/x.duckdb",
		SourceSharedDB: "/nonexistent/y.duckdb",
		SourceMetaDB:   "/nonexistent/z.duckdb",
		SourceXUID:     "1",
		OutDir:         "/tmp/out",
	}
	err := validateSeedDemoOpts(opts)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	if !strings.Contains(err.Error(), "introuvable") {
		t.Errorf("err = %v, want containing 'introuvable'", err)
	}
}

// ── ResolveSourceXUIDFromProfiles ────────────────────────────────────────────

func TestResolveSourceXUIDFromProfiles_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	content := `{
		"version": "2.1",
		"profiles": {
			"JGtm": {"xuid": "1234567890", "db_path": "data/players/JGtm/stats.duckdb"},
			"Other": {"xuid": "9876543210", "db_path": "data/players/Other/stats.duckdb"}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	xuid, dbPath, err := ResolveSourceXUIDFromProfiles(path, "JGtm")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if xuid != "1234567890" {
		t.Errorf("xuid = %q, want 1234567890", xuid)
	}
	if dbPath != "data/players/JGtm/stats.duckdb" {
		t.Errorf("dbPath = %q, want data/players/JGtm/stats.duckdb", dbPath)
	}
}

func TestResolveSourceXUIDFromProfiles_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	_ = os.WriteFile(path, []byte(`{"profiles": {"X": {"xuid": "1"}}}`), 0o644)
	_, _, err := ResolveSourceXUIDFromProfiles(path, "Missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "introuvable") {
		t.Errorf("err = %v, want 'introuvable'", err)
	}
}

func TestResolveSourceXUIDFromProfiles_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	_ = os.WriteFile(path, []byte(`{not valid json`), 0o644)
	_, _, err := ResolveSourceXUIDFromProfiles(path, "X")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestResolveSourceXUIDFromProfiles_FileMissing(t *testing.T) {
	_, _, err := ResolveSourceXUIDFromProfiles("/nonexistent.json", "X")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveSourceXUIDFromProfiles_V3MultiTitres(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	content := `{
		"version": "3.0",
		"profiles": {
			"halo_infinite": {
				"JGtm": {"xuid": "2533274823110022", "db_path": "data/titles/halo_infinite/players/JGtm/stats.duckdb"},
				"Other": {"xuid": "1111111111111111", "db_path": "data/titles/halo_infinite/players/Other/stats.duckdb"}
			},
			"halo_reach": {
				"LegacyPlayer": {"xuid": "9999999999999999", "db_path": "data/titles/halo_reach/players/LegacyPlayer/stats.duckdb"}
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Gamertag dans halo_infinite (titre par défaut) — résolution directe.
	xuid, dbPath, err := ResolveSourceXUIDFromProfiles(path, "JGtm")
	if err != nil {
		t.Fatalf("JGtm: %v", err)
	}
	if xuid != "2533274823110022" {
		t.Errorf("JGtm.xuid = %q", xuid)
	}
	if dbPath != "data/titles/halo_infinite/players/JGtm/stats.duckdb" {
		t.Errorf("JGtm.dbPath = %q", dbPath)
	}
	// Gamertag dans un autre titre — fallback cross-title.
	xuid2, _, err := ResolveSourceXUIDFromProfiles(path, "LegacyPlayer")
	if err != nil {
		t.Fatalf("LegacyPlayer: %v", err)
	}
	if xuid2 != "9999999999999999" {
		t.Errorf("LegacyPlayer.xuid = %q", xuid2)
	}
	// Gamertag inconnu.
	_, _, err = ResolveSourceXUIDFromProfiles(path, "Missing")
	if err == nil || !strings.Contains(err.Error(), "introuvable") {
		t.Errorf("Missing: expected 'introuvable', got %v", err)
	}
}

func TestResolveSourceXUIDFromProfiles_EmptyXUID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	_ = os.WriteFile(path, []byte(`{"profiles": {"X": {"db_path": "x"}}}`), 0o644)
	_, _, err := ResolveSourceXUIDFromProfiles(path, "X")
	if err == nil || !strings.Contains(err.Error(), "sans xuid") {
		t.Errorf("expected 'sans xuid' error, got %v", err)
	}
}

// ── copyMetadataFile ─────────────────────────────────────────────────────────

func TestCopyMetadataFile_BasicCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.duckdb")
	dst := filepath.Join(dir, "out", "warehouse", "metadata.duckdb")
	content := []byte("FAKE_DUCKDB_BINARY_DATA")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyMetadataFile(src, dst); err != nil {
		t.Fatalf("copy: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestCopyMetadataFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.duckdb")
	dst := filepath.Join(dir, "dst.duckdb")
	_ = os.WriteFile(src, []byte("NEW"), 0o644)
	_ = os.WriteFile(dst, []byte("OLD_VERSION"), 0o644)
	if err := copyMetadataFile(src, dst); err != nil {
		t.Fatalf("copy: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "NEW" {
		t.Errorf("got %q, want NEW", got)
	}
}

func TestCopyMetadataFile_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	err := copyMetadataFile(filepath.Join(dir, "missing.duckdb"), filepath.Join(dir, "out.duckdb"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

// ── writeDemoConfigs ─────────────────────────────────────────────────────────

func TestWriteDemoConfigs_StructureOK(t *testing.T) {
	dir := t.TempDir()
	err := writeDemoConfigs(dir, "0000000000000000", "JGtm", "SPTA", true)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// db_profiles.json
	profiles := readJSON(t, filepath.Join(dir, "db_profiles.json"))
	if v, _ := profiles["version"].(string); v != "2.1" {
		t.Errorf("version = %q, want 2.1", v)
	}
	if v, _ := profiles["warehouse_path"].(string); v != "data/warehouse" {
		t.Errorf("warehouse_path = %q, want data/warehouse", v)
	}
	demoProfile, ok := profiles["profiles"].(map[string]any)[DefaultDemoGamertag].(map[string]any)
	if !ok {
		t.Fatalf("profile DEMO absent")
	}
	if v, _ := demoProfile["xuid"].(string); v != "0000000000000000" {
		t.Errorf("DEMO.xuid = %q", v)
	}
	if v, _ := demoProfile["waypoint_player"].(string); v != "JGtm" {
		t.Errorf("DEMO.waypoint_player = %q, want JGtm", v)
	}

	// app_settings.json
	settings := readJSON(t, filepath.Join(dir, "app_settings.json"))
	if v, _ := settings["profile_service_tag"].(string); v != "SPTA" {
		t.Errorf("service_tag = %q", v)
	}
	if v, _ := settings["media_enabled"].(bool); !v {
		t.Errorf("media_enabled = false, want true")
	}
	// spnkr_refresh_on_start doit être false en démo.
	if v, _ := settings["spnkr_refresh_on_start"].(bool); v {
		t.Errorf("spnkr_refresh_on_start = true, want false")
	}
}

func TestWriteDemoConfigs_MediaDisabledFlag(t *testing.T) {
	dir := t.TempDir()
	if err := writeDemoConfigs(dir, "0000", "JGtm", "", false); err != nil {
		t.Fatal(err)
	}
	settings := readJSON(t, filepath.Join(dir, "app_settings.json"))
	if v, _ := settings["media_enabled"].(bool); v {
		t.Errorf("media_enabled = true, want false")
	}
	if v, _ := settings["profile_service_tag"].(string); v != "" {
		t.Errorf("service_tag = %q, want empty", v)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

// ── classifyMediaKind / parseRegistryTime / derefStr / buildDemoMediaRoot ────

func TestClassifyMediaKind(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{".mp4", "video"},
		{".mov", "video"},
		{".avi", "video"},
		{".webm", "video"},
		{".mkv", "video"},
		{".png", "image"},
		{".jpg", "image"},
		{".gif", "image"},
		{"", "image"},
	}
	for _, tc := range cases {
		if got := classifyMediaKind(tc.in); got != tc.want {
			t.Errorf("classifyMediaKind(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseRegistryTime(t *testing.T) {
	// nil → zero
	if !parseRegistryTime(nil).IsZero() {
		t.Error("nil should return zero time")
	}
	// empty → zero
	empty := ""
	if !parseRegistryTime(&empty).IsZero() {
		t.Error("empty should return zero time")
	}
	// invalid → zero
	bad := "not-a-date"
	if !parseRegistryTime(&bad).IsZero() {
		t.Error("invalid should return zero time")
	}
	// valid RFC3339
	v := "2026-05-22T10:30:00Z"
	got := parseRegistryTime(&v)
	if got.IsZero() {
		t.Errorf("valid RFC3339 returned zero: %v", got)
	}
	if got.Year() != 2026 || got.Month() != time.May || got.Day() != 22 {
		t.Errorf("parsed time = %v, want 2026-05-22", got)
	}
}

func TestDerefStr(t *testing.T) {
	if derefStr(nil) != nil {
		t.Error("nil should return nil")
	}
	v := "hello"
	got := derefStr(&v)
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf("got %v, want 'hello'", got)
	}
}

func TestBuildDemoMediaRoot(t *testing.T) {
	// Sans LEVELUP_ROOT → fallback sur outMediaDir.
	t.Setenv("LEVELUP_ROOT", "")
	got := buildDemoMediaRoot("/tmp/demo/media")
	if got != "/tmp/demo/media" {
		t.Errorf("without LEVELUP_ROOT: got %q", got)
	}
	// Avec LEVELUP_ROOT → chemin canonique.
	t.Setenv("LEVELUP_ROOT", "/app")
	got = buildDemoMediaRoot("/tmp/demo/media")
	if !strings.HasSuffix(got, "data/players/"+DefaultDemoGamertag+"/media") {
		t.Errorf("with LEVELUP_ROOT: got %q, want suffix data/players/DEMO/media", got)
	}
}

// ── media registry roundtrip ────────────────────────────────────────────────

func TestMediaRegistry_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "media_registry.json")

	mapName := "Aquarius"
	startTime := "2026-05-22T10:30:00Z"
	entries := []mediaRegistryEntry{
		{Filename: "clip1.mp4", FileHash: "abc123", MatchID: "m1", MatchStartTime: &startTime, MapName: &mapName},
		{Filename: "screenshot.png", FileHash: "def456", MatchID: "m2"},
	}
	if err := saveMediaRegistry(path, entries); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadMediaRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].Filename != "clip1.mp4" || got[0].FileHash != "abc123" {
		t.Errorf("entry[0] = %+v", got[0])
	}
	if got[0].MatchStartTime == nil || *got[0].MatchStartTime != startTime {
		t.Errorf("entry[0].MatchStartTime = %v", got[0].MatchStartTime)
	}
	if got[1].MapName != nil {
		t.Errorf("entry[1].MapName should be nil, got %v", got[1].MapName)
	}
}

func TestLoadMediaRegistry_FileMissing(t *testing.T) {
	_, err := loadMediaRegistry("/nonexistent/registry.json")
	if err == nil {
		t.Fatal("expected error")
	}
}
