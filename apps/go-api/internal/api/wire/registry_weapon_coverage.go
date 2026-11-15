// Package api — registry_weapon_coverage.go : câblage du runner monitoring « couverture de
// résolution d'arme » (admin). Lecture seule, dual-DB (shared + metadata), réutilise dataQualityHandles.
// La règle est dans service.WeaponCoverageService, les lectures dans duckdb.WeaponCoverageRepo.
package wire

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// WeaponCoverage calcule la couverture de résolution d'arme d'un titre.
func (r *ServiceRegistry) WeaponCoverage(ctx context.Context, titleSlug string) (domain.AdminWeaponCoverage, error) {
	return service.NewWeaponCoverageService(service.WeaponCoverageDeps{
		Open:       r.openWeaponCoverageRepo,
		Classifier: r.killSourceClassifierForSlug,
	}).WeaponCoverage(ctx, titleSlug)
}

// openWeaponCoverageRepo ouvre la base partagée du titre (lecture) et metadata, et rend le dépôt qui les lit.
func (r *ServiceRegistry) openWeaponCoverageRepo(ctx context.Context, titleSlug string) (port.WeaponCoverageRepo, func(), error) {
	h, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return nil, nil, err
	}
	return duckdb.NewWeaponCoverageRepo(h.Shared, h.Meta), h.Close, nil
}
