package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestRefreshLoop_MultiUserMirror_XSTS verifie que le refresh XSTS du tracker
// (legacy TokenStore) est aussi mirroré dans MultiUserTokenStore[XSTSXUID].
// PR 2.5b — pavement migration read-path future.
func TestRefreshLoop_MultiUserMirror_XSTS(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "watcher_tokens.json")
	multiDir := filepath.Join(dir, "watcher_tokens")

	store := NewTokenStore(storePath)
	multi := NewMultiUserTokenStore(multiDir)

	// Seed legacy store avec un tracker initial (XSTS valide + xuid).
	initial := &StoredTokens{
		AccessToken:    "access_initial",
		OAuthExpiresAt: time.Now().Add(1 * time.Hour),
		XSTSToken:      "xsts_initial",
		XSTSUserHash:   "uhs_initial",
		XSTSGamertag:   "Madina97294",
		XSTSXUID:       "2533274858283686",
		XSTSExpiresAt:  time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(initial); err != nil {
		t.Fatalf("Save legacy: %v", err)
	}

	// Construire un RefreshLoop avec mirror + acquireXSTSFn mockée
	// (retourne un nouveau XSTS sans réseau).
	mockXSTS := &XSTSResult{
		Token:    "xsts_refreshed",
		UserHash: "uhs_refreshed",
		Gamertag: "Madina97294",
		XUID:     "2533274858283686",
		NotAfter: time.Now().Add(2 * time.Hour),
	}
	r := NewRefreshLoop(store, nil).WithMultiUserMirror(multi)
	r.acquireXSTSFn = func(_ context.Context, _ string) (*XSTSResult, error) {
		return mockXSTS, nil
	}

	// Charger les tokens (pour passer à refreshXSTS qui lit AccessToken).
	tokens, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	r.refreshXSTS(context.Background(), tokens)

	// Vérifier : multi-user store a la nouvelle row pour le xuid.
	mirrored, err := multi.Load("2533274858283686")
	if err != nil {
		t.Fatalf("multi.Load: %v", err)
	}
	if mirrored.XSTSToken != "xsts_refreshed" {
		t.Errorf("mirrored XSTSToken = %q, want xsts_refreshed", mirrored.XSTSToken)
	}
	if mirrored.XSTSUserHash != "uhs_refreshed" {
		t.Errorf("mirrored XSTSUserHash = %q, want uhs_refreshed", mirrored.XSTSUserHash)
	}
	if mirrored.Gamertag != "Madina97294" {
		t.Errorf("mirrored Gamertag = %q, want Madina97294", mirrored.Gamertag)
	}
}

// TestRefreshLoop_MultiUserMirror_PreservesRefreshToken : le mirror NE doit PAS
// faire disparaître le refresh_token déjà dans le multi-user store
// (ex. RT e1cb35ab frais semé par le callback SSO). Régression incident 2026-06-13 :
// Upsert d'un UserTokens neuf (sans RT) écrasait le RT à vide → migration boot le
// re-remplissait avec le RT env périmé (39829f7a) → AADSTS70000 en boucle.
func TestRefreshLoop_MultiUserMirror_PreservesRefreshToken(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "watcher_tokens.json"))
	multi := NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))

	const xuid = "2533274823110022"
	// Pré-seed le multi-user store comme le ferait le callback SSO : RT + MSAL frais.
	if err := multi.Upsert(&UserTokens{
		XUID:              xuid,
		Gamertag:          "JGtm",
		OAuthRefreshToken: "rt_frais_e1cb35ab",
	}); err != nil {
		t.Fatalf("seed multi: %v", err)
	}

	// Tracker legacy avec un RT DIFFÉRENT (périmé) + un XSTS à mirrorer.
	initial := &StoredTokens{
		AccessToken:    "access_tracker",
		OAuthExpiresAt: time.Now().Add(1 * time.Hour),
		XSTSToken:      "xsts_old",
		XSTSUserHash:   "uhs_old",
		XSTSGamertag:   "JGtm",
		XSTSXUID:       xuid,
		XSTSExpiresAt:  time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(initial); err != nil {
		t.Fatalf("Save legacy: %v", err)
	}

	r := NewRefreshLoop(store, nil).WithMultiUserMirror(multi)
	r.acquireXSTSFn = func(_ context.Context, _ string) (*XSTSResult, error) {
		return &XSTSResult{
			Token: "xsts_refreshed", UserHash: "uhs_refreshed",
			Gamertag: "JGtm", XUID: xuid, NotAfter: time.Now().Add(2 * time.Hour),
		}, nil
	}

	tokens, _ := store.Load()
	r.refreshXSTS(context.Background(), tokens) // déclenche le mirror

	got, err := multi.Load(xuid)
	if err != nil {
		t.Fatalf("multi.Load: %v", err)
	}
	// RT + MSAL du store PRÉSERVÉS (pas écrasés par le mirror).
	if got.OAuthRefreshToken != "rt_frais_e1cb35ab" {
		t.Errorf("RT écrasé par le mirror = %q, want rt_frais_e1cb35ab (préservé)", got.OAuthRefreshToken)
	}
	// Champs XSTS bien mis à jour par le mirror.
	if got.XSTSToken != "xsts_refreshed" {
		t.Errorf("XSTSToken = %q, want xsts_refreshed (mis à jour)", got.XSTSToken)
	}
}

// TestRefreshLoop_MultiUserMirror_NilMirror_NoOp verifie que sans mirror
// configuré, refresh marche normalement sans toucher multi-user store.
func TestRefreshLoop_MultiUserMirror_NilMirror_NoOp(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "watcher_tokens.json"))

	initial := &StoredTokens{
		AccessToken:    "access",
		OAuthExpiresAt: time.Now().Add(1 * time.Hour),
		XSTSToken:      "xsts",
		XSTSUserHash:   "uhs",
		XSTSGamertag:   "Test",
		XSTSXUID:       "1234",
		XSTSExpiresAt:  time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(initial); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// RefreshLoop SANS WithMultiUserMirror.
	r := NewRefreshLoop(store, nil)
	r.acquireXSTSFn = func(_ context.Context, _ string) (*XSTSResult, error) {
		return &XSTSResult{
			Token: "xsts_v2", UserHash: "uhs", Gamertag: "Test", XUID: "1234",
			NotAfter: time.Now().Add(2 * time.Hour),
		}, nil
	}

	tokens, _ := store.Load()
	r.refreshXSTS(context.Background(), tokens)

	// Le legacy store doit être updaté ; multi-user dir n'existe même pas
	// (jamais touché par le RefreshLoop sans mirror).
	updated, _ := store.Load()
	if updated.XSTSToken != "xsts_v2" {
		t.Errorf("legacy XSTS not updated: got %q", updated.XSTSToken)
	}
}

// TestRefreshLoop_MultiUserMirror_EmptyXUID_Skip verifie que si le tracker
// n'a pas de XSTSXUID renseigné, le mirror skip silencieusement (pas de
// fail sur Upsert avec xuid vide).
func TestRefreshLoop_MultiUserMirror_EmptyXUID_Skip(t *testing.T) {
	dir := t.TempDir()
	store := NewTokenStore(filepath.Join(dir, "watcher_tokens.json"))
	multi := NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))

	initial := &StoredTokens{
		AccessToken:    "access",
		OAuthExpiresAt: time.Now().Add(1 * time.Hour),
		XSTSToken:      "xsts",
		XSTSUserHash:   "uhs",
		XSTSGamertag:   "Test",
		// XSTSXUID intentionnellement vide
		XSTSExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(initial); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r := NewRefreshLoop(store, nil).WithMultiUserMirror(multi)
	r.acquireXSTSFn = func(_ context.Context, _ string) (*XSTSResult, error) {
		return &XSTSResult{
			Token: "xsts_v2", UserHash: "uhs", Gamertag: "Test",
			// XUID vide aussi dans le result mock pour réalisme
			NotAfter: time.Now().Add(2 * time.Hour),
		}, nil
	}

	tokens, _ := store.Load()
	// Ne doit PAS paniquer.
	r.refreshXSTS(context.Background(), tokens)

	// Le multi-user dir ne doit avoir aucun fichier (xuid vide rejected).
	all, _ := multi.LoadAll()
	if len(all) != 0 {
		t.Errorf("multi.LoadAll = %d entries, want 0 (xuid vide doit skip)", len(all))
	}
}
