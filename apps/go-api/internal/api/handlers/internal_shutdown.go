// Package handlers — internal_shutdown.go : POST /api/v1/internal/shutdown, l'arrêt propre
// demandé par le processus qui a lancé le serveur (coquille desktop, pré-commande d'air).
//
// Trois portes, dans cet ordre : la route n'est montée que si LEVELUP_DESKTOP_SECRET est
// posé (un serveur hébergé n'a pas cette route) ; LoopbackOnly (montage, server_apiv1.go)
// refuse tout pair TCP hors boucle locale ; l'en-tête X-LevelUp-Shutdown-Secret est comparé
// en temps constant au secret. Le secret ne vit jamais dans un argument de ligne de commande
// et n'est jamais journalisé.
//
// La réponse (202) part avant l'arrêt : la demande est transmise en arrière-plan à
// config.AppConfig.ShutdownRequester, qui rejoint le chemin unique du signal d'arrêt
// (cmd/server : bootServer.requestShutdown, même budget).
package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

// ShutdownSecretHeader : en-tête qui porte le secret d'arrêt.
const ShutdownSecretHeader = "X-LevelUp-Shutdown-Secret"

// shutdownReasonHTTP : cause journalisée d'un arrêt demandé par cette route.
const shutdownReasonHTTP = "http"

// InternalShutdownHandler sert la route d'arrêt propre.
type InternalShutdownHandler struct {
	// secretDigest : empreinte du secret attendu. Comparer des empreintes de même longueur
	// garde la comparaison à temps constant quelle que soit la longueur de l'en-tête reçu.
	secretDigest [sha256.Size]byte
	configured   bool
	request      func(reason string)
}

// NewInternalShutdownHandler : secret vide ou request nil -> la route répond 404 (non
// configurée) ; le montage ne l'expose de toute façon qu'avec un secret.
func NewInternalShutdownHandler(secret string, request func(reason string)) *InternalShutdownHandler {
	return &InternalShutdownHandler{
		secretDigest: sha256.Sum256([]byte(secret)),
		configured:   secret != "" && request != nil,
		request:      request,
	}
}

// Handle : 404 si non configurée, 403 si le secret manque ou est faux, 202 sinon puis
// demande d'arrêt asynchrone.
func (h *InternalShutdownHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if !h.configured {
		writeErrorFrom(r.Context(), w, http.StatusNotFound, "not_found", nil)
		return
	}
	got := sha256.Sum256([]byte(r.Header.Get(ShutdownSecretHeader)))
	if subtle.ConstantTimeCompare(got[:], h.secretDigest[:]) != 1 {
		writeErrorFrom(r.Context(), w, http.StatusForbidden, "forbidden", nil)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	go h.request(shutdownReasonHTTP)
}
