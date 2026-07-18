// Package api — server_apiv1.go : assembleur DI de l'API v1 (montage /api/v1).
//
// EXEMPTION FICHIER (CLAUDE.md seuil ≤ 500 L — VF-10 / K3d) : ce fichier dépasse
// délibérément le seuil (~1290 L). C'est un assembleur d'injection de dépendances
// SÉQUENTIEL, sans logique métier : mountAPIV1 monte ~80 endpoints sur le routeur,
// buildAPIV1Deps construit et câble les services/handlers, mountSPA branche le
// fallback SPA. La longueur reflète la surface API (nombre de routes/services),
// pas une responsabilité mélangée — le découper en sous-fichiers arbitraires
// éparpillerait une liste de montage cohésive et lisible d'un bloc.
//
// CONDITION DE RE-DÉCOUPE : dès qu'une VRAIE logique métier s'y glisse (calcul,
// transformation, décision autre que du wiring/routing), l'extraire dans un
// service/handler dédié (internal/service, internal/api/handlers, internal/api/wire)
// — l'exemption ne couvre QUE l'assemblage DI, pas de la logique.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
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
	"levelup/go-api/internal/worldenrich"
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
	// Garde de propriété joueur (ADR 0029), construite UNE fois et réutilisée par
	// tous les groupes player-scoped : /players/{slug}, /profiles/.../titles et les
	// diagnostics par joueur (lot S — csr-coverage, progression). Transparente en
	// démo / auth non activée. Source unique (règle ≤2 copies) : ne pas ré-inliner.
	ownershipMW := middleware.RequirePlayerOwnership(cfg.DemoMode, cfg.AuthMode,
		playerOwnershipXUIDResolver(cfg), users, familyXUIDResolver(groupStore, users))
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
	// S6 (sécurité, lot S) : /healthz/home construit la home du joueur de la
	// session (banner, peaks CSR/LUSR, arme favorite = révélateur d'identité) →
	// RequireAuth. Pas de {player_slug} : la propriété est implicite (session). No-op
	// en démo / auth non activée (probe CI/dev inchangée).
	handlers.NewHealthHomeHandler(reg.HomeCtxWithAuth).Mount(
		r.With(middleware.NoStore, middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode)))

	// Phase 9 du plan pipeline CSR : diagnostic coverage CSR pour un joueur.
	// Permet de vérifier en 1 ligne si le pipeline a bien capturé les CSR
	// (matured + placement) ou s'il faut lancer un backfill.
	// S6 : révèle la couverture CSR d'un {player_slug} → RequireAuth + ownership.
	handlers.NewDiagCSRHandler(reg.CSRCoverageProvider).Mount(
		r.With(middleware.NoStore, middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode), ownershipMW))

	// Phase 4 plan stabilisation 2026-05-22 : diagnostic progression V2
	// (Ascension). Compte les rows dans streak/player_records/record_history/
	// milestone_earned + milestone_catalog. Permet de vérifier que
	// EvaluateProgressionAfterSync tourne bien sur l'auto-sync (avant Phase 4
	// ces tables restaient vides — cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED).
	// S6 : diagnostic par {player_slug} → RequireAuth + ownership.
	handlers.NewDiagProgressionHandler(reg.ProgressionDiagProvider).Mount(
		r.With(middleware.NoStore, middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode), ownershipMW))

	// ADR 0020 (coach→pont Prestige) : agrège prestige_telemetry par origine du
	// défi (coach / user / pilot_mode / unknown) → taux d'acceptation/complétion.
	// Permet de mesurer l'efficacité du coach proactif sans schéma dédié.
	// S6 : diagnostic par {player_slug} → RequireAuth + ownership.
	handlers.NewDiagPrestigeTelemetryHandler(reg.PrestigeTelemetryDiagProvider).Mount(
		r.With(middleware.NoStore, middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode), ownershipMW))

	// Fix 2026-05-30 : backfill progression V2 in-process. Force une
	// évaluation idempotente (streaks/records/milestones) pour un joueur
	// dont l'historique existe mais dont le pipeline post-sync n'avait
	// jamais abouti (incident timeout shared reader). Renvoie le diag
	// post-exécution.
	// S2 (sécurité, lot S) : POST /_admin/progression/backfill/{player_slug} MUTE
	// des données → RequireAuth + RequireAdmin (admin = accès à tous les joueurs).
	handlers.NewProgressionBackfillHandler(reg.ProgressionBackfillProvider).Mount(
		r.With(middleware.NoStore, middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode),
			middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode)))

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

	// Diagnostic d'instance (ex-Lab). A3.5 (DC-9, 2026-07-10) : le Lab est
	// retiré de l'app — seule la route GET /lab/diagnostics reste montée
	// (panneau parité + garde-fous médailles de l'onglet admin Données). Les
	// explorateurs /lab/{resources,contracts,waypoint} sont supprimés (workflow
	// de dev servi par les CLI + docs/RUNBOOK_ADD_TITLE.md). Gardé
	// RequireAuth+RequireAdmin (outil opérateur, durcissement 2026-06-18) ; le
	// gate service can_manage_instance subsiste comme kill-switch d'instance.
	// Anti-régression : lab_routes_mounted_test.go (chi.Walk sur le vrai routeur).
	labHandler := handlers.NewLabHandler(service.NewLabService(cfg, lab_platform.NewProvider(cfg)))
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

	// Diagnostic — loopback (127.0.0.1) uniquement, ET admin (S5, lot S :
	// défense en profondeur). /probe résout des tokens (sensibles) → n'est plus
	// accessible au seul fait d'être sur la loopback. Permet de comprendre pourquoi
	// le scheduler ne sync pas un joueur (raison du skip/failure). Non-loopback → 403 ;
	// non-admin → 401/403. No-op auth en démo / auth non activée (dev inchangé).
	if autoSyncScheduler != nil {
		autoSyncH := handlers.NewAdminAutoSyncHandler(autoSyncScheduler, cfg, tokenProvider)
		r.Route("/_diag/auto-sync", func(r chi.Router) {
			r.Use(middleware.LoopbackOnly)
			r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
			r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
			autoSyncH.Mount(r) // /snapshot, /run, /probe (loopback + admin)
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
	// Fraîcheur A4.2 : le runner monitoring lit l'âge du dernier backup depuis
	// le même scheduler (manifest duckdbbackup) — nil toléré (section absente).
	reg.WithBackupScheduler(backupScheduler)
	// S1 (sécurité, lot S) : /settings mute la config (PATCH + POST media/sessions/backup)
	// → RequireAuth + RequireAdmin. No-op en démo/single-user (cfg.DemoMode court-circuite
	// le middleware), donc le mode public/démo reste inchangé.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		settingsHandler.Mount(r) // /settings + /settings/{media,sessions,backup}/...
	})

	// ProfileService PARTAGÉ : writer UNIQUE de db_profiles.json. Le store
	// porte un verrou process par-instance → toutes les écritures (onboarding
	// setup ET réglages titre B.5) DOIVENT passer par la MÊME instance, sinon
	// deux read-modify-write concurrents pourraient s'écraser (lost update).
	profileService := service.NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot).
		WithDBEvictor(func(playerDBPath string) { platform_duckdb.EvictAndCloseCached(playerDBPath) })
	setupHandler := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore, profileService)
	// S8 (sécurité, lot S) : /setup/players (écrit db_profiles.json) et
	// /setup/smoke-test → RequireAuth par cohérence (gardes internes conservées).
	// No-op en démo / auth non activée.
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		setupHandler.Mount(r) // /setup/players, /setup/smoke-test
	})

	// Sprint 17 : Jobs longs persistants + sync initiale.
	// GET /jobs/{job_id} migré vers Huma (Phase 3b, shape path-param).
	// V3 (sécurité) : le statut de job expose PlayerSlug + type + messages d'erreur
	// (révélateur d'identité) → RequireAuth. Tous les jobs sont créés depuis des flux
	// déjà authentifiés → no-op en démo/single-user (cfg.DemoMode court-circuite le
	// middleware). L'API Huma est adossée à un sous-routeur gardé (humachi hérite du
	// middleware du sous-groupe, cf. registerGamertagHuma/Mount des handlers gardés).
	registerJobsHuma(
		newHumaAPI(r.With(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))),
		handlers.NewJobsHandler(jobStore))
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
		// S3 (lot S) : import MUTANT (upload d'un .db → shared_matches). La revue
		// exhaustive route→garde a trouvé cette route sur `r` nu (hors du groupe
		// RequireAuth+RequireAdmin fermé plus haut) : seul `!cfg.DemoMode` la
		// protégeait. RequireAuth par cohérence avec /setup et /sync (la validation
		// XUID interne via session SSO est conservée). No-op si auth non activée
		// (single-user) ; en multi-user, réservé aux utilisateurs connectés.
		r.With(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode)).
			Post("/import/openspartan", osImportH.StartImport)
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
		r.Use(ownershipMW)
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
		r.Use(ownershipMW)

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

		// Sprint 11 : Accueil/Home + Battle Pass (migrés Huma ; en-têtes de cache
		// ETag/max-age/no-store posés dans les Output). Le défunt GET /challenges
		// (doublon de collision avec le module Prestige) a été retiré : la home web
		// lit les défis via le payload season pass (pages/palmares/season-pass).
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
			WithProduction(cfg.IsProduction()).
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
			//
			// Câblage du player_slug du chemin dans le contexte de résolution
			// Prestige (prestigePlayerSlugCtx) : répare les routes {id} et clôt le
			// BOLA objet-level par isolation player DB. Rationale : voir le doc du
			// middleware (prestige_player_slug_mw.go). Les défis d'escouade
			// (shared_social) sont gardés à part par assertMemberUser.
			r.Group(func(r chi.Router) {
				r.Use(prestigePlayerSlugCtx)
				ph.Mount(r)
			})
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

// apiV1Inputs regroupe les entrées de buildAPIV1Deps (portée NewRouter).
type apiV1Inputs struct {
	serverCtx         context.Context
	cfg               *config.AppConfig
	bootRepo          port.BootstrapRepository
	bootSvc           *service.BootstrapService
	daemon            watcher.DaemonController
	tokenProvider     auth_platform.TokenProvider
	autoSyncScheduler *scheduler.AutoSyncScheduler
	backupScheduler   *duckdbbackup.Scheduler
	groupStore        *groupstore.GroupStore
	sessionStore      *session_platform.Store
	attemptStore      *auth_platform.AttemptStore
	settingsStore     *settings_platform.Store
	jobStore          *jobs_platform.Store
	users             *userstore.Store
	invites           *userstore.InviteStore
	titleRegistry     *titlePkg.Registry
}

//nolint:funlen,gocyclo // phase de construction DI + montage diag racine : séquentiel.
func buildAPIV1Deps(r chi.Router, in apiV1Inputs) apiV1Deps {
	serverCtx := in.serverCtx
	cfg := in.cfg
	bootRepo := in.bootRepo
	bootSvc := in.bootSvc
	daemon := in.daemon
	tokenProvider := in.tokenProvider
	autoSyncScheduler := in.autoSyncScheduler
	backupScheduler := in.backupScheduler
	groupStore := in.groupStore
	sessionStore := in.sessionStore
	attemptStore := in.attemptStore
	settingsStore := in.settingsStore
	jobStore := in.jobStore
	users := in.users
	invites := in.invites
	titleRegistry := in.titleRegistry

	fieldMappingsRegistry := mappings.NewRegistry()
	multiTitleSlugs := make([]string, 0)
	for _, td := range titleRegistry.NonArchived() {
		multiTitleSlugs = append(multiTitleSlugs, td.Slug)
	}
	if len(multiTitleSlugs) == 0 {
		multiTitleSlugs = []string{titlePkg.DefaultSlug}
	}
	for _, err := range fieldMappingsRegistry.LoadFromConfigDir(cfg.RepoRoot, multiTitleSlugs, slog.Default()) {
		slog.Warn("field_mappings_load_warning", "err", err)
	}

	// PMT-1 / MT-01 : câble le resolver d'hosts d'ingestion title-aware partagé.
	// Les clients d'ingestion (sync HaloAPIClient, platform/halo HaloProvider,
	// assets) routent désormais via [endpoints] de constants.toml (fallback const
	// Halo byte-identique pour halo_infinite).
	games.SetDefaultEndpointResolver(games.NewMappingsEndpointResolver(fieldMappingsRegistry, titlePkg.DefaultSlug))

	// PMT-5 / MT-06 : câble le resolver d'issues (outcome) title-aware partagé. Les
	// repos platform/duckdb bâtissent leurs agrégats wins/losses/draws via
	// [outcomes].raw_code (fallback littéral `outcome = N` byte-identique pour Halo).
	games.SetDefaultOutcomeResolver(games.NewMappingsOutcomeResolver(fieldMappingsRegistry, titlePkg.DefaultSlug))

	// PMT-12 / MT-21 : validateur boot des mappings TOML requis. Le required-set
	// est dérivé des capabilities du titre (RequiredTOMLFor). Un titre ACTIF à
	// moitié configuré fait fail-fast (os.Exit) ; un coming_soon/archived est
	// loggé mais non bloquant (il n'est de toute façon pas servable, cf. gate
	// RequireActiveTitle/PMT-8). Skip en DemoMode (tests/démo : repoRoot sans
	// config TOML, cf. buildTestRouter).
	if !cfg.DemoMode {
		for _, td := range titleRegistry.All() {
			errs := mappings.ValidateRequiredTOML(cfg.RepoRoot, td)
			if len(errs) == 0 {
				continue
			}
			for _, e := range errs {
				if m, ok := e.(mappings.MissingRequiredTOML); ok {
					slog.ErrorContext(serverCtx, "required_toml_missing",
						"title", td.Slug, "path", m.Path, "required_by", m.RequiredBy)
				} else {
					slog.ErrorContext(serverCtx, "required_toml_missing", "title", td.Slug, "err", e)
				}
			}
			if td.IsActive() {
				slog.ErrorContext(serverCtx, "boot_validation_failed", "title", td.Slug, "errors_count", len(errs))
				os.Exit(1)
			}
		}
	}

	// Phase B multi-titres : resolver d adapters par titre + catalogues de rang
	// (extraits en buildTitleRuntime, K2a).
	tr := buildTitleRuntime(serverCtx, cfg, titleRegistry, fieldMappingsRegistry)
	titleResolver := tr.resolver
	hiRanks := tr.hiRanks
	rankImageURLsByTitle := tr.rankImages
	hiCaps := tr.hiCaps
	hiAssetURL := tr.hiAssetURL

	// P8.3 (revue 2026-04-29, ADR 0009) : error_tracker.Middleware retiré.
	// L'alerting Discord 500 / taux d'erreur n'est pas souhaité (commentaire
	// explicite en code). Code mort supprimé pour éliminer la confusion.

	// Sprint 37 : wire.ServiceRegistry — câblage par injection de dépendances.
	// titleResolver est attaché pour que les services puissent résoudre les
	// SemanticAdapter (libellés rangs etc.) selon le titre courant.
	// Overrides de libellé de playlist PAR TITRE (playlist_labels.toml), chargés
	// au boot depuis le registre des mappings. Ex. Halo 5 « Super Fiesta Fête » →
	// « Super Fiesta ». Titre sans fichier → absent de la map → no-op.
	playlistLabelOverrides := make(map[string]map[string]string)
	for _, slug := range multiTitleSlugs {
		if plset, ok := fieldMappingsRegistry.GetPlaylistLabels(slug); ok {
			playlistLabelOverrides[slug] = plset.OverridesMap()
		}
	}

	reg := wire.NewServiceRegistry(cfg, tokenProvider).
		WithTitleResolver(titleResolver).
		WithCapabilities(hiCaps).
		WithSettingsStore(settingsStore).
		WithRankCatalog(hiRanks).
		WithRankImageURLsByTitle(rankImageURLsByTitle).
		WithPlaylistLabelOverrides(playlistLabelOverrides)

	// MT-09 (PMT-12) : factory player-scoped de Halo enregistrée par SLUG (clé de
	// map, pas de comparaison littérale). Le builder lit reg.HiCapabilities() à la
	// volée (posé ci-dessus). Un 2e titre enregistrerait ICI son propre builder,
	// sans toucher aux factories dataAdapterForPDB/TitleDataAdapter.
	reg.RegisterPlayerDataBuilder(titlePkg.DefaultSlug, func(pdb *platform_duckdb.PlayerDB) games.TitleDataAdapter {
		a := halo_games.NewDataAdapter(platform_duckdb.NewCareerRepo(pdb), slog.Default())
		if reg.HiCapabilities() != nil {
			a = a.WithCapabilities(reg.HiCapabilities())
		}
		// HIGH-B : sources Explorer canonical-typées (profil de combat récent +
		// agrégat sample stats). Le même ExplorerRepo satisfait les 2.
		explorerRepo := platform_duckdb.NewExplorerRepo(pdb, pdb.XUID)
		a = a.WithRecentSource(explorerRepo).
			WithParticipantSource(explorerRepo).
			WithCrossPlayerSource(explorerRepo)
		// Canonical MatchEvents (Phase 2) : timeline reconstruite depuis
		// highlight_events + match_registry (T0) via la player DB courante.
		a = a.WithEventsSource(platform_duckdb.NewMatchEventsSource(pdb))
		return a
	})

	// Activation 1b multi-titre : enregistre les adapters des titres additionnels
	// ACTIFS (≠ défaut), pilotée par le registre. NO-OP tant qu'aucun 2e titre
	// n'est actif → Halo Infinite reste l'unique chemin (byte-identique). Cf.
	// server_titles_additional.go (gating registry-driven, jamais par slug).
	wire.RegisterAdditionalTitles(titleRegistry, titleResolver, reg, fieldMappingsRegistry)

	// Module Prestige — initialisation du bundle (best-effort, désactivable via flag).
	// Charge tuning.toml + templates + preset arcs Halo, ouvre shared_social et metadata.
	// Si le flag PRESTIGE_ENABLED est désactivé, les routes ne sont pas montées et
	// le sync hook est no-op — mais le boot du bundle reste utile pour valider la
	// config au démarrage.
	var prestigeBundle *wire.PrestigeBundle
	if pb, err := wire.NewPrestigeBundle(cfg.RepoRoot, reg.Resolve(), cfg.PrestigeEnabled); err != nil {
		slog.Warn("prestige_bundle_init_failed", "err", err)
	} else {
		prestigeBundle = pb.WithSquadProfile(wire.NewSquadPerfProfileProvider(
			func() ([]domain.PlayerSummary, error) { return cfg.LoadPlayers() },
			reg.Resolve(),
			titlePkg.DefaultSlug,
		))
		// Phase 2 plan stabilisation 2026-05-22 : enregistrer le bundle sur
		// le registry pour fermeture au shutdown (évite la fuite de refCount
		// sur metadata.duckdb qui causait le verrou au hot-reload Air).
		reg.WithPrestigeBundle(pb)
	}

	// Coach Advisor bundle (Phase 8 ADR 0020) — charge la grammaire de
	// synthèse une fois au boot. Pas de DB-handle à fermer ; toujours non-nil
	// (fallback grammaire vide si TOML absent, synthèse désactivée mais
	// matching catalogue reste fonctionnel).
	reg.WithCoachAdvisorBundle(wire.NewCoachAdvisorBundle(cfg.RepoRoot))

	// MultiUserTokenStore (ADR 0023) — source unique des tokens auth (RT + MSAL).
	// refreshTokensFromDB le lit AVANT de tomber sur les fallbacks legacy
	// (sync_meta DuckDB + env var). Idempotent : peut être re-créé à chaque boot
	// (pointe sur le même répertoire `data/auth/watcher_tokens/`).
	authStore := auth_platform.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	reg.WithAuthStore(authStore)

	// Transcoding média HLS asynchrone : le registry partage le jobStore process-wide
	// (cf. service.WithMediaTranscoding, injecté dans MediaUpload).
	reg.WithJobStore(jobStore)

	var gamertagSvc port.GamertagSearchService
	// Sprint B1 commit 11a : route via cfg.SharedProvider (sharedprovider.Provider)
	// au lieu d'un OpenReadOnly direct. Élimine le dernier handle RO non-coordonné
	// qui pinnait le fichier shared pendant les swaps RW du sync engine (bug
	// latent : le handle dédié à gamertag search bloquait Provider.AcquireWriter
	// avec "Unique file handle conflict" et faisait régresser le bug Madina97294).
	// En mode kill-switch (LEVELUP_USE_SHARED_PROVIDER=0), cfg.SharedProvider
	// reste un LegacySharedReader wrappant le handle global — comportement
	// identique au pre-sprint.
	if cfg.SharedProvider != nil {
		local := platform_duckdb.NewGamertagRepo(cfg.SharedProvider)
		// Fallback live (joueur JAMAIS croisé) : résolveur PeopleHub→profil Xbox
		// construit depuis le pool de tokens (db_profiles). OFF en démo/offline (pas
		// de tokens) → recherche purement locale. Instance partagée avec
		// Explorer/Compare via reg (cache mutualisé).
		var liveResolver service.GamertagXUIDResolver
		if !cfg.DemoMode {
			if dr, derr := worldenrich.BuildDirectoryResolver(cfg); derr != nil {
				slog.Warn("directory live resolver indisponible — recherche live désactivée", "err", derr)
			} else {
				liveResolver = dr
				reg.WithLiveGamertagResolver(dr)
			}
		}
		gamertagSvc = service.NewLiveFallbackGamertagSearch(local, liveResolver)
	} else {
		slog.Warn("shared DB unavailable for gamertag search — cfg.SharedProvider nil")
	}

	// AssetHandler — couche d'abstraction unifiée (local-first → API-fallback).
	// Le resolver est créé ici pour accéder à reg.AnyPlayerTokens.
	// Il est aussi passé au wire.ServiceRegistry pour que les HaloProviders délèguent
	// le cache/fetch des définitions BP/challenges au resolver (P4/P5).
	assetCfg := assets.AssetConfig{
		CacheRootDir:  titlePkg.NewPathResolver(cfg.RepoRoot).CacheRootDir(),
		MetaDBPath:    metadataDBPathFor(cfg),
		TokenProvider: reg.AnyPlayerTokens,
		// Résolution d'image de carte inconnue (KindMapImage) via DiscoveryUGC.
		// Injectée ici (et non dans internal/assets) pour éviter le cycle
		// assets→halo : réutilise halo.FetchAsset (→ DiscoveryAsset.ImageURL).
		// Tokens Spartan+clearance via reg.AnyPlayerTokens. Résultat caché par le
		// resolver → fetch ~1×/carte inconnue.
		MapImageURLFetcher: func(ctx context.Context, titleID, mapID, versionID string) (string, error) {
			tokens, terr := reg.AnyPlayerTokens(ctx)
			if terr != nil {
				return "", terr
			}
			a, ferr := halo.NewHaloProvider().WithTokens(tokens).FetchAsset(
				ctx, halo.AssetTypeMap, titleID, mapID, versionID, "en-US")
			if ferr != nil {
				return "", ferr
			}
			if a == nil {
				return "", nil
			}
			return a.ImageURL, nil
		},
	}
	assetResolver, err := assets.New(assetCfg)
	if err != nil {
		slog.ErrorContext(serverCtx, "assets resolver non disponible — arrêt du serveur", "err", err)
		os.Exit(1)
	}
	assetHandler := handlers.NewAssetHandler(assetResolver)
	reg.WithAssetResolver(assetResolver)

	// V2 saisons — résolveur unifié (TOML + DB live + lazy fetch via Waypoint).
	// Pattern symétrique à season_pass_service : DB d'abord, fetch + persist
	// si vide, merge avec le TOML (libellés FR + display_order). Le metaDB
	// reste ouvert pour la durée du process (OpenReadWriteShared = pool
	// reference-counted, le close de cmd/server décrémente).
	if seasonsAssets, ok := fieldMappingsRegistry.GetAssets(titlePkg.DefaultSlug); ok {
		// La DB metadata est OPTIONNELLE : si elle est indisponible (verrou RW pris
		// par un autre process) ou si EnsureSeasonTables échoue, on câble quand même
		// le catalogue en TOML-seul (repo=nil → Load court-circuite DB + provider).
		// Sinon les saisons du TOML sont perdues et le breakdown « matchs par saison »
		// (Explorer) disparaît silencieusement.
		//
		// NB : seasonsRepo est typé en interface (port.MetadataRepository), PAS en
		// *MetadataRepo concret — un pointeur concret nil emballé dans une interface
		// n'est pas == nil, ce qui ferait échouer le court-circuit repo==nil du
		// catalogue (piège classique Go).
		var seasonsRepo port.MetadataRepository // nil ⇒ catalogue TOML-seul
		seasonsMetaPath := metadataDBPathFor(cfg)
		if seasonsMetaDB, err := platform_duckdb.OpenReadWriteShared(seasonsMetaPath); err != nil {
			slog.Warn("seasons_catalog_meta_db_unavailable",
				"err", err, "fallback", "static_toml_only")
		} else {
			candidateRepo := platform_duckdb.NewMetadataRepoFromDB(seasonsMetaDB)
			// Tables idempotentes : la migration peut ne pas avoir tourné encore.
			if ensureErr := candidateRepo.EnsureSeasonTables(context.Background()); ensureErr != nil {
				// Handle ouvert mais inexploitable : le fermer pour ne pas fuiter le
				// refCount du pool (cf. INCIDENT_2026-05-21). Repli en TOML-seul.
				_ = seasonsMetaDB.Close()
				slog.Warn("seasons_catalog_ensure_tables_failed",
					"err", ensureErr, "fallback", "static_toml_only")
			} else {
				// Handle RW persistant sur metadata.duckdb : le SeasonsCatalog le
				// garde pour la vie du process. Tracker pour fermeture au shutdown
				// via reg.Close(), sinon fuite de refCount (cf. INCIDENT_2026-05-21).
				reg.TrackMetadataHandle(seasonsMetaDB)
				seasonsRepo = candidateRepo
			}
		}
		// Toujours câbler le catalogue : repo nil ⇒ TOML-seul. db_wired distingue le
		// mode complet (DB fraîche + lazy fetch) du mode dégradé (TOML-seul) en prod.
		catalog := service.NewSeasonsCatalog(seasonsAssets, seasonsRepo, halo.DefaultHaloProvider, slog.Default())
		reg.WithSeasonsCatalog(catalog)
		slog.Info("seasons_catalog_ready",
			"title_slug", titlePkg.DefaultSlug,
			"toml_count", len(seasonsAssets.AllOfKind("season")),
			"db_wired", seasonsRepo != nil,
		)
	}

	// Assets du titre additionnel (Halo 5) chargés UNE FOIS depuis sa metadata isolée —
	// réutilisés par l'AssetMetadataHandler ET l'adapter TitleAssetURLAdapter ci-dessous
	// (K1g : supprime le double chargement metadata h5 au boot). Best-effort : nil si pas de seed.
	h5Maps, h5Weapons, h5Medals := loadTitleAssetDrawerData(
		config.MetadataDBPath(cfg, halo5.TitleSlug), halo5.TitleSlug)

	// AssetMetadataHandler (drawer) — construit hors NewRouter (K2a). nil si metadata
	// indisponible après retries (non fatal).
	assetMetaHandler := buildAssetMetadataHandler(cfg, hiAssetURL, titleRegistry, h5Maps, h5Weapons, h5Medals)

	// Résolveurs d'assets title-aware Halo 5 (badge CSR + AssetURLAdapter + sprites
	// médailles), extraits en helper (K2a). Best-effort : nil/vide → HINF inchangé.
	wireHalo5AssetAdapters(cfg, titleResolver, h5Maps, h5Weapons, h5Medals)

	// Fichiers statiques (images maps, médailles, armes…)
	staticDir := filepath.Join(cfg.RepoRoot, "static")
	// Handler spécial pour /static/commendations/* : fallback vers noms URL-encodés
	// pour les fichiers dont le nom décodé contient des caractères interdits Windows (ex: ?).
	r.Handle("/static/commendations/*", newCommendationHandler(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel).
	//
	// P8.11 (revue 2026-04-29 axe 8 amende) : sémantique séparée
	// liveness vs readiness pour orchestrateurs K8s/LB multi-user.
	//   - /health   : Deprecated, mixte (200 si DB OK), gardé en rétrocompat.
	//   - /healthz  : liveness — process vivant, 0 I/O DB, latence < 5ms.
	//   - /readyz   : readiness — vérifie DuckDB + fs, retourne 503 si un check KO.
	// Outillage média sondé UNE fois au boot (borné : 3 execs ffmpeg) puis figé
	// dans le handler — /health ne réexécute jamais ffmpeg par requête. Rend la
	// disponibilité observable en prod malgré LEVELUP_LOG_LEVEL=warn qui masque
	// la ligne INFO du démarrage.
	mediaCtx, mediaCancel := context.WithTimeout(serverCtx, 5*time.Second)
	mediaStatus := ops.InspectMediaTooling(mediaCtx).ToHealthStatus()
	mediaCancel()
	healthH := handlers.NewHealthHandlerWithVersion(bootRepo, cfg.AppVersion).
		WithMediaTooling(mediaStatus)
	healthH.Mount(r) // /health, /healthz, /readyz (racine, Huma)

	// P8.3 (revue 2026-04-29, ADR 0009) : monitoring expvar minimal.
	// Expose /debug/vars (stdlib) avec les compteurs LevelUp publiés sous la
	// clé "levelup". Pas de Prometheus/OpenTelemetry — observability basique
	// pour multi-user. Les hot paths sont instrumentés progressivement via
	// observability.RecordDurationMS / IncCounter.
	//
	// P8.3 finalisé : protégé derrière RequireAuth + RequireAdmin (transparent
	// en mode démo / auth=none, refus 403 sinon).
	//
	// J1 (ADR 0009) : publie les sql.DBStats des pools DuckDB sous
	// "levelup"/"duckdb_pool_stats" (WaitCount/WaitDuration = contention pool).
	observability.PublishDuckDBPoolStats(func() any { return platform_duckdb.PoolStatsSnapshot() })
	// J2 : publie les bornes ressources (memory_limit/threads/pool) sous
	// "levelup"/"duckdb_budgets" — rend la config mémoire appliquée observable.
	observability.PublishDuckDBBudgets(func() any { return platform_duckdb.BudgetsSnapshot() })
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		r.Mount("/debug/vars", http.DefaultServeMux)
	})

	// PR 4 (fix routing OAuth) — Le redirect_uri Azure pointe sur le chemin racine

	return apiV1Deps{
		cfg:                   cfg,
		bootSvc:               bootSvc,
		reg:                   reg,
		fieldMappingsRegistry: fieldMappingsRegistry,
		attemptStore:          attemptStore,
		users:                 users,
		invites:               invites,
		sessionStore:          sessionStore,
		tokenProvider:         tokenProvider,
		groupStore:            groupStore,
		settingsStore:         settingsStore,
		assetHandler:          assetHandler,
		assetMetaHandler:      assetMetaHandler,
		gamertagSvc:           gamertagSvc,
		prestigeBundle:        prestigeBundle,
		serverCtx:             serverCtx,
		daemon:                daemon,
		autoSyncScheduler:     autoSyncScheduler,
		authStore:             authStore,
		jobStore:              jobStore,
		titleRegistry:         titleRegistry,
		backupScheduler:       backupScheduler,
	}
}

// immutableAssetCacheControl : politique de cache des assets Vite au nom hashé.
// max-age 1 an + immutable — sûr car le hash de contenu change à chaque build,
// donc l'URL elle-même est un cache-buster (le navigateur ne revalide jamais).
const immutableAssetCacheControl = "public, max-age=31536000, immutable"

// viteHashedAssetPattern reconnaît les fichiers émis par Vite avec un hash de
// contenu : `/assets/<nom>-<hash>.<ext>`. Le hash par défaut de Vite fait 8
// caractères base64url ([A-Za-z0-9_-]) ; on exige >= 8 pour ne PAS confondre un
// simple nom à tirets (`/assets/mon-fichier.css`) avec un asset hashé. index.html
// et tout fichier non hashé (racine, /favicon.ico…) ne matchent pas → jamais
// marqués immutable (ils DOIVENT rester revalidables pour livrer un nouveau build).
var viteHashedAssetPattern = regexp.MustCompile(`^/assets/.+-[A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`)

// isViteHashedAsset indique si le chemin URL cible un asset Vite au nom hashé.
func isViteHashedAsset(urlPath string) bool {
	return viteHashedAssetPattern.MatchString(urlPath)
}

// serveStaticFile délègue au FileServer (qui pose ETag/Last-Modified/Content-Type)
// après avoir ajouté Cache-Control: immutable UNIQUEMENT pour les assets hashés.
// Le header est posé AVANT ServeHTTP : il survit aussi bien au 200 qu'au 304.
func serveStaticFile(w http.ResponseWriter, req *http.Request, fileServer http.Handler) {
	if isViteHashedAsset(req.URL.Path) {
		w.Header().Set("Cache-Control", immutableAssetCacheControl)
	}
	fileServer.ServeHTTP(w, req)
}

// mountSPA sert le build Vite (LEVELUP_WEB_DIST) en catch-all /* : un fichier du
// dist servi tel quel, sinon index.html (route client-side React) avec injection
// Open Graph. Inactif si WebDistDir vide ou index.html absent. Extrait de NewRouter (K2a).
func mountSPA(r chi.Router, serverCtx context.Context, cfg *config.AppConfig, reg *wire.ServiceRegistry) {
	if dist := cfg.WebDistDir; dist != "" {
		indexPath := filepath.Join(dist, "index.html")
		if _, statErr := os.Stat(indexPath); statErr == nil {
			fileServer := http.FileServer(http.Dir(dist))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				if fi, err := os.Stat(filepath.Join(dist, filepath.Clean(req.URL.Path))); err == nil && !fi.IsDir() {
					serveStaticFile(w, req, fileServer)
					return
				}
				reg.ServeIndexWithOG(w, req, indexPath)
			})
			slog.InfoContext(serverCtx, "SPA: front React servi depuis le dist", "dir", dist)
		} else {
			slog.WarnContext(serverCtx, "LEVELUP_WEB_DIST défini mais index.html introuvable — SPA non montée", "dir", dist)
		}
	}
}
