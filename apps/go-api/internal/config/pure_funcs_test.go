package config

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ── buildPoolConfig ──────────────────────────────────────────────────────────

func TestBuildPoolConfig_LegacyTitle(t *testing.T) {
	cfg := &AppConfig{RepoRoot: "/tmp/repo"}
	p := &domain.PlayerSummary{
		Gamertag: "TestPlayer",
		XUID:     "12345",
	}
	pcfg := buildPoolConfig(cfg, p, "unknown_title")
	if pcfg.Gamertag != "TestPlayer" {
		t.Fatalf("expected TestPlayer, got %s", pcfg.Gamertag)
	}
	if pcfg.XUID != "12345" {
		t.Fatalf("expected 12345, got %s", pcfg.XUID)
	}
	if pcfg.TitleSlug != "unknown_title" {
		t.Fatalf("expected unknown_title, got %s", pcfg.TitleSlug)
	}
	if pcfg.PlayerDBPath == "" {
		t.Fatal("expected non-empty PlayerDBPath")
	}
}

func TestBuildPoolConfig_HaloInfinite(t *testing.T) {
	cfg := &AppConfig{RepoRoot: "/tmp/repo"}
	p := &domain.PlayerSummary{
		Gamertag: "Player2",
		XUID:     "999",
	}
	pcfg := buildPoolConfig(cfg, p, "halo_infinite")
	if pcfg.Gamertag != "Player2" {
		t.Fatalf("expected Player2, got %s", pcfg.Gamertag)
	}
	if pcfg.TitleSlug != "halo_infinite" {
		t.Fatalf("expected halo_infinite, got %s", pcfg.TitleSlug)
	}
	if pcfg.SharedDBPath == "" {
		t.Fatal("expected non-empty SharedDBPath")
	}
}

// ── ResolveMediaDeleteSource ─────────────────────────────────────────────────

// TestResolveMediaDeleteSource couvre la matrice env{unset,"1","0"} ×
// store{nil,true,false} × isProd{true,false} : env prime, sinon store, sinon isProd.
func TestResolveMediaDeleteSource(t *testing.T) {
	bptr := func(b bool) *bool { return &b }
	cases := []struct {
		name   string
		envRaw string
		storeV *bool
		isProd bool
		want   bool
	}{
		// env défini → gagne toujours, quels que soient store/isProd.
		{"env1_overrides_store_false_prodfalse", "1", bptr(false), false, true},
		{"env0_overrides_store_true_prodtrue", "0", bptr(true), true, false},
		{"env_true_word", "true", nil, false, true},
		{"env_off_word", "off", bptr(true), true, false},
		{"env_whitespace_is_unset", "   ", bptr(true), false, true}, // vide-après-trim → store
		// env non défini → store gagne si non-nil.
		{"unset_store_true", "", bptr(true), false, true},
		{"unset_store_false_prodtrue", "", bptr(false), true, false},
		// env non défini + store nil → défaut isProd.
		{"unset_nil_prodtrue", "", nil, true, true},
		{"unset_nil_prodfalse", "", nil, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveMediaDeleteSource(tc.envRaw, tc.storeV, tc.isProd)
			if got != tc.want {
				t.Fatalf("ResolveMediaDeleteSource(%q, %v, %v) = %v, want %v",
					tc.envRaw, tc.storeV, tc.isProd, got, tc.want)
			}
		})
	}
}

// ── loadPlayersV2 ────────────────────────────────────────────────────────────

func TestLoadPlayersV2_ValidJSON(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{
		"version": "2.1",
		"profiles": {
			"Player1": {"xuid": "111", "db_path": "/a"},
			"Player2": {"xuid": "222", "waypoint_player": "WP2"}
		}
	}`)
	players, err := cfg.loadPlayersV2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
}

func TestLoadPlayersV2_EmptyProfiles(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{"version": "2.1", "profiles": {}}`)
	players, err := cfg.loadPlayersV2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("expected 0, got %d", len(players))
	}
}

func TestLoadPlayersV2_InvalidJSON(t *testing.T) {
	cfg := &AppConfig{}
	_, err := cfg.loadPlayersV2([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadPlayersV2_TitleFilterHaloInfinite(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{"version": "2.1", "profiles": {"P1": {"xuid": "1"}}}`)
	players, err := cfg.loadPlayersV2(data, "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 with halo_infinite filter, got %d", len(players))
	}
}

func TestLoadPlayersV2_TitleFilterOther(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{"version": "2.1", "profiles": {"P1": {"xuid": "1"}}}`)
	players, err := cfg.loadPlayersV2(data, "other_title")
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 0 {
		t.Fatalf("expected 0 with other title filter, got %d", len(players))
	}
}

func TestLoadPlayersV2_WaypointPlayerFallback(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{"version": "2.1", "profiles": {"GT1": {"xuid": "1"}}}`)
	players, err := cfg.loadPlayersV2(data)
	if err != nil {
		t.Fatal(err)
	}
	if players[0].WaypointPlayer != "GT1" {
		t.Fatalf("expected WaypointPlayer=GT1, got %s", players[0].WaypointPlayer)
	}
}

// ── loadPlayersV3 ────────────────────────────────────────────────────────────

func TestLoadPlayersV3_ValidJSON(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{
		"version": "3.0",
		"profiles": {
			"halo_infinite": {
				"Player1": {"xuid": "111"},
				"Player2": {"xuid": "222"}
			},
			"other_game": {
				"Player3": {"xuid": "333"}
			}
		}
	}`)
	players, err := cfg.loadPlayersV3(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 3 {
		t.Fatalf("expected 3, got %d", len(players))
	}
}

func TestLoadPlayersV3_TitleFilter(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{
		"version": "3.0",
		"profiles": {
			"halo_infinite": {"P1": {"xuid": "1"}},
			"other": {"P2": {"xuid": "2"}}
		}
	}`)
	players, err := cfg.loadPlayersV3(data, "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1, got %d", len(players))
	}
}

func TestLoadPlayersV3_AuthOnlyFlag(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{
		"version": "3.0",
		"profiles": {
			"halo_infinite": {
				"RealPlayer": {"xuid": "111", "db_path": "data/x/stats.duckdb"},
				"TokenOnly": {"xuid": "222", "auth_only": true}
			}
		}
	}`)
	players, err := cfg.loadPlayersV3(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := map[string]bool{}
	for _, p := range players {
		got[p.Gamertag] = p.AuthOnly
	}
	if got["RealPlayer"] {
		t.Errorf("RealPlayer should not be auth_only")
	}
	if !got["TokenOnly"] {
		t.Errorf("TokenOnly should be auth_only=true")
	}
}

func TestLoadPlayersV3_InvalidJSON(t *testing.T) {
	cfg := &AppConfig{}
	_, err := cfg.loadPlayersV3([]byte(`notjson`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadPlayersV3_EmptyProfiles(t *testing.T) {
	cfg := &AppConfig{}
	data := []byte(`{"version": "3.0", "profiles": {}}`)
	players, err := cfg.loadPlayersV3(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 0 {
		t.Fatal("expected 0")
	}
}
