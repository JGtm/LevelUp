// Package service orchestre la logique métier en combinant config + repositories.
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	settings_platform "levelup/go-api/internal/platform/settings"
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
	privacyProvider  port.PrivacyProvider              // optionnel — nil = pas de check privacy
	privacyStateRepo port.PrivacyStateRepository       // optionnel — nil = pas de fallback persisté
	userStoreEmpty   func() (bool, error)              // optionnel — nil = first_launch toujours false
	userLookup       authz.UserLookup                  // optionnel — nil = pas de filtrage ownership (ADR 0029)
	reauthCheck      func(xuid string) bool            // optionnel — nil = reauth_required toujours false (PR-B)
	poolSharing      func(xuid string) string          // optionnel — nil = pool_sharing « unavailable »
	coMembers        func(xuid string) map[string]bool // optionnel — co-membres de groupe du user (nil = strict owner-only)
	// matchCountForTitle compte les matchs du shared du TITRE COURANT (title-aware).
	// nil = fallback sur bootRepo.GetMatchCount (shared figé Infinite au boot). Injecté
	// au composition root car la résolution per-titre vit dans config (import duckdb),
	// que le package service ne doit pas importer directement (règle de couches).
	matchCountForTitle func(ctx context.Context, titleSlug string) (int, error)
	// instanceLocked résout le verrou « instance fermée » exposé au front. Injecté
	// depuis le point de décision unique (authz.InstanceLocked, ADR 0035 D5) : ce
	// service ne recalcule JAMAIS le verrou de son côté (garde-rail archlint).
	// nil = jamais verrouillé (seam de test, cohérent avec les autres consommateurs).
	instanceLocked func() bool
	// privacyLive borne l'appel live de privacy et mémorise ses échecs
	// (bootstrap_privacy.go, plan perf 2026-09-23 D5b.7).
	privacyLive *privacyLiveFetch
	// mediaH264Missing : aucun encodeur H.264 utilisable dans le ffmpeg de l'instance, sondé une fois au
	// démarrage (WithMediaH264Missing).
	mediaH264Missing bool
	// replayArtifactTitle : le titre sait fabriquer un rejeu (capacité `film.replay_artifact`), injecté au câblage
	// (WithReplayArtifactTitleCheck). nil = faux : sans lui, can_generate_replays est faux.
	replayArtifactTitle func(titleSlug string) bool
}

// NewBootstrapService crée un BootstrapService.
func NewBootstrapService(cfg *config.AppConfig, bootRepo port.BootstrapRepository) *BootstrapService {
	return &BootstrapService{cfg: cfg, bootRepo: bootRepo, privacyLive: newPrivacyLiveFetch()}
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

// WithReauthChecker injecte la fonction qui dit si un xuid a son refresh_token
// mort (reauth_required). Sans elle, reauth_required est toujours false. PR-B.
func (s *BootstrapService) WithReauthChecker(fn func(xuid string) bool) *BootstrapService {
	s.reauthCheck = fn
	return s
}

// WithMediaH264Missing pose le verdict de la sonde de l'outillage média faite au démarrage : aucun
// encodeur H.264 utilisable (ops.MediaToolingReport.ToHealthStatus). L'interface l'explique sur la page
// Médias au lieu de laisser un clip refusé sans raison.
func (s *BootstrapService) WithMediaH264Missing(missing bool) *BootstrapService {
	s.mediaH264Missing = missing
	return s
}

// WithReplayArtifactTitleCheck injecte le prédicat « ce titre déclare la capacité `film.replay_artifact` »,
// qui conditionne `can_generate_replays` (le service ne lit aucune configuration de titre lui-même).
func (s *BootstrapService) WithReplayArtifactTitleCheck(fn func(titleSlug string) bool) *BootstrapService {
	s.replayArtifactTitle = fn
	return s
}

// titleHasReplay : le titre sait-il fabriquer un rejeu ? Faux sans prédicat injecté.
func (s *BootstrapService) titleHasReplay(titleSlug string) bool {
	return s.replayArtifactTitle != nil && s.replayArtifactTitle(titleSlug)
}

// WithUserLookup injecte le user store pour le filtrage ownership des joueurs
// (ADR 0029). Sans lui, available_players n'est pas filtré (mono-utilisateur).
func (s *BootstrapService) WithUserLookup(lookup authz.UserLookup) *BootstrapService {
	s.userLookup = lookup
	return s
}

// WithInstanceLock injecte le résolveur du verrou « instance fermée » (ADR 0035
// D5). Sans lui, instance_locked est rendu false au front.
func (s *BootstrapService) WithInstanceLock(fn func() bool) *BootstrapService {
	s.instanceLocked = fn
	return s
}

// WithCoMemberResolver injecte le résolveur des xuids co-membres de groupe du
// user courant (groupstore.CoMemberXUIDs). Pilote le filtrage ownership famille
// (available_players). Sans lui, l'accès retombe sur propriétaire-only strict.
func (s *BootstrapService) WithCoMemberResolver(fn func(xuid string) map[string]bool) *BootstrapService {
	s.coMembers = fn
	return s
}

// WithMatchCountResolver injecte un décompte de matchs TITLE-AWARE : la fonction
// reçoit le titre courant et compte les matchs de SON shared (pas celui, figé au
// boot, du titre par défaut). Élimine le couplage qui faisait attendre à un switch
// Halo 5 le provider Infinite tenu par le sync Infinite. Sans elle, fallback sur
// bootRepo.GetMatchCount (comportement legacy mono-titre).
func (s *BootstrapService) WithMatchCountResolver(fn func(ctx context.Context, titleSlug string) (int, error)) *BootstrapService {
	s.matchCountForTitle = fn
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
	capabilities := buildCapabilities(s.cfg, appSettings, s.titleHasReplay(currentTitleSlug))
	settingsExcerpt := buildSettingsExcerpt(s.cfg, appSettings)
	flags := buildFeatureFlags(s.cfg, appSettings)

	setupState := s.resolveSetupState(ctx, sess, currentTitleSlug, players)

	// Les profils auth-only (gestion des tokens, pas de vrais joueurs) sont exclus
	// des listes front-facing : sélecteur L1 + favoris gamertag (Escouade/Explorer).
	// Le setup_state ci-dessus reste calculé sur la liste complète (côté instance).
	visiblePlayers := excludeAuthOnly(players)

	// Couche A (ADR 0029) : profils lisibles (siens + co-membres) ; inscriptibles : les siens.
	familyXUIDs := s.resolveCoMembers(sess)
	ownedPlayers := s.filterOwnedPlayers(sess, visiblePlayers, familyXUIDs)
	if len(ownedPlayers) != len(visiblePlayers) {
		slog.DebugContext(ctx, "bootstrap: available_players filtré par ownership",
			"owned", len(ownedPlayers), "total", len(visiblePlayers), "username", resolveUsername(sess))
	}

	var currentPlayer *domain.PlayerSummary
	if len(ownedPlayers) > 0 {
		// Respecter le choix de joueur stocké en session, sinon le profil par défaut.
		idx := s.defaultPlayerIndex(sess, ownedPlayers)
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

	// Locale par défaut : en démo, on force la locale vitrine (cfg.DemoLocale, défaut
	// "en" — audience internationale de la vitrine publique). Le visiteur peut toujours
	// basculer la langue en session côté client (store appShell + header X-LevelUp-Locale) ;
	// le PATCH /settings restant refusé en démo, ce choix n'est pas persisté côté serveur.
	// Les tests E2E pinnent LEVELUP_DEMO_LOCALE=fr pour exercer l'UI française.
	locale := settingsExcerpt.Lang
	if s.cfg.DemoMode {
		locale = s.cfg.DemoLocale
		if locale == "" {
			locale = "en"
		}
	}

	return &domain.BootstrapResponse{
		SetupRequired:        setupRequired,
		AuthState:            ResolveAuthState(sess),
		SetupState:           setupState,
		LinkedHaloIdentity:   ResolveLinkedIdentity(sess),
		CurrentPlayer:        currentPlayer,
		AvailablePlayers:     ownedPlayers,
		WritablePlayerSlugs:  s.writablePlayerSlugs(sess, ownedPlayers),
		CurrentTitleSlug:     currentTitleSlug,
		AvailableTitles:      BuildAvailableTitles(),
		Locale:               locale,
		HintsVisibleDefault:  true,
		FeatureFlags:         flags,
		Capabilities:         capabilities,
		SettingsExcerpt:      settingsExcerpt,
		Privacy:              privacy,
		DemoMode:             s.cfg.DemoMode,
		AuthMode:             s.cfg.PublishedAuthMode(),
		RegistrationMode:     s.cfg.RegistrationMode,
		InstanceLocked:       s.instanceLocked != nil && s.instanceLocked(),
		ReauthRequired:       s.resolveReauthRequired(ctx, sess),
		OwnHaloXUID:          s.ownXUID(sess),
		PoolSharing:          s.resolvePoolSharing(sess),
		HasPassword:          s.currentUserHasPassword(sess),
		IsAdmin:              sess != nil && sess.Role != nil && *sess.Role == "admin",
		CurrentUsername:      resolveUsername(sess),
		FirstLaunch:          s.isFirstLaunch(),
		OAuthCodeFlowEnabled: s.cfg.AuthMode == "xbox" && s.cfg.OAuthRedirectURI != "",
		AppInfo:              buildAppInfo(s.cfg),
		MediaH264Missing:     s.mediaH264Missing,
	}, nil
}

// resolveReauthRequired indique si le user CONNECTÉ doit re-passer le SSO Xbox
// (son refresh_token Microsoft est mort). Scopé à SON propre joueur via ownXUID,
// PAS au joueur courant affiché : un admin qui consulte un autre joueur ne doit
// pas hériter du bandeau d'un tiers — il ne peut rien faire pour le RT d'autrui,
// seul le propriétaire du compte peut se reconnecter. Retourne false sans checker
// câblé (PR-B) ou sans identité propre (compte password sans lien Halo).
func (s *BootstrapService) resolveReauthRequired(ctx context.Context, sess *domain.SessionData) bool {
	if s.reauthCheck == nil {
		return false
	}
	xuid := s.ownXUID(sess)
	if xuid == "" {
		return false
	}
	if s.reauthCheck(xuid) {
		slog.DebugContext(ctx, "bootstrap: reauth_required pour le compte connecté", "xuid", xuid)
		return true
	}
	return false
}

// ownXUID résout le xuid du joueur PROPRE au user de session (son compte Xbox
// lié), indépendamment du joueur courant affiché. Priorité au user persisté
// (authz.CurrentUser → username puis identité liée), fallback sur l'identité Halo
// de session quand aucun userLookup n'est câblé (mono-utilisateur). "" si aucune
// identité propre.
func (s *BootstrapService) ownXUID(sess *domain.SessionData) string {
	if u := authz.CurrentUser(sess, s.userLookup); u != nil && u.XUID != "" {
		return u.XUID
	}
	if id := ResolveLinkedIdentity(sess); id != nil {
		return id.XUID
	}
	return ""
}

// currentUserHasPassword indique si l'utilisateur connecté a défini un mot de
// passe (opt-in PR-C). False si pas de session/username, pas de userLookup, ou
// utilisateur introuvable. Best-effort, sans erreur propagée.
func (s *BootstrapService) currentUserHasPassword(sess *domain.SessionData) bool {
	if s.userLookup == nil || sess == nil || sess.Username == nil || *sess.Username == "" {
		return false
	}
	user, err := s.userLookup.Get(*sess.Username)
	return err == nil && user != nil && user.PasswordHash != ""
}

// --- helpers ---

// buildCapabilities : titleHasReplay dit que le titre courant sait fabriquer un rejeu (capacité `film.replay_artifact`) ;
// la génération à la demande en dépend en plus du profil de déploiement.
func buildCapabilities(cfg *config.AppConfig, settings map[string]interface{}, titleHasReplay bool) domain.CapabilityMap {
	mediaEnabled := getBoolSetting(settings, "media_enabled", true)
	dc := cfg.Capabilities
	return domain.CapabilityMap{
		CanReadLocalData:   true,
		CanRunSync:         !cfg.DemoMode,
		CanUseLiveHalo:     !cfg.DemoMode,
		CanManageSettings:  true,
		CanResetMediaIndex: true,
		CanViewMedia:       mediaEnabled,
		// Défaut de can_self_provision : MÊME règle que settings.Store (ADR 0035 D5)
		// — permissif hors mode appliqué, fermé quand l'instance applique la
		// propriété des joueurs. Sans cet alignement, le front proposerait une
		// création de profil que POST /setup/players refuse en 403.
		CanSelfProvision:    getBoolSetting(settings, "can_self_provision", !authz.Enforced(cfg.DemoMode, cfg.AuthMode)),
		CanStartInitialSync: getBoolSetting(settings, "can_start_initial_sync", !cfg.DemoMode),
		CanManageInstance:   true,

		// Capacités du profil de déploiement : décidées par config (ADR 0044), recopiées ici.
		CanUseAccounts:           dc.UseAccounts,
		CanSignInWithXbox:        dc.SignInWithXbox,
		CanUseInvitations:        dc.UseInvitations,
		CanUseGroups:             dc.UseGroups,
		CanPoolTokens:            dc.PoolTokens,
		CanUseAdmin:              dc.UseAdmin,
		CanUseRestic:             dc.UseResticBackup,
		CanUseDiscord:            dc.UseDiscord,
		CanUseOutboundNetwork:    dc.UseOutboundNetwork,
		CanUpdateApp:             dc.UpdateApp,
		CanOfferWindowsDownload:  dc.OfferWindowsDownload,
		CanEditSettings:          dc.EditSettings,
		CanUploadMedia:           dc.UploadMedia,
		CanUseCompactChrome:      dc.CompactChrome,
		CanLoginViaBrowser:       dc.LoginViaBrowser,
		CanCompareAccounts:       dc.CompareAccounts,
		CanWatchGameProcess:      dc.GameProcessWatch,
		CanIntegrateShell:        dc.ShellIntegration,
		CanScheduleRefresh:       dc.ScheduledRefresh,
		CanBackupToFolder:        dc.ParquetBackup,
		CanImportFromSite:        dc.ImportFromSite,
		CanExportForDesktop:      dc.ExportForDesktop,
		CanReportDiagnostic:      dc.DiagnosticReport,
		CanLimitMemory:           dc.MemoryLimit,
		CanRegenerateReplays:     dc.ReplayRegen,
		CanGenerateReplays:       dc.GenerateReplays && titleHasReplay,
		CanKeepMediaSources:      dc.KeepMediaSources,
		CanRevealFilesInExplorer: dc.RevealFilesInExplorer,
		CanTrashMediaSources:     dc.TrashMediaSources,
		CanPickFolder:            dc.PickFolder,
		Deployment:               string(cfg.Deployment),
	}
}

// buildSettingsExcerpt : langue et fuseau servis à l'interface. Une clé écrite gagne ; une clé non
// écrite (absente ou vide, config.DeclaredString) prend le défaut, qui est celui du magasin de
// réglages (le serveur et l'interface voient la même valeur) : sous la capacité AdoptSystemLocale,
// la langue et le fuseau du système, et « en » tant que le système ne désigne aucune langue de
// l'application (l'interface choisit alors la sienne) ; hors capacité, LEVELUP_LANG et le fuseau
// historique.
func buildSettingsExcerpt(cfg *config.AppConfig, settings map[string]interface{}) domain.SettingsExcerpt {
	defaults := settings_platform.Defaults()
	fallbackLang := cfg.Lang
	if cfg.Capabilities.AdoptSystemLocale {
		fallbackLang = defaults.Lang
	}
	lang, ok := config.DeclaredString(settings["lang"])
	if !ok {
		lang = cfg.SystemLocale.LangOr(fallbackLang)
	}
	tz, ok := config.DeclaredString(settings["user_timezone"])
	if !ok {
		tz = cfg.SystemLocale.TimezoneOr(defaults.UserTimezone)
	}
	return domain.SettingsExcerpt{
		Lang:                lang,
		UserTimezone:        tz,
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
