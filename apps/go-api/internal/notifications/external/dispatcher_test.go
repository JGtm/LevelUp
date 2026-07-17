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

// TestDecodeParams couvre les trois issues de la désérialisation best-effort des
// params : payload vide → nil (aucun bruit), JSON valide → map, JSON illisible →
// nil APRÈS un log Debug (embed sans détails, jamais de crash ni de 500).
func TestDecodeParams(t *testing.T) {
	ctx := context.Background()
	if got := decodeParams(ctx, nil); got != nil {
		t.Errorf("payload nil → nil attendu, obtenu %v", got)
	}
	if got := decodeParams(ctx, json.RawMessage(``)); got != nil {
		t.Errorf("payload vide → nil attendu, obtenu %v", got)
	}
	got := decodeParams(ctx, json.RawMessage(`{"metric":"kills","value":100}`))
	if got == nil || got["metric"] != "kills" {
		t.Errorf("JSON valide → map décodée attendue, obtenu %v", got)
	}
	if got := decodeParams(ctx, json.RawMessage(`{not-json`)); got != nil {
		t.Errorf("JSON illisible → nil attendu (dégradation propre), obtenu %v", got)
	}
}

// TestAppLink couvre la construction du lien app profond : base + route (avec
// normalisation des slashes), et les deux cas « pas de lien » (base vide, route
// vide) qui doivent retourner "" pour omettre le champ.
func TestAppLink(t *testing.T) {
	withBase := NewDispatcher(Config{AppBaseURL: "https://app.example.com/"})
	if got := withBase.appLink("/players/JGtm/matches/m1"); got != "https://app.example.com/players/JGtm/matches/m1" {
		t.Errorf("lien profond attendu normalisé, obtenu %q", got)
	}
	if got := withBase.appLink(""); got != "" {
		t.Errorf("route vide → pas de lien attendu, obtenu %q", got)
	}
	noBase := NewDispatcher(Config{})
	if got := noBase.appLink("/players/JGtm"); got != "" {
		t.Errorf("base vide → pas de lien attendu, obtenu %q", got)
	}
}

// TestDeliver_NotifierError couvre la branche d'échec best-effort de deliver :
// l'erreur du notifier est loguée + comptée, mais deliver ne panique pas et
// retourne proprement (appel synchrone direct pour éviter la course goroutine).
func TestDeliver_NotifierError(t *testing.T) {
	fake := &fakeNotifier{calls: make(chan ExternalNotification, 1), err: errInjected}
	d := newDispatcherWith(t, activeSettings(t), fake)
	d.deliver(context.Background(), validWebhook, ExternalNotification{Category: "milestone_unlocked"})
	select {
	case <-fake.calls:
	default:
		t.Fatal("le notifier aurait dû être appelé même en erreur")
	}
}

var errInjected = errInjectedType{}

type errInjectedType struct{}

func (errInjectedType) Error() string { return "injected notifier error" }

// assertNoCall vérifie qu'aucun relais n'est reçu dans une courte fenêtre.
func assertNoCall(t *testing.T, fake *fakeNotifier) {
	t.Helper()
	select {
	case n := <-fake.calls:
		t.Fatalf("relais inattendu : %+v", n)
	case <-time.After(150 * time.Millisecond):
	}
}
