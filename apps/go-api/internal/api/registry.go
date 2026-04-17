// Package api — registry.go : câblage des services par injection de dépendances.
//
// Sprint 37 : ServiceRegistry centralise la construction des services
// à partir du PlayerDB résolu. Les handlers reçoivent des factory
// functions typées plutôt que cfg — testabilité et découplage.
package api

import (
	"context"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

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
	port.MediaService, string, string, string, error,
) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", "", err
	}
	svc := service.NewMediaService(duckdb.NewMediaRepo(pdb))
	return svc, pdb.Gamertag, pdb.TitleSlug, pdb.Player.Path(), nil
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

// TeammatesCtx retourne un TeammatesService + identifiants joueur.
func (r *ServiceRegistry) TeammatesCtx(ctx context.Context, slug string) (port.TeammatesService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewTeammatesService(duckdb.NewSquadRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}
