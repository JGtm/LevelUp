// Package middleware fournit les middlewares HTTP transverses.
// Sprint 4 : logging structuré slog avec durée de requête et request_id.
// Sprint 35 : ajout response_bytes dans chaque log de requête.
// Sprint B1 commit 18 : injection event_id auto par requête (http.<METHOD>:<id>).
// Plan perf 2026-09-23, lot L1 : requêtes lentes en INFO + sections chronométrées.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/observability/timing"
)

// envSlowRequestMS : seuil (millisecondes) à partir duquel une requête est « lente ». Lu
// UNE fois, au montage du middleware (plan perf 2026-09-23, décision D1.1).
const envSlowRequestMS = "LEVELUP_SLOW_REQUEST_MS"

// defaultSlowRequestMS : seuil appliqué quand la variable est absente ou invalide.
const defaultSlowRequestMS = 1000

// SlogLogger est un middleware chi qui log chaque requête via slog.
// Remplace chimiddleware.Logger pour utiliser slog natif Go 1.21+.
// Niveaux : 2xx/3xx → DEBUG (silencieux en prod), 4xx → WARN, 5xx → ERROR.
// Requête lente (durée ≥ LEVELUP_SLOW_REQUEST_MS, défaut 1000) : même ligne `http` avec
// `slow: true`, au moins en INFO — un 2xx/3xx lent passe en INFO, un 4xx/5xx lent garde
// son niveau WARN/ERROR (jamais rétrogradé).
//
// Sprint B1 commit 18 : injecte un event_id "http.<METHOD>:<short_id>" dans
// le ctx AVANT next.ServeHTTP. Tous les logs émis par les handlers et
// services downstream hériteront automatiquement cet event_id via le
// ContextHandler — permet de grep une requête HTTP cross-module dans
// logs/{handlers,sync,duckdb,...}.log.
//
// Plan perf, lot L1 : pose aussi un `*timing.Timings` sur le ctx de CHAQUE requête ; les
// services y chronomètrent leurs sections (`defer timing.FromContext(ctx).Section("x")()`).
// Après le handler, s'il existe au moins une section ET que la requête est lente ou que le
// niveau DEBUG est actif, une ligne INFO `http_timings` les détaille (cf. logSectionTimings).
func SlogLogger(next http.Handler) http.Handler {
	slowThreshold := slowRequestThreshold()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		// Injecte event_id auto dans le ctx. Prefix = "http.<METHOD>" pour
		// permettre des greps catégoriels (`grep 'http.POST' logs/`) en plus
		// du grep par event_id exact.
		ctx, evID := logging.WithEvent(r.Context(), "http."+r.Method)
		ctx, timings := timing.WithTimings(ctx)
		r = r.WithContext(ctx)

		next.ServeHTTP(ww, r)

		elapsed := time.Since(start)
		slow := elapsed >= slowThreshold
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", elapsed.Milliseconds(),
			"response_bytes", ww.bytesWritten,
			"request_id", w.Header().Get(headerRequestID),
			"remote_addr", r.RemoteAddr,
			"title_slug", ctxkeys.TitleSlug(r.Context()),
			"event", evID,
		}
		if slow {
			attrs = append(attrs, "slow", true)
		}
		logRequest(ctx, ww.status, slow, attrs)
		logSectionTimings(ctx, r.URL.Path, elapsed, slow, timings)

		// Compteurs HTTP par classe de statut (plan monitoring A7, DC-6) : expvar
		// process-wide + variante par titre (observability.titled — cardinalité
		// bornée au titre, JAMAIS par route). Remplace l'error tracker HTTP
		// supprimé (ADR 0009) par un simple compteur.
		countHTTPStatusClass(ctxkeys.TitleSlug(r.Context()), ww.status)
	})
}

// logRequest émet la ligne `http` au niveau de la requête : ERROR (5xx), WARN (4xx), INFO
// (2xx/3xx lente), DEBUG (2xx/3xx sous le seuil).
func logRequest(ctx context.Context, status int, slow bool, attrs []any) {
	switch {
	case status >= 500:
		slog.ErrorContext(ctx, "http", attrs...)
	case status >= 400:
		slog.WarnContext(ctx, "http", attrs...)
	case slow:
		slog.InfoContext(ctx, "http", attrs...)
	default:
		slog.DebugContext(ctx, "http", attrs...)
	}
}

// logSectionTimings émet la ligne INFO `http_timings` (path, duration_ms, puis
// timing.LogAttrs : total_ms, sections, calls) quand la requête a chronométré au moins une
// section ET qu'elle est lente ou que le niveau DEBUG est actif. Une requête rapide sans
// DEBUG ne coûte donc aucun formatage.
func logSectionTimings(ctx context.Context, path string, elapsed time.Duration, slow bool, t *timing.Timings) {
	if !slow && !slog.Default().Enabled(ctx, slog.LevelDebug) {
		return
	}
	sections := t.LogAttrs()
	if len(sections) == 0 {
		return
	}
	attrs := make([]any, 0, 4+len(sections))
	attrs = append(attrs, "path", path, "duration_ms", elapsed.Milliseconds())
	attrs = append(attrs, sections...)
	slog.InfoContext(ctx, "http_timings", attrs...)
}

// slowRequestThreshold lit LEVELUP_SLOW_REQUEST_MS (entier strictement positif, en
// millisecondes). Absente → défaut ; invalide → défaut, et l'erreur est journalisée (une
// faute de frappe ne doit pas désactiver le journal des requêtes lentes en silence).
func slowRequestThreshold() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envSlowRequestMS))
	if raw == "" {
		return defaultSlowRequestMS * time.Millisecond
	}
	ms, err := strconv.Atoi(raw)
	if err == nil && ms <= 0 {
		err = fmt.Errorf("%s doit être un entier strictement positif, reçu %d", envSlowRequestMS, ms)
	}
	if err != nil {
		slog.ErrorContext(context.Background(), "http: seuil de requête lente invalide, défaut appliqué",
			"err", err, "value", raw, "default_ms", defaultSlowRequestMS)
		return defaultSlowRequestMS * time.Millisecond
	}
	return time.Duration(ms) * time.Millisecond
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
