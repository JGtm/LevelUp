// Package port — database_memory.go : le plafond mémoire des bases DuckDB ouvertes, réglable à chaud
// depuis les réglages de données.
package port

import "context"

// DatabaseMemory : le plafond mémoire des bases ouvertes, réglable à chaud.
type DatabaseMemory interface {
	// Apply pose le plafond (en Mo ; 0 = défaut du serveur) pour les connexions à venir et l'applique aux
	// bases déjà ouvertes ; rend le nombre de bases mises à jour et en échec.
	Apply(ctx context.Context, limitMB int) (done, failed int)
	// Effective rend le plafond que reçoit une nouvelle connexion (« 4096MiB »).
	Effective() string
}
