package assets

import (
	"context"
	"time"
)

// IndexEntry est l'entrée DuckDB représentant un asset indexé.
type IndexEntry struct {
	// Ref est la référence canonique de l'asset.
	Ref Ref
	// URL est l'URL source (distante) si connue.
	URL string
	// LocalPath est le chemin FS local si le binaire est disponible.
	LocalPath string
	// ContentHash est le SHA-256 hex du contenu (8 premiers octets).
	ContentHash string
	// FetchedAt est l'horodatage de dernière récupération.
	FetchedAt time.Time
	// RawJSON est le contenu JSON brut (pour les kinds data uniquement).
	RawJSON []byte
}

// BinaryStore gère le stockage et la récupération des fichiers binaires sur le filesystem.
// Il est toujours disponible (pas de verrouillage externe possible).
type BinaryStore interface {
	// LookupBinary retourne le payload binaire pour la ref donnée.
	// Retourne (nil, nil) si le fichier n'existe pas (cache miss propre).
	LookupBinary(ctx context.Context, ref Ref) (*BinaryPayload, error)

	// PersistBinary écrit le payload de façon atomique (write sur tmp + rename).
	// Retourne ErrPersistFailed si l'écriture échoue.
	PersistBinary(ctx context.Context, ref Ref, p BinaryPayload) error

	// Path retourne le chemin FS attendu pour une Ref (sans vérifier l'existence).
	Path(ref Ref) string
}

// IndexStore gère l'index DuckDB des assets (URLs, paths, hashes, timestamps).
// Il est best-effort : Available() peut retourner false si la DB est verrouillée.
type IndexStore interface {
	// LookupIndex retourne l'entrée d'index pour la ref donnée.
	// Retourne (nil, nil) si l'entrée n'existe pas (cache miss propre).
	// Retourne une erreur si Available() est false ou si la requête échoue.
	LookupIndex(ctx context.Context, ref Ref) (*IndexEntry, error)

	// PersistIndex enregistre ou met à jour une entrée dans l'index.
	// Ne bloque jamais : délègue au WriteQueue asynchrone.
	// Retourne une erreur si Available() est false au moment de l'appel.
	PersistIndex(ctx context.Context, ref Ref, e IndexEntry) error

	// Available retourne true si la DB est accessible en lecture ET en écriture.
	// false si la DB est verrouillée par un autre process ou injoignable.
	Available(ctx context.Context) bool
}
