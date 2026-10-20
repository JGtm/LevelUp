package config

import (
	"os"
	"testing"
)

// ---------------------------------------------------------------------------
// getEnvOrDefault
// ---------------------------------------------------------------------------

func TestGetEnvOrDefault_EnvSet(t *testing.T) {
	t.Setenv("_TEST_LEVELUP_KEY", "fromenv")
	got := getEnvOrDefault("_TEST_LEVELUP_KEY", "default")
	if got != "fromenv" {
		t.Errorf("expected fromenv, got %s", got)
	}
}

func TestGetEnvOrDefault_EnvNotSet(t *testing.T) {
	_ = os.Unsetenv("_TEST_LEVELUP_KEY2")
	got := getEnvOrDefault("_TEST_LEVELUP_KEY2", "default")
	if got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// getEnvInt
// ---------------------------------------------------------------------------

func TestGetEnvInt_ValidInt(t *testing.T) {
	t.Setenv("_TEST_PORT", "8080")
	got := getEnvInt("_TEST_PORT", 3000)
	if got != 8080 {
		t.Errorf("expected 8080, got %d", got)
	}
}

func TestGetEnvInt_InvalidInt(t *testing.T) {
	t.Setenv("_TEST_PORT_BAD", "notanint")
	got := getEnvInt("_TEST_PORT_BAD", 3000)
	if got != 3000 {
		t.Errorf("expected default 3000, got %d", got)
	}
}

func TestGetEnvInt_Empty(t *testing.T) {
	_ = os.Unsetenv("_TEST_PORT_EMPTY")
	got := getEnvInt("_TEST_PORT_EMPTY", 3000)
	if got != 3000 {
		t.Errorf("expected default 3000, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// parseCORSOrigins
// ---------------------------------------------------------------------------

func TestParseCORSOrigins_Empty(t *testing.T) {
	origins := parseCORSOrigins("")
	if len(origins) != 4 {
		t.Errorf("expected 4 defaults, got %d", len(origins))
	}
}

func TestParseCORSOrigins_Custom(t *testing.T) {
	origins := parseCORSOrigins("http://a.com, http://b.com")
	if len(origins) != 2 {
		t.Errorf("expected 2, got %d", len(origins))
	}
	if origins[0] != "http://a.com" || origins[1] != "http://b.com" {
		t.Errorf("unexpected origins: %v", origins)
	}
}

func TestParseCORSOrigins_TrailingComma(t *testing.T) {
	origins := parseCORSOrigins("http://a.com,,")
	if len(origins) != 1 {
		t.Errorf("expected 1, got %d: %v", len(origins), origins)
	}
}

// ---------------------------------------------------------------------------
// ServerAddr
// ---------------------------------------------------------------------------

func TestServerAddr(t *testing.T) {
	cfg := &AppConfig{APIHost: "0.0.0.0", APIPort: 8080}
	if cfg.ServerAddr() != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", cfg.ServerAddr())
	}
}

// ---------------------------------------------------------------------------
// loadDiscordWebhookURL
// ---------------------------------------------------------------------------

func TestLoadDiscordWebhookURL_FromEnv(t *testing.T) {
	url := "https://discord.com/api/webhooks/123/abc"
	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", url)
	got := loadDiscordWebhookURL("/nonexistent")
	if got != url {
		t.Errorf("expected env url, got %s", got)
	}
}

func TestLoadDiscordWebhookURL_FromFile(t *testing.T) {
	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", "")
	t.Setenv("DISCORD_WEBHOOK_URL", "") // neutraliser la variable legacy
	tmpDir := t.TempDir()
	settingsPath := tmpDir + "/app_settings.json"
	content := `{"discord_webhook_url":"https://discord.com/api/webhooks/456/def"}`
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadDiscordWebhookURL(settingsPath)
	if got != "https://discord.com/api/webhooks/456/def" {
		t.Errorf("expected file url, got %s", got)
	}
}

func TestLoadDiscordWebhookURL_InvalidJSON(t *testing.T) {
	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", "")
	t.Setenv("DISCORD_WEBHOOK_URL", "") // neutraliser la variable legacy
	tmpDir := t.TempDir()
	settingsPath := tmpDir + "/app_settings.json"
	if err := os.WriteFile(settingsPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadDiscordWebhookURL(settingsPath)
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestLoadDiscordWebhookURL_BadPrefix(t *testing.T) {
	t.Setenv("LEVELUP_DISCORD_WEBHOOK_URL", "")
	t.Setenv("DISCORD_WEBHOOK_URL", "") // neutraliser la variable legacy
	tmpDir := t.TempDir()
	settingsPath := tmpDir + "/app_settings.json"
	content := `{"discord_webhook_url":"http://evil.com/hook"}`
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadDiscordWebhookURL(settingsPath)
	if got != "" {
		t.Errorf("expected empty for bad prefix, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// LoadAppSettings
// ---------------------------------------------------------------------------

func TestLoadAppSettings_NotExist(t *testing.T) {
	cfg := &AppConfig{AppSettingsPath: "/nonexistent/settings.json"}
	m, err := cfg.LoadAppSettings()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestLoadAppSettings_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/app_settings.json"
	content := `{"lang":"fr","debug":true}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &AppConfig{AppSettingsPath: path}
	m, err := cfg.LoadAppSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["lang"] != "fr" {
		t.Errorf("expected lang=fr, got %v", m["lang"])
	}
}

func TestLoadAppSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/app_settings.json"
	if err := os.WriteFile(path, []byte("{bad json}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &AppConfig{AppSettingsPath: path}
	_, err := cfg.LoadAppSettings()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// loadEnvLocal
// ---------------------------------------------------------------------------

func TestLoadEnvLocal_SetsVars(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := tmpDir + "/.env.local"
	content := "LEVELUP_TEST_ENVLOCAL_SET=my_token\nLEVELUP_DEMO_MODE=false\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Unsetenv("LEVELUP_TEST_ENVLOCAL_SET")
	loadEnvLocal(envFile)
	if got := os.Getenv("LEVELUP_TEST_ENVLOCAL_SET"); got != "my_token" {
		t.Errorf("expected my_token, got %q", got)
	}
}

func TestLoadEnvLocal_DoesNotOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := tmpDir + "/.env.local"
	content := "LEVELUP_TEST_ENVLOCAL_OVER=from_file\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LEVELUP_TEST_ENVLOCAL_OVER", "from_env")
	loadEnvLocal(envFile)
	if got := os.Getenv("LEVELUP_TEST_ENVLOCAL_OVER"); got != "from_env" {
		t.Errorf("env var existante ne doit pas être écrasée, got %q", got)
	}
}

func TestLoadEnvLocal_FileMissing(t *testing.T) {
	// Aucune panique — retour silencieux.
	loadEnvLocal("/nonexistent/.env.local")
}

func TestLoadEnvLocal_QuotedValue(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := tmpDir + "/.env.local"
	content := "LEVELUP_TEST_ENVLOCAL_QUOTED=\"quoted_token\"\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Unsetenv("LEVELUP_TEST_ENVLOCAL_QUOTED")
	loadEnvLocal(envFile)
	if got := os.Getenv("LEVELUP_TEST_ENVLOCAL_QUOTED"); got != "quoted_token" {
		t.Errorf("expected quoted_token, got %q", got)
	}
}
