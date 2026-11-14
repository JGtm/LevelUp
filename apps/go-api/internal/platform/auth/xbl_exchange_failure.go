package auth

import (
	"context"
	"errors"
	"net"
	"net/http"

	"levelup/go-api/internal/util/httpstatus"
)

// HaloServicesUnreachable dit si l'échec de la chaîne d'échange Xbox -> Halo vient des services
// eux-mêmes (délai dépassé, réseau coupé, panne 5xx, demande de ralentir 429, délai de requête
// 408) et non d'un refus du compte (pas de profil Xbox, mauvais compte : réponses 4xx, jeton
// absent de la réponse). Le premier cas se règle en réessayant plus tard, le second non.
//
// La brique d'échange rend des erreurs enveloppées (`%w`) pour le réseau et un texte « HTTP <code> »
// pour les statuts : on lit l'un par errors.Is / errors.As, l'autre par util/httpstatus.
func HaloServicesUnreachable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	status := httpstatus.FromError(err)
	return httpstatus.IsServerError(status) || status == http.StatusTooManyRequests || status == http.StatusRequestTimeout
}
