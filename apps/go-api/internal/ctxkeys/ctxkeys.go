// Package ctxkeys fournit les clés de contexte partagées entre middleware et services.
package ctxkeys

import "context"

type contextKey string

const (
	// TitleSlugKey est la clé de contexte pour le slug du titre courant.
	titleSlugKey contextKey = "title_slug"
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
