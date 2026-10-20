// Package settings — store_test.go : tests unitaires du Store de settings.
package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/settings"
)

func newTestStore(t *testing.T, content map[string]interface{}) *settings.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if content != nil {
		data, _ := json.Marshal(content)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return settings.NewStore(path)
}

func TestStore_Load_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	store := settings.NewStore(filepath.Join(dir, "nonexistent.json"))
	cfg, err := store.Load()

	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default settings, got nil")
	}
	// CanStartInitialSync et CanSelfProvision doivent être true par défaut
	if !cfg.CanStartInitialSync {
		t.Error("CanStartInitialSync should be true by default")
	}
	if !cfg.CanSelfProvision {
		t.Error("CanSelfProvision should be true by default")
	}
}

func TestStore_Load_ValidFile(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang":                   "en",
		"can_start_initial_sync": false,
		"can_self_provision":     true,
	})
	cfg, err := store.Load()

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Lang != "en" {
		t.Errorf("expected lang='en', got %q", cfg.Lang)
	}
	if cfg.CanStartInitialSync {
		t.Error("CanStartInitialSync should be false from file")
	}
}

func TestStore_Load_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	_ = os.WriteFile(path, []byte("{invalid json"), 0o600)
	store := settings.NewStore(path)

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStore_Save_RoundTrip(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang": "fr",
	})

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	cfg.Lang = "de"

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Recharger
	cfg2, err := store.Load()
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if cfg2.Lang != "de" {
		t.Errorf("expected lang='de' after save, got %q", cfg2.Lang)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Apply
// ─────────────────────────────────────────────────────────────────────────────

func TestApply_UpdatesFields(t *testing.T) {
	cfg := settings.Defaults()
	lang := "fr"
	discordLang := "en"
	req := &domain.UpdateSettingsRequest{
		Lang:        &lang,
		DiscordLang: &discordLang,
	}
	settings.Apply(cfg, req)
	if cfg.Lang != "fr" {
		t.Errorf("expected lang=fr, got %q", cfg.Lang)
	}
	if cfg.DiscordLang != "en" {
		t.Errorf("expected discord_lang=en, got %q", cfg.DiscordLang)
	}
}

func TestApply_NilFieldsUnchanged(t *testing.T) {
	cfg := settings.Defaults()
	cfg.Lang = "de"
	req := &domain.UpdateSettingsRequest{} // tous les champs nil
	settings.Apply(cfg, req)
	if cfg.Lang != "de" {
		t.Errorf("nil req should not change lang, got %q", cfg.Lang)
	}
}

func TestApply_BoolFields(t *testing.T) {
	cfg := settings.Defaults()
	tr := true
	req := &domain.UpdateSettingsRequest{
		NormalizeModeLabels:         &tr,
		DiscordNotificationsEnabled: &tr,
	}
	settings.Apply(cfg, req)
	if !cfg.NormalizeModeLabels {
		t.Error("NormalizeModeLabels should be true")
	}
	if !cfg.DiscordNotificationsEnabled {
		t.Error("DiscordNotificationsEnabled should be true")
	}
}

func TestApply_MediaFields(t *testing.T) {
	cfg := settings.Defaults()
	dir := "/captures"
	tol := 20
	req := &domain.UpdateSettingsRequest{
		MediaCapturesBaseDir:  &dir,
		MediaToleranceMinutes: &tol,
	}
	settings.Apply(cfg, req)
	if cfg.MediaCapturesBaseDir != "/captures" {
		t.Errorf("expected /captures, got %q", cfg.MediaCapturesBaseDir)
	}
	if cfg.MediaBufferMinutes != 20 {
		t.Errorf("expected 20, got %d", cfg.MediaBufferMinutes)
	}
}

func TestApply_MediaWatcherEnabled(t *testing.T) {
	cfg := settings.Defaults()
	tr := true
	req := &domain.UpdateSettingsRequest{MediaWatcherEnabled: &tr}
	settings.Apply(cfg, req)
	if !cfg.MediaWatcherEnabled {
		t.Error("expected MediaWatcherEnabled=true after Apply")
	}
}

func TestApply_SpnkrFields(t *testing.T) {
	cfg := settings.Defaults()
	tr := true
	req := &domain.UpdateSettingsRequest{
		SpnkrRefreshWithBackfill:           &tr,
		SpnkrRefreshBackfillMedals:         &tr,
		SpnkrRefreshBackfillSkill:          &tr,
		SpnkrRefreshBackfillAliases:        &tr,
		SpnkrRefreshBackfillPersonalScores: &tr,
		SpnkrRefreshBackfillPerfScores:     &tr,
		SpnkrRefreshBackfillLUSR:           &tr,
		SpnkrRefreshBackfillEvents:         &tr,
	}
	settings.Apply(cfg, req)
	if !cfg.SpnkrRefreshWithBackfill {
		t.Error("SpnkrRefreshWithBackfill should be true")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ToResponse
// ─────────────────────────────────────────────────────────────────────────────

func TestToResponse_HidesWebhookURL(t *testing.T) {
	cfg := settings.Defaults()
	cfg.DiscordWebhookURL = "https://discord.com/api/webhooks/secret"
	resp := settings.ToResponse(cfg)
	if !resp.DiscordWebhookURLPresent {
		t.Error("DiscordWebhookURLPresent should be true when URL set")
	}
}

func TestToResponse_EmptyWebhookURL(t *testing.T) {
	// La présence résout désormais env > store : neutraliser l'env pour tester le cas
	// « ni env ni store » de façon déterministe (sinon un DISCORD_WEBHOOK_URL présent
	// dans l'environnement du runner ferait basculer le flag).
	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", "")
	t.Setenv("DISCORD_WEBHOOK_URL", "")
	cfg := settings.Defaults()
	cfg.DiscordWebhookURL = ""
	resp := settings.ToResponse(cfg)
	if resp.DiscordWebhookURLPresent {
		t.Error("DiscordWebhookURLPresent should be false when URL empty")
	}
}

func TestToResponse_FieldsMapped(t *testing.T) {
	cfg := settings.Defaults()
	cfg.Lang = "fr"
	cfg.UserTimezone = "America/New_York"
	resp := settings.ToResponse(cfg)
	if resp.Lang != "fr" {
		t.Errorf("Lang: expected fr, got %q", resp.Lang)
	}
	if resp.UserTimezone != "America/New_York" {
		t.Errorf("UserTimezone: expected America/New_York, got %q", resp.UserTimezone)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Defaults
// ─────────────────────────────────────────────────────────────────────────────

func TestDefaults_ReturnsNonNil(t *testing.T) {
	d := settings.Defaults()
	if d == nil {
		t.Fatal("Defaults() should not return nil")
	}
}

func TestDefaults_CanStartInitialSync(t *testing.T) {
	d := settings.Defaults()
	if !d.CanStartInitialSync {
		t.Error("CanStartInitialSync default should be true")
	}
	if !d.CanSelfProvision {
		t.Error("CanSelfProvision default should be true")
	}
}

func TestDefaults_Lang(t *testing.T) {
	d := settings.Defaults()
	if d.Lang != "en" {
		t.Errorf("default lang should be 'en', got %q", d.Lang)
	}
	if d.UserTimezone != "Europe/Paris" {
		t.Errorf("default timezone should be 'Europe/Paris', got %q", d.UserTimezone)
	}
}

func TestStore_Load_DefaultCapabilities(t *testing.T) {
	// Fichier sans can_start_initial_sync → default true
	store := newTestStore(t, map[string]interface{}{
		"lang": "fr",
		// pas de can_start_initial_sync
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CanStartInitialSync {
		t.Error("absent can_start_initial_sync should default to true")
	}
}

// ─── Nouveaux champs Analyse (sessions + badges) ─────────────────────────────

func TestDefaults_SessionAndBadgeFields(t *testing.T) {
	d := settings.Defaults()
	if d.SessionGapMinutes != 120 {
		t.Errorf("SessionGapMinutes default = %d, want 120", d.SessionGapMinutes)
	}
	if d.SessionTeamChangeMode != "friends" {
		t.Errorf("SessionTeamChangeMode default = %q, want 'friends'", d.SessionTeamChangeMode)
	}
	if d.SessionSplitOnRankedChange {
		t.Error("SessionSplitOnRankedChange default should be false")
	}
	if d.OutcomeBadgeSensitivity != "standard" {
		t.Errorf("OutcomeBadgeSensitivity default = %q, want 'standard'", d.OutcomeBadgeSensitivity)
	}
	if !d.OutcomeExcludeBotMatchesFromBadges {
		t.Error("OutcomeExcludeBotMatchesFromBadges default should be true")
	}
	if d.OutcomeExcludeBotMatchesFromRecords {
		t.Error("OutcomeExcludeBotMatchesFromRecords default should be false")
	}
}

func TestApply_SessionGapMinutes(t *testing.T) {
	d := settings.Defaults()
	gap := 60
	req := &domain.UpdateSettingsRequest{SessionGapMinutes: &gap}
	settings.Apply(d, req)
	if d.SessionGapMinutes != 60 {
		t.Errorf("SessionGapMinutes = %d after Apply, want 60", d.SessionGapMinutes)
	}
}

func TestApply_SessionTeamChangeMode(t *testing.T) {
	d := settings.Defaults()
	mode := "ignore"
	req := &domain.UpdateSettingsRequest{SessionTeamChangeMode: &mode}
	settings.Apply(d, req)
	if d.SessionTeamChangeMode != "ignore" {
		t.Errorf("SessionTeamChangeMode = %q after Apply, want 'ignore'", d.SessionTeamChangeMode)
	}
}

func TestApply_SessionSplitOnRankedChange(t *testing.T) {
	d := settings.Defaults()
	v := true
	req := &domain.UpdateSettingsRequest{SessionSplitOnRankedChange: &v}
	settings.Apply(d, req)
	if !d.SessionSplitOnRankedChange {
		t.Error("SessionSplitOnRankedChange should be true after Apply(true)")
	}
}

func TestApply_OutcomeBadgeSensitivity(t *testing.T) {
	d := settings.Defaults()
	sens := "strict"
	req := &domain.UpdateSettingsRequest{OutcomeBadgeSensitivity: &sens}
	settings.Apply(d, req)
	if d.OutcomeBadgeSensitivity != "strict" {
		t.Errorf("OutcomeBadgeSensitivity = %q after Apply, want 'strict'", d.OutcomeBadgeSensitivity)
	}
}

func TestApply_OutcomeExcludeBotFields(t *testing.T) {
	d := settings.Defaults()
	exclBadges := false
	exclRecords := true
	req := &domain.UpdateSettingsRequest{
		OutcomeExcludeBotMatchesFromBadges:  &exclBadges,
		OutcomeExcludeBotMatchesFromRecords: &exclRecords,
	}
	settings.Apply(d, req)
	if d.OutcomeExcludeBotMatchesFromBadges {
		t.Error("OutcomeExcludeBotMatchesFromBadges should be false after Apply(false)")
	}
	if !d.OutcomeExcludeBotMatchesFromRecords {
		t.Error("OutcomeExcludeBotMatchesFromRecords should be true after Apply(true)")
	}
}

func TestToResponse_NewAnalyseFields(t *testing.T) {
	d := settings.Defaults()
	resp := settings.ToResponse(d)
	if resp.SessionGapMinutes != d.SessionGapMinutes {
		t.Error("ToResponse: SessionGapMinutes not mapped")
	}
	if resp.SessionTeamChangeMode != d.SessionTeamChangeMode {
		t.Error("ToResponse: SessionTeamChangeMode not mapped")
	}
	if resp.OutcomeBadgeSensitivity != d.OutcomeBadgeSensitivity {
		t.Error("ToResponse: OutcomeBadgeSensitivity not mapped")
	}
	if resp.OutcomeExcludeBotMatchesFromBadges != d.OutcomeExcludeBotMatchesFromBadges {
		t.Error("ToResponse: OutcomeExcludeBotMatchesFromBadges not mapped")
	}
	if resp.OutcomeExcludeBotMatchesFromRecords != d.OutcomeExcludeBotMatchesFromRecords {
		t.Error("ToResponse: OutcomeExcludeBotMatchesFromRecords not mapped")
	}
}

// ─── ShowProgression (toggle Objectifs/Prestige) ────────────────────────────

func TestDefaults_ShowProgression(t *testing.T) {
	d := settings.Defaults()
	if !d.ShowProgression {
		t.Error("ShowProgression default should be true")
	}
}

func TestStore_Load_ShowProgressionDefaultsTrueWhenAbsent(t *testing.T) {
	// Fichier existant sans show_progression → rétrocompat : default true
	store := newTestStore(t, map[string]interface{}{
		"lang": "fr",
		// show_progression absent
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.ShowProgression {
		t.Error("absent show_progression should default to true")
	}
}

func TestStore_Load_ShowProgressionFalseRespected(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang":             "fr",
		"show_progression": false,
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ShowProgression {
		t.Error("explicit show_progression=false must be respected")
	}
}

func TestApply_ShowProgression(t *testing.T) {
	cfg := settings.Defaults()
	v := false
	req := &domain.UpdateSettingsRequest{ShowProgression: &v}
	settings.Apply(cfg, req)
	if cfg.ShowProgression {
		t.Error("ShowProgression should be false after Apply(false)")
	}
}

func TestToResponse_ShowProgressionMapped(t *testing.T) {
	cfg := settings.Defaults()
	cfg.ShowProgression = false
	resp := settings.ToResponse(cfg)
	if resp.ShowProgression {
		t.Error("ToResponse: ShowProgression=false not propagated")
	}

	cfg.ShowProgression = true
	resp = settings.ToResponse(cfg)
	if !resp.ShowProgression {
		t.Error("ToResponse: ShowProgression=true not propagated")
	}
}

func TestStore_SaveLoadRoundTrip_ShowProgression(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{"lang": "fr"})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.ShowProgression = false
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg2, err := store.Load()
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if cfg2.ShowProgression {
		t.Error("ShowProgression=false not persisted across save/load")
	}
}

// ─── CoachProactiveMode (toggle pont coach → Prestige, ADR 0020) ────────────

func TestDefaults_CoachProactiveMode(t *testing.T) {
	// DEC-2 : bascule du défaut à true (2026-07-22).
	d := settings.Defaults()
	if !d.CoachProactiveMode {
		t.Error("CoachProactiveMode default should be true (DEC-2)")
	}
}

func TestStore_Load_CoachProactiveModeDefaultsTrueWhenAbsent(t *testing.T) {
	// Fichier existant sans coach_proactive_mode → rétrocompat : default true (DEC-2).
	store := newTestStore(t, map[string]interface{}{"lang": "fr"})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CoachProactiveMode {
		t.Error("absent coach_proactive_mode should default to true (DEC-2)")
	}
}

func TestStore_Load_CoachProactiveModeFalseRespected(t *testing.T) {
	// Opt-out explicite (false dans le fichier) reste respecté malgré le défaut true.
	store := newTestStore(t, map[string]interface{}{
		"lang":                 "fr",
		"coach_proactive_mode": false,
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CoachProactiveMode {
		t.Error("explicit coach_proactive_mode=false must be respected")
	}
}

func TestStore_Load_CoachProactiveModeTrueRespected(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{
		"lang":                 "fr",
		"coach_proactive_mode": true,
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CoachProactiveMode {
		t.Error("explicit coach_proactive_mode=true must be respected")
	}
}

func TestApply_CoachProactiveMode(t *testing.T) {
	cfg := settings.Defaults()
	v := true
	req := &domain.UpdateSettingsRequest{CoachProactiveMode: &v}
	settings.Apply(cfg, req)
	if !cfg.CoachProactiveMode {
		t.Error("CoachProactiveMode should be true after Apply(true)")
	}
}

func TestToResponse_CoachProactiveModeMapped(t *testing.T) {
	cfg := settings.Defaults()
	cfg.CoachProactiveMode = true
	resp := settings.ToResponse(cfg)
	if !resp.CoachProactiveMode {
		t.Error("ToResponse: CoachProactiveMode=true not propagated")
	}

	cfg.CoachProactiveMode = false
	resp = settings.ToResponse(cfg)
	if resp.CoachProactiveMode {
		t.Error("ToResponse: CoachProactiveMode=false not propagated")
	}
}

func TestStore_SaveLoadRoundTrip_CoachProactiveMode(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{"lang": "fr"})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.CoachProactiveMode = true
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg2, err := store.Load()
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if !cfg2.CoachProactiveMode {
		t.Error("CoachProactiveMode=true not persisted across save/load")
	}
}
