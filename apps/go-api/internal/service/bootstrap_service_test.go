package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// mockBootRepo is a minimal mock for BootstrapRepository used in tests.
type mockBootRepo struct {
	matchCount int
}

func (m *mockBootRepo) GetMatchCount(context.Context) (int, error) { return m.matchCount, nil }
func (m *mockBootRepo) GetDBVersion(context.Context) (string, error) {
	return "test", nil
}
func (m *mockBootRepo) GetPlayerCount(context.Context) (int, error) { return 0, nil }
func (m *mockBootRepo) GetLastSyncAt(context.Context) (*time.Time, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// excludeAuthOnly
// ---------------------------------------------------------------------------

func TestExcludeAuthOnly(t *testing.T) {
	in := []domain.PlayerSummary{
		{Gamertag: "Real1"},
		{Gamertag: "Token1", AuthOnly: true},
		{Gamertag: "Real2"},
		{Gamertag: "Token2", AuthOnly: true},
	}
	out := excludeAuthOnly(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 visible players, got %d", len(out))
	}
	for _, p := range out {
		if p.AuthOnly {
			t.Errorf("auth-only player %q leaked into visible list", p.Gamertag)
		}
	}
}

// ---------------------------------------------------------------------------
// getBoolSetting / getStringSetting
// ---------------------------------------------------------------------------

func TestGetBoolSetting_Found(t *testing.T) {
	s := map[string]interface{}{"k": true}
	if !getBoolSetting(s, "k", false) {
		t.Error("expected true")
	}
}

func TestGetBoolSetting_Missing(t *testing.T) {
	s := map[string]interface{}{}
	if getBoolSetting(s, "k", true) != true {
		t.Error("expected default true")
	}
}

func TestGetBoolSetting_WrongType(t *testing.T) {
	s := map[string]interface{}{"k": "notbool"}
	if getBoolSetting(s, "k", true) != true {
		t.Error("expected fallback to default when wrong type")
	}
}

func TestGetBoolSetting_NilMap(t *testing.T) {
	if getBoolSetting(nil, "k", false) {
		t.Error("expected false for nil map")
	}
}

func TestGetStringSetting_Found(t *testing.T) {
	s := map[string]interface{}{"lang": "en"}
	if getStringSetting(s, "lang", "fr") != "en" {
		t.Error("expected en")
	}
}

func TestGetStringSetting_Missing(t *testing.T) {
	s := map[string]interface{}{}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default fr")
	}
}

func TestGetStringSetting_EmptyValue(t *testing.T) {
	s := map[string]interface{}{"lang": ""}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default for empty string")
	}
}

func TestGetStringSetting_WrongType(t *testing.T) {
	s := map[string]interface{}{"lang": 42}
	if getStringSetting(s, "lang", "fr") != "fr" {
		t.Error("expected default for int type")
	}
}

// ---------------------------------------------------------------------------
// resolveSetupState
// ---------------------------------------------------------------------------

func TestResolveSetupState_NoPlayers(t *testing.T) {
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 0})
	got := svc.resolveSetupState(context.Background(), nil, "halo_infinite", nil)
	if got != "no_halo_link" {
		t.Errorf("expected no_halo_link, got %s", got)
	}
}

func TestResolveSetupState_HaloLinkedNoProfile(t *testing.T) {
	// SSO terminé (identité liée en session) mais aucun profil local : doit
	// router vers StepPlayer, pas reboucler sur le Device Code Flow.
	sess := &domain.SessionData{
		AuthReady:          true,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 0})
	got := svc.resolveSetupState(context.Background(), sess, "halo_infinite", nil)
	if got != "halo_linked_no_profile" {
		t.Errorf("expected halo_linked_no_profile, got %s", got)
	}
}

func TestResolveSetupState_WithPlayers(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 0})
	got := svc.resolveSetupState(context.Background(), nil, "halo_infinite", players)
	if got != "profile_ready_no_sync" {
		t.Errorf("expected profile_ready_no_sync, got %s", got)
	}
}

func TestResolveSetupState_WithMatchesReady(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 42})
	got := svc.resolveSetupState(context.Background(), nil, "halo_infinite", players)
	if got != "ready" {
		t.Errorf("expected ready, got %s", got)
	}
}

// Le résolveur title-aware doit être préféré au bootRepo (figé Infinite) et
// recevoir le titre COURANT — sinon un switch Halo 5 compterait le mauvais shared.
func TestResolveSetupState_UsesTitleAwareResolver(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	var gotTitle string
	// bootRepo renvoie 0 (Infinite vide) ; le résolveur renvoie 3 pour halo_5.
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 0}).
		WithMatchCountResolver(func(_ context.Context, titleSlug string) (int, error) {
			gotTitle = titleSlug
			return 3, nil
		})
	got := svc.resolveSetupState(context.Background(), nil, "halo_5", players)
	if got != "ready" {
		t.Errorf("expected ready (résolveur title-aware), got %s", got)
	}
	if gotTitle != "halo_5" {
		t.Errorf("expected resolver appelé avec halo_5, got %s", gotTitle)
	}
}

// Un échec du décompte (provider contendu, erreur) dégrade proprement en
// profile_ready_no_sync sans jamais faire échouer le bootstrap.
func TestResolveSetupState_ResolverErrorDegrades(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 99}).
		WithMatchCountResolver(func(_ context.Context, _ string) (int, error) {
			return 0, context.DeadlineExceeded
		})
	got := svc.resolveSetupState(context.Background(), nil, "halo_infinite", players)
	if got != "profile_ready_no_sync" {
		t.Errorf("expected profile_ready_no_sync (dégradation), got %s", got)
	}
}

// Raison d'être de la feature : un décompte qui DÉPASSE le budget (provider
// contendu par un sync) ne doit JAMAIS faire pendre le bootstrap — il dégrade en
// profile_ready_no_sync sous ~setupCountBudget. Couvre la branche timeout
// (case <-cctx.Done()) de matchCountForSetup.
func TestResolveSetupState_CountTimeoutDegrades(t *testing.T) {
	players := []domain.PlayerSummary{{Gamertag: "GT"}}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{matchCount: 5}).
		WithMatchCountResolver(func(ctx context.Context, _ string) (int, error) {
			<-ctx.Done() // bloque jusqu'à l'expiration du budget (respecte le ctx)
			return 0, ctx.Err()
		})

	start := time.Now()
	got := svc.resolveSetupState(context.Background(), nil, "halo_infinite", players)
	elapsed := time.Since(start)

	if got != "profile_ready_no_sync" {
		t.Errorf("un décompte au-delà du budget doit dégrader en profile_ready_no_sync, got %s", got)
	}
	// Le budget (setupCountBudget=2s) borne l'attente — pas de blocage indéfini.
	if elapsed > 4*time.Second {
		t.Errorf("resolveSetupState doit rendre la main sous ~budget, pris %v", elapsed)
	}
}

// ---------------------------------------------------------------------------
// Build — wiring complet post-SSO (simulation onboarding sans login Microsoft)
// ---------------------------------------------------------------------------

func TestBuild_HaloLinkedNoProfile(t *testing.T) {
	// État post-SSO : le Device Code Flow a posé une identité Halo en session
	// (sess.LinkedHaloIdentity) mais aucun profil local n'est encore créé.
	// Build() doit router vers StepPlayer via setup_state=halo_linked_no_profile
	// au lieu de reboucler sur StepDeviceCode (no_halo_link).
	cfg := &config.AppConfig{} // 0 joueur : DBProfilesPath vide → LoadPlayers = []
	svc := NewBootstrapService(cfg, &mockBootRepo{matchCount: 0})
	sess := &domain.SessionData{
		AuthReady:          true,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}

	resp, err := svc.Build(context.Background(), sess)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.SetupState != "halo_linked_no_profile" {
		t.Errorf("SetupState = %q, want halo_linked_no_profile", resp.SetupState)
	}
	if !resp.SetupRequired {
		t.Error("SetupRequired devrait être true (0 joueur)")
	}
	if resp.LinkedHaloIdentity == nil || resp.LinkedHaloIdentity.Gamertag != "GT" {
		t.Error("LinkedHaloIdentity devrait être propagé dans la réponse bootstrap")
	}
}

// TestBuild_ReauthRequired_NoOwnIdentity_False : sans identité propre en session
// (ni username ni identité Halo liée), le flag reauth_required reste false même
// si le checker retournerait true (garde PR-B / scope compte).
func TestBuild_ReauthRequired_NoOwnIdentity_False(t *testing.T) {
	cfg := &config.AppConfig{}
	svc := NewBootstrapService(cfg, &mockBootRepo{}).
		WithReauthChecker(func(string) bool { return true })

	resp, err := svc.Build(context.Background(), &domain.SessionData{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.ReauthRequired {
		t.Error("reauth_required doit être false sans identité propre (garde)")
	}
}

// TestBuild_ReauthRequired_ScopedToOwnAccount : un admin dont le RT est sain ne
// doit JAMAIS voir le bandeau, même quand d'autres joueurs (qu'il peut consulter)
// ont un refresh_token mort. Le flag est scopé à SON propre xuid, pas au joueur
// courant affiché.
func TestBuild_ReauthRequired_ScopedToOwnAccount(t *testing.T) {
	const ownXUID = "2533274823110022" // JGtm, RT sain
	admin := &domain.User{Username: "jgtm_xbox", XUID: ownXUID, Role: domain.RoleAdmin}
	lookup := fakeBootstrapLookup{
		byName: map[string]*domain.User{"jgtm_xbox": admin},
		byXUID: map[string]*domain.User{ownXUID: admin},
	}
	// Checker "mort pour tout le monde SAUF le compte propre" : simule
	// Chocoboflor/Madina/XxDaemon morts, JGtm vivant.
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).
		WithUserLookup(lookup).
		WithReauthChecker(func(xuid string) bool { return xuid != ownXUID })

	username := "jgtm_xbox"
	resp, err := svc.Build(context.Background(), &domain.SessionData{Username: &username})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.ReauthRequired {
		t.Error("admin au RT sain : reauth_required doit être false (scope compte, pas joueur courant)")
	}
}

// TestBuild_ReauthRequired_OwnAccountDead_True : un user dont SON propre RT est
// mort voit bien le bandeau (il peut, lui, se reconnecter via le SSO Xbox).
func TestBuild_ReauthRequired_OwnAccountDead_True(t *testing.T) {
	const ownXUID = "2535469190789936" // Chocoboflor, RT mort
	user := &domain.User{Username: "chocoboflor", XUID: ownXUID, Role: domain.RoleUser}
	lookup := fakeBootstrapLookup{
		byName: map[string]*domain.User{"chocoboflor": user},
		byXUID: map[string]*domain.User{ownXUID: user},
	}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).
		WithUserLookup(lookup).
		WithReauthChecker(func(xuid string) bool { return xuid == ownXUID })

	username := "chocoboflor"
	resp, err := svc.Build(context.Background(), &domain.SessionData{Username: &username})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !resp.ReauthRequired {
		t.Error("user au RT mort : reauth_required doit être true (il peut se reconnecter)")
	}
}

// TestCurrentUserHasPassword : has_password reflète le PasswordHash du user
// courant (opt-in PR-C). False si pas de session/username/lookup.
func TestCurrentUserHasPassword(t *testing.T) {
	lookup := fakeBootstrapLookup{byName: map[string]*domain.User{
		"alice": {Username: "alice", PasswordHash: "$2a$12$hashhashhashhashhashhash"},
		"bob":   {Username: "bob"}, // pas de mot de passe (compte SSO)
	}}
	svc := NewBootstrapService(&config.AppConfig{}, &mockBootRepo{}).WithUserLookup(lookup)

	alice, bob := "alice", "bob"
	if !svc.currentUserHasPassword(&domain.SessionData{Username: &alice}) {
		t.Error("alice a un mot de passe → true attendu")
	}
	if svc.currentUserHasPassword(&domain.SessionData{Username: &bob}) {
		t.Error("bob (SSO sans MDP) → false attendu")
	}
	if svc.currentUserHasPassword(nil) {
		t.Error("session nil → false")
	}
	if svc.currentUserHasPassword(&domain.SessionData{}) {
		t.Error("sans username → false")
	}
}

// ---------------------------------------------------------------------------
// ResolveAuthState
// ---------------------------------------------------------------------------

func TestResolveAuthState_NilSession(t *testing.T) {
	if ResolveAuthState(nil) != "missing" {
		t.Error("expected missing")
	}
}

func TestResolveAuthState_NotReady(t *testing.T) {
	sess := &domain.SessionData{AuthReady: false}
	if ResolveAuthState(sess) != "missing" {
		t.Error("expected missing")
	}
}

func TestResolveAuthState_NoIdentity(t *testing.T) {
	sess := &domain.SessionData{AuthReady: true}
	if ResolveAuthState(sess) != "partial" {
		t.Error("expected partial")
	}
}

func TestResolveAuthState_Ready(t *testing.T) {
	sess := &domain.SessionData{
		AuthReady:          true,
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}
	if ResolveAuthState(sess) != "ready" {
		t.Error("expected ready")
	}
}

// ---------------------------------------------------------------------------
// ResolveLinkedIdentity
// ---------------------------------------------------------------------------

func TestResolveLinkedIdentity_Nil(t *testing.T) {
	if ResolveLinkedIdentity(nil) != nil {
		t.Error("expected nil")
	}
}

func TestResolveLinkedIdentity_NoIdentity(t *testing.T) {
	sess := &domain.SessionData{}
	if ResolveLinkedIdentity(sess) != nil {
		t.Error("expected nil when no identity")
	}
}

func TestResolveLinkedIdentity_WithIdentity(t *testing.T) {
	sess := &domain.SessionData{
		LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT", XUID: "123"},
	}
	identity := ResolveLinkedIdentity(sess)
	if identity == nil || identity.Gamertag != "GT" {
		t.Error("expected identity with GT")
	}
}

// ---------------------------------------------------------------------------
// buildCapabilities
// ---------------------------------------------------------------------------

func TestBuildCapabilities_DemoMode(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: true}
	settings := map[string]interface{}{}
	caps := buildCapabilities(cfg, settings)
	if caps.CanRunSync {
		t.Error("DemoMode should disable sync")
	}
	if caps.CanUseLiveHalo {
		t.Error("DemoMode should disable live Halo")
	}
}

func TestBuildCapabilities_Normal(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: false}
	settings := map[string]interface{}{"media_enabled": false}
	caps := buildCapabilities(cfg, settings)
	if !caps.CanRunSync {
		t.Error("normal mode should allow sync")
	}
	if caps.CanViewMedia {
		t.Error("media_enabled=false should disable CanViewMedia")
	}
}

// ---------------------------------------------------------------------------
// buildSettingsExcerpt
// ---------------------------------------------------------------------------

func TestBuildSettingsExcerpt_Defaults(t *testing.T) {
	cfg := &config.AppConfig{Lang: "fr"}
	settings := map[string]interface{}{}
	excerpt := buildSettingsExcerpt(cfg, settings)
	if excerpt.Lang != "fr" {
		t.Errorf("expected fr, got %s", excerpt.Lang)
	}
	if excerpt.UserTimezone != "Europe/Paris" {
		t.Errorf("expected Europe/Paris, got %s", excerpt.UserTimezone)
	}
}

func TestBuildSettingsExcerpt_Override(t *testing.T) {
	cfg := &config.AppConfig{Lang: "fr"}
	settings := map[string]interface{}{
		"lang":          "en",
		"user_timezone": "America/New_York",
	}
	excerpt := buildSettingsExcerpt(cfg, settings)
	if excerpt.Lang != "en" {
		t.Errorf("expected en, got %s", excerpt.Lang)
	}
	if excerpt.UserTimezone != "America/New_York" {
		t.Errorf("expected America/New_York, got %s", excerpt.UserTimezone)
	}
}

// ---------------------------------------------------------------------------
// buildFeatureFlags
// ---------------------------------------------------------------------------

func TestBuildFeatureFlags_Demo(t *testing.T) {
	cfg := &config.AppConfig{DemoMode: true}
	settings := map[string]interface{}{}
	flags := buildFeatureFlags(cfg, settings)
	if !flags.DemoMode {
		t.Error("expected DemoMode true")
	}
	if !flags.V7Enabled {
		t.Error("expected V7Enabled true")
	}
}

func TestBuildFeatureFlags_Discord(t *testing.T) {
	cfg := &config.AppConfig{}
	settings := map[string]interface{}{
		"discord_webhook_url": "https://discord.com/hook",
	}
	flags := buildFeatureFlags(cfg, settings)
	if !flags.DiscordConfigured {
		t.Error("expected DiscordConfigured true")
	}
}

func TestBuildFeatureFlags_Tailscale(t *testing.T) {
	cfg := &config.AppConfig{}

	flagsOff := buildFeatureFlags(cfg, map[string]interface{}{})
	if flagsOff.TailscaleEnabled {
		t.Error("TailscaleEnabled should default to false")
	}

	flagsOn := buildFeatureFlags(cfg, map[string]interface{}{"tailscale_enabled": true})
	if !flagsOn.TailscaleEnabled {
		t.Error("TailscaleEnabled should be true when tailscale_enabled=true in settings")
	}
}

// ---------------------------------------------------------------------------
// BuildAvailableTitles
// ---------------------------------------------------------------------------

func TestBuildAvailableTitles(t *testing.T) {
	titles := BuildAvailableTitles()
	if len(titles) == 0 {
		t.Error("expected at least one title")
	}
	foundDefault := false
	for _, title := range titles {
		if title.IsDefault {
			foundDefault = true
		}
		if title.Slug == "" {
			t.Error("title slug should not be empty")
		}
	}
	if !foundDefault {
		t.Error("expected at least one default title")
	}
}

// TestBuildAvailableTitles_FiltersArchivedKeepsComingSoon — MT-22 (PMT-8) :
// le switcher exclut archived mais conserve coming_soon AVEC son status.
func TestBuildAvailableTitles_FiltersArchivedKeepsComingSoon(t *testing.T) {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{Slug: "soon", Name: "Soon", Status: titlePkg.StatusComingSoon})
	reg.Register(&titlePkg.TitleDescriptor{Slug: "old", Name: "Old", Status: titlePkg.StatusArchived})

	titles := buildAvailableTitlesFrom(reg)

	byslug := map[string]domain.TitleSummary{}
	for _, ts := range titles {
		byslug[ts.Slug] = ts
	}
	if _, ok := byslug["old"]; ok {
		t.Error("le titre archived ne doit pas apparaître dans le switcher")
	}
	soon, ok := byslug["soon"]
	if !ok {
		t.Fatal("le titre coming_soon doit apparaître dans le switcher")
	}
	if soon.Status != string(titlePkg.StatusComingSoon) {
		t.Errorf("status coming_soon doit être conservé, got %q", soon.Status)
	}
	if _, ok := byslug[titlePkg.DefaultSlug]; !ok {
		t.Errorf("le titre actif par défaut %q doit apparaître", titlePkg.DefaultSlug)
	}
}

// TestBuildAvailableTitles_ExcludesInternal — revue UX H5 : une fixture de test
// interne (IsInternal=true, ex synthetic_title_b) ne doit JAMAIS apparaître dans
// le switcher utilisateur, même en coming_soon. Garde-fou anti-régression contre
// la fuite « Synthetic Title B » vue en prod.
func TestBuildAvailableTitles_ExcludesInternal(t *testing.T) {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{Slug: "internal_fixture", Name: "Internal", Status: titlePkg.StatusComingSoon, IsInternal: true})
	reg.Register(&titlePkg.TitleDescriptor{Slug: "real_soon", Name: "Real Soon", Status: titlePkg.StatusComingSoon})

	titles := buildAvailableTitlesFrom(reg)

	byslug := map[string]domain.TitleSummary{}
	for _, ts := range titles {
		byslug[ts.Slug] = ts
	}
	if _, ok := byslug["internal_fixture"]; ok {
		t.Error("un titre IsInternal ne doit pas apparaître dans le switcher utilisateur")
	}
	if _, ok := byslug["real_soon"]; !ok {
		t.Error("un coming_soon NON interne doit rester visible dans le switcher")
	}
}
