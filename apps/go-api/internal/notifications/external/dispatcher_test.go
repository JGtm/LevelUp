package external

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
)

// fakeNotifier enregistre les appels sur un canal (pour synchroniser l'async).
type fakeNotifier struct {
	calls chan ExternalNotification
	err   error
}

func (f *fakeNotifier) Notify(_ context.Context, _ string, n ExternalNotification) error {
	f.calls <- n
	return f.err
}

// writeSettings écrit un app_settings.json temporaire et retourne son chemin.
func writeSettings(t *testing.T, m map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "app_settings.json")
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return p
}

const validWebhook = "https://discord.com/api/webhooks/1/abc"

func activeSettings(t *testing.T) string {
	return writeSettings(t, map[string]any{
		"discord_notifications_enabled": true,
		"discord_notify_coach":          true,
		"discord_webhook_url":           validWebhook,
	})
}

func newDispatcherWith(t *testing.T, settingsPath string, fake *fakeNotifier) *Dispatcher {
	return NewDispatcher(Config{
		AppSettingsPath: settingsPath,
		Player:          "JGtm",
		TitleSlug:       "halo_infinite",
		Notifier:        fake,
	})
}

func coachNotif() *notifications.Notification {
	return &notifications.Notification{
		Category: notifications.CategoryMilestoneUnlocked,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.milestone_unlocked.title",
	}
}

// TestForward_ActiveForwardedCategory : catégorie coach + relais actif → appel.
func TestForward_ActiveForwardedCategory(t *testing.T) {
	fake := &fakeNotifier{calls: make(chan ExternalNotification, 1)}
	d := newDispatcherWith(t, activeSettings(t), fake)

	d.Forward(context.Background(), coachNotif())

	select {
	case n := <-fake.calls:
		if n.Category != "milestone_unlocked" || n.Player != "JGtm" {
			t.Errorf("notification relayée inattendue : %+v", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("aucun relais reçu ; l'envoi async n'a pas eu lieu")
	}
}

// TestForward_FlagOff : discord_notify_coach=false → aucun appel.
func TestForward_FlagOff(t *testing.T) {
	settings := writeSettings(t, map[string]any{
		"discord_notifications_enabled": true,
		"discord_notify_coach":          false,
		"discord_webhook_url":           validWebhook,
	})
	fake := &fakeNotifier{calls: make(chan ExternalNotification, 1)}
	d := newDispatcherWith(t, settings, fake)

	d.Forward(context.Background(), coachNotif())
	assertNoCall(t, fake)
}

// TestForward_WebhookAbsent : master Discord OFF (webhook blanchi) → aucun appel.
func TestForward_WebhookAbsent(t *testing.T) {
	settings := writeSettings(t, map[string]any{
		"discord_notifications_enabled": false,
		"discord_notify_coach":          true,
		"discord_webhook_url":           validWebhook,
	})
	fake := &fakeNotifier{calls: make(chan ExternalNotification, 1)}
	d := newDispatcherWith(t, settings, fake)

	d.Forward(context.Background(), coachNotif())
	assertNoCall(t, fake)
}

// TestForward_NonForwardedCategory : catégorie hors coach → aucun appel, même actif.
func TestForward_NonForwardedCategory(t *testing.T) {
	fake := &fakeNotifier{calls: make(chan ExternalNotification, 1)}
	d := newDispatcherWith(t, activeSettings(t), fake)

	n := coachNotif()
	n.Category = notifications.CategoryMatchSynced // notification de sync, non coach
	d.Forward(context.Background(), n)
	assertNoCall(t, fake)
}

// TestForward_NilSafe : dispatcher/notif nil → pas de panic.
func TestForward_NilSafe(t *testing.T) {
	var d *Dispatcher
	d.Forward(context.Background(), coachNotif()) // dispatcher nil
	d2 := newDispatcherWith(t, activeSettings(t), &fakeNotifier{calls: make(chan ExternalNotification, 1)})
	d2.Forward(context.Background(), nil) // notif nil
}

// assertNoCall vérifie qu'aucun relais n'est reçu dans une courte fenêtre.
func assertNoCall(t *testing.T, fake *fakeNotifier) {
	t.Helper()
	select {
	case n := <-fake.calls:
		t.Fatalf("relais inattendu : %+v", n)
	case <-time.After(150 * time.Millisecond):
	}
}
