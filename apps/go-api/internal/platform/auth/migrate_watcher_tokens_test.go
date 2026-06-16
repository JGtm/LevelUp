package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTokenFile(t *testing.T, dir, xuid, gamertag string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	data, _ := json.Marshal(UserTokens{XUID: xuid, Gamertag: gamertag, OAuthRefreshToken: "rt-" + xuid})
	if err := os.WriteFile(filepath.Join(dir, xuid+".json"), data, 0o600); err != nil {
		t.Fatalf("write %s: %v", xuid, err)
	}
}

// TestMigrateWatcherTokens_CopiesLegacy : les tokens legacy sont recopiés vers le
// nouveau dossier, contenu préservé, ET le legacy reste intact (filet de retour).
func TestMigrateWatcherTokens_CopiesLegacy(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy")
	newDir := filepath.Join(root, "new")
	writeTokenFile(t, legacy, "111", "Madina97294")
	writeTokenFile(t, legacy, "222", "Chocoboflor")

	n, err := MigrateWatcherTokens(legacy, newDir)
	if err != nil {
		t.Fatalf("MigrateWatcherTokens: %v", err)
	}
	if n != 2 {
		t.Errorf("copied = %d, want 2", n)
	}

	// Contenu présent + correct dans le nouveau dossier.
	store := NewMultiUserTokenStore(newDir)
	got, err := store.Load("111")
	if err != nil {
		t.Fatalf("Load(111) après migration: %v", err)
	}
	if got.Gamertag != "Madina97294" || got.OAuthRefreshToken != "rt-111" {
		t.Errorf("token 111 altéré: %+v", got)
	}

	// Legacy PRÉSERVÉ (non destructif).
	if _, err := os.Stat(filepath.Join(legacy, "111.json")); err != nil {
		t.Errorf("legacy 111.json devrait être préservé: %v", err)
	}
}

// TestMigrateWatcherTokens_Idempotent_NoOverwrite : 2e run = 0 copie ; et un token déjà
// présent dans le nouveau dossier (potentiellement plus récent) n'est JAMAIS écrasé.
func TestMigrateWatcherTokens_Idempotent_NoOverwrite(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy")
	newDir := filepath.Join(root, "new")
	writeTokenFile(t, legacy, "111", "OldName")

	// Pré-existe dans new avec un RT plus récent (simule un refresh post-migration).
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fresh, _ := json.Marshal(UserTokens{XUID: "111", Gamertag: "FreshName", OAuthRefreshToken: "rt-fresh"})
	if err := os.WriteFile(filepath.Join(newDir, "111.json"), fresh, 0o600); err != nil {
		t.Fatal(err)
	}

	n, err := MigrateWatcherTokens(legacy, newDir)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 0 {
		t.Errorf("copied = %d, want 0 (existant non écrasé)", n)
	}

	store := NewMultiUserTokenStore(newDir)
	got, _ := store.Load("111")
	if got.OAuthRefreshToken != "rt-fresh" {
		t.Errorf("token frais écrasé par le legacy périmé : rt=%q", got.OAuthRefreshToken)
	}

	// 2e run idempotent.
	n2, _ := MigrateWatcherTokens(legacy, newDir)
	if n2 != 0 {
		t.Errorf("2e run: copied = %d, want 0", n2)
	}
}

// TestMigrateWatcherTokens_NoLegacy_NoOp : install neuve (pas de legacy) → no-op silencieux.
func TestMigrateWatcherTokens_NoLegacy_NoOp(t *testing.T) {
	root := t.TempDir()
	n, err := MigrateWatcherTokens(filepath.Join(root, "absent"), filepath.Join(root, "new"))
	if err != nil {
		t.Fatalf("migrate no-legacy devrait être no-op: %v", err)
	}
	if n != 0 {
		t.Errorf("copied = %d, want 0", n)
	}
}

// TestMigrateWatcherTokens_SameDir_NoOp : legacy == new → no-op (pas de cutover).
func TestMigrateWatcherTokens_SameDir_NoOp(t *testing.T) {
	dir := t.TempDir()
	writeTokenFile(t, dir, "111", "X")
	n, err := MigrateWatcherTokens(dir, dir)
	if err != nil || n != 0 {
		t.Errorf("same dir: n=%d err=%v, want 0/nil", n, err)
	}
}
