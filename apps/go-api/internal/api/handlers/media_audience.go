// Package handlers — media_audience.go : les auteurs dont une requête peut montrer les
// médias (galerie, liste des auteurs, fichiers, onglet Médias d'un match et du rejeu).
package handlers

import (
	"context"
	"log/slog"
	"path"
	"strings"

	"levelup/go-api/internal/domain"
)

// MediaAudienceResolver rend l'audience média de la page du joueur `pageSlug` pour la
// session de ctx (règle : authz.MediaAuthorXUIDs).
type MediaAudienceResolver func(ctx context.Context, pageSlug string) (domain.MediaAudience, error)

// WithMediaAudience câble le résolveur d'audience de la galerie, de la liste des
// auteurs et du service des fichiers.
func (h *MediaHandler) WithMediaAudience(resolve MediaAudienceResolver) *MediaHandler {
	h.audience = resolve
	return h
}

// resolveMediaAudience rend l'audience de la page. Sans résolveur câblé ou en échec
// (journalisé), elle se replie sur le seul joueur de la page : jamais sur tout le monde.
func resolveMediaAudience(ctx context.Context, resolve MediaAudienceResolver, pageSlug string) domain.MediaAudience {
	if resolve == nil {
		return domain.NewMediaAudience(pageSlug)
	}
	audience, err := resolve(ctx, pageSlug)
	if err != nil {
		slog.ErrorContext(ctx, "media: audience non résolue — repli sur les seuls médias du joueur",
			"player", pageSlug, "err", err)
		return domain.NewMediaAudience(pageSlug)
	}
	return audience
}

// filterMatchMediaByAudience retire d'un onglet Médias les médias dont l'auteur est hors
// de l'audience ; un média sans auteur connu est retiré.
func filterMatchMediaByAudience(items []domain.MatchAssociatedMedia, audience domain.MediaAudience) []domain.MatchAssociatedMedia {
	out := make([]domain.MatchAssociatedMedia, 0, len(items))
	for _, it := range items {
		if it.OwnerGamertag != nil && audience.Allows(*it.OwnerGamertag) {
			out = append(out, it)
		}
	}
	return out
}

// mediaURLOwner rend le premier segment d'un chemin de fichier média servi, qui nomme
// son propriétaire dans le format canonique {owner}/{rel}.
func mediaURLOwner(cleanURL string) string {
	first, _, _ := strings.Cut(path.Clean(cleanURL), "/")
	return first
}
