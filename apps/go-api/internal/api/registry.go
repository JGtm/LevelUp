// Package api — registry.go : câblage des services par injection de dépendances.
//
// Sprint 37 : ServiceRegistry centralise la construction des services
// à partir du PlayerDB résolu. Les handlers reçoivent des factory
// functions typées plutôt que cfg — testabilité et découplage.
package api

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"fmt"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// haloProvider est l'instance globale du provider Halo (partagée).
var haloProvider = halo.DefaultHaloProvider

// PlayerResolver traduit un slug joueur en PlayerDB (pool-cached).
type PlayerResolver func(ctx context.Context, slug string) (*duckdb.PlayerDB, error)

// ServiceRegistry centralise la construction des services métier.
// Chaque méthode résout le joueur puis construit le service injecté.
type ServiceRegistry struct {
	cfg              *config.AppConfig
	resolve          PlayerResolver
	resolveByGT      duckdb.TitlePlayerResolver // résolution (titleSlug, gamertag) → PlayerDB pour Squad V2
	provider         auth.TokenProvider
	assetResolver    assets.Resolver           // nil si le resolver n'est pas configuré (mode legacy)
	timezone         string                    // IANA (ex: "Europe/Paris"), propagé aux services médias
	notifiers        sync.Map                  // xuid → port.SessionNotifier (HomeService par joueur)
	titleResolver    games.Resolver            // nil → services tournent sans semantic adapter (libellés via fallbacks)
	homeMatchesCache *service.HomeMatchesCache // cache TTL process-level matches+sessions
	settingsStore    *settings_platform.Store  // nil → services qui dépendent des settings (TeammatesService friend filter) tournent en mode legacy
	seasonsCatalog   *service.SeasonsCatalog   // nil → FiltersService.Resolve ne renvoie pas SeasonCounts (dégradation gracieuse)
	rankCatalog      *mappings.RankCatalog     // nil → CareerService.next_rank_name reste vide
	rankImageURLs    map[int]*string           // nil → CareerService.rank_image_url et next_rank_image_url restent absents
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

// WithRankImageURLs attache la map rank_id → imageURL chargée au démarrage
// depuis career_ranks (metadata.duckdb). Consommée par CareerService pour
// rank_image_url et next_rank_image_url. Si nil, ces champs sont absents.
func (r *ServiceRegistry) WithRankImageURLs(imgs map[int]*string) *ServiceRegistry {
	r.rankImageURLs = imgs
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
	if pdb == nil || pdb.TitleSlug != title.DefaultSlug {
		return nil
	}
	return halo_games.NewDataAdapter(duckdb.NewCareerRepo(pdb), slog.Default())
}

// GetSessionNotifier retourne le SessionNotifier enregistré pour le xuid donné.
// Retourne nil si aucun HomeService n'a encore été créé pour ce joueur (cold start).
func (r *ServiceRegistry) GetSessionNotifier(xuid string) port.SessionNotifier {
	if v, ok := r.notifiers.Load(xuid); ok {
		return v.(port.SessionNotifier)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Factory methods — retournent des interfaces port.*Service
// ---------------------------------------------------------------------------

// Career retourne un CareerService pour le joueur identifié par slug.
//
// Phase C+ multi-titres : quand le titre courant est HI, un DataAdapter
// player-scoped est injecté dans le service. GetEncounters passe alors par
// games.TitleDataAdapter.LoadEncounters → projection canonique →
// domain.EncounterDTO, avec parité de payload par construction. Si la
// capability LoadEncounters retourne ErrCapabilityNotSupported, le service
// retombe automatiquement sur s.repo.GetEncounters.
func (r *ServiceRegistry) Career(ctx context.Context, slug string) (port.CareerService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	careerRepo := duckdb.NewCareerRepo(pdb)
	svc := service.NewCareerService(careerRepo).WithTitleSlug(pdb.TitleSlug)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	if loader := r.buildFriendsXPLoader(pdb); loader != nil {
		svc = svc.WithFriendsXPLoader(loader)
	}
	if r.rankCatalog != nil {
		svc = svc.WithRankCatalog(r.rankCatalog)
	}
	if r.rankImageURLs != nil {
		svc = svc.WithRankImageURLs(r.rankImageURLs)
	}
	// Wiring des amis — utilisé par GetTopEncounters pour le tableau "joueurs
	// les plus croisés (hors amis)". Si le settingsStore n'est pas attaché ou
	// que la liste est vide, GetTopEncounters n'exclut personne (dégradation
	// gracieuse).
	if resolver := r.friendGamertagsResolver(); resolver != nil {
		svc = svc.WithFriendGamertagsResolver(resolver)
	}
	// Résolveur gamertag → xuid : ExplorerRepo.ResolveXUIDByGamertag interroge
	// shared.v_gamertag_lookup (cascade xuid_aliases ∪ match_participants),
	// donc capture les amis qui ne sont pas encore dans xuid_aliases mais
	// déjà apparus en match. Source unique de vérité partagée avec Explorer.
	explorerRepo := duckdb.NewExplorerRepo(pdb, pdb.XUID)
	svc = svc.WithFriendXUIDResolver(explorerRepo.ResolveXUIDByGamertag)
	// SeasonsCatalog (TOML + DB + lazy-fetch) — alimente le filtre Saisons
	// + cascade counts dans la section "Matchs marquants". Mêmes seasons
	// que la SaisonPill côté Squad/Explorer.
	if r.seasonsCatalog != nil {
		svc = svc.WithSeasonsCatalog(r.seasonsCatalog)
	}
	if r.cfg != nil && r.cfg.CurrentCSRSeasonID != "" {
		svc = svc.WithCSRSeasonID(r.cfg.CurrentCSRSeasonID)
	}
	return svc, nil
}

// buildFriendsXPLoader construit un loader d'historique XP pour tous les amis
// du joueur courant (joueurs référencés dans db_profiles.json, hors joueur
// courant). Retourne nil si cfg est indisponible ou s'il n'y a aucun autre joueur.
func (r *ServiceRegistry) buildFriendsXPLoader(mainPDB *duckdb.PlayerDB) service.FriendsXPLoader {
	if r.cfg == nil {
		return nil
	}
	players, err := r.cfg.LoadPlayers(mainPDB.TitleSlug)
	if err != nil {
		slog.Warn("friends_xp: load players failed (loader disabled)",
			"titleSlug", mainPDB.TitleSlug, "err", err)
		return nil
	}
	// Filtrer pour ne garder que les amis (≠ joueur courant, avec XUID renseigné).
	type friendEntry struct{ gamertag, playerSlug string }
	var friends []friendEntry
	for _, p := range players {
		if p.XUID == "" || p.XUID == mainPDB.XUID {
			continue
		}
		friends = append(friends, friendEntry{gamertag: p.Gamertag, playerSlug: p.PlayerSlug})
	}
	if len(friends) == 0 {
		return nil
	}
	cfg := r.cfg
	titleSlug := mainPDB.TitleSlug
	return func(ctx context.Context, _ string) ([]domain.FriendXPHistory, error) {
		var results []domain.FriendXPHistory
		for _, f := range friends {
			pdb, perr := config.ResolvePlayer(ctx, cfg, f.playerSlug, titleSlug)
			if perr != nil {
				slog.WarnContext(ctx, "friends_xp: resolve failed",
					"gamertag", f.gamertag, "err", perr)
				continue
			}
			history, herr := duckdb.NewCareerRepo(pdb).GetXPHistory(ctx)
			if herr != nil {
				slog.WarnContext(ctx, "friends_xp: get history failed",
					"gamertag", f.gamertag, "err", herr)
				continue
			}
			if len(history) == 0 {
				continue
			}
			results = append(results, domain.FriendXPHistory{
				Gamertag: f.gamertag,
				History:  history,
			})
		}
		return results, nil
	}
}

// Achievements retourne un AchievementsService pour le joueur identifié par slug.
//
// Le service merge deux sources :
//   - AchievementsRepo (player_achievements dans stats.duckdb du joueur)
//   - MetadataRepo.GetAchievementDefinitions (xbox_achievement_definitions dans
//     metadata.duckdb partagée)
//
// Aucun DataAdapter requis : les achievements ne sont pas des données canoniques
// de match, donc l'accès direct via repos suit le même pattern que Career.
func (r *ServiceRegistry) Achievements(ctx context.Context, slug string) (port.AchievementsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewAchievementsRepo(pdb)
	metaRepo := duckdb.NewMetadataRepo(pdb)
	return service.NewAchievementsService(repo, metaRepo).WithTitleSlug(pdb.TitleSlug), nil
}

// TitleDataAdapter retourne un games.TitleDataAdapter player-scoped pour le
// joueur courant.
//
// Phase C+ du plan multi-titres : cette méthode est le point d'injection
// utilisé par les handlers /api/v1/players/{slug}/... pour consommer la
// couche canonique. Le PlayerDB est résolu avec son CareerRepo, ce qui
// active la capability career.progression pour ce DataAdapter.
//
// Retourne ErrTitleNotResolved si le slug courant n'est pas halo_infinite
// (les autres titres viendront avec leurs propres factories).
func (r *ServiceRegistry) TitleDataAdapter(ctx context.Context, slug string) (games.TitleDataAdapter, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	if pdb.TitleSlug != title.DefaultSlug {
		return nil, fmt.Errorf("%w: %q (seul halo_infinite a un DataAdapter player-scoped pour le moment)",
			games.ErrTitleNotResolved, pdb.TitleSlug)
	}
	careerRepo := duckdb.NewCareerRepo(pdb)
	return halo_games.NewDataAdapter(careerRepo, slog.Default()), nil
}

// Filters retourne un FiltersService pour le joueur.
//
// Injecte le catalog unifié des saisons (TOML + DB live + lazy fetch) pour
// alimenter les SeasonCounts du folding SaisonPill. Si le catalog n'est
// pas câblé OU si le titre n'a aucune saison résolue → aucun SeasonCount
// renvoyé (dégradation gracieuse, le frontend affiche les saisons sans
// folding).
func (r *ServiceRegistry) Filters(ctx context.Context, slug string) (port.FiltersService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewFiltersService(duckdb.NewFiltersRepo(pdb))
	if r.seasonsCatalog != nil {
		// Bug fix 2026-05-08 : passer pdb.TitleSlug (le titre du joueur, ex
		// "halo_infinite") et NON le `slug` paramètre qui est le **player
		// slug** URL (ex "JGtm"). Avant, seasons_catalog recevait "JGtm"
		// comme titleID → fetch live échouait toujours sur fallback TOML
		// (logs : `seasons_catalog: fetch live échec titleSlug=JGtm`).
		svc = svc.WithSeasonsCatalog(pdb.TitleSlug, r.seasonsCatalog)
	}
	return svc, nil
}

// MatchView retourne un MatchViewService pour le joueur.
//
// Phase C+ multi-titres : injecte le DataAdapter HI pour permettre une
// future bascule LoadMatchDetail (le service utilise le hook WithDataAdapter
// pour préparer la migration sans la déclencher tant que canonical.MatchDetail
// ne couvre pas la totalité du payload Match View).
func (r *ServiceRegistry) MatchView(ctx context.Context, slug string) (port.MatchViewService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewMatchViewService(duckdb.NewMatchViewRepo(pdb, pdb.XUID), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	svc = svc.WithCitationsRepo(duckdb.NewCitationsRepo(pdb)).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithAssetURL(r.assetURLFor(pdb.TitleSlug)).
		WithTitleSlug(pdb.TitleSlug).
		WithMetadataRepo(duckdb.NewMetadataRepo(pdb))
	if loader := r.buildFriendsExtrasResolver(pdb); loader != nil {
		svc = svc.WithFriendsExtras(loader)
	}
	return svc, nil
}

// buildFriendsExtrasResolver construit un loader d'extras per-friend pour le
// panneau d'expander scoreboard (Match View). Lookup xuid → (titleSlug,
// gamertag) depuis cfg.LoadPlayers, puis ouverture lazy de la player DB de
// l'ami via le pool DuckDB (cached). Retourne nil si cfg indisponible (tests).
func (r *ServiceRegistry) buildFriendsExtrasResolver(mainPDB *duckdb.PlayerDB) port.FriendsExtrasResolver {
	if r.cfg == nil {
		return nil
	}
	players, err := r.cfg.LoadPlayers(mainPDB.TitleSlug)
	if err != nil {
		slog.Warn("friends_extras: load players failed (resolver disabled)",
			"titleSlug", mainPDB.TitleSlug, "err", err)
		return nil
	}
	if len(players) == 0 {
		return nil
	}
	friendsByXUID := make(map[string]service.FriendProfile, len(players))
	for _, p := range players {
		if p.XUID == "" || p.XUID == mainPDB.XUID {
			continue
		}
		friendsByXUID[p.XUID] = service.FriendProfile{
			XUID:      p.XUID,
			Gamertag:  p.Gamertag,
			TitleSlug: mainPDB.TitleSlug,
		}
	}
	if len(friendsByXUID) == 0 {
		return nil
	}
	cfg := r.cfg
	opener := func(ctx context.Context, titleSlug, gamertag string) (service.FriendMatchExtrasRepo, error) {
		// Trouve le slug du joueur depuis son gamertag (cfg.LoadPlayers).
		ps, lerr := cfg.LoadPlayers(titleSlug)
		if lerr != nil {
			return nil, lerr
		}
		var slug string
		for i := range ps {
			if ps[i].Gamertag == gamertag {
				slug = ps[i].PlayerSlug
				break
			}
		}
		if slug == "" {
			return nil, fmt.Errorf("friends_extras: player not found gamertag=%q title=%q", gamertag, titleSlug)
		}
		pdb, perr := config.ResolvePlayer(ctx, cfg, slug, titleSlug)
		if perr != nil {
			return nil, perr
		}
		return duckdb.NewMatchViewRepo(pdb, pdb.XUID), nil
	}
	return service.NewFriendsExtrasResolver(friendsByXUID, opener)
}

// Engagement retourne un PlayerEngagementService pour le joueur.
// (Phase 4 plan engagement — endpoint /matches/{id}/engagement et /engagement_profile)
func (r *ServiceRegistry) Engagement(ctx context.Context, slug string) (*service.PlayerEngagementService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewEngagementScoreRepo(pdb)
	return service.NewPlayerEngagementService(repo, pdb.XUID, pdb.Gamertag), nil
}

// mediaWriterAcquirerFor construit l'acquéreur shared_social pour un PlayerDB.
// Factorise la création de l'option pour les deux factories (Media, MediaUpload).
// Cf. commit 6 du refactor leased-writer-enforcement (atomicité likes).
func mediaWriterAcquirerFor(pdb *duckdb.PlayerDB) func() (*dblease.LeasedWriter, error) {
	return func() (*dblease.LeasedWriter, error) {
		return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
	}
}

// Media retourne un MediaService pour le joueur.
//
// Configure WithMediaWriterAcquirer pour activer le chemin atomique de
// SetMediaLike (transaction unique sur shared_social.duckdb).
func (r *ServiceRegistry) Media(ctx context.Context, slug string) (port.MediaService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewMediaService(duckdb.NewMediaRepo(pdb), r.timezone,
		service.WithMediaWriterAcquirer(mediaWriterAcquirerFor(pdb))), nil
}

// MediaUpload retourne un MediaService + métadonnées joueur pour l'upload.
// Signature conforme à handlers.MediaUploadContextFactory.
func (r *ServiceRegistry) MediaUpload(ctx context.Context, slug string) (
	port.MediaService, string, string, string, string, string, error,
) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", "", "", "", err
	}
	svc := service.NewMediaService(duckdb.NewMediaRepo(pdb), r.timezone,
		service.WithMediaWriterAcquirer(mediaWriterAcquirerFor(pdb)))
	sharedSocialPath := ""
	if pdb.SharedSocial != nil {
		sharedSocialPath = pdb.SharedSocial.Path()
	}
	sharedMatchesPath := ""
	if pdb.Shared != nil {
		sharedMatchesPath = pdb.Shared.Path()
	}
	return svc, pdb.Gamertag, pdb.TitleSlug, pdb.Player.Path(), sharedSocialPath, sharedMatchesPath, nil
}

// MediaPlayerCtx résout slug → (titleSlug, gamertag) sans construire de service.
// Signature conforme à handlers.MediaPlayerContextFactory.
func (r *ServiceRegistry) MediaPlayerCtx(ctx context.Context, slug string) (string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return "", "", err
	}
	return pdb.TitleSlug, pdb.Gamertag, nil
}

// Social retourne un SocialService pour le joueur.
//
// Configure un WriterAcquirer sur shared_social.duckdb pour sérialiser
// ToggleMatchFavorite avec les autres écritures (sync engine, autres handlers).
// Cf. commit 5 du refactor leased-writer-enforcement.
func (r *ServiceRegistry) Social(ctx context.Context, slug string) (port.SocialService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	acquirer := func() (*dblease.LeasedWriter, error) {
		return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
	}
	return service.NewSocialService(duckdb.NewSocialRepo(pdb), service.WithWriterAcquirer(acquirer)), nil
}

// Sessions retourne un SessionsService pour le joueur.
func (r *ServiceRegistry) Sessions(ctx context.Context, slug string) (port.SessionsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewSessionsService(duckdb.NewSessionsRepo(pdb)), nil
}

// playerMatchesAdapterFor construit un adapter PlayerMatchesRepository pour le
// joueur (pdb). P4.3 finale : permet aux services match-rows de consommer
// canonical exclusivement (legacy fallback path supprimé).
func (r *ServiceRegistry) playerMatchesAdapterFor(pdb *duckdb.PlayerDB) port.PlayerMatchesRepository {
	pmRepo := duckdb.NewPlayerMatchesRepo(pdb)
	return duckdb.NewPlayerMatchesAdapter(pmRepo, pdb.TitleSlug, pdb.Gamertag)
}

// SessionCompare retourne un SessionCompareService pour le joueur.
func (r *ServiceRegistry) SessionCompare(ctx context.Context, slug string) (port.SessionCompareService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	sessionsRepo := duckdb.NewSessionsRepo(pdb)
	statsRepo := duckdb.NewStatsRepo(pdb)
	svc := service.NewSessionCompareService(sessionsRepo, statsRepo).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, nil
}

// SessionPage retourne un SessionPageService pour le joueur.
func (r *ServiceRegistry) SessionPage(ctx context.Context, slug string) (port.SessionPageService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewSessionPageService(duckdb.NewStatsRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, nil
}

// Stats retourne un StatsService pour le joueur.
func (r *ServiceRegistry) Stats(ctx context.Context, slug string) (port.StatsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewStatsService(duckdb.NewStatsRepo(pdb)).
		WithTitleSlug(pdb.TitleSlug).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.Gamertag), nil
}

// Timeseries retourne un TimeseriesService pour le joueur.
//
// Phase C+ multi-titres : injecte le DataAdapter HI pour permettre une
// future bascule LoadTimeseries.
func (r *ServiceRegistry) Timeseries(ctx context.Context, slug string) (port.TimeseriesService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewTimeseriesService(duckdb.NewStatsRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb)).
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, nil
}

// ---------------------------------------------------------------------------
// Factories avec contexte joueur (service + XUID + Gamertag)
// Pour les handlers qui ont besoin des identifiants joueur dans les appels.
// Signature : func(ctx, slug) → (service, xuid, gamertag, error)
// ---------------------------------------------------------------------------

// CitationsCtx retourne un CitationsService + identifiants joueur.
func (r *ServiceRegistry) CitationsCtx(ctx context.Context, slug string) (port.CitationsService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewCitationsService(duckdb.NewCitationsRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ExplorerCtx retourne un ExplorerService + identifiants joueur.
func (r *ServiceRegistry) ExplorerCtx(ctx context.Context, slug string) (port.ExplorerService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewExplorerService(duckdb.NewExplorerRepo(pdb, pdb.XUID), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// HomeCtx retourne un HomeService + identifiants joueur.
func (r *ServiceRegistry) HomeCtx(ctx context.Context, slug string) (port.HomeService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewHomeService(r.newHomeRepo(pdb)).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// newHomeRepo construit un HomeRepo avec l'AssetURLAdapter du titre câblé
// quand disponible (fallback static FS pour les images map non encore
// peuplées dans map_images_registry — évite la dépendance à
// cmd/migrate-static-maps à chaque ajout de fichier static). Dégradation
// gracieuse : sans titleResolver ou sans adapter, le HomeRepo reste nu et
// ne fait que le lookup registry.
func (r *ServiceRegistry) newHomeRepo(pdb *duckdb.PlayerDB) *duckdb.HomeRepo {
	repo := duckdb.NewHomeRepo(pdb)
	if r.titleResolver == nil {
		return repo
	}
	adapter, err := r.titleResolver.AssetURL(pdb.TitleSlug)
	if err != nil || adapter == nil {
		return repo
	}
	return repo.WithAssetURL(adapter)
}

// HomeCtxWithAuth retourne un HomeService + contexte enrichi avec les HaloTokens du joueur.
// Si la session HTTP porte déjà des tokens, ils sont réutilisés.
// Sinon, tente un refresh silencieux depuis le cache MSAL stocké dans sync_meta.
// Un PersistSink est configuré pour la persistance fire-and-forget des données BP/challenges.
// Le HomeService créé est enregistré comme SessionNotifier pour ce joueur (TTL dynamique).
func (r *ServiceRegistry) HomeCtxWithAuth(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)
	homeRepo := r.newHomeRepo(pdb)
	haloProvider := r.buildHaloProvider(pdb).WithTrackDefPersister(sink).WithItemDefPersister(sink)
	svc := service.NewHomeService(homeRepo).
		WithPersistSink(sink).
		WithCacheRepo(homeRepo).
		WithHaloProvider(haloProvider).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(svc))
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// SeasonPassCtxWithAuth retourne un SeasonPassService + contexte enrichi avec les HaloTokens.
// Réutilise HomeCtxWithAuth pour la résolution des tokens et le cacheRepo BP/challenges.
// Le HomeService créé est enregistré comme SessionNotifier pour ce joueur (TTL dynamique).
func (r *ServiceRegistry) SeasonPassCtxWithAuth(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, err
	}
	homeRepo := r.newHomeRepo(pdb)
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)
	haloProvider := r.buildHaloProvider(pdb).WithTrackDefPersister(sink).WithItemDefPersister(sink)
	homeSvc := service.NewHomeService(homeRepo).
		WithPersistSink(sink).
		WithCacheRepo(homeRepo).
		WithHaloProvider(haloProvider).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(homeSvc))
	spRepo := duckdb.NewSeasonPassRepo(pdb)
	svc := service.NewSeasonPassService(spRepo, homeSvc, pdb.XUID, pdb.TitleSlug)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, nil
}

// buildHaloProvider construit un HaloProvider configuré avec le resolver unifié et le titre du joueur.
func (r *ServiceRegistry) buildHaloProvider(pdb *duckdb.PlayerDB) *halo.HaloProvider {
	titleSlug := title.DefaultSlug
	if pdb != nil && pdb.TitleSlug != "" {
		titleSlug = pdb.TitleSlug
	}
	return halo.DefaultHaloProvider.
		WithAssetResolver(r.assetResolver).
		WithTitleSlug(titleSlug)
}

// enrichWithHaloTokens injecte les HaloTokens dans le contexte si absents.
func (r *ServiceRegistry) enrichWithHaloTokens(ctx context.Context, pdb *duckdb.PlayerDB) context.Context {
	if ctxkeys.HaloTokens(ctx) != nil {
		return ctx // tokens déjà présents via session HTTP
	}
	xuid := pdb.XUID
	if cached := halo.GetCachedPlayerTokens(xuid); cached != nil {
		return ctxkeys.WithHaloAuth(ctx, cached, xuid)
	}
	result := r.refreshTokensFromDB(ctx, pdb, xuid)
	if result != nil {
		halo.SetCachedPlayerTokens(xuid, result.Tokens)
		return ctxkeys.WithHaloAuth(ctx, result.Tokens, xuid)
	}
	return ctx
}

// refreshTokensFromDB charge le cache MSAL ou le refresh_token OAuth v2 depuis sync_meta,
// puis tente un refresh silencieux pour obtenir les tokens Halo.
// Ordre :
//  1. MSAL cache (sync_meta.msal_token_cache) → TrySilentRefresh
//  2. OAuth v2 refresh_token (env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>) → TryOAuthRefresh
//  3. OAuth v2 refresh_token (sync_meta.oauth_refresh_token) → TryOAuthRefresh
func (r *ServiceRegistry) refreshTokensFromDB(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	// --- Chemin 1 : MSAL cache ---
	cacheJSON, err := duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	if err == nil && cacheJSON != "" {
		accessToken, err := r.provider.TrySilentRefresh(ctx, cacheJSON)
		if err == nil && accessToken != "" {
			result, err := r.provider.Exchange(ctx, accessToken)
			if err == nil && result != nil {
				slog.DebugContext(ctx, "halo_auth: tokens obtenus via MSAL cache", "xuid", xuid)
				return result
			}
			slog.WarnContext(ctx, "halo_auth: échange access_token échoué (MSAL)", "xuid", xuid, "err", err)
		} else if err != nil {
			slog.WarnContext(ctx, "halo_auth: MSAL silent refresh échoué", "xuid", xuid, "err", err)
		}
	}

	// --- Chemin 2 : refresh_token OAuth v2 ---
	// Priorité : variable d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_UPPER>
	// puis clé oauth_refresh_token dans sync_meta.
	refreshToken := oauthRefreshTokenForPlayer(pdb.Gamertag)
	if refreshToken == "" {
		refreshToken, _ = duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)
	}
	if refreshToken != "" {
		accessToken, err := r.provider.TryOAuthRefresh(ctx, refreshToken)
		if err == nil && accessToken != "" {
			result, err := r.provider.Exchange(ctx, accessToken)
			if err == nil && result != nil {
				slog.DebugContext(ctx, "halo_auth: tokens obtenus via OAuth v2 refresh", "xuid", xuid)
				return result
			}
			slog.WarnContext(ctx, "halo_auth: échange access_token échoué (OAuth v2)", "xuid", xuid, "err", err)
		} else if err != nil {
			slog.WarnContext(ctx, "halo_auth: OAuth v2 refresh échoué", "xuid", xuid, "err", err)
		}
	}

	slog.WarnContext(ctx, "halo_auth: aucun token disponible pour le joueur", "xuid", xuid, "gamertag", pdb.Gamertag)
	return nil
}

// MatchHistoryCtx retourne un MatchHistoryService + identifiants joueur.
func (r *ServiceRegistry) MatchHistoryCtx(ctx context.Context, slug string) (port.MatchHistoryService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewMatchHistoryService(duckdb.NewMatchHistoryRepo(pdb), pdb.Gamertag).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadCtx retourne un SquadService + identifiants joueur.
func (r *ServiceRegistry) SquadCtx(ctx context.Context, slug string) (port.SquadService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSquadService(duckdb.NewSquadRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadV2Ctx retourne un SquadV2Service + identifiants joueur. Le service
// utilise un loader production qui résout (titleSlug, gamertag) → PlayerDB
// via le pool global, ce qui permet de charger les matchs des coéquipiers
// sans avoir leur player_slug — ils sont sélectionnés par gamertag dans l'UI.
func (r *ServiceRegistry) SquadV2Ctx(ctx context.Context, slug string) (port.SquadV2Service, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	loader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	// Le loader resout les DBs `shared` (events / weapons / medals) via le main
	// player ; on lui propage le gamertag de la session courante (chunk S11).
	loader.SetDefaultGamertag(pdb.Gamertag)
	svc := service.NewSquadServiceV2(loader)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// MatchExclusion retourne un MatchExclusionService pour le joueur.
func (r *ServiceRegistry) MatchExclusion(ctx context.Context, slug string) (port.MatchExclusionService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewMatchExclusionService(duckdb.NewMatchExclusionRepo(pdb)), nil
}

// TeammatesCtx retourne un TeammatesService + identifiants joueur.
//
// Le resolver friend_gamertags est branché sur r.settingsStore quand le
// store est attaché (cf. WithSettingsStore). Sans store → comportement
// legacy : top dropdown brut sans filtre amis.
func (r *ServiceRegistry) TeammatesCtx(ctx context.Context, slug string) (port.TeammatesService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	// SessionBriefing mode squad : besoin d'un loader per-gamertag (le
	// playerMatchesAdapterFor est bound au main, ne sait pas charger les
	// canonical rows d'un coequipier different). On reutilise le SquadV2Loader.
	briefingLoader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	briefingLoader.SetDefaultGamertag(pdb.Gamertag)
	svc := service.NewTeammatesService(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithSquadLoader(briefingLoader).
		WithMedalDefs(duckdb.NewMedalDefinitionsRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// friendGamertagsResolver construit un resolver lisant app_settings.friend_gamertags
// à chaque appel. Retourne nil si aucun settings store n'est attaché — le
// service tourne alors en mode legacy.
func (r *ServiceRegistry) friendGamertagsResolver() service.FriendGamertagsResolver {
	if r.settingsStore == nil {
		return nil
	}
	return func(ctx context.Context) []string {
		s, err := r.settingsStore.Load()
		if err != nil {
			slog.WarnContext(ctx, "friend_gamertags_load_failed", "err", err)
			return nil
		}
		return s.FriendGamertags
	}
}

// ─── Sprint 54 : Compare + Leaderboard ───────────────────────────────────────

// Compare retourne un CompareService pour le joueur (slug = joueur A).
// Le PlayerStatsProvider (Waypoint) est injecté via DefaultHaloProvider.
// Le contexte est enrichi avec les HaloTokens pour que FetchRemoteStats puisse
// s'authentifier auprès de Waypoint même si la session HTTP ne porte pas de tokens.
func (r *ServiceRegistry) Compare(ctx context.Context, slug string) (port.CompareService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	svc := service.NewCompareService(
		duckdb.NewCompareRepo(pdb),
		haloProvider,
		pdb.XUID,
		pdb.TitleSlug,
	)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// Leaderboard retourne un LeaderboardService pour le joueur.
func (r *ServiceRegistry) Leaderboard(ctx context.Context, slug string) (port.LeaderboardService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewLeaderboardService(duckdb.NewLeaderboardRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ─── Sprint 55 : Synthesis (extrait de Squad) ────────────────────────────────

// SynthesisCtx retourne un SynthesisService + identifiants joueur.
// Sprint 55 D1 : séparé de SquadCtx pour refléter la frontière produit.
func (r *ServiceRegistry) SynthesisCtx(ctx context.Context, slug string) (port.SynthesisService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSynthesisService(duckdb.NewSynthesisRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithPersonalScoreAwardsRepo(duckdb.NewPersonalScoreAwardsRepo(pdb), pdb.XUID).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb))
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// =============================================================================
// Helpers auth
// =============================================================================

// oauthRefreshTokenForPlayer retourne le refresh_token OAuth v2 depuis l'environnement
// pour un gamertag donné.
// Convention : SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_MAJUSCULES_SANS_ESPACES>
// Exemple : gamertag "JGtm" → SPNKR_OAUTH_REFRESH_TOKEN_JGTM
func oauthRefreshTokenForPlayer(gamertag string) string {
	if gamertag == "" {
		return ""
	}
	// Normalisation : majuscules, espaces/tirets/points → underscore.
	key := strings.ToUpper(gamertag)
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}

// AnyPlayerTokens retourne les tokens Halo du premier joueur disponible dans le pool.
// Utilisé par les handlers d'assets qui ont besoin de tokens mais ne sont pas
// rattachés à un joueur spécifique.
func (r *ServiceRegistry) AnyPlayerTokens(ctx context.Context) (*domain.HaloTokens, error) {
	var tokens *domain.HaloTokens
	duckdb.IteratePool(func(pdb *duckdb.PlayerDB) bool {
		t, err := r.RefreshTokensForXUID(ctx, pdb.XUID)
		if err == nil && t != nil {
			tokens = t
			return false // stop
		}
		return true // continuer avec le joueur suivant
	})
	if tokens == nil {
		return nil, fmt.Errorf("aucun token Halo disponible")
	}
	return tokens, nil
}

// RefreshTokensForXUID tente un refresh silencieux pour le joueur identifié par son XUID.
// Recherche le PlayerDB dans le pool, puis tente MSAL ou OAuth v2 refresh.
// Met à jour le cache process si le refresh réussit.
// Appelé par PlayerLiveRefresher quand le cache process est expiré.
func (r *ServiceRegistry) RefreshTokensForXUID(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
	if cached := halo.GetCachedPlayerTokens(xuid); cached != nil {
		return cached, nil
	}
	var pdb *duckdb.PlayerDB
	duckdb.IteratePool(func(p *duckdb.PlayerDB) bool {
		if p.XUID == xuid {
			pdb = p
			return false // stop
		}
		return true
	})
	if pdb == nil {
		return nil, fmt.Errorf("halo_auth: joueur xuid=%s introuvable dans le pool", xuid)
	}
	result := r.refreshTokensFromDB(ctx, pdb, xuid)
	if result == nil {
		return nil, fmt.Errorf("halo_auth: refresh impossible pour xuid=%s", xuid)
	}
	halo.SetCachedPlayerTokens(xuid, result.Tokens)
	return result.Tokens, nil
}
