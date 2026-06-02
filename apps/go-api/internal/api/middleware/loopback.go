// Package middleware — loopback.go : garde-fou pour endpoints diagnostic.
//
// LoopbackOnly retourne 403 si la requête ne provient pas de 127.0.0.1 / ::1.
// Utilisé pour les endpoints /_diag/* qui exposent des informations sensibles
// (état interne du scheduler, force d'un sync) sans authentification.
//
// L'idée : en dev local, tout fonctionne par défaut ; en prod derrière un
// reverse proxy, les requêtes externes arrivent avec X-Forwarded-For et
// RemoteAddr = IP du proxy (souvent loopback si même machine), mais si le
// proxy est sur une autre machine, RemoteAddr ne sera pas loopback.
//
// Sécurité : ne pas activer ces endpoints derrière un proxy qui forwarde
// sans assainir RemoteAddr — préférer une vraie auth admin.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
)

// ctxKeyRealRemoteAddr porte l'adresse TCP réelle du peer, capturée avant toute
// réécriture de RemoteAddr par chi RealIP.
type ctxKeyRealRemoteAddr struct{}

// PreserveRemoteAddr stocke l'adresse TCP réelle du peer (r.RemoteAddr tel que vu
// par le serveur HTTP, AVANT toute réécriture par chi RealIP à partir des en-têtes
// X-Real-IP / X-Forwarded-For / True-Client-IP) dans le contexte de la requête.
//
// À monter en TÊTE de chaîne, avant RealIP. LoopbackOnly lit cette valeur, ce qui
// rend le garde insensible aux en-têtes d'IP client falsifiables : sans ça, un
// attaquant externe envoyant `X-Real-IP: 127.0.0.1` derrière un proxy/CDN pouvait
// se faire passer pour loopback et atteindre les endpoints /_diag (revue P0
// 2026-06-02).
func PreserveRemoteAddr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyRealRemoteAddr{}, r.RemoteAddr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// realRemoteAddr retourne l'adresse peer capturée par PreserveRemoteAddr, ou
// r.RemoteAddr en fallback si le middleware n'a pas été monté en amont.
func realRemoteAddr(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyRealRemoteAddr{}).(string); ok && v != "" {
		return v
	}
	return r.RemoteAddr
}

// LoopbackOnly bloque les requêtes non-loopback avec 403.
// Compare l'adresse TCP réelle du peer (capturée par PreserveRemoteAddr avant
// RealIP), pas le RemoteAddr courant qui peut avoir été réécrit depuis des en-têtes
// X-Forwarded-* / X-Real-IP facilement falsifiables.
func LoopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := realRemoteAddr(r)
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
