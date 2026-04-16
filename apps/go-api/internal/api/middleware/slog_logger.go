// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : logging structuré slog avec durée de requête et request_id.
// Sprint 35 : ajout response_bytes dans chaque log de requête.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/ctxkeys"
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
			"response_bytes", ww.bytesWritten,
			"request_id", w.Header().Get(headerRequestID),
			"remote_addr", r.RemoteAddr,
			"title_slug", ctxkeys.TitleSlug(r.Context()),
		)
	})
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
