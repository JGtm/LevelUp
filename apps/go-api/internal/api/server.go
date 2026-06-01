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
	"levelup/go-api/internal/ctxkeys"
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
	"levelup/go-api/internal/platform/halo"
	jobs_platform "levelup/go-api/internal/platform/jobs"
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

// playerOwnershipXUIDResolver mappe un slug joueur vers le xuid de son profil
// pour le titre courant (lu depuis le contexte), via db_profiles.json sans
// ouvrir de DuckDB. Alimente middleware.RequirePlayerOwnership (ADR 0024).
func playerOwnershipXUIDResolver(cfg *config.AppConfig) middleware.PlayerXUIDResolver {
	return func(ctx context.Context, slug string) (string, bool) {
		players, err := cfg.LoadPlayers(ctxkeys.TitleSlug(ctx))
		if err != nil {
			return "", false
		}
		for i := range players {
			if players[i].PlayerSlug == slug {
				return players[i].XUID, true
			}
		}
		return "", false
	}
}

// NewRouter construit le routeur chi avec tous les endpoints.
// Construction par injection de dépendances — pas d'état global.
// daemon peut être nil si le watcher n'est pas actif au démarrage.
// tokenProvider peut être nil : MSALProvider est utilisé par défaut.
// Retourne aussi le *ServiceRegistry pour permettre au démon watcher de lier le TTL dynamique.
//
// conditionnels (MULTI_TITLE_API_ENABLED, PRESTIGE_ENABLED, etc.). Complexité
// reflète la surface API, pas un défaut de conception.
//
//nolint:gocyclo // Routeur central : mount de ~80 endpoints avec feature flags
func NewRouter(
	cfg *config.AppConfig,
	bootRepo port.BootstrapRepository,
	bootSvc *service.BootstrapService,
	daemon watcher.DaemonController,
	tokenProvider auth_platform.TokenProvider,
	autoSyncScheduler *scheduler.AutoSyncScheduler,
	backupScheduler *duckdbbackup.Scheduler,
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
	var hiRanks *mappings.RankCatalog
	var hiRankImageURLs map[int]*string
	if hiFields, ok := fieldMappingsRegistry.Get(titlePkg.DefaultSlug); ok {
		// Charger le catalog des rangs HI depuis metadata.duckdb (career_rank_translations).
		// OpenReadWriteShared est cached par path → réutilise le pool existant ouvert dans
		// cmd/server. IMPORTANT : Close() pour décrémenter le refCount sinon le sql.DB
		// reste ouvert au shutdown (le metaDB.Close() de cmd/server décrémente seulement
		// d'un cran), ce qui retient le HANDLE Windows et provoque le verrou
		// "metadata verrouillée" au prochain hot-reload Air.
		hiMetaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
		if metaDB, err := platform_duckdb.OpenReadWriteShared(hiMetaPath); err == nil {
			if catalog, err := platform_duckdb.LoadRankCatalog(context.Background(), metaDB); err == nil {
				hiRanks = catalog
				slog.Info("rank_catalog_loaded", "title_slug", titlePkg.DefaultSlug, "ranks", catalog.Len())
			} else {
				slog.Warn("rank_catalog_load_failed", "err", err.Error())
			}
			if imgs, err := platform_duckdb.LoadCareerRankImageURLs(context.Background(), metaDB, titlePkg.DefaultSlug); err == nil {
				hiRankImageURLs = imgs
				slog.Info("rank_image_urls_loaded", "title_slug", titlePkg.DefaultSlug, "images", len(imgs))
			} else {
				slog.Warn("rank_image_urls_load_failed", "err", err.Error())
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
		WithSettingsStore(settingsStore).
		WithRankCatalog(hiRanks).
		WithRankImageURLs(hiRankImageURLs)

	// Module Prestige — initialisation du bundle (best-effort, désactivable via flag).
	// Charge tuning.toml + templates + preset arcs Halo, ouvre shared_social et metadata.
	// Si le flag PRESTIGE_ENABLED est désactivé, les routes ne sont pas montées et
	// le sync hook est no-op — mais le boot du bundle reste utile pour valider la
	// config au démarrage.
	var prestigeBundle *PrestigeBundle
	if pb, err := NewPrestigeBundle(cfg.RepoRoot, reg.resolve, cfg.PrestigeEnabled); err != nil {
		slog.Warn("prestige_bundle_init_failed", "err", err.Error())
	} else {
		prestigeBundle = pb
		// Phase 2 plan stabilisation 2026-05-22 : enregistrer le bundle sur
		// le registry pour fermeture au shutdown (évite la fuite de refCount
		// sur metadata.duckdb qui causait le verrou au hot-reload Air).
		reg.WithPrestigeBundle(pb)
	}

	// Coach Advisor bundle (Phase 8 ADR 0020) — charge la grammaire de
	// synthèse une fois au boot. Pas de DB-handle à fermer ; toujours non-nil
	// (fallback grammaire vide si TOML absent, synthèse désactivée mais
	// matching catalogue reste fonctionnel).
	reg.WithCoachAdvisorBundle(NewCoachAdvisorBundle(cfg.RepoRoot))

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
		gamertagSvc = platform_duckdb.NewGamertagRepo(cfg.SharedProvider)
	} else {
		slog.Warn("shared DB unavailable for gamertag search — cfg.SharedProvider nil")
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

		// Smoke endpoint pour la home page : sonde le contenu (banner, peaks
		// CSR/LUSR, playlists récentes, arme favorite) et renvoie 503 si une
		// section est vide sans raison. Pensé pour CI post-backfill et alerte
		// dev. Cf. handlers/health_home.go.
		r.With(middleware.NoStore).Get("/healthz/home", handlers.NewHealthHomeHandler(reg.HomeCtxWithAuth).Check)

		// Phase 9 du plan pipeline CSR : diagnostic coverage CSR pour un joueur.
		// Permet de vérifier en 1 ligne si le pipeline a bien capturé les CSR
		// (matured + placement) ou s'il faut lancer un backfill.
		r.With(middleware.NoStore).Get("/_diag/csr-coverage/{player_slug}",
			handlers.NewDiagCSRHandler(reg.CSRCoverageProvider).GetCoverage)

		// Phase 4 plan stabilisation 2026-05-22 : diagnostic progression V2
		// (Ascension). Compte les rows dans streak/player_records/record_history/
		// milestone_earned + milestone_catalog. Permet de vérifier que
		// EvaluateProgressionAfterSync tourne bien sur l'auto-sync (avant Phase 4
		// ces tables restaient vides — cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED).
		r.With(middleware.NoStore).Get("/_diag/progression/{player_slug}",
			handlers.NewDiagProgressionHandler(reg.ProgressionDiagProvider).GetDiag)

		// Fix 2026-05-30 : backfill progression V2 in-process. Force une
		// évaluation idempotente (streaks/records/milestones) pour un joueur
		// dont l'historique existe mais dont le pipeline post-sync n'avait
		// jamais abouti (incident timeout shared reader). Renvoie le diag
		// post-exécution.
		r.With(middleware.NoStore).Post("/_admin/progression/backfill/{player_slug}",
			handlers.NewProgressionBackfillHandler(reg.ProgressionBackfillProvider).RunBackfill)

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
			r.Get("/titles/{slug}/field-mappings", fieldMappingsHandler.ServeHTTP)

			// Phase H.bis — catalogue Playlists/Pairs/Maps (title-aware).
			// OpenReadWriteShared pour compatibilité avec les connexions RW existantes
			// (prestige presets, rank catalog) sur le même fichier DuckDB.
			if catalogMetaDB, err := platform_duckdb.OpenReadWriteShared(
				titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug),
			); err != nil {
				slog.Warn("catalog_meta_db_unavailable", "err", err)
			} else {
				catalogH := handlers.NewCatalogHandler(platform_duckdb.NewCatalogRepo(catalogMetaDB, nil))
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
		releaseBuilder := service.NewReleaseNotesService(cfg.RepoRoot)
		help := handlers.NewHelpHandler(releaseBuilder, filepath.Join(cfg.RepoRoot, "data", "cache"))
		r.Get("/help/release-notes", help.GetReleaseNotes)

		// Sprint 14 : contexte de session
		sessionHandler := handlers.NewSessionHandler(sessionStore)
		r.Post("/session/context", sessionHandler.PostContext)

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
				WithDaemonGetter(daemonGetter)
			authHandler.WithLinkStrategy(xboxLinkStrategy)
		} else {
			authHandler.WithUserStore(users)
		}
		r.Post("/auth/device-flow/start", authHandler.StartDeviceFlow)
		r.Get("/auth/device-flow/{attempt_id}", authHandler.GetDeviceFlowStatus)

		// PR 4 — Authorization Code Flow SSO Xbox (UX redirect, plus aboutie que
		// le Device Code). Enregistré uniquement en mode "xbox" + redirect URI
		// configuré. Sans la config Azure (plateforme "Web" + redirect URI dans
		// le portail), /authorize retourne AADSTS50011.
		if cfg.AuthMode == "xbox" && cfg.OAuthRedirectURI != "" {
			xboxOAuthHandler := handlers.NewXboxOAuthHandler(sessionStore, tokenProvider, cfg.DemoMode, cfg.OAuthRedirectURI).
				WithLinkStrategy(xboxLinkStrategy).
				WithAuthStore(authStore)
			r.Get("/auth/xbox/login", xboxOAuthHandler.LoginRedirect)
			r.Get("/auth/xbox/callback", xboxOAuthHandler.Callback)
		}

		// Auth locale : login/register/logout (mode password).
		// D3 cohabitation : en mode "xbox", login réservé aux admins, register au bootstrap.
		userAuthHandler := handlers.NewUserAuthHandler(users, invites, sessionStore, cfg.RegistrationMode).
			WithAuthMode(cfg.AuthMode)
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

		// Diagnostic — accessible en loopback (127.0.0.1) uniquement, sans auth.
		// Permet de comprendre pourquoi le scheduler ne sync pas un joueur sans
		// avoir à fouiller dans les logs serveur (raison du skip/failure par joueur).
		// Bloqué en non-loopback : retourne 403.
		if autoSyncScheduler != nil {
			autoSyncH := handlers.NewAdminAutoSyncHandler(autoSyncScheduler, cfg, tokenProvider)
			r.Route("/_diag/auto-sync", func(r chi.Router) {
				r.Use(middleware.LoopbackOnly)
				r.Get("/snapshot", autoSyncH.GetSnapshot)
				r.Post("/run", autoSyncH.RunOnce)
				r.Get("/probe", autoSyncH.ProbeTokens)
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
		r.Get("/settings", settingsHandler.GetSettings)
		r.Patch("/settings", settingsHandler.PatchSettings)
		r.Post("/settings/media/reset-index", settingsHandler.PostMediaResetIndex)
		r.Post("/settings/media/scan", settingsHandler.PostMediaScan)
		r.Post("/settings/sessions/recalculate", settingsHandler.PostRecalculateSessions)
		r.Get("/settings/backup/status", settingsHandler.GetBackupStatus)
		r.Post("/settings/backup/run", settingsHandler.PostBackupRun)

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
		// D3-01 (revue 2026-06-01) : /sync/initial et /sync/all sont des opérations
		// admin/setup (sync d'un joueur arbitraire lu dans le body / de TOUS les
		// joueurs). Auparavant montées sous /api/v1 SANS auth → contournaient
		// l'ownership (ADR 0024) : n'importe qui pouvait déclencher le sync de
		// n'importe quel joueur. Protégées par RequireAuth + RequireAdmin comme le
		// groupe /admin. En mode demo/single-user les middlewares no-opent (onboarding
		// préservé) ; en multi-user, seul un admin peut déclencher.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.DemoMode, cfg.AuthMode))
			r.Use(middleware.RequireAdmin(cfg.DemoMode, cfg.AuthMode))
			r.Post("/sync/initial", syncH.StartInitialSync)
			r.Post("/sync/all", syncH.StartSyncAll)
		})
		// Sprint 51-B3 : Pipeline backfill (weapon kills + détection des autres types)
		r.Post("/backfill/start", handlers.NewBackfillHandler(cfg, jobStore).StartBackfill)

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
			osImportH := handlers.NewOpenSpartanImportHandler(handlers.OpenSpartanImportConfig{
				ImportService:     osImportSvc,
				PostImportService: osPostImportSvc,
				JobStore:          jobStore,
				StashDir:          filepath.Join(cfg.RepoRoot, "data", "players"),
				DemoMode:          cfg.DemoMode,
			})
			r.Post("/import/openspartan", osImportH.StartImport)
		}

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
			// Couche A (ADR 0024) : garde de propriété joueur. Chokepoint unique —
			// 403 player_forbidden si l'utilisateur courant ne possède pas le slug.
			// Transparent en mode demo / auth non activée. Toute route player-scoped
			// DOIT rester montée sous ce groupe pour être protégée.
			r.Use(middleware.RequirePlayerOwnership(cfg.DemoMode, cfg.AuthMode, playerOwnershipXUIDResolver(cfg), users))

			filters := handlers.NewFiltersHandler(reg.Filters)
			r.Post("/filters/resolve", filters.Resolve)

			mh := handlers.NewMatchHistoryHandler(reg.MatchHistoryCtx)
			r.Post("/pages/match-history/query", mh.Query)

			// P6.3 : guard de capability — career routes nécessitent CapCareer.
			career := handlers.NewCareerHandler(reg.Career, reg.MatchHistoryCtx)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapCareer))
				r.Get("/pages/career", career.GetCareer)
				r.Get("/pages/career/top-matches", career.GetTopMatches)
				r.Get("/pages/career/encounters", career.GetEncounters)
				r.Get("/pages/career/highlight-matches", career.GetHighlightMatches)
				r.Get("/pages/career/top-encounters", career.GetTopEncountersRich)
				r.Get("/pages/career/rivals", career.GetRivals)
				r.Get("/pages/career/csrs", career.GetCareerCSRs)
			})

			// Achievements (Xbox bilingues) : guard CapAchievements.
			achievements := handlers.NewAchievementsHandler(reg.Achievements)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapAchievements))
				r.Get("/pages/achievements", achievements.GetAchievementsPage)
			})

			// Sprint 8 : Match View + Explorer
			// WithMediaURLs : transforme les chemins média de l'onglet médias en
			// URLs servables, comme la galerie (settingsStore + repoRoot déjà en portée).
			mv := handlers.NewMatchViewHandler(reg.MatchView).
				WithMediaURLs(settingsStore, cfg.RepoRoot)
			r.Get("/matches/{match_id}", mv.GetMatchView)
			r.Get("/matches/{match_id}/neighbors", mv.GetMatchNeighbors)

			// Phase 4 plan engagement : score + courbe par match + profil + timeseries + squad
			// + admin recompute. Toutes les routes sont gated par CapEngagement
			// (titre doit declarer la capability — halo_infinite=oui, autres=non
			// par defaut, degradation gracieuse via 404).
			eng := handlers.NewEngagementHandler(reg.Engagement)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireCapability(titleRegistry, titlePkg.CapEngagement))
				r.Get("/matches/{match_id}/engagement", eng.GetMatchEngagement)
				r.Get("/engagement_profile", eng.GetEngagementProfile)
				r.Post("/engagement/timeseries", eng.GetEngagementTimeseries)
				r.Get("/pages/squad/v2/engagement", eng.GetSquadEngagementSession)
				r.Post("/engagement/recompute_coefficients", eng.PostRecomputeCoefficients)
			})

			explorer := handlers.NewExplorerHandler(reg.ExplorerCtxWithAuth, reg.MatchHistoryCtx)
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

			// Couche progression V2 (Ascension) — streaks / records / milestones.
			// Cf. .ai/PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1.
			progressionResolve := func(ctx context.Context, slug string) (*platform_duckdb.PlayerDB, error) {
				return reg.resolve(ctx, slug)
			}
			progressionH := handlers.NewProgressionHandler(progressionResolve, defaultProgressionTitleSlug())
			progressionH.Mount(r)

			// Coach Advisor — proposals coach proactives (ADR 0020 Phase 9).
			// Resolver compose PlayerDB + bundles → coach_advisor.Service.
			coachResolve := func(ctx context.Context, slug string) (coach_advisor.Service, string, error) {
				pdb, err := reg.resolve(ctx, slug)
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
			coachH := handlers.NewCoachProposalsHandler(coachResolve, defaultProgressionTitleSlug())
			coachH.Mount(r)

			// PlayerProfile V1 (Ascension) — endpoint /profile complet.
			// Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §8.1.
			profileH := handlers.NewPlayerProfileHandler(progressionResolve, defaultProgressionTitleSlug())
			// V2 §2 : injection optionnelle du mapping awards→axes (Section A1 radar).
			// Chargement lazy depuis config/titles/{slug}/mappings/awards.toml.
			// Absence du fichier ou erreur de parse : log + fallback V1 silencieux.
			awardsPath := filepath.Join(cfg.RepoRoot, "config", "titles", defaultProgressionTitleSlug(), "mappings", "awards.toml")
			if awardSet, err := mappings.LoadAwardsFromFile(awardsPath); err != nil {
				slog.Warn("player_profile_awards_load_failed", "path", awardsPath, "err", err.Error())
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
			patternsH := handlers.NewPatternsHandler(patternsRepoResolve, defaultProgressionTitleSlug())
			patternsH.Mount(r)

			// ImprovementCampaign V1 — endpoints start/active/pause/close/abandon.
			// Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4.5 + §5.1.
			campaignH := handlers.NewCampaignHandler(progressionResolve, defaultProgressionTitleSlug())
			campaignH.Mount(r)

			// Match favoris (shared_social.duckdb)
			fav := handlers.NewMatchFavoriteHandler(reg.Social)
			r.Patch("/matches/{match_id}/favorite", fav.PatchMatchFavorite)

			// Module Prestige — routes derrière feature flag PRESTIGE_ENABLED.
			// Le bundle a été initialisé au boot ; si nil ou flag off, routes non montées.
			if prestigeBundle != nil && cfg.PrestigeEnabled {
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
