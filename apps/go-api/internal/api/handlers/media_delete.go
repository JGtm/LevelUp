// Package handlers — media_delete.go : suppression définitive d'un média
// (v7.3 lot 2, item 3.1).
//
// Endpoint : DELETE /api/v1/players/{player_slug}/media?file_path=...
//
// La route est enregistrée dans le groupe /players/{player_slug}, déjà gardé par
// RequirePlayerOwnership + RequireCapability(CapMedia) — aucune allowlist du
// ratchet bare_routes n'est nécessaire.
//
// Ce fichier ne contient AUCUNE règle d'autorisation : il traduit la session en
// identité de demandeur, la règle vit dans domain.CanDeleteMedia et son
// application dans MediaService.DeleteMedia. Le middleware d'ownership ne peut
// pas la porter — il laisse passer les co-membres de groupe (authz.CanAccessPlayer),
// que la décision utilisateur exclut.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// mediaDeleteInput : {player_slug} + query file_path (requis).
// Query plutôt que corps : un DELETE à corps est mal supporté par les clients
// HTTP, et le voisin /media/match-candidates prend déjà file_path en query.
type mediaDeleteInput struct {
	PlayerSlug string `path:"player_slug"`
	FilePath   string `query:"file_path"`
}

type mediaDeleteOutput struct{ Body *domain.MediaDeleteResponse }

// handleDeleteMedia supprime définitivement un média.
// DELETE /api/v1/players/{player_slug}/media?file_path=...
func (h *MediaHandler) handleDeleteMedia(ctx context.Context, in *mediaDeleteInput) (*mediaDeleteOutput, error) {
	if in.FilePath == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_request", "file_path est requis")
	}
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	// Le client rejoue le file_path qu'il a reçu, c'est-à-dire une URL servable :
	// on la reverse vers la clé de stockage via l'unique helper, comme le like et
	// l'association (cf. media_likes.go — le bug 1.5 est né d'une conversion
	// divergente à ce point précis).
	rawPath := in.FilePath
	req := domain.MediaDeleteRequest{
		FilePath: mediaServableURLToStoredPath(in.PlayerSlug, in.FilePath),
	}
	if err := h.resolveDeleteRequester(ctx, &req); err != nil {
		return nil, err
	}

	resp, err := svc.DeleteMedia(ctx, req)
	if err != nil {
		return nil, mediaDeleteError(ctx, err)
	}

	// Écho du chemin REÇU : le cache client indexe ses items par cette URL
	// (même contrat que le like).
	resp.FilePath = rawPath
	BumpMediaFeedVersion()
	return &mediaDeleteOutput{Body: resp}, nil
}

// resolveDeleteRequester traduit la session HTTP en identité de demandeur.
//
// Traduction, pas décision : la règle propriétaire/admin vit dans
// domain.CanDeleteMedia. La seule chose tranchée ici est le cas « aucune
// identité opposable », qui relève de l'authentification et non des droits.
//
// AuthEnforced reflète le mode de déploiement — en mono-utilisateur et en démo,
// RequireAdmin et RequirePlayerOwnership sont eux-mêmes inertes (authz.Enforced),
// et exiger une session rendrait la suppression impossible sur une instance
// locale, qui est le cas d'usage principal.
func (h *MediaHandler) resolveDeleteRequester(ctx context.Context, req *domain.MediaDeleteRequest) error {
	req.AuthEnforced = h.authEnforced
	sess := middleware.GetSession(ctx)
	if sess != nil {
		if sess.CurrentPlayerSlug != nil {
			req.RequesterSlug = *sess.CurrentPlayerSlug
		}
		if sess.Role != nil && *sess.Role == string(domain.RoleAdmin) {
			req.RequesterIsAdmin = true
		}
	}
	if !req.AuthEnforced || req.RequesterIsAdmin || req.RequesterSlug != "" {
		return nil
	}
	// GARDE ANTI-SILENCE (même motif que le like, item 1.5) : en multi-utilisateur
	// authentifié, une suppression sans identité ne doit pas être évaluée comme un
	// refus de droits — 403 enverrait l'utilisateur chercher une permission alors
	// qu'il lui manque une session.
	slog.WarnContext(ctx, "media_delete refusé: aucun joueur courant en session",
		"file_path", req.FilePath)
	return humacore.NewError(http.StatusUnauthorized, "delete_requires_session",
		"Aucun joueur courant en session : impossible de supprimer ce média.")
}

// mediaDeleteError traduit une erreur de service en erreur HTTP.
// 401 quand l'auth est appliquée mais qu'aucune identité n'a pu être opposée :
// sans session, un refus 403 laisserait croire à un problème de droits alors que
// le client n'a qu'à se reconnecter.
func mediaDeleteError(ctx context.Context, err error) error {
	if errors.Is(err, dblease.ErrDBLocked) {
		return errDBBusy()
	}
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "bad_request":
			return humacore.NewError(http.StatusBadRequest, apiErr.Code, apiErr.Message)
		case "not_found":
			return humacore.NewError(http.StatusNotFound, apiErr.Code, apiErr.Message)
		case "forbidden":
			return humacore.NewError(http.StatusForbidden, "media_delete_forbidden", apiErr.Message)
		}
	}
	slog.ErrorContext(ctx, "media_delete: erreur non typée", "err", err)
	return humacore.NewError(http.StatusInternalServerError, "media_delete_error", err.Error())
}
