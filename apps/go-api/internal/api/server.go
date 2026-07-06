// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
// Sprint 37 : Architecture handlers & injection DI via wire.ServiceRegistry.
// ContractValidate (dev) retiré L4 (2026-07-05 : Huma dérive le contrat). ErrorTracker retiré P8.3 (ADR 0009).
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
	"levelup/go-api/internal/api/wire"
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

	// init() de observability publie le namespace expvar "levelup" ; le handler
	// /debug/vars (stdlib) découvre ces compteurs via http.DefaultServeMux (P8.3,
	// ADR 0009). Import nommé depuis J1 : PublishDuckDBPoolStats (stats de pool).
	"levelup/go-api/internal/observability"
	auth_platform "levelup/go-api/internal/platform/auth"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/halo"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
	"levelup/go-api/internal/worldenrich"
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
			// Lot B (audit #8) : ne pas dégrader en owner-only SANS trace — un
			// groups.json corrompu supprimerait silencieusement tous les grants
			// famille (ADR 0029) → 403 inexplicables. Loguer AVANT le repli sûr.
			slog.ErrorContext(ctx, "family resolver: lecture groups.json échouée — accès cross-membre dégradé en owner-only", "xuid", user.XUID, "err", err)
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
// Délègue à config.MetadataDBPath (source unique de la redirection démo title-scopée).
func metadataDBPathFor(cfg *config.AppConfig) string {
	return metadataDBPathForTitle(cfg, titlePkg.DefaultSlug)
}

// metadataDBPathForTitle : variante title-aware (un titre additionnel comme Halo 5
// lit SA metadata démo title-scopée data/demo/titles/{slug}/warehouse/metadata.duckdb).
func metadataDBPathForTitle(cfg *config.AppConfig, titleSlug string) string {
	return config.MetadataDBPath(cfg, titleSlug)
}

// NewRouter construit le routeur chi avec tous les endpoints.
// Construction par injection de dépendances — pas d'état global.
// daemon peut être nil si le watcher n'est pas actif au démarrage.
// tokenProvider peut être nil : MSALProvider est utilisé par défaut.
// Retourne aussi le *wire.ServiceRegistry pour permettre au démon watcher de lier le TTL dynamique.
//
// conditionnels (MULTI_TITLE_API_ENABLED, PRESTIGE_ENABLED, etc.). Complexité
// reflète la surface API, pas un défaut de conception.
//
// loadTitleAssetDrawerData charge les maps + armes (avec leurs URLs d'image) d'un
// titre additionnel depuis sa metadata.duckdb ISOLÉE, pour l'Asset Drawer
// (référentiel). Peuplé par cmd/h5-metadata-fetch (API Metadata officielle :
// maps_catalog.image_url + weapon_labels.icon_url). Best-effort : DB absente / tables
// vides → slices nil (le titre n'apparaît simplement pas dans le drawer).
func loadTitleAssetDrawerData(metaPath, slug string) (maps, weapons, medals []canonical.AssetMeta) {
	metaDB, err := platform_duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = metaDB.Close() }()
	return platform_duckdb.LoadTitleAssetDrawerData(context.Background(), metaDB, slug)
}

// loadCSRBadgeResolver construit un résolveur d'insignes CSR pour un titre
// additionnel depuis sa metadata (csr_designations → icon_url par designation+tier,
// URLs CDN officielles). Retourne nil si la DB est absente / vide ; le résolveur ne
// répond QUE pour `slug` (autres titres → "" → chemin HINF inchangé).
func loadCSRBadgeResolver(metaPath, slug string) func(string, string, int) string {
	metaDB, err := platform_duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return nil
	}
	defer func() { _ = metaDB.Close() }()
	m := platform_duckdb.LoadCSRBadgeMap(context.Background(), metaDB)
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

// buildAssetMetadataHandler construit l'AssetMetadataHandler (drawer d'assets) : charge
// maps/armes/médailles Infinite depuis metadata.duckdb (in-memory, retry 3× × 500ms pour
// absorber la fenêtre de chevauchement Air) + câble le titre additionnel (Halo 5) via
// WithTitle. Extrait de NewRouter (K2a, 2026-07-06). Retourne nil si metadata reste
// indisponible après les 3 tentatives (drawer désactivé, non fatal).
func buildAssetMetadataHandler(cfg *config.AppConfig, hiAssetURL *halo_games.AssetURLAdapter,
	titleRegistry *titlePkg.Registry, h5Maps, h5Weapons, h5Medals []canonical.AssetMeta,
) *handlers.AssetMetadataHandler {
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
		// Médailles Infinite : sans ce chargement, le tab « Médailles » du drawer reste
		// VIDE pour Infinite. Noms FR/EN depuis medal_definitions ; icône = PNG
		// /static/medals/halo_infinite/{id}.png (résolu via static.MedalImage).
		medals, errMed := liveRepo.ListMedalsByTitle(loadCtx, titlePkg.DefaultSlug, "")
		_ = metaDB.Close() // relâche le lock Windows immédiatement après chargement
		if errM != nil || errW != nil || errMed != nil {
			slog.Warn("asset_metadata_load_failed", "err_maps", errM, "err_weapons", errW, "err_medals", errMed)
			continue
		}
		for i := range medals {
			if id, perr := strconv.ParseInt(medals[i].ID, 10, 64); perr == nil {
				if png, _ := static.MedalImage(titlePkg.DefaultSlug, id); png != "" {
					medals[i].ImageURL = png
				}
			}
		}
		// Titres additionnels (H5) : maps/armes + URLs depuis leur metadata isolée
		// (h5Maps/h5Weapons/h5Medals chargés une fois par l'appelant). Title-aware via
		// WithTitle ; les URLs DB priment sur les builders HINF.
		h := handlers.NewAssetMetadataHandler(
			service.NewAssetService(
				service.NewStaticAssetMetaRepo(maps, weapons).
					WithFallbackMedals(medals).
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
			"maps", len(maps), "weapons", len(weapons), "medals", len(medals),
			"attempt", attempt+1,
		)
		return h
	}
	return nil
}

// wireHalo5AssetAdapters câble les résolveurs d'assets title-aware pour Halo 5 : badge CSR
// (csr_designations → SetCSRBadgeResolver), TitleAssetURLAdapter (maps/armes CDN + CSR sur
// titleResolver) et sprites de médailles (SetMedalSpriteResolver). Tout best-effort :
// nil/vide → HINF strictement inchangé. Extrait de NewRouter (K2a). Les assets h5
// (h5Maps/h5Weapons/h5Medals) sont chargés une fois par l'appelant.
func wireHalo5AssetAdapters(cfg *config.AppConfig, titleResolver *games.StaticResolver,
	h5Maps, h5Weapons, h5Medals []canonical.AssetMeta,
) {
	h5CSRResolver := loadCSRBadgeResolver(config.MetadataDBPath(cfg, halo5.TitleSlug), halo5.TitleSlug)
	platform_duckdb.SetCSRBadgeResolver(h5CSRResolver)

	// C0 / G1 : TitleAssetURLAdapter pour Halo 5. Sans lui, titleResolver.AssetURL("halo_5")
	// renvoie ErrTitleNotResolved → assetURL==nil sur la Match View. Adapter PUR (couche 3).
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
			"title_slug", h5AssetURL.TitleSlug(), "kind", "asset_url",
			"maps", len(h5Maps), "weapons", len(h5Weapons), "csr", h5CSRResolver != nil)
	}

	// G2/G7/D.1 : sprite médaille title-aware. Les médailles H5 sont des SPRITES (feuille +
	// offset) — aucun PNG par médaille → résolveur sprite title-keyé peuplé depuis h5Medals.
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
		slog.Info("medal_sprite_resolver_ready", "title_slug", halo5.TitleSlug, "sprites", len(h5MedalSprites))
	}
}

// titleRuntime regroupe les sorties du boot multi-titres (Phase B) consommees par NewRouter.
type titleRuntime struct {
	resolver   *games.StaticResolver
	hiRanks    *mappings.RankCatalog
	rankImages map[string]map[int]*string
	hiCaps     games.CapabilityMap
	hiAssetURL *halo_games.AssetURLAdapter
}

// buildTitleRuntime construit le resolver d adapters par titre (semantic/data/assetURL
// Halo Infinite + images de rang par titre actif) et les capabilities HI. Extrait de
// NewRouter (K2a, 2026-07-06) : deplacement pur du bloc Phase B multi-titres.
func buildTitleRuntime(serverCtx context.Context, cfg *config.AppConfig, titleRegistry *titlePkg.Registry, fieldMappingsRegistry *mappings.Registry) titleRuntime {
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
			if catalog, err := platform_duckdb.LoadRankCatalog(context.Background(), metaDB, titlePkg.DefaultSlug); err == nil {
				hiRanks = catalog
				slog.Info("rank_catalog_loaded", "title_slug", titlePkg.DefaultSlug, "ranks", catalog.Len())
			} else {
				slog.Warn("rank_catalog_load_failed", "err", err)
			}
			if imgs, err := platform_duckdb.LoadCareerRankImageURLs(context.Background(), metaDB, titlePkg.DefaultSlug); err == nil {
				rankImageURLsByTitle[titlePkg.DefaultSlug] = imgs
				slog.Info("rank_image_urls_loaded", "title_slug", titlePkg.DefaultSlug, "images", len(imgs))
			} else {
				slog.Warn("rank_image_urls_load_failed", "err", err)
			}
			if closeErr := metaDB.Close(); closeErr != nil {
				slog.Warn("rank_catalog_meta_db_close_failed", "err", closeErr)
			}
		} else {
			slog.Warn("rank_catalog_meta_db_open_failed", "err", err)
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
			slog.ErrorContext(serverCtx, "adapter_load_failed",
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
			slog.Warn("capabilities_convert_failed", "title_slug", titlePkg.DefaultSlug, "err", err)
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

	return titleRuntime{
		resolver:   titleResolver,
		hiRanks:    hiRanks,
		rankImages: rankImageURLsByTitle,
		hiCaps:     hiCaps,
		hiAssetURL: hiAssetURL,
	}
}

// startSessionPurgeLoop lance la purge périodique des sessions expirées (TTL
// dépassé) : purge immédiate au boot (rattrape le backlog) puis toutes les 6h,
// arrêt propre au shutdown via serverCtx. Extrait de NewRouter (K2a — réduction
// du god-func d'assemblage DI).
//
//nolint:gocyclo // Routeur central : mount de ~80 endpoints avec feature flags
func startSessionPurgeLoop(serverCtx context.Context, sessionStore *session_platform.Store) {
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
}

// applyTransverseMiddlewares monte les middlewares transverses (l'ORDRE importe)
// sur le routeur racine : recovery, préservation du RemoteAddr, RealIP conditionnel
// (uniquement derrière un proxy de confiance), en-têtes de sécurité, RequestID,
// CORS, CSRF, rate-limit, logs slog, compression, session, puis TitleExtractor
// (injecte title_slug dans le contexte). Extrait de NewRouter (K2a).
func applyTransverseMiddlewares(
	r chi.Router,
	cfg *config.AppConfig,
	sessionStore *session_platform.Store,
	cookiePolicy middleware.SecureCookiePolicy,
	titleRegistry *titlePkg.Registry,
) {
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
	// Sprint 44 : TitleExtractor — injecte title_slug dans le contexte (registre
	// PARTAGÉ piloté par config, MT-16 / day-one 2e titre).
	r.Use(middleware.TitleExtractor(titleRegistry))
}

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
) (http.Handler, *wire.ServiceRegistry) {
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
	// Purge périodique des sessions expirées (TTL dépassé) — cf. startSessionPurgeLoop.
	startSessionPurgeLoop(serverCtx, sessionStore)
	attemptStore := auth_platform.NewAttemptStore()

	// Sprint 16 : settings store + Sprint 17 : job store
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobsPath := titlePkg.NewPathResolver(cfg.RepoRoot).JobsCachePath()
	jobStore := jobs_platform.NewStore(jobsPath)

	// Auth locale : user store + invite store (mode password).
	usersPath := filepath.Join(cfg.AuthDir, "users.json")
	invitesPath := filepath.Join(cfg.AuthDir, "invites.json")
	users := userstore.NewStore(usersPath)
	invites := userstore.NewInviteStore(invitesPath)

	r := chi.NewRouter()

	// Registre de titres PARTAGÉ piloté par config (built-in halo_infinite + titres
	// additionnels découverts en config/titles ; posé par SetDefaultRegistry au boot,
	// retombe sur le built-in mono-titre en test/CLI). Créé ici car réutilisé dans
	// tout NewRouter (mappings, capabilities, TitleExtractor).
	titleRegistry := titlePkg.DefaultRegistry()
	// Middlewares transverses (l'ordre importe) — cf. applyTransverseMiddlewares.
	applyTransverseMiddlewares(r, cfg, sessionStore, cookiePolicy, titleRegistry)

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
	reg := wire.NewServiceRegistry(cfg, tokenProvider).
		WithTitleResolver(titleResolver).
		WithCapabilities(hiCaps).
		WithSettingsStore(settingsStore).
		WithRankCatalog(hiRanks).
		WithRankImageURLsByTitle(rankImageURLsByTitle)

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
	//
	// J1 (ADR 0009) : publie les sql.DBStats des pools DuckDB sous
	// "levelup"/"duckdb_pool_stats" (WaitCount/WaitDuration = contention pool).
	observability.PublishDuckDBPoolStats(func() any { return platform_duckdb.PoolStatsSnapshot() })
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
		xboxOAuthRoot = mountAPIV1(r, apiV1Deps{cfg: cfg, bootSvc: bootSvc, reg: reg, fieldMappingsRegistry: fieldMappingsRegistry, attemptStore: attemptStore, users: users, invites: invites, sessionStore: sessionStore, tokenProvider: tokenProvider, groupStore: groupStore, settingsStore: settingsStore, assetHandler: assetHandler, assetMetaHandler: assetMetaHandler, gamertagSvc: gamertagSvc, prestigeBundle: prestigeBundle, serverCtx: serverCtx, daemon: daemon, autoSyncScheduler: autoSyncScheduler, authStore: authStore, jobStore: jobStore, titleRegistry: titleRegistry, backupScheduler: backupScheduler})
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
				// Route client-side / dossier / fichier absent → index.html, avec
				// injection des balises Open Graph (apercus de liens sociaux).
				// no-cache géré dans serveIndexWithOG.
				reg.ServeIndexWithOG(w, req, indexPath)
			})
			slog.InfoContext(serverCtx, "SPA: front React servi depuis le dist", "dir", dist)
		} else {
			slog.WarnContext(serverCtx, "LEVELUP_WEB_DIST défini mais index.html introuvable — SPA non montée", "dir", dist)
		}
	}

	return r, reg
}
