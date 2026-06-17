package settings

// overlay_test.go — PMT-4 : oracle DOUBLE du seam « base globale + overlay titre ».
//   (a) parité Halo : sans overlay, ResolveForTitle == Load (byte-identique) ;
//   (b) overlay synthétique : champs surchargés routés, non-surchargés hérités.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeSettingsJSON(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestResolveForTitle_HaloParity (oracle a) : overlayPath vide / fichier absent /
// overlay vide ⇒ ResolveForTitle est strictement égal à Load (le titre par défaut
// hérite intégralement du global).
func TestResolveForTitle_HaloParity(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "app_settings.json")
	writeSettingsJSON(t, global, `{
		"discord_webhook_url": "https://hook.global",
		"session_gap_minutes": 90,
		"friend_gamertags": ["A", "B"],
		"discord_notify_sync": true,
		"coach_proactive_mode": true
	}`)
	store := NewStore(global)

	base, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := map[string]string{
		"overlayPath vide": "",
		"overlay absent":   filepath.Join(dir, "absent", "settings.json"),
	}
	empty := filepath.Join(dir, "empty.json")
	writeSettingsJSON(t, empty, `{}`)
	cases["overlay vide {}"] = empty

	for name, overlayPath := range cases {
		got, err := store.ResolveForTitle(overlayPath)
		if err != nil {
			t.Fatalf("[%s] ResolveForTitle: %v", name, err)
		}
		if !reflect.DeepEqual(got, base) {
			t.Errorf("[%s] ResolveForTitle != Load (parité Halo cassée)", name)
		}
	}
}

// TestResolveForTitle_SyntheticOverlay (oracle b) : l'overlay surcharge certains
// champs (routage réel) ; les champs absents de l'overlay héritent du global.
func TestResolveForTitle_SyntheticOverlay(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "app_settings.json")
	writeSettingsJSON(t, global, `{
		"discord_webhook_url": "https://hook.global",
		"session_gap_minutes": 90,
		"friend_gamertags": ["A", "B"],
		"discord_notify_sync": true,
		"show_records": true
	}`)
	store := NewStore(global)

	overlay := filepath.Join(dir, "overlay.json")
	writeSettingsJSON(t, overlay, `{
		"discord_webhook_url": "https://hook.synth",
		"session_gap_minutes": 30,
		"friend_gamertags": ["X"]
	}`)

	got, err := store.ResolveForTitle(overlay)
	if err != nil {
		t.Fatalf("ResolveForTitle: %v", err)
	}

	// Surchargés par l'overlay.
	if got.DiscordWebhookURL != "https://hook.synth" {
		t.Errorf("webhook = %q, want https://hook.synth (overlay)", got.DiscordWebhookURL)
	}
	if got.SessionGapMinutes != 30 {
		t.Errorf("session_gap = %d, want 30 (overlay)", got.SessionGapMinutes)
	}
	if len(got.FriendGamertags) != 1 || got.FriendGamertags[0] != "X" {
		t.Errorf("friends = %v, want [X] (overlay)", got.FriendGamertags)
	}
	// Hérités du global (absents de l'overlay).
	if !got.DiscordNotifySync {
		t.Error("discord_notify_sync devrait hériter du global (true)")
	}
	if !got.ShowRecords {
		t.Error("show_records devrait hériter du global (true)")
	}
}

// TestTitleSettingsPath_AbsentMeansInherit : le fichier overlay est optionnel —
// son absence (cas nominal Halo) est gérée comme un no-op par ResolveForTitle.
// (Le chemin lui-même est testé côté title.PathResolver.)
func TestResolveForTitle_GlobalAbsentDefaults(t *testing.T) {
	dir := t.TempDir()
	// Global minimal SANS can_self_provision/can_start_initial_sync/show_progression :
	// ResolveForTitle doit réappliquer les défauts « absent → true » (comme Load).
	global := filepath.Join(dir, "app_settings.json")
	writeSettingsJSON(t, global, `{"lang":"fr"}`)
	store := NewStore(global)

	overlay := filepath.Join(dir, "overlay.json")
	writeSettingsJSON(t, overlay, `{"session_gap_minutes": 45}`)

	got, err := store.ResolveForTitle(overlay)
	if err != nil {
		t.Fatalf("ResolveForTitle: %v", err)
	}
	if !got.CanSelfProvision || !got.CanStartInitialSync || !got.ShowProgression {
		t.Errorf("défauts absent→true non réappliqués : self=%v initial=%v progression=%v",
			got.CanSelfProvision, got.CanStartInitialSync, got.ShowProgression)
	}
	if got.SessionGapMinutes != 45 {
		t.Errorf("session_gap = %d, want 45 (overlay sur global sans le champ)", got.SessionGapMinutes)
	}
}
