package service

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// getBoolSetting / getStringSetting
// ---------------------------------------------------------------------------

func TestGetBoolSetting_Found(t *testing.T) {
	s := map[string]interface{}{"k": true}
	if !getBoolSetting(s, "k", false) {
		t.Error("expected true")
	}
}

func TestGetBoolSetting_Missing(t *testing.T) {
	s := map[string]interface{}{}
	if getBoolSetting(s, "k", true) != true {
		t.Error("expected default true")
	}
}

func TestGetBoolSetting_WrongType(t *testing.T) {
	s := map[string]interface{}{"k": "notbool"}
	if getBoolSetting(s, "k", true) != true {
		t.Error("expected fallback to default when wrong type")
	}
}

func TestGetBoolSetting_NilMap(t *testing.T) {
	if getBoolSetting(nil, "k", false) {
		t.Error("expected false for nil map")
	}
}

func TestGetStringSetting_Found(t *testing.T) {
	s := map[string]interface{}{"lang": "en"}
	if getStringSetting(s, "lang", "fr") != "en" {
		t.Error("expected en")
	}
}

func TestGetStringSetting_Missing(t *testing.T) {
	s := map[string]interface{}{}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default fr")
	}
}

func TestGetStringSetting_EmptyValue(t *testing.T) {
	s := map[string]interface{}{"lang": ""}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default for empty string")
	}
}

func TestGetStringSetting_WrongType(t *testing.T) {
	s := map[string]interface{}{"lang": 42}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default for int type")
	}
}

// ---------------------------------------------------------------------------
// resolveSetupState
// ---------------------------------------------------------------------------

func TestResolveSetupState_NoPlayers(t *testing.T) {
	got := resolveSetupState(nil)
	if got != "no_halo_link" {
		t.Errorf("expected no_halo_link, got %s", got)
	}
}

func TestResolveSetupState_WithPlayers(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	got := resolveSetupState(players)
	if got != "profile_ready_no_sync" {
		t.Errorf("expected profile_ready_no_sync, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// ResolveAuthState
// ---------------------------------------------------------------------------

func TestResolveAuthState_NilSession(t *testing.T) {
	if ResolveAuthState(nil) != "missing" {
		t.Error("expected missing")
	}
}

func TestResolveAuthState_NotReady(t *testing.T) {
	sess := &domain.SessionData{AuthReady: false}
	if ResolveAuthState(sess) != "missing" {
		t.Error("expected missing")
	}
}

func TestResolveAuthState_NoIdentity(t *testing.T) {
	sess := &domain.SessionData{AuthReady: true}
	if ResolveAuthState(sess) != "partial" {
		t.Error("expected partial")
	}
}

func TestResolveAuthState_Ready(t *testing.T) {
	sess := &domain.SessionData{
		AuthReady:          true,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}
	if ResolveAuthState(sess) != "ready" {
		t.Error("expected ready")
	}
}

// ---------------------------------------------------------------------------
// ResolveLinkedIdentity
// ---------------------------------------------------------------------------

func TestResolveLinkedIdentity_Nil(t *testing.T) {
	if ResolveLinkedIdentity(nil) != nil {
		t.Error("expected nil")
	}
}

func TestResolveLinkedIdentity_NoIdentity(t *testing.T) {
	sess := &domain.SessionData{}
	if ResolveLinkedIdentity(sess) != nil {
		t.Error("expected nil when no identity")
	}
}

func TestResolveLinkedIdentity_WithIdentity(t *testing.T) {
	sess := &domain.SessionData{
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}
	identity := ResolveLinkedIdentity(sess)
	if identity == nil || identity.Gamertag != "GT" {
		t.Error("expected identity with GT")
	}
}

// ---------------------------------------------------------------------------
// buildCapabilities
// ---------------------------------------------------------------------------

func TestBuildCapabilities_DemoMode(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: true}
	settings := map[string]interface{}{}
	caps := buildCapabilities(cfg, settings)
	if caps.CanRunSync {
		t.Error("DemoMode should disable sync")
	}
	if caps.CanUseLiveHalo {
		t.Error("DemoMode should disable live Halo")
	}
}

func TestBuildCapabilities_Normal(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: false}
	settings := map[string]interface{}{"media_enabled": false}
	caps := buildCapabilities(cfg, settings)
	if !caps.CanRunSync {
		t.Error("normal mode should allow sync")
	}
	if caps.CanViewMedia {
		t.Error("media_enabled=false should disable CanViewMedia")
	}
}

// ---------------------------------------------------------------------------
// buildSettingsExcerpt
// ---------------------------------------------------------------------------

func TestBuildSettingsExcerpt_Defaults(t *testing.T) {
	cfg := &config.AppConfig{Lang: "fr"}
	settings := map[string]interface{}{}
	excerpt := buildSettingsExcerpt(cfg, settings)
	if excerpt.Lang != "fr" {
		t.Errorf("expected fr, got %s", excerpt.Lang)
	}
	if excerpt.UserTimezone != "Europe/Paris" {
		t.Errorf("expected Europe/Paris, got %s", excerpt.UserTimezone)
	}
}

func TestBuildSettingsExcerpt_Override(t *testing.T) {
	cfg := &config.AppConfig{Lang: "fr"}
	settings := map[string]interface{}{
		"lang":          "en",
		"user_timezone": "America/New_York",
	}
	excerpt := buildSettingsExcerpt(cfg, settings)
	if excerpt.Lang != "en" {
		t.Errorf("expected en, got %s", excerpt.Lang)
	}
	if excerpt.UserTimezone != "America/New_York" {
		t.Errorf("expected America/New_York, got %s", excerpt.UserTimezone)
	}
}

// ---------------------------------------------------------------------------
// buildFeatureFlags
// ---------------------------------------------------------------------------

func TestBuildFeatureFlags_Demo(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: true}
	settings := map[string]interface{}{}
	flags := buildFeatureFlags(cfg, settings)
	if !flags.DemoMode {
		t.Error("expected DemoMode true")
	}
	if !flags.V7Enabled {
		t.Error("expected V7Enabled true")
	}
}

func TestBuildFeatureFlags_Discord(t *testing.T) {
	cfg := &config.AppConfig{}
	settings := map[string]interface{}{
		"discord_webhook_url": "https://discord.com/hook",
	}
	flags := buildFeatureFlags(cfg, settings)
	if !flags.DiscordConfigured {
		t.Error("expected DiscordConfigured true")
	}
}

// ---------------------------------------------------------------------------
// BuildAvailableTitles
// ---------------------------------------------------------------------------

func TestBuildAvailableTitles(t *testing.T) {
	titles := BuildAvailableTitles()
	if len(titles) == 0 {
		t.Error("expected at least one title")
	}
	foundDefault := false
	for _, title := range titles {
		if title.IsDefault {
			foundDefault = true
		}
		if title.Slug == "" {
			t.Error("title slug should not be empty")
		}
	}
	if !foundDefault {
		t.Error("expected at least one default title")
	}
}
