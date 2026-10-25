// Package api — media_audience.go : câblage de l'audience des médias (auteurs visibles).
package api

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/groupstore"
)

// coMemberLookup rend les xuids co-membres de groupe d'un xuid (lui compris), nil sans
// groupe (cf. groupstore.CoMemberXUIDs).
type coMemberLookup func(xuid string) (map[string]bool, error)

// mediaAudienceResolver résout les auteurs visibles sur la page d'un joueur : la règle
// est authz.MediaAuthorXUIDs (joueur de la page et co-membres de ses groupes, bornés par
// la portée du lecteur, authz.MediaViewerScope) ; ce câblage traduit les xuids en noms
// d'auteur (slug et gamertag des profils du titre courant). Les groupes sont transverses
// aux titres (PMT-4, cf. familyXUIDResolver).
func mediaAudienceResolver(
	enforced bool,
	loadPlayers func(titleSlug string) ([]domain.PlayerSummary, error),
	coMembers coMemberLookup,
	users authz.UserLookup,
) handlers.MediaAudienceResolver {
	return func(ctx context.Context, pageSlug string) (domain.MediaAudience, error) {
		players, err := loadPlayers(ctxkeys.TitleSlug(ctx))
		if err != nil {
			return domain.MediaAudience{}, fmt.Errorf("media audience: profils: %w", err)
		}
		pageXUID := ""
		for i := range players {
			if strings.EqualFold(players[i].PlayerSlug, pageSlug) {
				pageXUID = players[i].XUID
				break
			}
		}
		if pageXUID == "" {
			return domain.NewMediaAudience(pageSlug), nil
		}
		pageCo, err := coMembers(pageXUID)
		if err != nil {
			return domain.MediaAudience{}, fmt.Errorf("media audience: groupes du joueur: %w", err)
		}
		var viewerScope map[string]bool
		if enforced {
			user := authz.CurrentUser(middleware.GetSession(ctx), users)
			var viewerCo map[string]bool
			if user != nil && user.XUID != "" {
				if viewerCo, err = coMembers(user.XUID); err != nil {
					return domain.MediaAudience{}, fmt.Errorf("media audience: groupes du lecteur: %w", err)
				}
			}
			viewerScope = authz.MediaViewerScope(true, user, viewerCo)
		}
		xuids := authz.MediaAuthorXUIDs(pageXUID, pageCo, viewerScope)
		names := []string{pageSlug}
		for _, p := range players {
			if xuids[p.XUID] {
				names = append(names, p.PlayerSlug, p.Gamertag)
			}
		}
		return domain.NewMediaAudience(names...), nil
	}
}

// groupCoMembers adapte le magasin de groupes ; sans magasin, personne n'a de groupe.
func groupCoMembers(groups *groupstore.GroupStore) coMemberLookup {
	if groups == nil {
		return func(string) (map[string]bool, error) { return nil, nil }
	}
	return groups.CoMemberXUIDs
}
