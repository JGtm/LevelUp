package api

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability/logging"
	auth_platform "levelup/go-api/internal/platform/auth"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/halo"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	lab_platform "levelup/go-api/internal/platform/lab"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/coach_advisor"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
	"levelup/go-api/pkg/duckdbbackup"
)

// server_apiv1.go — montage des routes /api/v1, extrait de NewRouter (K2a) pour
// dé-goder la fonction d'assemblage DI. Le bloc construit ses ~55 handlers en
// interne et les monte ; les dépendances de la portée NewRouter sont regroupées
// dans apiV1Deps.
// apiV1Deps regroupe les dépendances de la portée NewRouter consommées par le
// montage /api/v1 (extraction K2a) — évite une liste de paramètres > 5.
type apiV1Deps struct {
	cfg                   *config.AppConfig
	bootSvc               *service.BootstrapService
	reg                   *wire.ServiceRegistry
	fieldMappingsRegistry *mappings.Registry
	attemptStore          *auth_platform.AttemptStore
	users                 *userstore.Store
	invites               *userstore.InviteStore
	sessionStore          *session_platform.Store
	tokenProvider         auth_platform.TokenProvider
	groupStore            *groupstore.GroupStore
	settingsStore         *settings_platform.Store
	assetHandler          *handlers.AssetHandler
	assetMetaHandler      *handlers.AssetMetadataHandler
	gamertagSvc           port.GamertagSearchService
	prestigeBundle        *wire.PrestigeBundle
	serverCtx             context.Context
	daemon                watcher.DaemonController
	autoSyncScheduler     *scheduler.AutoSyncScheduler
	authStore             *auth_platform.MultiUserTokenStore
	jobStore              *jobs_platform.Store
	titleRegistry         *titlePkg.Registry
	backupScheduler       *duckdbbackup.Scheduler
}

// (Mount + sous-routeurs) ; longueur/complexité inhérentes à un assembleur de routes.
//
//nolint:funlen,gocyclo // montage de routes /api/v1 : liste séquentielle de ~55 handlers
func mountAPIV1(r chi.Router, d apiV1Deps) *handlers.XboxOAuthHandler {
	cfg := d.cfg
	bootSvc := d.bootSvc
	reg := d.reg
	fieldMappingsRegistry := d.fieldMappingsRegistry
	attemptStore := d.attemptStore
	users := d.users
	invites := d.invites
	sessionStore := d.sessionStore
	tokenProvider := d.tokenProvider
	groupStore := d.groupStore
	settingsStore := d.settingsStore
	assetHandler := d.assetHandler
	assetMetaHandler := d.assetMetaHandler
	gamertagSvc := d.gamertagSvc
	prestigeBundle := d.prestigeBundle
	serverCtx := d.serverCtx
	daemon := d.daemon
	autoSyncScheduler := d.autoSyncScheduler
	authStore := d.authStore
	jobStore := d.jobStore
	titleRegistry := d.titleRegistry
	backupScheduler := d.backupScheduler
	var xboxOAuthRoot *handlers.XboxOAuthHandler
	// Phase 3b : API Huma COEXISTANTE sur ce sous-routeur /api/v1. Les routes
	// migrées (huma.Register/huma.Get) cohabitent avec les routes chi non
	// migrées sur le MÊME *chi.Mux → mêmes middlewares racine + visibles à
	// chi.Walk/contract_test. La migration est incrémentale, route par route.
	humaAPI := newHumaAPI(r)
	registerChangelogHuma(humaAPI, handlers.NewChangelogHandler(cfg.RepoRoot))

	// Endpoints P0 : bootstrap + liste joueurs
	handlers.NewBootstrapHandler(bootSvc).Mount(r)
	handlers.NewPlayersHandler(bootSvc).Mount(r)

	// Smoke endpoint pour la home page : sonde le contenu (banner, peaks
	// CSR/LUSR, playlists récentes, arme favorite) et renvoie 503 si une
	// section est vide sans raison. Pensé pour CI post-backfill et alerte
	// dev. Cf. handlers/health_home.go.
	handlers.NewHealthHomeHandler(reg.HomeCtxWithAuth).Mount(r.With(middleware.NoStore))

	// Phase 9 du plan pipeline CSR : diagnostic coverage CSR pour un joueur.
	// Permet de vérifier en 1 ligne si le pipeline a bien capturé les CSR
	// (matured + placement) ou s'il faut lancer un backfill.
	handlers.NewDiagCSRHandler(reg.CSRCoverageProvider).Mount(r.With(middleware.NoStore))

	// Phase 4 plan stabilisation 2026-05-22 : diagnostic progression V2
	// (Ascension). Compte les rows dans streak/player_records/record_history/
	// milestone_earned + milestone_catalog. Permet de vérifier que
	// EvaluateProgressionAfterSync tourne bien sur l'auto-sync (avant Phase 4
	// ces tables restaient vides — cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED).
	handlers.NewDiagProgressionHandler(reg.ProgressionDiagProvider).Mount(r.With(middleware.NoStore))

	// Fix 2026-05-30 : backfill progression V2 in-process. Force une
	// évaluation idempotente (streaks/records/milestones) pour un joueur
	// dont l'historique existe mais dont le pipeline post-sync n'avait
	// jamais abouti (incident timeout shared reader). Renvoie le diag
	// post-exécution.
	handlers.NewProgressionBackfillHandler(reg.ProgressionBackfillProvider).Mount(r.With(middleware.NoStore))

	// Phase A multi-titres : exposition des field mappings TOML.
	// Derrière MULTI_TITLE_API_ENABLED.
	//
	// NOTE : les endpoints preview/career et preview/career-multi-title
	// (proof-of-concept Phase C) ont été supprimés en revue 2026-04-29 P0.2
	// Q6 — orphelins côté front même flag activé. À réintroduire en endpoint
	// admin/debug si besoin de re-valider le pipeline canonique.
	if cfg.MultiTitleAPIEnabled {
		slog.Info("multi_title_api_enabled", "routes", []string{"/titles/{slug}/field-mappings", "/titles/{slug}/catalog"})
		fieldMappingsHandler := handlers.NewFieldMappingsHandler(fieldMappingsRegistry, slog.Default())
		// V2 saisons : si le catalog est câblé, on enrichit le DTO assets
		// avec l'union TOML + DB (cf. SeasonsCatalogForHandler ci-dessous).
		if seasonsResolver := reg.SeasonsCatalogForHandler(); seasonsResolver != nil {
			fieldMappingsHandler = fieldMappingsHandler.WithSeasonsCatalog(seasonsResolver)
		}
		fieldMappingsHandler.Mount(r) // GET /titles/{slug}/field-mappings (Huma, ETag préservé)

		// Phase 1.7a — capabilities produit déclarées par le titre (TOML).
		capabilitiesHandler := handlers.NewCapabilitiesHandler(fieldMappingsRegistry, slog.Default())
		capabilitiesHandler.Mount(r)

		// Phase 1.7b — matrice de features (cascade capabilities → 3 états).
		featureMatrixHandler := handlers.NewFeatureMatrixHandler(fieldMappingsRegistry, slog.Default())
		featureMatrixHandler.Mount(r)

		// Phase H.bis — catalogue Playlists/Pairs/Maps (title-aware).
		// OpenReadWriteShared pour compatibilité avec les connexions RW existantes
		// (prestige presets, rank catalog) sur le même fichier DuckDB.
		if catalogMetaDB, err := platform_duckdb.OpenReadWriteShared(
			metadataDBPathFor(cfg),
		); err != nil {
			slog.Warn("catalog_meta_db_unavailable", "err", err)
		} else {
			// Handle RW persistant : le CatalogHandler le garde pour servir
			// /catalog/*. Tracker pour fermeture au shutdown via reg.Close(),
			// sinon fuite de refCount metadata (cf. INCIDENT_2026-05-21).
			reg.TrackMetadataHandle(catalogMetaDB)
			catalogH := handlers.NewCatalogHandler(platform_duckdb.NewCatalogRepo(catalogMetaDB, nil))
			catalogH.Mount(r) // /titles/{slug}/catalog/{playlists,pairs,maps}
		}

		slog.Info("multi_title_api_enabled",
			"slugs", fieldMappingsRegistry.Slugs(),
			"endpoints", []string{
				"/api/v1/titles/{slug}/field-mappings",
				"/api/v1/titles/{slug}/capabilities",
				"/api/v1/titles/{slug}/feature-matrix",
				"/api/v1/titles/{slug}/catalog/playlists",
				"/api/v1/titles/{slug}/catalog/pairs",
				"/api/v1/titles/{slug}/catalog/maps",
			},
		)
	}

	// Réhabilitation Lab (2026-06-14, PMT-14 volet C) : le backend du Lab
	// interne (handlers/service/provider + tests) existait mais n'était
	// JAMAIS monté ici → /lab/{resources,contracts,diagnostics} renvoyait
	// 404 en prod. La casse était masquée par les mocks MSW du front + les
	// tests chi-local du handler (aucun ne vérifiait l'intégration serveur).
	// L'accès reste gardé au niveau service (requireAccess → can_manage_instance).
	// Anti-régression : lab_routes_mounted_test.go (chi.Walk sur le vrai routeur).
	// Contracts = diff OpenAPI Go ↔ FastAPI legacy : MARQUÉ POUR RETRAIT
	// (PMT-14 volet C). Monté via Mount (resources + contracts + diagnostics
	// + waypoint, tous Huma).
	//
	// Explorateur d'API live (Lab, Stage 1b) : résout un token Spartan via
	// reg.AnyPlayerTokens (seam canonique) puis FetchAsset sur Discovery UGC.
	// Réutilise le pattern MapImageURLFetcher (supra). Injecté dans le service
	// pour le garder découplé de halo/auth + testable. Les erreurs d'appel
	// (404/auth/token absent) sont portées dans la réponse (ResolvedOK=false),
	// pas en erreur HTTP — le panneau affiche le détail.
	waypointExplore := func(ctx context.Context, q domain.LabWaypointQuery) (*domain.LabWaypointResponse, error) {
		assetType := halo.AssetType(q.Segment)
		lang := q.Lang
		if lang == "" {
			lang = "en-US"
		}
		titleSlug := ctxkeys.TitleSlug(ctx)
		resp := &domain.LabWaypointResponse{
			Segment:   q.Segment,
			Endpoint:  halo.AssetTypeToEndpoint[assetType],
			AssetID:   q.AssetID,
			VersionID: q.VersionID,
			Lang:      lang,
		}
		start := time.Now()
		tokens, terr := reg.AnyPlayerTokens(ctx)
		if terr != nil {
			resp.Error = "aucun token Spartan disponible : " + terr.Error()
			resp.LatencyMS = time.Since(start).Milliseconds()
			slog.WarnContext(ctx, "lab waypoint: token Spartan indisponible",
				"module", logging.ModuleLab, "segment", q.Segment, "asset_id", q.AssetID,
				"titleSlug", titleSlug, "err", terr)
			return resp, nil
		}
		asset, ferr := halo.NewHaloProvider().WithTokens(tokens).FetchAsset(
			ctx, assetType, titleSlug, q.AssetID, q.VersionID, lang)
		resp.LatencyMS = time.Since(start).Milliseconds()
		if ferr != nil {
			resp.Error = ferr.Error()
			slog.WarnContext(ctx, "lab waypoint: fetch échoué",
				"module", logging.ModuleLab, "segment", q.Segment, "asset_id", q.AssetID,
				"version_id", q.VersionID, "titleSlug", titleSlug,
				"duration_ms", resp.LatencyMS, "err", ferr)
			return resp, nil
		}
		if asset != nil {
			resp.ResolvedOK = true
			resp.AssetName = asset.PublicName
			resp.Description = asset.Description
			resp.ImageURL = asset.ImageURL
		}
		slog.InfoContext(ctx, "lab waypoint: exploration",
			"module", logging.ModuleLab, "segment", q.Segment, "asset_id", q.AssetID,
			"version_id", q.VersionID, "titleSlug", titleSlug,
			"resolved", resp.ResolvedOK, "duration_ms", resp.LatencyMS)
		return resp, nil
	}
	labHandler := handlers.NewLabHandler(
		service.NewLabService(cfg, lab_platform.NewProvider(cfg)).WithWaypointExplorer(waypointExplore))
	// Durcissement (2026-06-18) : le Lab (désormais sous l'Admin) est
	// un outil opérateur → gardé RequireAuth+RequireAdmin comme /admin/*. Auparavant
	// /lab/* n'était filtré qu'au niveau service (can_manage_instance, hardcodé true
	// au bootstrap → de fait ouvert à tout utilisateur connecté). Le gate service
	// subsiste comme kill-switch d'instance. Routes montées via Huma (Mount).
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		labHandler.Mount(r)
	})

	// Sprint 43 : changelog (markdown brut) — MIGRÉ vers Huma (Phase 3b),
	// enregistré en tête de bloc via registerChangelogHuma.

	// Aide : notes de version extraites du README (EN/FR).
	// P8.10 : la logique git + parsing markdown vit dans
	// service.ReleaseNotesService ; le handler ne fait que cache + I/O HTTP.
	releaseBuilder := service.NewReleaseNotesService(cfg.RepoRoot)
	help := handlers.NewHelpHandler(releaseBuilder, titlePkg.NewPathResolver(cfg.RepoRoot).CacheRootDir())
	help.Mount(r)

	// Sprint 14 : contexte de session
	sessionHandler := handlers.NewSessionHandler(sessionStore)
	sessionHandler.Mount(r)

	// Verrou « instance fermée » (lockdown) : effectif = env (LEVELUP_INSTANCE_LOCKED,
	// verrou forcé au boot) OU app_settings.instance_locked (mutable à chaud via
	// PATCH /settings admin). Résolu live pour refléter une bascule runtime.
	instanceLockedFn := func() bool {
		if cfg.InstanceLocked {
			return true
		}
		s, err := settingsStore.Load()
		return err == nil && s.InstanceLocked
	}

	// Sprint 15 : Device Code Flow + authentification Halo
	// D3 cohabitation (cf. SPRINT_XBOX_SSO §0bis) : en mode "xbox", la LinkStrategy
	// est XboxSSOLinkStrategy (login direct via XUID + création user si nouveau).
	// Hors mode xbox, c'est PasswordLinkStrategy (LinkIdentity sur user déjà connecté).
	authHandler := handlers.NewAuthHandler(sessionStore, attemptStore, cfg.DemoMode, tokenProvider)
	var xboxLinkStrategy auth_platform.LinkStrategy
	if cfg.AuthMode == "xbox" {
		// PR 2.5a : injection du MultiUserTokenStore pour persister les tokens RTA
		// après login (data/auth/watcher_tokens/{xuid}.json).
		watcherTokensDir := titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
		multiUserTokens := auth_platform.NewMultiUserTokenStore(watcherTokensDir)

		// PR 2.5b : daemonGetter retourne le daemon courant (capturé par closure).
		// Nil si le watcher n'est pas démarré (watcher_presence_enabled=false ou
		// pas de tokens initiaux). XboxSSOLinkStrategy résout lazy à chaque login.
		daemonGetter := func() service.WatcherDaemon {
			if daemon == nil {
				return nil
			}
			return daemon
		}
		xboxLinkStrategy = service.NewXboxSSOLinkStrategy(users).
			WithTokenStore(multiUserTokens).
			WithDaemonGetter(daemonGetter).
			WithInstanceLock(instanceLockedFn).
			WithInviteStore(invites).
			WithGroupStore(groupStore)
		authHandler.WithLinkStrategy(xboxLinkStrategy)
	} else {
		authHandler.WithUserStore(users)
	}
	authHandler.MountDeviceFlow(r) // POST /auth/device-flow/start + GET /auth/device-flow/{attempt_id} (Huma)

	// PR 4 — Authorization Code Flow SSO Xbox (UX redirect, plus aboutie que
	// le Device Code). Enregistré uniquement en mode "xbox" + redirect URI
	// configuré. Sans la config Azure (plateforme "Web" + redirect URI dans
	// le portail), /authorize retourne AADSTS50011.
	if cfg.AuthMode == "xbox" && cfg.OAuthRedirectURI != "" {
		xboxOAuthHandler := handlers.NewXboxOAuthHandler(sessionStore, tokenProvider, cfg.DemoMode, cfg.OAuthRedirectURI).
			WithLinkStrategy(xboxLinkStrategy).
			WithAuthStore(authStore).
			WithInviteStore(invites)
		// Lie les routes racine /auth/xbox/* déclarées avant ce groupe (cf. supra) :
		// le redirect_uri Azure pointe sur le chemin racine, pas /api/v1.
		xboxOAuthRoot = xboxOAuthHandler
		// Alias /api/v1 conservés (le front initie via /api/v1/auth/xbox/login).
		r.Get("/auth/xbox/login", xboxOAuthHandler.LoginRedirect)
		r.Get("/auth/xbox/callback", xboxOAuthHandler.Callback)
	}

	// Auth locale : login/register/logout (mode password).
	// D3 cohabitation : en mode "xbox", login réservé aux admins, register au bootstrap.
	userAuthHandler := handlers.NewUserAuthHandler(users, invites, sessionStore, cfg.RegistrationMode).
		WithAuthMode(cfg.AuthMode).
		WithInstanceLock(instanceLockedFn)
	userAuthHandler.Mount(r) // login/register/logout/password migrés vers Huma (Phase 3b)

	// Groupes/familles : gestion end-user (tout user authentifié + lié à une
	// identité Halo). Inviter à un groupe = générer un code "rejoindre le groupe"
	// (consommé via le login Xbox SSO, cf. XboxSSOLinkStrategy). RequireAuth seul
	// (pas RequireAdmin) : c'est de la fonction utilisateur, pas de l'ops.
	groupsHandler := handlers.NewGroupsHandler(groupStore, invites, users)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Get("/groups", groupsHandler.ListMyGroups)
		r.Post("/groups", groupsHandler.CreateGroup)
		r.Patch("/groups/{id}", groupsHandler.RenameGroup)
		r.Delete("/groups/{id}", groupsHandler.DeleteGroup)
		r.Post("/groups/{id}/invites", groupsHandler.GenerateInvite)
		r.Delete("/groups/{id}/members/me", groupsHandler.LeaveGroup)
		r.Delete("/groups/{id}/members/{xuid}", groupsHandler.RemoveMember)
	})

	// Admin : gestion utilisateurs + invitations (protégé par RequireAuth + RequireAdmin).
	adminHandler := handlers.NewAdminHandler(users, invites)
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		adminHandler.Mount(r) // users/invites (Huma, sous RequireAuth+RequireAdmin)
		// Intégrité des données : invariants du pipeline sync par joueur
		// (Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md). NoStore : le
		// résultat reflète l'état courant des DBs, jamais de cache.
		invariantsHandler := handlers.NewAdminInvariantsHandler(reg.RunDataInvariants)
		invariantsHandler.Mount(r.With(middleware.NoStore))
		// Contention DB (B-swap shared) : compteurs de swap RO↔RW pendant le
		// sync (cadence + lectures rejetées en 503). Lecture seule des
		// métriques expvar du sharedprovider. NoStore : état courant.
		contentionHandler := handlers.NewAdminDBContentionHandler(reg.DBContention)
		contentionHandler.Mount(r.With(middleware.NoStore))
		// Santé des tokens auth (MSAL / XSTS / Refresh) par joueur. Lecture
		// seule du MultiUserTokenStore (ADR 0023), sans refresh réseau.
		tokenHealthHandler := handlers.NewAdminTokenHealthHandler(reg.TokenHealth)
		tokenHealthHandler.Mount(r.With(middleware.NoStore))
		// Dashboard monitoring admin : overview/scheduler/convergence/jobs
		// + actions correctives (data-health run, cycle auto-sync forcé).
		// Cf. server_admin_monitoring.go.
		wire.MountAdminMonitoringRoutes(r, reg, autoSyncScheduler, jobStore, serverCtx)
		// Gestion des titres (PMT-14 volet A) : liste + détail (Status lifecycle
		// MT-22 enfin lu/exposé, capabilities + feature-matrix réutilisés de
		// 1.7a/b sans recalcul). Read-only, admin-gated, NoStore (reflète l'état
		// du registre des titres au boot).
		adminTitlesHandler := handlers.NewAdminTitlesHandler(titleRegistry, fieldMappingsRegistry, slog.Default())
		// /titles, /titles/{slug}, /titles/{slug}/toml-draft (Huma, NoStore).
		adminTitlesHandler.Mount(r.With(middleware.NoStore))
		// Diagnostic santé d'un titre (PMT-14 volet A — productise Phase 1.8) :
		// présence des mappings TOML + réalité DB (lignes des tables cœur),
		// read-only via port.TableInspector.
		adminTitleDiagHandler := handlers.NewAdminTitleDiagnosticHandler(
			service.NewTitleDiagnosticService(cfg.RepoRoot, platform_duckdb.NewTableInspector()).
				WithCapabilities(fieldMappingsRegistry),
			slog.Default(),
		)
		adminTitleDiagHandler.Mount(r.With(middleware.NoStore))
	})

	// Diagnostic — accessible en loopback (127.0.0.1) uniquement, sans auth.
	// Permet de comprendre pourquoi le scheduler ne sync pas un joueur sans
	// avoir à fouiller dans les logs serveur (raison du skip/failure par joueur).
	// Bloqué en non-loopback : retourne 403.
	if autoSyncScheduler != nil {
		autoSyncH := handlers.NewAdminAutoSyncHandler(autoSyncScheduler, cfg, tokenProvider)
		r.Route("/_diag/auto-sync", func(r chi.Router) {
			r.Use(middleware.LoopbackOnly)
			autoSyncH.Mount(r) // /snapshot, /run, /probe (sous LoopbackOnly)
		})
	}

	// Sprint 16 : Settings + Setup joueur
	// §4 plan Squad/Sessions : orchestrator recompute is_with_friends, déclenché
	// async sur diff friend_gamertags lors d'un PATCH /settings.
	friendsOrchestrator := service.NewFriendsOrchestratorService(cfg, func() ([]string, error) {
		s, err := settingsStore.Load()
		if err != nil {
			return nil, err
		}
		return s.FriendGamertags, nil
	}).WithNotifier(reg.NotificationsEmitter)
	settingsHandler := handlers.NewSettingsHandler(cfg, settingsStore, jobStore).
		WithFriendsOrchestrator(friendsOrchestrator).
		WithNotificationsEmitter(reg.NotificationsEmitter).
		WithBackupScheduler(backupScheduler)
	settingsHandler.Mount(r) // /settings + /settings/{media,sessions,backup}/...

	// ProfileService PARTAGÉ : writer UNIQUE de db_profiles.json. Le store
	// porte un verrou process par-instance → toutes les écritures (onboarding
	// setup ET réglages titre B.5) DOIVENT passer par la MÊME instance, sinon
	// deux read-modify-write concurrents pourraient s'écraser (lost update).
	profileService := service.NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot).
		WithDBEvictor(func(playerDBPath string) { platform_duckdb.EvictAndCloseCached(playerDBPath) })
	setupHandler := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, profileService)
	setupHandler.Mount(r) // /setup/players, /setup/smoke-test

	// Sprint 17 : Jobs longs persistants + sync initiale.
	// GET /jobs/{job_id} migré vers Huma (Phase 3b, shape path-param).
	registerJobsHuma(humaAPI, handlers.NewJobsHandler(jobStore))
	syncH := handlers.NewSyncHandler(cfg, settingsStore, jobStore, tokenProvider)
	// Branche le hook Prestige post-sync (best-effort, no-op si flag off ou bundle nil).
	if prestigeBundle != nil {
		syncH = syncH.WithPrestigeHook(prestigeBundle.RunPostSync)
	}
	// Branche la factory d'émetteurs de notifications (match_synced / sync_error).
	syncH = syncH.WithNotificationsEmitterFactory(reg.NotificationsEmitter)
	// Branche le hook delta-detection post-sync (season_pass_level / objective_completed / challenge_completed).
	syncH = syncH.WithPostSyncDeltaHook(wire.BuildPostSyncDeltaHook(reg))
	// Dédup cross-source (unification 2026-06-02) : le gate provient du
	// Coordinator partagé du watcher, exposé via le scheduler (main.go a injecté
	// autoScheduler.SyncGate). Si le watcher est désactivé, Gate() renvoie le
	// NopSyncGate par défaut → comportement legacy (lease seul rempart).
	if autoSyncScheduler != nil {
		syncH = syncH.WithSyncGate(autoSyncScheduler.Gate())
		// Alignement sync manuel ↔ auto-sync (2026-06-14) : le sync HTTP delta
		// (/sync/all, /players/{slug}/sync) construit son moteur via le MÊME
		// BuildEngine que l'auto-sync → même PooledHaloClient (pool de tokens
		// partagé, source unique ADR 0023), post-sync runner, batch queue.
		// L'auth ne dépend plus des HaloTokens de session (le store a le RT).
		syncH = syncH.WithEngineBuilder(autoSyncScheduler.BuildEngine)
	}
	// serverCtx (annulé au shutdown) : les syncs HTTP en dérivent leur bgCtx
	// pour être annulés proprement à l'arrêt (avant duckdb.CloseAll).
	syncH = syncH.WithServerContext(serverCtx)
	// D3-01 (revue 2026-06-01) : /sync/initial et /sync/all sont des opérations
	// admin/setup (sync d'un joueur arbitraire lu dans le body / de TOUS les
	// joueurs). Auparavant montées sous /api/v1 SANS auth → contournaient
	// l'ownership (ADR 0029) : n'importe qui pouvait déclencher le sync de
	// n'importe quel joueur. Protégées par RequireAuth + RequireAdmin comme le
	// groupe /admin. En mode demo/single-user les middlewares no-opent (onboarding
	// préservé) ; en multi-user, seul un admin peut déclencher.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		syncH.MountInitialAndAll(r) // POST /sync/initial, /sync/all (Huma)
		// Sprint 51-B3 : Pipeline backfill (weapon kills + détection des autres types).
		// Audit ownership 2026-06-08 (PR-A) : backfill lit player_slug dans le BODY
		// (hors chokepoint /players/{slug}) → opération lourde sur un joueur arbitraire.
		// Aligné sur /sync/* : admin-gated (no-op en demo/single-user).
		handlers.NewBackfillHandler(cfg, jobStore).Mount(r) // POST /backfill/start
	})

	// OpenSpartan import (Sprint OpenSpartan PR 3.B + commit 15 sprint B1) :
	// multipart upload du .db SQLite OpenSpartan, validation XUID via session
	// SSO, exécution asynchrone via jobStore. Skipped en mode démo.
	//
	// Acquisition shared à la demande via cfg.SharedProvider — plus de handle
	// RW persistant au boot. Empêche le conflit "different configuration"
	// (Provider tient shared en RO steady state, OpenReadWriteShared au boot
	// échouait). OpenSpartan import = flow d'onboarding rare (un user qui
	// importe ses données), pas de raison d'ouvrir RW au boot.
	if !cfg.DemoMode {
		osImportSvc := service.NewOpenSpartanImportService(cfg.SharedProvider, config.SharedDBPath(cfg, ""))
		osPostImportSvc := service.NewOpenSpartanPostImportService(cfg)
		osCfg := handlers.OpenSpartanImportConfig{
			ImportService:     osImportSvc,
			PostImportService: osPostImportSvc,
			JobStore:          jobStore,
			StashDir:          filepath.Join(cfg.RepoRoot, "data", "players"),
			DemoMode:          cfg.DemoMode,
		}
		// Trigger de convergence events immédiat post-import (réutilise le pool
		// d'auth du scheduler). Conditionnel : éviter un typed-nil dans l'interface
		// si le scheduler n'est pas câblé. nil → backfill repris au prochain cycle.
		if autoSyncScheduler != nil {
			osCfg.Convergence = autoSyncScheduler
		}
		osImportH := handlers.NewOpenSpartanImportHandler(osCfg)
		r.Post("/import/openspartan", osImportH.StartImport)
	}

	// Galerie médias — version de flux pour polling léger
	handlers.NewMediaFeedVersionHandler().Mount(r) // GET /media/feed-version (Huma)

	// Assets cache-aside unifiés (médailles, maps, battlepass, badges de défi).
	// Couche d'abstraction DefaultResolver : local-first → API-fallback + DuckDB index.
	r.Get("/assets/medals/{title_id}/{medal_id}/image", assetHandler.GetMedalImage)
	r.Get("/assets/maps/{title_id}/{map_id}/image", assetHandler.GetMapImage)
	r.Get("/assets/battlepass/{subdir}/*", assetHandler.GetBattlePassImage)
	r.Get("/assets/challenge-badge/{title_id}/{badge_id}", assetHandler.GetChallengeBadge)
	r.Get("/assets/spartan/{image_type}/{title_id}/*", assetHandler.GetSpartanImage)

	// Asset Drawer — toujours enregistré ; renvoie [] si metaDB indisponible (best-effort).
	// Les 2 branches sont sur Huma (Phase 3b) : handler réel gaté par capability, ou
	// fallback vide. if/else mutuellement exclusif → pas de double-registration.
	if assetMetaHandler != nil {
		assetMetaHandler.Mount(r) // GET /assets/{title_id}/{maps,weapons}
	} else {
		handlers.NewEmptyAssetMetadataHandler().Mount(r) // fallback Huma → []
	}

	// Sélection par titre (Pass B.5) : activer / mettre en pause / purger un
	// titre d'un joueur. Owner-gated SANS RequireActiveTitle (doit fonctionner
	// sur un titre coming_soon/archivé, ex. purger un jeu qu'on vient de
	// désactiver). TitleSlugFromPath aligne le titre du ctx sur le titre CIBLÉ
	// (param de path) afin que la garde d'ownership raisonne sur ce titre et
	// non sur le header (anti-bypass).
	r.Route("/profiles/{player_slug}/titles/{slug}", func(r chi.Router) {
		r.Use(middleware.TitleSlugFromPath("slug"))
		r.Use(middleware.RequirePlayerOwnership(cfg.DemoMode, cfg.AuthMode, playerOwnershipXUIDResolver(cfg), users, familyXUIDResolver(groupStore, users)))
		handlers.NewTitleSyncHandler(profileService).Mount(r)
	})

	// Endpoints P1 : pages par joueur (Sprint 37 — DI via wire.ServiceRegistry)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		// MT-22 (PMT-8) : gate du cycle de vie du titre. Un titre courant
		// coming_soon/archived/inconnu → 503 title_unavailable (machine-readable)
		// au lieu de servir des données. No-op aujourd'hui (seul halo_infinite
		// actif). Avant la garde de propriété : indisponibilité du titre =
		// plus fondamental que l'appartenance du joueur.
		r.Use(middleware.RequireActiveTitle(titleRegistry))

		// Budget d'attente court des lectures shared user-facing : une page échoue
		// vite (503 Retry-After) au lieu de pendre jusqu'à 30s quand un sync tient
		// le writer RW. Ne borne QUE l'attente d'un swap, pas l'exécution des
		// requêtes (cf. sharedprovider.WithSwapWaitBudget). Chokepoint unique des
		// routes player-scoped, donc couvre toutes les pages.
		r.Use(middleware.UserFacingReadBudget(0))

		// Couche A (ADR 0029) : garde de propriété joueur. Chokepoint unique —
		// 403 player_forbidden si l'utilisateur courant ne possède pas le slug.
		// Transparent en mode demo / auth non activée. Toute route player-scoped
		// DOIT rester montée sous ce groupe pour être protégée.
		r.Use(middleware.RequirePlayerOwnership(cfg.DemoMode, cfg.AuthMode, playerOwnershipXUIDResolver(cfg), users, familyXUIDResolver(groupStore, users)))

		filters := handlers.NewFiltersHandler(reg.Filters)
		filters.Mount(r)

		mh := handlers.NewMatchHistoryHandler(reg.MatchHistoryCtx)
		mh.Mount(r) // POST /pages/match-history/query (export CSV reste chi, plus bas)

		// P6.3 : guard de capability — career routes nécessitent CapCareer.
		// Migré vers Huma (Phase 3b) : Mount sur le sous-groupe capability
		// (humacore.NewAPI hérite du middleware RequireCapability + ownership).
		career := handlers.NewCareerHandler(reg.Career, reg.MatchHistoryCtx)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapCareer))
			career.Mount(r)
		})

		// Achievements (Xbox bilingues) : guard CapAchievements.
		achievements := handlers.NewAchievementsHandler(reg.Achievements)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapAchievements))
			achievements.Mount(r)
		})

		// Sprint 8 : Match View + Explorer
		// WithMediaURLs : transforme les chemins média de l'onglet médias en
		// URLs servables, comme la galerie (settingsStore + repoRoot déjà en portée).
		mv := handlers.NewMatchViewHandler(reg.MatchView).
			WithMediaURLs(settingsStore, cfg.RepoRoot)
		mv.Mount(r)

		// Canonical MatchEvents (Phase 3) : GET .../matches/{match_id}/events —
		// timeline d'events on-demand (kill-feed/timeline), capability-gated.
		handlers.NewMatchEventsHandler(reg.MatchEvents).Mount(r)

		// Phase 4 plan engagement : score + courbe par match + profil + timeseries + squad
		// + admin recompute. Toutes les routes sont gated par CapEngagement
		// (titre doit declarer la capability — halo_infinite=oui, autres=non
		// par defaut, degradation gracieuse via 503 capability_unavailable,
		// cf. middleware.RequireCapability).
		eng := handlers.NewEngagementHandler(reg.Engagement)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapEngagement))
			eng.Mount(r)
		})

		explorer := handlers.NewExplorerHandler(reg.ExplorerCtxWithAuth, reg.MatchHistoryCtx)
		explorer.Mount(r) // player-query + matches-query (2 routes)

		// Sprint 9 : Sessions
		sessions := handlers.NewSessionsHandler(reg.Sessions)
		sessions.Mount(r)
		sessionPage := handlers.NewSessionPageHandler(reg.SessionPage)
		sessionPage.Mount(r)

		// Sprint 10 : Stats/Séries temporelles
		stats := handlers.NewStatsHandler(reg.Stats)
		stats.Mount(r)

		// Sprint 11 : Accueil/Home + Battle Pass + Challenges (migrés Huma ;
		// en-têtes de cache ETag/max-age/no-store posés dans les Output).
		home := handlers.NewHomeHandler(reg.HomeCtxWithAuth, settingsStore)
		home.Mount(r)

		// Season Pass (palmares)
		seasonPass := handlers.NewSeasonPassHandler(reg.SeasonPassCtxWithAuth)
		seasonPass.Mount(r)

		// Relations (hub Communauté > Relations) — page transverse non gatée.
		relations := handlers.NewRelationsHandler(reg.RelationsCtx)
		relations.Mount(r)

		// Sprint 12 : Escouade | Sprint 55 D1 : Synthèse → handler autonome
		squad := handlers.NewSquadHandler(reg.SquadCtx)
		squad.Mount(r)

		// Phase 1 chunk S1b : Squad V2 (multi-coéquipiers, fondations Phase 0)
		squadV2 := handlers.NewSquadV2Handler(reg.SquadV2Ctx)
		squadV2.Mount(r)

		// Sprint 55 D1 : SynthesisHandler extrait de SquadHandler (frontière produit)
		synthesis := handlers.NewSynthesisHandler(reg.SynthesisCtx)
		synthesis.Mount(r)

		// Sprint 13 → Sprint 32 : Citations + Commendations + Médias → POST
		citations := handlers.NewCitationsHandler(reg.CitationsCtx)
		citations.Mount(r)

		// AXE B : Totaux à vie des commendations NATIVES (Halo 5). Title-agnostic
		// (loader type-asserté depuis l'adapter ; titre sans capability → vide).
		commendationTotals := handlers.NewCommendationTotalsHandler(reg.CommendationTotalsCtx)
		commendationTotals.Mount(r)

		// P6.3 : guard de capability — media routes nécessitent CapMedia.
		media := handlers.NewMediaHandler(reg.Media, reg.MediaUpload, cfg.RepoRoot).
			WithSettingsStore(settingsStore).
			WithDemoMode(cfg.DemoMode).
			WithAuthorsContext(reg.MediaPlayerCtx, func(_ context.Context, titleSlug string) ([]domain.PlayerSummary, error) {
				return cfg.LoadPlayers(titleSlug)
			}).
			WithNotificationsEmitterFactory(reg.NotificationsEmitter).
			WithMediaRecipientResolver(reg.MediaRecipientResolver(cfg))
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapMedia))
			// 5 routes JSON migrées vers Huma (pages/media, likes, match-candidates,
			// associate, authors). humacore.NewAPI(r) hérite de CapMedia + ownership/title.
			media.Mount(r)
			// /media/reassociate supprimé en revue 2026-04-29 P0.2 Q6 (doublon non utilisé,
			// le front consomme /media/associate seulement).
			// upload (multipart) + files (serve binaire) restent chi — hors scope JSON.
			r.Post("/media/upload", media.PostUploadMedia)
			r.Get("/media/files/*", media.ServeMediaFile)
		})

		// Sprint 32 : Match History export (CSV — reste chi, hors scope migration
		// JSON ; explorer matches-query est migré dans explorer.Mount ci-dessus).
		r.Get("/pages/match-history/export", mh.Export)

		// Sprint 33 : Teammates (contrat FastAPI)
		teammates := handlers.NewTeammatesHandler(reg.TeammatesCtx)
		teammates.Mount(r)

		// Sprint 33 : Timeseries (contrat FastAPI)
		timeseries := handlers.NewTimeseriesHandler(reg.Timeseries)
		timeseries.Mount(r)

		// Exclusion manuelle de matchs non pertinents
		// NOTE : GET /match-exclusions supprimé en revue 2026-04-29 P0.2 Q6
		// (orphelin côté front, vue admin jamais implémentée).
		excl := handlers.NewMatchExclusionHandler(reg.MatchExclusion)
		excl.Mount(r)

		// Système de notifications in-app (per-player).
		notifH := handlers.NewNotificationsHandler(reg.Notifications)
		notifH.Mount(r)

		// Couche progression V2 (Ascension) — streaks / records / milestones.
		// Cf. .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1.
		progressionResolve := func(ctx context.Context, slug string) (*platform_duckdb.PlayerDB, error) {
			return reg.Resolve()(ctx, slug)
		}
		progressionH := handlers.NewProgressionHandler(progressionResolve, titlePkg.DefaultSlug).
			WithDemoMode(cfg.DemoMode)
		progressionH.Mount(r)

		// Coach Advisor — proposals coach proactives (ADR 0020 Phase 9).
		// Resolver compose PlayerDB + bundles → coach_advisor.Service.
		coachResolve := func(ctx context.Context, slug string) (coach_advisor.Service, string, error) {
			pdb, err := reg.Resolve()(ctx, slug)
			if err != nil {
				return nil, "", err
			}
			if pdb == nil || pdb.Player == nil {
				return nil, "", nil
			}
			ab := reg.CoachAdvisorBundle()
			pb := reg.PrestigeBundle()
			if ab == nil || pb == nil {
				return nil, pdb.XUID, nil
			}
			prestigeSvc, perr := pb.ServiceForPlayer(ctx, slug)
			if perr != nil {
				return nil, pdb.XUID, nil
			}
			return ab.ServiceForPlayer(pdb, pb.TemplateRepoForCoach(), prestigeSvc), pdb.XUID, nil
		}
		coachH := handlers.NewCoachProposalsHandler(coachResolve, titlePkg.DefaultSlug)
		coachH.Mount(r)

		// PlayerProfile V1 (Ascension) — endpoint /profile complet.
		// Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §8.1.
		profileH := handlers.NewPlayerProfileHandler(progressionResolve, titlePkg.DefaultSlug)
		// V2 §2 : injection optionnelle du mapping awards→axes (Section A1 radar).
		// Chargement lazy depuis config/titles/{slug}/mappings/awards.toml.
		// Absence du fichier ou erreur de parse : log + fallback V1 silencieux.
		awardsPath := filepath.Join(cfg.RepoRoot, "config", "titles", titlePkg.DefaultSlug, "mappings", "awards.toml")
		if awardSet, err := mappings.LoadAwardsFromFile(awardsPath); err != nil {
			slog.Warn("player_profile_awards_load_failed", "path", awardsPath, "err", err)
		} else {
			slog.Info("player_profile_awards_loaded",
				"path", awardsPath, "awards_count", len(awardSet.All()))
			profileH = profileH.WithAwardMapping(awardSet)
		}
		profileH.Mount(r)

		// Pattern Engine v3 (PLAN_PATTERN_ENGINE.md phases 1-3).
		// GET /api/v1/players/{player_slug}/patterns?n=50
		// Le handler dépend de port.PatternsRepository : on adapte le
		// ProgressionResolver (→ PlayerDB) en résolveur de repo DuckDB.
		patternsRepoResolve := func(ctx context.Context, slug string) (port.PatternsRepository, error) {
			pdb, err := progressionResolve(ctx, slug)
			if err != nil {
				return nil, err
			}
			return platform_duckdb.NewPatternsRepo(pdb), nil
		}
		patternsH := handlers.NewPatternsHandler(patternsRepoResolve, titlePkg.DefaultSlug)
		patternsH.Mount(r)

		// ImprovementCampaign V1 — endpoints start/active/pause/close/abandon.
		// Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.5 + §5.1.
		campaignH := handlers.NewCampaignHandler(progressionResolve, titlePkg.DefaultSlug)
		campaignH.Mount(r)

		// Match favoris (shared_social.duckdb)
		fav := handlers.NewMatchFavoriteHandler(reg.Social)
		fav.Mount(r) // PATCH /matches/{match_id}/favorite

		// Module Prestige — routes derrière feature flag PRESTIGE_ENABLED.
		// Le bundle a été initialisé au boot ; si nil ou flag off, routes non montées.
		if prestigeBundle != nil && cfg.PrestigeEnabled {
			lazy := wire.NewLazyPrestigeService(prestigeBundle, nil, cfg.DemoMode)
			appPlayers := func(context.Context) ([]domain.PlayerSummary, error) {
				return cfg.LoadPlayers()
			}
			// Garde d'autorisation acteur (ADR 0029 étendu aux routes squad
			// top-level) : created_by/requested_by/user_id doivent désigner un
			// profil possédé par la session. Réutilise les primitives de
			// RequirePlayerOwnership. Transparent en demo / auth désactivée.
			squadXUIDResolve := playerOwnershipXUIDResolver(cfg)
			squadFamilyResolve := familyXUIDResolver(groupStore, users)
			squadActorGuard := func(ctx context.Context, actorSlug string) bool {
				if !authz.Enforced(cfg.DemoMode, cfg.AuthMode) {
					return true
				}
				sess := middleware.GetSession(ctx)
				if sess == nil {
					return false
				}
				xuid, found := squadXUIDResolve(ctx, actorSlug)
				if !found {
					return false
				}
				var fam map[string]bool
				if squadFamilyResolve != nil {
					fam = squadFamilyResolve(ctx)
				}
				return authz.CanAccessPlayer(true, authz.CurrentUser(sess, users), xuid, fam)
			}
			// Migré vers Huma (Phase 3b) : 26 routes (challenges/arcs/prestige/
			// templates/squads/squad-challenges/pilot-mode) via ph.Mount.
			ph := handlers.NewPrestigeHandler(lazy, appPlayers).WithActorGuard(squadActorGuard)
			// Migré vers Huma (Phase 3b) : tout passe par ph.Mount. Les 2 routes
			// squad de main (PATCH/DELETE /squads/{squad_id} = Rename/Delete) sont
			// portées dans ph.Mount côté handlers (prestige.go) → 26 + 2 = 28.
			ph.Mount(r)
			slog.Info("prestige_routes_mounted", "endpoints_count", 28)
		}

		// Sprint 54 : Compare joueur vs joueur
		compare := handlers.NewCompareHandler(reg.Compare)
		compare.Mount(r)

		// Sprint 54 : Classement CSR (Leaderboard)
		leaderboard := handlers.NewLeaderboardHandler(reg.Leaderboard)
		leaderboard.Mount(r)

		// Sync delta par joueur
		syncH.MountDelta(r) // POST /sync (Huma, sous /players/{player_slug})
	})

	// Endpoints P1 : répertoire gamertags
	// Sprint 49 : route inconditionnelle — retourne 503 si shared DB absente.
	// Migré vers Huma (Phase 3b, shape query-param ?q=).
	registerGamertagHuma(humaAPI, handlers.NewGamertagHandler(gamertagSvc))

	// Watcher présence Xbox RTA — RequireAuth + RequireAdmin.
	watcherAttempts := auth_platform.NewWatcherAttemptStore()
	watcherHandler := handlers.NewWatcherHandler(cfg, settingsStore, daemon, tokenProvider, watcherAttempts)
	r.Route("/watcher", func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		// status/auth-status/subscriptions/auth-start migrés vers Huma (watcherHandler.Mount).
		watcherHandler.Mount(r)
	})
	return xboxOAuthRoot
}
