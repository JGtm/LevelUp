// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
// Sprint 37 : Architecture handlers & injection DI via ServiceRegistry.
// Sprint 40 : ContractValidate (dev). ErrorTracker retiré P8.3 (ADR 0009).
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	// Blank import : déclenche l'init() de observability qui publie le namespace
	// expvar "levelup". Le handler /debug/vars (stdlib) découvre ces compteurs
	// automatiquement via http.DefaultServeMux (P8.3, ADR 0009).
	_ "levelup/go-api/internal/observability"
	auth_platform "levelup/go-api/internal/platform/auth"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	gitcli "levelup/go-api/internal/platform/git"
	"levelup/go-api/internal/platform/halo"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
)

// NewRouter construit le routeur chi avec tous les endpoints.
// Construction par injection de dépendances — pas d'état global.
// daemon peut être nil si le watcher n'est pas actif au démarrage.
// tokenProvider peut être nil : MSALProvider est utilisé par défaut.
// Retourne aussi le *ServiceRegistry pour permettre au démon watcher de lier le TTL dynamique.
func NewRouter(
	cfg *config.AppConfig,
	bootRepo port.BootstrapRepository,
	bootSvc *service.BootstrapService,
	daemon watcher.DaemonController,
	tokenProvider auth_platform.TokenProvider,
) (http.Handler, *ServiceRegistry) {
	if tokenProvider == nil {
		tokenProvider = auth_platform.NewMSALProvider()
	}
	// Sprint 14 : session store + Sprint 15 : attempt store auth
	isProduction := cfg.SessionSecret != "CHANGE_ME_IN_PRODUCTION" // pragma: allowlist secret
	sessionStore := session_platform.NewStore(cfg.SessionDir, session_platform.DefaultTTL, cfg.SessionSecret)
	attemptStore := auth_platform.NewAttemptStore()

	// Sprint 16 : settings store + Sprint 17 : job store
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobsPath := filepath.Join(cfg.RepoRoot, "data", "cache", "jobs.json")
	jobStore := jobs_platform.NewStore(jobsPath)

	// Auth locale : user store + invite store (mode password).
	usersPath := filepath.Join(cfg.AuthDir, "users.json")
	invitesPath := filepath.Join(cfg.AuthDir, "invites.json")
	users := userstore.NewStore(usersPath)
	invites := userstore.NewInviteStore(invitesPath)

	r := chi.NewRouter()

	// Middlewares transverses (ordre important)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.CSRF(cfg.CORSOrigins))
	r.Use(middleware.RateLimit(cfg.DemoMode))
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.WithSession(sessionStore, isProduction))

	// Sprint 44 : TitleExtractor — injecte title_slug dans le contexte.
	titleRegistry := titlePkg.NewRegistry()
	r.Use(middleware.TitleExtractor(titleRegistry))

	// Phase A multi-titres : chargement des FieldMappingSet TOML par titre.
	// Erreur de chargement → log mais ne bloque pas le boot (les autres titres
	// restent disponibles). L'endpoint /field-mappings n'est exposé que si le
	// flag MULTI_TITLE_API_ENABLED est activé.
	fieldMappingsRegistry := mappings.NewRegistry()
	multiTitleSlugs := []string{titlePkg.DefaultSlug}
	for _, err := range fieldMappingsRegistry.LoadFromConfigDir(cfg.RepoRoot, multiTitleSlugs, slog.Default()) {
		slog.Warn("field_mappings_load_warning", "err", err.Error())
	}

	// Phase B multi-titres : resolver des adapters par titre.
	// Le resolver est exposé aux services produit qui veulent consommer la
	// couche canonique.
	titleResolver := games.NewStaticResolver(titlePkg.DefaultSlug)
	if hiFields, ok := fieldMappingsRegistry.Get(titlePkg.DefaultSlug); ok {
		// Charger le catalog des rangs HI depuis metadata.duckdb (career_rank_translations).
		// OpenReadWriteShared est cached par path → réutilise le pool existant ouvert dans
		// cmd/server. IMPORTANT : Close() pour décrémenter le refCount sinon le sql.DB
		// reste ouvert au shutdown (le metaDB.Close() de cmd/server décrémente seulement
		// d'un cran), ce qui retient le HANDLE Windows et provoque le verrou
		// "metadata verrouillée" au prochain hot-reload Air.
		var hiRanks *mappings.RankCatalog
		hiMetaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
		if metaDB, err := platform_duckdb.OpenReadWriteShared(hiMetaPath); err == nil {
			if catalog, err := platform_duckdb.LoadRankCatalog(context.Background(), metaDB); err == nil {
				hiRanks = catalog
				slog.Info("rank_catalog_loaded", "title_slug", titlePkg.DefaultSlug, "ranks", catalog.Len())
			} else {
				slog.Warn("rank_catalog_load_failed", "err", err.Error())
			}
			if closeErr := metaDB.Close(); closeErr != nil {
				slog.Warn("rank_catalog_meta_db_close_failed", "err", closeErr.Error())
			}
		} else {
			slog.Warn("rank_catalog_meta_db_open_failed", "err", err.Error())
		}
		hiAssets, _ := fieldMappingsRegistry.GetAssets(titlePkg.DefaultSlug)
		hiOutcomes, _ := fieldMappingsRegistry.GetOutcomes(titlePkg.DefaultSlug)
		if sem := halo_games.NewSemanticAdapter(hiFields, hiRanks, hiAssets, hiOutcomes); sem != nil {
			titleResolver.RegisterSemantic(sem)
			// Plan multi-titres §8.1 : event adapter_loaded au boot du semantic adapter.
			slog.Info("adapter_loaded",
				"title_slug", sem.TitleSlug(),
				"kind", "semantic",
				"schema_version", sem.SchemaVersion(),
				"assets_loaded", hiAssets != nil,
				"outcomes_loaded", hiOutcomes != nil,
				"ranks_count", hiRanks.Len(),
			)
		} else {
			slog.Error("adapter_load_failed",
				"title_slug", titlePkg.DefaultSlug,
				"kind", "semantic",
				"reason", "fields_mapping_set_nil",
			)
		}
	}
	// Phase C : DataAdapter HI registré sans CareerSource player-scoped au boot.
	// La capability career.progression sera "not_exposed" pour ce DataAdapter
	// global ; les futurs handlers player-scoped instancieront leur propre
	// DataAdapter avec le CareerRepo du joueur courant via un MiddleWare DI.
	hiData := halo_games.NewDataAdapter(nil, slog.Default())
	titleResolver.RegisterData(hiData)
	// Plan multi-titres §8.1 : event adapter_loaded au boot du data adapter.
	slog.Info("adapter_loaded",
		"title_slug", titlePkg.DefaultSlug,
		"kind", "data",
		"capabilities_count", len(hiData.Capabilities()),
		"note", "player-scoped CareerSource sera injectée endpoint par endpoint",
	)

	// Phase 6 finition multi-titres : 3e adapter — TitleAssetURLAdapter.
	// Compose les URLs /static/... title-scopées (post-Phase 6.5 migration FS).
	hiAssetURL := halo_games.NewAssetURLAdapter().
		WithMapImagesDir(static.AbsKindRoot(cfg.RepoRoot, static.KindMap, halo_games.TitleSlug))
	titleResolver.RegisterAssetURL(hiAssetURL)
	slog.Info("adapter_loaded",
		"title_slug", hiAssetURL.TitleSlug(),
		"kind", "asset_url",
	)

	// Sprint 40 T1 : validation de contrat (dev mode, no-op si LEVELUP_CONTRACT_VALIDATE != 1).
	r.Use(middleware.ContractValidate)

	// P8.3 (revue 2026-04-29, ADR 0009) : error_tracker.Middleware retiré.
	// L'alerting Discord 500 / taux d'erreur n'est pas souhaité (commentaire
	// explicite en code). Code mort supprimé pour éliminer la confusion.

	// Sprint 37 : ServiceRegistry — câblage par injection de dépendances.
	// titleResolver est attaché pour que les services puissent résoudre les
	// SemanticAdapter (libellés rangs etc.) selon le titre courant.
	reg := NewServiceRegistry(cfg, tokenProvider).
		WithTitleResolver(titleResolver).
		WithSettingsStore(settingsStore)

	// Module Prestige — initialisation du bundle (best-effort, désactivable via flag).
	// Charge tuning.toml + templates + preset arcs Halo, ouvre shared_social et metadata.
	// Si le flag PRESTIGE_ENABLED est désactivé, les routes ne sont pas montées et
	// le sync hook est no-op — mais le boot du bundle reste utile pour valider la
	// config au démarrage.
	var prestigeBundle *PrestigeBundle
	if pb, err := NewPrestigeBundle(cfg.RepoRoot, reg.resolve); err != nil {
		slog.Warn("prestige_bundle_init_failed", "err", err.Error())
	} else {
		prestigeBundle = pb
	}

	var gamertagSvc port.GamertagSearchService
	if sharedDB, err := platform_duckdb.OpenReadOnly(config.SharedDBPath(cfg, "")); err != nil {
		slog.Warn("shared DB unavailable for gamertag search", "err", err)
	} else {
		gamertagSvc = platform_duckdb.NewGamertagRepo(sharedDB)
	}

	// AssetHandler — couche d'abstraction unifiée (local-first → API-fallback).
	// Le resolver est créé ici pour accéder à reg.AnyPlayerTokens.
	// Il est aussi passé au ServiceRegistry pour que les HaloProviders délèguent
	// le cache/fetch des définitions BP/challenges au resolver (P4/P5).
	assetCfg := assets.AssetConfig{
		CacheRootDir:  filepath.Join(cfg.RepoRoot, "data", "cache"),
		MetaDBPath:    titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug),
		TokenProvider: reg.AnyPlayerTokens,
	}
	assetResolver, err := assets.New(assetCfg)
	if err != nil {
		slog.Error("assets resolver non disponible — arrêt du serveur", "err", err)
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
		seasonsMetaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
		if seasonsMetaDB, err := platform_duckdb.OpenReadWriteShared(seasonsMetaPath); err != nil {
			slog.Warn("seasons_catalog_meta_db_unavailable",
				"err", err, "fallback", "static_toml_only")
		} else {
			seasonsRepo := platform_duckdb.NewMetadataRepoFromDB(seasonsMetaDB)
			// Tables idempotentes : la migration peut ne pas avoir tourné encore.
			if ensureErr := seasonsRepo.EnsureSeasonTables(context.Background()); ensureErr != nil {
				slog.Warn("seasons_catalog_ensure_tables_failed",
					"err", ensureErr, "fallback", "static_toml_only")
			} else {
				catalog := service.NewSeasonsCatalog(seasonsAssets, seasonsRepo, halo.DefaultHaloProvider, slog.Default())
				reg.WithSeasonsCatalog(catalog)
				slog.Info("seasons_catalog_ready",
					"title_slug", titlePkg.DefaultSlug,
					"toml_count", len(seasonsAssets.AllOfKind("season")),
				)
			}
		}
	}

	// AssetMetadataHandler — listing maps & armes pour l'Asset Drawer.
	// Stratégie in-memory : on ouvre metadata.duckdb, on charge tout en RAM, on ferme
	// la connexion immédiatement. Avantage : aucun lock Windows persistant entre processus
	// (Air hot-reload). Retry 3× (500ms) pour absorber la fenêtre de chevauchement Air.
	var assetMetaHandler *handlers.AssetMetadataHandler
	{
		metaDBPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(500 * time.Millisecond)
			}
			metaDB, err := platform_duckdb.OpenReadWriteShared(metaDBPath)
			if err != nil {
				slog.Warn("asset_metadata_db_unavailable", "attempt", attempt+1, "err", err)
				continue
			}
			liveRepo := platform_duckdb.NewMetadataRepoFromDB(metaDB)
			loadCtx := context.Background()
			maps, errM := liveRepo.ListMapsByTitle(loadCtx, titlePkg.DefaultSlug, "")
			weapons, errW := liveRepo.ListWeaponsByTitle(loadCtx, titlePkg.DefaultSlug, "")
			_ = metaDB.Close() // relâche le lock Windows immédiatement après chargement
			if errM != nil || errW != nil {
				slog.Warn("asset_metadata_load_failed", "err_maps", errM, "err_weapons", errW)
				continue
			}
			assetMetaHandler = handlers.NewAssetMetadataHandler(
				service.NewAssetService(service.NewStaticAssetMetaRepo(maps, weapons)).
					WithMapImageURL(func(_ string, nameEN string) string {
						return hiAssetURL.MapImageURL(nameEN)
					}).
					WithWeaponImageURL(func(_ string, nameEN string) string {
						return hiAssetURL.WeaponImageURL(nameEN)
					}),
				func(slug string, cap titlePkg.Capability) bool {
					d := titleRegistry.Get(slug)
					return d != nil && d.HasCapability(cap)
				},
			)
			slog.Info("asset_metadata_handler_ready",
				"title", titlePkg.DefaultSlug,
				"maps", len(maps),
				"weapons", len(weapons),
				"attempt", attempt+1,
			)
			break
		}
	}

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
	healthH := handlers.NewHealthHandlerWithVersion(bootRepo, cfg.AppVersion)
	r.Get("/health", healthH.ServeHTTP)
	r.Get("/healthz", healthH.Liveness)
	r.Get("/readyz", healthH.Readiness)

	// P8.3 (revue 2026-04-29, ADR 0009) : monitoring expvar minimal.
	// Expose /debug/vars (stdlib) avec les compteurs LevelUp publiés sous la
	// clé "levelup". Pas de Prometheus/OpenTelemetry — observability basique
	// pour multi-user. Les hot paths sont instrumentés progressivement via
	// observability.RecordDurationMS / IncCounter.
	//
	// P8.3 finalisé : protégé derrière RequireAuth + RequireAdmin (transparent
	// en mode démo / auth=none, refus 403 sinon).
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
		r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
		r.Mount("/debug/vars", http.DefaultServeMux)
	})

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		// Endpoints P0 : bootstrap + liste joueurs
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)

		// Phase A multi-titres : exposition des field mappings TOML.
		// Derrière MULTI_TITLE_API_ENABLED.
		//
		// NOTE : les endpoints preview/career et preview/career-multi-title
		// (proof-of-concept Phase C) ont été supprimés en revue 2026-04-29 P0.2
		// Q6 — orphelins côté front même flag activé. À réintroduire en endpoint
		// admin/debug si besoin de re-valider le pipeline canonique.
		if handlers.MultiTitleAPIEnabled() {
			fieldMappingsHandler := handlers.NewFieldMappingsHandler(fieldMappingsRegistry, slog.Default())
			// V2 saisons : si le catalog est câblé, on enrichit le DTO assets
			// avec l'union TOML + DB (cf. SeasonsCatalogForHandler ci-dessous).
			if seasonsResolver := reg.SeasonsCatalogForHandler(); seasonsResolver != nil {
				fieldMappingsHandler = fieldMappingsHandler.WithSeasonsCatalog(seasonsResolver)
			}
			r.Get("/titles/{slug}/field-mappings", fieldMappingsHandler.ServeHTTP)

			// Phase H.bis — catalogue Playlists/Pairs/Maps (title-aware).
			// OpenReadWriteShared pour compatibilité avec les connexions RW existantes
			// (prestige presets, rank catalog) sur le même fichier DuckDB.
			if catalogMetaDB, err := platform_duckdb.OpenReadWriteShared(
				titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug),
			); err != nil {
				slog.Warn("catalog_meta_db_unavailable", "err", err)
			} else {
				catalogH := handlers.NewCatalogHandler(platform_duckdb.NewCatalogRepo(catalogMetaDB.SQLDb(), nil))
				r.Get("/titles/{slug}/catalog/playlists", catalogH.PlaylistsHandler)
				r.Get("/titles/{slug}/catalog/pairs", catalogH.PairsHandler)
				r.Get("/titles/{slug}/catalog/maps", catalogH.MapsHandler)
			}

			slog.Info("multi_title_api_enabled",
				"slugs", fieldMappingsRegistry.Slugs(),
				"endpoints", []string{
					"/api/v1/titles/{slug}/field-mappings",
					"/api/v1/titles/{slug}/catalog/playlists",
					"/api/v1/titles/{slug}/catalog/pairs",
					"/api/v1/titles/{slug}/catalog/maps",
				},
			)
		}

		// Sprint 43 : changelog (markdown brut)
		changelog := handlers.NewChangelogHandler(cfg.RepoRoot)
		r.Get("/changelog", changelog.GetChangelog)

		// Aide : notes de version extraites du README (EN/FR).
		// P8.10 : la logique git + parsing markdown vit dans
		// service.ReleaseNotesService ; le handler ne fait que cache + I/O HTTP.
		releaseBuilder := service.NewReleaseNotesService(cfg.RepoRoot, gitcli.NewCLI())
		help := handlers.NewHelpHandler(releaseBuilder, filepath.Join(cfg.RepoRoot, "data", "cache"))
		r.Get("/help/release-notes", help.GetReleaseNotes)

		// Sprint 14 : contexte de session
		sessionHandler := handlers.NewSessionHandler(sessionStore)
		r.Post("/session/context", sessionHandler.PostContext)

		// Sprint 15 : Device Code Flow + authentification Halo
		authHandler := handlers.NewAuthHandler(sessionStore, attemptStore, cfg.DemoMode, tokenProvider).
			WithUserStore(users)
		r.Post("/auth/device-flow/start", authHandler.StartDeviceFlow)
		r.Get("/auth/device-flow/{attempt_id}", authHandler.GetDeviceFlowStatus)

		// Auth locale : login/register/logout (mode password).
		userAuthHandler := handlers.NewUserAuthHandler(users, invites, sessionStore, cfg.RegistrationMode)
		r.Post("/auth/login", userAuthHandler.Login)
		r.Post("/auth/register", userAuthHandler.Register)
		r.Post("/auth/logout", userAuthHandler.Logout)

		// Admin : gestion utilisateurs + invitations (protégé par RequireAuth + RequireAdmin).
		adminHandler := handlers.NewAdminHandler(users, invites)
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
			r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
			r.Get("/users", adminHandler.ListUsers)
			r.Delete("/users/{username}", adminHandler.DeleteUser)
			r.Patch("/users/{username}/role", adminHandler.ChangeRole)
			r.Patch("/users/{username}/password", adminHandler.ResetPassword)
			r.Get("/invites", adminHandler.ListInvites)
			r.Post("/invites", adminHandler.GenerateInvite)
			r.Delete("/invites/{code}", adminHandler.RevokeInvite)
		})

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
			WithNotificationsEmitter(reg.NotificationsEmitter)
		r.Get("/settings", settingsHandler.GetSettings)
		r.Patch("/settings", settingsHandler.PatchSettings)
		r.Post("/settings/media/reset-index", settingsHandler.PostMediaResetIndex)
		r.Post("/settings/media/scan", settingsHandler.PostMediaScan)
		r.Post("/settings/sessions/recalculate", settingsHandler.PostRecalculateSessions)

		setupHandler := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore,
			service.NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot))
		r.Post("/setup/players", setupHandler.CreatePlayer)
		r.Post("/setup/smoke-test", setupHandler.SmokeTest)

		// Sprint 17 : Jobs longs persistants + sync initiale
		r.Get("/jobs/{job_id}", handlers.NewJobsHandler(jobStore).GetJob)
		syncH := handlers.NewSyncHandler(cfg, settingsStore, jobStore, tokenProvider)
		// Branche le hook Prestige post-sync (best-effort, no-op si flag off ou bundle nil).
		if prestigeBundle != nil {
			syncH = syncH.WithPrestigeHook(prestigeBundle.RunPostSync)
		}
		// Branche la factory d'émetteurs de notifications (match_synced / sync_error).
		syncH = syncH.WithNotificationsEmitterFactory(reg.NotificationsEmitter)
		// Branche le hook delta-detection post-sync (season_pass_level / objective_completed / challenge_completed).
		syncH = syncH.WithPostSyncDeltaHook(buildPostSyncDeltaHook(reg))
		r.Post("/sync/initial", syncH.StartInitialSync)
		r.Post("/sync/all", syncH.StartSyncAll)
		// Sprint 51-B3 : Pipeline backfill (weapon kills + détection des autres types)
		r.Post("/backfill/start", handlers.NewBackfillHandler(cfg, jobStore).StartBackfill)

		// Galerie médias — version de flux pour polling léger
		r.Get("/media/feed-version", handlers.GetMediaFeedVersion)

		// Assets cache-aside unifiés (médailles, maps, battlepass, badges de défi).
		// Couche d'abstraction DefaultResolver : local-first → API-fallback + DuckDB index.
		r.Get("/assets/medals/{title_id}/{medal_id}/image", assetHandler.GetMedalImage)
		r.Get("/assets/maps/{title_id}/{map_id}/image", assetHandler.GetMapImage)
		r.Get("/assets/battlepass/{subdir}/*", assetHandler.GetBattlePassImage)
		r.Get("/assets/challenge-badge/{title_id}/{badge_id}", assetHandler.GetChallengeBadge)
		r.Get("/assets/spartan/{image_type}/{title_id}/*", assetHandler.GetSpartanImage)

		// Asset Drawer — toujours enregistré ; renvoie [] si metaDB indisponible (best-effort).
		r.Get("/assets/{title_id}/maps", func(w http.ResponseWriter, r *http.Request) {
			if assetMetaHandler == nil {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("[]"))
				return
			}
			assetMetaHandler.ListMaps(w, r)
		})
		r.Get("/assets/{title_id}/weapons", func(w http.ResponseWriter, r *http.Request) {
			if assetMetaHandler == nil {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("[]"))
				return
			}
			assetMetaHandler.ListWeapons(w, r)
		})

		// Endpoints P1 : pages par joueur (Sprint 37 — DI via ServiceRegistry)
		r.Route("/players/{player_slug}", func(r chi.Router) {
			filters := handlers.NewFiltersHandler(reg.Filters)
			r.Post("/filters/resolve", filters.Resolve)

			mh := handlers.NewMatchHistoryHandler(reg.MatchHistoryCtx)
			r.Post("/pages/match-history/query", mh.Query)

			// P6.3 : guard de capability — career routes nécessitent CapCareer.
			career := handlers.NewCareerHandler(reg.Career)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapCareer))
				r.Get("/pages/career", career.GetCareer)
				r.Get("/pages/career/top-matches", career.GetTopMatches)
				r.Get("/pages/career/encounters", career.GetEncounters)
			})

			// Sprint 8 : Match View + Explorer
			mv := handlers.NewMatchViewHandler(reg.MatchView)
			r.Get("/matches/{match_id}", mv.GetMatchView)
			r.Get("/matches/{match_id}/neighbors", mv.GetMatchNeighbors)

			// Phase 4 plan engagement : score + courbe par match + profil + timeseries + squad
			eng := handlers.NewEngagementHandler(reg.Engagement)
			r.Get("/matches/{match_id}/engagement", eng.GetMatchEngagement)
			r.Get("/engagement_profile", eng.GetEngagementProfile)
			r.Get("/engagement/timeseries", eng.GetEngagementTimeseries)
			r.Get("/pages/squad/v2/engagement", eng.GetSquadEngagementSession)

			explorer := handlers.NewExplorerHandler(reg.ExplorerCtx, reg.MatchHistoryCtx)
			r.Post("/pages/explorer/player-query", explorer.QueryPlayer)

			// Sprint 9 : Sessions
			sessions := handlers.NewSessionsHandler(reg.Sessions)
			r.Get("/pages/sessions", sessions.GetSessions)
			sessionPage := handlers.NewSessionPageHandler(reg.SessionPage)
			r.Post("/pages/sessions/detail", sessionPage.GetPage)

			// Sprint 10 : Stats/Séries temporelles
			stats := handlers.NewStatsHandler(reg.Stats)
			r.Post("/pages/stats/query", stats.GetPage)

			// Sprint 11 : Accueil/Home + Battle Pass + Challenges
			home := handlers.NewHomeHandler(reg.HomeCtxWithAuth, settingsStore)
			r.With(middleware.CacheMaxAge(30)).Get("/pages/home", home.GetHomePage)
			r.With(middleware.NoStore).Get("/battlepass", home.GetBattlePass)
			r.With(middleware.NoStore).Get("/challenges", home.GetChallenges)

			// Season Pass (palmares)
			seasonPass := handlers.NewSeasonPassHandler(reg.SeasonPassCtxWithAuth)
			r.Get("/pages/palmares/season-pass", seasonPass.GetSeasonPass)

			// Sprint 12 : Escouade | Sprint 55 D1 : Synthèse → handler autonome
			squad := handlers.NewSquadHandler(reg.SquadCtx)
			r.Get("/pages/squad", squad.GetSquadPage)

			// Phase 1 chunk S1b : Squad V2 (multi-coéquipiers, fondations Phase 0)
			squadV2 := handlers.NewSquadV2Handler(reg.SquadV2Ctx)
			r.Get("/pages/squad/v2", squadV2.GetSquadPage)

			// Sprint 55 D1 : SynthesisHandler extrait de SquadHandler (frontière produit)
			synthesis := handlers.NewSynthesisHandler(reg.SynthesisCtx)
			r.Post("/pages/synthesis", synthesis.GetSynthesisPage)

			// Sprint 13 → Sprint 32 : Citations + Commendations + Médias → POST
			citations := handlers.NewCitationsHandler(reg.CitationsCtx)
			r.Post("/pages/citations", citations.GetCitations)
			r.Post("/pages/commendations", citations.GetCommendations)

			// P6.3 : guard de capability — media routes nécessitent CapMedia.
			media := handlers.NewMediaHandler(reg.Media, reg.MediaUpload, cfg.RepoRoot).
				WithSettingsStore(settingsStore).
				WithAuthorsContext(reg.MediaPlayerCtx, func(_ context.Context, titleSlug string) ([]domain.PlayerSummary, error) {
					return cfg.LoadPlayers(titleSlug)
				}).
				WithNotificationsEmitterFactory(reg.NotificationsEmitter).
				WithMediaRecipientResolver(reg.MediaRecipientResolver(cfg))
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapMedia))
				r.Post("/pages/media", media.GetMediaLibrary)
				r.Patch("/media/likes", media.PatchMediaLike)
				r.Post("/media/upload", media.PostUploadMedia)
				// /media/reassociate supprimé en revue 2026-04-29 P0.2 Q6 (doublon non utilisé,
				// le front consomme /media/associate seulement).
				r.Get("/media/match-candidates", media.GetMediaMatchCandidates)
				r.Post("/media/associate", media.PostMediaAssociate)
				r.Get("/media/authors", media.GetMediaAuthors)
				r.Get("/media/files/*", media.ServeMediaFile)
			})

			// Sprint 32 : Explorer matches-query + Match History export
			r.Post("/pages/explorer/matches-query", explorer.QueryMatches)
			r.Get("/pages/match-history/export", mh.Export)

			// Sprint 33 : Teammates (contrat FastAPI)
			teammates := handlers.NewTeammatesHandler(reg.TeammatesCtx)
			r.Post("/pages/teammates", teammates.GetPage)

			// Sprint 33 : Timeseries (contrat FastAPI)
			timeseries := handlers.NewTimeseriesHandler(reg.Timeseries)
			r.Post("/pages/timeseries", timeseries.GetPage)

			// Sprint 33 : Session Compare
			sc := handlers.NewSessionCompareHandler(reg.SessionCompare)
			r.Post("/pages/session-compare", sc.Compare)

			// Exclusion manuelle de matchs non pertinents
			// NOTE : GET /match-exclusions supprimé en revue 2026-04-29 P0.2 Q6
			// (orphelin côté front, vue admin jamais implémentée).
			excl := handlers.NewMatchExclusionHandler(reg.MatchExclusion)
			r.Patch("/matches/{match_id}/exclusion", excl.SetExclusion)

			// Système de notifications in-app (per-player).
			notifH := handlers.NewNotificationsHandler(reg.Notifications)
			notifH.Mount(r)

			// Match favoris (shared_social.duckdb)
			fav := handlers.NewMatchFavoriteHandler(reg.Social)
			r.Patch("/matches/{match_id}/favorite", fav.PatchMatchFavorite)

			// Module Prestige — routes derrière feature flag PRESTIGE_ENABLED.
			// Le bundle a été initialisé au boot ; si nil ou flag off, routes non montées.
			if prestigeBundle != nil && prestige.IsEnabled() {
				lazy := NewLazyPrestigeService(prestigeBundle, nil)
				ph := handlers.NewPrestigeHandler(lazy)
				// Défis
				r.Post("/challenges", ph.CreateChallenge)
				r.Get("/challenges", ph.ListActiveChallenges)
				r.Get("/challenges/{id}", ph.GetChallenge)
				r.Patch("/challenges/{id}", ph.UpdateChallenge)
				r.Delete("/challenges/{id}", ph.AbandonChallenge)
				r.Post("/challenges/{id}/suggest-next", ph.SuggestNext)
				// Arcs
				r.Post("/arcs", ph.CreateArc)
				r.Get("/arcs", ph.ListArcs)
				r.Get("/arcs/{id}", ph.GetArc)
				// Prestige (PP + niveau)
				r.Get("/prestige/me", ph.GetMyPrestige)
				r.Get("/templates/suggest", ph.SuggestTemplates)
				// Squad challenges
				r.Post("/squads/{squad_id}/challenges", ph.CreateSquadChallenge)
				r.Get("/squads/{squad_id}/challenges", ph.ListSquadChallenges)
				r.Post("/squads/{squad_id}/challenges/pool/refresh", ph.RefreshSquadPool)
				r.Post("/squad-challenges/{id}/join", ph.JoinSquadChallenge)
				// Mode pilote
				r.Post("/pilot-mode/enable", ph.EnablePilotMode)
				r.Post("/pilot-mode/disable", ph.DisablePilotMode)
				slog.Info("prestige_routes_mounted", "endpoints_count", 16)
			}

			// Sprint 54 : Compare joueur vs joueur
			compare := handlers.NewCompareHandler(reg.Compare)
			r.Post("/pages/compare", compare.PostComparePage)

			// Sprint 54 : Classement CSR (Leaderboard)
			leaderboard := handlers.NewLeaderboardHandler(reg.Leaderboard)
			r.Get("/pages/leaderboard", leaderboard.GetLeaderboardPage)

			// Sync delta par joueur
			r.Post("/sync", syncH.StartDeltaSync)
		})

		// Endpoints P1 : répertoire gamertags
		// Sprint 49 : route inconditionnelle — retourne 503 si shared DB absente.
		r.Get("/directory/gamertags/search", handlers.NewGamertagHandler(gamertagSvc).Search)

		// Watcher présence Xbox RTA — RequireAuth + RequireAdmin.
		watcherAttempts := auth_platform.NewWatcherAttemptStore()
		watcherHandler := handlers.NewWatcherHandler(cfg, settingsStore, daemon, tokenProvider, watcherAttempts)
		r.Route("/watcher", func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
			r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
			r.Get("/status", watcherHandler.GetStatus)
			r.Post("/auth/start", watcherHandler.StartAuth)
			r.Get("/auth/{attempt_id}", watcherHandler.GetAuthStatus)
			r.Patch("/subscriptions", watcherHandler.PatchSubscriptions)
		})
	})

	return r, reg
}
