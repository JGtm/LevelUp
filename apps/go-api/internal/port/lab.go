package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// LabProvider charge le diagnostic d'instance (ex-Lab, A3.5/DC-9 : seul le
// diagnostic survit — consommé par l'onglet admin Données).
type LabProvider interface {
	GetDiagnostics(ctx context.Context, titleSlug string) (*domain.LabDiagnosticsResponse, error)
}
