package port

import "context"

// PlayerReadCacheInvalidator retire des caches de lecture du processus les entrées d'un joueur pour un titre. Un
// écrivain qui n'est pas la synchronisation (l'import des données depuis le site) l'appelle une fois ses lignes
// écrites : sans lui, la liste des matchs et les filtres déjà lus resteraient ceux d'avant. L'implémentation vit
// dans internal/api/wire (elle seule connaît le paquet de données) ; un service ne l'importe jamais.
type PlayerReadCacheInvalidator interface {
	// InvalidatePlayerReadCaches : titleSlug vide = tous les titres du joueur ; xuid vide = sans effet.
	InvalidatePlayerReadCaches(ctx context.Context, xuid, titleSlug string)
}
