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
	"bytes"
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
	assertNoCopyTempLeftover(t, dir)
}

// TestCopyMetadataFile_ReplacesInode : au déploiement, l'ancien conteneur démo tient
// encore la metadata ouverte quand le seed la republie. Le fichier publié doit donc
// être un inode NEUF — une écriture en place tronquerait la DB que ce conteneur
// utilise (et bloquerait l'ATTACH READ_ONLY de la phase Prestige).
func TestCopyMetadataFile_ReplacesInode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.duckdb")
	dst := filepath.Join(dir, "metadata.duckdb")
	content := []byte{0x44, 0x55, 0x43, 0x4b, 0x00, 0x01, 0xff, 0xfe} // binaire : copie octet à octet
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("ANCIENNE_GENERATION"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := statFileIdentity(t, dst)

	if err := copyMetadataFile(src, dst); err != nil {
		t.Fatalf("copy: %v", err)
	}

	after := statFileIdentity(t, dst)
	if os.SameFile(before, after) {
		t.Error("dst pointe sur le même fichier qu'avant : écriture en place — la DB tenue par l'ancien conteneur démo serait tronquée")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("contenu = %v, want %v", got, content)
	}
	assertNoCopyTempLeftover(t, dir)
}

// TestCopyMetadataFile_PreviousInodeUntouched : un lien dur vers l'ancien fichier
// joue le rôle du descripteur que l'ancien conteneur démo garde ouvert — il doit
// continuer à voir SES octets après la publication de la nouvelle metadata.
func TestCopyMetadataFile_PreviousInodeUntouched(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.duckdb")
	dst := filepath.Join(dir, "metadata.duckdb")
	if err := os.WriteFile(src, []byte("NOUVELLE_GENERATION"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("ANCIENNE_GENERATION"), 0o644); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "handle_ancien_conteneur")
	if err := os.Link(dst, alias); err != nil {
		t.Skipf("liens durs indisponibles sur ce système de fichiers: %v", err)
	}

	if err := copyMetadataFile(src, dst); err != nil {
		t.Fatalf("copy: %v", err)
	}

	old, err := os.ReadFile(alias)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "ANCIENNE_GENERATION" {
		t.Errorf("ancien fichier modifié: %q — le lecteur qui le tient ouvert verrait sa DB changer sous lui", old)
	}
}

// TestCopyMetadataFile_DstNotRemovable : si dst ne peut pas être retiré, le seed doit
// échouer franchement. Le repli « écrire dans le fichier existant » ressusciterait la
// troncature d'une DB vivante. (Cas réaliste : Docker crée un répertoire à la place
// d'un fichier de bind-mount absent — cf. workflow test-deploy-precheck.)
func TestCopyMetadataFile_DstNotRemovable(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.duckdb")
	if err := os.WriteFile(src, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "metadata.duckdb")
	if err := os.MkdirAll(filepath.Join(dst, "inner"), 0o755); err != nil { // non vide → Remove échoue
		t.Fatal(err)
	}

	if err := copyMetadataFile(src, dst); err == nil {
		t.Fatal("attendu : erreur quand dst ne peut pas être retiré")
	}

	if _, err := os.Stat(filepath.Join(dst, "inner")); err != nil {
		t.Errorf("dst touché malgré l'échec: %v", err)
	}
	assertNoCopyTempLeftover(t, dir)
}

// statFileIdentity retourne l'identité du fichier (volume+index sous Windows, dev+ino
// ailleurs) LUE DEPUIS UN HANDLE OUVERT, comparable ensuite avec os.SameFile.
//
// os.Stat ne convient pas ici : sous Windows il ne résout l'identifiant du fichier
// qu'au moment de la comparaison, en rouvrant le CHEMIN — deux stats pris avant et
// après un remplacement d'inode désigneraient alors le même fichier (le nouveau) et
// l'assertion serait toujours vraie. File.Stat() capture l'identifiant tout de suite
// (GetFileInformationByHandle / fstat), donc il survit au remplacement.
func statFileIdentity(t *testing.T, path string) os.FileInfo {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// assertNoCopyTempLeftover : aucun temporaire de copie ne survit, ni au succès ni à
// l'échec (le répertoire warehouse démo est monté tel quel dans le conteneur).
func assertNoCopyTempLeftover(t *testing.T, dir string) {
	t.Helper()
	leftover, err := filepath.Glob(filepath.Join(dir, "*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftover) > 0 {
		t.Errorf("temporaires laissés derrière: %v", leftover)
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

// ── médias HLS Halo Infinite (scan + parse + attribution) ────────────────────

func TestParseHaloCaptureTime(t *testing.T) {
	got := parseHaloCaptureTime("Halo Infinite 2025-12-18 21-48-59")
	if !got.Valid {
		t.Fatal("nom valide → devrait parser")
	}
	if got.Time.Year() != 2025 || got.Time.Month() != time.December || got.Time.Day() != 18 {
		t.Errorf("parsed = %v, want 2025-12-18", got.Time)
	}
	if bad := parseHaloCaptureTime("Halo Infinite pasunedate"); bad.Valid {
		t.Error("nom invalide → devrait être invalide")
	}
}

func TestScanHaloInfiniteHLS(t *testing.T) {
	dir := t.TempDir()
	hlsRoot := filepath.Join(dir, "hls")
	thumbsRoot := filepath.Join(dir, "thumbs")
	// 2 captures Halo Infinite (avec master.m3u8) + 1 sans m3u8 + 1 autre préfixe.
	for _, n := range []string{"Halo Infinite 2025-12-18 21-48-59", "Halo Infinite 2026-02-09 22-40-56"} {
		d := filepath.Join(hlsRoot, n)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "master.m3u8"), []byte("#EXTM3U"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.MkdirAll(filepath.Join(hlsRoot, "Halo Infinite incomplet"), 0o755) // pas de master.m3u8
	_ = os.MkdirAll(filepath.Join(hlsRoot, "Replay 2026-01-01 00-00-00"), 0o755)
	_ = os.MkdirAll(thumbsRoot, 0o755)
	_ = os.WriteFile(filepath.Join(thumbsRoot, "Halo Infinite 2026-02-09 22-40-56.webp"), []byte("x"), 0o644)

	got := scanHaloInfiniteHLS(dir, 10)
	if len(got) != 2 {
		t.Fatalf("got %d médias, want 2 (Halo Infinite avec master.m3u8)", len(got))
	}
	// Plus récents d'abord.
	if got[0].Name != "Halo Infinite 2026-02-09 22-40-56" {
		t.Errorf("ordre: got[0] = %q, want le plus récent", got[0].Name)
	}
	if got[0].ThumbPath == "" {
		t.Error("vignette attendue pour la capture 2026-02-09")
	}
}
