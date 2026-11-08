// Package handlers — media_requester.go : l'identité du demandeur des écritures
// de médias (suppression, like, association).
//
// Le demandeur est le profil POSSÉDÉ par l'utilisateur de la session (son xuid
// lié), résolu par le câblage du routeur ; jamais le « joueur courant » de la
// session, que POST /session/context fixe au gré de l'interface et qui ne prouve
// aucune propriété.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// RequesterSlugResolver rend le slug du profil possédé par l'utilisateur de la
// session de ctx, dans le titre courant ; "" quand il n'en possède aucun. Hors
// propriété appliquée (mono-utilisateur, démo), le câblage rend le joueur choisi
// dans l'interface : il n'y a alors aucune identité à opposer.
type RequesterSlugResolver func(ctx context.Context) string

// WithRequester câble la résolution du demandeur.
func (h *MediaHandler) WithRequester(resolve RequesterSlugResolver) *MediaHandler {
	h.requester = resolve
	return h
}

// requesterSlug rend le demandeur de la requête, "" sans résolveur câblé.
func (h *MediaHandler) requesterSlug(ctx context.Context) string {
	if h.requester == nil {
		return ""
	}
	return h.requester(ctx)
}

// requesterIsAdmin dit si le demandeur est un compte administrateur, rôle relu dans
// le magasin d'utilisateurs (authz.IsAdmin) ; faux sans magasin câblé.
func (h *MediaHandler) requesterIsAdmin(ctx context.Context) bool {
	return authz.IsAdmin(middleware.GetSession(ctx), h.users)
}

// resolveMediaWriteRequester traduit la session HTTP en identité du demandeur d'une écriture de
// média (suppression, association, fusion, oubli...) : le profil possédé par l'utilisateur de la
// session, le rôle admin de son compte, et si l'authentification est appliquée.
//
// Traduction, pas décision : la règle propriétaire/admin vit dans domain.CanModifyMedia, appliquée
// par le service. Seul le cas « aucune identité opposable » est tranché ici, en 401
// `<opName>_requires_session` : en multi-utilisateur authentifié, un refus 403 enverrait l'utilisateur
// chercher une permission alors qu'il lui manque une session. Hors authentification appliquée
// (mono-utilisateur, démo) il n'y a aucune identité à exiger.
func (h *MediaHandler) resolveMediaWriteRequester(
	ctx context.Context,
	opName string,
) (slug string, isAdmin, enforced bool, err error) {
	enforced = h.authEnforced
	slug = h.requesterSlug(ctx)
	isAdmin = h.requesterIsAdmin(ctx)
	if !enforced || isAdmin || slug != "" {
		return slug, isAdmin, enforced, nil
	}
	slog.WarnContext(ctx, "media: écriture refusée, aucun profil possédé par la session", "op", opName)
	return slug, isAdmin, enforced, humacore.NewError(http.StatusUnauthorized, opName+"_requires_session",
		"No owned player profile in session: cannot "+opName+" this media.")
}

// resolveMediaRequester rend l'identité du demandeur sous la forme que portent les requêtes du
// domaine (resolveMediaWriteRequester).
func (h *MediaHandler) resolveMediaRequester(ctx context.Context, opName string) (domain.MediaRequester, error) {
	slug, isAdmin, enforced, err := h.resolveMediaWriteRequester(ctx, opName)
	return domain.MediaRequester{RequesterSlug: slug, RequesterIsAdmin: isAdmin, AuthEnforced: enforced}, err
}

// resolveAssociateRequester pose l'identité du demandeur d'une association média -> match.
func (h *MediaHandler) resolveAssociateRequester(ctx context.Context, req *domain.MediaAssociateRequest) error {
	slug, isAdmin, enforced, err := h.resolveMediaWriteRequester(ctx, "associate")
	req.RequesterSlug, req.RequesterIsAdmin, req.AuthEnforced = slug, isAdmin, enforced
	return err
}

// mediaAssociateError traduit une erreur du service d'association en erreur HTTP.
func mediaAssociateError(ctx context.Context, err error) error {
	return mediaWriteError(ctx, err, "media_associate_forbidden", "associate_error")
}

// mediaWriteError traduit l'erreur d'une écriture sur un média existant (suppression,
// association) : 503 sur base occupée, 400 / 404 / 403 (forbiddenCode) sur erreur
// métier typée, 500 (fallbackCode) sinon, journalisé.
func mediaWriteError(ctx context.Context, err error, forbiddenCode, fallbackCode string) error {
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
			return humacore.NewError(http.StatusForbidden, forbiddenCode, apiErr.Message)
		}
	}
	slog.ErrorContext(ctx, "media: erreur d'écriture non typée", "err", err, "code", fallbackCode)
	return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, fallbackCode, err)
}

// WithUserLookup câble le magasin d'utilisateurs : le rôle admin du demandeur se lit
// dans son compte, jamais dans la session.
func (h *MediaHandler) WithUserLookup(users authz.UserLookup) *MediaHandler {
	h.users = users
	return h
}
