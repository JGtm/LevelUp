// Package api — registry_media.go : factories services Media + Social du
// ServiceRegistry. Découpé de registry.go (god-file split, refactor 2026-05-27).
package wire

import (
	"context"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// haloInfiniteModeTaxonomy est la taxonomie de classification des modes injectée dans
// MediaRepo (F1) et MatchViewRepo (F15-2, filtrage neighbors). Le couplage
// games/halo_infinite vit désormais au niveau composition (registry = racine DI,
// autorisée à importer games) et NON plus dans platform/duckdb. Actuellement Halo
// Infinite (seul titre avec ces surfaces en prod ; comportement byte-identique au code
// d'avant F1/F15-2). FOLLOW-UP per-titre : quand un 2e titre exposera média/neighbors
// filtrés, résoudre la taxonomie via le title resolver (comme assetURLFor).
func haloInfiniteModeTaxonomy() analysis.ModeTaxonomy {
	return analysis.ModeTaxonomy{
		InferCategory: halo_infinite.InferModeCategoryFromPairName,
		PrefixesFor:   halo_infinite.PairNamePrefixesForCategory,
		AllPrefixes:   halo_infinite.AllKnownPairNamePrefixes,
		Other:         halo_infinite.ModeCategoryOther,
	}
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
// SetMediaLike (transaction unique sur shared_social.duckdb), et WithAssetURL
// pour le fallback name-based d'image map dans le picker de réassociation
// (aligné sur HomeRepo / MatchViewService).
func (r *ServiceRegistry) Media(ctx context.Context, slug string) (port.MediaService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewMediaRepo(pdb).WithModeTaxonomy(haloInfiniteModeTaxonomy())
	if a := r.assetURLFor(pdb.TitleSlug); a != nil {
		repo = repo.WithAssetURL(a)
	}
	return service.NewMediaService(repo, r.timezone,
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
	repo := duckdb.NewMediaRepo(pdb).WithModeTaxonomy(haloInfiniteModeTaxonomy())
	if a := r.assetURLFor(pdb.TitleSlug); a != nil {
		repo = repo.WithAssetURL(a)
	}
	mediaOpts := []service.MediaOption{service.WithMediaWriterAcquirer(mediaWriterAcquirerFor(pdb))}
	if r.jobStore != nil {
		// Transcoding HLS async à l'upload : feed-bump rafraîchit la galerie
		// quand un clip 'processing' devient 'ready'.
		mediaOpts = append(mediaOpts, service.WithMediaTranscoding(r.jobStore, handlers.BumpMediaFeedVersion))
	}
	svc := service.NewMediaService(repo, r.timezone, mediaOpts...)
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
