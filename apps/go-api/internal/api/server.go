// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
// Sprint 37 : Architecture handlers & injection DI via ServiceRegistry.
// Sprint 40 : ContractValidate (dev). ErrorTracker retiré P8.3 (ADR 0009).
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/notify"

	// Blank import : déclenche l'init() de observability qui publie le namespace
	// expvar "levelup". Le handler /debug/vars (stdlib) découvre ces compteurs
	// automatiquement via http.DefaultServeMux (P8.3, ADR 0009).
	_ "levelup/go-api/internal/observability"
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

// playerOwnershipXUIDResolver mappe un slug joueur vers le xuid de son profil
// pour le titre courant (lu depuis le contexte), via db_profiles.json sans
// ouvrir de DuckDB. Alimente middleware.RequirePlayerOwnership (ADR 0029).
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

// familyXUIDResolver résout l'ensemble des xuids co-membres de groupe DU USER
// courant (groups.json) pour autoriser le switch de BDD entre membres d'un même
// groupe/famille (ADR 0029). Lit la session depuis le contexte → user → groupes.
// Retourne nil (→ accès strict propriétaire-only) si pas de session, user non lié,
// ou aucun groupe partagé. Title-agnostic : les groupes sont indexés par xuid.
//
// PMT-4 — DÉCISION : ce cercle de confiance est cross_title_global (les groupes
// sont des PERSONNES transverses aux titres), JAMAIS résolu par overlay titre. Le
// grant d'accès cross-player (ownership famille) ne doit pas rétrécir/élargir
// silencieusement par jeu : un overlay per-titre serait un footgun authz. Même
// logique pour tous les sites is_with_friends (loaders engine V1/V2/auto_sync,
// CLI recompute-friends). Seuls les settings UX (sessions/coach/progression/
// outcomes) passent par l'overlay. (Décision PMT-4, 2026-06-17.)
func familyXUIDResolver(groupStore *groupstore.GroupStore, users authz.UserLookup) middleware.FamilyXUIDResolver {
	return func(ctx context.Context) map[string]bool {
		user := authz.CurrentUser(middleware.GetSession(ctx), users)
		if user == nil || user.XUID == "" {
			return nil
		}
		co, err := groupStore.CoMemberXUIDs(user.XUID)
		if err != nil {
			return nil
		}
		return co
	}
}

// metadataDBPathFor retourne le chemin de la metadata.duckdb du titre par défaut.
// En démo, la metadata title (data/titles/...) est une coquille vide créée au boot ;
// les référentiels (rangs, maps, armes, saisons, catalogue) vivent dans la metadata
// des fixtures démo (data/demo/warehouse/metadata.duckdb, copiée intégralement de la
// prod par seed-demo). Sans cette redirection, ces référentiels sont vides en démo.
func metadataDBPathFor(cfg *config.AppConfig) string {
	if cfg.DemoMode && cfg.DemoFixturesDir != "" {
		return filepath.Join(cfg.DemoFixturesDir, "warehouse", "metadata.duckdb")
	}
	return titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
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
// loadTitleAssetDrawerData charge les maps + armes (avec leurs URLs d'image) d'un
// titre additionnel depuis sa metadata.duckdb ISOLÉE, pour l'Asset Drawer
// (référentiel). Peuplé par cmd/h5-metadata-fetch (API Metadata officielle :
// maps_catalog.image_url + weapon_labels.icon_url). Best-effort : DB absente / tables
// vides → slices nil (le titre n'apparaît simplement pas dans le drawer).
func loadTitleAssetDrawerData(pr *titlePkg.PathResolver, slug string) (maps, weapons, medals []canonical.AssetMeta) {
	metaDB, err := platform_duckdb.OpenReadWriteShared(pr.MetadataDBPath(slug))
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = metaDB.Close() }()
	ctx := context.Background()

	if rows, qerr := metaDB.Query(ctx,
		`SELECT map_asset_id, COALESCE(name_canonical, ''), COALESCE(image_url, '')
		 FROM maps_catalog WHERE title_slug = ? ORDER BY name_canonical`, slug); qerr == nil {
		for rows.Next() {
			var m canonical.AssetMeta
			if rows.Scan(&m.ID, &m.NameEN, &m.ImageURL) == nil && m.NameEN != "" {
				maps = append(maps, m)
			}
		}
		_ = rows.Close()
	}
	if rows, qerr := metaDB.Query(ctx,
		`SELECT weapon_id::VARCHAR, name_en, COALESCE(icon_url, '')
		 FROM weapon_labels ORDER BY name_en`); qerr == nil {
		for rows.Next() {
			var w canonical.AssetMeta
			if rows.Scan(&w.ID, &w.NameEN, &w.ImageURL) == nil && w.NameEN != "" {
				weapons = append(weapons, w)
			}
		}
		_ = rows.Close()
	}
	// Médailles : icône SPRITE (feuille + offset) depuis medal_definitions.
	if rows, qerr := metaDB.Query(ctx,
		`SELECT medal_name_id::VARCHAR, name_en, COALESCE(sprite_sheet_url, ''),
		        COALESCE(sprite_left, 0), COALESCE(sprite_top, 0),
		        COALESCE(sprite_width, 0), COALESCE(sprite_height, 0)
		 FROM medal_definitions ORDER BY name_en`); qerr == nil {
		for rows.Next() {
			var m canonical.AssetMeta
			if rows.Scan(&m.ID, &m.NameEN, &m.SpriteSheet,
				&m.SpriteLeft, &m.SpriteTop, &m.SpriteWidth, &m.SpriteHeight) == nil && m.NameEN != "" {
				medals = append(medals, m)
			}
		}
		_ = rows.Close()
	}
	return maps, weapons, medals
}

// loadCSRBadgeResolver construit un résolveur d'insignes CSR pour un titre
// additionnel depuis sa metadata (csr_designations → icon_url par designation+tier,
// URLs CDN officielles). Retourne nil si la DB est absente / vide ; le résolveur ne
// répond QUE pour `slug` (autres titres → "" → chemin HINF inchangé).
func loadCSRBadgeResolver(pr *titlePkg.PathResolver, slug string) func(string, string, int) string {
	metaDB, err := platform_duckdb.OpenReadWriteShared(pr.MetadataDBPath(slug))
	if err != nil {
		return nil
	}
	defer func() { _ = metaDB.Close() }()
	m := map[string]string{}
	if rows, qerr := metaDB.Query(context.Background(),
		`SELECT designation_name, tier_id, COALESCE(icon_url, '') FROM csr_designations`); qerr == nil {
		for rows.Next() {
			var name, url string
			var tier int
			if rows.Scan(&name, &tier, &url) == nil && url != "" {
				m[fmt.Sprintf("%s|%d", strings.ToLower(name), tier)] = url
			}
		}
		_ = rows.Close()
	}
	if len(m) == 0 {
		return nil
	}
	return func(titleSlug, designation string, subTier int) string {
		if titleSlug != slug {
			return ""
		}
		return m[fmt.Sprintf("%s|%d", strings.ToLower(designation), subTier)]
	}
}

// loadTitleRankImageURLs charge les images de rang carrière d'un titre additionnel
// depuis sa metadata.duckdb ISOLÉE (career_ranks.large_icon_path/icon_path). Pattern
// best-effort calqué sur loadTitleAssetDrawerData : DB absente / table vide / erreur
// → map vide (jamais fatal). Le titre du joueur reçoit SA map (keyée par ses propres
// numéros de rang) ; un titre sans image de rang par niveau (Halo 5, SR en chiffre)
// obtient simplement une map vide. Title-agnostic : le slug est un paramètre, jamais
// une comparaison littérale dans le data-path.
func loadTitleRankImageURLs(pr *titlePkg.PathResolver, slug string) map[int]*string {
	metaDB, err := platform_duckdb.OpenReadWriteShared(pr.MetadataDBPath(slug))
	if err != nil {
		return nil
	}
	defer func() { _ = metaDB.Close() }()
	imgs, err := platform_duckdb.LoadCareerRankImageURLs(context.Background(), metaDB, slug)
	if err != nil {
		return nil
	}
	return imgs
}

//nolint:gocyclo // Routeur central : mount de ~80 endpoints avec feature flags
func NewRouter(
	serverCtx context.Context,
	cfg *config.AppConfig,
	bootRepo port.BootstrapRepository,
	bootSvc *service.BootstrapService,
	daemon watcher.DaemonController,
	tokenProvider auth_platform.TokenProvider,
	autoSyncScheduler *scheduler.AutoSyncScheduler,
	backupScheduler *duckdbbackup.Scheduler,
	groupStore *groupstore.GroupStore,
) (http.Handler, *ServiceRegistry) {
	if tokenProvider == nil {
		tokenProvider = auth_platform.NewMSALProvider()
	}
	// Sprint 14 : session store + Sprint 15 : attempt store auth
	// Le flag Secure du cookie + HSTS sont décidés PAR REQUÊTE selon le schéma réel
	// (TLS natif ou X-Forwarded-Proto derrière un proxy de confiance), pas figés au
	// boot : ne plus coupler « secret custom » à « HTTPS » (sinon cookie Secure jeté
	// sur http://localhost → onboarding bloqué). Override via LEVELUP_COOKIE_SECURE.
	cookiePolicy := middleware.SecureCookiePolicy{Mode: cfg.CookieSecure, TrustProxy: cfg.TrustProxyHeaders}
	sessionStore := session_platform.NewStore(cfg.SessionDir, session_platform.DefaultTTL, cfg.SessionSecret)
	// Purge périodique des sessions expirées (TTL dépassé). Sans ça, data/sessions/
	// accumule indéfiniment. Purge immédiate au boot (rattrape le backlog), puis
	// toutes les 6h ; arrêt propre au shutdown via serverCtx.
	go func() {
		if n := sessionStore.PurgeExpired(); n > 0 {
			slog.InfoContext(serverCtx, "session: purge initiale sessions expirées", "removed", n)
		}
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-serverCtx.Done():
				return
			case <-ticker.C:
				if n := sessionStore.PurgeExpired(); n > 0 {
					slog.InfoContext(serverCtx, "session: purge sessions expirées", "removed", n)
				}
			}
		}
	}()
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
	// Capture l'adresse TCP réelle du peer AVANT tout middleware susceptible de
	// réécrire RemoteAddr : le garde LoopbackOnly des endpoints /_diag s'appuie
	// dessus pour ne pas être falsifiable via un en-tête X-Real-IP/X-Forwarded-For.
	r.Use(middleware.PreserveRemoteAddr)
	// chi RealIP réécrit RemoteAddr depuis les en-têtes d'IP client : on ne
	// l'active QUE derrière un reverse proxy de confiance qui assainit ces en-têtes
	// (LEVELUP_TRUST_PROXY_HEADERS=1). Sinon RemoteAddr reste le peer TCP réel, ce
	// qui empêche un client externe d'usurper une IP (rate-limit, logs d'audit et
	// LoopbackOnly non falsifiables). Revue P0 2026-06-02 (faille RealIP/LoopbackOnly).
	if cfg.TrustProxyHeaders {
		r.Use(chimiddleware.RealIP)
	}
	// En-têtes de sécurité HTTP sur toutes les réponses (HSTS seulement en prod/TLS).
	r.Use(middleware.SecurityHeaders(cfg.TrustProxyHeaders))
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.CSRF(cfg.CORSOrigins))
	r.Use(middleware.RateLimit(cfg.DemoMode, cfg.RateLimitRPM))
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.WithSession(sessionStore, cookiePolicy))

	// Sprint 44 : TitleExtractor — injecte title_slug dans le contexte.
	// MT-16 / day-one 2e titre : registre PARTAGÉ piloté par config (built-in
	// halo_infinite + titres additionnels découverts en config/titles). Posé par
	// SetDefaultRegistry au boot ; retombe sur le built-in mono-titre en test/CLI.
	titleRegistry := titlePkg.DefaultRegistry()
	r.Use(middleware.TitleExtractor(titleRegistry))

	// Phase A multi-titres : chargement des FieldMappingSet TOML par titre.
	// Erreur de chargement → log mais ne bloque pas le boot (les autres titres
	// restent disponibles). L'endpoint /field-mappings n'est exposé que si le
	// flag MULTI_TITLE_API_ENABLED est activé. Les slugs viennent du registre
	// (non-archivés) → un 2e titre config charge automatiquement ses mappings.
	fieldMappingsRegistry := mappings.NewRegistry()
	multiTitleSlugs := make([]string, 0)
	for _, td := range titleRegistry.NonArchived() {
		multiTitleSlugs = append(multiTitleSlugs, td.Slug)
	}
	if len(multiTitleSlugs) == 0 {
		multiTitleSlugs = []string{titlePkg.DefaultSlug}
	}
	for _, err := range fieldMappingsRegistry.LoadFromConfigDir(cfg.RepoRoot, multiTitleSlugs, slog.Default()) {
		slog.Warn("field_mappings_load_warning", "err", err.Error())
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
					slog.Error("required_toml_missing",
						"title", td.Slug, "path", m.Path, "required_by", m.RequiredBy)
				} else {
					slog.Error("required_toml_missing", "title", td.Slug, "err", e.Error())
				}
			}
			if td.IsActive() {
				slog.Error("boot_validation_failed", "title", td.Slug, "errors_count", len(errs))
				os.Exit(1)
			}
		}
	}

	// Phase B multi-titres : resolver des adapters par titre.
	// Le resolver est exposé aux services produit qui veulent consommer la
	// couche canonique.
	titleResolver := games.NewStaticResolver(titlePkg.DefaultSlug)
	var hiRanks *mappings.RankCatalog
	// Title-agnostic (D.2) : images de rang carrière PAR TITRE (slug → rank_id →
	// imageURL). La map HINF est keyée par numéros de rang HINF (1..272) ; l'injecter
	// telle quelle dans le CareerService d'un joueur Halo 5 (RankNumber = SR 1..152)
	// renvoyait une image de rang HINF pour un SR. On charge donc une map par titre et
	// on n'injecte que celle du titre du joueur (Halo 5 → vide, le SR s'affiche en chiffre).
	rankImageURLsByTitle := make(map[string]map[int]*string)
	if hiFields, ok := fieldMappingsRegistry.Get(titlePkg.DefaultSlug); ok {
		// Charger le catalog des rangs HI depuis metadata.duckdb (career_rank_translations).
		// OpenReadWriteShared est cached par path → réutilise le pool existant ouvert dans
		// cmd/server. IMPORTANT : Close() pour décrémenter le refCount sinon le sql.DB
		// reste ouvert au shutdown (le metaDB.Close() de cmd/server décrémente seulement
		// d'un cran), ce qui retient le HANDLE Windows et provoque le verrou
		// "metadata verrouillée" au prochain hot-reload Air.
		// En démo, la metadata title est une coquille vide ; metadataDBPathFor
		// redirige vers les fixtures démo (sinon RankCatalog vide → rangs en EN).
		hiMetaPath := metadataDBPathFor(cfg)
		if metaDB, err := platform_duckdb.OpenReadWriteShared(hiMetaPath); err == nil {
			if catalog, err := platform_duckdb.LoadRankCatalog(context.Background(), metaDB); err == nil {
				hiRanks = catalog
				slog.Info("rank_catalog_loaded", "title_slug", titlePkg.DefaultSlug, "ranks", catalog.Len())
			} else {
				slog.Warn("rank_catalog_load_failed", "err", err.Error())
			}
			if imgs, err := platform_duckdb.LoadCareerRankImageURLs(context.Background(), metaDB, titlePkg.DefaultSlug); err == nil {
				rankImageURLsByTitle[titlePkg.DefaultSlug] = imgs
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
	// Title-agnostic (D.2) : images de rang carrière des TITRES ADDITIONNELS actifs.
	// Itère le registre (clé de map, jamais de comparaison littérale dans le data-path)
	// et charge la metadata ISOLÉE de chaque titre non-défaut depuis son chemin propre.
	// Best-effort comme loadTitleAssetDrawerData : DB absente / table vide → map vide
	// (Halo 5 attendu vide : aucune image de rang SR → le SR s'affiche en chiffre, plus
	// d'image de rang HINF erronée). En démo on saute (metadata title = coquille vide).
	if !cfg.DemoMode {
		rankImgPathResolver := titlePkg.NewPathResolver(cfg.RepoRoot)
		for _, td := range titleRegistry.Active() {
			if td.Slug == titlePkg.DefaultSlug {
				continue // HINF déjà chargé ci-dessus (byte-identique)
			}
			// Gate title-agnostic (capability, jamais slug) : ne charger les images de
			// rang QUE pour un titre déclarant career.rank_catalog (rangs = catalogue
			// table-backed avec icône par palier). Un titre à rang numérique (Halo 5, SR)
			// ne la déclare pas → on n'ouvre NI ne requête sa metadata (pas de table
			// career_ranks chez lui) : zéro requête échouée, décision prise en amont.
			hasRankCatalog := false
			if cset, ok := fieldMappingsRegistry.GetCapabilities(td.Slug); ok {
				if caps, err := games.CapabilityMapFromMappings(cset); err == nil {
					hasRankCatalog = caps.Has(games.CapCareerRankCatalog)
				}
			}
			if !hasRankCatalog {
				slog.Debug("rank_image_urls_skipped", "title_slug", td.Slug, "reason", "no career.rank_catalog")
				continue
			}
			imgs := loadTitleRankImageURLs(rankImgPathResolver, td.Slug)
			rankImageURLsByTitle[td.Slug] = imgs // map vide acceptée (titre sans image de rang)
			slog.Info("rank_image_urls_loaded", "title_slug", td.Slug, "images", len(imgs))
		}
	}
	// Phase 1.7a : capabilities chargées depuis capabilities.toml (source nominale,
	// remplace la map codée en dur de l'adapter). nil si TOML absent → l'adapter
	// retombe sur son fallback (parité garantie par capabilities_parity_test.go).
	var hiCaps games.CapabilityMap
	if cset, ok := fieldMappingsRegistry.GetCapabilities(titlePkg.DefaultSlug); ok {
		if caps, err := games.CapabilityMapFromMappings(cset); err == nil {
			hiCaps = caps
		} else {
			slog.Warn("capabilities_convert_failed", "title_slug", titlePkg.DefaultSlug, "err", err.Error())
		}
	}

	// Phase C : DataAdapter HI registré sans CareerSource player-scoped au boot.
	// La capability career.progression sera "not_exposed" pour ce DataAdapter
	// global ; les futurs handlers player-scoped instancieront leur propre
	// DataAdapter avec le CareerRepo du joueur courant via un MiddleWare DI.
	hiData := halo_games.NewDataAdapter(nil, slog.Default())
	if hiCaps != nil {
		hiData = hiData.WithCapabilities(hiCaps)
	}
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

	// PMT-11 / MT-26 : câble le resolver de libellés Discord title-aware. Les embeds
	// (outcomes win/loss + footer « LevelUp · {titre} Stats ») suivent le titre via
	// son adapter sémantique (outcomes.toml) + son nom de registre. Titre/adapter
	// absent → HaloLabels (byte-identique Halo). Les call-sites notify injectent
	// `cfg.Labels = notify.LabelsForSlug(slug)`.
	notify.SetDefaultLabelsResolver(func(slug string) notify.NotifyLabels {
		sem, err := titleResolver.Semantic(slug)
		if err != nil || sem == nil {
			return nil
		}
		name := ""
		if d := titleRegistry.Get(slug); d != nil {
			name = d.Name
		}
		return notify.LabelsFor(sem, name)
	})

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
		WithCapabilities(hiCaps).
		WithSettingsStore(settingsStore).
		WithRankCatalog(hiRanks).
		WithRankImageURLsByTitle(rankImageURLsByTitle)

	// MT-09 (PMT-12) : factory player-scoped de Halo enregistrée par SLUG (clé de
	// map, pas de comparaison littérale). Le builder lit reg.hiCapabilities à la
	// volée (posé ci-dessus). Un 2e titre enregistrerait ICI son propre builder,
	// sans toucher aux factories dataAdapterForPDB/TitleDataAdapter.
	reg.RegisterPlayerDataBuilder(titlePkg.DefaultSlug, func(pdb *platform_duckdb.PlayerDB) games.TitleDataAdapter {
		a := halo_games.NewDataAdapter(platform_duckdb.NewCareerRepo(pdb), slog.Default())
		if reg.hiCapabilities != nil {
			a = a.WithCapabilities(reg.hiCapabilities)
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
	registerAdditionalTitles(titleRegistry, titleResolver, reg, fieldMappingsRegistry)

	// Module Prestige — initialisation du bundle (best-effort, désactivable via flag).
	// Charge tuning.toml + templates + preset arcs Halo, ouvre shared_social et metadata.
	// Si le flag PRESTIGE_ENABLED est désactivé, les routes ne sont pas montées et
	// le sync hook est no-op — mais le boot du bundle reste utile pour valider la
	// config au démarrage.
	var prestigeBundle *PrestigeBundle
	if pb, err := NewPrestigeBundle(cfg.RepoRoot, reg.resolve, cfg.PrestigeEnabled); err != nil {
		slog.Warn("prestige_bundle_init_failed", "err", err.Error())
	} else {
		prestigeBundle = pb.WithSquadProfile(newSquadPerfProfileProvider(
			func() ([]domain.PlayerSummary, error) { return cfg.LoadPlayers() },
			reg.resolve,
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

	// AssetMetadataHandler — listing maps & armes pour l'Asset Drawer.
	// Stratégie in-memory : on ouvre metadata.duckdb, on charge tout en RAM, on ferme
	// la connexion immédiatement. Avantage : aucun lock Windows persistant entre processus
	// (Air hot-reload). Retry 3× (500ms) pour absorber la fenêtre de chevauchement Air.
	var assetMetaHandler *handlers.AssetMetadataHandler
	{
		metaDBPath := metadataDBPathFor(cfg)
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
			// Titres additionnels (ex. Halo 5) : maps/armes + URLs d'image viennent de
			// leur metadata.duckdb isolée (seedée par cmd/h5-metadata-fetch depuis l'API
			// Metadata officielle). Title-aware via WithTitle ; les URLs DB priment sur
			// les builders HINF (cf. AssetService : ImageURL non vide conservée).
			h5Maps, h5Weapons, h5Medals := loadTitleAssetDrawerData(
				titlePkg.NewPathResolver(cfg.RepoRoot), halo5.TitleSlug)
			assetMetaHandler = handlers.NewAssetMetadataHandler(
				service.NewAssetService(
					service.NewStaticAssetMetaRepo(maps, weapons).
						WithTitle(halo5.TitleSlug, h5Maps, h5Weapons, h5Medals)).
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

	// Insignes CSR des titres additionnels (Halo 5 : table csr_designations, URLs
	// officielles) — rend l'image de rang CSR title-aware dans le builder de badge
	// (buildHomeSkillPeakBadgeURL). nil si pas de seed → HINF strictement inchangé.
	// Chargé UNE fois et réutilisé par l'adapter TitleAssetURLAdapter h5 ci-dessous.
	h5CSRResolver := loadCSRBadgeResolver(
		titlePkg.NewPathResolver(cfg.RepoRoot), halo5.TitleSlug)
	platform_duckdb.SetCSRBadgeResolver(h5CSRResolver)

	// C0 / G1 : TitleAssetURLAdapter pour Halo 5. Sans lui,
	// titleResolver.AssetURL("halo_5") renvoie ErrTitleNotResolved → assetURL==nil
	// sur la Match View → image de map + badge CSR absents. Adapter PUR (couche 3) :
	// les URLs CDN officielles (maps/armes) et le résolveur CSR sont injectés depuis
	// la metadata h5 DÉJÀ chargée (aucun accès DB dans l'adapter).
	{
		h5Maps, h5Weapons, h5Medals := loadTitleAssetDrawerData(
			titlePkg.NewPathResolver(cfg.RepoRoot), halo5.TitleSlug)
		if h5CSRResolver != nil || len(h5Maps) > 0 || len(h5Weapons) > 0 {
			h5AssetURL := halo5.NewAssetURLAdapter().
				WithMaps(h5Maps).
				WithWeapons(h5Weapons)
			if h5CSRResolver != nil {
				h5AssetURL = h5AssetURL.WithCSRResolver(
					func(designation string, subTier int) string {
						return h5CSRResolver(halo5.TitleSlug, designation, subTier)
					})
			}
			titleResolver.RegisterAssetURL(h5AssetURL)
			slog.Info("adapter_loaded",
				"title_slug", h5AssetURL.TitleSlug(),
				"kind", "asset_url",
				"maps", len(h5Maps),
				"weapons", len(h5Weapons),
				"csr", h5CSRResolver != nil,
			)
		}

		// G2/G7/D.1 : sprite médaille title-aware. Les médailles Halo 5 sont des
		// SPRITES (feuille + offset, medal_definitions) — aucun PNG par médaille
		// n'existe → <img> /static/medals/halo_5/{id}.png 404 partout hors Asset
		// Drawer. On câble un résolveur sprite title-keyé (miroir csrBadgeResolver)
		// peuplé depuis h5Medals DÉJÀ chargé. nil/vide → pas de câblage → HINF sert
		// les PNG strictement inchangé.
		h5MedalSprites := make(map[int64]static.MedalSprite, len(h5Medals))
		for _, m := range h5Medals {
			if m.SpriteSheet == "" {
				continue
			}
			id, perr := strconv.ParseInt(m.ID, 10, 64)
			if perr != nil {
				continue
			}
			h5MedalSprites[id] = static.MedalSprite{
				SheetURL: m.SpriteSheet,
				Left:     m.SpriteLeft,
				Top:      m.SpriteTop,
				Width:    m.SpriteWidth,
				Height:   m.SpriteHeight,
			}
		}
		if len(h5MedalSprites) > 0 {
			static.SetMedalSpriteResolver(func(titleSlug string, medalID int64) *static.MedalSprite {
				if titleSlug != halo5.TitleSlug {
					return nil
				}
				if sp, ok := h5MedalSprites[medalID]; ok {
					return &sp
				}
				return nil
			})
			slog.Info("medal_sprite_resolver_ready",
				"title_slug", halo5.TitleSlug,
				"sprites", len(h5MedalSprites),
			)
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
	healthH.Mount(r) // /health, /healthz, /readyz (racine, Huma)

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

	// PR 4 (fix routing OAuth) — Le redirect_uri Azure pointe sur le chemin racine
	// /auth/xbox/callback (sans /api/v1). Comme le SPA est servi par ce backend, ce
	// chemin tombait sur le catch-all `/*` (index.html) → le handler OAuth ne tournait
	// jamais → l'utilisateur retombait sur /login. On expose donc login + callback AUSSI
	// à la racine du routeur. Le handler est construit dans le groupe /api/v1 ci-dessous
	// (mêmes deps), et lié tardivement ici : NewRouter s'exécute entièrement avant de
	// servir la moindre requête, donc l'assignation précède tout appel. Le middleware de
	// session est appliqué à la racine (cf. r.Use(WithSession) plus haut), donc l'OAuthState
	// est bien lu sur ces routes racine. Les alias /api/v1/auth/xbox/* restent pour le front.
	var xboxOAuthRoot *handlers.XboxOAuthHandler
	if cfg.AuthMode == "xbox" && cfg.OAuthRedirectURI != "" {
		r.Get("/auth/xbox/login", func(w http.ResponseWriter, req *http.Request) {
			if xboxOAuthRoot == nil {
				http.NotFound(w, req)
				return
			}
			xboxOAuthRoot.LoginRedirect(w, req)
		})
		r.Get("/auth/xbox/callback", func(w http.ResponseWriter, req *http.Request) {
			if xboxOAuthRoot == nil {
				http.NotFound(w, req)
				return
			}
			xboxOAuthRoot.Callback(w, req)
		})
	}

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
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
		help := handlers.NewHelpHandler(releaseBuilder, filepath.Join(cfg.RepoRoot, "data", "cache"))
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
			mountAdminMonitoringRoutes(r, reg, autoSyncScheduler, jobStore, serverCtx)
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
		syncH = syncH.WithPostSyncDeltaHook(buildPostSyncDeltaHook(reg))
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

		// Endpoints P1 : pages par joueur (Sprint 37 — DI via ServiceRegistry)
		r.Route("/players/{player_slug}", func(r chi.Router) {
			// MT-22 (PMT-8) : gate du cycle de vie du titre. Un titre courant
			// coming_soon/archived/inconnu → 503 title_unavailable (machine-readable)
			// au lieu de servir des données. No-op aujourd'hui (seul halo_infinite
			// actif). Avant la garde de propriété : indisponibilité du titre =
			// plus fondamental que l'appartenance du joueur.
			r.Use(middleware.RequireActiveTitle(titleRegistry))

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

			// Sprint 33 : Session Compare
			sc := handlers.NewSessionCompareHandler(reg.SessionCompare)
			sc.Mount(r)

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
				return reg.resolve(ctx, slug)
			}
			progressionH := handlers.NewProgressionHandler(progressionResolve, titlePkg.DefaultSlug).
				WithDemoMode(cfg.DemoMode)
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
				lazy := NewLazyPrestigeService(prestigeBundle, nil, cfg.DemoMode)
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
	})

	// SPA React : sert le build Vite (LEVELUP_WEB_DIST, cf. Dockerfile/compose) en
	// catch-all `/*`. chi route d'abord les patterns plus spécifiques (/api/v1/*,
	// /static/*, /health*, /debug/vars…), donc ce handler ne reçoit que les requêtes
	// front : un fichier du dist → servi tel quel ; sinon (route client-side React
	// Router, ou dossier) → index.html. Inactif si WebDistDir vide ou index.html
	// absent (dev : Vite sert le front sur :5173).
	if dist := cfg.WebDistDir; dist != "" {
		indexPath := filepath.Join(dist, "index.html")
		if _, statErr := os.Stat(indexPath); statErr == nil {
			fileServer := http.FileServer(http.Dir(dist))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				if fi, err := os.Stat(filepath.Join(dist, filepath.Clean(req.URL.Path))); err == nil && !fi.IsDir() {
					fileServer.ServeHTTP(w, req)
					return
				}
				// Route client-side / dossier / fichier absent → index.html. no-cache
				// pour que l'index ne masque pas un nouveau build après redéploiement.
				w.Header().Set("Cache-Control", "no-cache")
				http.ServeFile(w, req, indexPath)
			})
			slog.InfoContext(serverCtx, "SPA: front React servi depuis le dist", "dir", dist)
		} else {
			slog.WarnContext(serverCtx, "LEVELUP_WEB_DIST défini mais index.html introuvable — SPA non montée", "dir", dist)
		}
	}

	return r, reg
}
