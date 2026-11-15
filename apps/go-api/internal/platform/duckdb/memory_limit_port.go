package duckdb

// memory_limit_port.go : l'adaptateur de port.DatabaseMemory sur le plafond mémoire du paquet
// (memory_limit.go).

import (
	"context"

	"levelup/go-api/internal/port"
)

// MemoryLimit applique le plafond mémoire des bases ouvertes (port.DatabaseMemory) ; l'état vit dans le
// paquet (memory_limit.go), le type n'en porte pas.
type MemoryLimit struct{}

var _ port.DatabaseMemory = MemoryLimit{}

// Apply pose le plafond pour les connexions à venir et l'applique aux bases déjà ouvertes.
func (MemoryLimit) Apply(ctx context.Context, limitMB int) (done, failed int) {
	SetMemoryLimitMB(limitMB)
	return ApplyMemoryLimitToOpen(ctx)
}

// Effective rend le plafond que reçoit une nouvelle connexion.
func (MemoryLimit) Effective() string { return EffectiveMemoryLimit() }
