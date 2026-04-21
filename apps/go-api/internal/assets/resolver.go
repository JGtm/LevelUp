package assets

import "context"

// Resolver est l'interface publique de la couche d'abstraction assets.
// Les handlers HTTP, le HaloProvider et les CLIs utilisent exclusivement cette interface.
// Jamais d'accès direct au filesystem, DuckDB ou GameCMS en dehors du package assets.
type Resolver interface {
	// Get retourne l'asset identifié par ref.
	// Séquence : FS local → index DuckDB (best-effort) → fetch distant → persist.
	// Erreurs possibles : ErrNotFound, ErrUpstreamUnavailable, ErrUnsupportedKind.
	Get(ctx context.Context, ref Ref) (Resolved, error)

	// Refresh force un re-fetch depuis la source distante, même si l'asset est en cache.
	// Utile pour les CLIs de populate et de migration.
	Refresh(ctx context.Context, ref Ref) (Resolved, error)

	// Warm pré-cache les refs de façon asynchrone (fire-and-forget).
	// Remplace les anciens appels fire-and-forget dans battlepass_details.go.
	Warm(ctx context.Context, refs ...Ref)

	// RegisterLocalFile enregistre un fichier FS existant dans l'index DuckDB.
	// Utilisé par cmd/migrate-static-maps pour indexer les maps statiques.
	RegisterLocalFile(ctx context.Context, ref Ref, path string) error

	// Close flush la WriteQueue et libère les ressources.
	// Appeler depuis le graceful shutdown du serveur.
	Close(ctx context.Context) error
}
