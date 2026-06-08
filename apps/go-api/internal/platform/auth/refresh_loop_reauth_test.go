// Package auth — refresh_loop_reauth_test.go : hook reauth_required de la RefreshLoop (PR-B).
package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRefreshLoop_ReauthSignalAndClear(t *testing.T) {
	store := NewTokenStore(filepath.Join(t.TempDir(), "tracker.json"))
	mirror := NewMultiUserTokenStore(tempTokenDir(t))

	var notifies []string
	rl := NewRefreshLoop(store, nil).
		WithMultiUserMirror(mirror).
		WithReauthNotify(func(xuid, _ string) { notifies = append(notifies, xuid) })

	tokens := &StoredTokens{XSTSXUID: "111", XSTSGamertag: "Alice"}

	// 1ʳᵉ détection : marque + notifie une fois.
	rl.signalReauthRequired(context.Background(), tokens)
	if !mirror.IsReauthRequired("111") {
		t.Fatal("reauth_required attendu après signalReauthRequired")
	}
	if len(notifies) != 1 {
		t.Fatalf("notify = %d, want 1", len(notifies))
	}

	// 2ᵉ détection (déjà marqué) : pas de nouveau notify (transition unique).
	rl.signalReauthRequired(context.Background(), tokens)
	if len(notifies) != 1 {
		t.Errorf("notify ne doit pas se redéclencher, got %d", len(notifies))
	}

	// Clear (refresh OK) : flag effacé.
	rl.clearReauthRequired(context.Background(), tokens)
	if mirror.IsReauthRequired("111") {
		t.Error("reauth_required devrait être effacé après clearReauthRequired")
	}
}

// TestRefreshLoop_ReauthNoMirror_NoCrash : sans multiMirror, les helpers sont
// des no-op sûrs (le watcher peut tourner sans store multi-user configuré).
func TestRefreshLoop_ReauthNoMirror_NoCrash(t *testing.T) {
	store := NewTokenStore(filepath.Join(t.TempDir(), "tracker.json"))
	rl := NewRefreshLoop(store, nil) // pas de mirror
	tokens := &StoredTokens{XSTSXUID: "111"}
	rl.signalReauthRequired(context.Background(), tokens) // ne doit pas paniquer
	rl.clearReauthRequired(context.Background(), tokens)
}
