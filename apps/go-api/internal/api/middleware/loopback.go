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
	"net"
	"net/http"
)

// LoopbackOnly bloque les requêtes non-loopback avec 403.
// Compare uniquement RemoteAddr (IP du peer TCP direct), ignore les headers
// X-Forwarded-* qui sont facilement falsifiables.
func LoopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden: loopback only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
