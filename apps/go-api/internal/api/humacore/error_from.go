package humacore

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/domain"
)

// internalErrorMessage : message de toute réponse 5xx.
const internalErrorMessage = "internal error"

// FixedMessage rend le message envoyé au client pour (statut, code) : « internal error » en
// 5xx, sinon le message fixe du code (domain.ErrorMessageForCode). Le texte d'une erreur
// interne ne part jamais au client.
func FixedMessage(status int, code string) string {
	if status >= http.StatusInternalServerError {
		return internalErrorMessage
	}
	return domain.ErrorMessageForCode(code)
}

// LogErrorResponse journalise une réponse d'erreur avec son niveau selon le statut :
//   - 5xx -> ERROR (erreur serveur, toujours visible) ;
//   - 401/403/422 -> WARN (refus d'authentification ou de validation, tracé pour audit) ;
//   - autres 4xx -> DEBUG (volume, à la demande).
//
// Le ctx fait remonter event_id / request_id injectés par le ContextHandler.
func LogErrorResponse(ctx context.Context, status int, code, message string, err error) {
	attrs := []any{"status", status, "code", code, "message", message}
	if err != nil {
		attrs = append(attrs, "err", err)
	}
	switch {
	case status >= http.StatusInternalServerError:
		slog.ErrorContext(ctx, "http: erreur réponse", attrs...)
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusUnprocessableEntity:
		slog.WarnContext(ctx, "http: refus", attrs...)
	case status >= http.StatusBadRequest:
		slog.DebugContext(ctx, "http: client error", attrs...)
	}
}

// NewErrorFrom construit l'erreur Huma d'un refus dont la cause est err : la cause est
// journalisée (LogErrorResponse), le client reçoit le message fixe du code. C'est le seul
// constructeur à employer quand le refus vient d'une erreur (garde-rail archlint
// err_error_response_test) ; NewError reste pour un message écrit à la main.
func NewErrorFrom(ctx context.Context, status int, code string, err error) huma.StatusError {
	msg := FixedMessage(status, code)
	LogErrorResponse(ctx, status, code, msg, err)
	return NewError(status, code, msg)
}
