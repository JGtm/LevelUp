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
	"levelup/go-api/internal/games"
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
	coMembers        func(xuid string) map[string]bool // optionnel — co-membres de groupe du user (nil = strict owner-only)
	// matchCountForTitle compte les matchs du shared du TITRE COURANT (title-aware).
	// nil = fallback sur bootRepo.GetMatchCount (shared figé Infinite au boot). Injecté
	// au composition root car la résolution per-titre vit dans config (import duckdb),
	// que le package service ne doit pas importer directement (règle de couches).
	matchCountForTitle func(ctx context.Context, titleSlug string) (int, error)
}

// setupCountBudget borne le temps d'attente du décompte des matchs servant à
// distinguer ready vs profile_ready_no_sync. Court volontairement : ce décompte
// est cosmétique (signal du wizard), et sous contention DB (sync tenant le writer
// RW) une lecture shared peut pendre jusqu'au readyTimeout du provider (~30s), ce
// qui ferait échouer/rollback un switch de titre. Au-delà du budget, on dégrade.
const setupCountBudget = 2 * time.Second

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

// WithReauthChecker injecte la fonction qui dit si un xuid a son refresh_token
// mort (reauth_required). Sans elle, reauth_required est toujours false. PR-B.
func (s *BootstrapService) WithReauthChecker(fn func(xuid string) bool) *BootstrapService {
	s.reauthCheck = fn
	return s
}

// WithUserLookup injecte le user store pour le filtrage ownership des joueurs
// (ADR 0029). Sans lui, available_players n'est pas filtré (mono-utilisateur).
func (s *BootstrapService) WithUserLookup(lookup authz.UserLookup) *BootstrapService {
	s.userLookup = lookup
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
	capabilities := buildCapabilities(s.cfg, appSettings)
	settingsExcerpt := buildSettingsExcerpt(s.cfg, appSettings)
	flags := buildFeatureFlags(s.cfg, appSettings)

	setupState := s.resolveSetupState(ctx, sess, currentTitleSlug, players)

	// Les profils auth-only (gestion des tokens, pas de vrais joueurs) sont exclus
	// des listes front-facing : sélecteur L1 + favoris gamertag (Escouade/Explorer).
	// Le setup_state ci-dessus reste calculé sur la liste complète (côté instance).
	visiblePlayers := excludeAuthOnly(players)

	// Couche A (ADR 0029) : available_players et le joueur courant sont restreints
	// aux profils accessibles par l'utilisateur (les siens + ses co-membres de groupe).
	familyXUIDs := s.resolveCoMembers(sess)
	ownedPlayers := s.filterOwnedPlayers(sess, visiblePlayers, familyXUIDs)
	if len(ownedPlayers) != len(visiblePlayers) {
		slog.DebugContext(ctx, "bootstrap: available_players filtré par ownership",
			"owned", len(ownedPlayers), "total", len(visiblePlayers), "username", resolveUsername(sess))
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

	// Locale par défaut : en démo, on force l'anglais (audience internationale de
	// la vitrine publique). Le visiteur peut toujours basculer la langue en session
	// côté client (store appShell + header X-LevelUp-Locale) ; le PATCH /settings
	// restant refusé en démo, ce choix n'est pas persisté côté serveur.
	locale := settingsExcerpt.Lang
	if s.cfg.DemoMode {
		locale = "en"
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
		Locale:               locale,
		HintsVisibleDefault:  true,
		FeatureFlags:         flags,
		Capabilities:         capabilities,
		SettingsExcerpt:      settingsExcerpt,
		Privacy:              privacy,
		DemoMode:             s.cfg.DemoMode,
		AuthMode:             s.cfg.AuthMode,
		RegistrationMode:     s.cfg.RegistrationMode,
		InstanceLocked:       s.cfg.InstanceLocked || getBoolSetting(appSettings, "instance_locked", false),
		ReauthRequired:       s.resolveReauthRequired(ctx, sess),
		HasPassword:          s.currentUserHasPassword(sess),
		IsAdmin:              sess != nil && sess.Role != nil && *sess.Role == "admin",
		CurrentUsername:      resolveUsername(sess),
		FirstLaunch:          s.isFirstLaunch(),
		OAuthCodeFlowEnabled: s.cfg.AuthMode == "xbox" && s.cfg.OAuthRedirectURI != "",
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

// filterOwnedPlayers restreint la liste aux profils accessibles par l'utilisateur
// courant (ADR 0029). Retourne la liste intacte si l'enforcement est désactivé
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

// resolveCoMembers retourne l'ensemble des xuids accessibles par le user courant
// via partage de groupe (co-membres). Nil si pas de résolveur câblé, pas d'user
// lié, ou aucun groupe → CanAccessPlayer retombe sur propriétaire-only strict.
func (s *BootstrapService) resolveCoMembers(sess *domain.SessionData) map[string]bool {
	if s.coMembers == nil || s.userLookup == nil {
		return nil
	}
	user := authz.CurrentUser(sess, s.userLookup)
	if user == nil || user.XUID == "" {
		return nil
	}
	return s.coMembers(user.XUID)
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
	// Exclure les profils auth-only : cette liste alimente les mêmes surfaces
	// front-facing que available_players (favoris gamertag, sélecteur joueur).
	players = excludeAuthOnly(players)
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

// excludeAuthOnly retire les profils auth-only (existant uniquement pour la
// gestion des tokens, sans suivi de stats) d'une liste destinée au front. Ces
// profils ne doivent jamais apparaître dans le sélecteur L1 ni les favoris de
// gamertag (Escouade/Explorer). La résolution token côté serveur continue de
// les voir via cfg.LoadPlayers (non filtré).
func excludeAuthOnly(players []domain.PlayerSummary) []domain.PlayerSummary {
	out := make([]domain.PlayerSummary, 0, len(players))
	for _, p := range players {
		if p.AuthOnly {
			continue
		}
		out = append(out, p)
	}
	return out
}

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

func (s *BootstrapService) resolveSetupState(ctx context.Context, sess *domain.SessionData, currentTitleSlug string, players []domain.PlayerSummary) string {
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
	// Compter les matchs du shared du TITRE COURANT, best-effort borné : sous
	// contention (sync en cours) la dégradation profile_ready_no_sync est bénigne
	// (le routing /setup est gaté par players==0, pas par setup_state) et n'empêche
	// jamais un switch de titre d'aboutir.
	count, ok := s.matchCountForSetup(ctx, currentTitleSlug)
	if !ok {
		return bootstrapSetupProfileReadyNoSync
	}
	if count > 0 {
		return bootstrapSetupReady
	}
	return bootstrapSetupProfileReadyNoSync
}

// matchCountForSetup retourne (count, true) si le décompte des matchs du titre
// courant a été obtenu dans setupCountBudget, (0, false) sinon (dégradation). Le
// décompte passe par le résolveur title-aware s'il est câblé, sinon par bootRepo
// (legacy). Exécuté dans une goroutine bornée pour qu'un provider contendu
// (StateError, swap RW long) ne puisse jamais faire pendre le bootstrap.
func (s *BootstrapService) matchCountForSetup(ctx context.Context, currentTitleSlug string) (int, bool) {
	countFn := func(cctx context.Context) (int, error) {
		if s.matchCountForTitle != nil {
			return s.matchCountForTitle(cctx, currentTitleSlug)
		}
		if s.bootRepo != nil {
			return s.bootRepo.GetMatchCount(cctx)
		}
		return 0, fmt.Errorf("matchCountForSetup: aucun décompte disponible")
	}

	cctx, cancel := context.WithTimeout(ctx, setupCountBudget)
	defer cancel()

	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1) // bufferisé : la goroutine ne fuit pas si on timeout
	go func() {
		n, err := countFn(cctx)
		ch <- result{n: n, err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			slog.WarnContext(ctx, "bootstrap: décompte matchs setup_state indisponible (dégradé)",
				"err", r.err, "title", currentTitleSlug)
			return 0, false
		}
		return r.n, true
	case <-cctx.Done():
		slog.WarnContext(ctx, "bootstrap: décompte matchs setup_state au-delà du budget, dégradation",
			"title", currentTitleSlug, "budget_ms", setupCountBudget.Milliseconds())
		return 0, false
	}
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

// BuildAvailableTitles construit la liste des titres depuis le registre PARTAGÉ
// (MT-16 / day-one 2e titre : DefaultRegistry inclut les titres découverts en
// config). Sprint 49 : exportée pour réutilisation par le handler session_context.
func BuildAvailableTitles() []domain.TitleSummary {
	return buildAvailableTitlesFrom(titlePkg.DefaultRegistry())
}

// buildAvailableTitlesFrom projette les titres servables d'un registre donné.
// Extrait pour testabilité (registre injectable avec coming_soon/archived).
func buildAvailableTitlesFrom(reg *titlePkg.Registry) []domain.TitleSummary {
	// MT-22 (PMT-8) : le switcher liste les titres jouables (active) ET ceux
	// « bientôt disponibles » (coming_soon), en conservant leur Status pour que
	// le front affiche l'état. Exclus : les retirés (archived) ET les internes
	// (fixtures de test comme synthetic_title_b — cf. IsInternal). Tous deux
	// restent inspectables côté admin (/admin/titles via reg.All/NonArchived).
	all := reg.PublicTitles()
	out := make([]domain.TitleSummary, 0, len(all))
	for _, t := range all {
		caps := make([]string, len(t.Capabilities))
		for i, c := range t.Capabilities {
			caps[i] = string(c)
		}
		out = append(out, domain.TitleSummary{
			Slug:                    t.Slug,
			Name:                    t.Name,
			IconURL:                 t.IconURL,
			Status:                  string(t.Status),
			Capabilities:            caps,
			IsDefault:               t.IsDefault,
			EffectiveHpToKill:       games.EffectiveHpToKill(t.Slug),
			OffensiveConversionP80:  games.OffensiveConversionP80(t.Slug),
			ProvidesDamageTaken:     games.ProvidesDamageTaken(t.Slug),
			ProvidesTeamMMR:         games.ProvidesTeamMMR(t.Slug),
			ProvidesMaxKillingSpree: games.ProvidesMaxKillingSpree(t.Slug),
		})
	}
	return out
}
