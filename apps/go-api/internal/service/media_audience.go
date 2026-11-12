// Package service — media_audience.go : les auteurs dont une requête peut montrer les médias.
package service

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// CoMemberLookup rend les xuids co-membres de groupe d'un xuid (lui compris), nil sans groupe
// (cf. groupstore.CoMemberXUIDs).
type CoMemberLookup func(xuid string) (map[string]bool, error)

// PlayersLoader rend les profils suivis d'un titre.
type PlayersLoader func(titleSlug string) ([]domain.PlayerSummary, error)

// MediaAudienceService résout les auteurs visibles sur la page d'un joueur : la règle est
// authz.MediaAuthorXUIDs (joueur de la page et co-membres de ses groupes, bornés par la portée du
// lecteur, authz.MediaViewerScope) ; le service traduit les xuids en noms d'auteur (slug et
// gamertag des profils du titre courant). Les groupes sont transverses aux titres.
type MediaAudienceService struct {
	enforced    bool
	loadPlayers PlayersLoader
	coMembers   CoMemberLookup
	users       authz.UserLookup
}

// NewMediaAudienceService construit le service. enforced : la propriété des joueurs s'applique
// (sinon aucune portée de lecteur ne borne l'audience).
func NewMediaAudienceService(
	enforced bool, loadPlayers PlayersLoader, coMembers CoMemberLookup, users authz.UserLookup,
) *MediaAudienceService {
	return &MediaAudienceService{enforced: enforced, loadPlayers: loadPlayers, coMembers: coMembers, users: users}
}

// Resolve rend l'audience média de la page du joueur pageSlug pour le lecteur de session sess
// (nil : anonyme), sur le titre du contexte. Une page sans profil suivi ne montre que ses médias.
func (s *MediaAudienceService) Resolve(ctx context.Context, sess *domain.SessionData, pageSlug string) (domain.MediaAudience, error) {
	players, err := s.loadPlayers(ctxkeys.TitleSlug(ctx))
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
	pageCo, err := s.coMembers(pageXUID)
	if err != nil {
		return domain.MediaAudience{}, fmt.Errorf("media audience: groupes du joueur: %w", err)
	}
	viewerScope, err := s.viewerScope(sess)
	if err != nil {
		return domain.MediaAudience{}, err
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

// viewerScope rend la portée du lecteur ; nil quand la propriété des joueurs ne s'applique pas.
func (s *MediaAudienceService) viewerScope(sess *domain.SessionData) (map[string]bool, error) {
	if !s.enforced {
		return nil, nil
	}
	user := authz.CurrentUser(sess, s.users)
	var viewerCo map[string]bool
	if user != nil && user.XUID != "" {
		var err error
		if viewerCo, err = s.coMembers(user.XUID); err != nil {
			return nil, fmt.Errorf("media audience: groupes du lecteur: %w", err)
		}
	}
	return authz.MediaViewerScope(true, user, viewerCo), nil
}
