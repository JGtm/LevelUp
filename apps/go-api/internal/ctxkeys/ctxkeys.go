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
)

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
