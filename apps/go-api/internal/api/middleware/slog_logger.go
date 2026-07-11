// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : logging structuré slog avec durée de requête et request_id.
// Sprint 35 : ajout response_bytes dans chaque log de requête.
// Sprint B1 commit 18 : injection event_id auto par requête (http.<METHOD>:<id>).
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
)

// SlogLogger est un middleware chi qui log chaque requête via slog.
// Remplace chimiddleware.Logger pour utiliser slog natif Go 1.21+.
// Niveaux : 2xx/3xx → DEBUG (silencieux en prod), 4xx → WARN, 5xx → ERROR.
//
// Sprint B1 commit 18 : injecte un event_id "http.<METHOD>:<short_id>" dans
// le ctx AVANT next.ServeHTTP. Tous les logs émis par les handlers et
// services downstream hériteront automatiquement cet event_id via le
// ContextHandler — permet de grep une requête HTTP cross-module dans
// logs/{handlers,sync,duckdb,...}.log.
func SlogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		// Injecte event_id auto dans le ctx. Prefix = "http.<METHOD>" pour
		// permettre des greps catégoriels (`grep 'http.POST' logs/`) en plus
		// du grep par event_id exact.
		ctx, evID := logging.WithEvent(r.Context(), "http."+r.Method)
		r = r.WithContext(ctx)

		next.ServeHTTP(ww, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"response_bytes", ww.bytesWritten,
			"request_id", w.Header().Get(headerRequestID),
			"remote_addr", r.RemoteAddr,
			"title_slug", ctxkeys.TitleSlug(r.Context()),
			"event", evID,
		}
		switch {
		case ww.status >= 500:
			slog.ErrorContext(ctx, "http", attrs...)
		case ww.status >= 400:
			slog.WarnContext(ctx, "http", attrs...)
		default:
			slog.DebugContext(ctx, "http", attrs...)
		}

		// Compteurs HTTP par classe de statut (plan monitoring A7, DC-6) : expvar
		// process-wide + variante par titre (observability.titled — cardinalité
		// bornée au titre, JAMAIS par route). Remplace l'error tracker HTTP
		// supprimé (ADR 0009) par un simple compteur.
		countHTTPStatusClass(ctxkeys.TitleSlug(r.Context()), ww.status)
	})
}

// countHTTPStatusClass incrémente le compteur de la classe de statut (2xx/3xx/
// 4xx/5xx), titre-aware (DC-6 : pas de dimension route). Convention MT-05 :
// titre par défaut → clé NUE (un seul incrément, jamais de double comptage) ;
// un 2e titre obtient sa clé dédiée `<title>.http_status_*`.
func countHTTPStatusClass(titleSlug string, status int) {
	var class string
	switch {
	case status >= 500:
		class = "5xx"
	case status >= 400:
		class = "4xx"
	case status >= 300:
		class = "3xx"
	default:
		class = "2xx"
	}
	observability.IncCounterT(titleSlug, "http_status_"+class+"_total")
}

// statusResponseWriter capture le status code et le nombre d'octets écrits.
type statusResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	wroteHeader  bool
}

func (sw *statusResponseWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusResponseWriter) Write(b []byte) (int, error) {
	n, err := sw.ResponseWriter.Write(b)
	sw.bytesWritten += int64(n)
	return n, err
}
