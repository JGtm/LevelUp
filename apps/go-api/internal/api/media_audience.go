// Package api — media_audience.go : câblage du service d'audience des médias sur les handlers.
package api

import (
	"context"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/service"
)

// mediaAudienceResolver adapte service.MediaAudienceService au résolveur des handlers : la session
// du lecteur se lit dans le contexte de la requête (middleware) et se passe au service.
func mediaAudienceResolver(svc *service.MediaAudienceService) handlers.MediaAudienceResolver {
	return func(ctx context.Context, pageSlug string) (domain.MediaAudience, error) {
		return svc.Resolve(ctx, middleware.GetSession(ctx), pageSlug)
	}
}

// groupCoMembers adapte le magasin de groupes ; sans magasin, personne n'a de groupe.
func groupCoMembers(groups *groupstore.GroupStore) service.CoMemberLookup {
	if groups == nil {
		return func(string) (map[string]bool, error) { return nil, nil }
	}
	return groups.CoMemberXUIDs
}
