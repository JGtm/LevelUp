// Package api — registry.go : câblage des services par injection de dépendances.
//
// Sprint 37 : ServiceRegistry centralise la construction des services
// à partir du PlayerDB résolu. Les handlers reçoivent des factory
// functions typées plutôt que cfg — testabilité et découplage.
//
// Le code est découpé en fichiers thématiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type
// ServiceRegistry, le constructeur, les Withers, les bundles, les adapters
// (semantic/asset/data) et les helpers minimums (Close, GetSessionNotifier).
// Les autres responsabilités vivent dans :
//
//   - registry_career.go : Career, Achievements, TitleDataAdapter, CareerLiveCtx,
//     CSRCoverageProvider, ProgressionDiagProvider +
//     buildFriendsXPLoader, allZeroXPTotal
//   - registry_pages.go  : Filters, MatchView, Engagement, Sessions, Stats,
//     Timeseries + tous les XxxCtx page-context +
//     newHomeRepo, playerMatchesAdapterFor,
//     buildFriendsExtrasResolver, friendGamertagsResolver
//   - registry_media.go  : Media, MediaUpload, MediaPlayerCtx, Social +
//     mediaWriterAcquirerFor
//   - registry_auth.go   : HomeCtxWithAuth, SeasonPassCtxWithAuth +
//     buildHaloProvider, enrichWithHaloTokens,
//     refreshTokensFromDB, oauthRefreshTokenForPlayer,
//     AnyPlayerTokens, RefreshTokensForXUID
package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	sync_pkg "levelup/go-api/internal/sync"
)

// haloProvider est l'instance globale du provider Halo (partagée).
var haloProvider = halo.DefaultHaloProvider

// PlayerResolver traduit un slug joueur en PlayerDB (pool-cached).
type PlayerResolver func(ctx context.Context, slug string) (*duckdb.PlayerDB, error)

// ServiceRegistry centralise la construction des services métier.
// Chaque méthode résout le joueur puis construit le service injecté.
type ServiceRegistry struct {
	cfg            *config.AppConfig
	resolve        PlayerResolver
	resolveByGT    duckdb.TitlePlayerResolver // résolution (titleSlug, gamertag) → PlayerDB pour Squad V2
	provider       auth.TokenProvider
	assetResolver  assets.Resolver     // nil si le resolver n'est pas configuré (mode legacy)
	timezone       string              // IANA (ex: "Europe/Paris"), propagé aux services médias
	notifiers      sync.Map            // xuid → port.SessionNotifier (HomeService par joueur)
	titleResolver  games.Resolver      // nil → services tournent sans semantic adapter (libellés via fallbacks)
	hiCapabilities games.CapabilityMap // capabilities.toml HI chargées au boot (Phase 1.7a) ; nil → adapter player-scoped retombe sur son fallback
	// MT-09 (PMT-12) : factory player-scoped PAR TITRE. Le slug est une CLÉ de
	// map (lookup title-agnostic), jamais une comparaison littérale — un 2e titre
	// enregistre son builder au boot, sans toucher aux factories. Remplace les
	// 2 cutoffs `pdb.TitleSlug != DefaultSlug` (allowlist archlint vidée).
	playerDataBuilders   map[string]func(*duckdb.PlayerDB) games.TitleDataAdapter
	homeMatchesCache     *service.HomeMatchesCache            // cache TTL process-level matches+sessions
	careerLiveCache      *service.CareerLiveCache             // cache TTL process-level XP (5 min) + customisation (6 h)
	remoteStats          *service.CachedStatsProvider         // cache TTL process-level stats carrière remote (5 min), partagé Explorer/Compare
	recentMatches        *service.CachedRecentMatchesProvider // cache TTL process-level 20 derniers matchs live (20 min), partagé Explorer/Compare (cibles non-locales)
	settingsStore        *settings_platform.Store             // nil → services qui dépendent des settings (TeammatesService friend filter) tournent en mode legacy
	seasonsCatalog       *service.SeasonsCatalog              // nil → FiltersService.Resolve ne renvoie pas SeasonCounts (dégradation gracieuse)
	rankCatalog          *mappings.RankCatalog                // nil → CareerService.next_rank_name reste vide
	rankImageURLsByTitle map[string]map[int]*string           // PAR TITRE (slug → rank_id → imageURL) ; map manquante/nil → CareerService.rank_image_url et next_rank_image_url absents pour ce titre. Title-agnostic : les images HINF (keyées 1..272) ne fuient plus sur un SR Halo 5 (D.2)
	prestigeBundle       *PrestigeBundle                      // nil si feature désactivée ; possède 2 *DB (sharedSocial + metadata) à fermer au shutdown
	advisorBundle        *CoachAdvisorBundle                  // nil → coach_advisor désactivé (ADR 0020 Phase 8)
	authStore            auth.UserTokenStore                  // ADR 0023 : source unique tokens auth (nil → fallback legacy DuckDB+env)
	jobStore             *jobs_platform.Store                 // nil → transcoding HLS désactivé (médias servis via remux WebM live)
	autoSyncScheduler    *scheduler.AutoSyncScheduler         // dashboard monitoring : snapshot scheduler agrégé dans l'overview (nil → section indisponible)
	healthScheduler      *scheduler.HealthScheduler           // dashboard monitoring : dernier audit data health + action run (nil → section indisponible)
	startedAt            time.Time                            // boot du process (uptime overview — posé par NewServiceRegistry)
	metaHandles          []*duckdb.DB                         // handles RW "annexes" sur metadata.duckdb (seasons/playlists catalog, ouverts hors pool joueur dans NewRouter) fermés au shutdown via Close() — sinon fuite de refCount sur le cache duckdb.openDBs (cf. INCIDENT_2026-05-21)
	snapReaders          sync.Map                             // titleSlug → *sync.SnapshotPreferredSharedReader (pilote lecture snapshot SCOPED MatchView) — singleton par titre, cache de queriers :memory: partagé entre requêtes, fermé au shutdown
}

// WithJobStore attache le JobStore au registry — porte le cycle de vie des jobs
// asynchrones (transcoding média HLS). Nil possible : la feature dégrade alors
// vers le remux WebM live (cf. service.WithMediaTranscoding).
func (r *ServiceRegistry) WithJobStore(store *jobs_platform.Store) *ServiceRegistry {
	r.jobStore = store
	return r
}

// WithAuthStore attache le MultiUserTokenStore au registry — source unique des
// tokens auth (RT + MSAL cache) post-ADR 0023. `refreshTokensFromDB` lit le store
// AVANT de tomber sur les fallbacks legacy (sync_meta DuckDB, env var). Nil
// possible pour les tests qui veulent l'ancien comportement legacy-only.
func (r *ServiceRegistry) WithAuthStore(store auth.UserTokenStore) *ServiceRegistry {
	r.authStore = store
	return r
}

// WithCoachAdvisorBundle attache le bundle coach_advisor au registry.
// Stateless côté ressources DB (le bundle ne détient qu'une grammaire en
// mémoire) — pas de Close nécessaire.
func (r *ServiceRegistry) WithCoachAdvisorBundle(b *CoachAdvisorBundle) *ServiceRegistry {
	r.advisorBundle = b
	return r
}

// CoachAdvisorBundle retourne le bundle si attaché, nil sinon.
func (r *ServiceRegistry) CoachAdvisorBundle() *CoachAdvisorBundle {
	if r == nil {
		return nil
	}
	return r.advisorBundle
}

// PrestigeBundle retourne le bundle Prestige si attaché, nil sinon.
// Exporté pour permettre au post-sync hook de construire les deps
// progression V2 avec coach_advisor.
func (r *ServiceRegistry) PrestigeBundle() *PrestigeBundle {
	if r == nil {
		return nil
	}
	return r.prestigeBundle
}

// SettingsStore retourne le store de settings si attaché, nil sinon.
// Exporté pour permettre au post-sync hook de lire CoachProactiveMode.
func (r *ServiceRegistry) SettingsStore() *settings_platform.Store {
	if r == nil {
		return nil
	}
	return r.settingsStore
}

// WithPrestigeBundle attache le bundle Prestige au registry. Le bundle détient
// des handles OpenReadWriteShared(sharedSocial) + OpenReadWriteShared(metadata)
// qui DOIVENT être fermés au shutdown via reg.Close() — sinon fuite de
// refCount sur le cache duckdb.openDBs (cf. INCIDENT_2026-05-21_metadata
// _duckdb_lock_air_hot_reload.md).
func (r *ServiceRegistry) WithPrestigeBundle(b *PrestigeBundle) *ServiceRegistry {
	r.prestigeBundle = b
	return r
}

// Close libère les ressources détenues par le registry — actuellement le
// PrestigeBundle (handles sharedSocial + metadata). Idempotent.
// Phase 2 plan stabilisation 2026-05-22 : appelé depuis cmd/server/main.go
// au shutdown AVANT metaDB.Close() pour décrémenter proprement le refCount
// sur metadata.
func (r *ServiceRegistry) Close() {
	if r == nil {
		return
	}
	if r.prestigeBundle != nil {
		r.prestigeBundle.Close()
		r.prestigeBundle = nil
	}
	// Resolver d'assets : flush la WriteQueue ET ferme l'index store DuckDB,
	// qui tient un handle RW persistant sur metadata.duckdb. Cf.
	// DefaultResolver.Close + INCIDENT_2026-05-21 (fuite refCount).
	if r.assetResolver != nil {
		_ = r.assetResolver.Close(context.Background())
		r.assetResolver = nil
	}
	// Handles metadata "annexes" (seasons/playlists catalog) ouverts hors pool
	// joueur dans NewRouter : chaque Close décrémente le refCount partagé du
	// cache duckdb. Sans ça ils restent orphelins au shutdown (verrou Air).
	for _, db := range r.metaHandles {
		if db != nil {
			_ = db.Close()
		}
	}
	r.metaHandles = nil
	// Readers snapshot MatchView (pilote scoped) : ferment leurs queriers :memory:
	// cachés. Sans ça, une recréation du registry (hot-reload) fuiterait les handles.
	r.snapReaders.Range(func(key, value any) bool {
		if c, ok := value.(interface{ Close() }); ok {
			c.Close()
		}
		r.snapReaders.Delete(key)
		return true
	})
}

// TrackMetadataHandle enregistre un handle RW "annexe" sur metadata.duckdb
// (ouvert hors pool joueur dans NewRouter, ex. seasons catalog, playlists
// catalog) pour qu'il soit fermé au shutdown via Close(). Sans ça, son refCount
// dans le cache duckdb.openDBs ne retombe jamais à 0 → handle Windows tenu
// jusqu'à exit process → verrou metadata.duckdb au prochain hot-reload Air
// (cf. docs/INCIDENT_2026-05-21_metadata_duckdb_lock_air_hot_reload.md).
func (r *ServiceRegistry) TrackMetadataHandle(db *duckdb.DB) *ServiceRegistry {
	if r == nil || db == nil {
		return r
	}
	r.metaHandles = append(r.metaHandles, db)
	return r
}

// NewServiceRegistry crée un ServiceRegistry câblé avec config.ResolvePlayer.
// Le titleSlug est lu depuis le contexte (ctxkeys.TitleSlug), fallback "halo_infinite".
func NewServiceRegistry(cfg *config.AppConfig, provider auth.TokenProvider) *ServiceRegistry {
	return &ServiceRegistry{
		cfg: cfg,
		resolve: func(ctx context.Context, slug string) (*duckdb.PlayerDB, error) {
			titleSlug := ctxkeys.TitleSlug(ctx)
			return config.ResolvePlayer(ctx, cfg, slug, titleSlug)
		},
		resolveByGT:      makeTitlePlayerResolver(cfg),
		provider:         provider,
		timezone:         cfg.UserTimezone,
		homeMatchesCache: service.NewHomeMatchesCache(),
		careerLiveCache:  service.NewCareerLiveCache(service.CareerLiveCacheConfig{}),
		remoteStats:      service.NewCachedStatsProvider(haloProvider, 0, nil),
		recentMatches:    service.NewCachedRecentMatchesProvider(sync_pkg.NewRecentMatchesFetcher(0), 0, nil),
		startedAt:        time.Now(),
	}
}

// makeTitlePlayerResolver construit la résolution (titleSlug, gamertag) →
// PlayerDB en s'appuyant sur cfg.LoadPlayers (lookup db_profiles.json) puis
// config.ResolvePlayer pour ouvrir/cacher la base via le pool global.
//
// Sert exclusivement à la page Squad V2, où les coéquipiers sont identifiés
// par leur gamertag (et pas par leur slug URL). Si le gamertag n'est pas
// référencé pour le titre demandé, retourne config.ErrPlayerNotFound — le
// SquadV2LoaderAdapter traduit ensuite en games.ErrCapabilityNotSupported.
func makeTitlePlayerResolver(cfg *config.AppConfig) duckdb.TitlePlayerResolver {
	return func(ctx context.Context, titleSlug, gamertag string) (*duckdb.PlayerDB, error) {
		players, err := cfg.LoadPlayers(titleSlug)
		if err != nil {
			return nil, fmt.Errorf("makeTitlePlayerResolver: load players for %q: %w", titleSlug, err)
		}
		for i := range players {
			if strings.EqualFold(players[i].Gamertag, gamertag) {
				return config.ResolvePlayer(ctx, cfg, players[i].PlayerSlug, titleSlug)
			}
		}
		return nil, fmt.Errorf("%w: title=%q gamertag=%q", config.ErrPlayerNotFound, titleSlug, gamertag)
	}
}

// WithAssetResolver attache le resolver unifié d'assets au ServiceRegistry.
// Quand présent, les providers Halo créés par HomeCtxWithAuth et SeasonPassCtxWithAuth
// délèguent le cache/fetch au resolver au lieu d'écrire directement dans DuckDB.
func (r *ServiceRegistry) WithAssetResolver(resolver assets.Resolver) *ServiceRegistry {
	r.assetResolver = resolver
	return r
}

// WithTitleResolver attache le resolver multi-titres (Phase B). Les services qui
// ont besoin du SemanticAdapter (libellés rangs, fields) le récupèrent via
// r.titleResolver.Semantic(slug).
func (r *ServiceRegistry) WithTitleResolver(resolver games.Resolver) *ServiceRegistry {
	r.titleResolver = resolver
	return r
}

// WithCapabilities attache la CapabilityMap HI chargée depuis capabilities.toml
// (Phase 1.7a). Injectée dans les DataAdapter player-scoped construits à la
// volée (dataAdapterForPDB / careerDataAdapterForPDB) pour qu'ils consomment le
// TOML plutôt que leur fallback codé.
func (r *ServiceRegistry) WithCapabilities(caps games.CapabilityMap) *ServiceRegistry {
	r.hiCapabilities = caps
	return r
}

// RegisterPlayerDataBuilder enregistre la factory player-scoped d'un titre
// (MT-09). Le builder reçoit le PlayerDB résolu et construit le TitleDataAdapter
// du titre (Halo : CareerRepo + capabilities). Title-agnostic : les factories
// `dataAdapterForPDB`/`TitleDataAdapter` font un simple lookup par slug, plus
// aucune comparaison littérale.
func (r *ServiceRegistry) RegisterPlayerDataBuilder(slug string, build func(*duckdb.PlayerDB) games.TitleDataAdapter) *ServiceRegistry {
	if r.playerDataBuilders == nil {
		r.playerDataBuilders = make(map[string]func(*duckdb.PlayerDB) games.TitleDataAdapter)
	}
	r.playerDataBuilders[slug] = build
	return r
}

// WithSettingsStore attache un settings.Store au registry. Les services qui
// dépendent de app_settings.json (TeammatesService.friendGamertags pour le
// filtre amis-only du dropdown) le récupèrent via r.settingsStore.
func (r *ServiceRegistry) WithSettingsStore(store *settings_platform.Store) *ServiceRegistry {
	r.settingsStore = store
	return r
}

// WithSeasonsCatalog attache le résolveur unifié des saisons (V2 saisons :
// pattern lazy-fetch + persist symétrique au battle pass). Le catalog merge
// TOML (libellés FR + display_order) et DB (dates fraîches Waypoint), avec
// fallback live via SeasonProvider quand la DB est vide. Si non câblé →
// FiltersService.Resolve ne renvoie pas SeasonCounts (frontend tombe en mode
// "saisons sans counts" sans folding, dégradation gracieuse).
func (r *ServiceRegistry) WithSeasonsCatalog(catalog *service.SeasonsCatalog) *ServiceRegistry {
	r.seasonsCatalog = catalog
	return r
}

// WithRankCatalog attache le catalog des rangs carrière (libellés FR/EN/etc.).
// Consommé par CareerService pour résoudre next_rank_name. Si nil, ce champ
// reste vide dans la réponse API (dégradation gracieuse).
func (r *ServiceRegistry) WithRankCatalog(catalog *mappings.RankCatalog) *ServiceRegistry {
	r.rankCatalog = catalog
	return r
}

// WithRankImageURLsByTitle attache les maps rank_id → imageURL chargées au
// démarrage depuis career_ranks (metadata.duckdb), INDEXÉES PAR SLUG DE TITRE.
// Consommée par CareerService (via registry_career.Career) pour rank_image_url
// et next_rank_image_url. Le slug est une clé de map (lookup title-agnostic) :
// un joueur Halo 5 reçoit la map de son titre (vide → pas d'image, le SR
// s'affiche en chiffre) au lieu des images de rang HINF erronées (D.2).
// nil/clé absente → ces champs sont absents pour ce titre (dégradation gracieuse).
func (r *ServiceRegistry) WithRankImageURLsByTitle(byTitle map[string]map[int]*string) *ServiceRegistry {
	r.rankImageURLsByTitle = byTitle
	return r
}

// SeasonsCatalogForHandler retourne un resolver compatible avec
// handlers.SeasonsCatalogResolver, ou nil si le catalog n'est pas câblé.
//
// L'adaptation projette service.SeasonCatalogEntry → handlers.SeasonCatalogEntry
// (champs minimaux nécessaires au DTO assets). Cette frontière maintient le
// handler découplé de l'implémentation concrète du catalog (testabilité +
// pas d'import service côté handlers).
func (r *ServiceRegistry) SeasonsCatalogForHandler() handlers.SeasonsCatalogResolver {
	if r.seasonsCatalog == nil {
		return nil
	}
	return &seasonsCatalogHandlerAdapter{inner: r.seasonsCatalog}
}

type seasonsCatalogHandlerAdapter struct {
	inner *service.SeasonsCatalog
}

func (a *seasonsCatalogHandlerAdapter) Load(ctx context.Context, titleID string) []handlers.SeasonCatalogEntry {
	src := a.inner.Load(ctx, titleID)
	out := make([]handlers.SeasonCatalogEntry, 0, len(src))
	for _, e := range src {
		out = append(out, handlers.SeasonCatalogEntry{
			ID:           e.ID,
			Label:        e.Label,
			Start:        e.Start,
			End:          e.End,
			DisplayOrder: e.DisplayOrder,
			Extra:        e.Extra,
		})
	}
	return out
}

// semanticFor retourne le SemanticAdapter du titre slug, ou nil si le resolver
// n'est pas câblé ou si le titre est inconnu (les consommateurs dégradent
// gracieusement vers les fallbacks de libellés).
func (r *ServiceRegistry) semanticFor(slug string) games.TitleSemanticAdapter {
	if r.titleResolver == nil {
		return nil
	}
	sem, err := r.titleResolver.Semantic(slug)
	if err != nil {
		return nil
	}
	return sem
}

// assetURLFor retourne le TitleAssetURLAdapter d'un titre ou nil si non
// résolu. Permet aux services (HomeService, MatchViewService, ...) de
// produire des URLs d'images sans coupler leur factory au resolver.
func (r *ServiceRegistry) assetURLFor(slug string) games.TitleAssetURLAdapter {
	if r.titleResolver == nil {
		return nil
	}
	a, err := r.titleResolver.AssetURL(slug)
	if err != nil {
		return nil
	}
	return a
}

// dataAdapterForPDB retourne un TitleDataAdapter player-scoped HI ou nil si
// le titre courant ne supporte pas la couche multi-titres player-scoped.
// Utilisé par les factories de services pour câbler la bascule Phase C+
// sans la dupliquer à chaque endroit.
func (r *ServiceRegistry) dataAdapterForPDB(pdb *duckdb.PlayerDB) games.TitleDataAdapter {
	if pdb == nil {
		return nil
	}
	build, ok := r.playerDataBuilders[pdb.TitleSlug]
	if !ok {
		return nil // titre sans builder player-scoped enregistré → dégradation
	}
	return build(pdb)
}

// GetSessionNotifier retourne le SessionNotifier enregistré pour le xuid donné.
// Retourne nil si aucun HomeService n'a encore été créé pour ce joueur (cold start).
func (r *ServiceRegistry) GetSessionNotifier(xuid string) port.SessionNotifier {
	if v, ok := r.notifiers.Load(xuid); ok {
		return v.(port.SessionNotifier)
	}
	return nil
}
