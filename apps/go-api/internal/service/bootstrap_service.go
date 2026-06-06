// Package service orchestre la logique métier en combinant config + repositories.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// États possibles renvoyés par resolveSetupState et ResolveAuthState.
const (
	bootstrapSetupReady               = "ready"
	bootstrapSetupProfileReadyNoSync  = "profile_ready_no_sync"
	bootstrapSetupNoHaloLink          = "no_halo_link"
	bootstrapSetupHaloLinkedNoProfile = "halo_linked_no_profile"
	bootstrapAuthMissing              = "missing"
)

// BootstrapService construit le BootstrapResponse pour l'endpoint /api/v1/bootstrap.
type BootstrapService struct {
	cfg              *config.AppConfig
	bootRepo         port.BootstrapRepository
	privacyProvider  port.PrivacyProvider        // optionnel — nil = pas de check privacy
	privacyStateRepo port.PrivacyStateRepository // optionnel — nil = pas de fallback persisté
	userStoreEmpty   func() (bool, error)        // optionnel — nil = first_launch toujours false
	userLookup       authz.UserLookup            // optionnel — nil = pas de filtrage ownership (ADR 0024)
}

// NewBootstrapService crée un BootstrapService.
func NewBootstrapService(cfg *config.AppConfig, bootRepo port.BootstrapRepository) *BootstrapService {
	return &BootstrapService{cfg: cfg, bootRepo: bootRepo}
}

// WithPrivacyProvider injecte le provider de match privacy (optionnel).
func (s *BootstrapService) WithPrivacyProvider(p port.PrivacyProvider) *BootstrapService {
	s.privacyProvider = p
	return s
}

// WithPrivacyStateRepo injecte le repo de persistance du state privacy (optionnel).
// Sprint 55 E4 : permet le fallback gracieux quand Waypoint est indisponible.
func (s *BootstrapService) WithPrivacyStateRepo(r port.PrivacyStateRepository) *BootstrapService {
	s.privacyStateRepo = r
	return s
}

// WithUserStoreEmpty injecte la fonction de vérification "user store vide" (mode password).
func (s *BootstrapService) WithUserStoreEmpty(fn func() (bool, error)) *BootstrapService {
	s.userStoreEmpty = fn
	return s
}

// WithUserLookup injecte le user store pour le filtrage ownership des joueurs
// (ADR 0024). Sans lui, available_players n'est pas filtré (mono-utilisateur).
func (s *BootstrapService) WithUserLookup(lookup authz.UserLookup) *BootstrapService {
	s.userLookup = lookup
	return s
}

// Build construit la réponse bootstrap complète.
// sess peut être nil si la session n'est pas encore initialisée.
func (s *BootstrapService) Build(ctx context.Context, sess *domain.SessionData) (*domain.BootstrapResponse, error) {
	// Sprint 44 : titre courant depuis la session, fallback halo_infinite.
	currentTitleSlug := titlePkg.DefaultSlug
	if sess != nil && sess.CurrentTitleSlug != "" {
		currentTitleSlug = sess.CurrentTitleSlug
	}

	players, err := s.cfg.LoadPlayers(currentTitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "bootstrap: erreur chargement joueurs", "err", err)
		players = []domain.PlayerSummary{}
	}
	slog.DebugContext(ctx, "bootstrap: filtrage joueurs", "title", currentTitleSlug, "count", len(players))

	appSettings, err := s.cfg.LoadAppSettings()
	if err != nil {
		slog.WarnContext(ctx, "bootstrap: erreur chargement app_settings", "err", err)
		appSettings = map[string]interface{}{}
	}

	// setupRequired / setupState reflètent l'état de l'INSTANCE (joueurs configurés),
	// pas la propriété : on les calcule sur la liste complète.
	setupRequired := !s.cfg.DemoMode && len(players) == 0
	capabilities := buildCapabilities(s.cfg, appSettings)
	settingsExcerpt := buildSettingsExcerpt(s.cfg, appSettings)
	flags := buildFeatureFlags(s.cfg, appSettings)

	setupState := s.resolveSetupState(ctx, sess, players)

	// Couche A (ADR 0024) : available_players et le joueur courant sont restreints
	// aux profils accessibles par l'utilisateur (les siens + la famille, #21).
	familyXUIDs := authz.ResolveFamilyXUIDs(friendGamertagsFromSettings(appSettings), players)
	ownedPlayers := s.filterOwnedPlayers(sess, players, familyXUIDs)
	if len(ownedPlayers) != len(players) {
		slog.DebugContext(ctx, "bootstrap: available_players filtré par ownership",
			"owned", len(ownedPlayers), "total", len(players), "username", resolveUsername(sess))
	}

	var currentPlayer *domain.PlayerSummary
	if len(ownedPlayers) > 0 {
		// Respecter le choix de joueur stocké en session, sinon fallback ownedPlayers[0].
		idx := 0
		if sess != nil && sess.CurrentPlayerSlug != nil {
			for i, p := range ownedPlayers {
				if p.PlayerSlug == *sess.CurrentPlayerSlug {
					idx = i
					break
				}
			}
		}
		p := ownedPlayers[idx]
		currentPlayer = &p
	}

	// Sprint 54-B / 55-E : privacy match du joueur courant.
	// Priorité : Waypoint live → state persisté en DB → nil (inconnu).
	var privacy *domain.MatchPrivacyInfo
	if currentPlayer != nil && currentPlayer.XUID != "" {
		if s.privacyProvider != nil {
			privacy = s.fetchPrivacyNonBlocking(ctx, currentPlayer.XUID)
			// E2 : persister le résultat Waypoint pour les prochains appels
			if privacy != nil && s.privacyStateRepo != nil {
				state := domain.PlayerPrivacyState{
					XUID:       currentPlayer.XUID,
					IsPrivate:  privacy.IsPrivate,
					ObservedAt: time.Now().UTC(),
					Source:     "waypoint",
				}
				_ = s.privacyStateRepo.UpsertPrivacyState(ctx, state)
			}
		}
		// E3 : fallback sur le state persisté si Waypoint indisponible
		if privacy == nil && s.privacyStateRepo != nil {
			if stored, err := s.privacyStateRepo.LoadPrivacyState(ctx, currentPlayer.XUID); err == nil && stored != nil {
				slog.DebugContext(ctx, "bootstrap: privacy fallback sur state persisté", "xuid", currentPlayer.XUID)
				privacy = &domain.MatchPrivacyInfo{
					IsPrivate: stored.IsPrivate,
					IsPartial: false,
					Hint:      "cached",
				}
			}
		}
	}

	return &domain.BootstrapResponse{
		SetupRequired:        setupRequired,
		AuthState:            ResolveAuthState(sess),
		SetupState:           setupState,
		LinkedHaloIdentity:   ResolveLinkedIdentity(sess),
		CurrentPlayer:        currentPlayer,
		AvailablePlayers:     ownedPlayers,
		CurrentTitleSlug:     currentTitleSlug,
		AvailableTitles:      BuildAvailableTitles(),
		Locale:               settingsExcerpt.Lang,
		HintsVisibleDefault:  true,
		FeatureFlags:         flags,
		Capabilities:         capabilities,
		SettingsExcerpt:      settingsExcerpt,
		Privacy:              privacy,
		AuthMode:             s.cfg.AuthMode,
		RegistrationMode:     s.cfg.RegistrationMode,
		IsAdmin:              sess != nil && sess.Role != nil && *sess.Role == "admin",
		CurrentUsername:      resolveUsername(sess),
		FirstLaunch:          s.isFirstLaunch(),
		OAuthCodeFlowEnabled: s.cfg.AuthMode == "xbox" && s.cfg.OAuthRedirectURI != "",
	}, nil
}

// filterOwnedPlayers restreint la liste aux profils accessibles par l'utilisateur
// courant (ADR 0024). Retourne la liste intacte si l'enforcement est désactivé
// (mode demo / auth non activée) ou si le user store n'est pas câblé. Un admin
// voit tout ; un utilisateur standard voit son xuid + les profils de la famille
// (familyXUIDs, #21 Phase A) pour que le sélecteur L1 liste bien la famille.
func (s *BootstrapService) filterOwnedPlayers(sess *domain.SessionData, players []domain.PlayerSummary, familyXUIDs map[string]bool) []domain.PlayerSummary {
	if !authz.Enforced(s.cfg.DemoMode, s.cfg.AuthMode) || s.userLookup == nil {
		return players
	}
	user := authz.CurrentUser(sess, s.userLookup)
	out := make([]domain.PlayerSummary, 0, len(players))
	for _, p := range players {
		if authz.CanAccessPlayer(true, user, p.XUID, familyXUIDs) {
			out = append(out, p)
		}
	}
	return out
}

// friendGamertagsFromSettings extrait friend_gamertags de la map app_settings
// (LoadAppSettings retourne une map non typée → JSON décode en []interface{}).
func friendGamertagsFromSettings(appSettings map[string]interface{}) []string {
	raw, ok := appSettings["friend_gamertags"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if gt, ok := v.(string); ok && gt != "" {
			out = append(out, gt)
		}
	}
	return out
}

// fetchPrivacyNonBlocking fetche la privacy avec un timeout court (2 s).
// En cas d'échec, renvoie nil sans bloquer le bootstrap.
func (s *BootstrapService) fetchPrivacyNonBlocking(ctx context.Context, xuid string) *domain.MatchPrivacyInfo {
	type result struct {
		info *domain.MatchPrivacyInfo
	}
	ch := make(chan result, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	go func() {
		info, err := s.privacyProvider.GetMatchPrivacy(timeoutCtx, xuid)
		if err != nil {
			slog.DebugContext(ctx, "bootstrap: privacy fetch échoué", "xuid", xuid, "err", err)
			ch <- result{nil}
			return
		}
		ch <- result{info}
	}()

	select {
	case r := <-ch:
		return r.info
	case <-timeoutCtx.Done():
		slog.DebugContext(ctx, "bootstrap: privacy fetch timeout", "xuid", xuid)
		return nil
	}
}

// BuildPlayersList construit la liste des joueurs pour GET /api/v1/players.
func (s *BootstrapService) BuildPlayersList(ctx context.Context) (*domain.PlayersListResponse, error) {
	titleSlug := ctxkeys.TitleSlug(ctx)
	players, err := s.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, fmt.Errorf("BuildPlayersList: %w", err)
	}
	var defaultSlug *string
	if len(players) > 0 {
		slug := players[0].PlayerSlug
		defaultSlug = &slug
	}
	return &domain.PlayersListResponse{
		Items:             players,
		DefaultPlayerSlug: defaultSlug,
	}, nil
}

// --- helpers ---

func buildCapabilities(cfg *config.AppConfig, settings map[string]interface{}) domain.CapabilityMap {
	mediaEnabled := getBoolSetting(settings, "media_enabled", true)
	return domain.CapabilityMap{
		CanReadLocalData:    true,
		CanRunSync:          !cfg.DemoMode,
		CanUseLiveHalo:      !cfg.DemoMode,
		CanManageSettings:   true,
		CanResetMediaIndex:  true,
		CanViewMedia:        mediaEnabled,
		CanSelfProvision:    getBoolSetting(settings, "can_self_provision", true),
		CanStartInitialSync: getBoolSetting(settings, "can_start_initial_sync", !cfg.DemoMode),
		CanManageInstance:   true,
	}
}

func buildSettingsExcerpt(cfg *config.AppConfig, settings map[string]interface{}) domain.SettingsExcerpt {
	lang := getStringSetting(settings, "lang", cfg.Lang)
	return domain.SettingsExcerpt{
		Lang:                lang,
		UserTimezone:        getStringSetting(settings, "user_timezone", "Europe/Paris"),
		ShowRecords:         getBoolSetting(settings, "show_records", true),
		NormalizeModeLabels: getBoolSetting(settings, "normalize_mode_labels", true),
	}
}

func buildFeatureFlags(cfg *config.AppConfig, settings map[string]interface{}) domain.FeatureFlags {
	discordURL := cfg.DiscordWebhookURL
	if discordURL == "" {
		discordURL = getStringSetting(settings, "discord_webhook_url", "")
	}
	return domain.FeatureFlags{
		V7Enabled:         true,
		MediaEnabled:      getBoolSetting(settings, "media_enabled", true),
		DemoMode:          cfg.DemoMode,
		DiscordConfigured: discordURL != "",
		TailscaleEnabled:  getBoolSetting(settings, "tailscale_enabled", false),
	}
}

func (s *BootstrapService) resolveSetupState(ctx context.Context, sess *domain.SessionData, players []domain.PlayerSummary) string {
	if len(players) == 0 {
		// SSO terminé (identité Halo liée en session) mais aucun profil local créé :
		// router vers StepPlayer (confirmation/création de profil) au lieu de
		// reboucler sur StepDeviceCode. Le Device Code Flow finit en "authorized"
		// sans auto-provisionner — le profil est créé à l'étape suivante.
		if ResolveLinkedIdentity(sess) != nil {
			return bootstrapSetupHaloLinkedNoProfile
		}
		return bootstrapSetupNoHaloLink
	}
	// Vérifier si des matchs existent dans la shared DB.
	if s.bootRepo == nil {
		return bootstrapSetupProfileReadyNoSync
	}
	count, err := s.bootRepo.GetMatchCount(ctx)
	if err != nil {
		slog.WarnContext(ctx, "bootstrap: erreur GetMatchCount pour setup_state", "err", err)
		return bootstrapSetupProfileReadyNoSync
	}
	if count > 0 {
		return bootstrapSetupReady
	}
	return bootstrapSetupProfileReadyNoSync
}

// resolveUsername retourne le username de la session (ou nil).
func resolveUsername(sess *domain.SessionData) *string {
	if sess == nil {
		return nil
	}
	return sess.Username
}

// isFirstLaunch retourne true si le user store est vide (premier lancement en mode password).
func (s *BootstrapService) isFirstLaunch() bool {
	if s.userStoreEmpty == nil {
		return false
	}
	empty, err := s.userStoreEmpty()
	if err != nil {
		slog.Warn("bootstrap: erreur vérification first_launch", "err", err)
		return false
	}
	return empty
}

// ResolveAuthState déduit l'état d'authentification depuis la session.
func ResolveAuthState(sess *domain.SessionData) string {
	if sess == nil || !sess.AuthReady {
		return bootstrapAuthMissing
	}
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.Gamertag == "" {
		return "partial"
	}
	return bootstrapSetupReady
}

// ResolveLinkedIdentity extrait l'identité Halo liée si présente dans la session.
func ResolveLinkedIdentity(sess *domain.SessionData) *domain.HaloIdentitySummary {
	if sess == nil || sess.LinkedHaloIdentity == nil {
		return nil
	}
	return &domain.HaloIdentitySummary{
		Gamertag: sess.LinkedHaloIdentity.Gamertag,
		XUID:     sess.LinkedHaloIdentity.XUID,
	}
}

func getBoolSetting(settings map[string]interface{}, key string, def bool) bool {
	if v, ok := settings[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func getStringSetting(settings map[string]interface{}, key, def string) string {
	if v, ok := settings[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

// BuildAvailableTitles construit la liste des titres depuis le registre.
// Sprint 49 : exportée pour réutilisation par le handler session_context.
func BuildAvailableTitles() []domain.TitleSummary {
	reg := titlePkg.NewRegistry()
	all := reg.All()
	out := make([]domain.TitleSummary, 0, len(all))
	for _, t := range all {
		caps := make([]string, len(t.Capabilities))
		for i, c := range t.Capabilities {
			caps[i] = string(c)
		}
		out = append(out, domain.TitleSummary{
			Slug:         t.Slug,
			Name:         t.Name,
			IconURL:      t.IconURL,
			Status:       string(t.Status),
			Capabilities: caps,
			IsDefault:    t.IsDefault,
		})
	}
	return out
}
