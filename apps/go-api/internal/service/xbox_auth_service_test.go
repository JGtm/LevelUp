// Package service — xbox_auth_service_test.go : tests XboxSSOLinkStrategy.
package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

func newXboxStore(t *testing.T) *userstore.Store {
	t.Helper()
	return userstore.NewStore(filepath.Join(t.TempDir(), "users.json"))
}

func TestXboxSSOLinkStrategy_FirstLogin_CreatesUser(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "xuid-spartan-42",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// User créé.
	created, err := users.GetByXUID("xuid-spartan-42")
	if err != nil {
		t.Fatalf("user pas créé : %v", err)
	}
	if created.Role != domain.RoleUser {
		t.Errorf("role = %q, want user", created.Role)
	}
	if created.Gamertag != "Spartan42" {
		t.Errorf("gamertag = %q, want Spartan42", created.Gamertag)
	}

	// Session wirée.
	if sess.Username == nil || *sess.Username != "spartan42" {
		t.Errorf("Username = %v, want spartan42 (slug)", sess.Username)
	}
	if sess.Role == nil || *sess.Role != "user" {
		t.Errorf("Role = %v, want user", sess.Role)
	}
	if sess.CurrentPlayerSlug == nil || *sess.CurrentPlayerSlug != "Spartan42" {
		t.Errorf("CurrentPlayerSlug = %v, want Spartan42 (gamertag original)", sess.CurrentPlayerSlug)
	}
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID != "xuid-spartan-42" {
		t.Errorf("LinkedHaloIdentity = %v", sess.LinkedHaloIdentity)
	}
}

// TestXboxSSOLinkStrategy_InstanceLocked_UnknownXUIDRefused : sous lockdown, un
// XUID inconnu ne peut pas créer de compte (login SSO refusé).
func TestXboxSSOLinkStrategy_InstanceLocked_UnknownXUIDRefused(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users).WithInstanceLock(func() bool { return true })

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Newcomer", XUID: "xuid-unknown-999"}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if !errors.Is(err, service.ErrInstanceLocked) {
		t.Fatalf("OnAuthSuccess sous lockdown (xuid inconnu) : err = %v, want ErrInstanceLocked", err)
	}
	if _, gErr := users.GetByXUID("xuid-unknown-999"); gErr == nil {
		t.Error("aucun user ne doit être créé sous lockdown")
	}
	if sess.Username != nil {
		t.Error("la session ne doit pas être wirée pour un xuid refusé")
	}
}

// TestXboxSSOLinkStrategy_InstanceLocked_KnownXUIDAllowed : sous lockdown, un XUID
// DÉJÀ connu se connecte normalement (seules les nouvelles identités sont bloquées).
func TestXboxSSOLinkStrategy_InstanceLocked_KnownXUIDAllowed(t *testing.T) {
	users := newXboxStore(t)
	_, _ = users.CreateFromXbox("Spartan42", "xuid-spartan-42") // identité déjà connue
	s := service.NewXboxSSOLinkStrategy(users).WithInstanceLock(func() bool { return true })

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: "xuid-spartan-42"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("user connu sous lockdown : err = %v, want nil", err)
	}
	if sess.Username == nil || *sess.Username != "spartan42" {
		t.Errorf("session devrait être wirée pour un user connu, got %v", sess.Username)
	}
}

func TestXboxSSOLinkStrategy_ExistingUser_AuthenticatesAndTouchesLastLogin(t *testing.T) {
	users := newXboxStore(t)
	// Pré-créer un user via CreateFromXbox.
	pre, _ := users.CreateFromXbox("Spartan42", "xuid-spartan-42")
	if pre.LastLoginAt != "" {
		t.Fatal("LastLoginAt devrait être vide juste après création")
	}

	s := service.NewXboxSSOLinkStrategy(users)
	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "xuid-spartan-42",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// LastLoginAt touché.
	updated, _ := users.GetByXUID("xuid-spartan-42")
	if updated.LastLoginAt == "" {
		t.Error("LastLoginAt devrait être touché pour un user existant")
	}

	// Session wirée pareil.
	if sess.Username == nil || *sess.Username != "spartan42" {
		t.Errorf("Username = %v, want spartan42", sess.Username)
	}
}

func TestXboxSSOLinkStrategy_MissingXUID_ReturnsError(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: ""}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if err == nil {
		t.Fatal("attendu erreur pour XUID vide, got nil")
	}
}

func TestXboxSSOLinkStrategy_MissingGamertag_ReturnsError(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "", XUID: "xuid-1"}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if err == nil {
		t.Fatal("attendu erreur pour gamertag vide, got nil")
	}
}

func TestXboxSSOLinkStrategy_WithTokenStore_PersistsRTATokens(t *testing.T) {
	users := newXboxStore(t)
	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))

	s := service.NewXboxSSOLinkStrategy(users).WithTokenStore(tokenStore)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag:             "Spartan42",
		XUID:                 "2535471234567890",
		MicrosoftAccessToken: "ms-access-token",
		MSALCacheJSON:        `{"AccessToken":{"...":"..."}}`,
		XSTSRTAToken:         "xsts-rta-token",
		XSTSRTAUserHash:      "rta-user-hash",
		XSTSRTAExpiresAt:     time.Now().Add(55 * time.Minute),
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// Tokens persistés dans MultiUserTokenStore.
	stored, err := tokenStore.Load("2535471234567890")
	if err != nil {
		t.Fatalf("tokens pas persistés : %v", err)
	}
	if stored.Gamertag != "Spartan42" {
		t.Errorf("Gamertag persisté = %q", stored.Gamertag)
	}
	if stored.XSTSToken != "xsts-rta-token" {
		t.Errorf("XSTSToken = %q, want xsts-rta-token", stored.XSTSToken)
	}
	if stored.XSTSUserHash != "rta-user-hash" {
		t.Errorf("XSTSUserHash = %q", stored.XSTSUserHash)
	}
	if stored.AccessToken != "ms-access-token" {
		t.Errorf("AccessToken = %q", stored.AccessToken)
	}
	if stored.MSALCacheJSON == "" {
		t.Error("MSALCacheJSON devrait être persisté")
	}
}

// TestXboxSSOLinkStrategy_PersistRTA_MergePreserveCredentials : Upsert remplace
// le fichier entier — un login SISU (RT brut, pas de cache MSAL) ne doit pas
// écraser le cache MSAL existant, et réciproquement (fix 2026-07-15).
func TestXboxSSOLinkStrategy_PersistRTA_MergePreserveCredentials(t *testing.T) {
	users := newXboxStore(t)
	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
	s := service.NewXboxSSOLinkStrategy(users).WithTokenStore(tokenStore)

	// État existant : credentials des deux providers déjà semés.
	if err := tokenStore.Upsert(&auth.UserTokens{
		XUID:              "2535471234567890",
		Gamertag:          "Spartan42",
		OAuthRefreshToken: "rt-ancien",
		MSALCacheJSON:     `{"cache":"ancien"}`,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Login SISU : RT brut frais, PAS de cache MSAL.
	attempt := &auth.Attempt{
		Gamertag:          "Spartan42",
		XUID:              "2535471234567890",
		XSTSRTAToken:      "xsts-rta-token",
		OAuthRefreshToken: "rt-sisu-frais",
	}
	if err := s.OnAuthSuccess(context.Background(), attempt, &domain.SessionData{}); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	stored, err := tokenStore.Load("2535471234567890")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.OAuthRefreshToken != "rt-sisu-frais" {
		t.Errorf("OAuthRefreshToken = %q, attendu le RT SISU frais", stored.OAuthRefreshToken)
	}
	if stored.MSALCacheJSON != `{"cache":"ancien"}` {
		t.Errorf("MSALCacheJSON = %q, le cache MSAL existant doit être préservé", stored.MSALCacheJSON)
	}

	// Login MSAL ensuite : cache frais, PAS de RT brut → le RT SISU est préservé.
	attempt2 := &auth.Attempt{
		Gamertag:      "Spartan42",
		XUID:          "2535471234567890",
		XSTSRTAToken:  "xsts-rta-token-2",
		MSALCacheJSON: `{"cache":"frais"}`,
	}
	if err := s.OnAuthSuccess(context.Background(), attempt2, &domain.SessionData{}); err != nil {
		t.Fatalf("OnAuthSuccess (2e): %v", err)
	}
	stored, err = tokenStore.Load("2535471234567890")
	if err != nil {
		t.Fatalf("Load (2e): %v", err)
	}
	if stored.OAuthRefreshToken != "rt-sisu-frais" {
		t.Errorf("OAuthRefreshToken = %q, le RT SISU doit être préservé", stored.OAuthRefreshToken)
	}
	if stored.MSALCacheJSON != `{"cache":"frais"}` {
		t.Errorf("MSALCacheJSON = %q, attendu le cache frais", stored.MSALCacheJSON)
	}
}

func TestXboxSSOLinkStrategy_WithTokenStore_SkipPersistanceIfNoXSTSRTA(t *testing.T) {
	users := newXboxStore(t)
	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))

	s := service.NewXboxSSOLinkStrategy(users).WithTokenStore(tokenStore)

	sess := &domain.SessionData{}
	// XSTSRTAToken vide (AcquireXSTSForRTA a échoué dans pollDeviceFlow).
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "2535471234567890",
		// XSTSRTAToken absent → persistance skip
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess (no RTA): %v", err)
	}

	// Aucun fichier persisté.
	if _, err := tokenStore.Load("2535471234567890"); err == nil {
		t.Error("aucun token ne devrait être persisté si XSTSRTAToken vide")
	}

	// User créé quand même.
	if _, err := users.GetByXUID("2535471234567890"); err != nil {
		t.Errorf("user devrait être créé même sans XSTS RTA : %v", err)
	}
}

func TestXboxSSOLinkStrategy_WithoutTokenStore_StillWorks(t *testing.T) {
	users := newXboxStore(t)
	// Pas de tokenStore injecté.
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag:     "Spartan42",
		XUID:         "2535471234567890",
		XSTSRTAToken: "xsts-rta-token", // présent mais ignoré (pas de tokenStore)
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// User créé, pas de panic sur persistance.
	if _, err := users.GetByXUID("2535471234567890"); err != nil {
		t.Errorf("user devrait être créé : %v", err)
	}
}

// mockDaemon capture les appels AddPlayer pour test.
type mockDaemon struct {
	running   bool
	addCalls  []domain.PlayerSummary
	failError error
}

func (m *mockDaemon) IsRunning() bool { return m.running }

func (m *mockDaemon) AddPlayer(ctx context.Context, p domain.PlayerSummary) error {
	m.addCalls = append(m.addCalls, p)
	return m.failError
}

func TestXboxSSOLinkStrategy_WithDaemonGetter_CallsAddPlayer(t *testing.T) {
	users := newXboxStore(t)
	daemon := &mockDaemon{running: true}
	getter := func() service.WatcherDaemon { return daemon }

	s := service.NewXboxSSOLinkStrategy(users).WithDaemonGetter(getter)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "2535471234567890",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	if len(daemon.addCalls) != 1 {
		t.Fatalf("AddPlayer calls = %d, want 1", len(daemon.addCalls))
	}
	got := daemon.addCalls[0]
	if got.XUID != "2535471234567890" {
		t.Errorf("AddPlayer XUID = %q, want 2535471234567890", got.XUID)
	}
	if got.Gamertag != "Spartan42" {
		t.Errorf("AddPlayer Gamertag = %q, want Spartan42", got.Gamertag)
	}
}

func TestXboxSSOLinkStrategy_WithDaemonGetter_SkipIfNotRunning(t *testing.T) {
	users := newXboxStore(t)
	daemon := &mockDaemon{running: false} // daemon créé mais pas démarré
	getter := func() service.WatcherDaemon { return daemon }

	s := service.NewXboxSSOLinkStrategy(users).WithDaemonGetter(getter)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: "2535471234567890"}

	_ = s.OnAuthSuccess(context.Background(), attempt, sess)

	if len(daemon.addCalls) != 0 {
		t.Errorf("AddPlayer ne devrait pas être appelé si daemon pas running, got %d calls", len(daemon.addCalls))
	}
}

func TestXboxSSOLinkStrategy_WithDaemonGetter_NilGetterIsNoop(t *testing.T) {
	users := newXboxStore(t)
	getter := func() service.WatcherDaemon { return nil } // getter retourne nil

	s := service.NewXboxSSOLinkStrategy(users).WithDaemonGetter(getter)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: "2535471234567890"}

	// Pas de panic, login OK.
	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}
}

func TestXboxSSOLinkStrategy_WithDaemonGetter_AddPlayerFailIsNonBlocking(t *testing.T) {
	users := newXboxStore(t)
	daemon := &mockDaemon{running: true, failError: errors.New("RTA disconnected")}
	getter := func() service.WatcherDaemon { return daemon }

	s := service.NewXboxSSOLinkStrategy(users).WithDaemonGetter(getter)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: "2535471234567890"}

	// Login doit réussir même si AddPlayer échoue.
	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Errorf("login devrait réussir même si AddPlayer échoue, got err: %v", err)
	}
	// User créé.
	if _, err := users.GetByXUID("2535471234567890"); err != nil {
		t.Errorf("user devrait être créé : %v", err)
	}
}

// Cleanup 2026-05-26 : les tests AddUserClient PreferOver et FallbackOnFail
// ont été supprimés avec la méthode (RTA legacy retiré). AddPlayer couvre
// tous les cas via REST poll.
func TestXboxSSOLinkStrategy_WithDaemonGetter_AddPlayerEvenWithTokenStore(t *testing.T) {
	users := newXboxStore(t)
	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
	daemon := &mockDaemon{running: true}
	getter := func() service.WatcherDaemon { return daemon }

	s := service.NewXboxSSOLinkStrategy(users).
		WithTokenStore(tokenStore).
		WithDaemonGetter(getter)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag:         "Spartan42",
		XUID:             "2535471234567890",
		XSTSRTAToken:     "xsts-rta-token",
		XSTSRTAUserHash:  "rta-user-hash",
		XSTSRTAExpiresAt: time.Now().Add(55 * time.Minute),
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// AddPlayer est appelé même avec tokenStore (REST poll prend le relais).
	if len(daemon.addCalls) != 1 {
		t.Errorf("AddPlayer calls = %d, want 1", len(daemon.addCalls))
	}
}

func TestXboxSSOLinkStrategy_CollisionWithPasswordUser_FallbackXbox(t *testing.T) {
	users := newXboxStore(t)
	// Pré-créer un user password avec slug "alice".
	_, _ = users.Create("alice", "Pa55w0rd!", domain.RoleUser)

	s := service.NewXboxSSOLinkStrategy(users)
	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Alice",
		XUID:     "xuid-alice-xbox",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess collision: %v", err)
	}

	// User xbox créé avec suffixe.
	created, err := users.GetByXUID("xuid-alice-xbox")
	if err != nil {
		t.Fatalf("user xbox pas créé : %v", err)
	}
	if created.Username != "alice_xbox" {
		t.Errorf("username = %q, want alice_xbox (fallback)", created.Username)
	}

	// Session pointe vers le user xbox, pas le password.
	if sess.Username == nil || *sess.Username != "alice_xbox" {
		t.Errorf("session Username = %v, want alice_xbox", sess.Username)
	}
}

// ---------------------------------------------------------------------------
// Flow "rejoindre un groupe" : invitation portée par la session → bypass du
// verrou d'instance + ajout au groupe + consommation du code.
// ---------------------------------------------------------------------------

// newGroupInviteRig monte une strategy avec invite + group stores câblés.
func newGroupInviteRig(t *testing.T, locked bool) (*service.XboxSSOLinkStrategy, *userstore.Store, *userstore.InviteStore, *groupstore.GroupStore) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"))
	groups := groupstore.NewGroupStore(filepath.Join(dir, "groups.json"))
	s := service.NewXboxSSOLinkStrategy(users).
		WithInstanceLock(func() bool { return locked }).
		WithInviteStore(invites).
		WithGroupStore(groups)
	return s, users, invites, groups
}

// Invitation valide + XUID inconnu + instance verrouillée → bypass : user créé,
// ajouté au groupe ciblé, code consommé, PendingInviteCode vidé.
func TestXboxSSOLinkStrategy_InviteJoinsGroup_NewUser_BypassLock(t *testing.T) {
	s, users, invites, groups := newGroupInviteRig(t, true)
	g, _ := groups.Create("Fam", "owner-x", "Owner")
	inv, _ := invites.Generate("Owner", 7, g.ID)

	sess := &domain.SessionData{PendingInviteCode: inv.Code}
	attempt := &auth.Attempt{XUID: "newcomer-x", Gamertag: "Newbie"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}
	if _, err := users.GetByXUID("newcomer-x"); err != nil {
		t.Fatalf("user devrait être créé malgré le verrou : %v", err)
	}
	if got, _ := groups.Get(g.ID); !got.HasMember("newcomer-x") {
		t.Fatalf("newcomer devrait être membre du groupe : %+v", got.Members)
	}
	if gotInv, _ := invites.Get(inv.Code); !gotInv.IsUsed() {
		t.Fatal("l'invitation devrait être consommée")
	}
	if sess.PendingInviteCode != "" {
		t.Fatal("PendingInviteCode devrait être vidé après consommation")
	}
}

// Invitation introuvable + verrou → traitée comme absente → refus.
func TestXboxSSOLinkStrategy_InvalidInvite_Locked_Rejected(t *testing.T) {
	s, _, _, _ := newGroupInviteRig(t, true)
	sess := &domain.SessionData{PendingInviteCode: "NOPE"}
	attempt := &auth.Attempt{XUID: "x", Gamertag: "GT"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); !errors.Is(err, service.ErrInstanceLocked) {
		t.Fatalf("attendu ErrInstanceLocked (invite invalide ignorée), got %v", err)
	}
}

// User existant + invitation valide → ajouté au groupe (hors verrou).
func TestXboxSSOLinkStrategy_ExistingUser_InviteAddsToGroup(t *testing.T) {
	s, users, invites, groups := newGroupInviteRig(t, false)
	if _, err := users.CreateFromXbox("Existing", "exist-x"); err != nil {
		t.Fatalf("CreateFromXbox: %v", err)
	}
	g, _ := groups.Create("Fam", "owner-x", "Owner")
	inv, _ := invites.Generate("Owner", 7, g.ID)

	sess := &domain.SessionData{PendingInviteCode: inv.Code}
	attempt := &auth.Attempt{XUID: "exist-x", Gamertag: "Existing"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}
	if got, _ := groups.Get(g.ID); !got.HasMember("exist-x") {
		t.Fatalf("user existant devrait être ajouté au groupe : %+v", got.Members)
	}
	if gotInv, _ := invites.Get(inv.Code); !gotInv.IsUsed() {
		t.Fatal("l'invitation devrait être consommée")
	}
}

// Invitation legacy (sans groupe) + verrou → pas de groupe → bypass refusé.
func TestXboxSSOLinkStrategy_LegacyInviteNoGroup_Locked_Rejected(t *testing.T) {
	s, _, invites, _ := newGroupInviteRig(t, true)
	inv, _ := invites.Generate("admin", 7, "") // GroupID vide
	sess := &domain.SessionData{PendingInviteCode: inv.Code}
	attempt := &auth.Attempt{XUID: "x", Gamertag: "GT"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); !errors.Is(err, service.ErrInstanceLocked) {
		t.Fatalf("attendu ErrInstanceLocked (invite sans groupe), got %v", err)
	}
}
