// Package ctxkeys fournit les clés de contexte partagées entre middleware et services.
package ctxkeys

import (
	"context"

	"levelup/go-api/internal/domain"
)

type contextKey string

const (
	titleSlugKey  contextKey = "title_slug"
	haloTokensKey contextKey = "halo_tokens"
	haloXUIDKey   contextKey = "halo_xuid"
	requestIDKey  contextKey = "request_id"
	eventIDKey    contextKey = "event_id"
	localeKey     contextKey = "locale"
)

// WithLocale place la locale UI ("fr"/"en") dans le contexte. Utilisée par les
// services qui localisent des libellés (médailles, etc.) sans avoir à threader
// la locale dans chaque signature.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey, locale)
}

// Locale extrait la locale depuis le contexte. Retourne "fr" si absente.
func Locale(ctx context.Context) string {
	if v, ok := ctx.Value(localeKey).(string); ok && v != "" {
		return v
	}
	return "fr"
}

// WithTitleSlug place le slug du titre dans le contexte.
func WithTitleSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, titleSlugKey, slug)
}

// TitleSlug extrait le slug du titre depuis le contexte.
// Retourne "halo_infinite" si absent (rétrocompatibilité).
func TitleSlug(ctx context.Context) string {
	if v, ok := ctx.Value(titleSlugKey).(string); ok && v != "" {
		return v
	}
	return "halo_infinite"
}

// WithHaloAuth place les tokens Halo et le XUID du joueur connecté dans le contexte.
func WithHaloAuth(ctx context.Context, tokens *domain.HaloTokens, xuid string) context.Context {
	ctx = context.WithValue(ctx, haloTokensKey, tokens)
	return context.WithValue(ctx, haloXUIDKey, xuid)
}

// HaloTokens extrait les tokens Halo depuis le contexte. Retourne nil si absent.
func HaloTokens(ctx context.Context) *domain.HaloTokens {
	v, _ := ctx.Value(haloTokensKey).(*domain.HaloTokens)
	return v
}

// HaloXUID extrait le XUID depuis le contexte. Retourne "" si absent.
func HaloXUID(ctx context.Context) string {
	v, _ := ctx.Value(haloXUIDKey).(string)
	return v
}

// WithRequestID place l'identifiant de requête dans le contexte.
// P6.4 (revue 2026-04-29 axe 8 BLOQUANT) : permet de corréler une ligne
// d'accès middleware (`request_id` log) avec les `slog.*Context` émis par
// les services pour la même requête. Sans ça, debug prod cassé en multi-user.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extrait l'identifiant de requête depuis le contexte.
// Retourne "" si absent (cas non-HTTP : background jobs, sync tasks, tests).
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithEventID place un identifiant d'événement multi-module dans le contexte.
// Sprint B1 commit 16 : permet de corréler un événement business (sync,
// swap RW, backfill, recompute) à travers les fichiers logs/{module}.log.
//
// Contrairement à request_id (par requête HTTP), event_id couvre des
// opérations background longues qui n'ont pas de request HTTP associée.
// Les deux peuvent coexister : une requête HTTP qui déclenche un sync
// aura request_id (court) + event_id (sync.RunDelta:abc123).
//
// Préférer logging.WithEvent(ctx, prefix) qui génère l'id automatiquement.
func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, eventIDKey, id)
}

// EventID extrait l'identifiant d'événement depuis le contexte.
// Retourne "" si absent (logs hors d'une opération taguée WithEvent).
func EventID(ctx context.Context) string {
	v, _ := ctx.Value(eventIDKey).(string)
	return v
}
