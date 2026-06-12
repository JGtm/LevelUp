// Package observability — error_collector.go : collecteur en mémoire des logs
// WARN/ERROR, agrégés par (niveau, message) depuis le boot. Donne « quelles
// erreurs reviennent et combien de fois » sans grep des fichiers logs.
//
// Insight : slog sépare le message (template fixe, ex « player_watcher: sync
// échoué ») des attributs (valeurs variables : gamertag, err). Agréger par
// message regroupe donc naturellement les occurrences d'une même erreur. Le
// dernier attribut « err » est conservé comme échantillon concret.
//
// Ring borné (defaultErrorCap clés distinctes, éviction LRU par LastSeen) :
// aucune fuite mémoire même si un message porte par erreur une valeur variable.
// Branché comme tee handler (NewErrorCollectorHandler) après le ContextHandler
// dans cmd/server/main.go — n'altère jamais la sortie log existante.
package observability

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// defaultErrorCap borne le nombre de buckets distincts conservés.
const defaultErrorCap = 128

// ErrorBucket agrège les occurrences d'un même (niveau, message).
type ErrorBucket struct {
	Level      string
	Module     string // préfixe du message avant ':' (heuristique), "" sinon
	Message    string
	Count      int64
	FirstSeen  time.Time
	LastSeen   time.Time
	LastDetail string // dernière valeur de l'attribut "err", échantillon concret
}

type errorCollector struct {
	mu      sync.Mutex
	buckets map[string]*ErrorBucket
	cap     int
}

func newErrorCollector(capacity int) *errorCollector {
	return &errorCollector{buckets: make(map[string]*ErrorBucket), cap: capacity}
}

// record agrège un slog.Record (supposé déjà filtré >= WARN par le handler).
func (c *errorCollector) record(r slog.Record) {
	key := r.Level.String() + "|" + r.Message
	c.mu.Lock()
	defer c.mu.Unlock()

	if b, ok := c.buckets[key]; ok {
		b.Count++
		b.LastSeen = r.Time
		if d := extractErr(r); d != "" {
			b.LastDetail = d
		}
		return
	}

	// Nouvelle clé : éviction LRU si plein.
	if len(c.buckets) >= c.cap {
		c.evictOldest()
	}
	c.buckets[key] = &ErrorBucket{
		Level:      r.Level.String(),
		Module:     moduleFromMessage(r.Message),
		Message:    r.Message,
		Count:      1,
		FirstSeen:  r.Time,
		LastSeen:   r.Time,
		LastDetail: extractErr(r),
	}
}

// evictOldest retire le bucket au LastSeen le plus ancien (appelé sous lock).
func (c *errorCollector) evictOldest() {
	var oldestKey string
	var oldest time.Time
	for k, b := range c.buckets {
		if oldestKey == "" || b.LastSeen.Before(oldest) {
			oldestKey, oldest = k, b.LastSeen
		}
	}
	if oldestKey != "" {
		delete(c.buckets, oldestKey)
	}
}

// snapshot retourne une copie triée (Count desc, puis LastSeen desc).
func (c *errorCollector) snapshot() []ErrorBucket {
	c.mu.Lock()
	out := make([]ErrorBucket, 0, len(c.buckets))
	for _, b := range c.buckets {
		out = append(out, *b)
	}
	c.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}

func (c *errorCollector) reset() {
	c.mu.Lock()
	c.buckets = make(map[string]*ErrorBucket)
	c.mu.Unlock()
}

// extractErr renvoie la valeur de l'attribut "err" du record (échantillon),
// "" si absent.
func extractErr(r slog.Record) string {
	var detail string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" {
			detail = a.Value.String()
			return false
		}
		return true
	})
	return detail
}

// moduleFromMessage extrait le préfixe « module: » du message (convention de
// logging du projet, ex « player_watcher: ... »). "" si pas de ':' plausible.
func moduleFromMessage(msg string) string {
	i := strings.IndexByte(msg, ':')
	if i <= 0 || i > 40 {
		return ""
	}
	prefix := msg[:i]
	if strings.ContainsAny(prefix, " \t") {
		return "" // « Failed: ... » n'est pas un module
	}
	return prefix
}

// ─── Singleton + API publique ───────────────────────────────────────────────

var defaultErrorColl = newErrorCollector(defaultErrorCap)

// ErrorBuckets retourne l'état courant agrégé (trié Count desc), pour l'endpoint
// monitoring.
func ErrorBuckets() []ErrorBucket { return defaultErrorColl.snapshot() }

// ResetErrorBuckets vide le collecteur (tests).
func ResetErrorBuckets() { defaultErrorColl.reset() }

// ─── Tee handler ────────────────────────────────────────────────────────────

// ErrorCollectorHandler enveloppe un slog.Handler : feed le collecteur singleton
// pour les records >= WARN, puis délègue inchangé (sortie log préservée).
type ErrorCollectorHandler struct {
	inner slog.Handler
}

// NewErrorCollectorHandler crée le tee handler.
func NewErrorCollectorHandler(inner slog.Handler) *ErrorCollectorHandler {
	return &ErrorCollectorHandler{inner: inner}
}

func (h *ErrorCollectorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ErrorCollectorHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level >= slog.LevelWarn {
		defaultErrorColl.record(record)
	}
	return h.inner.Handle(ctx, record)
}

func (h *ErrorCollectorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ErrorCollectorHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ErrorCollectorHandler) WithGroup(name string) slog.Handler {
	return &ErrorCollectorHandler{inner: h.inner.WithGroup(name)}
}
