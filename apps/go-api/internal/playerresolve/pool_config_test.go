package playerresolve

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// ── buildPoolConfig ──────────────────────────────────────────────────────────

func TestBuildPoolConfig_LegacyTitle(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: "/tmp/repo"}
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
	cfg := &config.AppConfig{RepoRoot: "/tmp/repo"}
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
