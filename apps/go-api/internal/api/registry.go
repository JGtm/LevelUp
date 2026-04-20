// Package api — registry.go : câblage des services par injection de dépendances.
//
// Sprint 37 : ServiceRegistry centralise la construction des services
// à partir du PlayerDB résolu. Les handlers reçoivent des factory
// functions typées plutôt que cfg — testabilité et découplage.
package api

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
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
	resolve PlayerResolver
}

// NewServiceRegistry crée un ServiceRegistry câblé avec config.ResolvePlayer.
// Le titleSlug est lu depuis le contexte (ctxkeys.TitleSlug), fallback "halo_infinite".
func NewServiceRegistry(cfg *config.AppConfig) *ServiceRegistry {
	return &ServiceRegistry{
		resolve: func(ctx context.Context, slug string) (*duckdb.PlayerDB, error) {
			titleSlug := ctxkeys.TitleSlug(ctx)
			return config.ResolvePlayer(ctx, cfg, slug, titleSlug)
		},
	}
}

// ---------------------------------------------------------------------------
// Factory methods — retournent des interfaces port.*Service
// ---------------------------------------------------------------------------

// Career retourne un CareerService pour le joueur identifié par slug.
func (r *ServiceRegistry) Career(ctx context.Context, slug string) (port.CareerService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewCareerService(duckdb.NewCareerRepo(pdb)), nil
}

// Filters retourne un FiltersService pour le joueur.
func (r *ServiceRegistry) Filters(ctx context.Context, slug string) (port.FiltersService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewFiltersService(duckdb.NewFiltersRepo(pdb)), nil
}

// LastMatch retourne un LastMatchService pour le joueur.
func (r *ServiceRegistry) LastMatch(ctx context.Context, slug string) (port.LastMatchService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewLastMatchService(duckdb.NewStatsRepo(pdb)), nil
}

// MatchView retourne un MatchViewService pour le joueur.
func (r *ServiceRegistry) MatchView(ctx context.Context, slug string) (port.MatchViewService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewMatchViewService(duckdb.NewMatchViewRepo(pdb, pdb.XUID), pdb.XUID), nil
}

// Media retourne un MediaService pour le joueur.
func (r *ServiceRegistry) Media(ctx context.Context, slug string) (port.MediaService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewMediaService(duckdb.NewMediaRepo(pdb)), nil
}

// MediaUpload retourne un MediaService + métadonnées joueur pour l'upload.
// Signature conforme à handlers.MediaUploadContextFactory.
func (r *ServiceRegistry) MediaUpload(ctx context.Context, slug string) (
	port.MediaService, string, string, string, string, error,
) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", "", "", err
	}
	svc := service.NewMediaService(duckdb.NewMediaRepo(pdb))
	sharedSocialPath := ""
	if pdb.SharedSocial != nil {
		sharedSocialPath = pdb.SharedSocial.Path()
	}
	return svc, pdb.Gamertag, pdb.TitleSlug, pdb.Player.Path(), sharedSocialPath, nil
}

// Social retourne un SocialService pour le joueur.
func (r *ServiceRegistry) Social(ctx context.Context, slug string) (port.SocialService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewSocialService(duckdb.NewSocialRepo(pdb)), nil
}

// Sessions retourne un SessionsService pour le joueur.
func (r *ServiceRegistry) Sessions(ctx context.Context, slug string) (port.SessionsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewSessionsService(duckdb.NewSessionsRepo(pdb)), nil
}

// SessionCompare retourne un SessionCompareService pour le joueur.
func (r *ServiceRegistry) SessionCompare(ctx context.Context, slug string) (port.SessionCompareService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	sessionsRepo := duckdb.NewSessionsRepo(pdb)
	statsRepo := duckdb.NewStatsRepo(pdb)
	return service.NewSessionCompareService(sessionsRepo, statsRepo), nil
}

// Stats retourne un StatsService pour le joueur.
func (r *ServiceRegistry) Stats(ctx context.Context, slug string) (port.StatsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewStatsService(duckdb.NewStatsRepo(pdb)), nil
}

// Timeseries retourne un TimeseriesService pour le joueur.
func (r *ServiceRegistry) Timeseries(ctx context.Context, slug string) (port.TimeseriesService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewTimeseriesService(duckdb.NewStatsRepo(pdb)), nil
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
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// HomeCtx retourne un HomeService + identifiants joueur.
func (r *ServiceRegistry) HomeCtx(ctx context.Context, slug string) (port.HomeService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewHomeService(duckdb.NewHomeRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// HomeCtxWithAuth retourne un HomeService + contexte enrichi avec les HaloTokens du joueur.
// Si la session HTTP porte déjà des tokens, ils sont réutilisés.
// Sinon, tente un refresh silencieux depuis le cache MSAL stocké dans sync_meta.
func (r *ServiceRegistry) HomeCtxWithAuth(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	svc := service.NewHomeService(duckdb.NewHomeRepo(pdb))
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// enrichWithHaloTokens injecte les HaloTokens dans le contexte si absents.
// Ordre : 1) session HTTP (déjà injectée par le middleware), 2) cache process, 3) sync_meta DB.
func (r *ServiceRegistry) enrichWithHaloTokens(ctx context.Context, pdb *duckdb.PlayerDB) context.Context {
	if ctxkeys.HaloTokens(ctx) != nil {
		return ctx // tokens déjà présents via session HTTP
	}
	xuid := pdb.XUID
	if cached := halo.GetCachedPlayerTokens(xuid); cached != nil {
		return ctxkeys.WithHaloAuth(ctx, cached, xuid)
	}
	result := refreshTokensFromDB(ctx, pdb, xuid)
	if result != nil {
		halo.SetCachedPlayerTokens(xuid, result.Tokens)
		return ctxkeys.WithHaloAuth(ctx, result.Tokens, xuid)
	}
	return ctx
}

// refreshTokensFromDB charge le cache MSAL depuis sync_meta et tente un refresh silencieux.
func refreshTokensFromDB(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	cacheJSON, err := duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	if err != nil || cacheJSON == "" {
		return nil
	}
	accessor := auth.NewInMemoryCacheAccessorFromJSON(cacheJSON)
	accessToken, err := auth.AcquireTokenSilent(ctx, accessor)
	if err != nil || accessToken == "" {
		slog.WarnContext(ctx, "halo_auth: refresh silencieux échoué", "xuid", xuid, "err", err)
		return nil
	}
	result, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		slog.WarnContext(ctx, "halo_auth: échange access_token échoué", "xuid", xuid, "err", err)
		return nil
	}
	return result
}

// MatchHistoryCtx retourne un MatchHistoryService + identifiants joueur.
func (r *ServiceRegistry) MatchHistoryCtx(ctx context.Context, slug string) (port.MatchHistoryService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewMatchHistoryService(duckdb.NewMatchHistoryRepo(pdb), pdb.Gamertag)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadCtx retourne un SquadService + identifiants joueur.
func (r *ServiceRegistry) SquadCtx(ctx context.Context, slug string) (port.SquadService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSquadService(duckdb.NewSquadRepo(pdb))
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
func (r *ServiceRegistry) TeammatesCtx(ctx context.Context, slug string) (port.TeammatesService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewTeammatesService(duckdb.NewSquadRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ─── Sprint 54 : Compare + Leaderboard ───────────────────────────────────────

// Compare retourne un CompareService pour le joueur (slug = joueur A).
// Le PlayerStatsProvider (Waypoint) est injecté via DefaultHaloProvider.
func (r *ServiceRegistry) Compare(ctx context.Context, slug string) (port.CompareService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewCompareService(
		duckdb.NewCompareRepo(pdb),
		haloProvider,
		pdb.XUID,
		pdb.TitleSlug,
	)
	return svc, pdb.XUID, pdb.Gamertag, nil
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
	svc := service.NewSynthesisService(duckdb.NewSynthesisRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}
