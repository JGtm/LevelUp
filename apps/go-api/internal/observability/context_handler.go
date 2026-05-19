// Package observability — context_handler.go : slog.Handler wrapper qui attache
// automatiquement les valeurs de contexte (request_id, title_slug) à chaque
// log record émis via slog.*Context(...).
//
// P6.4 (revue 2026-04-29 axe 8 BLOQUANT) : sans ça, chaque caller de slog
// devait penser à passer "request_id" en attribut explicite — c'est oublié
// systématiquement et casse la corrélation des logs en multi-user prod.
package observability

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
)

// ContextHandler enveloppe un slog.Handler en lisant le contexte de chaque
// record et en y attachant request_id + title_slug si présents.
//
// Sans request_id dans le ctx (background jobs, sync tasks, tests) : aucun
// attribut ajouté — le handler reste transparent.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler retourne un ContextHandler qui délègue à `inner`.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

// Enabled délègue à l'inner handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle ajoute request_id et event_id (depuis le ctx) au record avant de
// déléguer à l'inner handler. Si l'un ou l'autre est vide (background jobs,
// tests, logs hors d'une opération taguée), l'attribut correspondant n'est
// pas ajouté — le handler reste transparent.
//
// Sprint B1 commit 16 : event_id propagé pour tracer les opérations
// background multi-module (sync, swap RW, backfill) à travers les
// fichiers logs/{module}.log.
func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if id := ctxkeys.RequestID(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	if id := ctxkeys.EventID(ctx); id != "" {
		record.AddAttrs(slog.String("event_id", id))
	}
	return h.inner.Handle(ctx, record)
}

// WithAttrs délègue à l'inner handler.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup délègue à l'inner handler.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
