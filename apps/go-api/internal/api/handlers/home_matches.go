// Package handlers — home_matches.go : GET /players/{player_slug}/pages/home/matches, la suite
// des tuiles de match de l'Accueil (D-8). La tête de la liste est servie avec /pages/home ; ce
// endpoint rend les pages suivantes par curseur et les favoris en entier. Le handler ne porte
// aucune logique : fenêtrage, construction et enrichissement des tuiles sont dans HomeService.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// homeMatchesCacheControl : la liste change au moindre sync ou favori, et le client la tient déjà
// en cache de requêtes — le navigateur revalide (ETag, 304) au lieu de resservir une liste
// périmée (un favori basculé doit se voir à la relecture).
const homeMatchesCacheControl = "private, no-cache"

// homeMatchesInput : {player_slug}, X-LevelUp-Locale, If-None-Match et la fenêtre demandée.
type homeMatchesInput struct {
	PlayerSlug  string `path:"player_slug"`
	Locale      string `header:"X-LevelUp-Locale"`
	IfNoneMatch string `header:"If-None-Match"`
	// Kind : « recent » (défaut) = l'historique par curseur ; « favorites » = les favoris en
	// entier (before et limit ignorés).
	Kind string `query:"kind" enum:"recent,favorites" default:"recent"`
	// Before : curseur `next_before` de la page précédente ; absent = depuis la plus récente.
	Before string `query:"before" maxLength:"256"`
	// Limit : tuiles par page (30 par défaut, 60 au plus).
	Limit int `query:"limit" minimum:"1" maximum:"60"`
}

// handleGetHomeMatches retourne une liste de tuiles de l'Accueil.
// GET /api/v1/players/{player_slug}/pages/home/matches
func (h *HomeHandler) handleGetHomeMatches(ctx context.Context, in *homeMatchesInput) (*homePageOutput, error) {
	svc, sctx, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.ErrorContext(ctx, "home: newSvc error", "slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}
	page, err := svc.GetHomeMatches(sctx, h.resolveLocaleFromHeader(ctx, in.Locale), domain.HomeMatchesRequest{
		Kind: in.Kind, Before: in.Before, Limit: in.Limit,
	})
	if err != nil {
		if errors.Is(err, domain.ErrHomeCursorInvalid) {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_cursor", "curseur de pagination invalide")
		}
		return nil, homePageError(sctx, err, in.PlayerSlug)
	}
	return homeJSONOutput(sctx, page, in.IfNoneMatch, homeMatchesCacheControl)
}
