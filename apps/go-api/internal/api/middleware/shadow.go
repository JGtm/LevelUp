// Package middleware — shadow.go : Shadow mode (Sprint 35).
//
// Quand LEVELUP_SHADOW_MODE=both, le middleware envoie en parallèle
// la même requête à l'API Python (LEVELUP_PYTHON_URL) et log les
// divergences de réponse via slog, sans affecter la réponse Go.
//
// Pattern : shadow read — Go répond normalement, Python est appelé
// en goroutine fire-and-forget avec un timeout court.
package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	shadowTimeout = 3 * time.Second
	// Limite lecture body pour shadow (évite de tout bufferer en mémoire)
	shadowBodyLimit = 512 * 1024 // 512 Ko
)

// ShadowConfig configure le middleware shadow.
type ShadowConfig struct {
	// PythonURL : URL de base de l'API Python (ex: http://127.0.0.1:8001)
	PythonURL string
}

// Shadow retourne un middleware chi qui, en mode "both",
// duplique chaque requête vers l'API Python et log les diffs.
func Shadow(cfg ShadowConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.PythonURL == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Lire et bufferiser le body pour pouvoir le rejouer.
			var bodyBuf []byte
			if r.Body != nil && r.ContentLength != 0 {
				limited := io.LimitReader(r.Body, shadowBodyLimit)
				bodyBuf, _ = io.ReadAll(limited)
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewReader(bodyBuf))
			}

			// Capturer la réponse Go.
			ww := &shadowResponseWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
			next.ServeHTTP(ww, r)

			// Appel shadow asynchrone — fire-and-forget.
			go shadowCall(r, bodyBuf, ww.buf.Bytes(), ww.status, cfg.PythonURL)
		})
	}
}

func shadowCall(
	r *http.Request,
	body []byte,
	goResponse []byte,
	goStatus int,
	pythonURL string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), shadowTimeout)
	defer cancel()

	// Construire la requête shadow vers Python.
	targetURL := strings.TrimRight(pythonURL, "/") + r.RequestURI
	req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		slog.Debug("shadow: erreur construction requête", "err", err)
		return
	}
	// Propager les headers pertinents (Content-Type, Accept).
	for _, h := range []string{"Content-Type", "Accept", "X-Request-Id"} {
		if v := r.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		slog.Debug("shadow: appel Python échec", "path", r.URL.Path, "err", err, "elapsed_ms", elapsed.Milliseconds())
		return
	}
	defer resp.Body.Close()

	pyResponse, err := io.ReadAll(io.LimitReader(resp.Body, shadowBodyLimit))
	if err != nil {
		slog.Debug("shadow: lecture réponse Python échec", "err", err)
		return
	}

	// Log la divergence si statuts ou corps diffèrent.
	goHash := fmt.Sprintf("%x", sha256.Sum256(goResponse))[:8]
	pyHash := fmt.Sprintf("%x", sha256.Sum256(pyResponse))[:8]

	match := resp.StatusCode == goStatus && goHash == pyHash
	if !match {
		slog.Warn("shadow: divergence Go vs Python",
			"path", r.URL.Path,
			"method", r.Method,
			"go_status", goStatus,
			"python_status", resp.StatusCode,
			"go_hash", goHash,
			"python_hash", pyHash,
			"go_bytes", len(goResponse),
			"python_bytes", len(pyResponse),
			"python_elapsed_ms", elapsed.Milliseconds(),
		)
	} else {
		slog.Debug("shadow: réponse identique Go=Python",
			"path", r.URL.Path,
			"hash", goHash,
			"python_elapsed_ms", elapsed.Milliseconds(),
		)
	}
}

// shadowResponseWriter capture le status et la réponse Go pour la comparaison.
type shadowResponseWriter struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	status      int
	wroteHeader bool
}

func (sw *shadowResponseWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *shadowResponseWriter) Write(b []byte) (int, error) {
	sw.buf.Write(b)
	return sw.ResponseWriter.Write(b)
}
