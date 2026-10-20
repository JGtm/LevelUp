// Package middleware — loopback.go : pair TCP de la connexion et garde-fou des
// endpoints de diagnostic.
//
// LoopbackOnly retourne 403 si le pair TCP n'est pas 127.0.0.1 / ::1. Utilisé pour
// les endpoints /_diag/* qui exposent des informations sensibles (état interne du
// scheduler, force d'un sync) sans authentification.
//
// Le pair TCP est l'adresse de la connexion, que le demandeur ne peut pas choisir.
// Derrière un reverse proxy, c'est le proxy : s'il tourne sur la même machine et
// joint le serveur par loopback, toute requête qu'il relaie est « locale » pour ce
// garde. Ne pas exposer ces endpoints derrière un tel proxy sans autre contrôle —
// préférer une vraie auth admin.
//
// Ce fichier est le seul lecteur de r.RemoteAddr du module (garde-rail
// archlint/client_ip_test.go) : l'IP cliente des journaux et du limiteur se lit par
// chimiddleware.GetClientIP, le pair TCP par PeerAddr.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
)

// ctxKeyPeerAddr porte l'adresse TCP du pair, capturée en tête de chaîne.
type ctxKeyPeerAddr struct{}

// PreserveRemoteAddr stocke l'adresse TCP du pair (r.RemoteAddr tel que vu par le
// serveur HTTP) dans le contexte de la requête, pour PeerAddr.
//
// À monter en TÊTE de chaîne (applyTransverseMiddlewares) : ce qui suit ne peut
// alors plus changer ce que PeerAddr rend, même en réécrivant r.RemoteAddr.
func PreserveRemoteAddr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyPeerAddr{}, r.RemoteAddr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PeerAddr rend l'adresse TCP du pair (« hôte:port ») capturée par
// PreserveRemoteAddr, ou r.RemoteAddr si ce middleware n'est pas monté en amont
// (routeur de test nu). Jamais un en-tête : c'est l'adresse sur laquelle décident
// les gardes « requête locale » (LoopbackOnly, garde du rejeu).
func PeerAddr(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyPeerAddr{}).(string); ok && v != "" {
		return v
	}
	return r.RemoteAddr
}

// LoopbackOnly bloque avec 403 les requêtes dont le pair TCP (PeerAddr) n'est pas
// une adresse de loopback.
func LoopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := PeerAddr(r)
		host, _, err := net.SplitHostPort(peer)
		if err != nil {
			host = peer
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			slog.WarnContext(r.Context(), "middleware.loopback: refus accès non-loopback",
				"remote_addr", peer, "path", r.URL.Path)
			http.Error(w, "forbidden: loopback only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
