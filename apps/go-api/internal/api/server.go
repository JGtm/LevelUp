// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
// Sprint 37 : Architecture handlers & injection DI via ServiceRegistry.
// Sprint 40 : ContractValidate (dev) + ErrorTracker (Discord 500 + alerting).
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	auth_platform "levelup/go-api/internal/platform/auth"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
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
			if catalog, err := halo_games.LoadRankCatalog(context.Background(), metaDB); err == nil {
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
		if sem := halo_games.NewSemanticAdapter(hiFields, hiRanks); sem != nil {
			titleResolver.RegisterSemantic(sem)
			slog.Info("title_semantic_adapter_registered",
				"title_slug", sem.TitleSlug(),
				"schema_version", sem.SchemaVersion(),
			)
		}
	}
	// Phase C : DataAdapter HI registré sans CareerSource player-scoped au boot.
	// La capability career.progression sera "not_exposed" pour ce DataAdapter
	// global ; les futurs handlers player-scoped instancieront leur propre
	// DataAdapter avec le CareerRepo du joueur courant via un MiddleWare DI.
	titleResolver.RegisterData(halo_games.NewDataAdapter(nil, slog.Default()))
	slog.Info("title_data_adapter_registered_global",
		"title_slug", titlePkg.DefaultSlug,
		"note", "player-scoped CareerSource sera injectée endpoint par endpoint",
	)

	// Sprint 40 T1 : validation de contrat (dev mode, no-op si LEVELUP_CONTRACT_VALIDATE != 1).
	r.Use(middleware.ContractValidate)

	// Sprint 40 T2+T3 : error tracking + alerting Discord.
	errorTracker := middleware.NewErrorTracker(middleware.ErrorTrackerConfig{
		WebhookURL: cfg.DiscordWebhookURL,
	})
	r.Use(errorTracker.Middleware)

	// Sprint 37 : ServiceRegistry — câblage par injection de dépendances.
	// titleResolver est attaché pour que les services puissent résoudre les
	// SemanticAdapter (libellés rangs etc.) selon le titre courant.
	reg := NewServiceRegistry(cfg, tokenProvider).WithTitleResolver(titleResolver)
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
		StaticMapDir:  filepath.Join(cfg.RepoRoot, "static", "maps"),
		TokenProvider: reg.AnyPlayerTokens,
	}
	assetResolver, err := assets.New(assetCfg)
	if err != nil {
		slog.Error("assets resolver non disponible — arrêt du serveur", "err", err)
		os.Exit(1)
	}
	assetHandler := handlers.NewAssetHandler(assetResolver)
	reg.WithAssetResolver(assetResolver)

	// Fichiers statiques (images maps, médailles, armes…)
	staticDir := filepath.Join(cfg.RepoRoot, "static")
	// Handler spécial pour /static/commendations/* : fallback vers noms URL-encodés
	// pour les fichiers dont le nom décodé contient des caractères interdits Windows (ex: ?).
	r.Handle("/static/commendations/*", newCommendationHandler(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel)
	r.Get("/health", handlers.NewHealthHandlerWithVersion(bootRepo, cfg.AppVersion).ServeHTTP)

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		// Endpoints P0 : bootstrap + liste joueurs
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)

		// Phase A+C multi-titres : exposition des field mappings TOML +
		// preview du pipeline canonique. Tout derrière MULTI_TITLE_API_ENABLED.
		if handlers.MultiTitleAPIEnabled() {
			fieldMappingsHandler := handlers.NewFieldMappingsHandler(fieldMappingsRegistry, slog.Default())
			r.Get("/titles/{slug}/field-mappings", fieldMappingsHandler.ServeHTTP)

			previewHandler := handlers.NewMultiTitlePreviewHandler(titleResolver, slog.Default())
			r.Get("/titles/{slug}/preview/career", previewHandler.GetCareerPreview)

			// Endpoint preview player-scoped : utilise un DataAdapter HI
			// instancié par requête avec le CareerRepo du joueur courant.
			// Capability career.progression réellement supportée ici.
			//
			// Adaptation : titleResolver.Semantic est synchrone (slug→adapter), alors
			// que SemanticAdapterFactory accepte un context. La closure ignore le
			// context — la résolution est in-memory, pas de risque d'annulation.
			semanticFactory := func(_ context.Context, slug string) (games.TitleSemanticAdapter, error) {
				return titleResolver.Semantic(slug)
			}
			playerPreviewHandler := handlers.NewMultiTitlePlayerPreviewHandler(
				reg.TitleDataAdapter,
				semanticFactory,
				titlePkg.DefaultSlug,
				slog.Default(),
			)
			r.Get("/players/{player_slug}/preview/career-multi-title", playerPreviewHandler.GetCareerPreview)

			slog.Info("multi_title_api_enabled",
				"slugs", fieldMappingsRegistry.Slugs(),
				"endpoints", []string{
					"/api/v1/titles/{slug}/field-mappings",
					"/api/v1/titles/{slug}/preview/career",
					"/api/v1/players/{player_slug}/preview/career-multi-title",
				},
			)
		}

		// Sprint 43 : changelog (markdown brut)
		changelog := handlers.NewChangelogHandler(cfg.RepoRoot)
		r.Get("/changelog", changelog.GetChangelog)

		// Aide : notes de version extraites du README (EN/FR)
		help := handlers.NewHelpHandler(cfg.RepoRoot)
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
		settingsHandler := handlers.NewSettingsHandler(cfg, settingsStore, jobStore)
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

		// Endpoints P1 : pages par joueur (Sprint 37 — DI via ServiceRegistry)
		r.Route("/players/{player_slug}", func(r chi.Router) {
			filters := handlers.NewFiltersHandler(reg.Filters)
			r.Post("/filters/resolve", filters.Resolve)

			mh := handlers.NewMatchHistoryHandler(reg.MatchHistoryCtx)
			r.Post("/pages/match-history/query", mh.Query)

			career := handlers.NewCareerHandler(reg.Career)
			r.Get("/pages/career", career.GetCareer)
			r.Get("/pages/career/top-matches", career.GetTopMatches)
			r.Get("/pages/career/encounters", career.GetEncounters)

			// Sprint 8 : Match View + Explorer
			mv := handlers.NewMatchViewHandler(reg.MatchView)
			r.Get("/matches/{match_id}", mv.GetMatchView)
			r.Get("/matches/{match_id}/neighbors", mv.GetMatchNeighbors)

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
			r.Get("/pages/home", home.GetHomePage)
			r.Get("/battlepass", home.GetBattlePass)
			r.Get("/challenges", home.GetChallenges)

			// Season Pass (palmares)
			seasonPass := handlers.NewSeasonPassHandler(reg.SeasonPassCtxWithAuth)
			r.Get("/pages/palmares/season-pass", seasonPass.GetSeasonPass)

			// Sprint 12 : Escouade | Sprint 55 D1 : Synthèse → handler autonome
			squad := handlers.NewSquadHandler(reg.SquadCtx)
			r.Get("/pages/squad", squad.GetSquadPage)

			// Sprint 55 D1 : SynthesisHandler extrait de SquadHandler (frontière produit)
			synthesis := handlers.NewSynthesisHandler(reg.SynthesisCtx)
			r.Post("/pages/synthesis", synthesis.GetSynthesisPage)

			// Sprint 13 → Sprint 32 : Citations + Commendations + Médias → POST
			citations := handlers.NewCitationsHandler(reg.CitationsCtx)
			r.Post("/pages/citations", citations.GetCitations)
			r.Post("/pages/commendations", citations.GetCommendations)

			media := handlers.NewMediaHandler(reg.Media, reg.MediaUpload, cfg.RepoRoot).
				WithSettingsStore(settingsStore).
				WithAuthorsContext(reg.MediaPlayerCtx, func(_ context.Context, titleSlug string) ([]domain.PlayerSummary, error) {
					return cfg.LoadPlayers(titleSlug)
				})
			r.Post("/pages/media", media.GetMediaLibrary)
			r.Patch("/media/likes", media.PatchMediaLike)
			r.Post("/media/upload", media.PostUploadMedia)
			r.Post("/media/reassociate", media.PostReassociateMedia)
			r.Get("/media/authors", media.GetMediaAuthors)
			r.Get("/media/files/*", media.ServeMediaFile)

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
			excl := handlers.NewMatchExclusionHandler(reg.MatchExclusion)
			r.Patch("/matches/{match_id}/exclusion", excl.SetExclusion)
			r.Get("/match-exclusions", excl.ListExclusions)

			// Match favoris (shared_social.duckdb)
			fav := handlers.NewMatchFavoriteHandler(reg.Social)
			r.Patch("/matches/{match_id}/favorite", fav.PatchMatchFavorite)

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
