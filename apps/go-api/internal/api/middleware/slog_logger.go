// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : logging structuré slog avec durée de requête et request_id.
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// SlogLogger est un middleware chi qui log chaque requête via slog.
// Remplace chimiddleware.Logger pour utiliser slog natif Go 1.21+.
func SlogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", w.Header().Get(headerRequestID),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// statusResponseWriter capture le status code de la réponse.
type statusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusResponseWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}
