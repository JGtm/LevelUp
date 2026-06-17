package notify

// discord_overlay_test.go — PMT-4 PR-2 : overlay Discord par titre.
//   parité Halo (overlay absent == global) · override synthetique (webhook/lang)
//   · héritage des champs non surchargés · gate discord_notifications_enabled
//   évalué sur la map résolue.

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDiscordJSON(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoadNotifyConfigForTitle(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "") // neutralise l'override env
	dir := t.TempDir()
	global := filepath.Join(dir, "app_settings.json")
	writeDiscordJSON(t, global, `{
		"discord_notifications_enabled": true,
		"discord_webhook_url": "https://discord.com/api/webhooks/111/global",
		"discord_lang": "en",
		"discord_notify_sync": true,
		"discord_notify_backfill": false
	}`)

	// (a) Parité Halo : pas d'overlay / overlay absent ⇒ == LoadNotifyConfig.
	base := LoadNotifyConfig(global)
	if got := LoadNotifyConfigForTitle(global, ""); got != base {
		t.Errorf("overlay vide != LoadNotifyConfig : %+v vs %+v", got, base)
	}
	if got := LoadNotifyConfigForTitle(global, filepath.Join(dir, "absent.json")); got != base {
		t.Errorf("overlay absent != LoadNotifyConfig")
	}

	// (b) Overlay synthétique : webhook + lang surchargés, toggles hérités.
	overlay := filepath.Join(dir, "overlay.json")
	writeDiscordJSON(t, overlay, `{
		"discord_webhook_url": "https://discord.com/api/webhooks/222/synth",
		"discord_lang": "fr"
	}`)
	o := LoadNotifyConfigForTitle(global, overlay)
	if o.WebhookURL != "https://discord.com/api/webhooks/222/synth" {
		t.Errorf("webhook = %q, want .../222/synth (overlay)", o.WebhookURL)
	}
	if o.Lang != "fr" {
		t.Errorf("lang = %q, want fr (overlay)", o.Lang)
	}
	if !o.NotifySync {
		t.Error("discord_notify_sync devrait hériter du global (true)")
	}
	if o.NotifyBackfill {
		t.Error("discord_notify_backfill devrait hériter du global (false)")
	}

	// Gate : overlay désactive les notifications ⇒ webhook vide (résolu sur l'overlay).
	overlayOff := filepath.Join(dir, "overlay_off.json")
	writeDiscordJSON(t, overlayOff, `{"discord_notifications_enabled": false}`)
	if d := LoadNotifyConfigForTitle(global, overlayOff); d.WebhookURL != "" {
		t.Errorf("overlay disable non respecté : webhook = %q, want \"\"", d.WebhookURL)
	}
}
