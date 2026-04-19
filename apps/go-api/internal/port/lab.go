package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// LabProvider charge les données réelles du Lab interne.
type LabProvider interface {
	GetResources(ctx context.Context, titleSlug string, query domain.LabResourcesQuery) (*domain.LabResourcesResponse, error)
	GetContracts(ctx context.Context) (*domain.LabContractsResponse, error)
	GetDiagnostics(ctx context.Context, titleSlug string) (*domain.LabDiagnosticsResponse, error)
}
