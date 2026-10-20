// Package handlers_test — setup_identity_pin_test.go : un non-admin ne crée QUE son
// propre profil, quel que soit profile_mode (registre sécurité C4-5).
//
// Avant : guardLinkedXboxIdentity ne s'appliquait qu'en mode "xbox" ; un compte
// envoyait profile_mode "manual" avec le gamertag d'autrui et SON xuid, et devenait
// propriétaire du profil d'autrui (ProfileService.CreatePlayer réécrivait le xuid).
package handlers_test

import (
	"bytes"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
)

// openEnforcedSetupRig : instance OUVERTE (non verrouillée, auto-provisioning
// autorisé) en propriété appliquée (auth xbox) — précondition de C4-5.
func openEnforcedSetupRig(t *testing.T, svc *mockDirectory, locked bool) (*chi.Mux, *userstore.Store, *session_platform.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{
		RepoRoot:        dir,
		DBProfilesPath:  filepath.Join(dir, "db_profiles.json"),
		SessionDir:      filepath.Join(dir, "sessions"),
		AppSettingsPath: filepath.Join(dir, "app_settings.json"),
		AuthMode:        "xbox",
	}
	sessions := session_platform.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32-bytesXXXXXXXXXX")
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	appCfg, _ := settingsStore.Load()
	appCfg.CanSelfProvision = true
	_ = settingsStore.Save(appCfg)
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	h := handlers.NewSetupHandler(cfg, sessions, settingsStore, jobs.NewStore(filepath.Join(dir, "jobs.json"))).
		WithDirectory(svc).
		WithInstanceLock(func() bool { return locked }).
		WithProvisionGrant(users, users)
	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessions, middleware.SecureCookiePolicy{}))
	h.Mount(r)
	return r, users, sessions
}

func TestSetupHandler_NonAdminPinnedToOwnIdentityInEveryMode(t *testing.T) {
	for _, mode := range []string{"", "xbox", "azure_manual", "manual"} {
		t.Run("mode="+mode, func(t *testing.T) {
			svc := &mockDirectory{playerKey: "Victim"}
			r, users, sessions := openEnforcedSetupRig(t, svc, false)
			cookie := guestSession(t, users, sessions, "Mallory", "222", "")
			w := postCreatePlayer(t, r, cookie,
				`{"gamertag": "Victim", "xuid": "222", "profile_mode": "`+mode+`"}`)
			if w.Code != http.StatusConflict {
				t.Fatalf("statut %d, attendu 409 (corps = %s)", w.Code, w.Body.String())
			}
			if !bytes.Contains(w.Body.Bytes(), []byte("identity_mismatch")) {
				t.Errorf("code attendu identity_mismatch, corps = %s", w.Body.String())
			}
			if svc.lastReq.Gamertag != "" {
				t.Fatalf("profil créé pour %q malgré le refus", svc.lastReq.Gamertag)
			}
		})
	}
}

// TestSetupHandler_NonAdminOwnProfileStillCreated : son propre profil reste créable,
// en mode manuel compris.
func TestSetupHandler_NonAdminOwnProfileStillCreated(t *testing.T) {
	svc := &mockDirectory{playerKey: "Mallory"}
	r, users, sessions := openEnforcedSetupRig(t, svc, false)
	cookie := guestSession(t, users, sessions, "Mallory", "222", "")
	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Mallory", "xuid": "222", "profile_mode": "manual"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("statut %d, attendu 201 (corps = %s)", w.Code, w.Body.String())
	}
}

// TestSetupHandler_LockedInstanceRefusesForeignProfileBeforeIdentity : état de la
// production (instance_locked=true) — sans droit de provisioning, la création est
// refusée par le verrou avant même la vérification d'identité.
func TestSetupHandler_LockedInstanceRefusesForeignProfileBeforeIdentity(t *testing.T) {
	svc := &mockDirectory{playerKey: "Victim"}
	r, users, sessions := openEnforcedSetupRig(t, svc, true)
	cookie := guestSession(t, users, sessions, "Mallory", "222", "")
	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Victim", "xuid": "222", "profile_mode": "manual"}`)
	if w.Code != http.StatusForbidden || !bytes.Contains(w.Body.Bytes(), []byte("instance_locked")) {
		t.Fatalf("statut %d, attendu 403 instance_locked (corps = %s)", w.Code, w.Body.String())
	}
}

// TestSetupHandler_XUIDConflictIs409 : l'annuaire refuse de réécrire le xuid d'un
// profil existant (non-admin) → 409 profile_xuid_conflict, et la demande porte
// ActorIsAdmin=false.
func TestSetupHandler_XUIDConflictIs409(t *testing.T) {
	svc := &mockDirectory{err: domain.ErrProfileXUIDConflict}
	r, users, sessions := openEnforcedSetupRig(t, svc, false)
	cookie := guestSession(t, users, sessions, "Mallory", "222", "")
	w := postCreatePlayer(t, r, cookie, `{"gamertag": "Mallory", "xuid": "222"}`)
	if w.Code != http.StatusConflict || !bytes.Contains(w.Body.Bytes(), []byte("profile_xuid_conflict")) {
		t.Fatalf("statut %d, attendu 409 profile_xuid_conflict (corps = %s)", w.Code, w.Body.String())
	}
	if svc.lastReq.ActorIsAdmin {
		t.Fatal("ActorIsAdmin=true pour un compte ordinaire")
	}
}
