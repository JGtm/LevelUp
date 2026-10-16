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

// sessionIsAdmin dit si la session porte le rôle admin.
func sessionIsAdmin(ctx context.Context) bool {
	sess := middleware.GetSession(ctx)
	return sess != nil && sess.Role != nil && *sess.Role == string(domain.RoleAdmin)
}

// resolveAssociateRequester pose l'identité du demandeur d'une association média ->
// match. Comme pour la suppression, la règle propriétaire/admin vit dans
// domain.CanModifyMedia (appliquée par le service) ; seul le cas « aucune identité
// opposable » est tranché ici, en 401.
func (h *MediaHandler) resolveAssociateRequester(ctx context.Context, req *domain.MediaAssociateRequest) error {
	req.AuthEnforced = h.authEnforced
	req.RequesterSlug = h.requesterSlug(ctx)
	req.RequesterIsAdmin = sessionIsAdmin(ctx)
	if !req.AuthEnforced || req.RequesterIsAdmin || req.RequesterSlug != "" {
		return nil
	}
	slog.WarnContext(ctx, "media_associate refusé: aucun profil possédé par la session",
		"file_path", req.FilePath)
	return humacore.NewError(http.StatusUnauthorized, "associate_requires_session",
		"No owned player profile in session: cannot associate this media.")
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
	return humacore.NewError(http.StatusInternalServerError, fallbackCode, err.Error())
}
