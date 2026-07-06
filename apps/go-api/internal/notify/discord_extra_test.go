package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNotifyConfig_MissingFile(t *testing.T) {
	cfg := LoadNotifyConfig("/nonexistent/path.json")
	if cfg.WebhookURL != "" {
		t.Fatal("expected empty webhook for missing file")
	}
	if cfg.Lang != "fr" {
		t.Fatalf("expected fr, got %s", cfg.Lang)
	}
}

func TestLoadNotifyConfig_InvalidJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(p, []byte("{invalid"), 0644)
	cfg := LoadNotifyConfig(p)
	if cfg.WebhookURL != "" {
		t.Fatal("expected empty webhook for bad JSON")
	}
}

func TestLoadNotifyConfig_NotEnabled(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	data, _ := json.Marshal(map[string]any{
		"discord_notifications_enabled": false,
		"discord_webhook_url":           "https://discord.com/api/webhooks/123/abc",
	})
	_ = os.WriteFile(p, data, 0644)
	cfg := LoadNotifyConfig(p)
	if cfg.WebhookURL != "" {
		t.Fatal("expected empty webhook when disabled")
	}
}

func TestLoadNotifyConfig_ValidConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	data, _ := json.Marshal(map[string]any{
		"discord_notifications_enabled": true,
		"discord_webhook_url":           "https://discord.com/api/webhooks/123/abc",
		"discord_lang":                  "en",
		"discord_notify_sync":           false,
		"discord_notify_new_version":    true,
	})
	_ = os.WriteFile(p, data, 0644)
	cfg := LoadNotifyConfig(p)
	if cfg.WebhookURL != "https://discord.com/api/webhooks/123/abc" {
		t.Fatalf("unexpected webhook: %s", cfg.WebhookURL)
	}
	if cfg.Lang != "en" {
		t.Fatalf("expected en, got %s", cfg.Lang)
	}
	if cfg.NotifySync {
		t.Fatal("expected NotifySync false")
	}
}

func TestLoadNotifyConfig_InvalidURL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	data, _ := json.Marshal(map[string]any{
		"discord_notifications_enabled": true,
		"discord_webhook_url":           "https://example.com/not-discord",
	})
	_ = os.WriteFile(p, data, 0644)
	cfg := LoadNotifyConfig(p)
	if cfg.WebhookURL != "" {
		t.Fatal("expected empty for invalid URL prefix")
	}
}

func TestSendWebhook_EmptyURL(t *testing.T) {
	ok := SendWebhook("", WebhookPayload{})
	if ok {
		t.Fatal("expected false for empty URL")
	}
}

func TestSendWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// SendWebhook checks for discord.com prefix — we need to bypass.
	// Actually SendWebhook accepts any URL, it just posts to it.
	ok := SendWebhook(srv.URL, WebhookPayload{})
	if !ok {
		t.Fatal("expected true for 204 response")
	}
}

func TestSendWebhook_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ok := SendWebhook(srv.URL, WebhookPayload{})
	if ok {
		t.Fatal("expected false for 500 response")
	}
}
